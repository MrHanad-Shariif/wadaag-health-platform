// Package notifications is the in-app notification feed: a per-user list of
// events (referral status changes, new consult/direct messages, record
// updates) with read/unread state, plus a per-user, per-channel preference
// row for future delivery channels (email/push) that don't exist yet.
//
// There is only one delivery channel today — the in-app feed itself, which
// is always on — so Service.Publish never consults notification_preferences
// before inserting a row. The preferences CRUD exists purely so a future
// Settings page (Phase 9) and future delivery channels have something to
// read/write against; it is not enforced anywhere yet.
package notifications

import (
	"time"

	"github.com/google/uuid"
)

// Notification is a single feed entry. Payload is raw JSON (shape varies by
// Type — e.g. {"referral_id": ..., "status": ...} for
// "referral_status_changed") passed straight through, same convention as
// facilities.Facility.WorkingHours / records.ClinicalObservation.Payload.
type Notification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Type      string
	Payload   []byte
	ReadAt    *time.Time
	CreatedAt time.Time
}

// Preference is one (user, channel) row. Enabled defaults true (see the
// migration's column default) — there is no UI to flip it yet, but the row
// shape is ready for one.
type Preference struct {
	UserID  uuid.UUID
	Channel string
	Enabled bool
}
