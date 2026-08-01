DROP TRIGGER IF EXISTS audit_log_no_delete ON audit_log;
DROP TRIGGER IF EXISTS audit_log_no_update ON audit_log;
DROP FUNCTION IF EXISTS prevent_audit_log_mutation();

DROP TABLE IF EXISTS audit_log;
DROP TYPE IF EXISTS audit_result;

DROP TABLE IF EXISTS consent_grants;
DROP TYPE IF EXISTS consent_granted_via;
DROP TYPE IF EXISTS consent_status;
DROP TYPE IF EXISTS consent_scope;
DROP TYPE IF EXISTS consent_grantee_type;
