import { useEffect, useState } from 'react'
import { Bookmark, Trash2 } from 'lucide-react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { PageHeader } from '../../shared/PageHeader'
import { ErrorState, LoadingState } from '../../shared/StatusMessage'
import { useFetch } from '../../shared/useFetch'
import { useToast } from '../../shared/useToast'
import { createSavedSearch, deleteSavedSearch, listSavedSearches, search } from './api'
import type { SavedSearch, SearchResult, SearchResults, SearchResultType } from './types'

const FACETS: { type: SearchResultType; label: string }[] = [
  { type: 'patient', label: 'Patients' },
  { type: 'doctor', label: 'Doctors' },
  { type: 'hospital', label: 'Hospitals' },
  { type: 'referral', label: 'Referrals' },
  { type: 'consultation', label: 'Consultations' },
]

function routeFor(result: SearchResult): string {
  switch (result.type) {
    case 'patient':
      return `/patients/${result.id}`
    case 'doctor':
      return `/providers/${result.id}`
    case 'hospital':
      return `/facilities/${result.id}`
    case 'referral':
      return `/referrals/${result.id}`
    case 'consultation':
      return `/consults/${result.id}`
    default:
      return '/'
  }
}

export function SearchResultsPage() {
  const navigate = useNavigate()
  const toast = useToast()
  const [searchParams, setSearchParams] = useSearchParams()
  const q = searchParams.get('q')?.trim() ?? ''
  const activeType = (searchParams.get('type') as SearchResultType | null) ?? null

  const state = useFetch<SearchResults>(() => {
    if (!q) return Promise.resolve({})
    return search(q, activeType ? [activeType] : undefined)
  }, [q, activeType])

  const [saved, setSaved] = useState<SavedSearch[] | null>(null)
  const [savedError, setSavedError] = useState<string | null>(null)

  function loadSaved() {
    listSavedSearches()
      .then((list) => setSaved(list))
      .catch((err) => setSavedError(err instanceof ApiError ? err.message : 'Failed to load saved searches.'))
  }

  useEffect(() => {
    loadSaved()
  }, [])

  function handleFilter(type: SearchResultType | null) {
    const next = new URLSearchParams(searchParams)
    if (type) next.set('type', type)
    else next.delete('type')
    setSearchParams(next)
  }

  function handleSave() {
    if (!q) return
    const name = window.prompt('Name this saved search:', q)
    if (!name) return
    createSavedSearch(name, q, activeType ? { type: activeType } : {})
      .then((created) => {
        setSaved((prev) => (prev ? [created, ...prev] : [created]))
        toast.show('Search saved.', 'success')
      })
      .catch((err) => toast.show(err instanceof ApiError ? err.message : 'Failed to save search.', 'error'))
  }

  function handleRunSaved(savedSearch: SavedSearch) {
    const type = typeof savedSearch.filters.type === 'string' ? (savedSearch.filters.type as SearchResultType) : null
    const next = new URLSearchParams()
    next.set('q', savedSearch.query)
    if (type) next.set('type', type)
    setSearchParams(next)
  }

  function handleDeleteSaved(id: string) {
    setSaved((prev) => prev?.filter((s) => s.id !== id) ?? prev)
    deleteSavedSearch(id).catch(() => {
      toast.show('Failed to delete saved search.', 'error')
      loadSaved()
    })
  }

  const results = state.data ?? {}
  const groups = FACETS.filter((f) => (results[f.type]?.length ?? 0) > 0)
  const totalCount = FACETS.reduce((sum, f) => sum + (results[f.type]?.length ?? 0), 0)

  return (
    <div className="page page--wide">
      <PageHeader
        eyebrow="Search"
        title={q ? `Results for "${q}"` : 'Search'}
        description={q ? undefined : 'Use the search bar in the top navigation to get started.'}
        actions={
          q ? (
            <button type="button" className="button-link" onClick={handleSave}>
              <Bookmark size={16} aria-hidden="true" />
              Save this search
            </button>
          ) : undefined
        }
      />

      {q && (
        <div className="search-filters">
          <button
            type="button"
            className={`search-filter${activeType === null ? ' search-filter--active' : ''}`}
            onClick={() => handleFilter(null)}
          >
            All
          </button>
          {FACETS.map((f) => (
            <button
              type="button"
              key={f.type}
              className={`search-filter${activeType === f.type ? ' search-filter--active' : ''}`}
              onClick={() => handleFilter(f.type)}
            >
              {f.label}
            </button>
          ))}
        </div>
      )}

      {savedError && <ErrorState message={savedError} />}
      {saved && saved.length > 0 && (
        <div className="saved-searches">
          <span className="saved-searches__label">Saved searches</span>
          <div className="saved-searches__list">
            {saved.map((s) => (
              <span className="saved-search-chip" key={s.id}>
                <button type="button" onClick={() => handleRunSaved(s)} title={s.query}>
                  {s.name}
                </button>
                <button
                  type="button"
                  className="saved-search-chip__delete"
                  aria-label={`Delete saved search "${s.name}"`}
                  onClick={() => handleDeleteSaved(s.id)}
                >
                  <Trash2 size={13} aria-hidden="true" />
                </button>
              </span>
            ))}
          </div>
        </div>
      )}

      {!q ? null : state.loading ? (
        <LoadingState />
      ) : state.error ? (
        <ErrorState message={state.error} />
      ) : totalCount === 0 ? (
        <div className="empty-state empty-state--block">
          <p>No results for &quot;{q}&quot;.</p>
        </div>
      ) : (
        <div className="search-results">
          {groups.map((f) => (
            <section key={f.type} className="search-results__group">
              <h2>{f.label}</h2>
              <ul className="record-list">
                {results[f.type]!.map((r) => (
                  <li key={r.id}>
                    <a
                      href={routeFor(r)}
                      onClick={(e) => {
                        e.preventDefault()
                        navigate(routeFor(r))
                      }}
                    >
                      {r.title}
                    </a>
                    {r.subtitle && <span className="record-list__note">{r.subtitle}</span>}
                    <span className="record-list__timestamp">{new Date(r.created_at).toLocaleDateString()}</span>
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </div>
      )}
    </div>
  )
}
