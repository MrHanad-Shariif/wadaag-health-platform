DROP TABLE IF EXISTS patient_medical_history;

ALTER TABLE patients DROP COLUMN IF EXISTS blood_group;
ALTER TABLE patients DROP COLUMN IF EXISTS gender;
