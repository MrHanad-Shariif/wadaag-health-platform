package identity

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

type Handler struct {
	service *Service
	tm      *platform.TokenManager
	lockout *platform.LoginLockout
}

func NewHandler(service *Service, tm *platform.TokenManager, lockout *platform.LoginLockout) *Handler {
	return &Handler{service: service, tm: tm, lockout: lockout}
}

// Routes mounts identity's endpoints. Register/Login are intentionally
// public; Me demonstrates the RequireAuth chain every other module's
// protected routes will also use.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.register)
	r.Post("/login", h.login)
	r.Post("/refresh", h.refresh)
	r.Post("/logout", h.logout)
	r.Get("/verify-email", h.verifyEmail)
	r.Post("/forgot-password", h.forgotPassword)
	r.Post("/reset-password", h.resetPassword)

	r.Group(func(protected chi.Router) {
		protected.Use(platform.RequireAuth(h.tm))
		protected.Get("/me", h.me)
		protected.Patch("/me", h.updateMe)
		protected.Patch("/me/password", h.changePassword)
		protected.Get("/sessions", h.listSessions)
		protected.Delete("/sessions/{id}", h.revokeSession)
		protected.Post("/send-verification", h.sendVerification)
		protected.Get("/me/profile", h.getProfile)
		protected.Patch("/me/profile", h.updateProfile)
		protected.Post("/me/profile/photo", h.uploadProfilePhoto)
		protected.Get("/users/{userID}", h.getPublicIdentity)
	})

	return r
}

// sessionMetadata reads the optional client context recorded alongside a
// refresh token. IP prefers X-Forwarded-For (set by a reverse proxy) and
// falls back to the raw connection address.
func sessionMetadata(r *http.Request) SessionMetadata {
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = fwd
	}

	var ipPtr, uaPtr *string
	if ip != "" {
		ipPtr = &ip
	}
	if ua := r.Header.Get("User-Agent"); ua != "" {
		uaPtr = &ua
	}

	return SessionMetadata{IP: ipPtr, UserAgent: uaPtr}
}

type registerRequest struct {
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
	Password string  `json:"password"`
	FullName string  `json:"full_name"`
}

type userResponse struct {
	ID          string        `json:"id"`
	Email       *string       `json:"email,omitempty"`
	Phone       *string       `json:"phone,omitempty"`
	Role        platform.Role `json:"role"`
	Status      string        `json:"status"`
	RoleID      *string       `json:"role_id,omitempty"`
	FacilityID  *string       `json:"facility_id,omitempty"`
	FullAccess  bool          `json:"full_access"`
	Permissions []string      `json:"permissions,omitempty"`
	Verified    bool          `json:"verified"`
	VerifiedAt  *time.Time    `json:"verified_at,omitempty"`
	FullName    *string       `json:"full_name,omitempty"`
}

