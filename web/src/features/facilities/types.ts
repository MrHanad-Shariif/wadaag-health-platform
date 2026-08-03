export type FacilityType = 'hospital' | 'clinic' | 'lab' | 'pharmacy' | 'insurer'

export const FACILITY_TYPES: FacilityType[] = ['hospital', 'clinic', 'lab', 'pharmacy', 'insurer']

export type WeekdayKey = 'mon' | 'tue' | 'wed' | 'thu' | 'fri' | 'sat' | 'sun'

export const WEEKDAYS: { key: WeekdayKey; label: string }[] = [
  { key: 'mon', label: 'Monday' },
  { key: 'tue', label: 'Tuesday' },
  { key: 'wed', label: 'Wednesday' },
  { key: 'thu', label: 'Thursday' },
  { key: 'fri', label: 'Friday' },
  { key: 'sat', label: 'Saturday' },
  { key: 'sun', label: 'Sunday' },
]

export interface WorkingHoursEntry {
  open: string
  close: string
}

// One entry per weekday key; an omitted day means "closed" — matches the
// working_hours jsonb comment in db/migrations/0012_extend_facilities.up.sql.
export type WorkingHours = Partial<Record<WeekdayKey, WorkingHoursEntry>>

export interface Facility {
  id: string
  name: string
  type: FacilityType
  region?: string
  district?: string
  phone?: string
  address?: string
  logo_url?: string
  working_hours?: WorkingHours
  referral_policies?: string
}

export interface CreateFacilityInput {
  name: string
  type: FacilityType
  region?: string
  district?: string
  phone?: string
  address?: string
}

// PATCH /facilities/{facilityID} only accepts logo_url/working_hours/
// referral_policies — a partial update where an omitted field leaves the
// current value unchanged (see UpdateFacilityInput in
// backend/internal/facilities/service.go).
export interface UpdateFacilityInput {
  logo_url?: string
  working_hours?: WorkingHours
  referral_policies?: string
}

export interface Branch {
  id: string
  facility_id: string
  name: string
  address?: string
  phone?: string
}

export interface BranchInput {
  name: string
  address?: string
  phone?: string
}

export interface Department {
  id: string
  facility_id: string
  name: string
}

export interface DepartmentInput {
  name: string
}
