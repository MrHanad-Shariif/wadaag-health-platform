package audit

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

// Logger is the single write path to audit_log. Every module that touches
// patient data calls Record directly (both on allowed and denied access) —
// see internal/consent/middleware.go, which is the main caller.
type Logger struct {
	repo *Repository
}

func NewLogger(repo *Repository) *Logger {
	return &Logger{repo: repo}
}

type RecordInput struct {
	ActorUserID  uuid.UUID
	ActorRole    platform.Role
	Action       Action
	ResourceType string
	ResourceID   *uuid.UUID
	PatientID    *uuid.UUID
	Result       Result
}

// Record never returns an error to the caller — a failure to write an
// audit entry must not itself block or silently bypass the request it's
// describing. It's logged loudly instead so an operator notices.
func (l *Logger) Record(ctx context.Context, in RecordInput) {
	err := l.repo.Insert(ctx, Entry{
		ActorUserID:  in.ActorUserID,
		ActorRole:    in.ActorRole,
		Action:       in.Action,
		ResourceType: in.ResourceType,
		ResourceID:   in.ResourceID,
		PatientID:    in.PatientID,
		Result:       in.Result,
	})
	if err != nil {
		slog.Error("failed to write audit log entry", "error", err, "action", in.Action, "actor", in.ActorUserID)
	}
}

func (l *Logger) ListForPatient(ctx context.Context, patientID uuid.UUID) ([]Entry, error) {
	return l.repo.ListForPatient(ctx, patientID)
}
