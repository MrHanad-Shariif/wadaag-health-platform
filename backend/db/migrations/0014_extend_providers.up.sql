ALTER TABLE providers ADD COLUMN qualifications TEXT[];
ALTER TABLE providers ADD COLUMN years_experience INT;
ALTER TABLE providers ADD COLUMN consultation_fee NUMERIC(10, 2);

ALTER TABLE providers ADD COLUMN verification_status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE providers ADD CONSTRAINT providers_verification_status_check
    CHECK (verification_status IN ('pending', 'verified', 'rejected'));

ALTER TABLE providers ADD COLUMN languages TEXT[];

-- certificates is free-form JSON, no DB-level schema enforcement. Expected
-- shape: a JSON array of objects, e.g.
-- [{"name": "Board Certification in Cardiology", "issuer": "Somali Medical
-- Board", "year": 2020, "url": "https://..."}] — "url" points at wherever the
-- certificate scan/document is stored (out of scope until Phase 3's
-- attachments module lands; until then this can hold an external link or be
-- omitted).
ALTER TABLE providers ADD COLUMN certificates JSONB;

ALTER TABLE providers ADD COLUMN areas_of_expertise TEXT[];
