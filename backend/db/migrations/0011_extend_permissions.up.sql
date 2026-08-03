-- Phase 1 (RBAC expansion): extend the dynamic permission catalog and seed
-- three new dynamic roles per the roadmap (docs/plans/full-feature-roadmap.md,
-- "Phase 1 — User Profiles & RBAC Expansion").
--
-- These bundles were hand-defined for this migration, not copied from a
-- pre-existing "Hospital Admin"/"Doctor" dynamic role — no such dynamic
-- roles exist yet (only the legacy fixed platform.Role enum has those
-- names). "Organization Admin" is intentionally NOT facility-scoped in this
-- phase: it is a global permission grant, matching how Administrator/Super
-- Admin already work. True facility/multi-branch scoping is deferred to
-- Phase 2 per the roadmap's design note.

-- 1. Extend the permission catalog with the missing (resource, action)
--    pairs from the owner's list. Some pairs may already exist from the
--    0006 cross-join (e.g. referrals:create) -- ON CONFLICT handles that.
INSERT INTO permissions (resource, action) VALUES
    ('patients', 'view'),
    ('patients', 'edit'),
    ('referrals', 'accept'),
    ('records', 'upload'),
    ('hospitals', 'manage'),
    ('reports', 'view'),
    ('consultations', 'accept')
ON CONFLICT (resource, action) DO NOTHING;

-- 2. Seed three new dynamic roles.

INSERT INTO roles (name, description, full_access) VALUES
    ('Super Admin', 'Unconditional full access, equivalent to Administrator.', true),
    ('Organization Admin', 'Hospital/organization administrator. Global (not facility-scoped) permission grant in this phase.', false),
    ('Specialist', 'Physician-equivalent dynamic role with consultation acceptance.', false);

-- Super Admin: full_access = true bypasses per-permission checks entirely
-- (see Claims.HasPermission), so no role_permissions rows are needed.

-- Organization Admin permission bundle. Deliberately EXCLUDES all roles:*
-- (role/permission management stays Super Admin/Administrator-only) and
-- users:delete (no destructive user management from a facility-level admin).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'Organization Admin'
  AND (p.resource, p.action) IN (
    ('patients', 'create'), ('patients', 'read'), ('patients', 'update'), ('patients', 'delete'),
    ('patients', 'view'), ('patients', 'edit'),
    ('referrals', 'create'), ('referrals', 'read'), ('referrals', 'update'), ('referrals', 'delete'),
    ('referrals', 'accept'),
    ('facilities', 'create'), ('facilities', 'read'), ('facilities', 'update'), ('facilities', 'delete'),
    ('records', 'upload'),
    ('hospitals', 'manage'),
    ('reports', 'view'),
    ('dashboard', 'read'),
    ('users', 'create'), ('users', 'read'), ('users', 'update')
  );

-- Specialist permission bundle. Deliberately EXCLUDES users, roles,
-- facilities, hospitals:manage, reports:view, patients:delete,
-- patients:create, referrals:delete.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'Specialist'
  AND (p.resource, p.action) IN (
    ('patients', 'read'), ('patients', 'update'), ('patients', 'view'), ('patients', 'edit'),
    ('referrals', 'create'), ('referrals', 'read'), ('referrals', 'update'), ('referrals', 'accept'),
    ('records', 'upload'),
    ('dashboard', 'read'),
    ('consultations', 'accept')
  );
