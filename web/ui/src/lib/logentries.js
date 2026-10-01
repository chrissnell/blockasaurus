// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

// Entry shaping, bounded buffering and search for the live log viewer.
// Kept out of Logs.svelte so the parts with behaviour worth asserting can be
// tested without a DOM.

// Matches the server-side ring the stream backfills from
// (logstream.NewBroadcaster(ctx, 1000)): anything past this is already
// unrecoverable, and keeping more only grows a long-lived tab.
export const MAX_ENTRIES = 1000

function mapLevel(lvl) {
  const l = (lvl || '').toLowerCase()
  if (l === 'error') return 'error'
  if (l === 'warn' || l === 'warning') return 'warn'
  if (l === 'debug') return 'debug'

  return 'info'
}

// Shape one raw log-stream message into a LogViewer row.
export function toLogEntry(raw) {
  const f = raw.fields || {}
  const blocked = f.response_type === 'BLOCKED'

  return {
    level: blocked ? 'error' : mapLevel(raw.level),
    // Explicit flag: level is also 'error' for genuine error logs, so the
    // blocked-only filter cannot key off it.
    blocked,
    timestamp: raw.timestamp,
    client_ip: f.client_ip,
    client_names: f.client_names,
    client_group: f.client_group,
    duration_ms: f.duration_ms,
    question_type: f.question_type,
    question_name: f.question_name || raw.message,
    response_code: f.response_code,
    response_reason: f.response_reason,
  }
}

// Append rows oldest-first, dropping from the front once the cap is reached.
export function appendBounded(entries, incoming, max = MAX_ENTRIES) {
  if (incoming.length === 0) return entries

  const next = entries.concat(incoming)

  return next.length > max ? next.slice(next.length - max) : next
}

function contains(value, needle) {
  return value != null && String(value).toLowerCase().includes(needle)
}

// Rows matching the search text and the blocked-only toggle. `search` is
// matched case-insensitively as a substring of the queried domain or of the
// client -- its IP, or its resolved name when the IP is not what you remember.
// Returns `entries` itself when nothing is filtering, so the common case costs
// no copy.
export function filterEntries(entries, search, blockedOnly) {
  const query = search.trim().toLowerCase()
  if (query === '' && !blockedOnly) return entries

  return entries.filter((e) => {
    if (blockedOnly && !e.blocked) return false
    if (query === '') return true

    return contains(e.question_name, query)
      || contains(e.client_ip, query)
      || contains(e.client_names, query)
  })
}
