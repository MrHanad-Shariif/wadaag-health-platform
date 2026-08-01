package identity

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

type Handler struct {
	service *Service
	tm      *platform.TokenManager
}

func NewHandler(service *Service, tm *platform.TokenManager) *Handler {
	return &Handler{service: service, tm: tm}
}

// Routes mounts identity's endpoints. Register/Login are intentionally
// public; Me demonstrates the RequireAuth chain every other module's
// protected routes will also use.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.register)
	r.Post("/login", h.login)

	r.Group(func(protected chi.Router) {
		protected.Use(platform.RequireAuth(h.tm))
		protected.Get("/me", h.me)
	})

	return r
}

type registerRequest struct {
	Email    *string       `json:"email"`
	Phone    *string       `json:"phone"`
	Password string        `json:"password"`
	Role     platform.Role `json:"role"`
}

type userResponse struct {
	ID     string        `json:"id"`
	Email  *string       `json:"email,omitempty"`
	Phone  *string       `json:"phone,omitempty"`
	Role   platform.Role `json:"role"`
	Status string        `json:"status"`
}

func toUserResponse(u User) userResponse {
	return userResponse{
		ID:     u.ID.String(),
		Email:  u.Email,
		Phone:  u.Phone,
		Role:   u.Role,
		Status: u.Status,
	}
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.Register(r.Context(), RegisterInput{
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
		Role:     req.Role,
	})
	if err != nil {
		if errors.Is(err, ErrDuplicateUser) {
			platform.WriteError(w, http.StatusConflict, err.Error())
			return
		}
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	platform.WriteJSON(w, http.StatusCreated, toUserResponse(user))
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

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, tokens, err := h.service.Login(r.Context(), req.Identifier, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			platform.WriteError(w, http.StatusUnauthorized, err.Error())
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "login failed")
		return
	}

	platform.WriteJSON(w, http.StatusOK, loginResponse{
		User:         toUserResponse(user),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
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

	platform.WriteJSON(w, http.StatusOK, toUserResponse(user))
}
