<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { LogViewer, Input, Toggle } from '@chrissnell/chonky-ui'
  import { connectLogStream } from '../lib/ws.js'
  import { MAX_ENTRIES, appendBounded, filterEntries, toLogEntry } from '../lib/logentries.js'
  import { onMount } from 'svelte'

  // Raw: the entries are display-only, so there is nothing to gain from
  // proxying every row and every field the viewer reads.
  let entries = $state.raw([])
  let search = $state('')
  let blockedOnly = $state(false)
  let connected = $state(false)
  let isMobile = $state(false)

  onMount(() => {
    const mql = window.matchMedia('(max-width: 768px)')
    isMobile = mql.matches
    const onMqlChange = (e) => { isMobile = e.matches }
    mql.addEventListener('change', onMqlChange)
    return () => mql.removeEventListener('change', onMqlChange)
  })

  onMount(() => {
    // Queries arrive one WS message at a time, so rebuilding the list per
    // message is a re-render per DNS query. Batching into animation frames
    // bounds the render cost by the display rate rather than by traffic.
    let pending = []
    let frame = 0

    function flush() {
      frame = 0
      entries = appendBounded(entries, pending)
      pending = []
    }

    const disconnect = connectLogStream(
      (raw) => {
        pending.push(toLogEntry(raw))
        // A backgrounded tab gets no animation frames, so the queue needs the
        // same cap as the buffer it drains into.
        if (pending.length > MAX_ENTRIES) {
          pending.splice(0, pending.length - MAX_ENTRIES)
        }
        if (frame === 0) frame = requestAnimationFrame(flush)
      },
      (status) => { connected = status },
    )

    return () => {
      if (frame !== 0) cancelAnimationFrame(frame)
      disconnect()
    }
  })

  function formatTime(ts) {
    if (!ts) return ''
    const d = new Date(ts)
    return d.toLocaleTimeString('en-GB', { hour12: false })
  }

  const allColumns = [
    { key: 'timestamp', label: 'Time', width: '90px', render: renderTime, mobile: true },
    { key: 'client_ip', label: 'Client', width: '120px', mobile: true },
    { key: 'client_group', label: 'Group', width: '120px', mobile: false },
    { key: 'duration_ms', label: 'Duration', width: '90px', render: renderDuration, mobile: false },
    { key: 'level', label: 'Level', width: '70px', mobile: true },
    { key: 'question_type', label: 'Type', width: '70px', mobile: false },
    { key: 'question_name', label: 'Name', width: '2fr', mobile: true },
    { key: 'response_code', label: 'Code', width: '100px', mobile: false },
    { key: 'response_reason', label: 'Reason', width: '2fr', mobile: true },
  ]

  const columns = $derived(
    isMobile ? allColumns.filter(c => c.mobile) : allColumns
  )

  const query = $derived(search.trim().toLowerCase())
  const filtering = $derived(query !== '' || blockedOnly)
  const visible = $derived(filterEntries(entries, search, blockedOnly))

  // LogViewer owns its scroll offset and an internal "is at bottom" latch that
  // gates autoscroll, and exposes no way to reset either. Changing the filter
  // replaces the list both refer to: a reader who had scrolled up stays latched
  // off, and the view then silently stops following live traffic even after the
  // filter is cleared. Remounting on filter change re-pins to the newest match.
  const viewKey = $derived(`${blockedOnly} ${query}`)
</script>

{#snippet renderTime(value)}
  {formatTime(value)}
{/snippet}

{#snippet renderDuration(value)}
  {value != null ? `${value}ms` : ''}
{/snippet}

<div class="page">
  <h1 class="page-title">Live Logs</h1>
  <div class="controls">
    <div class="search">
      <Input
        bind:value={search}
        placeholder="Filter by domain or client"
        aria-label="Filter by domain or client"
      />
    </div>
    <Toggle label="Blocked only" bind:checked={blockedOnly} />
    {#if filtering}
      <span class="match-count">{visible.length} of {entries.length}</span>
    {/if}
  </div>
  <div class="log-wrap">
    {#key viewKey}
      <LogViewer
        entries={visible}
        {columns}
        showHeader
        live={connected}
        height="100%"
      />
    {/key}
  </div>
</div>

<style>
  .page {
    max-width: 1500px;
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
  .controls {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    margin-bottom: var(--space-4);
    flex-shrink: 0;
    flex-wrap: wrap;
  }
  .search {
    flex: 1 1 260px;
    max-width: 420px;
  }
  /* Inputs carry a global margin-bottom for stacked form fields, which throws
     the toolbar row out of vertical alignment. */
  .search :global(input) {
    margin-bottom: 0;
  }
  .match-count {
    font-size: var(--text-sm);
    color: var(--color-text-dim);
    white-space: nowrap;
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
