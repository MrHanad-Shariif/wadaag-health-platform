-- Reverse of 0010_extend_permissions.up.sql.
-- role_permissions rows for these roles cascade-delete via
-- role_permissions.role_id's ON DELETE CASCADE, so deleting the roles is
-- sufficient.
DELETE FROM roles WHERE name IN ('Super Admin', 'Organization Admin', 'Specialist');

-- Only remove the 7 permission pairs newly inserted by the .up migration,
-- not the original 24 seeded by 0006_create_authz.up.sql.
DELETE FROM permissions WHERE (resource, action) IN (
    ('patients', 'view'),
    ('patients', 'edit'),
    ('referrals', 'accept'),
    ('records', 'upload'),
    ('hospitals', 'manage'),
    ('reports', 'view'),
    ('consultations', 'accept')
);
