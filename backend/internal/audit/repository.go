package audit

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/platform/sqlcgen"
)

type Repository struct {
	q *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(db)}
}

func (r *Repository) Insert(ctx context.Context, e Entry) error {
	_, err := r.q.InsertAuditLog(ctx, sqlcgen.InsertAuditLogParams{
		ActorUserID:  platform.PgUUID(e.ActorUserID),
		ActorRole:    sqlcgen.UserRole(e.ActorRole),
		Action:       string(e.Action),
		ResourceType: e.ResourceType,
		ResourceID:   platform.PgUUIDPtr(e.ResourceID),
		PatientID:    platform.PgUUIDPtr(e.PatientID),
		Result:       sqlcgen.AuditResult(e.Result),
	})
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

// ListEntries is the general, filtered audit-browser listing — see
// EntryFilter's doc comment for the filter semantics.
func (r *Repository) ListEntries(ctx context.Context, filter EntryFilter) ([]Entry, error) {
	var result pgtype.Text
	if filter.Result != nil {
		result = platform.PgText((*string)(filter.Result))
	}

	rows, err := r.q.ListAuditEntries(ctx, sqlcgen.ListAuditEntriesParams{
		ActorUserID:  platform.PgUUIDPtr(filter.ActorUserID),
		ResourceType: platform.PgText(filter.ResourceType),
		Result:       result,
		ResultLimit:  int32(filter.NormalizeLimit()),
	})
	if err != nil {
		return nil, fmt.Errorf("list audit entries: %w", err)
	}

	entries := make([]Entry, len(rows))
	for i, row := range rows {
		entries[i] = entryFromRow(row)
	}
	return entries, nil
}

func (r *Repository) ListForPatient(ctx context.Context, patientID uuid.UUID) ([]Entry, error) {
	rows, err := r.q.ListAuditLogForPatient(ctx, platform.PgUUIDPtr(&patientID))
	if err != nil {
		return nil, fmt.Errorf("list audit log: %w", err)
	}

	entries := make([]Entry, len(rows))
	for i, row := range rows {
		entries[i] = entryFromRow(row)
	}
	return entries, nil
}

func entryFromRow(row sqlcgen.AuditLog) Entry {
	return Entry{
		ID:           platform.FromPgUUID(row.ID),
		ActorUserID:  platform.FromPgUUID(row.ActorUserID),
		ActorRole:    platform.Role(row.ActorRole),
		Action:       Action(row.Action),
		ResourceType: row.ResourceType,
		ResourceID:   platform.FromPgUUIDPtr(row.ResourceID),
		PatientID:    platform.FromPgUUIDPtr(row.PatientID),
		Result:       Result(row.Result),
		OccurredAt:   row.OccurredAt.Time,
	}
}
