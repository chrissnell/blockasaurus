<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { Box, Button, Label, Select, Input, Toaster, toast } from '@chrissnell/chonky-ui'
  import { upstreamSettings } from '../lib/api.js'
  import { markDirty } from '../lib/dirty.svelte.js'

  let strategy = $state('parallel_best')
  let timeout = $state('2s')
  let userAgent = $state('')
  let initStrategy = $state('blocking')
  let loading = $state(true)
  let saving = $state(false)

  async function load() {
    loading = true
    try {
      const data = await upstreamSettings.get()
      strategy = data.strategy || 'parallel_best'
      timeout = data.timeout || '2s'
      userAgent = data.user_agent || ''
      initStrategy = data.init_strategy || 'blocking'
    } catch (e) {
      toast(e.message, 'danger')
    }
    loading = false
  }

  async function save() {
    saving = true
    try {
      await upstreamSettings.update({
        strategy,
        timeout,
        user_agent: userAgent,
        init_strategy: initStrategy,
      })
      markDirty()
      toast('Upstream settings saved', 'success')
    } catch (e) {
      toast(e.message, 'danger')
    }
    saving = false
  }

  load()
</script>

<div class="page">
  <h1 class="page-title">Upstream Settings</h1>

  <div class:loading-state={loading}>
    <Box title="Resolver Behavior">
      <div class="form-layout">
        <div class="form-field">
          <Label for="strategy">Resolution Strategy</Label>
          <Select
            id="strategy"
            bind:value={strategy}
            options={[
              { value: 'parallel_best', label: 'Parallel Best (query all, use fastest)' },
              { value: 'strict', label: 'Strict (try in order)' },
              { value: 'random', label: 'Random (pick one)' },
            ]}
          />
        </div>
        <div class="form-field">
          <Label for="timeout">Timeout</Label>
          <Input id="timeout" bind:value={timeout} placeholder="2s, 500ms, 1s" />
        </div>
        <div class="form-field">
          <Label for="init-strategy">Init Strategy</Label>
          <Select
            id="init-strategy"
            bind:value={initStrategy}
            options={[
              { value: 'blocking', label: 'Blocking (probe on apply, warn on failure)' },
              { value: 'failOnError', label: 'Fail on error (reject apply if any probe fails)' },
              { value: 'fast', label: 'Fast (no probe)' },
            ]}
          />
        </div>
        <div class="form-field">
          <Label for="user-agent">DoH User-Agent</Label>
          <Input id="user-agent" bind:value={userAgent} placeholder="(optional)" />
        </div>
        <div class="form-actions">
          <Button onclick={save} disabled={saving}>
            {saving ? 'Saving...' : 'Save Settings'}
          </Button>
        </div>
      </div>
    </Box>
  </div>
</div>

<Toaster />

<style>
  .page {
    max-width: 600px;
  }
  .page-title {
    font-size: var(--text-2xl);
    font-weight: 700;
    margin-bottom: var(--space-6);
  }
  .loading-state {
    opacity: 0.5;
    pointer-events: none;
  }
  .form-layout {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }
  .form-field {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }
  .form-actions {
    display: flex;
    justify-content: flex-end;
    padding-top: var(--space-2);
  }
</style>
