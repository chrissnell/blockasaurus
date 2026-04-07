// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

const BASE = '/api/config'

async function request(method, path, body) {
  const opts = {
    method,
    headers: {},
  }

  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json'
    opts.body = JSON.stringify(body)
  }

  const resp = await fetch(BASE + path, opts)

  if (resp.status === 204) return null

  const text = await resp.text()
  let data
  try {
    data = JSON.parse(text)
  } catch {
    if (!resp.ok) throw new Error(text || `${resp.status} ${resp.statusText}`)
    throw new Error(`unexpected response: ${text.slice(0, 200)}`)
  }

  if (!resp.ok) {
    throw new Error(data.message || data.error || `${resp.status} ${resp.statusText}`)
  }

  return data
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

// Discovered Clients (ARP-based network discovery)
export async function getDiscoveredClients() {
  const resp = await fetch('/api/discovered-clients')
  if (!resp.ok) return []
  return resp.json()
}

// Apply
export const apply = () => request('POST', '/apply')

// Stats
export async function getStats() {
  const resp = await fetch('/api/stats')
  if (!resp.ok) return null
  return resp.json()
}

export async function getStatsOvertime() {
  const resp = await fetch('/api/stats/overtime')
  if (!resp.ok) return null
  return resp.json()
}

export async function getStatsOvertimeClients() {
  const resp = await fetch('/api/stats/overtime/clients')
  if (!resp.ok) return null
  return resp.json()
}

export async function getStatsQueryTypes() {
  const resp = await fetch('/api/stats/query-types')
  if (!resp.ok) return null
  return resp.json()
}

export async function getStatsResponseTypes() {
  const resp = await fetch('/api/stats/response-types')
  if (!resp.ok) return null
  return resp.json()
}

export async function getStatsTopDomains() {
  const resp = await fetch('/api/stats/top-domains')
  if (!resp.ok) return null
  return resp.json()
}

export async function getStatsTopClients() {
  const resp = await fetch('/api/stats/top-clients')
  if (!resp.ok) return null
  return resp.json()
}

// Endpoint Info (client group endpoint configuration)
export async function getEndpointInfo() {
  const resp = await fetch('/api/endpoint-info')
  if (!resp.ok) return null
  return resp.json()
}

// Version
export async function getVersion() {
  const resp = await fetch('/api/version')
  if (!resp.ok) return ''
  const data = await resp.json()
  return data.version || ''
}
