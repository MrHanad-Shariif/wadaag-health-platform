package platform

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Role is shared across every module that needs to gate access by who is
// calling — identity owns issuing it, everyone else only reads it.
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
	UserID     uuid.UUID  `json:"sub"`
	Role       Role       `json:"role"`
	FacilityID *uuid.UUID `json:"facility_id,omitempty"`
	jwt.RegisteredClaims
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

func (m *TokenManager) IssueAccessToken(userID uuid.UUID, role Role, facilityID *uuid.UUID) (string, error) {
	return m.issue(userID, role, facilityID, m.accessTokenTTL)
}

func (m *TokenManager) IssueRefreshToken(userID uuid.UUID, role Role, facilityID *uuid.UUID) (string, error) {
	return m.issue(userID, role, facilityID, m.refreshTokenTTL)
}

func (m *TokenManager) issue(userID uuid.UUID, role Role, facilityID *uuid.UUID, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:     userID,
		Role:       role,
		FacilityID: facilityID,
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

// RequireRoles gates a route to a fixed set of roles. It must run after
// RequireAuth so Claims are already present in the context.
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

			if !allowed[claims.Role] {
				WriteError(w, http.StatusForbidden, "role not permitted for this route")
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
