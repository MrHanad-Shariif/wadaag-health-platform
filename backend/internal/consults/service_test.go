package consults

import (
	"testing"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

// TestIsParty covers the pure "who may access this consultation" rule:
// system_admin unconditionally, the requesting provider, the target
// provider — and nobody else, even a real provider unrelated to this
// particular consultation.
func TestIsParty(t *testing.T) {
	requesting := uuid.New()
	target := uuid.New()
	stranger := uuid.New()
	c := Consultation{RequestingProviderID: requesting, TargetProviderID: target}

	tests := []struct {
		name            string
		actorProviderID uuid.UUID
		actorRole       platform.Role
		want            bool
	}{
		{"requesting provider is a party", requesting, platform.RolePhysician, true},
		{"target provider is a party", target, platform.RolePhysician, true},
		{"unrelated provider is not a party", stranger, platform.RolePhysician, false},
		{"system_admin bypasses the check even with an unrelated provider id", stranger, platform.RoleSystemAdmin, true},
		{"system_admin bypasses the check with the zero-value provider id", uuid.UUID{}, platform.RoleSystemAdmin, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isParty(c, tt.actorProviderID, tt.actorRole); got != tt.want {
				t.Errorf("isParty() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestShouldRespond covers the requested -> responded transition rule: only
// the target provider's message counts as "the response" — the requester
// following up on their own request must not flip the status.
func TestShouldRespond(t *testing.T) {
	requesting := uuid.New()
	target := uuid.New()
	c := Consultation{RequestingProviderID: requesting, TargetProviderID: target}

	tests := []struct {
		name             string
		senderProviderID uuid.UUID
		want             bool
	}{
		{"target provider's message triggers the transition", target, true},
		{"requesting provider's own follow-up does not", requesting, false},
		{"an unrelated provider id does not", uuid.New(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRespond(c, tt.senderProviderID); got != tt.want {
				t.Errorf("shouldRespond() = %v, want %v", got, tt.want)
			}
		})
	}
}
