// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

import assert from 'node:assert/strict'
import { test } from 'node:test'

import {
  DEFAULT_RANGE,
  HISTORY_PAGE_SIZE,
  RANGE_OPTIONS,
  clientSuggestions,
  domainSuggestions,
  historyParams,
  matchSuggestions,
  rangeLabel,
  rangeStart,
  toHistoryRow,
  toHistoryRows,
} from './loghistory.js'

const NOW = new Date('2026-10-02T12:00:00.000Z')

test('rangeStart subtracts the named window', () => {
  assert.equal(rangeStart('1h', NOW).toISOString(), '2026-10-02T11:00:00.000Z')
  assert.equal(rangeStart('24h', NOW).toISOString(), '2026-10-01T12:00:00.000Z')
  assert.equal(rangeStart('7d', NOW).toISOString(), '2026-09-25T12:00:00.000Z')
  assert.equal(rangeStart('30d', NOW).toISOString(), '2026-09-02T12:00:00.000Z')
})

test('an unknown range falls back to the default instead of going unbounded', () => {
  assert.equal(
    rangeStart('all-time', NOW).toISOString(),
    rangeStart(DEFAULT_RANGE, NOW).toISOString(),
  )
  assert.equal(rangeLabel('all-time'), rangeLabel(DEFAULT_RANGE))
})

test('the default range is one of the offered options', () => {
  assert.ok(RANGE_OPTIONS.some((o) => o.value === DEFAULT_RANGE))
})

test('historyParams omits blank filters', () => {
  assert.deepEqual(historyParams({ client: '  ', domain: '', from: null }), {
    limit: String(HISTORY_PAGE_SIZE),
  })
})

test('historyParams trims filters and serializes the window', () => {
  assert.deepEqual(
    historyParams({
      client: '  192.168.1.5 ',
      domain: ' example.com',
      blockedOnly: true,
      from: new Date('2026-10-01T12:00:00.000Z'),
      limit: 25,
      cursor: 'opaque',
    }),
    {
      client: '192.168.1.5',
      domain: 'example.com',
      blocked_only: 'true',
      from: '2026-10-01T12:00:00.000Z',
      limit: '25',
      cursor: 'opaque',
    },
  )
})

test('historyParams omits blocked_only when off', () => {
  const params = historyParams({ blockedOnly: false })
  assert.equal('blocked_only' in params, false)
})

test('toHistoryRow marks blocked entries at error level', () => {
  const entry = {
    timestamp: '2026-10-01T12:00:00Z',
    client_ip: '192.168.1.5',
    client_names: 'laptop',
    client_group: 'default',
    duration_ms: 12,
    question_type: 'A',
    question_name: 'ads.example.net',
    response_code: 'NOERROR',
    response_reason: 'BLOCKED (adlist)',
    response_type: 'BLOCKED',
    blocked: true,
  }

  assert.deepEqual(toHistoryRow(entry), {
    level: 'error',
    blocked: true,
    timestamp: '2026-10-01T12:00:00Z',
    client_ip: '192.168.1.5',
    client_names: 'laptop',
    client_group: 'default',
    duration_ms: 12,
    question_type: 'A',
    question_name: 'ads.example.net',
    response_code: 'NOERROR',
    response_reason: 'BLOCKED (adlist)',
  })
})

test('toHistoryRow leaves unblocked entries at info level', () => {
  assert.equal(toHistoryRow({ blocked: false }).level, 'info')
})

test('toHistoryRows handles a missing list', () => {
  assert.deepEqual(toHistoryRows(undefined), [])
  assert.deepEqual(toHistoryRows([{ blocked: false }]), [toHistoryRow({ blocked: false })])
})

