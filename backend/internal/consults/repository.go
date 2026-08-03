package consults

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/platform/sqlcgen"
)

var ErrConsultationNotFound = errors.New("consultation not found")

type Repository struct {
	q *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(db)}
}

type CreateConsultationParams struct {
	PatientID            uuid.UUID
	RequestingProviderID uuid.UUID
	TargetProviderID     uuid.UUID
	Reason               string
}

func (r *Repository) Create(ctx context.Context, p CreateConsultationParams) (Consultation, error) {
	row, err := r.q.CreateConsultation(ctx, sqlcgen.CreateConsultationParams{
		PatientID:            platform.PgUUID(p.PatientID),
		RequestingProviderID: platform.PgUUID(p.RequestingProviderID),
		TargetProviderID:     platform.PgUUID(p.TargetProviderID),
		Reason:               p.Reason,
	})
	if err != nil {
		return Consultation{}, fmt.Errorf("insert consultation: %w", err)
	}
	return fromRow(row), nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (Consultation, error) {
	row, err := r.q.FindConsultationByID(ctx, platform.PgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Consultation{}, ErrConsultationNotFound
	}
	if err != nil {
		return Consultation{}, fmt.Errorf("query consultation: %w", err)
	}
	return fromRow(row), nil
}

func (r *Repository) ListForProvider(ctx context.Context, providerID uuid.UUID) ([]Consultation, error) {
	rows, err := r.q.ListConsultationsForProvider(ctx, platform.PgUUID(providerID))
	if err != nil {
		return nil, fmt.Errorf("list consultations for provider: %w", err)
	}
	out := make([]Consultation, len(rows))
	for i, row := range rows {
		out[i] = fromRow(row)
	}
	return out, nil
}

func (r *Repository) ListAll(ctx context.Context) ([]Consultation, error) {
	rows, err := r.q.ListAllConsultations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all consultations: %w", err)
	}
	out := make([]Consultation, len(rows))
	for i, row := range rows {
		out[i] = fromRow(row)
	}
	return out, nil
}

// SearchForProvider is ListForProvider's text-filtered counterpart — same
// requesting-or-target-provider scoping.
func (r *Repository) SearchForProvider(ctx context.Context, providerID uuid.UUID, query string, limit int) ([]Consultation, error) {
	rows, err := r.q.SearchConsultationsForProvider(ctx, sqlcgen.SearchConsultationsForProviderParams{
		ProviderID: platform.PgUUID(providerID), Query: query, ResultLimit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("search consultations for provider: %w", err)
	}
	out := make([]Consultation, len(rows))
	for i, row := range rows {
		out[i] = fromRow(row)
	}
	return out, nil
}

// SearchAll is ListAll's text-filtered counterpart — system_admin oversight
// only, same as its unfiltered sibling.
func (r *Repository) SearchAll(ctx context.Context, query string, limit int) ([]Consultation, error) {
	rows, err := r.q.SearchAllConsultations(ctx, sqlcgen.SearchAllConsultationsParams{
		Query: query, ResultLimit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("search all consultations: %w", err)
	}
	out := make([]Consultation, len(rows))
	for i, row := range rows {
		out[i] = fromRow(row)
	}
	return out, nil
}

// CountPendingForProvider is the physician-specific dashboard view's "my
// pending consults" count — consultations awaiting providerID's first
// reply as the target provider.
func (r *Repository) CountPendingForProvider(ctx context.Context, providerID uuid.UUID) (int64, error) {
	count, err := r.q.CountPendingConsultsForProvider(ctx, platform.PgUUID(providerID))
	if err != nil {
		return 0, fmt.Errorf("count pending consults for provider: %w", err)
	}
	return count, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) (Consultation, error) {
	row, err := r.q.UpdateConsultationStatus(ctx, sqlcgen.UpdateConsultationStatusParams{
		ID: platform.PgUUID(id), Status: sqlcgen.ConsultStatus(status),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Consultation{}, ErrConsultationNotFound
	}
	if err != nil {
		return Consultation{}, fmt.Errorf("update consultation status: %w", err)
	}
	return fromRow(row), nil
}

type CreateMessageParams struct {
	ConsultationID uuid.UUID
	SenderID       uuid.UUID
	Body           string
	AttachmentIDs  []uuid.UUID
}

func (r *Repository) CreateMessage(ctx context.Context, p CreateMessageParams) (Message, error) {
	row, err := r.q.CreateConsultationMessage(ctx, sqlcgen.CreateConsultationMessageParams{
		ConsultationID: platform.PgUUID(p.ConsultationID),
		SenderID:       platform.PgUUID(p.SenderID),
		Body:           p.Body,
		AttachmentIds:  toPgUUIDs(p.AttachmentIDs),
	})
	if err != nil {
		return Message{}, fmt.Errorf("insert consultation message: %w", err)
	}
	return fromMessageRow(row), nil
}

func (r *Repository) ListMessages(ctx context.Context, consultationID uuid.UUID) ([]Message, error) {
	rows, err := r.q.ListConsultationMessages(ctx, platform.PgUUID(consultationID))
	if err != nil {
		return nil, fmt.Errorf("list consultation messages: %w", err)
	}
	out := make([]Message, len(rows))
	for i, row := range rows {
		out[i] = fromMessageRow(row)
	}
	return out, nil
}

func toPgUUIDs(ids []uuid.UUID) []pgtype.UUID {
	out := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		out[i] = platform.PgUUID(id)
	}
	return out
}

func fromPgUUIDs(ids []pgtype.UUID) []uuid.UUID {
	out := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		out[i] = platform.FromPgUUID(id)
	}
	return out
}

func fromRow(row sqlcgen.Consultation) Consultation {
	return Consultation{
		ID:                   platform.FromPgUUID(row.ID),
		PatientID:            platform.FromPgUUID(row.PatientID),
		RequestingProviderID: platform.FromPgUUID(row.RequestingProviderID),
		TargetProviderID:     platform.FromPgUUID(row.TargetProviderID),
		Reason:               row.Reason,
		Status:               Status(row.Status),
		CreatedAt:            row.CreatedAt.Time,
		UpdatedAt:            row.UpdatedAt.Time,
	}
}

func fromMessageRow(row sqlcgen.ConsultationMessage) Message {
	return Message{
		ID:             platform.FromPgUUID(row.ID),
		ConsultationID: platform.FromPgUUID(row.ConsultationID),
		SenderID:       platform.FromPgUUID(row.SenderID),
		Body:           row.Body,
		AttachmentIDs:  fromPgUUIDs(row.AttachmentIds),
		CreatedAt:      row.CreatedAt.Time,
	}
}
