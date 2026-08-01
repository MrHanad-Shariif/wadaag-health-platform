import type { User } from './types'

export function hasPermission(user: User | null, resource: string, action: string): boolean {
  if (!user) return false
  if (user.full_access) return true
  return user.permissions?.includes(`${resource}:${action}`) ?? false
}
