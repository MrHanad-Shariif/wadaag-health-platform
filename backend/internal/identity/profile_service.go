package identity

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// maxProfilePhotoSize caps an uploaded profile photo at 5MB.
const maxProfilePhotoSize = 5 * 1024 * 1024

var ErrUnsupportedPhotoType = errors.New("unsupported photo content type: only image/jpeg, image/png, and image/webp are allowed")
var ErrPhotoTooLarge = errors.New("photo exceeds the 5MB maximum size")

// GetProfile returns userID's profile, or a zero-value UserProfile (empty
// bio, no photo, empty languages, offline) if no row has ever been written
// for them — see the row lifecycle note in
// db/migrations/0010_create_user_profiles.up.sql. Never returns
// ErrProfileNotFound to the caller.
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (UserProfile, error) {
	profile, err := s.repo.GetUserProfile(ctx, userID)
	if errors.Is(err, ErrProfileNotFound) {
		return UserProfile{UserID: userID, Languages: []string{}}, nil
	}
	if err != nil {
		return UserProfile{}, err
	}
	return profile, nil
}

// UpdateProfileInput is PATCH /me/profile's body, pre-decoded: a nil field
// means "leave this field unchanged," matching how a partial-update PATCH is
// expected to behave. Note this means there's no way to distinguish an
// omitted field from an explicit `null` in the JSON body — both leave the
// field untouched, which is an accepted simplification for this phase
// (nothing in the contract needs "clear this field to null" yet; Bio and
// AvailabilityStatus can be cleared by sending an empty string).
type UpdateProfileInput struct {
	Bio                *string
	Languages          *[]string
	AvailabilityStatus *string
}

// UpdateProfile is the "upsert-if-missing" partial update: it reads
// whatever row currently exists (treating a missing row as all-zero-value),
// overlays only the fields the caller supplied, and writes the merged
// result — so a PATCH that only sends {"bio": "..."} doesn't wipe out
// languages/availability_status a caller set earlier, and works identically
// whether or not a row existed yet. See mergeProfileUpdate for the pure
// merge logic this delegates to.
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, in UpdateProfileInput) (UserProfile, error) {
	current, err := s.repo.GetUserProfile(ctx, userID)
	if err != nil && !errors.Is(err, ErrProfileNotFound) {
		return UserProfile{}, err
	}
	// current is the zero-value UserProfile{} when ErrProfileNotFound —
	// exactly the right merge base for a first-ever write.

	bio, languages, availabilityStatus := mergeProfileUpdate(current, in)
	return s.repo.UpsertProfileFields(ctx, userID, bio, languages, availabilityStatus)
}

// mergeProfileUpdate overlays in's non-nil fields onto current, leaving
// everything else as-is. Pulled out as a pure function (no DB) so the
// partial-update semantics can be unit-tested directly.
func mergeProfileUpdate(current UserProfile, in UpdateProfileInput) (bio *string, languages []string, availabilityStatus *string) {
	bio = current.Bio
	if in.Bio != nil {
		bio = in.Bio
	}

	languages = current.Languages
	if in.Languages != nil {
		languages = *in.Languages
	}

	availabilityStatus = current.AvailabilityStatus
	if in.AvailabilityStatus != nil {
		availabilityStatus = in.AvailabilityStatus
	}

	return bio, languages, availabilityStatus
}

// UploadProfilePhoto validates data as a JPEG/PNG/WebP image (by magic
// bytes, not the client-declared Content-Type, which is easy to spoof and
// not trusted here), writes it to
// {uploadDir}/profile-photos/{userID}{ext}, and persists the resulting
// servable URL onto the profile row. Any previously stored photo for this
// user is removed first (regardless of its extension) so a re-upload that
// changes image type doesn't leave an orphaned file behind — the fixed
// {userID}{ext} filename otherwise only overwrites cleanly when the
// extension is unchanged.
func (s *Service) UploadProfilePhoto(ctx context.Context, userID uuid.UUID, data []byte) (UserProfile, error) {
	if len(data) > maxProfilePhotoSize {
		return UserProfile{}, ErrPhotoTooLarge
	}

	ext, ok := detectImageExt(data)
	if !ok {
		return UserProfile{}, ErrUnsupportedPhotoType
	}

	dir := filepath.Join(s.uploadDir, "profile-photos")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return UserProfile{}, fmt.Errorf("create upload dir: %w", err)
	}

	if matches, err := filepath.Glob(filepath.Join(dir, userID.String()+".*")); err == nil {
		for _, m := range matches {
			_ = os.Remove(m)
		}
	}

	filename := userID.String() + ext
	if err := os.WriteFile(filepath.Join(dir, filename), data, 0o644); err != nil {
		return UserProfile{}, fmt.Errorf("write photo file: %w", err)
	}

	// Forward-slash URL path regardless of OS path separator — this is
	// served over HTTP (see router.go's /uploads/* mount), not a filesystem
	// path.
	photoURL := "/uploads/profile-photos/" + filename

	return s.repo.UpsertProfilePhoto(ctx, userID, photoURL)
}

// detectImageExt sniffs the file extension from magic bytes rather than
// trusting the multipart part's declared Content-Type header, which a
// caller can set to anything. Only the three types this endpoint accepts
// are recognized.
func detectImageExt(data []byte) (string, bool) {
	switch {
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return ".jpg", true
	case len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}):
		return ".png", true
	case len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return ".webp", true
	default:
		return "", false
	}
}
