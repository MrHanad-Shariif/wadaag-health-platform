export type AuditResult = 'allowed' | 'denied'

export interface AuditEntry {
  id: string
  actor_user_id: string
  actor_role: string
  action: string
  resource_type: string
  resource_id?: string
  patient_id?: string
  result: AuditResult
  occurred_at: string
}

// All optional — an omitted field matches every row for that column, see
// backend/internal/audit/handler.go's listEntries.
export interface AuditEntryFilter {
  actor_user_id?: string
  resource_type?: string
  result?: AuditResult
  limit?: number
}
