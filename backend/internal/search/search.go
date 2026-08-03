// Package search fans a single free-text query out across the five
// directory/record facets the platform supports (patient, doctor, hospital,
// referral, consultation), reusing each owning module's existing
// access-scoping rather than inventing a new unscoped view — see
// Service.Search for the per-facet rules.
//
// Deliberately out of scope for this package: full-text search over
// records.ClinicalObservation's jsonb payloads ("disease"/"medical record"
// search) — that's a real full-text-over-jsonb design problem on its own,
// not something to bolt on here.
package search

import (
	"time"

	"github.com/google/uuid"
)

// Facet names — also the JSON keys grouping results in the GET /search
// response and the values accepted in the ?type= query parameter.
const (
	FacetPatient      = "patient"
	FacetDoctor       = "doctor"
	FacetHospital     = "hospital"
	FacetReferral     = "referral"
	FacetConsultation = "consultation"
)

// allFacets is every facet Search knows how to produce, in a fixed
// evaluation order.
var allFacets = []string{FacetPatient, FacetDoctor, FacetHospital, FacetReferral, FacetConsultation}

// Result is the single shape every facet's rows are mapped into before
// being grouped by Type in the handler's response.
type Result struct {
	Type      string
	ID        uuid.UUID
	Title     string
	Subtitle  string
	CreatedAt time.Time
}
