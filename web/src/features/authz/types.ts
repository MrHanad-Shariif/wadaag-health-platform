import type { Role as LegacyRole } from '../auth/types'

export interface Permission {
  id: string
  resource: string
  action: string
}

export interface AuthzRole {
  id: string
  name: string
  description?: string
  full_access: boolean
  created_at: string
  permissions?: Permission[]
}

export interface RoleInput {
  name: string
  description?: string
  full_access: boolean
  permission_ids: string[]
}

export interface AuthzUser {
  id: string
  email?: string
  phone?: string
  legacy_role: LegacyRole
  status: string
  created_at: string
  role_id?: string
  role_name?: string
}

export interface CreateUserInput {
  email?: string
  phone?: string
  password: string
  full_name?: string
  legacy_role: LegacyRole
  role_id?: string
  facility_id?: string
  specialty?: string
  license_number?: string
}

// system_admin-only platform toggles — see GET/PATCH
// /api/v1/authz/feature-flags in backend/internal/authz/handler.go.
export interface FeatureFlag {
  key: string
  enabled: boolean
  description?: string
  updated_at: string
}

export interface SetFeatureFlagInput {
  enabled: boolean
  description?: string
}
