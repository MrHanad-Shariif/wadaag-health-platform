import { useEffect, useState } from 'react'
import { getPublicIdentity } from '../features/auth/api'

// Truncated-id fallback used while a lookup is in flight or after a 404 —
// keeps something stable and recognizable on screen instead of a blank.
function truncatedId(userId: string): string {
  return userId.length > 8 ? `${userId.slice(0, 8)}…` : userId
}

// Resolves a raw user id to a human-readable display name via the
// GET /auth/users/{userId} lookup (see features/auth/api.ts's
// getPublicIdentity, which already caches resolved identities in-memory so
// repeat lookups of the same id — e.g. every message from the same sender
// in a thread — don't refetch each time).
//
// Returns a sensible fallback (truncated id, or "Unknown user" if the
// lookup 404s) while loading or on failure, so callers can render this
// directly without their own loading/error branching.
export function useUserDisplayName(userId?: string): string {
  const [name, setName] = useState<string | undefined>(undefined)

  useEffect(() => {
    if (!userId) {
      setName(undefined)
      return
    }
    let cancelled = false
    setName(undefined)
    getPublicIdentity(userId).then((identity) => {
      if (cancelled) return
      if (identity === null) {
        setName('Unknown user')
      } else {
        setName(identity.full_name ?? truncatedId(identity.id))
      }
    })
    return () => {
      cancelled = true
    }
  }, [userId])

  if (!userId) return 'Unknown user'
  return name ?? truncatedId(userId)
}
