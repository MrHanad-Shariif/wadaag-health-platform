package identity

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/wadaag/health-platform/backend/internal/platform"
)

type profileResponse struct {
	UserID             string     `json:"user_id"`
	PhotoURL           *string    `json:"photo_url"`
	Bio                *string    `json:"bio"`
	Languages          []string   `json:"languages"`
	AvailabilityStatus *string    `json:"availability_status"`
	IsOnline           bool       `json:"is_online"`
	LastSeenAt         *time.Time `json:"last_seen_at"`
}

func toProfileResponse(p UserProfile) profileResponse {
	languages := p.Languages
	if languages == nil {
		languages = []string{}
	}
	return profileResponse{
		UserID:             p.UserID.String(),
		PhotoURL:           p.PhotoURL,
		Bio:                p.Bio,
		Languages:          languages,
		AvailabilityStatus: p.AvailabilityStatus,
		IsOnline:           p.IsOnline,
		LastSeenAt:         p.LastSeenAt,
	}
}

// getProfile returns the caller's own profile. If no user_profiles row has
// ever been written for them this still returns 200 with zero-values
// (empty bio, empty languages, offline) — never a 404 — matching the
// "created lazily" row lifecycle documented in the migration.
func (h *Handler) getProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	profile, err := h.service.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to load profile")
		return
	}

	platform.WriteJSON(w, http.StatusOK, toProfileResponse(profile))
}

type updateProfileRequest struct {
	Bio                *string   `json:"bio"`
	Languages          *[]string `json:"languages"`
	AvailabilityStatus *string   `json:"availability_status"`
}

// updateProfile is a partial update (upsert-if-missing): any field omitted
// from the request body is left unchanged (or empty, if this is the first
// write) rather than being cleared. See Service.UpdateProfile /
// mergeProfileUpdate.
func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	var req updateProfileRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	profile, err := h.service.UpdateProfile(r.Context(), claims.UserID, UpdateProfileInput{
		Bio: req.Bio, Languages: req.Languages, AvailabilityStatus: req.AvailabilityStatus,
	})
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	platform.WriteJSON(w, http.StatusOK, toProfileResponse(profile))
}

// maxProfilePhotoUploadBody caps the raw request body read for
// POST /me/profile/photo: maxProfilePhotoSize for the file itself plus a
// fixed allowance for multipart boundary/header overhead. The authoritative
// size check is Service.UploadProfilePhoto's exact len(data) comparison;
// this is just a defensive ceiling on how much we'll ever read into memory.
const maxProfilePhotoUploadBody = maxProfilePhotoSize + 1<<20

// uploadProfilePhoto accepts a multipart/form-data body with a single file
// field named "photo". Content-type is validated by magic bytes (not the
// client-declared header) — see Service.detectImageExt — and capped at 5MB;
// either violation is a 400.
func (h *Handler) uploadProfilePhoto(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxProfilePhotoUploadBody)
	if err := r.ParseMultipartForm(maxProfilePhotoUploadBody); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "photo upload too large or malformed")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, _, err := r.FormFile("photo")
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "photo file is required (field name \"photo\")")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "failed to read uploaded photo")
		return
	}

	profile, err := h.service.UploadProfilePhoto(r.Context(), claims.UserID, data)
	if err != nil {
		if errors.Is(err, ErrUnsupportedPhotoType) || errors.Is(err, ErrPhotoTooLarge) {
			platform.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to upload photo")
		return
	}

	platform.WriteJSON(w, http.StatusOK, toProfileResponse(profile))
}
