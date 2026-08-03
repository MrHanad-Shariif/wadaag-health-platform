import { useProviderDisplayName } from './useProviderDisplayName'

// Same wrapper pattern as shared/UserDisplayName, but for the extra
// provider-id -> user-id hop (see useProviderDisplayName).
export function ProviderDisplayName({ providerId }: { providerId?: string }) {
  const name = useProviderDisplayName(providerId)
  return <>{name}</>
}
