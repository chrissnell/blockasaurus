// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

// Query building and row shaping for the Logs page's History tab, which reads
// the persisted query log over /api/config/query-log/history. Kept out of
// Logs.svelte so the parts worth asserting can be tested without a DOM, the
// same split logentries.js makes for the live view.

// Matches the endpoint's own default page size. Pagination is keyset, so this
// is the size of each "Load more" step, not a ceiling on what can be reached.
export const HISTORY_PAGE_SIZE = 100

const HOUR = 60 * 60 * 1000

// Every history query is bounded in time: request_ts is the one indexed column
// a substring domain match can lean on, so an unbounded search is the one shape
// that degrades on a full retention window. 24h is the default because that is
// the question people actually arrive with ("what did this device just do").
// The longest option matches the 30-day retention.
export const RANGE_OPTIONS = [
  { value: '1h', label: 'Last hour', ms: HOUR },
  { value: '24h', label: 'Last 24 hours', ms: 24 * HOUR },
  { value: '7d', label: 'Last 7 days', ms: 7 * 24 * HOUR },
  { value: '30d', label: 'Last 30 days', ms: 30 * 24 * HOUR },
]

export const DEFAULT_RANGE = '24h'

function rangeOption(value) {
  return RANGE_OPTIONS.find((o) => o.value === value)
    || RANGE_OPTIONS.find((o) => o.value === DEFAULT_RANGE)
}

// Start of the window named by `value`, as a Date. An unknown value falls back
// to the default rather than producing an unbounded query.
export function rangeStart(value, now = new Date()) {
  return new Date(now.getTime() - rangeOption(value).ms)
}

export function rangeLabel(value) {
  return rangeOption(value).label
}

// Query string parameters for one page request. Blank filters are omitted
// rather than sent empty, so the server sees "no filter" and not "match the
// empty string".
export function historyParams({ client, domain, blockedOnly, from, limit, cursor } = {}) {
  const params = {}

  const trimmedClient = (client || '').trim()
  if (trimmedClient) params.client = trimmedClient

  const trimmedDomain = (domain || '').trim()
  if (trimmedDomain) params.domain = trimmedDomain

  if (blockedOnly) params.blocked_only = 'true'
  if (from) params.from = from.toISOString()

  params.limit = String(limit || HISTORY_PAGE_SIZE)

  // Keyset continuation. The token is opaque by contract: it goes back exactly
  // as it came.
  if (cursor) params.cursor = cursor

  return params
}

// Shape one API entry into a LogViewer row. The field names already match the
// live stream's, so this only has to add the row-level `level` the viewer
// colors by -- the server sends `blocked` rather than letting the client
// re-derive it from the response type.
export function toHistoryRow(entry) {
  return {
    level: entry.blocked ? 'error' : 'info',
    blocked: entry.blocked,
    timestamp: entry.timestamp,
    client_ip: entry.client_ip,
    client_names: entry.client_names,
    client_group: entry.client_group,
    duration_ms: entry.duration_ms,
    question_type: entry.question_type,
    question_name: entry.question_name,
    response_code: entry.response_code,
    response_reason: entry.response_reason,
  }
}

export function toHistoryRows(entries) {
  return (entries || []).map(toHistoryRow)
}

// --- Filter autocomplete ---
//
// Candidates for the History tab's two filter boxes. Built from rows the page
// already holds -- the live ring buffer and the pages fetched into the tab --
// so typing costs no request, and every suggestion is a value the search will
// actually match. The flip side is coverage: a client that has been quiet for
// the whole session and is not in the loaded page will not be offered, so the
// boxes stay free-text and a suggestion is only ever a shortcut.

// Ranked lists are capped so a full ring buffer cannot hand the filter boxes
// more candidates than a reader would ever scroll.
const MAX_SUGGESTIONS = 200

// client_names arrives as the resolver's "; "-joined list. Each name is a
// filter on its own, so they are offered separately rather than as one string
// that would match nothing.
function splitNames(names) {
  return (names || '').split(';').map((n) => n.trim()).filter(Boolean)
}

function tally(map, value, hint) {
  const key = (value || '').trim()
  if (!key) return

  const existing = map.get(key)
  if (!existing) {
    map.set(key, { count: 1, hint: hint || '' })

    return
  }

  existing.count += 1
  // First non-empty hint wins; a later row carrying none must not blank it.
  if (!existing.hint && hint) existing.hint = hint
}

// Most-seen first, ties broken alphabetically so the order is stable between
// renders rather than dependent on insertion.
function ranked(map) {
  return [...map.entries()]
    .sort((a, b) => b[1].count - a[1].count || a[0].localeCompare(b[0]))
    .slice(0, MAX_SUGGESTIONS)
    .map(([value, meta]) => ({ value, hint: meta.hint }))
}

// Both the IP and each resolved name are offered as separate candidates: the
// server matches the client filter against either column, so whichever one the
// reader recognises is a valid filter by itself. Each carries the other as its
// hint, so picking one is an informed choice.
export function clientSuggestions(...rowLists) {
  const map = new Map()

  for (const rows of rowLists) {
    for (const row of rows || []) {
      tally(map, row.client_ip, row.client_names)

      for (const name of splitNames(row.client_names)) {
        tally(map, name, row.client_ip)
      }
    }
  }

  return ranked(map)
}

export function domainSuggestions(...rowLists) {
  const map = new Map()

  for (const rows of rowLists) {
    for (const row of rows || []) {
      tally(map, row.question_name)
    }
  }

  return ranked(map)
}
