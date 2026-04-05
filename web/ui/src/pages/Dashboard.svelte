<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { onDestroy } from 'svelte'
  import { StatCard, Box } from '@chrissnell/chonky-ui'
  import BarChart from '../components/BarChart.svelte'
  import DoughnutChart from '../components/DoughnutChart.svelte'
  import {
    getStats,
    getStatsOvertime,
    getStatsOvertimeClients,
    getStatsQueryTypes,
    getStatsResponseTypes,
    getStatsTopDomains,
    getStatsTopClients,
  } from '../lib/api.js'

  let stats = $state(null)
  let overtime = $state(null)
  let overtimeClients = $state(null)
  let queryTypes = $state(null)
  let responseTypes = $state(null)
  let topDomains = $state(null)
  let topClients = $state(null)
  let loaded = $state(false)

  function getCssVar(name) {
    return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  }

  const chartPalette = () => Array.from({ length: 10 }, (_, i) => getCssVar(`--chart-${i + 1}`))

  function fmt(n) {
    if (n == null) return '\u2014'
    return Number(n).toLocaleString('en-US', { maximumFractionDigits: 0 })
  }

  function fmtTime(ts) {
    if (!ts) return ''
    const d = new Date(ts)
    const h = String(d.getHours()).padStart(2, '0')
    const m = String(d.getMinutes()).padStart(2, '0')
    return `${h}:${m}`
  }

  // Overtime bar chart
  let overtimeLabels = $derived(
    overtime?.buckets?.map(b => fmtTime(b.ts)) ?? []
  )

  let overtimeDatasets = $derived(() => {
    if (!overtime?.buckets) return []
    return [
      { label: 'Allowed', data: overtime.buckets.map(b => b.total - b.blocked), color: getCssVar('--color-success') },
      { label: 'Blocked', data: overtime.buckets.map(b => b.blocked), color: getCssVar('--color-blocked') },
    ]
  })

  // Client activity bar chart
  let clientLabels = $derived(
    overtimeClients?.buckets?.map(b => fmtTime(b.ts)) ?? []
  )

  let clientDatasets = $derived(() => {
    if (!overtimeClients?.buckets) return []
    const palette = chartPalette()
    const totals = {}
    for (const b of overtimeClients.buckets) {
      if (!b.clients) continue
      for (const [c, n] of Object.entries(b.clients)) {
        totals[c] = (totals[c] || 0) + n
      }
    }
    return Object.entries(totals)
      .sort((a, b) => b[1] - a[1])
      .slice(0, 10)
      .map(([client], i) => ({
        label: client,
        data: overtimeClients.buckets.map(b => b.clients?.[client] || 0),
        color: palette[i % palette.length],
      }))
  })

  // Doughnut charts
  let qtLabels = $derived(queryTypes ? Object.keys(queryTypes) : [])
  let qtData = $derived(queryTypes ? Object.values(queryTypes) : [])
  let qtColors = $derived(() => chartPalette().slice(0, qtLabels.length))

  let rtLabels = $derived(responseTypes ? Object.keys(responseTypes) : [])
  let rtData = $derived(responseTypes ? Object.values(responseTypes) : [])

  // Semantic colors for response types
  const responseTypeColorMap = {
    RESOLVED: '--color-success',
    CACHED: '--color-info',
    BLOCKED: '--color-danger',
    CUSTOMDNS: '--color-primary',
    CONDITIONAL: '--chart-6',
    HOSTSFILE: '--chart-7',
    FILTERED: '--chart-8',
    SPECIAL: '--chart-9',
  }

  let rtColors = $derived(() => {
    const palette = chartPalette()
    return rtLabels.map((label, i) => {
      const varName = responseTypeColorMap[label]
      return varName ? getCssVar(varName) : palette[i % palette.length]
    })
  })

  function maxCount(items) {
    if (!items?.length) return 1
    return Math.max(...items.map(i => i.count)) || 1
  }

  async function refresh() {
    const results = await Promise.allSettled([
      getStats(),
      getStatsOvertime(),
      getStatsOvertimeClients(),
      getStatsQueryTypes(),
      getStatsResponseTypes(),
      getStatsTopDomains(),
      getStatsTopClients(),
    ])
    if (results[0].status === 'fulfilled' && results[0].value) stats = results[0].value
    if (results[1].status === 'fulfilled' && results[1].value) overtime = results[1].value
    if (results[2].status === 'fulfilled' && results[2].value) overtimeClients = results[2].value
    if (results[3].status === 'fulfilled' && results[3].value) queryTypes = results[3].value
    if (results[4].status === 'fulfilled' && results[4].value) responseTypes = results[4].value
    if (results[5].status === 'fulfilled' && results[5].value) topDomains = results[5].value
    if (results[6].status === 'fulfilled' && results[6].value) topClients = results[6].value
    loaded = true
  }

  refresh()
  const timer = setInterval(refresh, 10000)
  onDestroy(() => clearInterval(timer))
