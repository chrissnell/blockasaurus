<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { onMount, onDestroy } from 'svelte'
  import Highcharts from 'highcharts'

  let { labels = [], datasets = [], stacked = true } = $props()

  let container
  let chart
  let observer

  function readChartTokens() {
    const cs = getComputedStyle(document.documentElement)
    const series = Array.from({ length: 10 }, (_, i) =>
      cs.getPropertyValue(`--chart-${i + 1}`).trim()
    )
    return {
      series,
      text: cs.getPropertyValue('--color-text').trim(),
      textMuted: cs.getPropertyValue('--color-text-muted').trim(),
      border: cs.getPropertyValue('--color-border').trim(),
      font: cs.getPropertyValue('--font-mono').trim() || 'Inconsolata',
    }
  }

  function seriesColor(ds, i, tokens) {
    return ds.color || tokens.series[i % tokens.series.length]
  }

  onMount(() => {
    const tokens = readChartTokens()

    chart = Highcharts.chart(container, {
      colors: tokens.series,
      chart: {
        type: 'column',
        backgroundColor: 'transparent',
        style: { fontFamily: tokens.font },
        spacing: [10, 10, 10, 10],
      },
      title: { text: null },
      credits: { enabled: false },
      xAxis: {
        categories: labels,
        labels: { style: { color: tokens.textMuted, fontSize: '11px' } },
        lineColor: tokens.border,
        tickColor: tokens.border,
      },
      yAxis: {
        title: { text: null },
        labels: { style: { color: tokens.textMuted } },
        gridLineColor: tokens.border,
        min: 0,
      },
      legend: {
        itemStyle: { color: tokens.text, fontWeight: 'normal', fontSize: '12px' },
        itemHoverStyle: { color: tokens.text },
      },
      tooltip: {
        shared: true,
        style: { fontFamily: tokens.font, color: tokens.text },
      },
      plotOptions: {
        column: {
          stacking: stacked ? 'normal' : undefined,
          borderWidth: 0,
          pointPadding: 0.05,
          groupPadding: 0.05,
        },
      },
      series: datasets.map((ds, i) => ({
        name: ds.label,
        data: ds.data,
        color: seriesColor(ds, i, tokens),
      })),
    })

    // Re-theme when data-theme flips on <html>
    observer = new MutationObserver(() => {
      if (!chart) return
      const t = readChartTokens()
      chart.update(
        {
          colors: t.series,
          chart: {
            backgroundColor: 'transparent',
            style: { fontFamily: t.font },
          },
          xAxis: {
            labels: { style: { color: t.textMuted, fontSize: '11px' } },
            lineColor: t.border,
            tickColor: t.border,
          },
          yAxis: {
            labels: { style: { color: t.textMuted } },
            gridLineColor: t.border,
          },
          legend: {
            itemStyle: { color: t.text, fontWeight: 'normal', fontSize: '12px' },
            itemHoverStyle: { color: t.text },
          },
          tooltip: {
            style: { fontFamily: t.font, color: t.text },
          },
        },
        false
      )
      // Refresh series colors (only those without an explicit ds.color)
      chart.series.forEach((s, i) => {
        const ds = datasets[i]
        if (ds && !ds.color) {
          s.update({ color: t.series[i % t.series.length] }, false)
        }
      })
      chart.redraw(false)
    })
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['data-theme'],
    })
  })

  $effect(() => {
    if (!chart) return

    const tokens = readChartTokens()

    // Update categories
    chart.xAxis[0].setCategories(labels, false)

    // Sync series
    while (chart.series.length > datasets.length) {
      chart.series[chart.series.length - 1].remove(false)
    }

    datasets.forEach((ds, i) => {
      const color = seriesColor(ds, i, tokens)
      if (i < chart.series.length) {
        chart.series[i].setData(ds.data, false)
        chart.series[i].update({ name: ds.label, color }, false)
      } else {
        chart.addSeries({ name: ds.label, data: ds.data, color }, false)
      }
    })

    chart.redraw(false)
  })

  onDestroy(() => {
    if (observer) observer.disconnect()
    if (chart) chart.destroy()
  })
</script>

<div class="chart-container" bind:this={container}></div>

<style>
  .chart-container {
    height: 300px;
    width: 100%;
  }
</style>
