import { apiClient } from '../../api/client'
import type { LoginResponse, PublicIdentity, Session, User, VerifyEmailResponse } from './types'

export function login(identifier: string, password: string) {
  return apiClient.post<LoginResponse>('/api/v1/auth/login', { identifier, password })
}

export function register(email: string, password: string, fullName: string) {
  return apiClient.post<User>('/api/v1/auth/register', { email, password, full_name: fullName })
}

export function me() {
  return apiClient.get<User>('/api/v1/auth/me')
}

// Updates the caller's own account-identity full name. Distinct from the
// profile endpoint (bio/photo/languages, a different table/concern) — see
// features/profile/api.ts's updateMyProfile.
export function updateFullName(fullName: string) {
  return apiClient.patch<User>('/api/v1/auth/me', { full_name: fullName })
}

// In-memory cache so repeated lookups of the same user id (e.g. every
// message in a thread from the same sender) don't refetch each time.
const identityCache = new Map<string, PublicIdentity | null>()
const identityInFlight = new Map<string, Promise<PublicIdentity | null>>()

// Resolves a raw user id to a minimal, non-sensitive identity for display
// purposes (see PublicIdentity). Returns null (and caches the negative
// result) on a 404 rather than throwing, so callers can fall back to a
// placeholder without needing their own try/catch.
export function getPublicIdentity(userId: string): Promise<PublicIdentity | null> {
  if (identityCache.has(userId)) {
    return Promise.resolve(identityCache.get(userId)!)
  }
  const inFlight = identityInFlight.get(userId)
  if (inFlight) return inFlight

  const promise = apiClient
    .get<PublicIdentity>(`/api/v1/auth/users/${userId}`)
    .then((identity) => {
      identityCache.set(userId, identity)
      return identity
    })
    .catch(() => {
      identityCache.set(userId, null)
      return null
    })
    .finally(() => {
      identityInFlight.delete(userId)
    })

  identityInFlight.set(userId, promise)
  return promise
}

export function listSessions() {
  return apiClient.get<Session[]>('/api/v1/auth/sessions')
}

export function revokeSession(id: string) {
  return apiClient.del<void>(`/api/v1/auth/sessions/${id}`)
}

export function verifyEmail(token: string) {
  return apiClient.get<VerifyEmailResponse>(`/api/v1/auth/verify-email?token=${encodeURIComponent(token)}`)
}

export function sendVerification() {
  return apiClient.post<void>('/api/v1/auth/send-verification', undefined)
}

export function forgotPassword(identifier: string) {
  return apiClient.post<void>('/api/v1/auth/forgot-password', { identifier })
}

export function resetPassword(token: string, newPassword: string) {
  return apiClient.post<void>('/api/v1/auth/reset-password', { token, new_password: newPassword })
}

// Self-service password change for an already-authenticated user — distinct
// from forgotPassword/resetPassword's locked-out-user flow. 204 on success;
// every other session the caller holds is revoked server-side (see
// backend/internal/identity/handler.go's changePassword).
export function changePassword(currentPassword: string, newPassword: string) {
  return apiClient.patch<void>('/api/v1/auth/me/password', {
    current_password: currentPassword,
    new_password: newPassword,
  })
}
