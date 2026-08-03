package attachments

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/records"
)

// maxAttachmentSize caps an uploaded attachment at 25MB — deliberately a
// separate constant from identity's 5MB profile-photo cap, since these are
// clinical documents (scanned prescriptions, radiology images, PDFs) that
// tend to be larger than a profile photo.
const maxAttachmentSize = 25 * 1024 * 1024

// allowedAttachmentMimeTypes is the explicit allowlist attachments must
// match — same "trust nothing outside a fixed list" philosophy as
// identity's profile-photo upload (see ErrUnsupportedPhotoType).
var allowedAttachmentMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
	"application/pdf": true,
}

var (
	ErrUnsupportedMimeType = errors.New("unsupported attachment content type: only image/jpeg, image/png, image/webp, and application/pdf are allowed")
	ErrAttachmentTooLarge  = errors.New("attachment exceeds the 25MB maximum size")
	// ErrSupersedesNotFound covers both "no such attachment" and "that
	// attachment belongs to a different patient" — deliberately not
	// distinguished in the error message so a caller can't probe for the
	// existence of another patient's attachment IDs.
	ErrSupersedesNotFound       = errors.New("supersedes_id does not reference an existing attachment for this patient")
	ErrEncounterPatientMismatch = errors.New("encounter_id does not belong to this patient")
)

type Service struct {
	repo      *Repository
	records   *records.Service
	uploadDir string
}

func NewService(repo *Repository, recordsService *records.Service, uploadDir string) *Service {
	return &Service{repo: repo, records: recordsService, uploadDir: uploadDir}
}

// UploadInput is Service.Upload's request: PatientID/EncounterID/UploadedBy
// identify who the document is about and who uploaded it, Filename/MimeType/
// Data are the file itself, and SupersedesID — when set — marks this upload
// as a new version of an existing logical document (see Upload's doc
// comment for the versioning model).
type UploadInput struct {
	PatientID    uuid.UUID
	EncounterID  *uuid.UUID
	UploadedBy   uuid.UUID
	Filename     string
	MimeType     string
	Data         []byte
	SupersedesID *uuid.UUID
}

// Upload validates in against the mime-type allowlist and the 25MB size
// cap, confirms the target patient (and encounter, if given) exist,
// resolves the versioning fields, writes the file to
// {UploadDir}/attachments/{patientID}/{uuid}-{sanitized filename}, and
// inserts the resulting row.
//
// Versioning model: when SupersedesID is nil, this is a brand-new logical
// document — root_id is left NULL (this row IS the root) and version is 1.
// When SupersedesID is set, it must reference an existing attachment
// belonging to the same PatientID; the new row's root_id is resolved to
// that attachment's *true* root (its own root_id if it already has one,
// otherwise its own id — never chained through an intermediate version),
// and version is set to one past the highest version already recorded for
// that root's lineage.
func (s *Service) Upload(ctx context.Context, in UploadInput) (Attachment, error) {
	if !allowedAttachmentMimeTypes[in.MimeType] {
		return Attachment{}, ErrUnsupportedMimeType
	}
	if len(in.Data) > maxAttachmentSize {
		return Attachment{}, ErrAttachmentTooLarge
	}

	if _, err := s.records.GetPatient(ctx, in.PatientID); err != nil {
		return Attachment{}, fmt.Errorf("resolve patient: %w", err)
	}

	if in.EncounterID != nil {
		encounter, err := s.records.GetEncounter(ctx, *in.EncounterID)
		if err != nil {
			return Attachment{}, fmt.Errorf("resolve encounter: %w", err)
		}
		if encounter.PatientID != in.PatientID {
			return Attachment{}, ErrEncounterPatientMismatch
		}
	}

	rootID, version, err := s.resolveVersion(ctx, in.PatientID, in.SupersedesID)
	if err != nil {
		return Attachment{}, err
	}

	dir := filepath.Join(s.uploadDir, "attachments", in.PatientID.String())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Attachment{}, fmt.Errorf("create upload dir: %w", err)
	}

	filename := sanitizeFilename(in.Filename)
	diskName := uuid.New().String() + "-" + filename
	// fileKey is forward-slash regardless of OS separator, stored relative
	// to UploadDir (not an absolute filesystem path) — the handler joins it
	// back onto UploadDir when it needs to open the file. See
	// Service.FilePath.
	fileKey := "attachments/" + in.PatientID.String() + "/" + diskName

	if err := os.WriteFile(filepath.Join(dir, diskName), in.Data, 0o644); err != nil {
		return Attachment{}, fmt.Errorf("write attachment file: %w", err)
	}

	return s.repo.CreateAttachment(ctx, CreateAttachmentParams{
		PatientID: in.PatientID, EncounterID: in.EncounterID, UploadedBy: in.UploadedBy,
		FileKey: fileKey, Filename: filename, MimeType: in.MimeType, Size: int64(len(in.Data)),
		Version: version, RootID: rootID,
	})
}

