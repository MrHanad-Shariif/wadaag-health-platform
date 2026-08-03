package messaging

import (
	"testing"

	"github.com/google/uuid"
)

// TestHubRegisterUnregister covers the hub's pure bookkeeping: registering a
// client makes it show up under its conversation, unregistering removes it,
// and unregistering the last client for a conversation cleans up the map
// entry entirely rather than leaking an empty set. None of this exercises
// an actual network connection — conn is left nil throughout since these
// assertions never call Client.Send.
func TestHubRegisterUnregister(t *testing.T) {
	h := NewHub()
	conversationID := uuid.New()
	clientA := &Client{userID: uuid.New(), conversationID: conversationID}
	clientB := &Client{userID: uuid.New(), conversationID: conversationID}

	h.Register(conversationID, clientA)
	h.Register(conversationID, clientB)

	if got := len(h.clients[conversationID]); got != 2 {
		t.Fatalf("after registering 2 clients, len(clients[conversationID]) = %d, want 2", got)
	}

	h.Unregister(conversationID, clientA)
	if got := len(h.clients[conversationID]); got != 1 {
		t.Fatalf("after unregistering 1 of 2 clients, len(clients[conversationID]) = %d, want 1", got)
	}
	if _, stillThere := h.clients[conversationID][clientA]; stillThere {
		t.Error("clientA should no longer be registered")
	}

	h.Unregister(conversationID, clientB)
	if _, ok := h.clients[conversationID]; ok {
		t.Error("conversation entry should be removed entirely once its last client unregisters, not left as an empty set")
	}
}

// TestHubUnregisterUnknownIsNoop covers unregistering a client from a
// conversation the hub never registered anything for (e.g. double-close, or
// a client that errored out before ever registering) — must not panic.
func TestHubUnregisterUnknownIsNoop(t *testing.T) {
	h := NewHub()
	conversationID := uuid.New()
	client := &Client{userID: uuid.New(), conversationID: conversationID}

	h.Unregister(conversationID, client)

	if got := len(h.clients); got != 0 {
		t.Errorf("unregistering an unknown client should not create a map entry, got %d entries", got)
	}
}

// TestHubRegisterSeparatesConversations covers that clients registered under
// different conversation ids don't leak into each other's sets — otherwise
// Broadcast would deliver a message from one conversation to a member of an
// unrelated one.
func TestHubRegisterSeparatesConversations(t *testing.T) {
	h := NewHub()
	convA := uuid.New()
	convB := uuid.New()
	clientA := &Client{userID: uuid.New(), conversationID: convA}
	clientB := &Client{userID: uuid.New(), conversationID: convB}

	h.Register(convA, clientA)
	h.Register(convB, clientB)

	if _, ok := h.clients[convA][clientB]; ok {
		t.Error("clientB must not be registered under convA")
	}
	if _, ok := h.clients[convB][clientA]; ok {
		t.Error("clientA must not be registered under convB")
	}
}
