import { useMemo, useState, type ReactNode } from 'react'
import { ChevronDown, ChevronUp, Inbox, Search } from 'lucide-react'
import { ErrorState } from './StatusMessage'

export interface DataTableColumn<T> {
  key: string
  header: string
  render: (row: T) => ReactNode
  /** Raw comparable value for sorting/search — falls back to `render`'s output when omitted. */
  value?: (row: T) => string | number
  sortable?: boolean
  align?: 'left' | 'right'
}

interface DataTableProps<T> {
  columns: DataTableColumn<T>[]
  data: T[] | null
  getRowKey: (row: T) => string
  loading?: boolean
  error?: string | null
  searchPlaceholder?: string
  onRowClick?: (row: T) => void
  emptyMessage?: string
  pageSize?: number
}

// A single reusable table used across Users, Roles, Patients, and
// Referrals: client-side search + sort + pagination. Client-side is a
// deliberate simplification — these lists run in the low hundreds of rows
// at this stage, not the tens of thousands a server-paginated table would
// justify.
export function DataTable<T>({
  columns,
  data,
  getRowKey,
  loading,
  error,
  searchPlaceholder = 'Search…',
  onRowClick,
  emptyMessage = 'Nothing here yet.',
  pageSize = 10,
}: DataTableProps<T>) {
  const [search, setSearch] = useState('')
  const [sortKey, setSortKey] = useState<string | null>(null)
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc')
  const [page, setPage] = useState(0)

  const filtered = useMemo(() => {
    if (!data) return []
    if (!search.trim()) return data
    const term = search.trim().toLowerCase()
    return data.filter((row) =>
      columns.some((col) => String(col.value ? col.value(row) : col.render(row)).toLowerCase().includes(term)),
    )
  }, [data, search, columns])

  const sorted = useMemo(() => {
    if (!sortKey) return filtered
    const col = columns.find((c) => c.key === sortKey)
    if (!col) return filtered
    const getVal = col.value ?? ((row: T) => String(col.render(row)))
    const copy = [...filtered]
    copy.sort((a, b) => {
      const av = getVal(a)
      const bv = getVal(b)
      const cmp = typeof av === 'number' && typeof bv === 'number' ? av - bv : String(av).localeCompare(String(bv))
      return sortDir === 'asc' ? cmp : -cmp
    })
    return copy
  }, [filtered, sortKey, sortDir, columns])

  const pageCount = Math.max(1, Math.ceil(sorted.length / pageSize))
  const currentPage = Math.min(page, pageCount - 1)
  const pageRows = sorted.slice(currentPage * pageSize, currentPage * pageSize + pageSize)

  function handleSort(col: DataTableColumn<T>) {
    if (!col.sortable) return
    if (sortKey === col.key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(col.key)
      setSortDir('asc')
    }
    setPage(0)
  }

  if (error) return <ErrorState message={error} />

  return (
    <div className="data-table">
      <div className="data-table__toolbar">
        <div className="data-table__search">
          <Search size={16} aria-hidden="true" />
          <input
            type="search"
            placeholder={searchPlaceholder}
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              setPage(0)
            }}
            disabled={loading}
            aria-label={searchPlaceholder}
          />
        </div>
        <span className="data-table__count">
          {loading ? '—' : `${sorted.length} ${sorted.length === 1 ? 'row' : 'rows'}`}
        </span>
      </div>

      {loading ? (
        <div className="data-table__scroll">
          <table>
            <thead>
              <tr>
                {columns.map((col) => (
                  <th key={col.key}>{col.header}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {Array.from({ length: 5 }).map((_, i) => (
                <tr key={i}>
                  {columns.map((col) => (
                    <td key={col.key}>
                      <span className="skeleton-bar" />
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : sorted.length === 0 ? (
        <div className="empty-state empty-state--block">
          <Inbox size={28} aria-hidden="true" />
          <p>{emptyMessage}</p>
        </div>
      ) : (
        <>
          <div className="data-table__scroll">
            <table>
              <thead>
                <tr>
                  {columns.map((col) => (
                    <th
                      key={col.key}
                      className={col.align === 'right' ? 'data-table__align-right' : undefined}
                      aria-sort={sortKey === col.key ? (sortDir === 'asc' ? 'ascending' : 'descending') : undefined}
                    >
                      {col.sortable ? (
                        <button type="button" className="data-table__sort" onClick={() => handleSort(col)}>
                          {col.header}
                          {sortKey === col.key &&
                            (sortDir === 'asc' ? (
                              <ChevronUp size={14} aria-hidden="true" />
                            ) : (
                              <ChevronDown size={14} aria-hidden="true" />
                            ))}
                        </button>
                      ) : (
                        col.header
                      )}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {pageRows.map((row) => (
                  <tr
                    key={getRowKey(row)}
                    className={onRowClick ? 'data-table__row--clickable' : undefined}
                    onClick={onRowClick ? () => onRowClick(row) : undefined}
                  >
                    {columns.map((col) => (
                      <td key={col.key} className={col.align === 'right' ? 'data-table__align-right' : undefined}>
                        {col.render(row)}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {pageCount > 1 && (
            <div className="data-table__pagination">
              <button type="button" disabled={currentPage === 0} onClick={() => setPage((p) => p - 1)}>
                Previous
              </button>
              <span>
                Page {currentPage + 1} of {pageCount}
              </span>
              <button type="button" disabled={currentPage >= pageCount - 1} onClick={() => setPage((p) => p + 1)}>
                Next
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
