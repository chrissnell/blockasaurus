// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

import assert from 'node:assert/strict'
import { test } from 'node:test'

import {
  DEFAULT_RANGE,
  HISTORY_PAGE_SIZE,
  RANGE_OPTIONS,
  historyParams,
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
