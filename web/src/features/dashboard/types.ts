export interface DoctorActivity {
  provider_id: string
  full_name?: string
  referral_count: number
}

export interface Summary {
  total_patients: number
  total_encounters: number
  total_facilities: number
  total_users: number
  referrals_by_status: Record<string, number>
  most_active_doctors: DoctorActivity[]
  // The following three are only ever non-zero when the caller is a
  // physician — see backend/internal/dashboard/handler.go's
  // summaryResponse comment. Every other role gets plain zeroes.
  pending_consult_count: number
  my_referrals_pending_count: number
  patients_today_count: number
}
