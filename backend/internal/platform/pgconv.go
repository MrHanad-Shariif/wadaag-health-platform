package platform

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// This file centralizes conversion between pgx's wire types (pgtype.*) and
// the plain Go types every module's domain model uses, so each module's
// repository only has to import it once instead of re-deriving the same
// handful of conversions.

func PgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func FromPgUUID(id pgtype.UUID) uuid.UUID {
	return uuid.UUID(id.Bytes)
}

// PgUUIDPtr converts a nullable UUID (e.g. a FK that may be unset) to its
// pgx form.
func PgUUIDPtr(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func FromPgUUIDPtr(id pgtype.UUID) *uuid.UUID {
	if !id.Valid {
		return nil
	}
	u := uuid.UUID(id.Bytes)
	return &u
}

func PgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func FromPgText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func PgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func PgTimestamptzPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func FromPgTimestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// PgNumeric converts a decimal string (e.g. "125.50", as would come off a
// JSON request body for a NUMERIC column such as providers.consultation_fee)
// to its pgx wire form. A nil or empty string yields NULL. Returns an error
// if s isn't valid decimal text — callers should treat that as a client
// input error (400), not a server error.
func PgNumeric(s *string) (pgtype.Numeric, error) {
	if s == nil || *s == "" {
		return pgtype.Numeric{}, nil
	}
	var n pgtype.Numeric
	if err := n.Scan(*s); err != nil {
		return pgtype.Numeric{}, err
	}
	return n, nil
}

// FromPgNumeric renders a NUMERIC column back to decimal string form (e.g.
// "125.50"), or nil if the column is NULL. There's no existing decimal type
// in this codebase, and pgtype.Numeric has no plain String() method, so this
// goes through its database/sql/driver.Valuer implementation (which encodes
// to text) rather than hand-rolling big.Int/exponent formatting.
func FromPgNumeric(n pgtype.Numeric) *string {
	if !n.Valid {
		return nil
	}
	v, err := n.Value()
	if err != nil || v == nil {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return nil
	}
	return &s
}

// PgInt4Ptr converts a nullable int (e.g. providers.years_experience) to its
// pgx wire form. A nil pointer yields NULL.
func PgInt4Ptr(i *int) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*i), Valid: true}
}

// FromPgInt4Ptr renders a nullable INT4 column back to *int, or nil if the
// column is NULL.
func FromPgInt4Ptr(i pgtype.Int4) *int {
	if !i.Valid {
		return nil
	}
	v := int(i.Int32)
	return &v
}

func PgDatePtr(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func FromPgDatePtr(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	return &d.Time
}
