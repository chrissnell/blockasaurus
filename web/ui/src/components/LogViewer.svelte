<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  let { entries = [], connected = false, autoscroll = true } = $props()
  let container

  $effect(() => {
    if (autoscroll && entries.length && container) {
      container.scrollTop = container.scrollHeight
    }
  })

  function entryClass(entry) {
    if (entry.fields?.response_type === 'BLOCKED') return 'blocked'
    const map = { error: 'err', warn: 'warn', warning: 'warn', info: 'ok', debug: 'dim' }
    return map[entry.level?.toLowerCase()] || 'dim'
  }

  function formatTime(ts) {
    if (!ts) return ''.padEnd(8)
    const d = new Date(ts)
    return d.toLocaleTimeString('en-GB', { hour12: false })
  }

  function formatDuration(entry) {
    const f = entry.fields
    if (f && f.duration_ms != null) return `${f.duration_ms}ms`
    return ''
  }

  function formatMessage(entry) {
    const f = entry.fields
    if (f && f.question_name) {
      const qtype = (f.question_type || '').padEnd(5)
      const name = (f.question_name || '').padEnd(30)
      const rcode = (f.response_code || '').padEnd(10)
      const reason = f.response_reason || ''
      return `${qtype}  ${name}  ${rcode}  ${reason}`
    }
    return entry.message
  }
</script>

<div class="log-viewer">
  <div class="log-header">
    <span class="dot" class:connected></span>
    <span>{connected ? 'live' : 'disconnected'}</span>
    <span class="count">{entries.length} entries</span>
  </div>
  <div class="log-col-hdr">{"Time".padEnd(8)}  {"Duration".padEnd(10)}  {"Level".padEnd(5)}  {"Type".padEnd(5)}  {"Name".padEnd(30)}  {"Code".padEnd(10)}  {"Reason"}</div>
  <div class="log-body" bind:this={container}>
{#each entries as entry}<div class="log-{entryClass(entry)}">{formatTime(entry.timestamp)}  {formatDuration(entry).padEnd(10)}  {(entry.level || '').padEnd(5)}  {formatMessage(entry)}</div>{:else}<span class="log-dim">waiting for data...</span>{/each}
  </div>
</div>

<style>
  .log-viewer {
    display: flex;
    flex-direction: column;
    height: 100%;
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    overflow: hidden;
  }

  .log-header {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: 0.25rem 0.5rem;
    border-bottom: 1px dotted var(--color-border);
    font-size: var(--text-xs);
    color: var(--color-text-muted);
  }

  .dot {
    display: inline-block;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--color-danger);
  }

  .dot.connected {
    background: var(--color-accent);
    box-shadow: 0 0 6px var(--color-accent);
  }

  .count {
    margin-left: auto;
  }

  .log-body {
    flex: 1;
    overflow-y: auto;
    font-size: var(--text-xs);
    line-height: 1.2;
    padding: 0.5rem;
    white-space: pre;
    background: var(--color-bg);
    color: var(--color-text-muted);
  }

  .log-ok { color: var(--color-accent); }
  .log-err { color: var(--color-danger); }
  .log-warn { color: var(--color-warning); }
  .log-blocked { color: var(--color-blocked); }
  .log-dim { color: var(--color-text-dim); }

  .log-col-hdr {
    white-space: pre;
    font-size: var(--text-xs);
    line-height: 1.2;
    padding: 0.35rem 0.5rem;
    color: var(--color-text);
    font-weight: 700;
    border-bottom: 1px solid var(--color-border);
    background: var(--color-bg);
    flex-shrink: 0;
  }
</style>
