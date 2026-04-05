<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import {
    Box,
    Button,
    Table,
    Modal,
    Label,
    Input,
    Toggle,
    EmptyState,
    Toaster,
    toast,
  } from '@chrissnell/chonky-ui'
  import { upstreamGroups } from '../lib/api.js'
  import { markDirty } from '../lib/dirty.svelte.js'

  // --- Groups list ---
  let groups = $state([])
  let loading = $state(true)

  // Servers keyed by group name
  let serversByGroup = $state({})

  // Create group modal
  let createOpen = $state(false)
  let createName = $state('')

  // Server edit modal
  let serverOpen = $state(false)
  let serverEditId = $state(null)
  let serverGroup = $state('')
  let serverForm = $state({ url: '', enabled: true, position: 0 })

  async function loadGroups() {
    loading = true
    try {
      groups = (await upstreamGroups.list()) ?? []
      const entries = await Promise.all(
        groups.map(async (g) => [g.name, (await upstreamGroups.listServers(g.name)) ?? []]),
      )
      serversByGroup = Object.fromEntries(entries)
    } catch (e) {
      toast(e.message, 'danger')
      groups = []
      serversByGroup = {}
    }
    loading = false
  }

  // --- Group CRUD ---
  function openCreate() {
    createName = ''
    createOpen = true
  }

  async function createGroup() {
    const name = createName.trim()
    if (!name) return
    try {
      await upstreamGroups.put(name)
      createOpen = false
      markDirty()
      await loadGroups()
    } catch (e) {
      toast(e.message, 'danger')
    }
  }

  async function removeGroup(name) {
    if (name === 'default') return
    if (!confirm(`Delete upstream group "${name}" and all its servers?`)) return
    try {
      await upstreamGroups.delete(name)
      markDirty()
      await loadGroups()
    } catch (e) {
      toast(e.message, 'danger')
    }
  }

  // --- Server CRUD ---
  function openNewServer(groupName) {
    serverEditId = null
    serverGroup = groupName
    const existing = serversByGroup[groupName] ?? []
    serverForm = { url: '', enabled: true, position: existing.length }
    serverOpen = true
  }

  function openEditServer(groupName, row) {
    serverEditId = row.id
    serverGroup = groupName
    serverForm = { url: row.url, enabled: row.enabled, position: row.position }
    serverOpen = true
  }

  async function saveServer() {
    try {
      const body = {
        url: serverForm.url.trim(),
        enabled: serverForm.enabled,
        position: Number(serverForm.position) || 0,
      }
      if (serverEditId) {
        await upstreamGroups.updateServer(serverGroup, serverEditId, body)
      } else {
        await upstreamGroups.createServer(serverGroup, body)
      }
      serverOpen = false
      markDirty()
      await loadGroups()
    } catch (e) {
      toast(e.message, 'danger')
    }
  }

  async function removeServer(groupName, id) {
    try {
      await upstreamGroups.deleteServer(groupName, id)
      markDirty()
      await loadGroups()
    } catch (e) {
      toast(e.message, 'danger')
    }
  }

  loadGroups()
</script>

<div class="page">
  <div class="page-header">
    <h1 class="page-title">Upstream Groups</h1>
    <Button size="sm" onclick={openCreate}>Add Group</Button>
  </div>

  {#if loading}
    <Box><EmptyState>Loading...</EmptyState></Box>
  {:else if groups.length === 0}
    <Box>
      <EmptyState>
        <p>no upstream groups</p>
        <Button size="sm" onclick={openCreate}>Add your first group</Button>
      </EmptyState>
    </Box>
  {:else}
    {#each groups as g (g.id)}
      <Box title={g.name}>
        {#snippet actions()}
          <Button size="sm" onclick={() => openNewServer(g.name)}>Add Server</Button>
          <Button
            size="sm"
            variant="danger"
            disabled={g.name === 'default'}
            onclick={() => removeGroup(g.name)}
          >
            Delete Group
          </Button>
        {/snippet}
        {#if (serversByGroup[g.name]?.length ?? 0) === 0}
          <EmptyState>
            <p>no servers in this group</p>
            <Button size="sm" onclick={() => openNewServer(g.name)}>Add server</Button>
          </EmptyState>
        {:else}
          <Table>
            <thead>
              <tr>
                <th>URL</th>
                <th>Position</th>
                <th>Status</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {#each serversByGroup[g.name] as srv (srv.id)}
                <tr>
                  <td><code>{srv.url}</code></td>
                  <td>{srv.position}</td>
                  <td>{srv.enabled ? '✓' : '—'}</td>
                  <td class="row-actions">
                    <Button size="sm" onclick={() => openEditServer(g.name, srv)}>Edit</Button>
                    <Button
                      size="sm"
                      variant="danger"
                      onclick={() => removeServer(g.name, srv.id)}
                    >
                      Delete
                    </Button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </Table>
        {/if}
      </Box>
    {/each}
  {/if}
</div>

<Modal bind:open={createOpen}>
  <Modal.Header><h2>New Upstream Group</h2></Modal.Header>
  <Modal.Body>
    <Label>
      Group name
      <Input bind:value={createName} placeholder="kids, iot, work..." />
    </Label>
  </Modal.Body>
  <Modal.Footer>
    <Button onclick={() => (createOpen = false)}>Cancel</Button>
    <Button variant="primary" onclick={createGroup}>Create</Button>
  </Modal.Footer>
</Modal>

<Modal bind:open={serverOpen}>
  <Modal.Header>
    <h2>{serverEditId ? 'Edit Upstream Server' : 'New Upstream Server'}</h2>
  </Modal.Header>
  <Modal.Body>
    <Label>
      URL
      <Input
        bind:value={serverForm.url}
        placeholder="1.1.1.1  |  tcp-tls:dns.example.com  |  https://dns.google/dns-query"
      />
    </Label>
    <Label>
      Position
      <Input bind:value={serverForm.position} type="number" />
    </Label>
    <Toggle bind:checked={serverForm.enabled} label="Enabled" />
  </Modal.Body>
  <Modal.Footer>
    <Button onclick={() => (serverOpen = false)}>Cancel</Button>
    <Button variant="primary" onclick={saveServer}>Save</Button>
  </Modal.Footer>
</Modal>

<Toaster />

<style>
  .page {
    max-width: 1000px;
  }
  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    margin-bottom: var(--space-6);
  }
  .page-title {
    font-size: var(--text-2xl);
    font-weight: 700;
  }
  .row-actions {
    display: flex;
    gap: var(--space-2);
    justify-content: flex-end;
  }
</style>
