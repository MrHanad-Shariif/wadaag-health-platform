-- User profiles: self-service "manage my own profile" data, kept in a
-- separate table from `users` since identity (auth) and profile-editing are
-- different concerns with different write patterns (profile fields change
-- often and are self-edited; users/auth fields are security-sensitive and
-- narrowly written).
--
-- Row lifecycle: created lazily. There is no row inserted at user-creation
-- time (register/CreateAccount) — GET /me/profile returns zero-values when
-- no row exists yet, and PATCH /me/profile / POST /me/profile/photo upsert
-- (INSERT ... ON CONFLICT) on first write. This avoids adding profile-row
-- creation to every account-creation path (self-register, admin-created
-- accounts) for a row most users won't touch until they actually visit the
-- profile page.

CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    photo_url TEXT,
    bio TEXT,
    languages TEXT[] NOT NULL DEFAULT '{}',
    availability_status TEXT,
    is_online BOOLEAN NOT NULL DEFAULT false,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
