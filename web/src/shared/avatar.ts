// Shared helper for deriving a 1-2 letter avatar fallback from a user's
// display name (or email/phone, since that's what we usually have).
export function initials(name: string): string {
  const parts = name.replace(/@.*/, '').split(/[.\s_-]+/).filter(Boolean)
  const chars = parts.slice(0, 2).map((p) => p[0]?.toUpperCase() ?? '')
  return chars.join('') || name[0]?.toUpperCase() || '?'
}
