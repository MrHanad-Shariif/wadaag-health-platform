CREATE TABLE branches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    facility_id UUID NOT NULL REFERENCES facilities (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    address TEXT,
    phone TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_branches_facility ON branches (facility_id);

CREATE TABLE departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    facility_id UUID NOT NULL REFERENCES facilities (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_departments_facility ON departments (facility_id);

ALTER TABLE facilities ADD COLUMN logo_url TEXT;

-- working_hours is free-form JSON, no DB-level schema enforcement. Expected
-- shape: {"mon": {"open": "08:00", "close": "17:00"}, "tue": {...}, ...} —
-- one entry per weekday (three-letter lowercase key), each an
-- {"open", "close"} pair in 24-hour "HH:MM" form; a day can be omitted to
-- mean "closed."
ALTER TABLE facilities ADD COLUMN working_hours JSONB;
