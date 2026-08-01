CREATE TABLE patients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users (id),
    full_name TEXT NOT NULL,
    date_of_birth DATE,
    sex TEXT,
    national_id TEXT,
    phone TEXT,
    address TEXT,
    next_of_kin TEXT,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_patients_user ON patients (user_id);

CREATE TYPE encounter_type AS ENUM ('referral', 'consult', 'general_visit');

CREATE TABLE encounters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL REFERENCES patients (id),
    facility_id UUID NOT NULL REFERENCES facilities (id),
    provider_id UUID NOT NULL REFERENCES providers (id),
    type encounter_type NOT NULL,
    notes TEXT,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_encounters_patient ON encounters (patient_id);
CREATE INDEX idx_encounters_facility ON encounters (facility_id);

CREATE TYPE observation_type AS ENUM ('vitals', 'diagnosis', 'note', 'attachment');

CREATE TABLE clinical_observations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    encounter_id UUID NOT NULL REFERENCES encounters (id),
    type observation_type NOT NULL,
    payload JSONB NOT NULL,
    recorded_by UUID NOT NULL REFERENCES users (id),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_observations_encounter ON clinical_observations (encounter_id);
