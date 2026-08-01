export type Role =
  | 'patient'
  | 'physician'
  | 'hospital_admin'
  | 'lab_tech'
  | 'pharmacist'
  | 'insurer'
  | 'system_admin'

export interface User {
  id: string
  email?: string
  phone?: string
  role: Role
  status: string
}

export interface LoginResponse {
  user: User
  access_token: string
  refresh_token: string
}
