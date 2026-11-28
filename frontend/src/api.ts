import type {
  Job, Worker, PublicUser, Video, VideoMetadata, Metrics, AppEvent,
} from './types'

const API_BASE = '/api'

export class ApiError extends Error {
  constructor(message: string, public status: number, public errors?: string[]) {
    super(message)
  }
}

const getAccessToken  = () => localStorage.getItem('access_token')
const getRefreshToken = () => localStorage.getItem('refresh_token')

// Dispatched (after tokens are cleared) when the session cannot be restored:
// the AuthProvider listens for it and RequireAuth redirects to /login.
export const AUTH_EXPIRED_EVENT = 'auth:expired'

function notifyAuthExpired() {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
  window.dispatchEvent(new Event(AUTH_EXPIRED_EVENT))
}

let refreshing: Promise<boolean> | null = null

async function doRefresh(): Promise<boolean> {
  if (refreshing) return refreshing
  refreshing = (async () => {
    const rt = getRefreshToken()
    if (!rt) return false
    try {
      const res = await fetch(`${API_BASE}/users/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: rt }),
      })
      if (!res.ok) return false
      const data = await res.json()
      if (!data?.access_token || !data?.refresh_token) return false
      localStorage.setItem('access_token', data.access_token)
      localStorage.setItem('refresh_token', data.refresh_token)
      return true
    } catch {
      return false
    }
  })()
  try { return await refreshing } finally { refreshing = null }
}

async function request<T>(
  path: string,
  options: RequestInit = {},
  auth = true,
  retry = true,
): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((options.headers as Record<string, string>) || {}),
  }
  if (auth) {
    const t = getAccessToken()
    if (t) headers['Authorization'] = `Bearer ${t}`
  }
  const res = await fetch(`${API_BASE}${path}`, { ...options, headers })

  if (res.status === 401 && auth && retry) {
    if (await doRefresh()) return request<T>(path, options, auth, false)
    // Refresh failed: the session is over, drop tokens and let the app react
    notifyAuthExpired()
  }

  if (!res.ok) {
    let errors: string[] | undefined
    try { errors = (await res.json()).errors } catch {}
    throw new ApiError(errors?.join('; ') || `HTTP ${res.status}`, res.status, errors)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

/**
 * EventSource wrapper that authenticates via a `token` query parameter
 * (EventSource cannot set an Authorization header) and transparently
 * reconnects with a fresh token after the access token is refreshed.
 */
export class EventStream {
  private listeners = new Map<string, Set<(e: MessageEvent) => void>>()
  private es: EventSource | null = null
  private stopped = false
  private attempts = 0

  constructor(private path: string) {
    this.connect()
  }

  private connect(delay = 0) {
    if (this.stopped) return
    const attempt = () => {
      if (this.stopped) return
      const token = getAccessToken() || ''
      const sep = this.path.includes('?') ? '&' : '?'
      const es = new EventSource(
        `${API_BASE}${this.path}${sep}token=${encodeURIComponent(token)}`,
      )
      this.es = es

      es.onopen = () => { this.attempts = 0 }
      es.onerror = () => {
        if (this.stopped) return
        // CLOSED means the server rejected the connection (e.g. expired or
        // missing token). CONNECTING means a transient network issue and the
        // browser will retry on its own.
        if (es.readyState !== EventSource.CLOSED) return
        es.close()
        void (async () => {
          const ok = await doRefresh()
          if (this.stopped) return
          if (!ok) { notifyAuthExpired(); return }
          this.attempts++
          this.connect(Math.min(5000, 250 * this.attempts))
        })()
      }
    }
    if (delay > 0) setTimeout(attempt, delay)
    else attempt()
  }

  addEventListener(type: string, listener: (e: MessageEvent) => void) {
    let set = this.listeners.get(type)
    if (!set) {
      set = new Set()
      this.listeners.set(type, set)
    }
    set.add(listener)
    this.es?.addEventListener(type, listener as EventListener)
    return this
  }

  close() {
    this.stopped = true
    this.es?.close()
  }
}

/** Accept both the array snapshots sent on connect and single live objects */
function asArray<T>(data: unknown): T[] {
  return Array.isArray(data) ? data as T[] : [data as T]
}

export const api = {
  asArray,

  // auth
  register: (username: string, email: string, password: string) =>
    request<PublicUser>('/users/register', {
      method: 'POST', body: JSON.stringify({ username, email, password }),
    }, false),

  login: (email: string, password: string) =>
    request<{ access_token: string; refresh_token: string }>('/users/login', {
      method: 'POST', body: JSON.stringify({ email, password }),
    }, false),

  // users
  listUsers: () => request<PublicUser[]>('/users'),
  getUser: (id: number) => request<PublicUser>(`/users/${id}`),
  patchUser: (id: number, patch: any) =>
    request<PublicUser>(`/users/${id}`, { method: 'PATCH', body: JSON.stringify(patch) }),
  changePassword: (id: number, current_password: string, new_password: string) =>
    request<void>(`/users/${id}/password`, {
      method: 'POST', body: JSON.stringify({ current_password, new_password }),
    }),
  changeRole: (id: number, user_role: string) =>
    request<PublicUser>(`/users/${id}/role`, {
      method: 'PATCH', body: JSON.stringify({ user_role }),
    }),
  deleteUser: (id: number) => request<void>(`/users/${id}`, { method: 'DELETE' }),

  // jobs
  listJobs: () => request<Job[]>('/jobs', {}, false),
  createJob: (config: any) =>
    request<Job>('/jobs', { method: 'POST', body: JSON.stringify(config) }),
  getJob: (id: number) => request<Job>(`/jobs/${id}`),
  patchJob: (id: number, patch: any) =>
    request<Job>(`/jobs/${id}`, { method: 'PATCH', body: JSON.stringify(patch) }),
  deleteJob: (id: number) => request<void>(`/jobs/${id}`, { method: 'DELETE' }),
  startJob:  (id: number) => request<void>(`/jobs/${id}/start`,  { method: 'POST' }),
  pauseJob:  (id: number) => request<void>(`/jobs/${id}/pause`,  { method: 'POST' }),
  resumeJob: (id: number) => request<void>(`/jobs/${id}/resume`, { method: 'POST' }),
  stopJob:   (id: number) => request<void>(`/jobs/${id}/stop`,   { method: 'POST' }),
  getJobItems: (id: number) => request<number[]>(`/jobs/${id}/items`),
  getJobEvents: (id: number) => request<AppEvent[]>(`/jobs/${id}/events`),

  // workers
  listWorkers: () => request<Worker[]>('/workers', {}, false),
  getWorker: (id: number) => request<Worker>(`/workers/${id}`),
  getWorkerMetrics: (id: number) => request<Metrics>(`/workers/${id}/metrics`),
  patchWorker: (id: number, patch: any) =>
    request<Worker>(`/workers/${id}`, { method: 'PATCH', body: JSON.stringify(patch) }),
  deleteWorker: (id: number) => request<void>(`/workers/${id}`, { method: 'DELETE' }),
  registerWorker: (payload: any) =>
    request<Worker>('/workers/register', { method: 'POST', body: JSON.stringify(payload) }),

  // service
  search: (q: string, limit: number, offset: number) =>
    request<{ items: Video[]; quantity: number }>(
      `/service/search?q=${encodeURIComponent(q)}&limit=${limit}&offset=${offset}`, {}, false),
  recommended: (limit: number, offset: number) =>
    request<{ items: Video[]; quantity: number }>(
      `/service/recommended?limit=${limit}&offset=${offset}`, {}, false),
  getVideo: (id: number) =>
    request<{ video_id: number; url: string; metadata: VideoMetadata }>(
      `/service/video/${id}`, {}, false),
  getRecommendedForVideo: (id: number, limit: number, offset: number) =>
    request<{ items: Video[]; quantity: number }>(
      `/service/video/${id}/recommended?limit=${limit}&offset=${offset}`, {}, false),
}
