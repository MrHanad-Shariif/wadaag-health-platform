import { apiClient } from '../../api/client'
import type { Consultation, ConsultMessage, CreateConsultInput, PostConsultMessageInput } from './types'

export function listInbox() {
  return apiClient.get<Consultation[]>('/api/v1/consults/inbox')
}

export function createConsult(input: CreateConsultInput) {
  return apiClient.post<Consultation>('/api/v1/consults', input)
}

export function getConsult(consultationId: string) {
  return apiClient.get<Consultation>(`/api/v1/consults/${consultationId}`)
}

export function listMessages(consultationId: string) {
  return apiClient.get<ConsultMessage[]>(`/api/v1/consults/${consultationId}/messages`)
}

export function postMessage(consultationId: string, input: PostConsultMessageInput) {
  return apiClient.post<ConsultMessage>(`/api/v1/consults/${consultationId}/messages`, input)
}

export function closeConsult(consultationId: string) {
  return apiClient.post<Consultation>(`/api/v1/consults/${consultationId}/close`, undefined)
}
