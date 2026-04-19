// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

const BASE = '/api/config'

// Handle session expiry on any non-auth URL: preserve the intended hash
// destination and redirect to #/login. Throws 'unauthorized' so callers that
// feed the result into .map() / {#each} don't TypeError on undefined.
function handle401(url) {
  // Loop guard: probing /api/auth/session legitimately returns 401 at boot.
  if (url.includes('/api/auth/')) return
  const current = window.location.hash.slice(1) || '/'
  if (current !== '/login' && current !== '/') {
    sessionStorage.setItem('authReturnTo', current)
  }
  window.location.hash = '#/login'
}

async function parseBody(resp) {
  if (resp.status === 204) return null
  const text = await resp.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    if (!resp.ok) throw new Error(text || `${resp.status} ${resp.statusText}`)
    throw new Error(`unexpected response: ${text.slice(0, 200)}`)
  }
}

function makeRateLimitError(resp, data) {
  const retry = parseInt(resp.headers.get('Retry-After') || '60', 10)
  const err = new Error((data && data.message) || 'rate limited')
  err.retryAfter = Number.isFinite(retry) ? retry : 60
  err.status = 429
  err.error = data && data.error
  return err
}

async function request(method, path, body) {
  const opts = {
    method,
    headers: {},
    credentials: 'include',
  }

  // CSRF header required on mutations (Phase 3 middleware).
  if (method !== 'GET') {
    opts.headers['X-Requested-With'] = 'fetch'
  }

  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json'
    opts.body = JSON.stringify(body)
  }

  const url = BASE + path
  const resp = await fetch(url, opts)

  if (resp.status === 401) {
    handle401(url)
    // Throw, not return — callers feed the result into {#each} / .map();
    // returning undefined would TypeError before navigation takes effect.
    throw new Error('unauthorized')
  }

  if (resp.status === 429) {
    const data = await parseBody(resp).catch(() => ({}))
    throw makeRateLimitError(resp, data)
  }

  const data = await parseBody(resp)

  if (!resp.ok) {
    const err = new Error(
      (data && (data.message || data.error)) || `${resp.status} ${resp.statusText}`
    )
    err.status = resp.status
    err.error = data && data.error
    throw err
  }

  return data
}

// Generic authRequest for non-config URLs (e.g. /api/auth/users).
// Mirrors request() semantics: 401 redirects + throws, 429 surfaces retryAfter.
// Optional `signal` lets callers cancel on unmount; AbortError bubbles up so
// callers can ignore it silently.
export async function authRequest(method, url, body, signal) {
  const opts = {
    method,
    headers: {},
    credentials: 'include',
  }

  if (method !== 'GET') {
    opts.headers['X-Requested-With'] = 'fetch'
  }

  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json'
    opts.body = JSON.stringify(body)
  }

  if (signal) opts.signal = signal

  const resp = await fetch(url, opts)

  if (resp.status === 401) {
    handle401(url)
    throw new Error('unauthorized')
  }

  if (resp.status === 429) {
    const data = await parseBody(resp).catch(() => ({}))
    throw makeRateLimitError(resp, data)
  }

  const data = await parseBody(resp)

  if (!resp.ok) {
    const err = new Error(
      (data && (data.message || data.error)) || `${resp.status} ${resp.statusText}`
    )
    err.status = resp.status
    err.error = data && data.error
    throw err
  }

  return data
}

// authAPI: thin surface for the auth/user endpoints used by pages.
// login/setup/password go through auth.svelte.js so they can update state;
// users CRUD / session probes use authRequest directly.
export const authAPI = {
  session: () => authRequest('GET', '/api/auth/session'),
  listUsers: (signal) => authRequest('GET', '/api/auth/users', undefined, signal),
  createUser: (body) => authRequest('POST', '/api/auth/users', body),
  deleteUser: (id) => authRequest('DELETE', `/api/auth/users/${id}`),
}

// Client Groups
export const clientGroups = {
  list: () => request('GET', '/client-groups'),
  get: (name) => request('GET', `/client-groups/${name}`),
  put: (name, body) => request('PUT', `/client-groups/${name}`, body),
  delete: (name) => request('DELETE', `/client-groups/${name}`),
}

