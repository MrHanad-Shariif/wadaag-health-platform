// Package attachments is the document store backing
// records.ClinicalObservation's "attachment" observation type: lab
// results, radiology, prescription scans, and similar patient documents.
// Unlike profile photos (internal/identity/profile_service.go), these are
// sensitive medical files and are never served through the public,
// unauthenticated /uploads/* static mount — every read goes through an
// authenticated, consent-gated handler that streams the file server-side.
// See handler.go and internal/consent/middleware.go.
package attachments

import (
	"time"

	"github.com/google/uuid"
)

// Attachment is one uploaded version of a logical document. See
// Service.Upload's doc comment for the versioning model (root_id/version).
type Attachment struct {
	ID          uuid.UUID
	PatientID   uuid.UUID
	EncounterID *uuid.UUID
	UploadedBy  uuid.UUID
	FileKey     string
	Filename    string
	MimeType    string
	Size        int64
	Version     int32
	RootID      *uuid.UUID
	CreatedAt   time.Time
}
