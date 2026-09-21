import type { Contributor, Favorite, Issue, Page, RateLimit, Release, Repository, SearchResult } from './types'

type APIErrorBody = { error?: { code?: string; message?: string; request_id?: string } }
export class APIError extends Error { constructor(message: string, public status: number, public requestId?: string) { super(message) } }
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, { ...init, headers: { Accept: 'application/json', ...(init?.body ? { 'Content-Type': 'application/json' } : {}), ...init?.headers } })
  if (!response.ok) { let body: APIErrorBody = {}; try { body = await response.json() } catch { /* non-JSON proxy error */ } throw new APIError(body.error?.message || `HTTP ${response.status}`, response.status, body.error?.request_id) }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}
export const api = {
  search: (p: URLSearchParams) => request<SearchResult>(`/api/v1/repositories/search?${p}`),
  repository: (o: string, r: string) => request<Repository>(`/api/v1/repositories/${encodeURIComponent(o)}/${encodeURIComponent(r)}`),
  languages: (o: string, r: string) => request<{languages: Record<string, number>}>(`/api/v1/repositories/${encodeURIComponent(o)}/${encodeURIComponent(r)}/languages`),
  contributors: (o: string, r: string) => request<Page<Contributor>>(`/api/v1/repositories/${encodeURIComponent(o)}/${encodeURIComponent(r)}/contributors?per_page=12`),
  issues: (o: string, r: string) => request<Page<Issue>>(`/api/v1/repositories/${encodeURIComponent(o)}/${encodeURIComponent(r)}/issues?per_page=10`),
  releases: (o: string, r: string) => request<Page<Release>>(`/api/v1/repositories/${encodeURIComponent(o)}/${encodeURIComponent(r)}/releases?per_page=5`),
  rate: () => request<RateLimit>('/api/v1/rate-limit'),
  favorites: () => request<{items: Favorite[]; count: number}>('/api/v1/favorites'),
  addFavorite: (owner: string, repo: string) => request<Favorite>('/api/v1/favorites', { method: 'POST', body: JSON.stringify({ owner, repo }) }),
  removeFavorite: (owner: string, repo: string) => request<void>(`/api/v1/favorites/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`, { method: 'DELETE' }),
}
