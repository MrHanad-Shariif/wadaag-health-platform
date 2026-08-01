package consent

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/audit"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

// ResourceInfo is what a route needs to resolve before the consent check
// can run: which patient the request is about, and (for the audit trail)
// what specific resource is being touched.
type ResourceInfo struct {
	PatientID     uuid.UUID
	PatientUserID *uuid.UUID
	ResourceType  string
	ResourceID    *uuid.UUID
}

// Resolver extracts ResourceInfo from a request — e.g. reading a chi URL
// param and looking up the owning patient_id. It's supplied by whichever
// module owns the route (records, referrals, ...), since only that module
// knows its own URL shape and resource lookups.
type Resolver func(r *http.Request) (ResourceInfo, error)

// Middleware is the structural guarantee behind "every access is logged":
// it runs after RequireAuth/RequireRoles and before the handler, and on
// every path — allowed or denied — it writes exactly one audit_log entry.
func Middleware(checker *Checker, logger *audit.Logger, action audit.Action, resolve Resolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := platform.ClaimsFromContext(r.Context())
			if !ok {
				platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
				return
			}

			info, err := resolve(r)
			if err != nil {
				platform.WriteError(w, http.StatusBadRequest, "could not resolve resource for consent check")
				return
			}

			allowed, err := checker.HasAccess(r.Context(), claims.UserID, claims.Role, claims.FacilityID, info.PatientUserID, info.PatientID)
			if err != nil {
				platform.WriteError(w, http.StatusInternalServerError, "consent check failed")
				return
			}

			result := audit.ResultDenied
			if allowed {
				result = audit.ResultAllowed
			}

			logger.Record(r.Context(), audit.RecordInput{
				ActorUserID: claims.UserID, ActorRole: claims.Role, Action: action,
				ResourceType: info.ResourceType, ResourceID: info.ResourceID,
				PatientID: &info.PatientID, Result: result,
			})

			if !allowed {
				platform.WriteError(w, http.StatusForbidden, "not authorized to access this patient's data")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
