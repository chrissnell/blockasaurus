<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { onMount, onDestroy } from 'svelte'
  import Highcharts from 'highcharts'

  let { labels = [], datasets = [], stacked = true } = $props()

  let container
  let chart

  function getThemeColors() {
    const style = getComputedStyle(document.documentElement)
    return {
      text: style.getPropertyValue('--color-text-muted').trim(),
      grid: style.getPropertyValue('--color-border-subtle').trim(),
      bg: 'transparent',
    }
  }

  onMount(() => {
    const colors = getThemeColors()

    chart = Highcharts.chart(container, {
      chart: {
        type: 'column',
        backgroundColor: colors.bg,
        style: { fontFamily: 'Inconsolata' },
        spacing: [10, 10, 10, 10],
      },
      title: { text: null },
      credits: { enabled: false },
      xAxis: {
        categories: labels,
        labels: { style: { color: colors.text, fontSize: '11px' } },
        lineColor: colors.grid,
        tickColor: colors.grid,
      },
      yAxis: {
        title: { text: null },
        labels: { style: { color: colors.text } },
        gridLineColor: colors.grid,
        min: 0,
      },
      legend: {
        itemStyle: { color: colors.text, fontWeight: 'normal', fontSize: '12px' },
        itemHoverStyle: { color: colors.text },
      },
      tooltip: {
        shared: true,
        style: { fontFamily: 'Inconsolata' },
      },
      plotOptions: {
        column: {
          stacking: stacked ? 'normal' : undefined,
          borderWidth: 0,
          pointPadding: 0.05,
          groupPadding: 0.05,
        },
      },
      series: datasets.map(ds => ({
        name: ds.label,
        data: ds.data,
        color: ds.color,
      })),
    })
  })

  $effect(() => {
    if (!chart) return

    // Update categories
    chart.xAxis[0].setCategories(labels, false)

    // Sync series
    while (chart.series.length > datasets.length) {
      chart.series[chart.series.length - 1].remove(false)
    }

    datasets.forEach((ds, i) => {
      if (i < chart.series.length) {
        chart.series[i].setData(ds.data, false)
        chart.series[i].update({ name: ds.label, color: ds.color }, false)
      } else {
        chart.addSeries({ name: ds.label, data: ds.data, color: ds.color }, false)
      }
    })

    chart.redraw(false)
  })

  onDestroy(() => {
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
