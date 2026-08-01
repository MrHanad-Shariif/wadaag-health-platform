package consent

import (
	"time"

	"github.com/google/uuid"
)

type GranteeType string

const (
	GranteeTypeProvider GranteeType = "provider"
	GranteeTypeFacility GranteeType = "facility"
)

type Scope string

const (
	ScopeFullRecord   Scope = "full_record"
	ScopeReferralOnly Scope = "referral_only"
)

type Status string

const (
	StatusActive  Status = "active"
	StatusRevoked Status = "revoked"
)

type GrantedVia string

const (
	GrantedViaPatientExplicit          GrantedVia = "patient_explicit"
	GrantedViaReferralImpliedTemporary GrantedVia = "referral_implied_temporary"
	GrantedViaProviderCreated          GrantedVia = "provider_created"
)

type Grant struct {
	ID          uuid.UUID
	PatientID   uuid.UUID
	GranteeType GranteeType
	GranteeID   uuid.UUID
	Scope       Scope
	ScopeRef    *uuid.UUID
	Status      Status
	GrantedVia  GrantedVia
	GrantedAt   time.Time
	RevokedAt   *time.Time
	ExpiresAt   *time.Time
}
