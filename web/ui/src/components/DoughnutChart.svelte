<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { onMount, onDestroy } from 'svelte'
  import Highcharts from 'highcharts'

  let { labels = [], data = [], colors = [] } = $props()

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

  function buildPoints(tokens) {
    return labels.map((name, i) => ({
      name,
      y: data[i] || 0,
      color: colors[i] || tokens.series[i % tokens.series.length],
    }))
  }

  onMount(() => {
    const tokens = readChartTokens()

    chart = Highcharts.chart(container, {
      colors: tokens.series,
      chart: {
        type: 'pie',
        backgroundColor: 'transparent',
        style: { fontFamily: tokens.font },
        spacing: [10, 10, 10, 10],
      },
      title: { text: null },
      credits: { enabled: false },
      tooltip: {
        pointFormat: '<b>{point.y}</b> ({point.percentage:.1f}%)',
        style: { fontFamily: tokens.font, color: tokens.text },
      },
      legend: {
        enabled: true,
        layout: 'vertical',
        align: 'right',
        verticalAlign: 'middle',
        itemStyle: { color: tokens.textMuted, fontWeight: 'normal', fontSize: '12px' },
        itemHoverStyle: { color: tokens.text },
      },
      plotOptions: {
        pie: {
          innerSize: '55%',
          borderWidth: 0,
          showInLegend: true,
          dataLabels: { enabled: false },
        },
      },
      series: [{
        name: 'Count',
        data: buildPoints(tokens),
      }],
    })

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
          tooltip: {
            style: { fontFamily: t.font, color: t.text },
          },
          legend: {
            itemStyle: { color: t.textMuted, fontWeight: 'normal', fontSize: '12px' },
            itemHoverStyle: { color: t.text },
          },
        },
        false
      )
      // Only refresh colors that weren't explicitly provided by caller
      chart.series[0].setData(buildPoints(t), false, false)
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
    chart.series[0].setData(buildPoints(tokens), true, false)
  })

  onDestroy(() => {
    if (observer) observer.disconnect()
    if (chart) chart.destroy()
  })
</script>

<div class="chart-container" bind:this={container}></div>

<style>
  .chart-container {
    height: 280px;
    width: 100%;
  }
</style>
