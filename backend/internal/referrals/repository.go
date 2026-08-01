package referrals

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

var ErrReferralNotFound = errors.New("referral not found")
var ErrStaleVersion = errors.New("referral was modified by someone else — reload and retry")

type Repository struct {
	q *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(db)}
}

func (r *Repository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	rows, err := r.q.CountReferralsByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("count referrals by status: %w", err)
	}
	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[string(row.Status)] = row.Total
	}
	return counts, nil
}

type CreateReferralParams struct {
	PatientID                  uuid.UUID
	ReferringProviderID        uuid.UUID
	ReferringFacilityID        uuid.UUID
	ReceivingFacilityID        uuid.UUID
	SpecialtyRequested         string
	Urgency                    Urgency
	Reason                     string
	ClinicalSummaryEncounterID *uuid.UUID
}

func (r *Repository) Create(ctx context.Context, p CreateReferralParams) (Referral, error) {
	row, err := r.q.CreateReferral(ctx, sqlcgen.CreateReferralParams{
		PatientID:                  platform.PgUUID(p.PatientID),
		ReferringProviderID:        platform.PgUUID(p.ReferringProviderID),
		ReferringFacilityID:        platform.PgUUID(p.ReferringFacilityID),
		ReceivingFacilityID:        platform.PgUUID(p.ReceivingFacilityID),
		SpecialtyRequested:         p.SpecialtyRequested,
		Urgency:                    sqlcgen.ReferralUrgency(p.Urgency),
		Reason:                     p.Reason,
		ClinicalSummaryEncounterID: platform.PgUUIDPtr(p.ClinicalSummaryEncounterID),
	})
	if err != nil {
		return Referral{}, fmt.Errorf("insert referral: %w", err)
	}
	return fromRow(row), nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (Referral, error) {
	row, err := r.q.FindReferralByID(ctx, platform.PgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Referral{}, ErrReferralNotFound
	}
	if err != nil {
		return Referral{}, fmt.Errorf("query referral: %w", err)
	}
	return fromRow(row), nil
}

func (r *Repository) ListForFacility(ctx context.Context, facilityID uuid.UUID) ([]Referral, error) {
	rows, err := r.q.ListReferralsForFacility(ctx, platform.PgUUID(facilityID))
	if err != nil {
		return nil, fmt.Errorf("list referrals for facility: %w", err)
	}
	return fromRows(rows), nil
}

func (r *Repository) ListForPatient(ctx context.Context, patientID uuid.UUID) ([]Referral, error) {
	rows, err := r.q.ListReferralsByPatient(ctx, platform.PgUUID(patientID))
	if err != nil {
		return nil, fmt.Errorf("list referrals for patient: %w", err)
	}
	return fromRows(rows), nil
}

func (r *Repository) Accept(ctx context.Context, id uuid.UUID, version int32, receivingProviderID uuid.UUID) (Referral, error) {
	row, err := r.q.AcceptReferral(ctx, sqlcgen.AcceptReferralParams{
		ID: platform.PgUUID(id), Version: version, ReceivingProviderID: platform.PgUUID(receivingProviderID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Referral{}, ErrStaleVersion
	}
	if err != nil {
		return Referral{}, fmt.Errorf("accept referral: %w", err)
	}
	return fromRow(row), nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, version int32, status Status) (Referral, error) {
	row, err := r.q.UpdateReferralStatus(ctx, sqlcgen.UpdateReferralStatusParams{
		ID: platform.PgUUID(id), Version: version, Status: sqlcgen.ReferralStatus(status),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Referral{}, ErrStaleVersion
	}
	if err != nil {
		return Referral{}, fmt.Errorf("update referral status: %w", err)
	}
	return fromRow(row), nil
}

func (r *Repository) CreateStatusEvent(ctx context.Context, referralID uuid.UUID, fromStatus *Status, toStatus Status, actorUserID uuid.UUID, note *string) error {
	var from sqlcgen.NullReferralStatus
	if fromStatus != nil {
		from = sqlcgen.NullReferralStatus{ReferralStatus: sqlcgen.ReferralStatus(*fromStatus), Valid: true}
	}

	_, err := r.q.CreateReferralStatusEvent(ctx, sqlcgen.CreateReferralStatusEventParams{
		ReferralID: platform.PgUUID(referralID), FromStatus: from, ToStatus: sqlcgen.ReferralStatus(toStatus),
		ActorUserID: platform.PgUUID(actorUserID), Note: platform.PgText(note),
	})
	if err != nil {
		return fmt.Errorf("insert referral status event: %w", err)
	}
	return nil
}

func (r *Repository) ListStatusEvents(ctx context.Context, referralID uuid.UUID) ([]StatusEvent, error) {
	rows, err := r.q.ListReferralStatusEvents(ctx, platform.PgUUID(referralID))
	if err != nil {
		return nil, fmt.Errorf("list referral status events: %w", err)
	}

	events := make([]StatusEvent, len(rows))
	for i, row := range rows {
		var from *Status
		if row.FromStatus.Valid {
			s := Status(row.FromStatus.ReferralStatus)
			from = &s
		}
		events[i] = StatusEvent{
			ID: platform.FromPgUUID(row.ID), ReferralID: platform.FromPgUUID(row.ReferralID),
			FromStatus: from, ToStatus: Status(row.ToStatus), ActorUserID: platform.FromPgUUID(row.ActorUserID),
			Note: platform.FromPgText(row.Note), OccurredAt: row.OccurredAt.Time,
		}
	}
	return events, nil
}

func fromRow(row sqlcgen.Referral) Referral {
	return Referral{
		ID:                         platform.FromPgUUID(row.ID),
		PatientID:                  platform.FromPgUUID(row.PatientID),
		ReferringProviderID:        platform.FromPgUUID(row.ReferringProviderID),
		ReferringFacilityID:        platform.FromPgUUID(row.ReferringFacilityID),
		ReceivingFacilityID:        platform.FromPgUUIDPtr(row.ReceivingFacilityID),
		ReceivingProviderID:        platform.FromPgUUIDPtr(row.ReceivingProviderID),
		SpecialtyRequested:         row.SpecialtyRequested,
		Urgency:                    Urgency(row.Urgency),
		Status:                     Status(row.Status),
		Reason:                     row.Reason,
		ClinicalSummaryEncounterID: platform.FromPgUUIDPtr(row.ClinicalSummaryEncounterID),
		Version:                    row.Version,
		CreatedAt:                  row.CreatedAt.Time,
		UpdatedAt:                  row.UpdatedAt.Time,
	}
}

func fromRows(rows []sqlcgen.Referral) []Referral {
	out := make([]Referral, len(rows))
	for i, row := range rows {
		out[i] = fromRow(row)
	}
	return out
}
