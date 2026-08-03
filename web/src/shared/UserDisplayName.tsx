import { useUserDisplayName } from './useUserDisplayName'

// Small wrapper component around useUserDisplayName so it can be dropped
// into places that only accept a ReactNode (e.g. DataTable's `render`
// callback) without violating the rules of hooks — the callback itself
// isn't a component, but the element it returns is.
export function UserDisplayName({ userId }: { userId?: string }) {
  const name = useUserDisplayName(userId)
  return <>{name}</>
}
