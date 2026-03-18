<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import Card from '../components/Card.svelte'
  import Button from '../components/Button.svelte'
  import CopyButton from '../components/CopyButton.svelte'
  import DataTable from '../components/DataTable.svelte'
  import Modal from '../components/Modal.svelte'
  import FormField from '../components/FormField.svelte'
  import TextInput from '../components/TextInput.svelte'
  import EmptyState from '../components/EmptyState.svelte'
  import Autocomplete from '../components/Autocomplete.svelte'
  import { clientGroups, getDiscoveredClients, getEndpointInfo } from '../lib/api.js'
  import { markDirty } from '../lib/dirty.svelte.js'

  // --- State ---
  let groups = $state([])
  let loading = $state(true)

  // Detail view
  let selected = $state(null)
  let detailLoading = $state(false)

  // Create modal
  let createOpen = $state(false)
  let createName = $state('')

  // Discovered clients from ARP
  let discovered = $state([])

  // Endpoint configuration (loaded once)
  let endpointInfo = $state(null)

  // --- List view ---
  const columns = [
    { key: 'name', label: 'Name', sortable: true },
    { key: 'clients', label: 'Clients', render: (r) => `${r.clients?.length || 0}` },
    { key: 'groups', label: 'Blocklist Groups', render: (r) => `${r.groups?.length || 0}` },
  ]

  async function load() {
    loading = true
    try { groups = await clientGroups.list() ?? [] } catch { groups = [] }
    loading = false
  }

  // --- Create group (name only) ---
  function openCreate() {
    createName = ''
    createOpen = true
  }

  async function create() {
    if (!createName.trim()) return
    await clientGroups.put(createName.trim(), {})
    createOpen = false
    markDirty()
    await load()
  }

  async function remove(name) {
    await clientGroups.delete(name)
    markDirty()
    if (selected?.name === name) selected = null
    await load()
  }

  // --- Detail view ---
  async function openDetail(row) {
    detailLoading = true
    selected = row

    const [disc, epInfo] = await Promise.all([
      getDiscoveredClients().catch(() => []),
      endpointInfo ? Promise.resolve(endpointInfo) : getEndpointInfo().catch(() => null),
    ])
    discovered = disc ?? []
    if (epInfo) endpointInfo = epInfo

    detailLoading = false
  }

  // Build per-OS setup instructions from endpoint info
  function getSetupGuides(slug, info) {
    if (!info || !slug) return []
    const guides = []
    const dohPath = info.dohPath || '/dns-query'
    const hasDomains = info.domains?.length > 0
    const domain = hasDomains ? info.domains[0] : null
    const fqdn = domain ? `${slug}.${domain}` : null

    // Android — Private DNS (needs TLS + domain)
    if (info.hasTls && fqdn) {
      guides.push({
        os: 'Android',
        steps: [
          'Open Settings and tap Network & internet.',
          'Tap Private DNS.',
          'Select "Private DNS provider hostname" and enter the hostname below.',
        ],
        fields: [
          { label: 'Hostname', value: fqdn },
        ],
      })
    }

    // iOS / macOS — DNS configuration profile
    if (info.hasTls && fqdn) {
      guides.push({
        os: 'Apple (iOS / macOS)',
        steps: [
          'Create or download a DNS configuration profile (.mobileconfig) for your device.',
          'You can generate one at dns.notjakob.com/tool.html or write one manually.',
          'Use the encrypted DNS server URL below as the server address in the profile.',
          'On iOS: open the profile in Settings \u2192 General \u2192 VPN & Device Management to install it.',
          'On macOS: double-click the profile to open System Settings and install it.',
        ],
        fields: [
          { label: 'Server URL', value: `https://${fqdn}${dohPath}` },
        ],
      })
    } else if (info.hasHttp && fqdn) {
      guides.push({
        os: 'Apple (iOS / macOS)',
        steps: [
          'Create or download a DNS configuration profile (.mobileconfig) for your device.',
          'Use the server URL below. Note: this is unencrypted because TLS is not configured.',
          'On iOS: open the profile in Settings \u2192 General \u2192 VPN & Device Management to install it.',
          'On macOS: double-click the profile to open System Settings and install it.',
        ],
        fields: [
          { label: 'Server URL', value: `http://${fqdn}${dohPath}` },
        ],
      })
    }

    // Windows 11
    if (fqdn) {
      const url = info.hasTls ? `https://${fqdn}${dohPath}` : `http://${fqdn}${dohPath}`
      guides.push({
        os: 'Windows 11',
        steps: [
          'Open Settings \u2192 Network & internet and click on your active connection (Wi-Fi or Ethernet).',
          'Click Edit next to "DNS server assignment".',
          'Switch to Manual DNS and enable IPv4.',
          'Enter the blockasaurus server IP as the Preferred DNS.',
          'Under "DNS over HTTPS", select "On (manual template)" and enter the template below.',
        ],
        fields: [
          { label: 'DNS over HTTPS template', value: url },
        ],
      })
    } else if (info.hasHttp) {
      guides.push({
        os: 'Windows 11',
        steps: [
          'Open Settings \u2192 Network & internet and click on your active connection.',
          'Click Edit next to "DNS server assignment".',
          'Switch to Manual DNS, enable IPv4, and enter the blockasaurus server IP.',
          'Under "DNS over HTTPS", select "On (manual template)" and enter the template below.',
          'Replace <server> with the IP address or hostname of your blockasaurus server.',
        ],
        fields: [
          { label: 'DNS over HTTPS template', value: `http://<server>${dohPath}/${slug}` },
        ],
      })
    }

    // Linux — systemd-resolved
    if (info.hasTls && fqdn) {
      guides.push({
        os: 'Linux (systemd-resolved)',
        steps: [
          'Edit /etc/systemd/resolved.conf and add or update the [Resolve] section with the values below.',
          'Replace <server-ip> with the IP address of your blockasaurus server.',
          'Run "sudo systemctl restart systemd-resolved" to apply the changes.',
        ],
        fields: [
          { label: 'DNS', value: `<server-ip>#${fqdn}` },
          { label: 'DNSOverTLS', value: 'yes' },
        ],
      })
    }

    // Browsers
    if (info.hasHttp || info.hasTls) {
      const url = fqdn
        ? (info.hasTls ? `https://${fqdn}${dohPath}` : `http://${fqdn}${dohPath}`)
        : `http://<server>${dohPath}/${slug}`

      guides.push({
        os: 'Chrome / Edge / Brave',
        steps: [
          'Open the browser and go to Settings.',
          'Navigate to Privacy and security \u2192 Security.',
          'Scroll to "Use secure DNS" and enable it.',
          'Select "With: Custom" and paste the URL below.',
        ],
        fields: [
          { label: 'Custom DNS provider URL', value: url },
        ],
      })

      guides.push({
        os: 'Firefox',
        steps: [
          'Open Firefox and go to Settings.',
          'Navigate to Privacy & Security and scroll to "DNS over HTTPS".',
          'Select "Max Protection" (or "Increased Protection" if you want a fallback).',
          'Choose "Custom" as the provider and paste the URL below.',
        ],
        fields: [
          { label: 'Custom DNS provider URL', value: url },
        ],
      })
    }

    // Router / dnsmasq
    if (info.cpeId) {
      guides.push({
        os: 'Router running dnsmasq',
        steps: [
          'This is for routers or DNS forwarders that run dnsmasq and forward queries to blockasaurus.',
          'Add the configuration line below to your dnsmasq.conf file.',
          'This tags all DNS queries forwarded by this router so blockasaurus can identify them as belonging to this group.',
          'Restart dnsmasq after making the change.',
        ],
        fields: [
          { label: 'dnsmasq.conf line', value: `add-cpe-id=${slug}` },
        ],
      })
    }

    // Additional domain endpoints
    if (hasDomains && info.domains.length > 1) {
      const extra = []
      for (const d of info.domains.slice(1)) {
        const f = `${slug}.${d}`
        if (info.hasTls) {
          extra.push({ label: `Encrypted DNS URL (${d})`, value: `https://${f}${dohPath}` })
          extra.push({ label: `Encrypted DNS hostname (${d})`, value: f })
        } else if (info.hasHttp) {
          extra.push({ label: `DNS URL (${d})`, value: `http://${f}${dohPath}` })
        }
      }
      if (extra.length) {
        guides.push({
          os: 'Additional Domains',
          steps: [
            'The addresses below are also available for this group using your other configured domains.',
            'You can use these interchangeably with the addresses shown above.',
          ],
          fields: extra,
        })
      }
    }

    return guides
  }

  function backToList() {
    selected = null
  }

  // --- Client management ---
  async function addClient(value) {
    if (!value || !selected) return
    if (selected.clients.includes(value)) return

    const updated = { clients: [...selected.clients, value], groups: selected.groups }
    await clientGroups.put(selected.name, updated)
    selected = { ...selected, clients: updated.clients }
    markDirty()
    load()
  }

  async function removeClient(client) {
    if (!selected) return
    const updated = { clients: selected.clients.filter(c => c !== client), groups: selected.groups }
    await clientGroups.put(selected.name, updated)
    selected = { ...selected, clients: updated.clients }
    markDirty()
    load()
  }

  load()
