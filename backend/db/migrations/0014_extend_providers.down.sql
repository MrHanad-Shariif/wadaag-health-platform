ALTER TABLE providers DROP COLUMN IF EXISTS areas_of_expertise;
ALTER TABLE providers DROP COLUMN IF EXISTS certificates;
ALTER TABLE providers DROP COLUMN IF EXISTS languages;
ALTER TABLE providers DROP CONSTRAINT IF EXISTS providers_verification_status_check;
ALTER TABLE providers DROP COLUMN IF EXISTS verification_status;
ALTER TABLE providers DROP COLUMN IF EXISTS consultation_fee;
ALTER TABLE providers DROP COLUMN IF EXISTS years_experience;
ALTER TABLE providers DROP COLUMN IF EXISTS qualifications;
