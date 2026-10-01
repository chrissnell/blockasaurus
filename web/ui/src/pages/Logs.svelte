<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { LogViewer, Input, Toggle } from '@chrissnell/chonky-ui'
  import { connectLogStream } from '../lib/ws.js'
  import {
    MAX_ENTRIES,
    appendBounded,
    createEntryBatcher,
    filterEntries,
    normalizeQuery,
    toLogEntry,
  } from '../lib/logentries.js'
  import { onMount } from 'svelte'

  // Raw: the rows are write-once and display-only, so proxying every row and
  // every field the viewer reads would be pure overhead.
  let entries = $state.raw([])
  let search = $state('')
  // What the list is actually filtered by. Trails `search` so that holding a
  // key down does not rebuild the grid once per character -- see `viewKey`.
  let appliedSearch = $state('')
  let blockedOnly = $state(false)
  let connected = $state(false)
  let isMobile = $state(false)

  const SEARCH_DEBOUNCE_MS = 150

  $effect(() => {
    const next = search
    const timer = setTimeout(() => { appliedSearch = next }, SEARCH_DEBOUNCE_MS)

    return () => clearTimeout(timer)
  })

  onMount(() => {
    const mql = window.matchMedia('(max-width: 768px)')
    isMobile = mql.matches
    const onMqlChange = (e) => { isMobile = e.matches }
    mql.addEventListener('change', onMqlChange)
    return () => mql.removeEventListener('change', onMqlChange)
  })

  onMount(() => {
    const batcher = createEntryBatcher((batch) => {
      entries = appendBounded(entries, batch)
    })

    const disconnect = connectLogStream(
      (raw) => batcher.push(toLogEntry(raw)),
      (status) => { connected = status },
    )

    return () => {
      batcher.stop()
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
    { key: 'client_ip', label: 'Client', width: '120px', render: renderClient, mobile: true },
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

  const query = $derived(normalizeQuery(appliedSearch))
  const filtering = $derived(query !== '' || blockedOnly)
  const visible = $derived(filterEntries(entries, appliedSearch, blockedOnly))
  // The buffer is capped, so once it is full the total is a floor, not a count.
  const totalLabel = $derived(
    entries.length === MAX_ENTRIES ? `${MAX_ENTRIES}+` : `${entries.length}`
  )

  // LogViewer keeps its own scroll offset and an internal "at bottom" latch
  // that gates autoscroll, with no prop to reset either. A filter change
  // replaces the list both refer to, so a reader who had scrolled up stays
  // latched off -- and once the filtered list is short enough not to scroll, no
  // scroll event fires to re-arm it, so the view stops following live traffic
  // even after the filter is cleared. Its jump-to-bottom button is the manual
  // way out; remounting on filter change is the only automatic one, which is
  // what this key buys. Keyed off the debounced query so the remount costs one
  // grid rebuild per typing pause rather than one per keystroke.
  const viewKey = $derived(`${blockedOnly} ${query}`)
</script>

{#snippet renderTime(value)}
  {formatTime(value)}
{/snippet}

{#snippet renderDuration(value)}
  {value != null ? `${value}ms` : ''}
{/snippet}

<!-- Resolved names are searchable, so show them: a row matched by name with
     only its IP on screen looks like an arbitrary match. -->
{#snippet renderClient(value, entry)}
  <span title={entry.client_names || undefined}>{value || entry.client_names || ''}</span>
{/snippet}

<div class="page">
  <h1 class="page-title">Live Logs</h1>
  <div class="controls">
    <div class="search">
      <Input
        bind:value={search}
        placeholder="Filter by domain or client"
        aria-label="Filter by domain or client"
        onkeydown={(e) => { if (e.key === 'Escape') search = '' }}
      />
    </div>
    <Toggle label="Blocked only" bind:checked={blockedOnly} />
    {#if filtering}
      <span class="match-count">{visible.length} of {totalLabel}</span>
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
