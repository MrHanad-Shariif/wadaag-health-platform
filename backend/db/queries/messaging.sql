-- name: CreateConversation :one
INSERT INTO conversations (type, created_by)
VALUES ($1, $2)
RETURNING *;

-- name: AddConversationMember :exec
INSERT INTO conversation_members (conversation_id, user_id)
VALUES ($1, $2);

-- name: FindDirectConversationBetween :one
-- De-duplicates direct conversations — reuse the existing 1:1 conversation
-- between two users instead of creating a new one every time.
SELECT c.* FROM conversations c
WHERE c.type = 'direct'
  AND EXISTS (SELECT 1 FROM conversation_members m1 WHERE m1.conversation_id = c.id AND m1.user_id = $1)
  AND EXISTS (SELECT 1 FROM conversation_members m2 WHERE m2.conversation_id = c.id AND m2.user_id = $2)
  AND (SELECT count(*) FROM conversation_members m WHERE m.conversation_id = c.id) = 2
LIMIT 1;

-- name: ListConversationsForUser :many
-- Includes a last-message preview via LATERAL join so the conversation list
-- can show it without N+1 queries. Also surfaces the caller's own
-- last_read_at (via the join condition, m is always the caller's own
-- membership row) and, for direct conversations, the other member's user_id
-- (via another LATERAL join) so the UI has someone to label the thread
-- with instead of just created_by, which is meaningless when the caller
-- didn't create it.
SELECT
    c.*,
    m.last_read_at AS caller_last_read_at,
    lm.body AS last_message_body,
    lm.created_at AS last_message_at,
    other.user_id AS other_user_id
FROM conversations c
JOIN conversation_members m ON m.conversation_id = c.id AND m.user_id = $1
LEFT JOIN LATERAL (
    SELECT body, created_at FROM messages WHERE conversation_id = c.id ORDER BY created_at DESC LIMIT 1
) lm ON true
LEFT JOIN LATERAL (
    SELECT user_id FROM conversation_members WHERE conversation_id = c.id AND user_id != $1 LIMIT 1
) other ON true
ORDER BY c.updated_at DESC;

-- name: FindConversationMember :one
SELECT * FROM conversation_members WHERE conversation_id = $1 AND user_id = $2;

-- name: ListConversationMembers :many
SELECT * FROM conversation_members WHERE conversation_id = $1;

-- name: TouchConversation :exec
UPDATE conversations SET updated_at = now() WHERE id = $1;

-- name: CreateMessage :one
INSERT INTO messages (conversation_id, sender_id, body, attachment_ids)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListRecentMessages :many
-- Returns the most recent 200 messages oldest-first: the inner query picks
-- the latest 200 by created_at DESC, then the outer query re-orders that
-- subset ascending. Plain "ORDER BY created_at ASC LIMIT 200" would instead
-- return the OLDEST 200 messages, which is wrong for a chat history view.
SELECT * FROM (
    SELECT * FROM messages WHERE conversation_id = $1 ORDER BY created_at DESC LIMIT 200
) recent ORDER BY created_at ASC;

-- name: MarkConversationRead :exec
UPDATE conversation_members SET last_read_at = now() WHERE conversation_id = $1 AND user_id = $2;
