package attachments

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/platform/sqlcgen"
)

var ErrAttachmentNotFound = errors.New("attachment not found")

type Repository struct {
	q *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(db)}
}

type CreateAttachmentParams struct {
	PatientID   uuid.UUID
	EncounterID *uuid.UUID
	UploadedBy  uuid.UUID
	FileKey     string
	Filename    string
	MimeType    string
	Size        int64
	Version     int32
	RootID      *uuid.UUID
}

func (r *Repository) CreateAttachment(ctx context.Context, p CreateAttachmentParams) (Attachment, error) {
	row, err := r.q.CreateAttachment(ctx, sqlcgen.CreateAttachmentParams{
		PatientID:   platform.PgUUID(p.PatientID),
		EncounterID: platform.PgUUIDPtr(p.EncounterID),
		UploadedBy:  platform.PgUUID(p.UploadedBy),
		FileKey:     p.FileKey,
		Filename:    p.Filename,
		MimeType:    p.MimeType,
		Size:        p.Size,
		Version:     p.Version,
		RootID:      platform.PgUUIDPtr(p.RootID),
	})
	if err != nil {
		return Attachment{}, fmt.Errorf("insert attachment: %w", err)
	}
	return attachmentFromRow(row), nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (Attachment, error) {
	row, err := r.q.FindAttachmentByID(ctx, platform.PgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Attachment{}, ErrAttachmentNotFound
	}
	if err != nil {
		return Attachment{}, fmt.Errorf("query attachment: %w", err)
	}
	return attachmentFromRow(row), nil
}

func (r *Repository) ListLatestByPatient(ctx context.Context, patientID uuid.UUID) ([]Attachment, error) {
	rows, err := r.q.ListLatestAttachmentsByPatient(ctx, platform.PgUUID(patientID))
	if err != nil {
		return nil, fmt.Errorf("list latest attachments: %w", err)
	}
	out := make([]Attachment, len(rows))
	for i, row := range rows {
		out[i] = attachmentFromRow(row)
	}
	return out, nil
}

func (r *Repository) ListVersionsByRoot(ctx context.Context, rootID uuid.UUID) ([]Attachment, error) {
	rows, err := r.q.ListAttachmentVersionsByRoot(ctx, platform.PgUUID(rootID))
	if err != nil {
		return nil, fmt.Errorf("list attachment versions: %w", err)
	}
	out := make([]Attachment, len(rows))
	for i, row := range rows {
		out[i] = attachmentFromRow(row)
	}
	return out, nil
}

// FindMaxVersionByRoot returns the highest version currently stored for
// rootID's lineage, or 0 if no rows exist for it yet.
func (r *Repository) FindMaxVersionByRoot(ctx context.Context, rootID uuid.UUID) (int32, error) {
	max, err := r.q.FindMaxAttachmentVersionByRoot(ctx, platform.PgUUID(rootID))
	if err != nil {
		return 0, fmt.Errorf("find max attachment version: %w", err)
	}
	return max, nil
}

func attachmentFromRow(row sqlcgen.Attachment) Attachment {
	return Attachment{
		ID:          platform.FromPgUUID(row.ID),
		PatientID:   platform.FromPgUUID(row.PatientID),
		EncounterID: platform.FromPgUUIDPtr(row.EncounterID),
		UploadedBy:  platform.FromPgUUID(row.UploadedBy),
		FileKey:     row.FileKey,
		Filename:    row.Filename,
		MimeType:    row.MimeType,
		Size:        row.Size,
		Version:     row.Version,
		RootID:      platform.FromPgUUIDPtr(row.RootID),
		CreatedAt:   row.CreatedAt.Time,
	}
}
