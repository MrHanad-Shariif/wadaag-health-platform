export interface Patient {
  id: string
  full_name: string
  date_of_birth?: string
  sex?: string
  national_id?: string
  phone?: string
  address?: string
  next_of_kin?: string
  version: number
}

export type EncounterType = 'referral' | 'consult' | 'general_visit'

export interface Encounter {
  id: string
  patient_id: string
  facility_id: string
  provider_id: string
  type: EncounterType
  notes?: string
  occurred_at: string
  version: number
}

export type ObservationType = 'vitals' | 'diagnosis' | 'note' | 'attachment'

export interface ClinicalObservation {
  id: string
  encounter_id: string
  type: ObservationType
  payload: Record<string, unknown>
  recorded_at: string
}

export interface CreatePatientInput {
  full_name: string
  date_of_birth?: string
  sex?: string
  national_id?: string
  phone?: string
  address?: string
  next_of_kin?: string
}

export interface CreateEncounterInput {
  type: EncounterType
  notes?: string
}

export interface CreateObservationInput {
  type: ObservationType
  payload: Record<string, unknown>
}
