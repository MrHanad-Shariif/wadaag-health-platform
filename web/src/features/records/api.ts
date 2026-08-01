import { apiClient } from '../../api/client'
import type {
  ClinicalObservation,
  CreateEncounterInput,
  CreateObservationInput,
  CreatePatientInput,
  Encounter,
  Patient,
} from './types'

export function createPatient(input: CreatePatientInput) {
  return apiClient.post<Patient>('/api/v1/records/patients', input)
}

export function getPatient(patientId: string) {
  return apiClient.get<Patient>(`/api/v1/records/patients/${patientId}`)
}

export function listPatients() {
  return apiClient.get<Patient[]>('/api/v1/records/patients')
}

export function listEncounters(patientId: string) {
  return apiClient.get<Encounter[]>(`/api/v1/records/patients/${patientId}/encounters`)
}

export function createEncounter(patientId: string, input: CreateEncounterInput) {
  return apiClient.post<Encounter>(`/api/v1/records/patients/${patientId}/encounters`, input)
}

export function getEncounter(encounterId: string) {
  return apiClient.get<Encounter>(`/api/v1/records/encounters/${encounterId}`)
}

export function listObservations(encounterId: string) {
  return apiClient.get<ClinicalObservation[]>(`/api/v1/records/encounters/${encounterId}/observations`)
}

export function createObservation(encounterId: string, input: CreateObservationInput) {
  return apiClient.post<ClinicalObservation>(`/api/v1/records/encounters/${encounterId}/observations`, input)
}
