import { apiClient } from '../../api/client'
import type { Profile, UpdateProfileInput } from './types'

export function getMyProfile() {
  return apiClient.get<Profile>('/api/v1/auth/me/profile')
}

export function updateMyProfile(patch: UpdateProfileInput) {
  return apiClient.patch<Profile>('/api/v1/auth/me/profile', patch)
}

export function uploadMyProfilePhoto(file: File) {
  const form = new FormData()
  form.append('photo', file)
  return apiClient.postForm<Profile>('/api/v1/auth/me/profile/photo', form)
}
