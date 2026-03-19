<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { onMount, onDestroy } from 'svelte'
  import Highcharts from 'highcharts'

  let { labels = [], data = [], colors = [] } = $props()

  let container
  let chart

  onMount(() => {
    const style = getComputedStyle(document.documentElement)
    const textColor = style.getPropertyValue('--color-text-muted').trim()

    chart = Highcharts.chart(container, {
      chart: {
        type: 'pie',
        backgroundColor: 'transparent',
        style: { fontFamily: 'Inconsolata' },
        spacing: [10, 10, 10, 10],
      },
      title: { text: null },
      credits: { enabled: false },
      tooltip: {
        pointFormat: '<b>{point.y}</b> ({point.percentage:.1f}%)',
        style: { fontFamily: 'Inconsolata' },
      },
      legend: {
        enabled: true,
        layout: 'vertical',
        align: 'right',
        verticalAlign: 'middle',
        itemStyle: { color: textColor, fontWeight: 'normal', fontSize: '12px' },
        itemHoverStyle: { color: textColor },
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
        data: labels.map((name, i) => ({
          name,
          y: data[i] || 0,
          color: colors[i] || undefined,
        })),
      }],
    })
  })

  $effect(() => {
    if (!chart) return

    const points = labels.map((name, i) => ({
      name,
      y: data[i] || 0,
      color: colors[i] || undefined,
    }))

    chart.series[0].setData(points, true, false)
  })

  onDestroy(() => {
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
