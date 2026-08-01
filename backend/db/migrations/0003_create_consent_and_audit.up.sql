CREATE TYPE consent_grantee_type AS ENUM ('provider', 'facility');
CREATE TYPE consent_scope AS ENUM ('full_record', 'referral_only');
CREATE TYPE consent_status AS ENUM ('active', 'revoked');
CREATE TYPE consent_granted_via AS ENUM ('patient_explicit', 'referral_implied_temporary', 'provider_created');

CREATE TABLE consent_grants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL,
    grantee_type consent_grantee_type NOT NULL,
    grantee_id UUID NOT NULL,
    scope consent_scope NOT NULL,
    scope_ref UUID,
    status consent_status NOT NULL DEFAULT 'active',
    granted_via consent_granted_via NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ
);

CREATE INDEX idx_consent_grants_patient ON consent_grants (patient_id);
CREATE INDEX idx_consent_grants_grantee ON consent_grants (grantee_type, grantee_id);

CREATE TYPE audit_result AS ENUM ('allowed', 'denied');

CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID NOT NULL,
    actor_role user_role NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id UUID,
    patient_id UUID,
    result audit_result NOT NULL,
    metadata JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_log_patient ON audit_log (patient_id);
CREATE INDEX idx_audit_log_actor ON audit_log (actor_user_id);

-- audit_log is insert-only: this is the platform's access-trust guarantee,
-- so it's enforced at the database level, not just by application code.
CREATE FUNCTION prevent_audit_log_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_log is insert-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_log_no_update
    BEFORE UPDATE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_log_mutation();

CREATE TRIGGER audit_log_no_delete
    BEFORE DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_log_mutation();
