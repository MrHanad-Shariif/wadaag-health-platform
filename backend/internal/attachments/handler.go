package attachments

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/audit"
	"github.com/wadaag/health-platform/backend/internal/consent"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/records"
)

type Handler struct {
	service *Service
	consent *consent.Checker
	audit   *audit.Logger
	tm      *platform.TokenManager
}

func NewHandler(service *Service, consentChecker *consent.Checker, auditLogger *audit.Logger, tm *platform.TokenManager) *Handler {
	return &Handler{service: service, consent: consentChecker, audit: auditLogger, tm: tm}
}

// Routes is mounted at "/attachments" (see server.NewRouter), so the final
// paths are:
//
//	POST /api/v1/attachments/patients/{patientID}
//	GET  /api/v1/attachments/patients/{patientID}
//	GET  /api/v1/attachments/{attachmentID}
//	GET  /api/v1/attachments/{attachmentID}/versions
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(platform.RequireAuth(h.tm))

	r.Group(func(up chi.Router) {
		up.Use(platform.RequireRoles(platform.RolePhysician, platform.RoleHospitalAdmin))
		up.Use(consent.Middleware(h.consent, h.audit, audit.ActionUploadAttachment, h.resolvePatientRoute))
		up.Post("/patients/{patientID}", h.uploadAttachment)
	})

	r.Group(func(list chi.Router) {
		list.Use(consent.Middleware(h.consent, h.audit, audit.ActionViewAttachment, h.resolvePatientRoute))
		list.Get("/patients/{patientID}", h.listAttachments)
	})

	r.Group(func(a chi.Router) {
		a.Use(consent.Middleware(h.consent, h.audit, audit.ActionViewAttachment, h.resolveAttachmentRoute))
		a.Get("/{attachmentID}", h.downloadAttachment)
		a.Get("/{attachmentID}/versions", h.listVersions)
	})

	return r
}

func (h *Handler) resolvePatientRoute(r *http.Request) (consent.ResourceInfo, error) {
	patientID, err := uuid.Parse(chi.URLParam(r, "patientID"))
	if err != nil {
		return consent.ResourceInfo{}, err
	}
	patientUserID, err := h.service.PatientUserID(r.Context(), patientID)
	if err != nil {
		return consent.ResourceInfo{}, err
	}
	return consent.ResourceInfo{
		PatientID: patientID, PatientUserID: patientUserID,
		ResourceType: "patient", ResourceID: &patientID,
	}, nil
}

func (h *Handler) resolveAttachmentRoute(r *http.Request) (consent.ResourceInfo, error) {
	attachmentID, err := uuid.Parse(chi.URLParam(r, "attachmentID"))
	if err != nil {
		return consent.ResourceInfo{}, err
	}
	attachment, err := h.service.Get(r.Context(), attachmentID)
	if err != nil {
		return consent.ResourceInfo{}, err
	}
	patientUserID, err := h.service.PatientUserID(r.Context(), attachment.PatientID)
	if err != nil {
		return consent.ResourceInfo{}, err
	}
	return consent.ResourceInfo{
		PatientID: attachment.PatientID, PatientUserID: patientUserID,
		ResourceType: "attachment", ResourceID: &attachmentID,
	}, nil
}

