package authz

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

type Handler struct {
	service *Service
	tm      *platform.TokenManager
}

func NewHandler(service *Service, tm *platform.TokenManager) *Handler {
	return &Handler{service: service, tm: tm}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(platform.RequireAuth(h.tm))

	r.Group(func(g chi.Router) {
		g.Use(platform.RequirePermission("roles", "read"))
		g.Get("/permissions", h.listPermissions)
		g.Get("/roles", h.listRoles)
		g.Get("/roles/{roleID}", h.getRole)
	})
	r.Group(func(g chi.Router) {
		g.Use(platform.RequirePermission("roles", "create"))
		g.Post("/roles", h.createRole)
	})
	r.Group(func(g chi.Router) {
		g.Use(platform.RequirePermission("roles", "update"))
		g.Put("/roles/{roleID}", h.updateRole)
	})
	r.Group(func(g chi.Router) {
		g.Use(platform.RequirePermission("roles", "delete"))
		g.Delete("/roles/{roleID}", h.deleteRole)
	})

	r.Group(func(g chi.Router) {
		g.Use(platform.RequirePermission("users", "read"))
		g.Get("/users", h.listUsers)
		g.Get("/users/{userID}", h.getUser)
	})
	r.Group(func(g chi.Router) {
		g.Use(platform.RequirePermission("users", "create"))
		g.Post("/users", h.createUser)
	})
	r.Group(func(g chi.Router) {
		g.Use(platform.RequirePermission("users", "update"))
		g.Patch("/users/{userID}/role", h.updateUserRole)
		g.Patch("/users/{userID}/status", h.updateUserStatus)
	})

	return r
}

