<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { Box, Button, Label, Select, Input, Toggle, Toaster, toast } from '@chrissnell/chonky-ui'
  import { blockSettings, rebindingSettings, http3Settings } from '../lib/api.js'
  import { markDirty } from '../lib/dirty.svelte.js'

  const namedBlockTypes = [
    { value: 'ZEROIP', label: 'Zero IP (0.0.0.0)' },
    { value: 'NXDOMAIN', label: 'NXDOMAIN' },
    { value: 'REFUSED', label: 'REFUSED' },
  ]

  const DEFAULT_BLOCK_TYPE = 'ZEROIP'
  const DEFAULT_BLOCK_TTL = '1m'

  let blockType = $state(DEFAULT_BLOCK_TYPE)
  let blockTTL = $state(DEFAULT_BLOCK_TTL)
  let loading = $state(true)
  let saving = $state(false)
  let blockLoadFailed = $state(false)

  let h3Enabled = $state(false)
  let h3Active = $state(false)
  let h3RestartRequired = $state(false)
  let h3Unavailable = $state('')
  let h3LoadFailed = $state(false)

  let rebindEnabled = $state(false)
  let allowedDomains = $state([])
  let domainInput = $state('')
  let domainError = $state('')
  let rebindLoadFailed = $state(false)

  // What the server last confirmed. The page saves through one button, so what
  // each section needs is not a hand-maintained dirty flag but an answer to
  // "does this differ from what is stored" -- which is also what keeps an
  // untouched section out of the save entirely.
  let saved = $state({
    blockType: DEFAULT_BLOCK_TYPE,
    blockTTL: DEFAULT_BLOCK_TTL,
    rebindEnabled: false,
    allowedDomains: [],
    h3Enabled: false,
  })

  const blockChanged = $derived(
    blockType !== saved.blockType || blockTTL !== saved.blockTTL,
  )
  const rebindChanged = $derived(
    rebindEnabled !== saved.rebindEnabled
      || allowedDomains.length !== saved.allowedDomains.length
      || allowedDomains.some((d, i) => d !== saved.allowedDomains[i]),
  )
  const h3Changed = $derived(h3Enabled !== saved.h3Enabled)
  // A section whose read failed is excluded outright. Its fields show defaults
  // we never confirmed, so "changed" there means changed from a guess, and
  // saving it would write that guess over whatever is really in effect.
  const blockSavable = $derived(blockChanged && !blockLoadFailed)
  const rebindSavable = $derived(rebindChanged && !rebindLoadFailed)
  const h3Savable = $derived(h3Changed && !h3LoadFailed)
  const dirty = $derived(blockSavable || rebindSavable || h3Savable)

  // The API also accepts a comma-separated list of block IP addresses, which no
  // named option can represent. Keep a stored value like that selectable so
  // saving an unrelated change doesn't silently rewrite the block type. Keyed
  // off what is stored rather than what is selected: reading it off the
  // selection would drop the custom entry the moment a named option was tried,
  // leaving no way back to it.
  let blockTypeOptions = $derived(
    namedBlockTypes.some((o) => o.value === saved.blockType)
      ? namedBlockTypes
      : [...namedBlockTypes, { value: saved.blockType, label: `${saved.blockType} (custom)` }],
  )

  async function load() {
    loading = true
    // Same no-silent-fallback rule the other two sections carry. Block type in
    // particular can be a comma-separated IP list no default could stand in for,
    // so a failed read that let Save proceed would quietly replace it with
    // ZEROIP the next time anyone touched the TTL.
    try {
      const data = await blockSettings.get()
      blockType = data.block_type || DEFAULT_BLOCK_TYPE
      blockTTL = data.block_ttl || DEFAULT_BLOCK_TTL
      saved.blockType = blockType
      saved.blockTTL = blockTTL
      blockLoadFailed = false
    } catch {
      blockLoadFailed = true
    }
    // Defaulting to "off, empty allowlist" would render a security control as
    // disabled when we simply failed to read it, and the next Save would make
    // that true.
    try {
      const data = await rebindingSettings.get()
      rebindEnabled = data.enabled ?? false
      allowedDomains = data.allowed_domains ?? []
      saved.rebindEnabled = rebindEnabled
      saved.allowedDomains = [...allowedDomains]
      rebindLoadFailed = false
    } catch {
      rebindLoadFailed = true
    }
    // Same no-silent-fallback rule as rebinding: rendering "off, not serving" on
    // a failed read would be indistinguishable from the real thing, and saving
    // that would turn DoH3 off on the next restart.
    try {
      applyH3(await http3Settings.get())
      h3LoadFailed = false
    } catch {
      h3LoadFailed = true
    }
    loading = false
  }

  function applyH3(data) {
    h3Enabled = data.enabled ?? false
    h3Active = data.active ?? false
    h3RestartRequired = data.restart_required ?? false
    h3Unavailable = data.unavailable_reason ?? ''
    saved.h3Enabled = h3Enabled
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

  function removeDomain(index) {
    allowedDomains = allowedDomains.filter((_, i) => i !== index)
  }

  // One save for the page. Only sections that actually changed are sent, and
  // never a section whose read failed -- see the `*Savable` deriveds. The
  // controls in such a section are disabled too, but that is the courtesy; this
  // is the guarantee.
  async function save() {
    saving = true

    const failed = []
    // DoH3 is deliberately excluded from this: the header's Apply rebuilds the
    // resolver chain and cannot open or close a listener, so sending the
    // operator there would promise an effect it does not have. The server says
    // so too, via restart_required.
    let needsApply = false

    if (blockSavable) {
      try {
        await blockSettings.update({ block_type: blockType, block_ttl: blockTTL })
        saved.blockType = blockType
        saved.blockTTL = blockTTL
        needsApply = true
      } catch (e) {
        failed.push(`response behavior: ${e.message}`)
      }
    }

    if (rebindSavable) {
      try {
        await rebindingSettings.update({
          enabled: rebindEnabled,
          allowed_domains: allowedDomains,
        })
        saved.rebindEnabled = rebindEnabled
        saved.allowedDomains = [...allowedDomains]
        needsApply = true
      } catch (e) {
        failed.push(`rebinding protection: ${e.message}`)
      }
    }

    if (h3Savable) {
      try {
        applyH3(await http3Settings.update({ enabled: h3Enabled }))
      } catch (e) {
        failed.push(`DoH3: ${e.message}`)
      }
    }

    if (needsApply) markDirty()

    if (failed.length) {
      toast(`Could not save ${failed.join('; ')}`, 'danger')
    } else {
      toast('Settings saved', 'success')
    }

    saving = false
  }

  load()
</script>

<div class="page">
  <h1 class="page-title">Settings</h1>

  <Box>
    <div class:loading-state={loading}>
      <section class="section">
        <h2 class="section-title">Response Behavior</h2>

        {#if blockLoadFailed}
          <p class="load-error" role="alert">
            Could not load the current blocking settings, so this section is not
            showing what is actually in effect. Reload the page before changing
            anything.
          </p>
        {/if}

        <div class="form-layout" class:loading-state={blockLoadFailed}>
          <div class="form-field">
            <Label for="block-type">Block Type</Label>
            <Select
              id="block-type"
              bind:value={blockType}
              options={blockTypeOptions}
              disabled={blockLoadFailed}
            />
          </div>
          <div class="form-field">
            <Label for="block-ttl">Block TTL</Label>
            <Input
              id="block-ttl"
              bind:value={blockTTL}
              disabled={blockLoadFailed}
              placeholder="1m, 30s, 1h"
            />
          </div>
        </div>
      </section>

      <section class="section">
        <h2 class="section-title">DNS Rebinding Protection</h2>
        <p class="section-hint">
          Drops answers from the upstream resolvers that map a name to a private,
          loopback, link-local or unspecified address — the DNS rebinding attack, where
          a hostile site resolves its own domain into your LAN. Answers from custom DNS,
          the hosts file and conditional upstreams are never inspected.
        </p>

        {#if rebindLoadFailed}
          <p class="load-error" role="alert">
            Could not load the current rebinding settings, so this section is not
            showing what is actually in effect. Reload the page before changing
            anything.
          </p>
        {/if}

        <div class="form-layout" class:loading-state={rebindLoadFailed}>
          <Toggle
            bind:checked={rebindEnabled}
            disabled={rebindLoadFailed}
            label="Enable rebinding protection"
          />

          <div class="form-field">
            <Label for="allowed-domain">Allowed Domains</Label>
            <p class="section-hint">
              Split-horizon names that legitimately resolve to a private address through
              the upstreams. Each entry also matches its subdomains. Without an entry
              here, such a name silently resolves to nothing for every client.
            </p>
            <div class="chip-list">
              {#each allowedDomains as domain, i (i)}
                <span class="chip">
                  {domain}
                  <button
                    type="button"
                    class="chip-remove"
                    disabled={rebindLoadFailed}
                    onclick={() => removeDomain(i)}
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
                disabled={rebindLoadFailed}
                placeholder="nas.example.com"
                oninput={() => (domainError = '')}
                onkeydown={(e) => e.key === 'Enter' && addDomain()}
              />
              <Button onclick={addDomain} disabled={rebindLoadFailed}>Add</Button>
            </div>
            {#if domainError}
              <p class="field-error" role="alert">{domainError}</p>
            {/if}
          </div>
        </div>
      </section>

      <section class="section">
        <h2 class="section-title">DNS over HTTP/3 (DoH3)</h2>
        <p class="section-hint">
          Answers the DoH endpoint over HTTP/3 (RFC 9114) on the UDP counterparts of
          the HTTPS port, in addition to HTTPS over TCP. Clients that support it
          switch transports on their own &mdash; the DoH URL does not change, so
          there is nothing to reconfigure on the client side.
        </p>

        {#if h3LoadFailed}
          <p class="load-error" role="alert">
            Could not load the current DoH3 setting, so this section is not showing
            what is actually in effect. Reload the page before changing anything.
          </p>
        {/if}

        <div class="form-layout" class:loading-state={h3LoadFailed}>
          <Toggle
            bind:checked={h3Enabled}
            disabled={h3LoadFailed}
            label="Serve DoH over HTTP/3"
          />

          <!-- Suppressed on a failed read: "not serving" is a claim about the
               running process, and we would not have grounds for it. -->
          {#if !h3LoadFailed}
            <p class="status-line">
              {h3Active
                ? 'This server is currently serving DoH over HTTP/3.'
                : 'This server is not currently serving DoH over HTTP/3.'}
            </p>
          {/if}

          <!-- An HTTP-only deployment is permanently unavailable for HTTP/3 and
               never asked for it, so the reason is only an error for someone who
               actually wants DoH3 on. Styling it red either way would cry wolf on
               the Settings page of a default install. -->
          {#if h3Unavailable && h3Enabled}
            <p class="field-error" role="alert">
              DoH3 is on, but HTTP/3 cannot start in this process: {h3Unavailable}.
              A restart will not change that &mdash; fix it in the YAML config first.
            </p>
          {:else if h3Unavailable}
            <p class="restart-hint">
              HTTP/3 is not available on this server: {h3Unavailable}. Turning this
              on will not change that until the YAML config does.
            </p>
          {:else if h3RestartRequired && !h3Changed}
            <p class="restart-hint" role="status">
              Saved, but not yet in effect. Unlike everything else on this page,
              this one needs a restart of blockasaurus &mdash; <b>Apply</b> rebuilds
              the resolver chain and cannot open or close a listener.
            </p>
          {/if}
        </div>
      </section>

      <div class="form-actions">
        {#if dirty}
          <span class="unsaved-hint">unsaved changes</span>
        {/if}
        <Button onclick={save} disabled={saving || !dirty}>
          {saving ? 'Saving...' : 'Save Settings'}
        </Button>
      </div>
    </div>
  </Box>
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
  .section + .section {
    margin-top: var(--space-6);
    padding-top: var(--space-6);
    border-top: 1px dotted var(--color-border);
  }
  /* Matches chonky's .box-title, which is what each of these sections used to
     have of its own before they moved into one box. */
  .section-title {
    font-size: var(--text-xs);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--color-text-muted);
    margin-bottom: var(--space-4);
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
  /* The layout spaces its own rows; the inputs' stacked-form margin only adds
     an uneven second gap on top of that. */
  .form-layout :global(input) {
    margin-bottom: 0;
  }
  .form-actions {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    margin-top: var(--space-6);
    padding-top: var(--space-4);
    border-top: 1px dotted var(--color-border);
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
    font-family: inherit;
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
  /* The flex item is Input's wrapper, not the <input> inside it: sizing the
     inner element leaves the wrapper to shrink-wrap and the field ends up
     narrower than the row. The wrapper also carries the input's stacked-form
     margin-bottom, which is what pushed the field off the Add button's centre. */
  .add-row :global(.input-wrapper) { flex: 1; }
  .add-row :global(input) { margin-bottom: 0; }
  .field-error {
    color: var(--color-danger);
    font-size: var(--text-sm);
    margin: var(--space-2) 0 0;
  }
  .load-error {
    color: var(--color-danger);
    font-size: var(--text-sm);
    line-height: 1.5;
    margin: 0 0 var(--space-3);
  }
  .status-line {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
    margin: 0;
  }
  .restart-hint {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
    line-height: 1.5;
    margin: 0;
  }
  .unsaved-hint {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
    margin-right: var(--space-3);
  }
</style>
