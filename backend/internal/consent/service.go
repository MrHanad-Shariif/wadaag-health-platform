package consent

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

// Checker is the single source of truth for "can this actor see this
// patient's data". Every route that reads or writes patient-scoped
// resources must go through HasAccess — see consent.Middleware.
type Checker struct {
	repo *Repository
}

func NewChecker(repo *Repository) *Checker {
	return &Checker{repo: repo}
}

// HasAccess implements the access rules in order of precedence:
//  1. system_admin always has access (still logged by the caller, not a
//     silent bypass — see the plan's "no unaudited internal access" rule).
//  2. the patient viewing their own record.
//  3. an active, unexpired consent grant naming the actor's user_id or
//     their facility_id as grantee.
func (c *Checker) HasAccess(ctx context.Context, actorUserID uuid.UUID, actorRole platform.Role, actorFacilityID *uuid.UUID, patientUserID *uuid.UUID, patientID uuid.UUID) (bool, error) {
	if actorRole == platform.RoleSystemAdmin {
		return true, nil
	}

	if actorRole == platform.RolePatient && patientUserID != nil && *patientUserID == actorUserID {
		return true, nil
	}

	facilityID := uuid.UUID{}
	if actorFacilityID != nil {
		facilityID = *actorFacilityID
	}

	return c.repo.FindActiveGrant(ctx, patientID, actorUserID, facilityID)
}

type GrantInput struct {
	PatientID  uuid.UUID
	Grantee    GranteeType
	GranteeID  uuid.UUID
	Scope      Scope
	ScopeRef   *uuid.UUID
	GrantedVia GrantedVia
	ExpiresAt  *time.Time
}

func (c *Checker) Grant(ctx context.Context, in GrantInput) (Grant, error) {
	return c.repo.CreateGrant(ctx, CreateGrantParams{
		PatientID: in.PatientID, Grantee: in.Grantee, GranteeID: in.GranteeID,
		Scope: in.Scope, ScopeRef: in.ScopeRef, GrantedVia: in.GrantedVia, ExpiresAt: in.ExpiresAt,
	})
}

func (c *Checker) Revoke(ctx context.Context, grantID, patientID uuid.UUID) (Grant, error) {
	return c.repo.Revoke(ctx, grantID, patientID)
}

func (c *Checker) ListForPatient(ctx context.Context, patientID uuid.UUID) ([]Grant, error) {
	return c.repo.ListForPatient(ctx, patientID)
}