type attachmentResponse struct {
	ID          string  `json:"id"`
	PatientID   string  `json:"patient_id"`
	EncounterID *string `json:"encounter_id,omitempty"`
	UploadedBy  string  `json:"uploaded_by"`
	Filename    string  `json:"filename"`
	MimeType    string  `json:"mime_type"`
	Size        int64   `json:"size"`
	Version     int32   `json:"version"`
	RootID      *string `json:"root_id,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

// toAttachmentResponse deliberately omits FileKey — it's an internal
// on-disk detail, not something a client needs or should be able to see.
func toAttachmentResponse(a Attachment) attachmentResponse {
	var encounterID *string
	if a.EncounterID != nil {
		s := a.EncounterID.String()
		encounterID = &s
	}
	var rootID *string
	if a.RootID != nil {
		s := a.RootID.String()
		rootID = &s
	}
	return attachmentResponse{
		ID: a.ID.String(), PatientID: a.PatientID.String(), EncounterID: encounterID,
		UploadedBy: a.UploadedBy.String(), Filename: a.Filename, MimeType: a.MimeType,
		Size: a.Size, Version: a.Version, RootID: rootID, CreatedAt: a.CreatedAt.Format(time.RFC3339),
	}
}

// maxAttachmentUploadBody caps the raw request body read for the upload
// endpoint: maxAttachmentSize for the file itself plus a fixed allowance
// for multipart boundary/header overhead — mirrors
// identity.maxProfilePhotoUploadBody. The authoritative size check is
// Service.Upload's exact len(data) comparison; this is just a defensive
// ceiling on how much is ever read into memory.
const maxAttachmentUploadBody = maxAttachmentSize + 1<<20

// uploadAttachment accepts a multipart/form-data body: a required "file"
// field, plus optional "encounter_id" and "supersedes_id" form fields (the
// latter marking this upload as a new version of an existing document).
func (h *Handler) uploadAttachment(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	patientID, err := uuid.Parse(chi.URLParam(r, "patientID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid patient id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentUploadBody)
	if err := r.ParseMultipartForm(maxAttachmentUploadBody); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "upload too large or malformed")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile("file")
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "file is required (field name \"file\")")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "failed to read uploaded file")
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	var encounterID *uuid.UUID
	if v := r.FormValue("encounter_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			platform.WriteError(w, http.StatusBadRequest, "invalid encounter_id")
			return
		}
		encounterID = &id
	}

	var supersedesID *uuid.UUID
	if v := r.FormValue("supersedes_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			platform.WriteError(w, http.StatusBadRequest, "invalid supersedes_id")
			return
		}
		supersedesID = &id
	}

	attachment, err := h.service.Upload(r.Context(), UploadInput{
		PatientID: patientID, EncounterID: encounterID, UploadedBy: claims.UserID,
		Filename: header.Filename, MimeType: mimeType, Data: data, SupersedesID: supersedesID,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrUnsupportedMimeType), errors.Is(err, ErrAttachmentTooLarge),
			errors.Is(err, ErrSupersedesNotFound), errors.Is(err, ErrEncounterPatientMismatch):
			platform.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, records.ErrPatientNotFound):
			platform.WriteError(w, http.StatusNotFound, "patient not found")
		case errors.Is(err, records.ErrEncounterNotFound):
			platform.WriteError(w, http.StatusNotFound, "encounter not found")
		default:
			platform.WriteError(w, http.StatusInternalServerError, "failed to upload attachment")
		}
		return
	}

	platform.WriteJSON(w, http.StatusCreated, toAttachmentResponse(attachment))
}

func (h *Handler) listAttachments(w http.ResponseWriter, r *http.Request) {
	patientID, err := uuid.Parse(chi.URLParam(r, "patientID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid patient id")
		return
	}

	list, err := h.service.ListForPatient(r.Context(), patientID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list attachments")
		return
	}

	out := make([]attachmentResponse, len(list))
	for i, a := range list {
		out[i] = toAttachmentResponse(a)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

// downloadAttachment streams the file server-side (never a redirect to a
// static URL — see the package doc comment) with Content-Type set from the
// stored mime_type and Content-Disposition chosen so browsers can preview
// images/PDFs inline while anything else downloads as an attachment.
func (h *Handler) downloadAttachment(w http.ResponseWriter, r *http.Request) {
	attachmentID, err := uuid.Parse(chi.URLParam(r, "attachmentID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid attachment id")
		return
	}

	attachment, err := h.service.Get(r.Context(), attachmentID)
	if err != nil {
		if errors.Is(err, ErrAttachmentNotFound) {
			platform.WriteError(w, http.StatusNotFound, "attachment not found")
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to load attachment")
		return
	}

	file, err := os.Open(h.service.FilePath(attachment))
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to open attachment file")
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to stat attachment file")
		return
	}

	disposition := "attachment"
	if strings.HasPrefix(attachment.MimeType, "image/") || attachment.MimeType == "application/pdf" {
		disposition = "inline"
	}

	w.Header().Set("Content-Type", attachment.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, attachment.Filename))

	http.ServeContent(w, r, attachment.Filename, stat.ModTime(), file)
}

func (h *Handler) listVersions(w http.ResponseWriter, r *http.Request) {
	attachmentID, err := uuid.Parse(chi.URLParam(r, "attachmentID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid attachment id")
		return
	}

	list, err := h.service.ListVersions(r.Context(), attachmentID)
	if err != nil {
		if errors.Is(err, ErrAttachmentNotFound) {
			platform.WriteError(w, http.StatusNotFound, "attachment not found")
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to list attachment versions")
		return
	}

	out := make([]attachmentResponse, len(list))
	for i, a := range list {
		out[i] = toAttachmentResponse(a)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}
