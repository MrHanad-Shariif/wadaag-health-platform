import { apiClient } from '../../api/client'
import type { Provider, UpdateProviderProfileInput, VerificationStatus } from './types'

export function getOwnProviderProfile() {
  return apiClient.get<Provider>('/api/v1/facilities/providers/me')
}

export function updateOwnProviderProfile(input: UpdateProviderProfileInput) {
  return apiClient.patch<Provider>('/api/v1/facilities/providers/me', input)
}

export function getProvider(providerId: string) {
  return apiClient.get<Provider>(`/api/v1/facilities/providers/${providerId}`)
}

export function updateProviderVerificationStatus(providerId: string, verificationStatus: VerificationStatus) {
  return apiClient.patch<Provider>(`/api/v1/facilities/providers/${providerId}/verification-status`, {
    verification_status: verificationStatus,
  })
}
