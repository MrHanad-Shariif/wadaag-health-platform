package identity

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestIsRefreshTokenUsable covers the gate that must reject a stale or
// already-rotated/logged-out refresh token before Refresh proceeds to
// revoke-and-reissue. This is the crux of "rotate-and-revoke-old must not
// let an old token get reused" — a revoked row must never be treated as
// usable again, regardless of its expiry.
func TestIsRefreshTokenUsable(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name   string
		record RefreshToken
		want   bool
	}{
		{
			name:   "valid unexpired unrevoked token is usable",
			record: RefreshToken{ExpiresAt: future, RevokedAt: nil},
			want:   true,
		},
		{
			name:   "expired token is not usable",
			record: RefreshToken{ExpiresAt: past, RevokedAt: nil},
			want:   false,
		},
		{
			name:   "revoked token is not usable even if not yet expired",
			record: RefreshToken{ExpiresAt: future, RevokedAt: &now},
			want:   false,
		},
		{
			name:   "revoked and expired token is not usable",
			record: RefreshToken{ExpiresAt: past, RevokedAt: &now},
			want:   false,
		},
		{
			name:   "token expiring at exactly now is not usable",
			record: RefreshToken{ExpiresAt: now, RevokedAt: nil},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.record.ID = uuid.New()
			got := isRefreshTokenUsable(tt.record, now)
			if got != tt.want {
				t.Errorf("isRefreshTokenUsable() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHashTokenIsDeterministicAndDistinct guards the property Refresh and
// Logout both rely on: hashing the same raw token twice must always find
// the same DB row, and two distinct random tokens must never collide on
// their hash (in practice, by construction of SHA-256).
func TestHashTokenIsDeterministicAndDistinct(t *testing.T) {
	a, err := generateOpaqueToken()
	if err != nil {
		t.Fatalf("generateOpaqueToken() error = %v", err)
	}
	b, err := generateOpaqueToken()
	if err != nil {
		t.Fatalf("generateOpaqueToken() error = %v", err)
	}

	if a == b {
		t.Fatalf("generateOpaqueToken() produced identical tokens across two calls")
	}

	if hashToken(a) != hashToken(a) {
		t.Errorf("hashToken() is not deterministic for the same input")
	}
	if hashToken(a) == hashToken(b) {
		t.Errorf("hashToken() produced the same hash for two distinct tokens")
	}
}
