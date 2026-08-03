import { apiClient } from '../../api/client'
import type { Attachment, UploadAttachmentOptions } from './types'

export function uploadAttachment(patientId: string, file: File, options: UploadAttachmentOptions = {}) {
  const form = new FormData()
  form.append('file', file)
  if (options.encounterId) form.append('encounter_id', options.encounterId)
  if (options.supersedesId) form.append('supersedes_id', options.supersedesId)
  return apiClient.postForm<Attachment>(`/api/v1/attachments/patients/${patientId}`, form)
}

export function listAttachments(patientId: string) {
  return apiClient.get<Attachment[]>(`/api/v1/attachments/patients/${patientId}`)
}

export function getAttachmentBlob(attachmentId: string) {
  return apiClient.getBlob(`/api/v1/attachments/${attachmentId}`)
}

export function listVersions(attachmentId: string) {
  return apiClient.get<Attachment[]>(`/api/v1/attachments/${attachmentId}/versions`)
}
