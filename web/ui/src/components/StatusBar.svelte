<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  let { version = '', pendingChanges = 0 } = $props()
</script>

<footer class="status">
  {#if version}
    <a href="https://github.com/chrissnell/blockasaurus" target="_blank" rel="noopener">Blockasaurus v{version}</a>
  {/if}
  {#if pendingChanges > 0}
    <span class="pending">{pendingChanges} unsaved</span>
  {/if}
</footer>

<style>
  .status {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: 0.5rem 1rem;
    border-top: 1px dotted var(--color-border);
    font-size: var(--text-sm);
    color: var(--color-text-dim);
  }

  a {
    color: var(--color-text-dim);
    text-decoration: none;
    border-bottom: 1px dotted var(--color-border);
    transition: color var(--transition);
  }

  a:hover {
    color: var(--color-text);
  }

  .pending {
    margin-left: auto;
    color: var(--color-warning);
  }
</style>
