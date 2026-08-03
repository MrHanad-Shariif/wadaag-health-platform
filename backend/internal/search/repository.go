package search

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

var ErrSavedSearchNotFound = errors.New("saved search not found")

type Repository struct {
	q *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(db)}
}

// SavedSearch is search's own domain type for a saved_searches row — this
// package owns that table, unlike the patient/referral/consultation/
// doctor/hospital facets it only reads via other modules' services.
type SavedSearch struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Name      string
	Query     string
	Filters   []byte
	CreatedAt time.Time
}

func (r *Repository) CreateSavedSearch(ctx context.Context, userID uuid.UUID, name, query string, filters []byte) (SavedSearch, error) {
	row, err := r.q.CreateSavedSearch(ctx, sqlcgen.CreateSavedSearchParams{
		UserID: platform.PgUUID(userID), Name: name, Query: query, Filters: filters,
	})
	if err != nil {
		return SavedSearch{}, fmt.Errorf("insert saved search: %w", err)
	}
	return savedSearchFromRow(row), nil
}

func (r *Repository) ListSavedSearchesForUser(ctx context.Context, userID uuid.UUID) ([]SavedSearch, error) {
	rows, err := r.q.ListSavedSearchesForUser(ctx, platform.PgUUID(userID))
	if err != nil {
		return nil, fmt.Errorf("list saved searches: %w", err)
	}
	out := make([]SavedSearch, len(rows))
	for i, row := range rows {
		out[i] = savedSearchFromRow(row)
	}
	return out, nil
}

func (r *Repository) FindSavedSearchByID(ctx context.Context, id uuid.UUID) (SavedSearch, error) {
	row, err := r.q.FindSavedSearchByID(ctx, platform.PgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return SavedSearch{}, ErrSavedSearchNotFound
	}
	if err != nil {
		return SavedSearch{}, fmt.Errorf("query saved search: %w", err)
	}
	return savedSearchFromRow(row), nil
}

// DeleteSavedSearch deletes id, scoped to userID at the SQL level (see the
// DeleteSavedSearch query's "AND user_id = $2") — the ownership check
// itself lives in Service.DeleteSavedSearch (which loads the row first to
// tell "doesn't exist" apart from "not yours"), this just reports how many
// rows the delete actually touched.
func (r *Repository) DeleteSavedSearch(ctx context.Context, id, userID uuid.UUID) (int64, error) {
	rows, err := r.q.DeleteSavedSearch(ctx, sqlcgen.DeleteSavedSearchParams{
		ID: platform.PgUUID(id), UserID: platform.PgUUID(userID),
	})
	if err != nil {
		return 0, fmt.Errorf("delete saved search: %w", err)
	}
	return rows, nil
}

func savedSearchFromRow(row sqlcgen.SavedSearch) SavedSearch {
	return SavedSearch{
		ID: platform.FromPgUUID(row.ID), UserID: platform.FromPgUUID(row.UserID),
		Name: row.Name, Query: row.Query, Filters: row.Filters, CreatedAt: row.CreatedAt.Time,
	}
}
