export interface Patient {
  id: string
  full_name: string
  date_of_birth?: string
  sex?: string
  gender?: string
  blood_group?: string
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
  facility_id?: string
}

export interface CreateEncounterInput {
  type: EncounterType
  notes?: string
}

export interface CreateObservationInput {
  type: ObservationType
  payload: Record<string, unknown>
}

export interface UpdatePatientDemographicsInput {
  gender?: string
  blood_group?: string
}

export interface MedicalHistory {
  patient_id: string
  allergies: string[]
  chronic_conditions: string[]
  current_medications: string[]
  past_surgeries: string[]
  family_history: string[]
  vaccination_history: string[]
  updated_at?: string
}

export interface UpdateMedicalHistoryInput {
  allergies: string[]
  chronic_conditions: string[]
  current_medications: string[]
  past_surgeries: string[]
  family_history: string[]
  vaccination_history: string[]
}
