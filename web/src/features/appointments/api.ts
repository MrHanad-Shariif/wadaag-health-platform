import { apiClient } from '../../api/client'
import type { Appointment, AvailabilityRule, BookAppointmentInput, SetAvailabilityInput, Slot } from './types'

export function setAvailability(input: SetAvailabilityInput) {
  return apiClient.post<AvailabilityRule>('/api/v1/appointments/availability', input)
}

export function getAvailability(providerId: string) {
  return apiClient.get<AvailabilityRule[]>(`/api/v1/appointments/availability/${providerId}`)
}

export function deleteAvailabilityRule(ruleId: string) {
  return apiClient.del<void>(`/api/v1/appointments/availability/${ruleId}`)
}

export function getSlots(providerId: string, date: string, slotMinutes = 30) {
  const params = new URLSearchParams({ date, slot_minutes: String(slotMinutes) })
  return apiClient.get<Slot[]>(`/api/v1/appointments/availability/${providerId}/slots?${params.toString()}`)
}

export function bookAppointment(input: BookAppointmentInput) {
  return apiClient.post<Appointment>('/api/v1/appointments', input)
}

export function listMine() {
  return apiClient.get<Appointment[]>('/api/v1/appointments/mine')
}

export function getAppointment(appointmentId: string) {
  return apiClient.get<Appointment>(`/api/v1/appointments/${appointmentId}`)
}

export function rescheduleAppointment(appointmentId: string, startAt: string, endAt: string) {
  return apiClient.patch<Appointment>(`/api/v1/appointments/${appointmentId}/reschedule`, {
    start_at: startAt,
    end_at: endAt,
  })
}

export function cancelAppointment(appointmentId: string) {
  return apiClient.post<Appointment>(`/api/v1/appointments/${appointmentId}/cancel`, {})
}

export function completeAppointment(appointmentId: string) {
  return apiClient.post<Appointment>(`/api/v1/appointments/${appointmentId}/complete`, {})
}

export function markNoShow(appointmentId: string) {
  return apiClient.post<Appointment>(`/api/v1/appointments/${appointmentId}/no-show`, {})
}
