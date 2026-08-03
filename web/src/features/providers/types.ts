export type VerificationStatus = 'pending' | 'verified' | 'rejected'

export const VERIFICATION_STATUSES: VerificationStatus[] = ['pending', 'verified', 'rejected']

// Matches the certificates JSONB shape documented on the providers table —
// see the comment above `certificates` in
// backend/db/migrations/0014_extend_providers.up.sql. Free-form JSON, so
// every field is optional; "url" (when present) is an external link until
// Phase 3's attachments module lands.
export interface Certificate {
  name?: string
  issuer?: string
  year?: number
  url?: string
}

// Matches providerResponse in backend/internal/facilities/handler.go.
export interface Provider {
  id: string
  user_id: string
  facility_id: string
  specialty?: string
  license_number?: string
  qualifications?: string[]
  years_experience?: number
  // Decimal, sent/received as a string (not a float) — same convention this
  // repo already uses elsewhere for money.
  consultation_fee?: string
  verification_status: VerificationStatus | string
  languages?: string[]
  certificates?: Certificate[]
  areas_of_expertise?: string[]
}

// PATCH /providers/me's body — a partial update where an omitted field
// leaves the current value unchanged (see updateProviderProfileRequest in
// backend/internal/facilities/handler.go). Deliberately excludes
// specialty/license_number/verification_status, which aren't editable
// through this endpoint.
export interface UpdateProviderProfileInput {
  qualifications?: string[]
  years_experience?: number
  consultation_fee?: string
  languages?: string[]
  certificates?: Certificate[]
  areas_of_expertise?: string[]
}
