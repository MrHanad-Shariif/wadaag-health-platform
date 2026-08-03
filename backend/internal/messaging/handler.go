package messaging

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Handler struct {
	service *Service
	hub     *Hub
	tm      *platform.TokenManager
}

func NewHandler(service *Service, hub *Hub, tm *platform.TokenManager) *Handler {
	return &Handler{service: service, hub: hub, tm: tm}
}

// Routes is mounted at "/conversations" (see server.NewRouter), so the
// final paths are:
//
//	POST /api/v1/conversations
//	GET  /api/v1/conversations
//	GET  /api/v1/conversations/{conversationID}/messages
//	POST /api/v1/conversations/{conversationID}/messages
//	POST /api/v1/conversations/{conversationID}/read
//	GET  /api/v1/conversations/{conversationID}/ws
//
// No consent.Middleware or audit.Logger is used anywhere in this tree —
// unlike patient-scoped modules, a conversation has no single patient_id to
// scope by, and the only authorization rule is "is the caller a member of
// this conversation" (see requireMember below).
//
// The WS upgrade route is deliberately mounted OUTSIDE the group that
// applies platform.RequireAuth: browsers' native WebSocket API cannot set
// an Authorization header, so serveWS authenticates via a `token` query
// parameter instead and does its own membership check.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/{conversationID}/ws", h.serveWS)

	r.Group(func(auth chi.Router) {
		auth.Use(platform.RequireAuth(h.tm))
		auth.Post("/", h.create)
		auth.Get("/", h.list)
		auth.Get("/{conversationID}/messages", h.listMessages)
		auth.Post("/{conversationID}/messages", h.postMessage)
		auth.Post("/{conversationID}/read", h.markRead)
	})

	return r
}

type conversationResponse struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`
	CreatedBy       string  `json:"created_by"`
	OtherUserID     *string `json:"other_user_id,omitempty"`
	LastMessageBody *string `json:"last_message_body,omitempty"`
	LastMessageAt   *string `json:"last_message_at,omitempty"`
	LastReadAt      *string `json:"last_read_at,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

func toConversationResponse(c Conversation) conversationResponse {
	resp := conversationResponse{
		ID:        c.ID.String(),
		Type:      string(c.Type),
		CreatedBy: c.CreatedBy.String(),
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
	}
	if c.OtherUserID != nil {
		s := c.OtherUserID.String()
		resp.OtherUserID = &s
	}
	if c.LastMessageBody != nil {
		resp.LastMessageBody = c.LastMessageBody
	}
	if c.LastMessageAt != nil {
		s := c.LastMessageAt.Format(time.RFC3339)
		resp.LastMessageAt = &s
	}
	if c.LastReadAt != nil {
		s := c.LastReadAt.Format(time.RFC3339)
		resp.LastReadAt = &s
	}
	return resp
}

type messageResponse struct {
	ID             string   `json:"id"`
	ConversationID string   `json:"conversation_id"`
	SenderID       string   `json:"sender_id"`
	Body           string   `json:"body"`
	AttachmentIDs  []string `json:"attachment_ids"`
	CreatedAt      string   `json:"created_at"`
}

func toMessageResponse(m Message) messageResponse {
	attachmentIDs := make([]string, len(m.AttachmentIDs))
	for i, id := range m.AttachmentIDs {
		attachmentIDs[i] = id.String()
	}
	return messageResponse{
		ID:             m.ID.String(),
		ConversationID: m.ConversationID.String(),
		SenderID:       m.SenderID.String(),
		Body:           m.Body,
		AttachmentIDs:  attachmentIDs,
		CreatedAt:      m.CreatedAt.Format(time.RFC3339),
	}
}

type createRequest struct {
	OtherUserID string `json:"other_user_id"`
}

// create starts (or reuses) a direct conversation with another user. Group
// conversations are schema-supported but not exposed by this endpoint — no
// group-creation UI is being built in this pass.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	var req createRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	otherUserID, err := uuid.Parse(req.OtherUserID)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid other_user_id")
		return
	}
	if otherUserID == claims.UserID {
		platform.WriteError(w, http.StatusBadRequest, "cannot start a conversation with yourself")
		return
	}

	conversation, err := h.service.CreateDirect(r.Context(), claims.UserID, otherUserID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to create conversation")
		return
	}
	platform.WriteJSON(w, http.StatusCreated, toConversationResponse(conversation))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	list, err := h.service.ListForUser(r.Context(), claims.UserID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list conversations")
		return
	}
	out := make([]conversationResponse, len(list))
	for i, c := range list {
		out[i] = toConversationResponse(c)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

// requireMember resolves conversationID from the URL and 403s if the caller
// (from claims already in context) is not a member — the sole authorization
// check for every conversation-scoped route in this package.
func (h *Handler) requireMember(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return uuid.UUID{}, uuid.UUID{}, false
	}

	conversationID, err := uuid.Parse(chi.URLParam(r, "conversationID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid conversation id")
		return uuid.UUID{}, uuid.UUID{}, false
	}

	isMember, err := h.service.IsMember(r.Context(), conversationID, claims.UserID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to verify membership")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	if !isMember {
		platform.WriteError(w, http.StatusForbidden, ErrNotMember.Error())
		return uuid.UUID{}, uuid.UUID{}, false
	}

	return conversationID, claims.UserID, true
}

