package platform

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Role is shared across every module that needs to gate access by who is
// calling — identity owns issuing it, everyone else only reads it. This is
// the original fixed-role system and still governs identity/records/
// referrals/facilities. The newer Authentication module (roles table,
// FullAccess/Permissions below) is a separate, additive permission system
// scoped to itself, the patient list, and the dashboard — see the module's
// docs for why the two coexist instead of one replacing the other.
type Role string

const (
	RolePatient       Role = "patient"
	RolePhysician     Role = "physician"
	RoleHospitalAdmin Role = "hospital_admin"
	RoleLabTech       Role = "lab_tech"
	RolePharmacist    Role = "pharmacist"
	RoleInsurer       Role = "insurer"
	RoleSystemAdmin   Role = "system_admin"
)

type Claims struct {
	UserID      uuid.UUID  `json:"sub"`
	Role        Role       `json:"role"`
	FacilityID  *uuid.UUID `json:"facility_id,omitempty"`
	RoleID      *uuid.UUID `json:"role_id,omitempty"`
	FullAccess  bool       `json:"full_access,omitempty"`
	Permissions []string   `json:"permissions,omitempty"`
	jwt.RegisteredClaims
}

// HasPermission checks the dynamic Authentication-module permission set —
// "resource:action" — with FullAccess as an unconditional bypass.
func (c *Claims) HasPermission(resource, action string) bool {
	return c.FullAccess || slices.Contains(c.Permissions, resource+":"+action)
}

// TokenClaims is the input to issuing a token — grouped into a struct
// because the set of things worth embedding (legacy role, facility,
// dynamic permissions) has grown past what reads cleanly as positional
// params.
type TokenClaims struct {
	UserID      uuid.UUID
	Role        Role
	FacilityID  *uuid.UUID
	RoleID      *uuid.UUID
	FullAccess  bool
	Permissions []string
}

// TokenManager issues and verifies the JWTs used across every module's
// protected routes. It is constructed once in main and handed to
// identity (to issue tokens on login) and to the RequireAuth middleware
// (to verify them on every subsequent request).
type TokenManager struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewTokenManager(cfg Config) *TokenManager {
	return &TokenManager{
		secret:          []byte(cfg.JWTSecret),
		accessTokenTTL:  cfg.AccessTokenTTL,
		refreshTokenTTL: cfg.RefreshTokenTTL,
	}
}

func (m *TokenManager) IssueAccessToken(c TokenClaims) (string, error) {
	return m.issue(c, m.accessTokenTTL)
}

// RefreshTokenTTL exposes the configured refresh-token lifetime so callers
// that store their own opaque refresh tokens (see identity.Service) can
// compute an expires_at consistent with the JWT-based TTL used elsewhere.
func (m *TokenManager) RefreshTokenTTL() time.Duration {
	return m.refreshTokenTTL
}

func (m *TokenManager) issue(c TokenClaims, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:      c.UserID,
		Role:        c.Role,
		FacilityID:  c.FacilityID,
		RoleID:      c.RoleID,
		FullAccess:  c.FullAccess,
		Permissions: c.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

var ErrInvalidToken = errors.New("invalid or expired token")

func (m *TokenManager) Verify(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

type contextKey string

const claimsContextKey contextKey = "claims"

// RequireAuth extracts and verifies the Bearer token, then injects the
// resulting Claims into the request context for downstream handlers and
// the consent/audit middleware chain to read.
func RequireAuth(tm *TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				WriteError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

			claims, err := tm.Verify(strings.TrimPrefix(header, "Bearer "))
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRoles gates a route to a fixed set of legacy roles. It must run
// after RequireAuth so Claims are already present in the context.
// RoleSystemAdmin unconditionally bypasses the check — the administrator
// is meant to have full access across every module, not just the ones
// that happen to list system_admin explicitly.
func RequireRoles(roles ...Role) func(http.Handler) http.Handler {
	allowed := make(map[Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				WriteError(w, http.StatusUnauthorized, "missing auth context")
				return
			}

			if claims.Role != RoleSystemAdmin && !allowed[claims.Role] {
				WriteError(w, http.StatusForbidden, "role not permitted for this route")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission gates a route by the dynamic Authentication-module
// permission set instead of the legacy role enum — used by the
// Authentication module itself, the patient list, and the dashboard.
// FullAccess (the administrator role) bypasses every check.
func RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				WriteError(w, http.StatusUnauthorized, "missing auth context")
				return
			}

			if !claims.HasPermission(resource, action) {
				WriteError(w, http.StatusForbidden, "missing required permission")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	return claims, ok
}
