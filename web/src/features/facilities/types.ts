export type FacilityType = 'hospital' | 'clinic' | 'lab' | 'pharmacy' | 'insurer'

export interface Facility {
  id: string
  name: string
  type: FacilityType
  region?: string
  district?: string
  phone?: string
  address?: string
}
