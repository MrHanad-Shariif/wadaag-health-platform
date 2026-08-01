import type { LucideIcon } from 'lucide-react'

export function StatCard({ label, value, icon: Icon }: { label: string; value: number; icon: LucideIcon }) {
  return (
    <div className="stat-card">
      <span className="stat-card__icon">
        <Icon size={18} aria-hidden="true" />
      </span>
      <div className="stat-card__body">
        <span className="stat-card__value">{value.toLocaleString()}</span>
        <span className="stat-card__label">{label}</span>
      </div>
    </div>
  )
}
