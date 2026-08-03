import { apiClient } from '../../api/client'
import type { LoginResponse, Session, User, VerifyEmailResponse } from './types'

export function login(identifier: string, password: string) {
  return apiClient.post<LoginResponse>('/api/v1/auth/login', { identifier, password })
}

export function register(email: string, password: string) {
  return apiClient.post<User>('/api/v1/auth/register', { email, password })
}

export function me() {
  return apiClient.get<User>('/api/v1/auth/me')
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
