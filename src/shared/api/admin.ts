export type AdminApi = <T>(path: string, init?: RequestInit) => Promise<T>

export function createAdminApi(getToken: () => string, onUnauthorized: () => void): AdminApi {
  return async <T>(path: string, init: RequestInit = {}): Promise<T> => {
    const headers = new Headers(init.headers)
    headers.set('Authorization', `Bearer ${getToken()}`)
    if (init.body && !(init.body instanceof FormData))
      headers.set('Content-Type', 'application/json')
    const res = await fetch(`/api/admin${path}`, {
      ...init,
      headers,
      signal: AbortSignal.timeout(60000),
    })
    if (!res.ok) {
      if (res.status === 401) onUnauthorized()
      const body = await res.json().catch(() => ({ error: `Request failed (${res.status})` }))
      throw new Error(body.error || 'Request failed')
    }
    return res.status === 204 ? (undefined as T) : res.json()
  }
}