func toUserResponse(u User, access Access) userResponse {
	var roleID *string
	if access.RoleID != nil {
		s := access.RoleID.String()
		roleID = &s
	}
	var facilityID *string
	if access.FacilityID != nil {
		s := access.FacilityID.String()
		facilityID = &s
	}
	return userResponse{
		ID: u.ID.String(), Email: u.Email, Phone: u.Phone, Role: u.Role, Status: u.Status,
		RoleID: roleID, FacilityID: facilityID, FullAccess: access.FullAccess, Permissions: access.Permissions,
		Verified: u.VerifiedAt != nil, VerifiedAt: u.VerifiedAt, FullName: u.FullName,
	}
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.FullName) == "" {
		platform.WriteError(w, http.StatusBadRequest, "full_name is required")
		return
	}

	user, err := h.service.Register(r.Context(), RegisterInput{
		Email: req.Email, Phone: req.Phone, Password: req.Password, FullName: req.FullName,
	})
	if err != nil {
		if errors.Is(err, ErrDuplicateUser) {
			platform.WriteError(w, http.StatusConflict, err.Error())
			return
		}
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	platform.WriteJSON(w, http.StatusCreated, toUserResponse(user, Access{}))
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type loginResponse struct {
	User         userResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

// login is guarded by h.lockout, keyed on identifier+IP (see
// loginLockoutKey): once that pair has racked up
// platform.defaultFailureThreshold consecutive failures, further attempts
// are rejected with 429 before the service layer — and therefore the
// bcrypt password compare — ever runs. 429 (rather than 423 Locked) is used
// here to match the general per-IP rate limiter's response shape and
// status code, since from the caller's point of view both are "you're
// being throttled, back off and retry later" and most HTTP clients already
// know how to handle 429 without special-casing a less common code.
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	key := loginLockoutKey(req.Identifier, r)
	if h.lockout.IsLocked(key) {
		platform.WriteError(w, http.StatusTooManyRequests, "too many failed login attempts, try again later")
		return
	}

	user, tokens, access, err := h.service.Login(r.Context(), req.Identifier, req.Password, sessionMetadata(r))
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			h.lockout.RecordFailure(key)
			platform.WriteError(w, http.StatusUnauthorized, err.Error())
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "login failed")
		return
	}

	h.lockout.RecordSuccess(key)

	platform.WriteJSON(w, http.StatusOK, loginResponse{
		User:         toUserResponse(user, access),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

// loginLockoutKey combines the presented identifier with the caller's IP
// (platform.ClientIP, the same extraction the router's general rate
// limiter uses) so a lockout only affects the specific identifier+IP pair
// that's failing, not the identifier from every IP or the IP across every
// identifier.
func loginLockoutKey(identifier string, r *http.Request) string {
	return identifier + "|" + platform.ClientIP(r)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// refresh rotates a presented refresh token for a new access/refresh pair.
// Public — the refresh token itself is the credential, no bearer required.
func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := platform.DecodeJSON(r, &req); err != nil || req.RefreshToken == "" {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tokens, _, err := h.service.Refresh(r.Context(), req.RefreshToken, sessionMetadata(r))
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			platform.WriteError(w, http.StatusUnauthorized, err.Error())
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "refresh failed")
		return
	}

	platform.WriteJSON(w, http.StatusOK, refreshResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// logout revokes the presented refresh token. It always returns 204 for a
// syntactically valid request, whether or not the token existed, so it
// can't be used to probe for valid tokens. Public — matches refresh's
// shape, the token itself is the credential.
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := platform.DecodeJSON(r, &req); err != nil || req.RefreshToken == "" {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.Logout(r.Context(), req.RefreshToken); err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "logout failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	user, err := h.service.GetByID(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			platform.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	platform.WriteJSON(w, http.StatusOK, toUserResponse(user, Access{
		RoleID: claims.RoleID, FullAccess: claims.FullAccess, Permissions: claims.Permissions,
		FacilityID: claims.FacilityID,
	}))
}

type updateMeRequest struct {
	FullName string `json:"full_name"`
}

