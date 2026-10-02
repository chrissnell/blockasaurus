<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { Button, EmptyState, Input, LogViewer, Select, Spinner, Toggle } from '@chrissnell/chonky-ui'
  import { connectLogStream } from '../lib/ws.js'
  import {
    MAX_ENTRIES,
    appendBounded,
    createEntryBatcher,
    filterEntries,
    normalizeQuery,
    toLogEntry,
  } from '../lib/logentries.js'
  import {
    DEFAULT_RANGE,
    RANGE_OPTIONS,
    clientSuggestions,
    domainSuggestions,
    historyParams,
    rangeLabel,
    rangeStart,
    toHistoryRows,
  } from '../lib/loghistory.js'
  import SuggestInput from '../components/SuggestInput.svelte'
  import { queryLogHistory } from '../lib/api.js'
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

  const TABS = [
    { id: 'live', label: 'Live' },
    { id: 'history', label: 'History' },
  ]

  let currentTab = $state('live')

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

  // The stream and the ring buffer live at page level and are never gated on
  // the active tab: the History panel's markup unmounts when it is not shown,
  // and if the live view owned the socket the same would happen to it --
  // switching tabs would drop the connection and discard every entry collected
  // so far, so coming back would show an empty log slowly refilling.
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

  function formatDateTime(ts) {
    if (!ts) return ''
    const d = new Date(ts)
    return `${d.toLocaleDateString('en-GB', { day: '2-digit', month: 'short' })} `
      + d.toLocaleTimeString('en-GB', { hour12: false })
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

  // Both tabs read through the same column set, including the mobile subset.
  // The one exception is the Time cell: the live buffer never spans more than
  // the last few thousand queries, but a history window can be 30 days wide,
  // where a bare clock time says nothing about which day a row is from.
  const historyColumns = $derived(
    columns.map(c => (
      c.key === 'timestamp' ? { ...c, width: '150px', render: renderDateTime } : c
    ))
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

  // --- History tab ---
  //
  // Deliberately shares no filter state with the Live tab. The live field is a
  // substring filter over whatever is in the ring buffer; these run a query
  // against the database. Carrying a string from one to the other would silently
  // change what it means.

  let historyClient = $state('')
  let historyDomain = $state('')
  let historyRange = $state(DEFAULT_RANGE)
  let historyBlockedOnly = $state(false)

  // Debounced copies, so typing fires one request per pause rather than one per
  // keystroke -- the same 150ms the live filter uses.
  let appliedHistoryClient = $state('')
  let appliedHistoryDomain = $state('')

  let historyRows = $state.raw([])
  let historyCursor = $state(null)
  let historyWindowStart = $state(null)
  let historyLoading = $state(false)
  let historyLoadingMore = $state(false)
  let historyError = $state('')
  // Set when the server reports that the query log is not a searchable target.
  // Kept apart from historyError and from an empty result: "no rows matched" and
  // "there is no database to match against" are different answers.
  let historyUnavailable = $state('')
  let historyViewKey = $state(0)

  // Guards against an out-of-order response overwriting a newer one: a slow
  // request for a wide window can land after the narrow one that superseded it.
  let historyRequestSeq = 0
  // The query parameters the current result set was fetched with, including its
  // resolved `from`. "Load more" reuses these rather than rebuilding them, so a
  // filter edited mid-pagination cannot apply a cursor to a different query --
  // and so the window cannot shift under the cursor and skip rows.
  let historyPageParams = null

  // The filters the controls describe. Derived so the effect below re-runs on a
  // filter change and on nothing else. The time window is deliberately NOT in
  // here: `rangeStart` reads the clock, which is not a tracked dependency, so a
  // memoised derived would pin "Last hour" to whenever it last recomputed and a
  // tab left open would keep searching a window that had slid into the past.
  const historyFilters = $derived({
    client: appliedHistoryClient,
    domain: appliedHistoryDomain,
    blockedOnly: historyBlockedOnly,
    range: historyRange,
  })

  $effect(() => {
    const client = historyClient
    const domain = historyDomain
    const timer = setTimeout(() => {
      appliedHistoryClient = client
      appliedHistoryDomain = domain
    }, SEARCH_DEBOUNCE_MS)

    return () => clearTimeout(timer)
  })

  // Runs on entering the tab and on every filter change while it is open. The
  // early return keeps the Live tab from issuing queries, and makes a tab switch
  // itself a refresh.
  $effect(() => {
    if (currentTab !== 'history') return

    runHistorySearch(historyFilters)
  })

  function applyHistoryFailure(err) {
    historyRows = []
    historyCursor = null
    // Drop the window too: leaving the superseded one on screen next to an error
    // claims a range the failed query never covered.
    historyWindowStart = null

    if (err.status === 503) {
      historyUnavailable = err.message
    } else if (err.message !== 'unauthorized') {
      historyError = err.message
    }
  }

  async function runHistorySearch(filters) {
    const seq = ++historyRequestSeq
    // Resolved now, not when `historyFilters` last recomputed, so the window is
    // always relative to this search.
    const from = rangeStart(filters.range)
    const params = historyParams({ ...filters, from })

    historyLoading = true
    // A superseded "Load more" never reaches its own reset (the seq guard
    // discards it), so clear the flag here or the button stays disabled forever.
    historyLoadingMore = false
    historyError = ''
    historyUnavailable = ''

    try {
      const page = await queryLogHistory.search(params)
      if (seq !== historyRequestSeq) return

      historyRows = toHistoryRows(page.entries)
      historyCursor = page.next_cursor || null
      historyPageParams = params
      historyWindowStart = from
      // Remount the viewer so a fresh result set starts at the top rather than
      // wherever the previous one was scrolled to. Not keyed on appended pages:
      // "Load more" should not throw away the reader's position.
      historyViewKey += 1
    } catch (err) {
      if (seq !== historyRequestSeq) return

      applyHistoryFailure(err)
    } finally {
      if (seq === historyRequestSeq) historyLoading = false
    }
  }

  async function loadMoreHistory() {
    if (!historyCursor || !historyPageParams || historyLoading || historyLoadingMore) return

    const seq = historyRequestSeq
    const cursor = historyCursor
    const params = { ...historyPageParams, cursor }

    historyLoadingMore = true

    try {
      const page = await queryLogHistory.search(params)
      if (seq !== historyRequestSeq) return

      historyRows = historyRows.concat(toHistoryRows(page.entries))
      historyCursor = page.next_cursor || null
    } catch (err) {
      if (seq !== historyRequestSeq) return

      historyError = err.message === 'unauthorized' ? '' : err.message
    } finally {
      // Unconditional: a stale load-more still has to release the button, which
      // a superseding search re-enables for the new result set.
      historyLoadingMore = false
    }
  }

  const historyWindowLabel = $derived(
    historyWindowStart
      ? `${rangeLabel(historyRange)} — since ${formatDateTime(historyWindowStart)}`
      : rangeLabel(historyRange)
  )

  // Autocomplete candidates for the two filter boxes. Both sources are already
  // in memory: the live ring buffer, which fills with whatever is querying the
  // server right now, and the history page on screen.
  const clientOptions = $derived(clientSuggestions(historyRows, entries))
  const domainOptions = $derived(domainSuggestions(historyRows, entries))
</script>

{#snippet renderTime(value)}
  {formatTime(value)}
{/snippet}

{#snippet renderDateTime(value)}
  {formatDateTime(value)}
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
  <h1 class="page-title">Logs</h1>

  <!-- A plain button bar, not a nested Tabs.Root: Shell.svelte already wraps the
       whole app in one for the top nav, and a second root inside it would have
       its triggers register against the outer context. ClientGroups.svelte uses
       the same pattern for the same reason. -->
  <div class="tab-bar">
    {#each TABS as tab}
      <button class="tab" class:active={tab.id === currentTab}
        onclick={() => currentTab = tab.id}
      >{tab.label}</button>
    {/each}
  </div>

  {#if currentTab === 'live'}
    <div class="controls">
      <div class="search">
        <Input
          bind:value={search}
          placeholder="Filter visible entries"
          aria-label="Filter visible entries by domain or client"
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
  {:else}
    <!-- Every cell holds its width whatever the state of the search, including
         the spinner's: a control row that reshuffles itself on each keystroke
         is unreadable to type into. -->
    <div class="controls history">
      <div class="search">
        <SuggestInput
          bind:value={historyDomain}
          options={domainOptions}
          placeholder="Domain"
          label="Search history by domain"
          onEscape={() => { historyDomain = '' }}
        />
      </div>
      <div class="search">
        <SuggestInput
          bind:value={historyClient}
          options={clientOptions}
          placeholder="Client IP or name"
          label="Search history by client IP or name"
          onEscape={() => { historyClient = '' }}
        />
      </div>
      <div class="range">
        <Select options={RANGE_OPTIONS} bind:value={historyRange} aria-label="Time range" />
      </div>
      <Toggle label="Blocked only" bind:checked={historyBlockedOnly} />
      <div class="busy" role="status" aria-label={historyLoading ? 'Searching' : ''}>
        {#if historyLoading}
          <Spinner size={16} />
        {/if}
      </div>
    </div>

    <p class="window">
      {historyWindowLabel} · newest first
      <!-- Counts the rows on screen, so it stays put while the next search is in
           flight instead of blinking out and back. Held back until the first
           result lands, when "0 queries" would be a claim about nothing. -->
      {#if historyWindowStart && !historyUnavailable && !historyError}
        <span class="match-count">
          · {historyRows.length}{historyCursor ? '+' : ''} {historyRows.length === 1 ? 'query' : 'queries'}
        </span>
      {/if}
    </p>

    <!-- One sized region for every outcome. Without it an empty result collapses
         the page around the table and the controls jump up to meet it. -->
    <div class="results">
      {#if historyUnavailable}
        <EmptyState>
          Query log history is unavailable: {historyUnavailable}.
          Set <code>queryLog.type</code> to <code>sqlite</code> to search past queries.
        </EmptyState>
      {:else if historyError}
        <EmptyState>Could not search the query log: {historyError}</EmptyState>
      {:else if historyRows.length === 0}
        <EmptyState>
          {historyLoading ? 'Searching…' : `No queries recorded in this window (${rangeLabel(historyRange)}).`}
        </EmptyState>
      {:else}
        <div class="log-wrap">
          {#key historyViewKey}
            <LogViewer
              entries={historyRows}
              columns={historyColumns}
              showHeader
              autoscroll={false}
              height="100%"
            />
          {/key}
        </div>
        {#if historyCursor}
          <div class="more">
            <Button variant="secondary" onclick={loadMoreHistory} disabled={historyLoadingMore}>
              {historyLoadingMore ? 'Loading…' : 'Load more'}
            </Button>
          </div>
        {/if}
      {/if}
    </div>
  {/if}
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
  .tab-bar {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem;
    border-bottom: 1px solid var(--color-border);
    margin-bottom: var(--space-4);
    padding-bottom: 0.5rem;
    flex-shrink: 0;
  }
  .tab {
    background: none;
    border: 1px solid var(--color-btn-border);
    border-radius: var(--radius);
    color: var(--color-text-muted);
    /* A bare <button> does not inherit the page font: without this the tabs
       render in the UA's sans default next to an all-monospace page. */
    font-family: inherit;
    font-size: var(--text-sm);
    padding: 0.3rem 0.75rem;
    cursor: pointer;
    transition: all 0.15s ease;
  }
  .tab:hover {
    color: var(--color-text);
    border-color: var(--color-text-dim);
  }
  .tab.active {
    background: var(--color-primary);
    color: var(--color-primary-fg, #fff);
    border-color: var(--color-primary);
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
  /* Two inputs plus a select share the history row, so they need a narrower
     basis than the single field the live row carries. */
  .controls.history .search {
    flex: 1 1 180px;
  }
  /* Inputs carry a global margin-bottom for stacked form fields, which throws
     the toolbar row out of vertical alignment. */
  .search :global(input) {
    margin-bottom: 0;
  }
  /* Reserved whether or not a search is running. Letting the spinner come and
     go as a flex item re-divides the row across the two filter boxes on every
     request, which is the cursor jumping sideways as you type. */
  .busy {
    flex: 0 0 16px;
    height: 16px;
    display: flex;
    align-items: center;
  }
  .range {
    flex: 0 0 auto;
    min-width: 160px;
  }
  .range :global(button) {
    margin-bottom: 0;
  }
  .match-count {
    font-size: var(--text-sm);
    color: var(--color-text-dim);
    white-space: nowrap;
  }
  /* The active window is part of the result: a row list means nothing without
     the range that produced it. */
  .window {
    font-size: var(--text-sm);
    color: var(--color-text-dim);
    margin: 0 0 var(--space-3);
    flex-shrink: 0;
  }
  .more {
    margin-top: var(--space-3);
    flex-shrink: 0;
  }
  .results {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
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
