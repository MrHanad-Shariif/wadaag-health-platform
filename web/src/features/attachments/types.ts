export type AttachmentMimeType = 'image/jpeg' | 'image/png' | 'image/webp' | 'application/pdf'

export interface Attachment {
  id: string
  patient_id: string
  encounter_id?: string
  uploaded_by: string
  filename: string
  mime_type: AttachmentMimeType
  size: number
  version: number
  root_id?: string
  created_at: string
}

export interface UploadAttachmentOptions {
  encounterId?: string
  supersedesId?: string
}
