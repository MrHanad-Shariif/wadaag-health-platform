import type { Notification } from './types'

interface Described {
  message: string
  to?: string
  // Raw sender user id, when the payload has one — the caller (NotificationBell)
  // resolves this to a display name via useUserDisplayName rather than doing
  // that resolution here, since this function is a plain (non-hook) mapper.
  senderId?: string
}

// Maps a notification's `type` + `payload` to a human-readable message and
// an optional link path. New `type` values may be added on the backend
// without a frontend change — anything unrecognized falls back to a generic
// message with no link.
export function describeNotification(n: Notification): Described {
  const payload = n.payload ?? {}

  switch (n.type) {
    case 'referral_status_changed': {
      const referralId = String(payload.referral_id ?? '')
      const status = String(payload.status ?? 'updated')
      return {
        message: `Referral status changed to ${status}`,
        to: referralId ? `/referrals/${referralId}` : undefined,
      }
    }
    case 'consult_message': {
      const consultationId = String(payload.consultation_id ?? '')
      return {
        message: 'New message on a consultation',
        to: consultationId ? `/consults/${consultationId}` : undefined,
        senderId: payload.sender_id ? String(payload.sender_id) : undefined,
      }
    }
    case 'new_message': {
      const conversationId = String(payload.conversation_id ?? '')
      return {
        message: 'New message',
        to: conversationId ? `/messages/${conversationId}` : undefined,
        senderId: payload.sender_id ? String(payload.sender_id) : undefined,
      }
    }
    case 'record_updated': {
      const patientId = String(payload.patient_id ?? '')
      return {
        message: 'Patient record updated',
        to: patientId ? `/patients/${patientId}` : undefined,
      }
    }
    default:
      return { message: 'New notification' }
  }
}
