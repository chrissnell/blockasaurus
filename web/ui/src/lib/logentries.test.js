// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

import { test } from 'node:test'
import assert from 'node:assert/strict'

import { MAX_ENTRIES, appendBounded, filterEntries, toLogEntry } from './logentries.js'

function query(fields) {
  return { timestamp: '2026-10-01T12:00:00Z', level: 'info', message: 'query resolved', fields }
}

function entry(fields) {
  return toLogEntry(query(fields))
}

test('toLogEntry marks blocked queries and colours them as errors', () => {
  const blocked = entry({ question_name: 'ads.example.com', response_type: 'BLOCKED' })
  assert.equal(blocked.blocked, true)
  assert.equal(blocked.level, 'error')

  const resolved = entry({ question_name: 'example.com', response_type: 'RESOLVED' })
  assert.equal(resolved.blocked, false)
  assert.equal(resolved.level, 'info')
})

test('toLogEntry keeps a non-query error log out of the blocked set', () => {
  const e = toLogEntry({ level: 'error', message: 'upstream unreachable' })
  assert.equal(e.level, 'error')
  assert.equal(e.blocked, false)
  // No fields at all: the message stands in for the question name.
  assert.equal(e.question_name, 'upstream unreachable')
})

test('appendBounded caps the buffer and drops the oldest rows', () => {
  let entries = []
  for (let i = 0; i < MAX_ENTRIES + 250; i++) {
    entries = appendBounded(entries, [entry({ question_name: `q${i}.example.com` })])
  }

  assert.equal(entries.length, MAX_ENTRIES)
  assert.equal(entries[0].question_name, 'q250.example.com')
  assert.equal(entries.at(-1).question_name, `q${MAX_ENTRIES + 249}.example.com`)
})

test('appendBounded caps a single oversized batch', () => {
  const batch = Array.from({ length: 5 }, (_, i) => entry({ question_name: `q${i}.example.com` }))
  const entries = appendBounded([], batch, 3)

  assert.deepEqual(entries.map(e => e.question_name), ['q2.example.com', 'q3.example.com', 'q4.example.com'])
})

test('appendBounded returns the same array when nothing arrived', () => {
  const entries = [entry({ question_name: 'example.com' })]
  assert.equal(appendBounded(entries, []), entries)
})

const sample = [
  entry({ question_name: 'ads.doubleclick.net', client_ip: '192.168.1.10', client_names: 'kitchen-tv', response_type: 'BLOCKED' }),
  entry({ question_name: 'news.example.com', client_ip: '192.168.1.10', client_names: 'kitchen-tv', response_type: 'RESOLVED' }),
  entry({ question_name: 'telemetry.example.org', client_ip: '192.168.1.42', client_names: 'laptop', response_type: 'BLOCKED' }),
  entry({ question_name: 'EXAMPLE.COM', client_ip: '10.0.0.5', response_type: 'CACHED' }),
]

test('filterEntries returns the input untouched when nothing is filtering', () => {
  assert.equal(filterEntries(sample, '   ', false), sample)
})

test('filterEntries matches a domain fragment case-insensitively', () => {
  assert.deepEqual(
    filterEntries(sample, 'example.com', false).map(e => e.question_name),
    ['news.example.com', 'EXAMPLE.COM'],
  )
  assert.deepEqual(
    filterEntries(sample, 'DOUBLEclick', false).map(e => e.question_name),
    ['ads.doubleclick.net'],
  )
})

test('filterEntries matches a client by IP and by resolved name', () => {
  assert.equal(filterEntries(sample, '192.168.1.10', false).length, 2)
  assert.deepEqual(
    filterEntries(sample, 'laptop', false).map(e => e.question_name),
    ['telemetry.example.org'],
  )
})

test('filterEntries ignores surrounding whitespace in the search box', () => {
  assert.equal(filterEntries(sample, '  laptop  ', false).length, 1)
})

test('blocked-only narrows to blocked queries and composes with the text filter', () => {
  assert.deepEqual(
    filterEntries(sample, '', true).map(e => e.question_name),
    ['ads.doubleclick.net', 'telemetry.example.org'],
  )
  assert.deepEqual(
    filterEntries(sample, '192.168.1.10', true).map(e => e.question_name),
    ['ads.doubleclick.net'],
  )
  assert.deepEqual(filterEntries(sample, 'news', true), [])
})

test('filterEntries tolerates rows with missing fields', () => {
  const sparse = [toLogEntry({ message: 'cache flushed' })]
  assert.deepEqual(filterEntries(sparse, '192.168', false), [])
  assert.equal(filterEntries(sparse, 'cache', false).length, 1)
})
