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
  role_id?: string
  full_access: boolean
  permissions?: string[]
}

export interface LoginResponse {
  user: User
  access_token: string
  refresh_token: string
}
