// Two linked arcs — "wadaag" (sharing) as a mark: independent providers
// brought into one shared record. Used at nav scale and on the login panel.
export function BrandMark({ size = 28 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" fill="none" aria-hidden="true">
      <circle cx="13" cy="13" r="9.5" stroke="currentColor" strokeWidth="2.5" />
      <circle cx="20" cy="20" r="9.5" stroke="currentColor" strokeWidth="2.5" opacity="0.55" />
    </svg>
  )
}
