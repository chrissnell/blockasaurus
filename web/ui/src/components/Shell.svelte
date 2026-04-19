<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { Tabs, StatusBar } from '@chrissnell/chonky-ui'
  import ApplyButton from './ApplyButton.svelte'
  import { getDirtyCount, clearDirty, onDirtyChange } from '../lib/dirty.svelte.js'
  import { apply, getVersion } from '../lib/api.js'
  import { getUser, isAdmin, logout } from '../lib/auth.svelte.js'

  let { currentPath = '/', children } = $props()
  let pendingCount = $state(getDirtyCount())
  let applying = $state(false)
  let version = $state('')
  let menuOpen = $state(false)

  function readInitialTheme() {
    if (typeof document === 'undefined') return 'light'
    const stored = localStorage.getItem('theme')
    if (stored === 'light' || stored === 'dark') return stored
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }

  let theme = $state(readInitialTheme())

  $effect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem('theme', theme)
  })

  function toggleTheme() {
    theme = theme === 'light' ? 'dark' : 'light'
  }

  $effect(() => {
    getVersion().then(v => { version = v })
  })

  $effect(() => {
    return onDirtyChange((n) => { pendingCount = n })
  })

  $effect(() => {
    if (!menuOpen) return
    function handleClick(e) {
      if (!e.target.closest('.header')) menuOpen = false
    }
    function handleKey(e) {
      if (e.key === 'Escape') menuOpen = false
    }
    document.addEventListener('click', handleClick)
    document.addEventListener('keydown', handleKey)
    return () => {
      document.removeEventListener('click', handleClick)
      document.removeEventListener('keydown', handleKey)
    }
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

  let user = $derived(getUser())
  let admin = $derived(isAdmin())

  const baseNav = [
    { path: '/', label: 'dashboard' },
    { path: '/logs', label: 'logs' },
    { path: '/client-groups', label: 'groups' },
    { path: '/blocklists', label: 'blocklists' },
    { path: '/domains', label: 'domains' },
    { path: '/custom-dns', label: 'dns' },
    { path: '/upstream-groups', label: 'upstreams' },
    { path: '/settings', label: 'settings' },
  ]

  let nav = $derived(admin ? [...baseNav, { path: '/users', label: 'users' }] : baseNav)
</script>

<Tabs.Root
  value={currentPath}
  onValueChange={(v) => { if (v) location.hash = v; menuOpen = false }}
  class="shell"
>
  <header class="header">
    <a href="#/" class="logo-link">
      <img src="/ui/blockasaurus-logo-face.svg" alt="" class="logo" />
      <h1>blockasaurus</h1>
    </a>
    <Tabs.List class="shell-tabs {menuOpen ? 'mobile-open' : ''}">
      {#each nav as item}
        <Tabs.Trigger value={item.path}>{item.label}</Tabs.Trigger>
      {/each}
    </Tabs.List>
    <div class="header-actions">
      {#if user}
        <span class="username">{user.username}</span>
        <button class="icon-btn" onclick={logout} title="Sign out" aria-label="Sign out">
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M6 2H3a1 1 0 0 0-1 1v10a1 1 0 0 0 1 1h3M11 11l3-3-3-3M14 8H6" />
          </svg>
        </button>
      {/if}
      <button
        class="icon-btn theme-btn"
        onclick={toggleTheme}
        title={theme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'}
        aria-label={theme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'}
      >
        {#if theme === 'light'}
          <!-- moon: clicking switches to dark -->
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M13.5 9.5A5.5 5.5 0 1 1 6.5 2.5a4.5 4.5 0 0 0 7 7Z" />
          </svg>
        {:else}
          <!-- sun: clicking switches to light -->
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="8" cy="8" r="3" />
            <path d="M8 1v2M8 13v2M1 8h2M13 8h2M3.05 3.05l1.4 1.4M11.55 11.55l1.4 1.4M3.05 12.95l1.4-1.4M11.55 4.45l1.4-1.4" />
          </svg>
        {/if}
      </button>
      <button
        class="menu-toggle"
        onclick={(e) => { e.stopPropagation(); menuOpen = !menuOpen }}
        aria-expanded={menuOpen}
        aria-label={menuOpen ? 'Close menu' : 'Open menu'}
      >
        {#if menuOpen}
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="4" y1="4" x2="16" y2="16" />
            <line x1="16" y1="4" x2="4" y2="16" />
          </svg>
        {:else}
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="3" y1="5" x2="17" y2="5" />
            <line x1="3" y1="10" x2="17" y2="10" />
            <line x1="3" y1="15" x2="17" y2="15" />
          </svg>
        {/if}
      </button>
    </div>
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
    max-width: 1280px;
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
    gap: 0.75rem;
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

  :global(.shell-tabs) :global([data-tabs-trigger]) {
    white-space: nowrap;
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

  .header-actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .username {
    font-size: var(--text-sm);
    color: var(--color-text-dim);
  }

  .icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    color: var(--color-text-dim);
    padding: 0.3rem;
    cursor: pointer;
    transition: color var(--transition);
  }

  .icon-btn:hover {
    color: var(--color-text);
  }

  .theme-btn {
    margin-left: 0.5rem;
  }

  .menu-toggle {
    display: none;
  }

  @media (max-width: 767px) {
    .menu-toggle {
      display: flex;
      align-items: center;
      justify-content: center;
      background: none;
      border: 1px solid var(--color-border);
      border-radius: var(--radius);
      color: var(--color-text);
      padding: 0.4rem;
      cursor: pointer;
    }

    .logo-link h1 {
      display: none;
    }

    .header {
      position: relative;
    }

    :global(.shell-tabs) {
      flex: unset;
      position: absolute;
      top: 100%;
      left: 0;
      right: 0;
      flex-direction: column;
      background: var(--color-bg);
      border: 1px solid var(--color-border);
      border-top: none;
      border-radius: 0 0 var(--radius) var(--radius);
      z-index: 50;
      display: none;
      transform: unset;
    }

    :global(.shell-tabs.mobile-open) {
      display: flex;
    }

    :global(.shell-tabs) :global([data-tabs-trigger]) {
      padding: 0.75rem 1rem;
      text-align: left;
      border-bottom: none;
      border-left: 3px solid transparent;
    }

    :global(.shell-tabs) :global([data-tabs-trigger][data-state="active"]) {
      border-bottom: none;
      border-left-color: var(--color-primary);
      background: var(--color-surface);
    }
  }
</style>