// resolveVersion is the DB-touching half of the versioning model described
// on Upload; resolveRootID below is the pure part.
func (s *Service) resolveVersion(ctx context.Context, patientID uuid.UUID, supersedesID *uuid.UUID) (*uuid.UUID, int32, error) {
	if supersedesID == nil {
		return nil, 1, nil
	}

	target, err := s.repo.FindByID(ctx, *supersedesID)
	if err != nil {
		if errors.Is(err, ErrAttachmentNotFound) {
			return nil, 0, ErrSupersedesNotFound
		}
		return nil, 0, err
	}
	if target.PatientID != patientID {
		return nil, 0, ErrSupersedesNotFound
	}

	root := resolveRootID(target)
	maxVersion, err := s.repo.FindMaxVersionByRoot(ctx, root)
	if err != nil {
		return nil, 0, err
	}
	return &root, maxVersion + 1, nil
}

// resolveRootID returns a's true root: its own root_id if it already has
// one, otherwise its own id. Never chains through an intermediate version —
// callers must already have loaded the row this is resolving from, not just
// an id, so there's nothing further to walk.
func resolveRootID(a Attachment) uuid.UUID {
	if a.RootID != nil {
		return *a.RootID
	}
	return a.ID
}

// sanitizeFilename strips any path-separator characters from a
// client-supplied filename before it's ever used to build a disk path —
// the client-declared filename is untrusted input, and a value like
// "../../etc/passwd" must not be allowed to escape the upload directory.
// Only the base name survives; an empty or entirely-separator result falls
// back to a fixed placeholder.
func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	if idx := strings.LastIndex(name, "/"); idx != -1 {
		name = name[idx+1:]
	}
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		name = "file"
	}
	return name
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Attachment, error) {
	return s.repo.FindByID(ctx, id)
}

// ListForPatient returns one row per logical document — the latest version
// only — for patientID.
func (s *Service) ListForPatient(ctx context.Context, patientID uuid.UUID) ([]Attachment, error) {
	return s.repo.ListLatestByPatient(ctx, patientID)
}

// ListVersions resolves id's true root first (id itself may be any version
// in the lineage, not necessarily the root) and returns every version
// sharing that root, ordered oldest to newest.
func (s *Service) ListVersions(ctx context.Context, id uuid.UUID) ([]Attachment, error) {
	target, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.repo.ListVersionsByRoot(ctx, resolveRootID(target))
}

// FilePath resolves a's stored file_key back to an absolute filesystem
// path under UploadDir, for the handler to open and stream.
func (s *Service) FilePath(a Attachment) string {
	return filepath.Join(s.uploadDir, filepath.FromSlash(a.FileKey))
}

// PatientUserID resolves patientID's linked user_id (if any) — needed by
// the handler's consent resolvers (see resolvePatientRoute /
// resolveAttachmentRoute in handler.go), mirroring
// referrals.PatientLookup's role but implemented as a plain method since
// Service already depends on records.Service directly.
func (s *Service) PatientUserID(ctx context.Context, patientID uuid.UUID) (*uuid.UUID, error) {
	return s.records.GetPatientUserID(ctx, patientID)
}
