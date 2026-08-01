export type Urgency = 'routine' | 'urgent' | 'emergency'

export type ReferralStatus =
  | 'created'
  | 'routed'
  | 'accepted'
  | 'declined'
  | 'in_progress'
  | 'completed'
  | 'cancelled'

export interface Referral {
  id: string
  patient_id: string
  referring_provider_id: string
  referring_facility_id: string
  receiving_facility_id?: string
  receiving_provider_id?: string
  specialty_requested: string
  urgency: Urgency
  status: ReferralStatus
  reason: string
  version: number
  created_at: string
  updated_at: string
}

export interface StatusEvent {
  from_status?: ReferralStatus
  to_status: ReferralStatus
  actor_user_id: string
  note?: string
  occurred_at: string
}

export interface CreateReferralInput {
  patient_id: string
  receiving_facility_id: string
  specialty_requested: string
  urgency: Urgency
  reason: string
  clinical_summary_encounter_id?: string
}
