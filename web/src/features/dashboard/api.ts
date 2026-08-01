import { apiClient } from '../../api/client'
import type { Summary } from './types'

export const getSummary = () => apiClient.get<Summary>('/api/v1/dashboard/summary')
