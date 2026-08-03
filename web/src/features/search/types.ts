export type SearchResultType = 'patient' | 'doctor' | 'hospital' | 'referral' | 'consultation'

export interface SearchResult {
  type: SearchResultType
  id: string
  title: string
  subtitle?: string
  created_at: string
}

// The backend returns an object keyed by facet type — a missing key means
// the caller doesn't have access to that facet, or it just returned nothing.
// Both cases are treated the same on the frontend (an empty group).
export type SearchResults = Partial<Record<SearchResultType, SearchResult[]>>

export interface SavedSearch {
  id: string
  name: string
  query: string
  filters: Record<string, unknown>
  created_at: string
}
