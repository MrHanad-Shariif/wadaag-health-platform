package records

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestResolveMedicalHistory covers Service.GetMedicalHistory's "never 404"
// contract: a patient with no patient_medical_history row yet
// (ErrMedicalHistoryNotFound from the repository) must come back as a
// zero-value PatientMedicalHistory with every jsonb field defaulted to an
// empty JSON array, not as an error — mirroring identity's
// TestMergeProfileUpdate pattern of testing the pure decision function
// directly rather than wiring a fake repository or a live database.
func TestResolveMedicalHistory(t *testing.T) {
	patientID := uuid.New()

	t.Run("not-found repository error returns zero-value history, not an error", func(t *testing.T) {
		got, err := resolveMedicalHistory(patientID, PatientMedicalHistory{}, ErrMedicalHistoryNotFound)
		if err != nil {
			t.Fatalf("resolveMedicalHistory() error = %v, want nil", err)
		}

		want := PatientMedicalHistory{
			PatientID:          patientID,
			Allergies:          emptyJSONArray,
			ChronicConditions:  emptyJSONArray,
			CurrentMedications: emptyJSONArray,
			PastSurgeries:      emptyJSONArray,
			FamilyHistory:      emptyJSONArray,
			VaccinationHistory: emptyJSONArray,
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("resolveMedicalHistory() = %+v, want %+v", got, want)
		}
	})

	t.Run("an existing row is passed through unchanged", func(t *testing.T) {
		existing := PatientMedicalHistory{
			PatientID:          patientID,
			Allergies:          []byte(`["Penicillin"]`),
			ChronicConditions:  []byte(`["Asthma"]`),
			CurrentMedications: emptyJSONArray,
			PastSurgeries:      emptyJSONArray,
			FamilyHistory:      emptyJSONArray,
			VaccinationHistory: emptyJSONArray,
			UpdatedAt:          time.Now(),
		}

		got, err := resolveMedicalHistory(patientID, existing, nil)
		if err != nil {
			t.Fatalf("resolveMedicalHistory() error = %v, want nil", err)
		}
		if !reflect.DeepEqual(got, existing) {
			t.Errorf("resolveMedicalHistory() = %+v, want %+v (unchanged)", got, existing)
		}
	})

	t.Run("an unexpected repository error is propagated, not swallowed", func(t *testing.T) {
		unexpected := errors.New("db exploded")

		_, err := resolveMedicalHistory(patientID, PatientMedicalHistory{}, unexpected)
		if err != unexpected {
			t.Errorf("resolveMedicalHistory() error = %v, want %v", err, unexpected)
		}
	})
}
