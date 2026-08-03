package messaging

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// Client is one connected WebSocket subscriber to a single conversation.
// The WebSocket connection itself is receive-only push from the server's
// point of view for message events — REST is the only write path for
// sending a message — but a Client does read typing/read signals sent by
// its own user (see Handler.serveWS's read loop).
type Client struct {
	conn           *websocket.Conn
	userID         uuid.UUID
	conversationID uuid.UUID
}

func NewClient(conn *websocket.Conn, userID, conversationID uuid.UUID) *Client {
	return &Client{conn: conn, userID: userID, conversationID: conversationID}
}

func (c *Client) UserID() uuid.UUID { return c.userID }

// Send writes v to the client's socket as JSON.
func (c *Client) Send(ctx context.Context, v any) error {
	return wsjson.Write(ctx, c.conn, v)
}

// Hub is the process-local WS broadcast registry: which clients are
// currently connected to which conversation. It has no external
// dependencies, so it's constructed once in main.go before anything else
// and handed to messaging.NewHandler.
type Hub struct {
	mu      sync.Mutex
	clients map[uuid.UUID]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[uuid.UUID]map[*Client]struct{})}
}

// Register adds c to the set of clients listening on conversationID.
func (h *Hub) Register(conversationID uuid.UUID, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.clients[conversationID]
	if !ok {
		set = make(map[*Client]struct{})
		h.clients[conversationID] = set
	}
	set[c] = struct{}{}
}

// Unregister removes c from conversationID's client set, cleaning up the
// set entirely once it's empty so the map doesn't grow unbounded with
// entries for conversations nobody is currently connected to.
func (h *Hub) Unregister(conversationID uuid.UUID, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.clients[conversationID]
	if !ok {
		return
	}
	delete(set, c)
	if len(set) == 0 {
		delete(h.clients, conversationID)
	}
}

// Broadcast sends event to every client currently registered on
// conversationID. It deliberately does not exclude the sender's own
// connection — including "message" events, the sender's own client will
// see its own message echoed back over the socket. That's simplest here:
// the alternative (excluding one specific *Client) buys nothing since a
// user's own REST response already gives them the created message, and
// client-side de-duplication by message id is trivial if it ever matters
// for a multi-tab sender. Errors writing to an individual client are
// ignored — a dead/slow connection shouldn't block delivery to the rest of
// the conversation; that connection will be cleaned up when its own read
// loop errors out and calls Unregister.
func (h *Hub) Broadcast(ctx context.Context, conversationID uuid.UUID, event any) {
	h.mu.Lock()
	set := h.clients[conversationID]
	targets := make([]*Client, 0, len(set))
	for c := range set {
		targets = append(targets, c)
	}
	h.mu.Unlock()

	for _, c := range targets {
		_ = c.Send(ctx, event)
	}
}
