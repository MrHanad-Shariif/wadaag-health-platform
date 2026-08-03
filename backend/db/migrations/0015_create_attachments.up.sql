-- attachments is the document store backing records.ClinicalObservation's
-- "attachment" observation type: lab results, radiology, prescription
-- scans, etc. It is NOT served through the public /uploads/* static mount
-- (unlike profile photos) — every read must go through an authenticated,
-- consent-gated handler in internal/attachments, since these are sensitive
-- medical documents.
--
-- Versioning: root_id is NULL on the first upload of a logical document
-- (that row IS the root, referenced by its own id). Re-uploading a newer
-- version creates a new row with root_id pointing at the true root (never
-- chained) and version = max existing version in that lineage + 1.
CREATE TABLE attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL REFERENCES patients (id),
    encounter_id UUID REFERENCES encounters (id),
    uploaded_by UUID NOT NULL REFERENCES users (id),
    file_key TEXT NOT NULL,
    filename TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size BIGINT NOT NULL,
    version INT NOT NULL DEFAULT 1,
    root_id UUID REFERENCES attachments (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_attachments_patient ON attachments (patient_id);
CREATE INDEX idx_attachments_encounter ON attachments (encounter_id);
CREATE INDEX idx_attachments_root ON attachments (root_id);