test('clientSuggestions offers the IP and each resolved name, cross-hinted', () => {
  const rows = [
    { client_ip: '192.168.1.5', client_names: 'laptop.lan; laptop' },
    { client_ip: '192.168.1.6', client_names: '' },
  ]

  assert.deepEqual(clientSuggestions(rows), [
    { value: '192.168.1.5', hint: 'laptop.lan; laptop' },
    { value: '192.168.1.6', hint: '' },
    { value: 'laptop', hint: '192.168.1.5' },
    { value: 'laptop.lan', hint: '192.168.1.5' },
  ])
})

test('clientSuggestions ranks the busiest clients first and dedupes across sources', () => {
  const history = [{ client_ip: '10.0.0.2' }, { client_ip: '10.0.0.1' }, { client_ip: '10.0.0.1' }]
  const live = [{ client_ip: '10.0.0.1' }]

  assert.deepEqual(clientSuggestions(history, live).map((o) => o.value), ['10.0.0.1', '10.0.0.2'])
})

test('clientSuggestions keeps a hint a later row no longer carries', () => {
  const rows = [
    { client_ip: '10.0.0.1', client_names: 'nas' },
    { client_ip: '10.0.0.1', client_names: '' },
  ]

  assert.equal(clientSuggestions(rows)[0].hint, 'nas')
})

test('clientSuggestions skips rows with nothing to offer', () => {
  assert.deepEqual(clientSuggestions([{ client_ip: '  ', client_names: ' ; ' }], undefined), [])
})

test('domainSuggestions ranks by hit count', () => {
  const rows = [
    { question_name: 'b.example.com' },
    { question_name: 'a.example.com' },
    { question_name: 'a.example.com' },
  ]

  assert.deepEqual(domainSuggestions(rows).map((o) => o.value), ['a.example.com', 'b.example.com'])
})

// The live stream carries the wire form off the question section; the query log
// stores it through util.ExtractDomainOnly. Offering the wire form would be a
// suggestion the server-side LIKE cannot match.
test('domainSuggestions normalizes the live wire form onto the stored shape', () => {
  const live = [{ question_name: 'Example.COM.' }]
  const history = [{ question_name: 'example.com' }]

  assert.deepEqual(domainSuggestions(history, live), [{ value: 'example.com', hint: '' }])
})

test('suggestions are not capped before the query is known', () => {
  const rows = []
  // One hit each for 300 domains, plus a hot one that outranks every one of them.
  for (let i = 0; i < 300; i++) rows.push({ question_name: `d${i}.example.com` })
  for (let i = 0; i < 50; i++) rows.push({ question_name: 'hot.example.com' })

  const options = domainSuggestions(rows)
  assert.equal(options.length, 301)
  assert.deepEqual(matchSuggestions(options, 'd299.', 8).map((o) => o.value), ['d299.example.com'])
})

test('matchSuggestions puts prefix matches ahead of substring ones', () => {
  const options = [
    { value: 'lab.example.com', hint: '' },
    { value: 'example.com', hint: '' },
  ]

  assert.deepEqual(
    matchSuggestions(options, 'exam', 8).map((o) => o.value),
    ['example.com', 'lab.example.com'],
  )
})

test('matchSuggestions matches the hint, so a client is findable by either half', () => {
  const options = [{ value: '192.168.1.5', hint: 'laptop.lan' }]

  assert.deepEqual(matchSuggestions(options, 'laptop', 8).map((o) => o.value), ['192.168.1.5'])
  assert.deepEqual(matchSuggestions(options, 'nope', 8), [])
})

test('matchSuggestions is case-insensitive and ignores surrounding space', () => {
  const options = [{ value: 'NAS.example.com', hint: '' }]

  assert.deepEqual(matchSuggestions(options, '  nas  ', 8).map((o) => o.value), ['NAS.example.com'])
})

test('matchSuggestions caps at max and offers everything for an empty query', () => {
  const options = Array.from({ length: 20 }, (_, i) => ({ value: `d${i}`, hint: '' }))

  assert.equal(matchSuggestions(options, '', 8).length, 8)
  assert.equal(matchSuggestions(options, 'd1', 8).length, 8)
})
