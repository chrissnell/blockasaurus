// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

// Single tagged auth state.
// Separate loading/user/setupRequired booleans can update non-atomically
// across async boundaries, producing a flash of the Login page before a
// transition to Setup. Driving the {#if} chain off a single status tag
// eliminates this class of flicker.
//
// status ∈ 'loading' | 'setup' | 'anon' | 'authed'
// user is populated only when status === 'authed'.
export const authState = $state({
  status: 'loading',
  user: null,
})

export function getUser() { return authState.user }
export function isAdmin() { return authState.user?.role === 'admin' }

// Atomic status transition: always set status and user together.
function setAuthed(user) {
  authState.user = user
  authState.status = 'authed'
}

function setAnon() {
  authState.user = null
  authState.status = 'anon'
}

function setSetup() {
  authState.user = null
  authState.status = 'setup'
}

async function parseJSONSafe(resp) {
  try {
    return await resp.json()
  } catch {
    return {}
  }
}

// checkSession() runs once on App mount. Subsequent transitions come from
// explicit action calls (login/logout/setup) or the api.js 401 handler.
export async function checkSession() {
  try {
    const resp = await fetch('/api/auth/session', {
      credentials: 'include',
      headers: { 'X-Requested-With': 'fetch' },
    })
    if (resp.status === 200) {
      const body = await parseJSONSafe(resp)
      setAuthed(body.data || null)
      return
    }
    if (resp.status === 401) {
      const body = await parseJSONSafe(resp)
      if (body.error === 'setup_required') setSetup()
      else setAnon()
      return
    }
    console.warn('unexpected /api/auth/session status:', resp.status)
    setAnon()
  } catch (err) {
    console.warn('session probe failed:', err)
    setAnon()
  }
}

// Consume and navigate to any stashed deep-link return path. Falls back to /.
function consumeReturnPath() {
  const returnTo = sessionStorage.getItem('authReturnTo')
  sessionStorage.removeItem('authReturnTo')
  const target = returnTo && returnTo !== '/login' ? returnTo : '/'
  window.location.hash = `#${target}`
}

// Auth actions use raw fetch (not api.js's request helper) because they are
// pre-session or self-referential: the 401 redirect in api.js must not fire
// on /api/auth/* URLs. Even so, we include credentials + X-Requested-With.
async function authFetch(url, body) {
  const resp = await fetch(url, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      'X-Requested-With': 'fetch',
    },
    body: JSON.stringify(body),
  })

  const data = await parseJSONSafe(resp)

  if (resp.status === 429) {
    const retry = parseInt(resp.headers.get('Retry-After') || '60', 10)
    const err = new Error(data.message || 'too many attempts')
    err.retryAfter = Number.isFinite(retry) ? retry : 60
    err.status = 429
    err.error = data.error
    throw err
  }

  if (!resp.ok) {
    const err = new Error(data.message || data.error || `${resp.status} ${resp.statusText}`)
    err.status = resp.status
    err.error = data.error
    throw err
  }

  return data
}

export async function login(username, password) {
  const data = await authFetch('/api/auth/login', { username, password })
  setAuthed(data.data || null)
  consumeReturnPath()
  return authState.user
}

export async function setup(username, password) {
  const data = await authFetch('/api/auth/setup', { username, password })
  setAuthed(data.data || null)
  consumeReturnPath()
  return authState.user
}

export async function changePassword(currentPassword, newPassword) {
  const data = await authFetch('/api/auth/password', {
    current_password: currentPassword,
    new_password: newPassword,
  })
  return data
}

export async function logout() {
  try {
    await fetch('/api/auth/logout', {
      method: 'POST',
      credentials: 'include',
      headers: { 'X-Requested-With': 'fetch' },
    })
  } catch (err) {
    // Best-effort; server session will expire regardless.
    console.warn('logout request failed:', err)
  }
  setAnon()
  window.location.hash = '#/login'
}