</script>

<div class="page">
  {#if selected}
    <!-- DETAIL VIEW -->
    <div class="detail-header">
      <button class="back-btn" onclick={backToList}>&larr; back</button>
      <h1 class="page-title">{selected.name}</h1>
    </div>

    {#if detailLoading}
      <EmptyState message="Loading..." />
    {:else}
      <!-- Clients Card -->
      <Card title="Clients">
        <div class="chip-list">
          {#each selected.clients as client}
            <span class="chip">
              {client}
              <button class="chip-remove" onclick={() => removeClient(client)}>&times;</button>
            </span>
          {:else}
            <span class="empty-hint">no clients added yet</span>
          {/each}
        </div>
        <div class="add-section">
          <Autocomplete
            suggestions={discovered}
            placeholder="type IP, CIDR, or hostname — autocomplete from network"
            onadd={addClient}
          />
          {#if discovered.length > 0}
            <p class="hint">{discovered.length} device{discovered.length === 1 ? '' : 's'} discovered on network</p>
          {/if}
        </div>
      </Card>

      <!-- Device Setup Card -->
      {#if endpointInfo && selected.slug}
        {@const guides = getSetupGuides(selected.slug, endpointInfo)}
        {#if guides.length > 0}
          <Card title="Device Setup">
            <p class="setup-intro">
              Configure devices to use blockasaurus with the <strong>{selected.name}</strong> group.
              Copy the values below into each device's DNS settings.
            </p>
            <div class="guide-list">
              {#each guides as guide}
                <div class="guide">
                  <h3 class="guide-os">{guide.os}</h3>
                  <ol class="guide-steps">
                    {#each guide.steps as step}
                      <li>{step}</li>
                    {/each}
                  </ol>
                  {#each guide.fields as field}
                    <div class="guide-field">
                      <span class="field-label">{field.label}</span>
                      <div class="field-value-row">
                        <code class="field-value">{field.value}</code>
                        <CopyButton value={field.value} />
                      </div>
                    </div>
                  {/each}
                </div>
              {/each}
            </div>
          </Card>
        {/if}
      {/if}

    {/if}
  {:else}
    <!-- LIST VIEW -->
    <h1 class="page-title">Client Groups</h1>

    <Card>
      {#snippet actions()}
        <Button size="sm" onclick={openCreate}>Add Group</Button>
      {/snippet}
      {#if loading}
        <EmptyState message="Loading..." />
      {:else if groups.length === 0}
        <EmptyState message="no client groups configured">
          <Button size="sm" onclick={openCreate}>Create your first group</Button>
        </EmptyState>
      {:else}
        <DataTable {columns} rows={groups}>
          {#snippet rowActions(row)}
            <Button size="sm" onclick={() => openDetail(row)}>Manage</Button>
            {#if row.name !== 'default'}
              <Button size="sm" variant="danger" onclick={() => remove(row.name)}>Delete</Button>
            {/if}
          {/snippet}
        </DataTable>
      {/if}
    </Card>
  {/if}
</div>

<!-- Create Group Modal (name only) -->
<Modal bind:open={createOpen} title="New Client Group">
  <FormField label="Name">
    <TextInput bind:value={createName} placeholder="e.g. kids" />
  </FormField>
  <p class="modal-hint">You can add clients after creating the group.</p>
  {#snippet actions()}
    <Button onclick={() => createOpen = false}>Cancel</Button>
    <Button onclick={create} disabled={!createName.trim()}>Create</Button>
  {/snippet}
</Modal>

<style>
  .page { max-width: 1000px; }
  .page-title {
    font-size: var(--text-2xl);
    font-weight: 700;
    margin-bottom: var(--space-6);
  }

  .detail-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: var(--space-6);
  }

  .back-btn {
    background: none;
    border: 1px solid var(--color-btn-border);
    border-radius: var(--radius);
    color: var(--color-text-muted);
    padding: 0.2rem 0.5rem;
    font-size: var(--text-sm);
    cursor: pointer;
  }

  .back-btn:hover { color: var(--color-text); }

  .chip-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin-bottom: 1rem;
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
    color: var(--color-text-dim);
    font-size: var(--text-sm);
    font-style: italic;
  }

  .add-section {
    border-top: 1px dotted var(--color-border);
    padding-top: 0.75rem;
  }

  .hint {
    font-size: var(--text-xs);
    color: var(--color-text-dim);
    margin-top: 0.25rem;
  }

  .modal-hint {
    font-size: var(--text-sm);
    color: var(--color-text-dim);
    margin-top: 0.5rem;
  }

  /* Device Setup */

  .setup-intro {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    margin-bottom: 1rem;
  }

  .guide-list {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .guide {
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    padding: 0.75rem 1rem;
  }

  .guide-os {
    font-size: var(--text-sm);
    font-weight: 600;
    margin: 0 0 0.25rem 0;
  }

  .guide-steps {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    margin: 0 0 0.5rem 0;
    padding-left: 1.4rem;
    line-height: 1.6;
  }

  .guide-steps li {
    margin-bottom: 0.15rem;
  }

  .guide-field {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
    margin-top: 0.35rem;
  }

  .field-label {
    font-size: var(--text-xs);
    color: var(--color-text-dim);
  }

  .field-value-row {
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }

  .field-value {
    font-family: var(--font-mono, monospace);
    font-size: var(--text-xs);
    background: var(--color-btn-bg);
    border: 1px solid var(--color-btn-border);
    border-radius: var(--radius);
    padding: 0.2rem 0.5rem;
    word-break: break-all;
    flex: 1;
    min-width: 0;
  }
</style>
