import { useState } from 'react'
import { Table2 } from 'lucide-react'

interface StatusMeta {
  label: string
  group: 'pending' | 'active' | 'good' | 'critical'
}

const STATUS_META: Record<string, StatusMeta> = {
  routed: { label: 'Routed', group: 'pending' },
  accepted: { label: 'Accepted', group: 'active' },
  in_progress: { label: 'In progress', group: 'active' },
  completed: { label: 'Completed', group: 'good' },
  declined: { label: 'Declined', group: 'critical' },
  cancelled: { label: 'Cancelled', group: 'critical' },
}

const STATUS_ORDER = ['routed', 'accepted', 'in_progress', 'completed', 'declined', 'cancelled']

const GROUP_COLOR: Record<StatusMeta['group'], string> = {
  pending: 'var(--chart-neutral)',
  active: 'var(--chart-warning)',
  good: 'var(--chart-good)',
  critical: 'var(--chart-critical)',
}

const LEGEND: { group: StatusMeta['group']; label: string }[] = [
  { group: 'pending', label: 'Pending' },
  { group: 'active', label: 'Active' },
  { group: 'good', label: 'Completed' },
  { group: 'critical', label: 'Stopped' },
]

export function ReferralsStatusChart({ counts }: { counts: Record<string, number> }) {
  const [showTable, setShowTable] = useState(false)

  const rows = STATUS_ORDER.map((status) => ({
    status,
    meta: STATUS_META[status],
    count: counts[status] ?? 0,
  }))
  const max = Math.max(1, ...rows.map((r) => r.count))
  const total = rows.reduce((sum, r) => sum + r.count, 0)

  return (
    <div className="chart-card">
      <div className="chart-card__header">
        <div>
          <h2 className="chart-card__title">Referrals by status</h2>
          <p className="chart-card__subtitle">{total} referrals total</p>
        </div>
        <div className="chart-card__actions">
          <div className="chart-legend">
            {LEGEND.map((l) => (
              <span key={l.group} className="chart-legend__item">
                <span className="chart-legend__swatch" style={{ background: GROUP_COLOR[l.group] }} />
                {l.label}
              </span>
            ))}
          </div>
          <button
            type="button"
            className="icon-button"
            onClick={() => setShowTable((v) => !v)}
            aria-label={showTable ? 'Show chart view' : 'Show table view'}
            aria-pressed={showTable}
          >
            <Table2 size={16} aria-hidden="true" />
          </button>
        </div>
      </div>

      {showTable ? (
        <table className="chart-table">
          <thead>
            <tr>
              <th>Status</th>
              <th className="chart-table__num">Count</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.status}>
                <td>{r.meta.label}</td>
                <td className="chart-table__num">{r.count}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <div className="bar-chart" role="img" aria-label={rows.map((r) => `${r.meta.label}: ${r.count}`).join(', ')}>
          {rows.map((r) => (
            <div className="bar-chart__row" key={r.status}>
              <span className="bar-chart__label">{r.meta.label}</span>
              <div className="bar-chart__track">
                <div
                  className="bar-chart__fill"
                  style={{ width: `${(r.count / max) * 100}%`, background: GROUP_COLOR[r.meta.group] }}
                />
              </div>
              <span className="bar-chart__value">{r.count}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
