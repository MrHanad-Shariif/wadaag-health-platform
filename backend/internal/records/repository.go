package records

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

var ErrPatientNotFound = errors.New("patient not found")
var ErrEncounterNotFound = errors.New("encounter not found")

type Repository struct {
	q *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(db)}
}

type CreatePatientParams struct {
	UserID      *uuid.UUID
	FullName    string
	DateOfBirth *time.Time
	Sex         *string
	NationalID  *string
	Phone       *string
	Address     *string
	NextOfKin   *string
}

func (r *Repository) CreatePatient(ctx context.Context, p CreatePatientParams) (Patient, error) {
	row, err := r.q.CreatePatient(ctx, sqlcgen.CreatePatientParams{
		UserID:      platform.PgUUIDPtr(p.UserID),
		FullName:    p.FullName,
		DateOfBirth: platform.PgDatePtr(p.DateOfBirth),
		Sex:         platform.PgText(p.Sex),
		NationalID:  platform.PgText(p.NationalID),
		Phone:       platform.PgText(p.Phone),
		Address:     platform.PgText(p.Address),
		NextOfKin:   platform.PgText(p.NextOfKin),
	})
	if err != nil {
		return Patient{}, fmt.Errorf("insert patient: %w", err)
	}
	return patientFromRow(row), nil
}

func (r *Repository) FindPatientByID(ctx context.Context, id uuid.UUID) (Patient, error) {
	row, err := r.q.FindPatientByID(ctx, platform.PgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Patient{}, ErrPatientNotFound
	}
	if err != nil {
		return Patient{}, fmt.Errorf("query patient: %w", err)
	}
	return patientFromRow(row), nil
}

func (r *Repository) ListPatients(ctx context.Context) ([]Patient, error) {
	rows, err := r.q.ListPatients(ctx)
	if err != nil {
		return nil, fmt.Errorf("list patients: %w", err)
	}
	patients := make([]Patient, len(rows))
	for i, row := range rows {
		patients[i] = patientFromRow(row)
	}
	return patients, nil
}

func (r *Repository) CountPatients(ctx context.Context) (int64, error) {
	count, err := r.q.CountPatients(ctx)
	if err != nil {
		return 0, fmt.Errorf("count patients: %w", err)
	}
	return count, nil
}

type CreateEncounterParams struct {
	PatientID  uuid.UUID
	FacilityID uuid.UUID
	ProviderID uuid.UUID
	Type       EncounterType
	Notes      *string
	OccurredAt time.Time
}

func (r *Repository) CreateEncounter(ctx context.Context, p CreateEncounterParams) (Encounter, error) {
	row, err := r.q.CreateEncounter(ctx, sqlcgen.CreateEncounterParams{
		PatientID:  platform.PgUUID(p.PatientID),
		FacilityID: platform.PgUUID(p.FacilityID),
		ProviderID: platform.PgUUID(p.ProviderID),
		Type:       sqlcgen.EncounterType(p.Type),
		Notes:      platform.PgText(p.Notes),
		OccurredAt: platform.PgTimestamptz(p.OccurredAt),
	})
	if err != nil {
		return Encounter{}, fmt.Errorf("insert encounter: %w", err)
	}
	return encounterFromRow(row), nil
}

func (r *Repository) FindEncounterByID(ctx context.Context, id uuid.UUID) (Encounter, error) {
	row, err := r.q.FindEncounterByID(ctx, platform.PgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Encounter{}, ErrEncounterNotFound
	}
	if err != nil {
		return Encounter{}, fmt.Errorf("query encounter: %w", err)
	}
	return encounterFromRow(row), nil
}

func (r *Repository) ListEncountersByPatient(ctx context.Context, patientID uuid.UUID) ([]Encounter, error) {
	rows, err := r.q.ListEncountersByPatient(ctx, platform.PgUUID(patientID))
	if err != nil {
		return nil, fmt.Errorf("list encounters: %w", err)
	}

	encounters := make([]Encounter, len(rows))
	for i, row := range rows {
		encounters[i] = encounterFromRow(row)
	}
	return encounters, nil
}

func (r *Repository) CountEncounters(ctx context.Context) (int64, error) {
	count, err := r.q.CountEncounters(ctx)
	if err != nil {
		return 0, fmt.Errorf("count encounters: %w", err)
	}
	return count, nil
}

func (r *Repository) CreateObservation(ctx context.Context, encounterID uuid.UUID, obsType ObservationType, payload []byte, recordedBy uuid.UUID) (ClinicalObservation, error) {
	row, err := r.q.CreateClinicalObservation(ctx, sqlcgen.CreateClinicalObservationParams{
		EncounterID: platform.PgUUID(encounterID),
		Type:        sqlcgen.ObservationType(obsType),
		Payload:     payload,
		RecordedBy:  platform.PgUUID(recordedBy),
	})
	if err != nil {
		return ClinicalObservation{}, fmt.Errorf("insert clinical observation: %w", err)
	}
	return observationFromRow(row), nil
}

func (r *Repository) ListObservationsByEncounter(ctx context.Context, encounterID uuid.UUID) ([]ClinicalObservation, error) {
	rows, err := r.q.ListObservationsByEncounter(ctx, platform.PgUUID(encounterID))
	if err != nil {
		return nil, fmt.Errorf("list clinical observations: %w", err)
	}

	observations := make([]ClinicalObservation, len(rows))
	for i, row := range rows {
		observations[i] = observationFromRow(row)
	}
	return observations, nil
}

func patientFromRow(row sqlcgen.Patient) Patient {
	return Patient{
		ID:          platform.FromPgUUID(row.ID),
		UserID:      platform.FromPgUUIDPtr(row.UserID),
		FullName:    row.FullName,
		DateOfBirth: platform.FromPgDatePtr(row.DateOfBirth),
		Sex:         platform.FromPgText(row.Sex),
		NationalID:  platform.FromPgText(row.NationalID),
		Phone:       platform.FromPgText(row.Phone),
		Address:     platform.FromPgText(row.Address),
		NextOfKin:   platform.FromPgText(row.NextOfKin),
		Version:     row.Version,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}

func encounterFromRow(row sqlcgen.Encounter) Encounter {
	return Encounter{
		ID:         platform.FromPgUUID(row.ID),
		PatientID:  platform.FromPgUUID(row.PatientID),
		FacilityID: platform.FromPgUUID(row.FacilityID),
		ProviderID: platform.FromPgUUID(row.ProviderID),
		Type:       EncounterType(row.Type),
		Notes:      platform.FromPgText(row.Notes),
		OccurredAt: row.OccurredAt.Time,
		Version:    row.Version,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}

func observationFromRow(row sqlcgen.ClinicalObservation) ClinicalObservation {
	return ClinicalObservation{
		ID:          platform.FromPgUUID(row.ID),
		EncounterID: platform.FromPgUUID(row.EncounterID),
		Type:        ObservationType(row.Type),
		Payload:     row.Payload,
		RecordedBy:  platform.FromPgUUID(row.RecordedBy),
		RecordedAt:  row.RecordedAt.Time,
	}
}