// updateMe lets the caller change their own account-identity display name.
// Distinct from PATCH /me/profile, which edits user_profiles (bio/photo/
// languages/availability) — full_name lives on the users table itself.
func (h *Handler) updateMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	var req updateMeRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.FullName) == "" {
		platform.WriteError(w, http.StatusBadRequest, "full_name is required")
		return
	}

	user, err := h.service.UpdateFullName(r.Context(), claims.UserID, req.FullName)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			platform.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	platform.WriteJSON(w, http.StatusOK, toUserResponse(user, Access{
		RoleID: claims.RoleID, FullAccess: claims.FullAccess, Permissions: claims.Permissions,
		FacilityID: claims.FacilityID,
	}))
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// changePassword lets a logged-in user change their own password, proving
// they know the current one first (see Service.ChangePassword). Distinct
// from the forgot/reset-password flow, which is for a locked-out user with
// no valid session at all. A wrong current password is a 400 (a request
// validation failure, not an auth failure at the request level — the
// caller is already authenticated via their bearer token) rather than 401.
// Every other one of the caller's sessions is revoked on success, same as
// ResetPassword, so 204 is returned rather than a fresh token pair — the
// caller's own current access token remains valid until it naturally
// expires, but any other refresh token they held is now dead.
func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	var req changePasswordRequest
	if err := platform.DecodeJSON(r, &req); err != nil || req.CurrentPassword == "" || req.NewPassword == "" {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.ChangePassword(r.Context(), claims.UserID, req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, ErrIncorrectCurrentPassword) {
			platform.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, ErrUserNotFound) {
			platform.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// publicIdentityResponse is a deliberately minimal "who is this" shape —
// id, full_name, role only. Used to resolve a raw user UUID (shown e.g. in
// a consult or message thread) into a readable name in the UI, without
// exposing email, phone, status, or verification info the way userResponse
// does. Kept as its own type rather than reusing/trimming userResponse so a
// future field added to userResponse doesn't leak here by accident.
type publicIdentityResponse struct {
	ID       string        `json:"id"`
	FullName *string       `json:"full_name,omitempty"`
	Role     platform.Role `json:"role"`
}

// getPublicIdentity returns the minimal public-identity shape for any user
// ID — authenticated (RequireAuth), but not scoped to any particular
// permission, since this is read-only display data, not sensitive account
// data. 404 if the ID doesn't resolve to a real user.
func (h *Handler) getPublicIdentity(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		platform.WriteError(w, http.StatusNotFound, "user not found")
		return
	}

	user, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			platform.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	platform.WriteJSON(w, http.StatusOK, publicIdentityResponse{
		ID: user.ID.String(), FullName: user.FullName, Role: user.Role,
	})
}

type sessionResponse struct {
	ID          string    `json:"id"`
	DeviceLabel *string   `json:"device_label,omitempty"`
	IP          *string   `json:"ip,omitempty"`
	UserAgent   *string   `json:"user_agent,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func toSessionResponse(t RefreshToken) sessionResponse {
	return sessionResponse{
		ID:          t.ID.String(),
		DeviceLabel: t.DeviceLabel,
		IP:          t.IP,
		UserAgent:   t.UserAgent,
		CreatedAt:   t.CreatedAt,
		ExpiresAt:   t.ExpiresAt,
	}
}

// listSessions returns the caller's currently-active sessions (refresh
// tokens that are neither revoked nor expired) — the token_hash is
// deliberately never included in the response.
func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	sessions, err := h.service.ListSessions(r.Context(), claims.UserID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}

	resp := make([]sessionResponse, len(sessions))
	for i, s := range sessions {
		resp[i] = toSessionResponse(s)
	}

	platform.WriteJSON(w, http.StatusOK, resp)
}

// revokeSession revokes one of the caller's own sessions by id. A session
// that doesn't exist, belongs to someone else, or is already revoked all
// produce the same 404 — the caller can't distinguish "not found" from
// "not yours" from another user's session id.
func (h *Handler) revokeSession(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.WriteError(w, http.StatusNotFound, "session not found")
		return
	}

	if err := h.service.RevokeSession(r.Context(), id, claims.UserID); err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			platform.WriteError(w, http.StatusNotFound, "session not found")
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to revoke session")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// sendVerification issues a fresh email verification token for the caller's
// own account and logs it server-side (standing in for an email send — no
// provider is wired up yet). An already-verified account gets a 409, not a
// silent no-op, so the frontend can tell the two cases apart.
func (h *Handler) sendVerification(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	if err := h.service.SendVerificationEmail(r.Context(), claims.UserID); err != nil {
		if errors.Is(err, ErrAlreadyVerified) {
			platform.WriteError(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, ErrUserNotFound) {
			platform.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to send verification")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type verifyEmailResponse struct {
	Verified bool `json:"verified"`
}

// verifyEmail redeems a token presented in the query string — public,
// because the token itself is the credential (matching how a user clicks a
// verification link from an email on a device they aren't logged in on).
func (h *Handler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		platform.WriteError(w, http.StatusBadRequest, "token is required")
		return
	}

	if err := h.service.VerifyEmail(r.Context(), token); err != nil {
		if errors.Is(err, ErrInvalidVerificationToken) {
			platform.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "verification failed")
		return
	}

	platform.WriteJSON(w, http.StatusOK, verifyEmailResponse{Verified: true})
}

type forgotPasswordRequest struct {
	Identifier string `json:"identifier"`
}

// forgotPassword always returns 204 for a syntactically valid request,
// whether or not identifier matches a real account — the response can't be
// used to probe for account existence via status code, timing, or body. If a
// match is found, a reset token is issued and logged server-side (standing
// in for an email send — no provider is wired up yet).
func (h *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := platform.DecodeJSON(r, &req); err != nil || req.Identifier == "" {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.ForgotPassword(r.Context(), req.Identifier); err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to process request")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// resetPassword redeems a presented password reset token and sets a new
// password, revoking every one of the owning user's existing sessions in
// the process. Public — the token itself is the credential, matching
// verifyEmail's shape.
func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := platform.DecodeJSON(r, &req); err != nil || req.Token == "" || req.NewPassword == "" {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Both an invalid/expired/used token and a too-short new password are
	// caller errors — both map to 400 with the underlying message.
	if err := h.service.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
