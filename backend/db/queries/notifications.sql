-- name: CreateNotification :one
INSERT INTO notifications (user_id, type, payload)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListNotificationsForUser :many
SELECT * FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50;

-- name: CountUnreadNotifications :one
SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL;

-- name: MarkNotificationRead :exec
UPDATE notifications SET read_at = now() WHERE id = $1 AND user_id = $2 AND read_at IS NULL;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications SET read_at = now() WHERE user_id = $1 AND read_at IS NULL;

-- name: GetNotificationPreferences :many
SELECT * FROM notification_preferences WHERE user_id = $1;

-- name: UpsertNotificationPreference :one
INSERT INTO notification_preferences (user_id, channel, enabled)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, channel) DO UPDATE SET enabled = excluded.enabled
RETURNING *;
