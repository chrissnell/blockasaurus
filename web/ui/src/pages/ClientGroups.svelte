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
    activeTab = null

    const [disc, epInfo] = await Promise.all([
      getDiscoveredClients().catch(() => []),
      endpointInfo ? Promise.resolve(endpointInfo) : getEndpointInfo().catch(() => null),
    ])
    discovered = disc ?? []
    if (epInfo) endpointInfo = epInfo

    detailLoading = false
  }

  // Tabbed setup guide data
  let activeTab = $state(null)

  function getSetupTabs(slug, info) {
    if (!info || !slug) return []
    const tabs = []
    const dohPath = info.dohPath || '/dns-query'
    const domain = info.domains?.length > 0 ? info.domains[0] : null
    const fqdn = domain ? `${slug}.${domain}` : null
    const httpsUrl = fqdn ? `https://${fqdn}${dohPath}` : null
    const httpUrl = fqdn ? `http://${fqdn}${dohPath}` : null
    const url = (info.hasTls && httpsUrl) ? httpsUrl : httpUrl
    const fallbackUrl = `http://&lt;server&gt;${dohPath}/${slug}`

    // Windows — most common desktop OS
    if (fqdn || info.hasHttp) {
      const sections = []
      if (fqdn) {
        sections.push({
          title: 'Secure DNS',
          subtitle: 'Windows 11',
          recommended: true,
          steps: [
            'Open <b>Settings</b> and go to <b>Network & internet</b>.',
            'Click your active connection (<b>Wi-Fi</b> or <b>Ethernet</b>).',
            'Click <b>Hardware properties</b>.',
            'Next to <b>DNS server assignment</b>, click <b>Edit</b>.',
            'Set to <b>Manual</b>, toggle <b>IPv4</b> on, and enter your blockasaurus server IP as the <b>Preferred DNS</b>.',
            'Set <b>DNS over HTTPS</b> to <b>On (manual template)</b> and enter the template below.',
            'Click <b>Save</b>.',
          ],
          fields: [{ label: 'DNS over HTTPS template', value: url }],
        })
      } else {
        sections.push({
          title: 'Secure DNS',
          subtitle: 'Windows 11',
          recommended: true,
          steps: [
            'Open <b>Settings</b> and go to <b>Network & internet</b>.',
            'Click your active connection (<b>Wi-Fi</b> or <b>Ethernet</b>).',
            'Click <b>Hardware properties</b>.',
            'Next to <b>DNS server assignment</b>, click <b>Edit</b>.',
            'Set to <b>Manual</b>, toggle <b>IPv4</b> on, and enter your blockasaurus server IP as the <b>Preferred DNS</b>.',
            'Set <b>DNS over HTTPS</b> to <b>On (manual template)</b> and enter the template below.',
            'Replace <b>&lt;server&gt;</b> with the IP or hostname of your blockasaurus server.',
            'Click <b>Save</b>.',
          ],
          fields: [{ label: 'DNS over HTTPS template', value: fallbackUrl }],
        })
      }
      tabs.push({ id: 'windows', label: 'Windows', sections })
    }

    // Android — most common mobile OS
    if (info.hasTls && fqdn) {
      tabs.push({
        id: 'android', label: 'Android', sections: [{
          title: 'Private DNS',
          subtitle: 'Android 9 or higher',
          recommended: true,
          steps: [
            'Open <b>Settings</b> and tap <b>Network & internet</b>.',
            'Tap <b>Private DNS</b>.',
            'Select <b>Private DNS provider hostname</b>.',
            'Enter the hostname below and tap <b>Save</b>.',
          ],
          fields: [{ label: 'Hostname', value: fqdn }],
        }],
      })
    }

    // iOS
    if (fqdn && (info.hasTls || info.hasHttp)) {
      tabs.push({
        id: 'ios', label: 'iOS', sections: [{
          title: 'Configuration Profile',
          subtitle: 'iOS 14 or higher',
          recommended: true,
          steps: [
            'Visit <b>dns.notjakob.com/tool.html</b> on your device to generate a DNS profile.',
            'Set the server URL to the value below and download the <b>.mobileconfig</b> file.',
            'Open <b>Settings</b> → <b>General</b> → <b>VPN & Device Management</b>.',
            'Tap the downloaded profile and tap <b>Install</b>.',
          ],
          fields: [{ label: 'Server URL', value: url }],
        }],
      })
    }

    // macOS
    if (fqdn && (info.hasTls || info.hasHttp)) {
      tabs.push({
        id: 'macos', label: 'macOS', sections: [{
          title: 'Configuration Profile',
          subtitle: 'macOS Big Sur or higher',
          recommended: true,
          steps: [
            'Visit <b>dns.notjakob.com/tool.html</b> to generate a DNS profile.',
            'Set the server URL to the value below and download the <b>.mobileconfig</b> file.',
            'Double-click the file to open it.',
            'Open <b>System Settings</b> → <b>Privacy & Security</b> → <b>Profiles</b> and install it.',
          ],
          fields: [{ label: 'Server URL', value: url }],
        }],
      })
    }

    // Browsers — easy cross-platform option
    if (info.hasTls || info.hasHttp) {
      const browserUrl = url || fallbackUrl
      tabs.push({
        id: 'browsers', label: 'Browsers', sections: [
          {
            title: 'Chrome / Edge / Brave',
            recommended: true,
            steps: [
              'Open the browser and go to <b>Settings</b>.',
              'Navigate to <b>Privacy and security</b> → <b>Security</b>.',
              'Enable <b>Use secure DNS</b>.',
              'Select <b>With: Custom</b> and paste the URL below.',
            ],
            fields: [{ label: 'Custom DNS URL', value: browserUrl }],
          },
          {
            title: 'Firefox',
            steps: [
              'Open Firefox and go to <b>Settings</b>.',
              'Navigate to <b>Privacy & Security</b> and scroll to <b>DNS over HTTPS</b>.',
              'Select <b>Max Protection</b> (or <b>Increased Protection</b> for a fallback).',
              'Choose <b>Custom</b> and paste the URL below.',
            ],
            fields: [{ label: 'Custom DNS URL', value: browserUrl }],
          },
        ],
      })
    }

    // Linux — multiple methods
    if (info.hasTls && fqdn) {
      tabs.push({
        id: 'linux', label: 'Linux', sections: [
          {
            title: 'systemd-resolved',
            recommended: true,
            steps: [
              'Edit <b>/etc/systemd/resolved.conf</b> and add or update the <b>[Resolve]</b> section with the block below.',
              'Replace <b>&lt;server-ip&gt;</b> with the IP address of your blockasaurus server.',
              'Run <b>sudo systemctl restart systemd-resolved</b> to apply.',
            ],
            codeBlock: `[Resolve]\nDNS=<server-ip>#${fqdn}\nDNSOverTLS=yes`,
          },
          {
            title: 'Stubby',
            steps: [
              'Add the block below to your <b>stubby.yml</b> configuration.',
              'Replace <b>&lt;server-ip&gt;</b> with the IP address of your blockasaurus server.',
              'Restart Stubby to apply.',
            ],
            codeBlock: `upstream_recursive_servers:\n  - address_data: <server-ip>\n    tls_auth_name: "${fqdn}"`,
          },
        ],
      })
    }

    // Routers — advanced / network-wide
    {
      const sections = []
      if (info.cpeId) {
        sections.push({
          title: 'dnsmasq',
          recommended: true,
          steps: [
            'Add the block below to your <b>dnsmasq.conf</b>.',
            'Replace <b>&lt;server-ip&gt;</b> with the IP address of your blockasaurus server.',
            'The <b>add-cpe-id</b> line tags queries so blockasaurus identifies them as this group.',
            'Restart dnsmasq to apply.',
          ],
          codeBlock: `server=<server-ip>\nadd-cpe-id=${slug}`,
        })
      }
      if (info.hasTls && fqdn) {
        sections.push({
          title: 'Unbound',
          steps: [
            'Add the block below to your <b>unbound.conf</b>.',
            'Replace <b>&lt;server-ip&gt;</b> with the IP address of your blockasaurus server.',
            'Restart Unbound to apply.',
          ],
          codeBlock: `forward-zone:\n  name: "."\n  forward-tls-upstream: yes\n  forward-addr: <server-ip>#${fqdn}`,
        })
      }
      if (sections.length > 0) {
        tabs.push({ id: 'routers', label: 'Routers', sections })
      }
    }

    return tabs
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

      <!-- Setup Guide Card -->
      {#if endpointInfo && selected.slug}
        {@const tabs = getSetupTabs(selected.slug, endpointInfo)}
        {@const currentTab = tabs.find(t => t.id === activeTab) ? activeTab : tabs[0]?.id}
        {#if tabs.length > 0}
          <Card title="Setup Guide">
            <p class="setup-intro">
              Follow the instructions below to set up blockasaurus on your device, browser, or router.
            </p>

            <div class="tab-bar">
              {#each tabs as tab}
                <button class="tab" class:active={tab.id === currentTab}
                  onclick={() => activeTab = tab.id}
                >{tab.label}</button>
              {/each}
            </div>

            {#each tabs as tab}
              {#if tab.id === currentTab}
                <div class="tab-content">
                  {#each tab.sections as section, i}
                    {#if i > 0}
                      <div class="section-divider"><span>OR</span></div>
                    {/if}
                    <div class="section">
                      {#if section.recommended}
                        <span class="badge-recommended">RECOMMENDED</span>
                      {/if}
                      <h3 class="section-title">{section.title}</h3>
                      {#if section.subtitle}
                        <p class="section-subtitle">{section.subtitle}</p>
                      {/if}
                      <ol class="section-steps">
                        {#each section.steps as step}
                          <li>{@html step}</li>
                        {/each}
                      </ol>
                      {#if section.fields}
                        {#each section.fields as field}
                          <div class="config-field">
                            <span class="field-label">{field.label}</span>
                            <div class="field-value-row">
                              <code class="field-value">{field.value}</code>
                              <CopyButton value={field.value} />
                            </div>
                          </div>
                        {/each}
                      {/if}
                      {#if section.codeBlock}
                        <div class="code-block-wrapper">
                          <pre class="code-block">{section.codeBlock}</pre>
                          <div class="code-block-copy">
                            <CopyButton value={section.codeBlock} />
                          </div>
                        </div>
                      {/if}
                    </div>
                  {/each}
                </div>
              {/if}
            {/each}
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

  /* Setup Guide */

  .setup-intro {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    margin-bottom: 1rem;
  }

  .tab-bar {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem;
    border-bottom: 1px solid var(--color-border);
    margin-bottom: 1.25rem;
    padding-bottom: 0.5rem;
  }

  .tab {
    background: none;
    border: 1px solid var(--color-btn-border);
    border-radius: var(--radius);
    color: var(--color-text-muted);
    font-size: var(--text-xs);
    padding: 0.3rem 0.75rem;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .tab:hover {
    color: var(--color-text);
    border-color: var(--color-text-dim);
  }

  .tab.active {
    background: var(--color-primary);
    color: var(--color-primary-fg, #fff);
    border-color: var(--color-primary);
  }

  .tab-content {
    display: flex;
    flex-direction: column;
  }

  .section {
    padding: 0.75rem 0;
  }

  .section-divider {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    color: var(--color-text-dim);
    font-size: var(--text-xs);
    margin: 0.25rem 0;
  }

  .section-divider::before,
  .section-divider::after {
    content: '';
    flex: 1;
    height: 1px;
    background: var(--color-border);
  }

  .badge-recommended {
    display: inline-block;
    font-size: 0.6rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    background: var(--color-primary);
    color: var(--color-primary-fg, #fff);
    padding: 0.1rem 0.4rem;
    border-radius: var(--radius);
    margin-bottom: 0.3rem;
  }

  .section-title {
    font-size: var(--text-base);
    font-weight: 600;
    margin: 0;
  }

  .section-subtitle {
    font-size: var(--text-xs);
    color: var(--color-text-dim);
    margin: 0.1rem 0 0 0;
  }

  .section-steps {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    padding-left: 1.5rem;
    line-height: 1.7;
    margin: 0.5rem 0;
  }

  .config-field {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    margin-top: 0.5rem;
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
    font-size: var(--text-sm);
    background: var(--color-btn-bg);
    border: 1px solid var(--color-btn-border);
    border-radius: var(--radius);
    padding: 0.3rem 0.6rem;
    word-break: break-all;
    flex: 1;
    min-width: 0;
  }

  .code-block-wrapper {
    position: relative;
    margin-top: 0.5rem;
  }

  .code-block {
    font-family: var(--font-mono, monospace);
    font-size: var(--text-xs);
    background: var(--color-btn-bg);
    border: 1px solid var(--color-btn-border);
    border-radius: var(--radius);
    padding: 0.75rem 2.5rem 0.75rem 1rem;
    white-space: pre-wrap;
    overflow-x: auto;
    margin: 0;
  }

  .code-block-copy {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
  }
</style>
