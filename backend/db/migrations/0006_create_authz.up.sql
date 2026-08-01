-- Dynamic RBAC for the new Authentication module. This is intentionally
-- separate from the fixed user_role enum used by the existing
-- identity/records/referrals/facilities modules (see platform.Role) — this
-- migration only wires up permission-driven access for the Authentication
-- module, the patient list, and the dashboard, per the scoping decision to
-- migrate the rest of the app in a later pass.

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    full_access BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    UNIQUE (resource, action)
);

CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

ALTER TABLE users ADD COLUMN role_id UUID REFERENCES roles (id);

-- Fixed permission catalog: what the code actually checks. Only the
-- role-to-permission mapping is admin-editable, not this list itself.
INSERT INTO permissions (resource, action)
SELECT resource, action
FROM (VALUES ('users'), ('roles'), ('patients'), ('referrals'), ('facilities'), ('dashboard')) AS resources (resource)
CROSS JOIN (VALUES ('create'), ('read'), ('update'), ('delete')) AS actions (action);