type permissionResponse struct {
	ID       string `json:"id"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

func toPermissionResponse(p Permission) permissionResponse {
	return permissionResponse{ID: p.ID.String(), Resource: p.Resource, Action: p.Action}
}

func (h *Handler) listPermissions(w http.ResponseWriter, r *http.Request) {
	permissions, err := h.service.ListPermissionCatalog(r.Context())
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list permissions")
		return
	}
	out := make([]permissionResponse, len(permissions))
	for i, p := range permissions {
		out[i] = toPermissionResponse(p)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

type roleResponse struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description *string              `json:"description,omitempty"`
	FullAccess  bool                 `json:"full_access"`
	CreatedAt   string               `json:"created_at"`
	Permissions []permissionResponse `json:"permissions,omitempty"`
}

func toRoleResponse(r RoleWithPermissions) roleResponse {
	perms := make([]permissionResponse, len(r.Permissions))
	for i, p := range r.Permissions {
		perms[i] = toPermissionResponse(p)
	}
	return roleResponse{
		ID: r.ID.String(), Name: r.Name, Description: r.Description, FullAccess: r.FullAccess,
		CreatedAt: r.CreatedAt.Format(time.RFC3339), Permissions: perms,
	}
}

type roleRequest struct {
	Name          string   `json:"name"`
	Description   *string  `json:"description"`
	FullAccess    bool     `json:"full_access"`
	PermissionIDs []string `json:"permission_ids"`
}

func (req roleRequest) toInput() (RoleInput, error) {
	ids := make([]uuid.UUID, len(req.PermissionIDs))
	for i, s := range req.PermissionIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			return RoleInput{}, err
		}
		ids[i] = id
	}
	return RoleInput{Name: req.Name, Description: req.Description, FullAccess: req.FullAccess, PermissionIDs: ids}, nil
}

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	var req roleRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input, err := req.toInput()
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid permission id")
		return
	}

	role, err := h.service.CreateRole(r.Context(), input)
	if err != nil {
		if errors.Is(err, ErrDuplicateRole) {
			platform.WriteError(w, http.StatusConflict, err.Error())
			return
		}
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	platform.WriteJSON(w, http.StatusCreated, toRoleResponse(role))
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.service.ListRoles(r.Context())
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list roles")
		return
	}
	out := make([]roleResponse, len(roles))
	for i, role := range roles {
		out[i] = toRoleResponse(RoleWithPermissions{Role: role})
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) getRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "roleID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid role id")
		return
	}
	role, err := h.service.GetRole(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			platform.WriteError(w, http.StatusNotFound, "role not found")
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to load role")
		return
	}
	platform.WriteJSON(w, http.StatusOK, toRoleResponse(role))
}

func (h *Handler) updateRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "roleID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid role id")
		return
	}
	var req roleRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input, err := req.toInput()
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid permission id")
		return
	}

	role, err := h.service.UpdateRole(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			platform.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, ErrDuplicateRole) {
			platform.WriteError(w, http.StatusConflict, err.Error())
			return
		}
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	platform.WriteJSON(w, http.StatusOK, toRoleResponse(role))
}

func (h *Handler) deleteRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "roleID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid role id")
		return
	}
	if err := h.service.DeleteRole(r.Context(), id); err != nil {
		if errors.Is(err, ErrRoleInUse) {
			platform.WriteError(w, http.StatusConflict, err.Error())
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to delete role")
		return
	}
	platform.WriteJSON(w, http.StatusNoContent, nil)
}

type userResponse struct {
	ID         string        `json:"id"`
	Email      *string       `json:"email,omitempty"`
	Phone      *string       `json:"phone,omitempty"`
	LegacyRole platform.Role `json:"legacy_role"`
	Status     string        `json:"status"`
	CreatedAt  string        `json:"created_at"`
	RoleID     *string       `json:"role_id,omitempty"`
	RoleName   *string       `json:"role_name,omitempty"`
}

func toUserResponse(u User) userResponse {
	var roleID *string
	if u.RoleID != nil {
		s := u.RoleID.String()
		roleID = &s
	}
	return userResponse{
		ID: u.ID.String(), Email: u.Email, Phone: u.Phone, LegacyRole: u.LegacyRole, Status: u.Status,
		CreatedAt: u.CreatedAt.Format(time.RFC3339), RoleID: roleID, RoleName: u.RoleName,
	}
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListUsers(r.Context())
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	out := make([]userResponse, len(users))
	for i, u := range users {
		out[i] = toUserResponse(u)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	user, err := h.service.GetUser(r.Context(), id)
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

type createUserRequest struct {
	Email         *string       `json:"email"`
	Phone         *string       `json:"phone"`
	Password      string        `json:"password"`
	LegacyRole    platform.Role `json:"legacy_role"`
	RoleID        *string       `json:"role_id"`
	FacilityID    *string       `json:"facility_id"`
	Specialty     *string       `json:"specialty"`
	LicenseNumber *string       `json:"license_number"`
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := CreateUserInput{
		Email: req.Email, Phone: req.Phone, Password: req.Password, LegacyRole: req.LegacyRole,
		Specialty: req.Specialty, LicenseNumber: req.LicenseNumber,
	}
	if req.RoleID != nil {
		id, err := uuid.Parse(*req.RoleID)
		if err != nil {
			platform.WriteError(w, http.StatusBadRequest, "invalid role_id")
			return
		}
		input.RoleID = &id
	}
	if req.FacilityID != nil {
		id, err := uuid.Parse(*req.FacilityID)
		if err != nil {
			platform.WriteError(w, http.StatusBadRequest, "invalid facility_id")
			return
		}
		input.FacilityID = &id
	}

	user, err := h.service.CreateUser(r.Context(), input)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	platform.WriteJSON(w, http.StatusCreated, toUserResponse(user))
}

type setRoleRequest struct {
	RoleID *string `json:"role_id"`
}

func (h *Handler) updateUserRole(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var req setRoleRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var roleID *uuid.UUID
	if req.RoleID != nil {
		id, err := uuid.Parse(*req.RoleID)
		if err != nil {
			platform.WriteError(w, http.StatusBadRequest, "invalid role_id")
			return
		}
		roleID = &id
	}

	user, err := h.service.SetUserRole(r.Context(), userID, roleID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to update user role")
		return
	}
	platform.WriteJSON(w, http.StatusOK, toUserResponse(user))
}

type setStatusRequest struct {
	Status string `json:"status"`
}

func (h *Handler) updateUserStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var req setStatusRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.SetUserStatus(r.Context(), userID, req.Status)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to update user status")
		return
	}
	platform.WriteJSON(w, http.StatusOK, toUserResponse(user))
}
