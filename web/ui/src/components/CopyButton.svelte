<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  let { value = '' } = $props()
  let copied = $state(false)

  async function copy() {
    try {
      await navigator.clipboard.writeText(value)
      copied = true
      setTimeout(() => copied = false, 1500)
    } catch {
      // fallback for non-HTTPS contexts
      const ta = document.createElement('textarea')
      ta.value = value
      ta.style.position = 'fixed'
      ta.style.opacity = '0'
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
      copied = true
      setTimeout(() => copied = false, 1500)
    }
  }
</script>

<button class="copy-btn" onclick={copy} title="Copy to clipboard">
  {#if copied}
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
      <polyline points="20 6 9 17 4 12" />
    </svg>
  {:else}
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
    </svg>
  {/if}
</button>

<style>
  .copy-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: 1px solid var(--color-btn-border);
    border-radius: var(--radius);
    color: var(--color-text-dim);
    cursor: pointer;
    padding: 0.2rem;
    line-height: 1;
    transition: color var(--transition), border-color var(--transition);
    flex-shrink: 0;
  }

  .copy-btn:hover {
    color: var(--color-text);
    border-color: var(--color-text-muted);
  }
</style>
