import { apiClient } from '../../api/client'
import type { AuditEntry, AuditEntryFilter } from './types'

// system_admin only — every other role gets a 403 (see the backend's
// listEntries route group). Every filter field is optional.
export function listAuditEntries(filter: AuditEntryFilter = {}) {
  const params = new URLSearchParams()
  if (filter.actor_user_id) params.set('actor_user_id', filter.actor_user_id)
  if (filter.resource_type) params.set('resource_type', filter.resource_type)
  if (filter.result) params.set('result', filter.result)
  if (filter.limit) params.set('limit', String(filter.limit))
  const qs = params.toString()
  return apiClient.get<AuditEntry[]>(`/api/v1/audit${qs ? `?${qs}` : ''}`)
}
