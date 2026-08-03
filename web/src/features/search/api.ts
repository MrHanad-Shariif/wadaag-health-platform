import { apiClient } from '../../api/client'
import type { SavedSearch, SearchResults, SearchResultType } from './types'

export function search(query: string, types?: SearchResultType[]) {
  const params = new URLSearchParams({ q: query })
  if (types && types.length > 0) {
    params.set('type', types.join(','))
  }
  return apiClient.get<SearchResults>(`/api/v1/search?${params.toString()}`)
}

export function createSavedSearch(name: string, query: string, filters: Record<string, unknown>) {
  return apiClient.post<SavedSearch>('/api/v1/search/saved', { name, query, filters })
}

export function listSavedSearches() {
  return apiClient.get<SavedSearch[]>('/api/v1/search/saved')
}

export function deleteSavedSearch(id: string) {
  return apiClient.del<void>(`/api/v1/search/saved/${id}`)
}
