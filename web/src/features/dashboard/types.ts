export interface Summary {
  total_patients: number
  total_encounters: number
  total_facilities: number
  total_users: number
  referrals_by_status: Record<string, number>
}
