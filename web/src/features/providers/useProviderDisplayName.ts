import { useEffect, useState } from 'react'
import { useUserDisplayName } from '../../shared/useUserDisplayName'
import { getProvider } from './api'

// consults.requesting_provider_id / target_provider_id are PROVIDER ids,
// not user ids — see Provider.user_id in features/providers/types.ts.
// Resolving a display name therefore needs an extra hop: provider id ->
// user id (via GET /facilities/providers/{id}) -> display name (via
// useUserDisplayName / GET /auth/users/{userId}). Cached the same way as
// getPublicIdentity so repeat lookups of the same provider (e.g. every
// consult in a list from the same requester) don't refetch.
const providerUserIdCache = new Map<string, string | null>()
const providerUserIdInFlight = new Map<string, Promise<string | null>>()

function resolveProviderUserId(providerId: string): Promise<string | null> {
  if (providerUserIdCache.has(providerId)) {
    return Promise.resolve(providerUserIdCache.get(providerId)!)
  }
  const inFlight = providerUserIdInFlight.get(providerId)
  if (inFlight) return inFlight

  const promise = getProvider(providerId)
    .then((provider) => {
      providerUserIdCache.set(providerId, provider.user_id)
      return provider.user_id
    })
    .catch(() => {
      providerUserIdCache.set(providerId, null)
      return null
    })
    .finally(() => {
      providerUserIdInFlight.delete(providerId)
    })

  providerUserIdInFlight.set(providerId, promise)
  return promise
}

export function useProviderDisplayName(providerId?: string): string {
  const [userId, setUserId] = useState<string | undefined>(undefined)

  useEffect(() => {
    if (!providerId) {
      setUserId(undefined)
      return
    }
    let cancelled = false
    resolveProviderUserId(providerId).then((uid) => {
      if (!cancelled) setUserId(uid ?? undefined)
    })
    return () => {
      cancelled = true
    }
  }, [providerId])

  return useUserDisplayName(userId)
}
