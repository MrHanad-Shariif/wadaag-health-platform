import { apiClient } from '../../api/client'
import type { Facility } from './types'

export function listFacilities() {
  return apiClient.get<Facility[]>('/api/v1/facilities')
}
