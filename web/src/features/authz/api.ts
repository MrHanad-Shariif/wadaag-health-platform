import { apiClient } from '../../api/client'
import type { AuthzRole, AuthzUser, CreateUserInput, FeatureFlag, Permission, RoleInput, SetFeatureFlagInput } from './types'

export const listPermissions = () => apiClient.get<Permission[]>('/api/v1/authz/permissions')

export const listRoles = () => apiClient.get<AuthzRole[]>('/api/v1/authz/roles')
export const getRole = (id: string) => apiClient.get<AuthzRole>(`/api/v1/authz/roles/${id}`)
export const createRole = (input: RoleInput) => apiClient.post<AuthzRole>('/api/v1/authz/roles', input)
export const updateRole = (id: string, input: RoleInput) => apiClient.put<AuthzRole>(`/api/v1/authz/roles/${id}`, input)
export const deleteRole = (id: string) => apiClient.del(`/api/v1/authz/roles/${id}`)

export const listUsers = () => apiClient.get<AuthzUser[]>('/api/v1/authz/users')
export const getUser = (id: string) => apiClient.get<AuthzUser>(`/api/v1/authz/users/${id}`)
export const createUser = (input: CreateUserInput) => apiClient.post<AuthzUser>('/api/v1/authz/users', input)
export const updateUserRole = (id: string, roleId: string | null) =>
  apiClient.patch<AuthzUser>(`/api/v1/authz/users/${id}/role`, { role_id: roleId })
export const updateUserStatus = (id: string, status: string) =>
  apiClient.patch<AuthzUser>(`/api/v1/authz/users/${id}/status`, { status })

export const listFeatureFlags = () => apiClient.get<FeatureFlag[]>('/api/v1/authz/feature-flags')

// Upserts — a key that doesn't exist yet gets created, so this also
// doubles as "create new flag" (see the roadmap-driven FeatureFlagsPage).
export const setFeatureFlag = (key: string, input: SetFeatureFlagInput) =>
  apiClient.patch<FeatureFlag>(`/api/v1/authz/feature-flags/${encodeURIComponent(key)}`, input)
