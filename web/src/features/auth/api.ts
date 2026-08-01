import { apiClient } from '../../api/client'
import type { LoginResponse, User } from './types'

export function login(identifier: string, password: string) {
  return apiClient.post<LoginResponse>('/api/v1/auth/login', { identifier, password })
}

export function me() {
  return apiClient.get<User>('/api/v1/auth/me')
}
