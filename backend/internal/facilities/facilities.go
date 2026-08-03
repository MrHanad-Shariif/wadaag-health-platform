package facilities

import (
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeHospital Type = "hospital"
	TypeClinic   Type = "clinic"
	TypeLab      Type = "lab"
	TypePharmacy Type = "pharmacy"
	TypeInsurer  Type = "insurer"
)

type Facility struct {
	ID        uuid.UUID
	Name      string
	Type      Type
	Region    *string
	District  *string
	Phone     *string
	Address   *string
	CreatedAt time.Time
	UpdatedAt time.Time

	// LogoURL and WorkingHours are optional facility branding/scheduling
	// fields — see the working_hours comment in
	// db/migrations/0012_extend_facilities.up.sql for the expected JSON
	// shape. WorkingHours is raw JSON passed straight through to/from the
	// jsonb column (same pattern as records.ClinicalObservation.Payload); a
	// nil slice represents NULL (unset).
	LogoURL      *string
	WorkingHours []byte

	// ReferralPolicies is free-text describing a facility's referral intake
	// rules/policies (e.g. what specialties it accepts, hours it reviews
	// referrals) — plain text rather than structured JSON since this is
	// meant to be admin-authored prose, not machine-parsed. Nil means unset.
	ReferralPolicies *string
}

// Branch is a physical location belonging to a facility (e.g. a hospital's
// satellite clinic).
type Branch struct {
	ID         uuid.UUID
	FacilityID uuid.UUID
	Name       string
	Address    *string
	Phone      *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Department is an organizational unit within a facility (e.g. "Pediatrics").
type Department struct {
	ID         uuid.UUID
	FacilityID uuid.UUID
	Name       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Provider struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	FacilityID    uuid.UUID
	Specialty     *string
	LicenseNumber *string
	CreatedAt     time.Time
	UpdatedAt     time.Time

	// Qualifications, YearsExperience, ConsultationFee, Languages,
	// Certificates, and AreasOfExpertise are self-editable provider profile
	// fields — see Service.UpdateOwnProviderProfile. VerificationStatus is
	// deliberately not settable through that path; it only changes via
	// Service.UpdateProviderVerificationStatus (admin-only).
	//
	// ConsultationFee is a decimal string (e.g. "125.50"), following the
	// same NUMERIC-as-string convention as platform.PgNumeric/FromPgNumeric
	// rather than a float, to avoid floating-point rounding on money.
	//
	// Certificates is raw JSON passed straight through to/from the jsonb
	// column (same pattern as Facility.WorkingHours); a nil slice
	// represents NULL (unset). See the certificates comment in
	// db/migrations/0014_extend_providers.up.sql for the expected shape.
	Qualifications     []string
	YearsExperience    *int
	ConsultationFee    *string
	VerificationStatus string
	Languages          []string
	Certificates       []byte
	AreasOfExpertise   []string
}

// ProviderSearchResult is a directory-style search hit — a thin projection
// of Provider plus the linked user's full_name (joined in specifically for
// search since the base Provider type has no view onto its own user's
// profile fields). See Service.SearchProviders.
type ProviderSearchResult struct {
	ProviderID   uuid.UUID
	FacilityID   uuid.UUID
	UserFullName *string
	Specialty    *string
	CreatedAt    time.Time
}
