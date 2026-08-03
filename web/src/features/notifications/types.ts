export interface Notification {
  id: string
  type: string
  payload: Record<string, unknown>
  read_at?: string
  created_at: string
}

export interface UnreadCount {
  count: number
}

// One row per (caller, channel) — see GET/PATCH /notifications/preferences
// in backend/internal/notifications/handler.go. There's no "list every
// channel" endpoint: a channel simply has no row until it's been set at
// least once, defaulting to enabled=true at the DB level (see the
// notification_preferences migration) once it does exist.
export interface NotificationPreference {
  channel: string
  enabled: boolean
}
