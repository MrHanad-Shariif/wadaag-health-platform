import { apiClient } from '../../api/client'
import type { CreateReferralInput, Referral, StatusEvent } from './types'

export function listInbox() {
  return apiClient.get<Referral[]>('/api/v1/referrals/inbox')
}

export function listForPatient(patientId: string) {
  return apiClient.get<Referral[]>(`/api/v1/referrals/patients/${patientId}`)
}

export function createReferral(input: CreateReferralInput) {
  return apiClient.post<Referral>('/api/v1/referrals', input)
}

export function getReferral(referralId: string) {
  return apiClient.get<Referral>(`/api/v1/referrals/${referralId}`)
}

export function getReferralEvents(referralId: string) {
  return apiClient.get<StatusEvent[]>(`/api/v1/referrals/${referralId}/events`)
}

type Action = 'accept' | 'decline' | 'start' | 'complete' | 'cancel'

function runAction(referralId: string, action: Action, version: number, note?: string) {
  return apiClient.post<Referral>(`/api/v1/referrals/${referralId}/${action}`, { version, note })
}

export const acceptReferral = (referralId: string, version: number) => runAction(referralId, 'accept', version)
export const declineReferral = (referralId: string, version: number, note?: string) =>
  runAction(referralId, 'decline', version, note)
export const startReferral = (referralId: string, version: number) => runAction(referralId, 'start', version)
export const completeReferral = (referralId: string, version: number) => runAction(referralId, 'complete', version)
export const cancelReferral = (referralId: string, version: number, note?: string) =>
  runAction(referralId, 'cancel', version, note)
