<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import Shell from './components/Shell.svelte'
  import Dashboard from './pages/Dashboard.svelte'
  import ClientGroups from './pages/ClientGroups.svelte'
  import Blocklists from './pages/Blocklists.svelte'
  import DomainEntries from './pages/DomainEntries.svelte'
  import CustomDNS from './pages/CustomDNS.svelte'
  import UpstreamGroups from './pages/UpstreamGroups.svelte'
  import BlockSettings from './pages/BlockSettings.svelte'
  import Logs from './pages/Logs.svelte'
  import DevComponents from './pages/DevComponents.svelte'
  import Login from './pages/Login.svelte'
  import Setup from './pages/Setup.svelte'
  import Users from './pages/Users.svelte'
  import { authState, checkSession } from './lib/auth.svelte.js'

  let currentPath = $state(location.hash.slice(1) || '/')

  // checkSession() runs exactly once on mount — NOT on hash changes. Wrapping
  // it in a zero-dep $effect with no reactive reads achieves that.
  $effect(() => {
    checkSession()
  })

  // Hash-based routing: a separate effect so it doesn't re-trigger session probe.
  $effect(() => {
    function onHash() {
      currentPath = location.hash.slice(1) || '/'
    }
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  })

  const routes = {
    '/': Dashboard,
    '/client-groups': ClientGroups,
    '/blocklists': Blocklists,
    '/domains': DomainEntries,
    '/custom-dns': CustomDNS,
    '/upstream-groups': UpstreamGroups,
    '/settings': BlockSettings,
    '/logs': Logs,
    '/users': Users,
    '/dev': DevComponents,
    '/login': Login,
  }

  // App-level route gating. Nav-hiding alone doesn't stop a viewer from
  // pasting #/users into the address bar (empty shell + 401 flash). Two
  // pieces work together:
  //   1. $derived below swaps the guarded Page for Dashboard so the
  //      protected component never mounts, even for a single render.
  //   2. The $effect below corrects the hash so the address bar reflects
  //      the effective route (and hashchange fires downstream listeners).
  function isRouteAllowed(path, user) {
    if (path === '/users') return user?.role === 'admin'
    return true
  }

  $effect(() => {
    if (
      authState.status === 'authed' &&
      !isRouteAllowed(currentPath, authState.user) &&
      window.location.hash !== '#/'
    ) {
      window.location.hash = '#/'
    }
  })

  let Page = $derived(
    isRouteAllowed(currentPath, authState.user)
      ? (routes[currentPath] || Dashboard)
      : Dashboard
  )
</script>

{#if authState.status === 'loading'}
  <div class="loading">
    <img src="/ui/blockasaurus-logo-face.svg" alt="" class="loading-logo" />
    <p>Loading...</p>
  </div>
{:else if authState.status === 'setup'}
  <Setup />
{:else if authState.status === 'anon'}
  <Login />
{:else}
  <Shell {currentPath}>
    <Page />
  </Shell>
{/if}

<style>
  .loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 60vh;
    color: var(--color-text-dim);
  }

  .loading-logo {
    width: 3rem;
    height: 3rem;
    margin-bottom: 1rem;
    opacity: 0.5;
  }
</style>
