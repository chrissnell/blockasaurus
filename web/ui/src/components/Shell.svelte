<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { Tabs, StatusBar, ThemeToggle } from '@chrissnell/chonky-ui'
  import ApplyButton from './ApplyButton.svelte'
  import { getDirtyCount, clearDirty, onDirtyChange } from '../lib/dirty.svelte.js'
  import { apply, getVersion } from '../lib/api.js'

  let { currentPath = '/', children } = $props()
  let pendingCount = $state(getDirtyCount())
  let applying = $state(false)
  let version = $state('')

  $effect(() => {
    getVersion().then(v => { version = v })
  })

  $effect(() => {
    return onDirtyChange((n) => { pendingCount = n })
  })

  async function handleApply() {
    applying = true
    try {
      await apply()
      clearDirty()
    } catch (e) {
      console.error('apply failed:', e)
    }
    applying = false
  }

  const nav = [
    { path: '/', label: 'dashboard' },
    { path: '/logs', label: 'live logs' },
    { path: '/client-groups', label: 'client groups' },
    { path: '/blocklists', label: 'blocklists' },
    { path: '/domains', label: 'domains' },
    { path: '/custom-dns', label: 'custom dns' },
    { path: '/upstream-groups', label: 'upstreams' },
    { path: '/settings', label: 'settings' },
  ]
</script>

<Tabs.Root
  value={currentPath}
  onValueChange={(v) => { if (v) location.hash = v }}
  class="shell"
>
  <header class="header">
    <a href="#/" class="logo-link">
      <img src="/ui/blockasaurus-logo-face.svg" alt="" class="logo" />
      <h1>blockasaurus</h1>
    </a>
    <Tabs.List class="shell-tabs">
      {#each nav as item}
        <Tabs.Trigger value={item.path}>{item.label}</Tabs.Trigger>
      {/each}
    </Tabs.List>
    <ThemeToggle />
  </header>

  <ApplyButton pending={pendingCount} loading={applying} onclick={handleApply} />

  <main class="content">
    {@render children()}
  </main>

  <StatusBar class="shell-status">
    {#if version}
      <a href="https://github.com/chrissnell/blockasaurus" target="_blank" rel="noopener">
        Blockasaurus {version}
      </a>
    {/if}
    {#if pendingCount > 0}
      <span class="pending">{pendingCount} unsaved</span>
    {/if}
  </StatusBar>
</Tabs.Root>

<style>
  :global(.shell) {
    max-width: 1100px;
    margin: 0 auto;
    padding: 2rem 1rem;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.5rem;
    border-bottom: 1px solid var(--color-border);
    padding-bottom: 0.75rem;
    gap: 1rem;
  }

  .logo-link {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    text-decoration: none;
    color: inherit;
  }

  .logo {
    width: 1.75rem;
    height: 1.75rem;
  }

  h1 {
    font-size: var(--text-2xl);
    font-weight: 700;
  }

  /* Let Chonky style the Tabs.List underline; we only need layout tweaks.
     translateY aligns tab text baseline with the h1 "blockasaurus" baseline —
     h1 is 1.4rem and tab triggers are 0.8rem, so centered in the same flex row
     the tab baseline sits ~6px above the h1 baseline. */
  :global(.shell-tabs) {
    flex: 1;
    border-bottom: none;
    transform: translateY(3px);
  }

  .content {
    min-height: 80vh;
  }

  :global(.shell-status) {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: 0.5rem 1rem;
    border-top: 1px dotted var(--color-border);
    font-size: var(--text-sm);
    color: var(--color-text-dim);
  }

  :global(.shell-status) a {
    color: var(--color-text-dim);
    text-decoration: none;
    border-bottom: 1px dotted var(--color-border);
    transition: color var(--transition);
  }

  :global(.shell-status) a:hover {
    color: var(--color-text);
  }

  .pending {
    margin-left: auto;
    color: var(--color-warning);
  }
</style>
