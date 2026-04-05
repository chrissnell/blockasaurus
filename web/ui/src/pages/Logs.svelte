<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { LogViewer } from '@chrissnell/chonky-ui'
  import { connectLogStream } from '../lib/ws.js'
  import { onMount } from 'svelte'

  let entries = $state([])
  let connected = $state(false)

  onMount(() => {
    const disconnect = connectLogStream(
      (entry) => { entries = [...entries, entry] },
      (status) => { connected = status },
    )
    return disconnect
  })

  function formatTime(ts) {
    if (!ts) return ''
    const d = new Date(ts)
    return d.toLocaleTimeString('en-GB', { hour12: false })
  }

  function mapLevel(lvl) {
    const l = (lvl || '').toLowerCase()
    if (l === 'error') return 'error'
    if (l === 'warn' || l === 'warning') return 'warn'
    if (l === 'debug') return 'debug'
    return 'info'
  }

  const columns = [
    { key: 'time', label: 'Time', width: '90px' },
    { key: 'duration', label: 'Duration', width: '90px' },
    { key: 'level', label: 'Level', width: '70px' },
    { key: 'type', label: 'Type', width: '70px' },
    { key: 'name', label: 'Name', width: '2fr' },
    { key: 'code', label: 'Code', width: '100px' },
    { key: 'reason', label: 'Reason', width: '2fr' },
  ]

  const rows = $derived(entries.map((entry) => {
    const f = entry.fields || {}
    const isBlocked = f.response_type === 'BLOCKED'
    return {
      level: isBlocked ? 'error' : mapLevel(entry.level),
      time: formatTime(entry.timestamp),
      duration: f.duration_ms != null ? `${f.duration_ms}ms` : '',
      type: f.question_type || '',
      name: f.question_name || entry.message || '',
      code: f.response_code || '',
      reason: f.response_reason || '',
    }
  }))
</script>

<div class="page">
  <h1 class="page-title">Live Logs</h1>
  <div class="log-wrap">
    <LogViewer
      entries={rows}
      {columns}
      showHeader
      live={connected}
      height="100%"
    />
  </div>
</div>

<style>
  .page {
    max-width: 1200px;
    display: flex;
    flex-direction: column;
    height: calc(65vh - var(--space-8) * 2);
  }
  .page-title {
    font-size: var(--text-2xl);
    font-weight: 700;
    margin-bottom: var(--space-6);
    flex-shrink: 0;
  }
  .log-wrap {
    flex: 1;
    min-height: 0;
  }
</style>