// Blocklist Sources
export const blocklistSources = {
  list: (params) => {
    const qs = params ? '?' + new URLSearchParams(params) : ''
    return request('GET', `/blocklist-sources${qs}`)
  },
  get: (id) => request('GET', `/blocklist-sources/${id}`),
  create: (body) => request('POST', '/blocklist-sources', body),
  update: (id, body) => request('PUT', `/blocklist-sources/${id}`, body),
  delete: (id) => request('DELETE', `/blocklist-sources/${id}`),
}

// Custom DNS
export const customDNS = {
  list: (params) => {
    const qs = params ? '?' + new URLSearchParams(params) : ''
    return request('GET', `/custom-dns${qs}`)
  },
  get: (id) => request('GET', `/custom-dns/${id}`),
  create: (body) => request('POST', '/custom-dns', body),
  update: (id, body) => request('PUT', `/custom-dns/${id}`, body),
  delete: (id) => request('DELETE', `/custom-dns/${id}`),
}

// Domain Entries
export const domainEntries = {
  list: (params) => {
    const qs = params ? '?' + new URLSearchParams(params) : ''
    return request('GET', `/domain-entries${qs}`)
  },
  get: (id) => request('GET', `/domain-entries/${id}`),
  create: (body) => request('POST', '/domain-entries', body),
  update: (id, body) => request('PUT', `/domain-entries/${id}`, body),
  delete: (id) => request('DELETE', `/domain-entries/${id}`),
}

// Upstream Groups
export const upstreamGroups = {
  list: () => request('GET', '/upstream-groups'),
  get: (name) => request('GET', `/upstream-groups/${encodeURIComponent(name)}`),
  put: (name) => request('PUT', `/upstream-groups/${encodeURIComponent(name)}`),
  delete: (name) => request('DELETE', `/upstream-groups/${encodeURIComponent(name)}`),
  listServers: (name) =>
    request('GET', `/upstream-groups/${encodeURIComponent(name)}/servers`),
  createServer: (name, body) =>
    request('POST', `/upstream-groups/${encodeURIComponent(name)}/servers`, body),
  updateServer: (name, id, body) =>
    request('PUT', `/upstream-groups/${encodeURIComponent(name)}/servers/${id}`, body),
  deleteServer: (name, id) =>
    request('DELETE', `/upstream-groups/${encodeURIComponent(name)}/servers/${id}`),
}

// Upstream Settings
export const upstreamSettings = {
  get: () => request('GET', '/upstream-settings'),
  update: (body) => request('PUT', '/upstream-settings', body),
}

// Block Settings
export const blockSettings = {
  get: () => request('GET', '/block-settings'),
  update: (body) => request('PUT', '/block-settings', body),
}

// Raw GET helper with 401 handling for non-config "read" endpoints that don't
// want an authAPI wrapper (stats/version/discovered-clients). Keeps the old
// "return sentinel on failure" ergonomics while still redirecting on 401.
async function rawGet(url, fallback) {
  const resp = await fetch(url, { credentials: 'include' })
  if (resp.status === 401) {
    handle401(url)
    throw new Error('unauthorized')
  }
  if (!resp.ok) return fallback
  return resp.json()
}

// Discovered Clients (ARP-based network discovery)
export async function getDiscoveredClients() {
  return rawGet('/api/discovered-clients', [])
}

// Apply
export const apply = () => request('POST', '/apply')

// Stats
export async function getStats() {
  return rawGet('/api/stats', null)
}

export async function getStatsOvertime() {
  return rawGet('/api/stats/overtime', null)
}

export async function getStatsOvertimeClients() {
  return rawGet('/api/stats/overtime/clients', null)
}

export async function getStatsQueryTypes() {
  return rawGet('/api/stats/query-types', null)
}

export async function getStatsResponseTypes() {
  return rawGet('/api/stats/response-types', null)
}

export async function getStatsTopDomains() {
  return rawGet('/api/stats/top-domains', null)
}

export async function getStatsTopClients() {
  return rawGet('/api/stats/top-clients', null)
}

// Endpoint Info (client group endpoint configuration)
export async function getEndpointInfo() {
  return rawGet('/api/endpoint-info', null)
}

// Version
export async function getVersion() {
  const resp = await fetch('/api/version', { credentials: 'include' })
  // Version is allowed to fail silently (pre-auth page chrome).
  if (!resp.ok) return ''
  const data = await resp.json().catch(() => ({}))
  return data.version || ''
}