</script>

<div class="page">
  <h1 class="page-title">Dashboard</h1>

  {#if !loaded}
    <div class="loading">loading dashboard data...</div>
  {:else}
    <!-- Stat Cards -->
    <div class="stat-grid">
      <StatCard label="Status" value="Running" variant="success" />
      <StatCard label="Total Queries" value={fmt(stats?.total_queries)} variant="info" />
      <StatCard label="Queries Blocked" value={fmt(stats?.blocked_queries)} variant="danger" />
      <StatCard label="Block Rate" value={stats ? stats.block_rate.toFixed(1) + '%' : '\u2014'} variant="primary" />
    </div>

    <!-- Queries Over Time -->
    <Box title="Queries over last 24 hours">
      <BarChart labels={overtimeLabels} datasets={overtimeDatasets()} stacked={true} />
    </Box>

    <!-- Client Activity -->
    <Box title="Client activity over last 24 hours">
      <BarChart labels={clientLabels} datasets={clientDatasets()} stacked={true} />
    </Box>

    <!-- Doughnut Charts -->
    <div class="two-col">
      <Box title="Query Types">
        <DoughnutChart labels={qtLabels} data={qtData} colors={qtColors()} />
      </Box>
      <Box title="Response Types">
        <DoughnutChart labels={rtLabels} data={rtData} colors={rtColors()} />
      </Box>
    </div>

    <!-- Top Domains -->
    <div class="two-col">
      <Box title="Top Permitted Domains">
        {#if topDomains?.permitted?.length}
          <table class="top-table">
            <thead><tr><th>Domain</th><th class="col-hits">Hits</th><th class="col-freq">Frequency</th></tr></thead>
            <tbody>
              {#each topDomains.permitted as item}
                <tr>
                  <td class="cell-domain" title={item.domain}>{item.domain}</td>
                  <td class="cell-hits">{item.count.toLocaleString()}</td>
                  <td class="cell-freq">
                    <div class="freq-track">
                      <div class="freq-bar freq-bar--success" style="width: {item.count / maxCount(topDomains.permitted) * 100}%"></div>
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        {:else}
          <p class="empty">No data yet</p>
        {/if}
      </Box>
      <Box title="Top Blocked Domains">
        {#if topDomains?.blocked?.length}
          <table class="top-table">
            <thead><tr><th>Domain</th><th class="col-hits">Hits</th><th class="col-freq">Frequency</th></tr></thead>
            <tbody>
              {#each topDomains.blocked as item}
                <tr>
                  <td class="cell-domain" title={item.domain}>{item.domain}</td>
                  <td class="cell-hits">{item.count.toLocaleString()}</td>
                  <td class="cell-freq">
                    <div class="freq-track">
                      <div class="freq-bar freq-bar--blocked" style="width: {item.count / maxCount(topDomains.blocked) * 100}%"></div>
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        {:else}
          <p class="empty">No data yet</p>
        {/if}
      </Box>
    </div>

    <!-- Top Clients -->
    <div class="two-col">
      <Box title="Top Clients (total)">
        {#if topClients?.total?.length}
          <table class="top-table">
            <thead><tr><th>Client</th><th class="col-hits">Requests</th><th class="col-freq">Frequency</th></tr></thead>
            <tbody>
              {#each topClients.total as item}
                <tr>
                  <td class="cell-domain" title={item.client}>{item.client}</td>
                  <td class="cell-hits">{item.count.toLocaleString()}</td>
                  <td class="cell-freq">
                    <div class="freq-track">
                      <div class="freq-bar freq-bar--info" style="width: {item.count / maxCount(topClients.total) * 100}%"></div>
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        {:else}
          <p class="empty">No data yet</p>
        {/if}
      </Box>
      <Box title="Top Clients (blocked)">
        {#if topClients?.blocked?.length}
          <table class="top-table">
            <thead><tr><th>Client</th><th class="col-hits">Blocked</th><th class="col-freq">Frequency</th></tr></thead>
            <tbody>
              {#each topClients.blocked as item}
                <tr>
                  <td class="cell-domain" title={item.client}>{item.client}</td>
                  <td class="cell-hits">{item.count.toLocaleString()}</td>
                  <td class="cell-freq">
                    <div class="freq-track">
                      <div class="freq-bar freq-bar--danger" style="width: {item.count / maxCount(topClients.blocked) * 100}%"></div>
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        {:else}
          <p class="empty">No data yet</p>
        {/if}
      </Box>
    </div>
  {/if}
</div>

<style>
  .page {
    max-width: 1200px;
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
  }

  .page-title {
    font-size: var(--text-2xl);
    font-weight: 700;
    margin-bottom: var(--space-2);
  }

  .loading {
    color: var(--color-text-dim);
    font-size: var(--text-sm);
    letter-spacing: 0.05em;
    text-align: center;
    padding: var(--space-12) 0;
  }

  .stat-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--space-4);
  }

  .two-col {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: var(--space-4);
  }

  /* Card spacing override — Card adds its own margin-bottom */
  .two-col :global(.box) {
    margin-bottom: 0;
  }

  /* Tables */
  .top-table {
    width: 100%;
    border-collapse: collapse;
    font-size: var(--text-xs);
  }

  .top-table th {
    text-align: left;
    font-size: var(--text-xs);
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--color-text-dim);
    padding: var(--space-2) var(--space-3);
    border-bottom: 1px solid var(--color-border-subtle);
    white-space: nowrap;
  }

  .top-table td {
    padding: var(--space-2) var(--space-3);
    border-bottom: 1px solid var(--color-border-subtle);
  }

  .top-table tr:last-child td { border-bottom: none; }
  .top-table tbody tr:hover td { background: var(--color-surface-raised); }

  .cell-domain {
    max-width: 0;
    width: 50%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .col-hits { width: 15%; text-align: right; }
  .cell-hits {
    text-align: right;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
    color: var(--color-text-muted);
  }

  .col-freq { width: 35%; }
  .cell-freq { padding-right: var(--space-4); }

  .freq-track {
    width: 100%;
    height: 6px;
    background: var(--color-surface-raised);
    border-radius: var(--radius);
    overflow: hidden;
  }

  .freq-bar {
    height: 100%;
    border-radius: var(--radius);
    transition: width 0.4s ease;
    min-width: 2px;
  }

  .freq-bar--success { background: var(--color-success); }
  .freq-bar--danger { background: var(--color-danger); }
  .freq-bar--info { background: var(--color-info); }
  .freq-bar--blocked { background: var(--color-blocked); }

  .empty {
    color: var(--color-text-dim);
    font-size: var(--text-xs);
    font-style: italic;
  }

  @media (max-width: 1024px) {
    .stat-grid { grid-template-columns: repeat(2, 1fr); }
  }

  @media (max-width: 768px) {
    .stat-grid { grid-template-columns: 1fr; }
    .two-col { grid-template-columns: 1fr; }
  }
</style>
