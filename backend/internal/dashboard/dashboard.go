// Package dashboard composes read-only counts from the other modules into
// one summary for the admin overview — it owns no data of its own.
package dashboard

import "context"

type Summary struct {
	TotalPatients     int64
	TotalEncounters   int64
	TotalFacilities   int64
	TotalUsers        int64
	ReferralsByStatus map[string]int64
}

type PatientCounter interface {
	CountPatients(ctx context.Context) (int64, error)
}

type EncounterCounter interface {
	CountEncounters(ctx context.Context) (int64, error)
}

type FacilityCounter interface {
	CountFacilities(ctx context.Context) (int64, error)
}

type UserCounter interface {
	CountUsers(ctx context.Context) (int64, error)
}

type ReferralCounter interface {
	CountByStatus(ctx context.Context) (map[string]int64, error)
}
