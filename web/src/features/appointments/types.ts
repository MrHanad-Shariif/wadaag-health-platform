export type AppointmentStatus = 'scheduled' | 'confirmed' | 'cancelled' | 'completed' | 'no_show'

// Matches the Appointment JSON shape returned by the appointments handler
// (Phase 8 backend — see docs/plans/full-feature-roadmap.md).
export interface Appointment {
  id: string
  patient_id: string
  provider_id: string
  facility_id: string
  start_at: string // RFC3339
  end_at: string // RFC3339
  status: AppointmentStatus
  reason?: string
  created_at: string
  updated_at: string
}

// weekday: 0=Sunday..6=Saturday. start_time/end_time are "HH:MM" strings,
// not full timestamps — a rule repeats every week.
export interface AvailabilityRule {
  id: string
  provider_id: string
  weekday: number
  start_time: string
  end_time: string
}

export interface SetAvailabilityInput {
  weekday: number
  start_time: string
  end_time: string
}

// An open bookable window on a given day — already excludes existing
// bookings server-side.
export interface Slot {
  start_at: string
  end_at: string
}

export interface BookAppointmentInput {
  // Ignored/overridden server-side when the caller is a patient (they can
  // only ever book for themselves); required when the caller is a
  // physician/hospital_admin/system_admin booking on someone's behalf.
  patient_id?: string
  provider_id: string
  facility_id: string
  start_at: string
  end_at: string
  reason?: string
}

export interface RescheduleAppointmentInput {
  start_at: string
  end_at: string
}
