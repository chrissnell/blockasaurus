<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { Box, Button, Label, Select, Input, Toggle, Toaster, toast } from '@chrissnell/chonky-ui'
  import { blockSettings, rebindingSettings } from '../lib/api.js'
  import { markDirty } from '../lib/dirty.svelte.js'

  const namedBlockTypes = [
    { value: 'ZEROIP', label: 'Zero IP (0.0.0.0)' },
    { value: 'NXDOMAIN', label: 'NXDOMAIN' },
    { value: 'REFUSED', label: 'REFUSED' },
  ]

  let blockType = $state('ZEROIP')
  let blockTTL = $state('1m')
  let loading = $state(true)
  let saving = $state(false)

  let rebindEnabled = $state(false)
  let allowedDomains = $state([])
  let domainInput = $state('')
  let domainError = $state('')
  let rebindSaving = $state(false)

  // The API also accepts a comma-separated list of block IP addresses, which no
  // named option can represent. Keep a stored value like that selectable so
  // saving an unrelated change doesn't silently rewrite the block type.
  let blockTypeOptions = $derived(
    namedBlockTypes.some((o) => o.value === blockType)
      ? namedBlockTypes
      : [...namedBlockTypes, { value: blockType, label: `${blockType} (custom)` }],
  )

  async function load() {
    loading = true
    try {
      const data = await blockSettings.get()
      blockType = data.block_type || 'ZEROIP'
      blockTTL = data.block_ttl || '1m'
    } catch { /* use defaults */ }
    try {
      const data = await rebindingSettings.get()
      rebindEnabled = data.enabled ?? false
      allowedDomains = data.allowed_domains ?? []
    } catch { /* use defaults */ }
    loading = false
  }

  // Mirrors config.ValidateAllowedDomain on the server. Duplicated here only to
  // fail fast in the field — the server stays authoritative.
  function domainProblem(domain) {
    if (!domain.trim()) return 'Enter a domain'
    if (/[*/]/.test(domain) || /\s/.test(domain)) {
      return 'Plain domains only — no wildcards, regexes or spaces. Subdomains match automatically.'
    }
    if (domain.startsWith('.') || domain.includes('..')) return 'Not a valid domain'
    const key = domain.toLowerCase().replace(/\.$/, '')
    if (allowedDomains.some((d) => d.toLowerCase().replace(/\.$/, '') === key)) {
      return 'Already in the list'
    }
    return ''
  }

  function addDomain() {
    const domain = domainInput.trim()
    const problem = domainProblem(domain)
    if (problem) {
      domainError = problem
      return
    }
    allowedDomains = [...allowedDomains, domain]
    domainInput = ''
    domainError = ''
  }

  function removeDomain(domain) {
    allowedDomains = allowedDomains.filter((d) => d !== domain)
  }

  async function saveRebinding() {
    rebindSaving = true
    try {
      await rebindingSettings.update({ enabled: rebindEnabled, allowed_domains: allowedDomains })
      markDirty()
      toast('Rebinding protection saved', 'success')
    } catch (e) {
      toast(e.message, 'danger')
    }
    rebindSaving = false
  }

  async function save() {
    saving = true
    try {
      await blockSettings.update({ block_type: blockType, block_ttl: blockTTL })
      markDirty()
      toast('Block settings saved', 'success')
    } catch (e) {
      toast(e.message, 'danger')
    }
    saving = false
  }

  load()
</script>

<div class="page">
  <h1 class="page-title">Settings</h1>

  <div class:loading-state={loading}>
    <Box title="Response Behavior">
      <div class="form-layout">
        <div class="form-field">
          <Label for="block-type">Block Type</Label>
          <Select id="block-type" bind:value={blockType} options={blockTypeOptions} />
        </div>
        <div class="form-field">
          <Label for="block-ttl">Block TTL</Label>
          <Input id="block-ttl" bind:value={blockTTL} placeholder="1m, 30s, 1h" />
        </div>
        <div class="form-actions">
          <Button onclick={save} disabled={saving}>
            {saving ? 'Saving...' : 'Save Settings'}
          </Button>
        </div>
      </div>
    </Box>

    <Box title="DNS Rebinding Protection">
      <p class="section-hint">
        Drops answers from the upstream resolvers that map a name to a private,
        loopback, link-local or unspecified address — the DNS rebinding attack, where
        a hostile site resolves its own domain into your LAN. Answers from custom DNS,
        the hosts file and conditional upstreams are never inspected.
      </p>

      <div class="form-layout">
        <Toggle bind:checked={rebindEnabled} label="Enable rebinding protection" />

        <div class="form-field">
          <Label for="allowed-domain">Allowed Domains</Label>
          <p class="section-hint">
            Split-horizon names that legitimately resolve to a private address through
            the upstreams. Each entry also matches its subdomains. Without an entry
            here, such a name silently resolves to nothing for every client.
          </p>
          <div class="chip-list">
            {#each allowedDomains as domain}
              <span class="chip">
                {domain}
                <button
                  class="chip-remove"
                  onclick={() => removeDomain(domain)}
                  aria-label={`Remove ${domain}`}
                >&times;</button>
              </span>
            {:else}
              <span class="empty-hint">no allowed domains</span>
            {/each}
          </div>
          <div class="add-row">
            <Input
              id="allowed-domain"
              bind:value={domainInput}
              placeholder="nas.example.com"
              oninput={() => (domainError = '')}
              onkeydown={(e) => e.key === 'Enter' && addDomain()}
            />
            <Button onclick={addDomain}>Add</Button>
          </div>
          {#if domainError}
            <p class="field-error">{domainError}</p>
          {/if}
        </div>

        <div class="form-actions">
          <Button onclick={saveRebinding} disabled={rebindSaving}>
            {rebindSaving ? 'Saving...' : 'Save Rebinding Settings'}
          </Button>
        </div>
      </div>
    </Box>
  </div>
</div>

<Toaster />

<style>
  .page { max-width: 600px; }
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
  .section-hint {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
    line-height: 1.5;
    margin: 0 0 var(--space-3);
  }
  .chip-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin-bottom: 0.75rem;
    min-height: 1.5rem;
  }
  .chip {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    font-size: var(--text-sm);
    padding: 0.15rem 0.5rem;
    border: 1px solid var(--color-btn-border);
    border-radius: var(--radius);
    background: var(--color-btn-bg);
  }
  .chip-remove {
    background: none;
    border: none;
    color: var(--color-text-dim);
    cursor: pointer;
    font-size: 1rem;
    line-height: 1;
    padding: 0 0.1rem;
  }
  .chip-remove:hover { color: var(--color-danger); }
  .empty-hint {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }
  .add-row {
    display: flex;
    gap: var(--space-2);
    align-items: center;
  }
  .add-row :global(input) { flex: 1; }
  .field-error {
    color: var(--color-danger);
    font-size: var(--text-sm);
    margin: var(--space-2) 0 0;
  }
</style>
