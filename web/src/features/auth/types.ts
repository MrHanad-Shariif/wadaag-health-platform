export type Role =
  | 'patient'
  | 'physician'
  | 'hospital_admin'
  | 'lab_tech'
  | 'pharmacist'
  | 'insurer'
  | 'system_admin'

export interface User {
  id: string
  email?: string
  phone?: string
  full_name?: string
  role: Role
  status: string
  role_id?: string
  // facility_id is only set for a user affiliated with a facility (via a
  // providers row — see facilities.Service.FacilityIDForUser on the
  // backend). system_admin has none, since they manage every facility.
  facility_id?: string
  full_access: boolean
  permissions?: string[]
  verified: boolean
  verified_at?: string
}

export interface LoginResponse {
  user: User
  access_token: string
  refresh_token: string
}

export interface VerifyEmailResponse {
  verified: boolean
}

export interface Session {
  id: string
  device_label?: string
  ip?: string
  user_agent?: string
  created_at: string
  expires_at: string
}

// Minimal, deliberately non-sensitive projection of a user — returned by
// GET /auth/users/{userId} so any authenticated user can resolve "who is
// this" for a raw user id shown elsewhere in the UI (message senders,
// consult parties, etc.) without exposing the full User shape.
export interface PublicIdentity {
  id: string
  full_name?: string
  role: string
}
