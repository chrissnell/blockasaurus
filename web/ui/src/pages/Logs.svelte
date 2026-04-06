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
      (raw) => {
        const f = raw.fields || {}
        const isBlocked = f.response_type === 'BLOCKED'
        entries = [...entries, {
          level: isBlocked ? 'error' : mapLevel(raw.level),
          timestamp: raw.timestamp,
          duration_ms: f.duration_ms,
          question_type: f.question_type,
          question_name: f.question_name || raw.message,
          response_code: f.response_code,
          response_reason: f.response_reason,
        }]
      },
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
    { key: 'timestamp', label: 'Time', width: '90px', render: renderTime },
    { key: 'duration_ms', label: 'Duration', width: '90px', render: renderDuration },
    { key: 'level', label: 'Level', width: '70px' },
    { key: 'question_type', label: 'Type', width: '70px' },
    { key: 'question_name', label: 'Name', width: '2fr' },
    { key: 'response_code', label: 'Code', width: '100px' },
    { key: 'response_reason', label: 'Reason', width: '2fr' },
  ]
</script>

{#snippet renderTime(value)}
  {formatTime(value)}
{/snippet}

{#snippet renderDuration(value)}
  {value != null ? `${value}ms` : ''}
{/snippet}

<div class="page">
  <h1 class="page-title">Live Logs</h1>
  <div class="log-wrap">
    <LogViewer
      {entries}
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
    display: flex;
  }
  /* LogViewer's outer wrapper has no intrinsic height; without this its
     inner .log-body (height:100%) resolves against content size and the
     list overflows the page, painting over the status bar footer. */
  .log-wrap :global(.log-viewer) {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }
  .log-wrap :global(.log-viewer .log-body) {
    flex: 1;
    min-height: 0;
    /* !important needed: LogViewer sets height:100% as inline style,
       which otherwise wins over flex sizing here. */
    height: auto !important;
  }
</style>
