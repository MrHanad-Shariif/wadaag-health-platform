CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE user_role AS ENUM (
    'patient',
    'physician',
    'hospital_admin',
    'lab_tech',
    'pharmacist',
    'insurer',
    'system_admin'
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE,
    phone TEXT UNIQUE,
    password_hash TEXT NOT NULL,
    role user_role NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_email_or_phone_present CHECK (email IS NOT NULL OR phone IS NOT NULL)
);

CREATE INDEX idx_users_role ON users (role);
