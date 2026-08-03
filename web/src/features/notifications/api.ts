import { apiClient } from '../../api/client'
import type { Notification, NotificationPreference, UnreadCount } from './types'

export function listNotifications() {
  return apiClient.get<Notification[]>('/api/v1/notifications')
}

export function getUnreadCount() {
  return apiClient.get<UnreadCount>('/api/v1/notifications/unread-count')
}

export function markRead(notificationId: string) {
  return apiClient.post<void>(`/api/v1/notifications/${notificationId}/read`, undefined)
}

export function markAllRead() {
  return apiClient.post<void>('/api/v1/notifications/read-all', undefined)
}

export function getPreferences() {
  return apiClient.get<NotificationPreference[]>('/api/v1/notifications/preferences')
}

// Upserts a single (channel, enabled) preference row for the caller.
export function setPreference(channel: string, enabled: boolean) {
  return apiClient.patch<NotificationPreference>('/api/v1/notifications/preferences', { channel, enabled })
}
