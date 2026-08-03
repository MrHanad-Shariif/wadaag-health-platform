ALTER TABLE patients ADD COLUMN gender TEXT;
ALTER TABLE patients ADD COLUMN blood_group TEXT;

-- patient_medical_history holds one row per patient. Each jsonb column is a
-- free-form JSON array (e.g. allergies: ["Penicillin", "Peanuts"], or an
-- array of richer objects like {"name": "...", "severity": "..."}) — there
-- is deliberately no DB-level schema enforcement on the array contents;
-- validation, if any, belongs in the application layer.
CREATE TABLE patient_medical_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL UNIQUE REFERENCES patients (id),
    allergies JSONB NOT NULL DEFAULT '[]'::jsonb,
    chronic_conditions JSONB NOT NULL DEFAULT '[]'::jsonb,
    current_medications JSONB NOT NULL DEFAULT '[]'::jsonb,
    past_surgeries JSONB NOT NULL DEFAULT '[]'::jsonb,
    family_history JSONB NOT NULL DEFAULT '[]'::jsonb,
    vaccination_history JSONB NOT NULL DEFAULT '[]'::jsonb,
    updated_by UUID REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_patient_medical_history_patient ON patient_medical_history (patient_id);
