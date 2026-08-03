export type ConversationType = 'direct' | 'group'

export interface Conversation {
  id: string
  type: ConversationType
  created_by: string
  last_message_body?: string
  last_message_at?: string
  // The CALLER's own read marker for this conversation.
  last_read_at?: string
  created_at: string
  updated_at: string
  // NOT part of the pre-specified contract — the spec'd shape has no way
  // to identify the other party of a direct conversation when the caller
  // is `created_by` (no members list on this type). Read defensively in
  // case the backend adds it; the list page falls back to `created_by`
  // when it's the other member, and to an "unknown" label otherwise. Flag
  // this for reconciliation with whatever the backend agent ships.
  other_user_id?: string
}

export interface Message {
  id: string
  conversation_id: string
  sender_id: string
  body: string
  attachment_ids: string[]
  created_at: string
}

export interface CreateConversationInput {
  other_user_id: string
}

export interface PostMessageInput {
  body: string
  attachment_ids?: string[]
}

// WebSocket wire protocol — see useConversationSocket.
export type ServerSocketEvent =
  | { type: 'message'; message: Message }
  | { type: 'typing'; user_id: string }
  | { type: 'read'; user_id: string; read_at: string }

export type ClientSocketEvent = { type: 'typing' } | { type: 'read' }

export function isUnread(conversation: Conversation): boolean {
  if (!conversation.last_message_at) return false
  if (!conversation.last_read_at) return true
  return new Date(conversation.last_message_at).getTime() > new Date(conversation.last_read_at).getTime()
}

// Best-effort "who am I talking to" user id — see the `other_user_id` note
// above. Falls back to `created_by` when the caller isn't the creator
// (true for a direct conversation's only other member), otherwise there's
// no way to know from this shape alone.
export function otherPartyId(conversation: Conversation, currentUserId: string | undefined): string | undefined {
  if (conversation.other_user_id) return conversation.other_user_id
  if (conversation.created_by && conversation.created_by !== currentUserId) return conversation.created_by
  return undefined
}

// Raw-id fallback label (used for sorting/search, and as a last resort
// before a name has resolved) — prefer resolving `otherPartyId` through
// useUserDisplayName/UserDisplayName for actual display.
export function otherPartyLabel(conversation: Conversation, currentUserId: string | undefined): string {
  return otherPartyId(conversation, currentUserId) ?? 'Unknown user'
}