func (h *Handler) listMessages(w http.ResponseWriter, r *http.Request) {
	conversationID, userID, ok := h.requireMember(w, r)
	if !ok {
		return
	}
	list, err := h.service.ListMessages(r.Context(), userID, conversationID)
	if err != nil {
		if errors.Is(err, ErrNotMember) {
			platform.WriteError(w, http.StatusForbidden, err.Error())
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to list messages")
		return
	}
	out := make([]messageResponse, len(list))
	for i, m := range list {
		out[i] = toMessageResponse(m)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

type postMessageRequest struct {
	Body          string   `json:"body"`
	AttachmentIDs []string `json:"attachment_ids"`
}

func (h *Handler) postMessage(w http.ResponseWriter, r *http.Request) {
	conversationID, userID, ok := h.requireMember(w, r)
	if !ok {
		return
	}

	var req postMessageRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Body == "" {
		platform.WriteError(w, http.StatusBadRequest, "body is required")
		return
	}

	attachmentIDs := make([]uuid.UUID, len(req.AttachmentIDs))
	for i, s := range req.AttachmentIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			platform.WriteError(w, http.StatusBadRequest, "invalid attachment_ids entry")
			return
		}
		attachmentIDs[i] = id
	}

	message, err := h.service.PostMessage(r.Context(), userID, conversationID, req.Body, attachmentIDs)
	if err != nil {
		if errors.Is(err, ErrNotMember) {
			platform.WriteError(w, http.StatusForbidden, err.Error())
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to send message")
		return
	}

	resp := toMessageResponse(message)
	h.hub.Broadcast(r.Context(), conversationID, map[string]any{"type": "message", "message": resp})

	platform.WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	conversationID, userID, ok := h.requireMember(w, r)
	if !ok {
		return
	}
	if err := h.service.MarkRead(r.Context(), userID, conversationID); err != nil {
		if errors.Is(err, ErrNotMember) {
			platform.WriteError(w, http.StatusForbidden, err.Error())
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to mark conversation read")
		return
	}

	h.hub.Broadcast(r.Context(), conversationID, map[string]any{
		"type": "read", "user_id": userID.String(), "read_at": time.Now().Format(time.RFC3339),
	})
	platform.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// wsIncoming is the shape of client-sent events over the WebSocket
// connection — the only two things a connected client sends to the server
// (as opposed to receiving) are ephemeral typing/read signals; sending an
// actual message always goes through POST .../messages instead.
type wsIncoming struct {
	Type string `json:"type"`
}

// serveWS upgrades to a WebSocket connection for receive-only push plus
// ephemeral typing/read signals. Not behind platform.RequireAuth (see
// Routes' doc comment) — browsers' native WebSocket API cannot set an
// Authorization header, so this authenticates via a `token` query parameter
// instead, using the same TokenManager.Verify RequireAuth uses internally.
func (h *Handler) serveWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		platform.WriteError(w, http.StatusUnauthorized, "missing token")
		return
	}
	claims, err := h.tm.Verify(token)
	if err != nil {
		platform.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	conversationID, err := uuid.Parse(chi.URLParam(r, "conversationID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}

	isMember, err := h.service.IsMember(r.Context(), conversationID, claims.UserID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to verify membership")
		return
	}
	if !isMember {
		platform.WriteError(w, http.StatusForbidden, ErrNotMember.Error())
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"localhost:5173", "127.0.0.1:5173"},
	})
	if err != nil {
		return
	}

	ctx := r.Context()
	client := NewClient(conn, claims.UserID, conversationID)
	h.hub.Register(conversationID, client)
	defer h.hub.Unregister(conversationID, client)
	defer conn.CloseNow()

	for {
		var incoming wsIncoming
		if err := wsjson.Read(ctx, conn, &incoming); err != nil {
			return
		}

		switch incoming.Type {
		case "typing":
			h.hub.Broadcast(ctx, conversationID, map[string]any{"type": "typing", "user_id": claims.UserID.String()})
		case "read":
			if err := h.service.MarkRead(ctx, claims.UserID, conversationID); err != nil {
				continue
			}
			h.hub.Broadcast(ctx, conversationID, map[string]any{
				"type": "read", "user_id": claims.UserID.String(), "read_at": time.Now().Format(time.RFC3339),
			})
		}
	}
}
