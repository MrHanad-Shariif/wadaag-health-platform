package consent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/platform/sqlcgen"
)

var ErrGrantNotFound = errors.New("consent grant not found or already revoked")

type Repository struct {
	q *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(db)}
}

type CreateGrantParams struct {
	PatientID  uuid.UUID
	Grantee    GranteeType
	GranteeID  uuid.UUID
	Scope      Scope
	ScopeRef   *uuid.UUID
	GrantedVia GrantedVia
	ExpiresAt  *time.Time
}

func (r *Repository) CreateGrant(ctx context.Context, p CreateGrantParams) (Grant, error) {
	row, err := r.q.CreateConsentGrant(ctx, sqlcgen.CreateConsentGrantParams{
		PatientID:   platform.PgUUID(p.PatientID),
		GranteeType: sqlcgen.ConsentGranteeType(p.Grantee),
		GranteeID:   platform.PgUUID(p.GranteeID),
		Scope:       sqlcgen.ConsentScope(p.Scope),
		ScopeRef:    platform.PgUUIDPtr(p.ScopeRef),
		GrantedVia:  sqlcgen.ConsentGrantedVia(p.GrantedVia),
		ExpiresAt:   platform.PgTimestamptzPtr(p.ExpiresAt),
	})
	if err != nil {
		return Grant{}, fmt.Errorf("insert consent grant: %w", err)
	}
	return fromRow(row), nil
}

// FindActiveGrant reports whether any active, unexpired grant lets
// (providerUserID or facilityID) access patientID. facilityID may be the
// zero UUID when the actor has no facility affiliation (e.g. a patient
// account) — the query simply never matches the facility branch then.
func (r *Repository) FindActiveGrant(ctx context.Context, patientID, providerUserID, facilityID uuid.UUID) (bool, error) {
	rows, err := r.q.FindActiveGrants(ctx, sqlcgen.FindActiveGrantsParams{
		PatientID:   platform.PgUUID(patientID),
		GranteeID:   platform.PgUUID(providerUserID),
		GranteeID_2: platform.PgUUID(facilityID),
	})
	if err != nil {
		return false, fmt.Errorf("query consent grants: %w", err)
	}
	return len(rows) > 0, nil
}

func (r *Repository) ListForPatient(ctx context.Context, patientID uuid.UUID) ([]Grant, error) {
	rows, err := r.q.ListConsentGrantsForPatient(ctx, platform.PgUUID(patientID))
	if err != nil {
		return nil, fmt.Errorf("list consent grants: %w", err)
	}

	grants := make([]Grant, len(rows))
	for i, row := range rows {
		grants[i] = fromRow(row)
	}
	return grants, nil
}

func (r *Repository) Revoke(ctx context.Context, grantID, patientID uuid.UUID) (Grant, error) {
	row, err := r.q.RevokeConsentGrant(ctx, sqlcgen.RevokeConsentGrantParams{
		ID:        platform.PgUUID(grantID),
		PatientID: platform.PgUUID(patientID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Grant{}, ErrGrantNotFound
	}
	if err != nil {
		return Grant{}, fmt.Errorf("revoke consent grant: %w", err)
	}
	return fromRow(row), nil
}

func fromRow(row sqlcgen.ConsentGrant) Grant {
	return Grant{
		ID:          platform.FromPgUUID(row.ID),
		PatientID:   platform.FromPgUUID(row.PatientID),
		GranteeType: GranteeType(row.GranteeType),
		GranteeID:   platform.FromPgUUID(row.GranteeID),
		Scope:       Scope(row.Scope),
		ScopeRef:    platform.FromPgUUIDPtr(row.ScopeRef),
		Status:      Status(row.Status),
		GrantedVia:  GrantedVia(row.GrantedVia),
		GrantedAt:   row.GrantedAt.Time,
		RevokedAt:   platform.FromPgTimestamptzPtr(row.RevokedAt),
		ExpiresAt:   platform.FromPgTimestamptzPtr(row.ExpiresAt),
	}
}
