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
    const serverIpNote = 'Replace <b>&lt;server-ip&gt;</b> with the IP address of your blockasaurus server.'

    // Android
    {
      const sections = []
      if (info.hasTls && fqdn) {
        sections.push({
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
        })
      }
      sections.push({
        title: 'Manual DNS',
        steps: [
          'Open <b>Settings</b> and tap <b>Wi-Fi</b>.',
          'Long-press your connected network and tap <b>Modify network</b>.',
          'Tap <b>Advanced options</b> and set <b>IP settings</b> to <b>Static</b>.',
          'Set <b>DNS 1</b> to your blockasaurus server\'s IP address.',
          'Tap <b>Save</b>.',
        ],
      })
      tabs.push({ id: 'android', label: 'Android', sections })
    }

    // iOS
    {
      const sections = []
      const profileUrl = `/api/mobileconfig/${slug}`
      sections.push({
        title: 'Configuration Profile',
        subtitle: 'iOS 14 or higher',
        recommended: true,
        steps: [
          'Download the profile using the button below.',
          'Open <b>Settings</b> → <b>General</b> → <b>VPN & Device Management</b>.',
          'Tap the downloaded profile and tap <b>Install</b>.',
          ...(info.hasSelfSignedCert ? ['If you included the CA certificate: go to <b>Settings</b> → <b>General</b> → <b>About</b> → <b>Certificate Trust Settings</b> and enable trust for the Blockasaurus CA.'] : []),
        ],
        download: { url: profileUrl, label: 'Download Configuration Profile' },
        certCheckbox: info.hasSelfSignedCert,
      })
      sections.push({
        title: 'Manual DNS',
        steps: [
          'Open <b>Settings</b> and tap <b>Wi-Fi</b>.',
          'Tap the <b>ⓘ</b> button next to your connected network.',
          'Tap <b>Configure DNS</b> and select <b>Manual</b>.',
          'Remove existing DNS servers and add your blockasaurus server\'s IP address.',
          'Tap <b>Save</b>.',
        ],
      })
      tabs.push({ id: 'ios', label: 'iOS', sections })
    }

    // Windows
    {
      const sections = []
      if (url) {
        sections.push({
          title: 'Secure DNS',
          subtitle: 'Windows 11',
          recommended: true,
          steps: [
            'Open <b>Settings</b> and go to <b>Network & internet</b>.',
            'Click your active connection (<b>Wi-Fi</b> or <b>Ethernet</b>).',
            'Click <b>Hardware properties</b>.',
            'Next to <b>DNS server assignment</b>, click <b>Edit</b>.',
            'Set to <b>Manual</b> and toggle <b>IPv4</b> on.',
            'Enter your blockasaurus server\'s IP as the <b>Preferred DNS</b>.',
            'Set <b>DNS over HTTPS</b> to <b>On (manual template)</b> and enter the template below.',
            'Click <b>Save</b>.',
          ],
          fields: [{ label: 'DNS over HTTPS template', value: url }],
        })
      }
      sections.push({
        title: 'Manual DNS',
        subtitle: 'All Windows versions',
        steps: [
          'Open <b>Control Panel</b> → <b>Network and Internet</b> → <b>Network and Sharing Center</b>.',
          'Click <b>Change adapter settings</b>.',
          'Right-click your active connection and select <b>Properties</b>.',
          'Select <b>Internet Protocol Version 4 (TCP/IPv4)</b> and click <b>Properties</b>.',
          'Select <b>Use the following DNS server addresses</b>.',
          'Enter your blockasaurus server\'s IP as the <b>Preferred DNS server</b>.',
          'Click <b>OK</b>, then <b>Close</b>.',
        ],
      })
      tabs.push({ id: 'windows', label: 'Windows', sections })
    }

    // macOS
    {
      const sections = []
      const profileUrl = `/api/mobileconfig/${slug}`
      sections.push({
        title: 'Configuration Profile',
        subtitle: 'macOS Big Sur or higher',
        recommended: true,
        steps: [
          'Download the profile using the button below.',
          'Double-click the downloaded file to open it.',
          'Open <b>System Settings</b> → <b>Privacy & Security</b> → <b>Profiles</b> and install it.',
          ...(info.hasSelfSignedCert ? ['If you included the CA certificate: open <b>Keychain Access</b>, find the Blockasaurus CA cert, and set it to <b>Always Trust</b>.'] : []),
        ],
        download: { url: profileUrl, label: 'Download Configuration Profile' },
        certCheckbox: info.hasSelfSignedCert,
      })
      sections.push({
        title: 'Manual DNS',
        steps: [
          'Open <b>System Settings</b> and click <b>Network</b>.',
          'Select your active connection and click <b>Details</b>.',
          'Go to the <b>DNS</b> tab.',
          'Remove existing DNS servers and add your blockasaurus server\'s IP address.',
          'Click <b>OK</b>.',
        ],
      })
      tabs.push({ id: 'macos', label: 'macOS', sections })
    }

    // Linux
    {
      const sections = []
      if (info.hasTls && fqdn) {
        sections.push({
          title: 'systemd-resolved',
          recommended: true,
          steps: [
            'Edit <b>/etc/systemd/resolved.conf</b> and add the block below.',
            serverIpNote,
            'Run <b>sudo systemctl restart systemd-resolved</b> to apply.',
          ],
          codeBlock: `[Resolve]\nDNS=<server-ip>#${fqdn}\nDNSOverTLS=yes`,
        })
      }
      sections.push({
        title: 'resolv.conf',
        ...(!info.hasTls || !fqdn ? { recommended: true } : {}),
        steps: [
          'Edit <b>/etc/resolv.conf</b> (or use your distribution\'s network manager).',
          serverIpNote,
        ],
        codeBlock: 'nameserver <server-ip>',
      })
      if (info.hasTls && fqdn) {
        sections.push({
          title: 'Stubby',
          steps: [
            'Add the block below to your <b>stubby.yml</b>.',
            serverIpNote,
            'Restart Stubby to apply.',
          ],
          codeBlock: `round_robin_upstreams: 1\nupstream_recursive_servers:\n  - address_data: <server-ip>\n    tls_auth_name: "${fqdn}"`,
        })
        sections.push({
          title: 'Unbound',
          steps: [
            'Add the block below to your <b>unbound.conf</b>.',
            serverIpNote,
            'Restart Unbound to apply.',
          ],
          codeBlock: `forward-zone:\n  name: "."\n  forward-tls-upstream: yes\n  forward-addr: <server-ip>#${fqdn}`,
        })
      }
      if (url) {
        sections.push({
          title: 'cloudflared',
          steps: [
            'Add the block below to <b>/usr/local/etc/cloudflared/config.yml</b>.',
            'Restart cloudflared to apply.',
          ],
          codeBlock: `proxy-dns: true\nproxy-dns-upstream:\n  - ${url}`,
        })
      }
      if (info.cpeId) {
        sections.push({
          title: 'dnsmasq',
          steps: [
            'Add the block below to your <b>dnsmasq.conf</b>.',
            serverIpNote,
            'The <b>add-cpe-id</b> line tags queries so blockasaurus identifies them as this group.',
            'Restart dnsmasq to apply.',
          ],
          codeBlock: `no-resolv\nbogus-priv\nstrict-order\nserver=<server-ip>\nadd-cpe-id=${slug}`,
        })
      }
      tabs.push({ id: 'linux', label: 'Linux', sections })
    }

    // ChromeOS
    {
      const sections = []
      if (url) {
        sections.push({
          title: 'Secure DNS',
          recommended: true,
          steps: [
            'Open the <b>Settings</b> app.',
            'Go to <b>Security and Privacy</b>.',
            'Enable <b>Use secure DNS</b>.',
            'Select <b>With: Custom</b> and enter the URL below.',
          ],
          fields: [{ label: 'DNS over HTTPS URL', value: url }],
        })
      }
      sections.push({
        title: 'Manual DNS',
        steps: [
          'Open <b>Settings</b> and go to <b>Network</b>.',
          'Select your active connection.',
          'Expand the <b>Network</b> section and set <b>Name servers</b> to <b>Custom name servers</b>.',
          'Enter your blockasaurus server\'s IP address.',
          'Close Settings to apply.',
        ],
      })
      tabs.push({ id: 'chromeos', label: 'ChromeOS', sections })
    }

    // Browsers
    if (url) {
      tabs.push({
        id: 'browsers', label: 'Browsers', sections: [
          {
            title: 'Chrome / Edge / Brave',
            recommended: true,
            steps: [
              'Open the browser and go to <b>Settings</b>.',
              'Navigate to <b>Privacy and security</b> → <b>Security</b>.',
              'Enable <b>Use secure DNS</b>.',
              'Select <b>With: Custom</b> and enter the URL below.',
            ],
            fields: [{ label: 'DNS over HTTPS URL', value: url }],
          },
          {
            title: 'Firefox',
            steps: [
              'Open Firefox and go to <b>Settings</b>.',
              'Navigate to <b>Privacy & Security</b> and scroll to <b>DNS over HTTPS</b>.',
              'Select <b>Max Protection</b> (or <b>Increased Protection</b> for a fallback).',
              'Choose <b>Custom</b> and enter the URL below.',
            ],
            fields: [{ label: 'DNS over HTTPS URL', value: url }],
          },
        ],
      })
    }

    // Routers
    {
      const sections = []
      sections.push({
        title: 'Plain DNS',
        recommended: true,
        steps: [
          'Open your router\'s admin interface (usually <b>http://192.168.1.1</b> or <b>http://192.168.0.1</b>).',
          'Locate the <b>DNS settings</b> (often under WAN, Internet, or DHCP settings).',
          'Set the primary DNS server to your blockasaurus server\'s IP address.',
          'Save and apply changes.',
        ],
      })
      if (info.cpeId) {
        sections.push({
          title: 'dnsmasq',
          steps: [
            'Add the block below to your <b>dnsmasq.conf</b>.',
            serverIpNote,
            'The <b>add-cpe-id</b> line tags queries so blockasaurus identifies them as this group.',
            'Restart dnsmasq to apply.',
          ],
          codeBlock: `no-resolv\nbogus-priv\nstrict-order\nserver=<server-ip>\nadd-cpe-id=${slug}`,
        })
      }
      if (info.hasTls && fqdn) {
        sections.push({
          title: 'Unbound',
          steps: [
            'Add the block below to your <b>unbound.conf</b>.',
            serverIpNote,
            'Restart Unbound to apply.',
          ],
          codeBlock: `forward-zone:\n  name: "."\n  forward-tls-upstream: yes\n  forward-addr: <server-ip>#${fqdn}`,
        })
        sections.push({
          title: 'pfSense',
          steps: [
            'Go to <b>Services</b> → <b>DNS Resolver</b>.',
            'On the <b>General Settings</b> tab, scroll to the <b>Custom Options</b> box.',
            'Enter the block below.',
          ],
          codeBlock: `server:\nforward-zone:\n  name: "."\n  forward-tls-upstream: yes\n  forward-addr: <server-ip>#${fqdn}`,
        })
      }
      if (url) {
        sections.push({
          title: 'MikroTik',
          steps: [
            'Open a terminal to your MikroTik router and run the commands below.',
            serverIpNote,
          ],
          codeBlock: `/ip dns set servers=""\n/ip dns static add name=${domain} address=<server-ip> type=A\n/ip dns set use-doh-server="${url}" verify-doh-cert=yes`,
        })
      }
      tabs.push({ id: 'routers', label: 'Routers', sections })
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

      <!-- Blocklists Card -->
      <Card title="Blocklists">
        <p class="blocklist-hint">
          Go to the <a href="/blocklists" class="inline-link">blocklists</a> tab and click <strong>Groups</strong> to add individual blocklists to your client group(s).
        </p>
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
                      {#if section.download}
                        <div class="download-section">
                          {#if section.certCheckbox}
                            <label class="cert-checkbox">
                              <input type="checkbox" bind:checked={section._includeCert} />
                              Include CA certificate in profile
                            </label>
                          {/if}
                          <a
                            class="download-btn"
                            href={section._includeCert ? section.download.url + '?includeCert=1' : section.download.url}
                            download
                          >{section.download.label}</a>
                        </div>
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
  }

  .detail-header {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
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

  .blocklist-hint {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
    line-height: 1.5;
  }

  .inline-link {
    color: var(--color-accent);
    text-decoration: underline;
    text-underline-offset: 2px;
  }

  .inline-link:hover {
    color: var(--color-text);
  }

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

  .download-section {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin: 0.75rem 0;
  }

  .download-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0.5rem 1rem;
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--color-primary-fg, #fff);
    background: var(--color-primary);
    border: 1px solid var(--color-primary);
    border-radius: var(--radius);
    text-decoration: none;
    cursor: pointer;
    transition: opacity 0.15s ease;
    width: fit-content;
  }

  .download-btn:hover {
    opacity: 0.85;
  }

  .cert-checkbox {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    cursor: pointer;
  }

  .cert-checkbox input[type="checkbox"] {
    accent-color: var(--color-primary);
  }
</style>
