// The one data layer. Every request goes through here so a 401 is handled in exactly one place:
// remember the URL we are on and go sign in. The URL is the state, so coming back lands here.
export type Scope = { school: string; room: string; queue: string }
export type Held = { capability: string; scope: Scope }
export type Me = {
  real: string
  effective: string
  impersonating: boolean
  read_only: boolean
  capabilities: Held[] | null
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) { super(message); this.status = status }
}

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, { credentials: 'same-origin', ...init })
  if (res.status === 401) {
    sessionStorage.setItem('tightship.returnTo', location.pathname + location.search)
    location.assign('/auth/login')
    throw new ApiError(401, 'not signed in')
  }
  if (!res.ok) {
    let msg = res.statusText
    try { msg = (await res.json()).error ?? msg } catch { /* not json */ }
    throw new ApiError(res.status, msg)
  }
  return res.json() as Promise<T>
}

/** True when /me lists the capability, optionally within a school. */
export function can(me: Me | null, capability: string, school?: string): boolean {
  if (!me?.capabilities) return false
  return me.capabilities.some(h =>
    h.capability === capability && (h.scope.school === '' || school === undefined || h.scope.school === school))
}
