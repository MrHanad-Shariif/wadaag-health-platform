export type ConsultStatus = 'requested' | 'responded' | 'closed'

export type ConsultDirection = 'incoming' | 'outgoing'

export interface Consultation {
  id: string
  patient_id: string
  requesting_provider_id: string
  target_provider_id: string
  reason: string
  status: ConsultStatus
  // Only present on items from the /consults/inbox list response — absent
  // when fetching a single consultation by id.
  direction?: ConsultDirection
  created_at: string
  updated_at: string
}

export interface ConsultMessage {
  id: string
  consultation_id: string
  sender_id: string
  body: string
  attachment_ids: string[]
  created_at: string
}

export interface CreateConsultInput {
  patient_id: string
  target_provider_id: string
  reason: string
}

export interface PostConsultMessageInput {
  body: string
  attachment_ids?: string[]
}
