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
    Select,
    Toggle,
    EmptyState,
  } from '@chrissnell/chonky-ui'
  import { customDNS } from '../lib/api.js'
  import { markDirty } from '../lib/dirty.svelte.js'

  let entries = $state([])
  let loading = $state(true)
  let editOpen = $state(false)
  let editId = $state(null)
  let form = $state({ domain: '', record_type: 'A', value: '', ttl: 3600, enabled: true })

  async function load() {
    loading = true
    try { entries = await customDNS.list() ?? [] } catch { entries = [] }
    loading = false
  }

  function openNew() {
    editId = null
    form = { domain: '', record_type: 'A', value: '', ttl: 3600, enabled: true }
    editOpen = true
  }

  function openEdit(row) {
    editId = row.id
    form = { ...row }
    editOpen = true
  }

  async function save() {
    if (editId) {
      await customDNS.update(editId, form)
    } else {
      await customDNS.create(form)
    }
    editOpen = false
    markDirty()
    await load()
  }

  async function remove(id) {
    await customDNS.delete(id)
    markDirty()
    await load()
  }

  load()
</script>

<div class="page">
  <div class="page-header">
    <h1 class="page-title">Custom DNS</h1>
    <Button size="sm" onclick={openNew}>Add Entry</Button>
  </div>

  <Box>
    {#if loading}
      <EmptyState>Loading...</EmptyState>
    {:else if entries.length === 0}
      <EmptyState>
        <p>no custom dns entries</p>
        <Button size="sm" onclick={openNew}>Add your first entry</Button>
      </EmptyState>
    {:else}
      <Table>
        <thead>
          <tr>
            <th>Domain</th>
            <th>Type</th>
            <th>Value</th>
            <th>TTL</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {#each entries as row (row.id)}
            <tr>
              <td>{row.domain}</td>
              <td>{row.record_type}</td>
              <td>{row.value}</td>
              <td>{row.ttl}</td>
              <td>{row.enabled ? '✓' : '—'}</td>
              <td>
                <Button size="sm" onclick={() => openEdit(row)}>Edit</Button>
                <Button size="sm" variant="danger" onclick={() => remove(row.id)}>Delete</Button>
              </td>
            </tr>
          {/each}
        </tbody>
      </Table>
    {/if}
  </Box>
</div>

<Modal bind:open={editOpen}>
  <Modal.Header>
    <h2>{editId ? 'Edit DNS Entry' : 'New DNS Entry'}</h2>
  </Modal.Header>
  <Modal.Body>
    <Label>
      Domain
      <Input bind:value={form.domain} placeholder="myhost.lan" />
    </Label>
    <Label>
      Record Type
      <Select bind:value={form.record_type} options={[
        { value: 'A', label: 'A (IPv4)' },
        { value: 'AAAA', label: 'AAAA (IPv6)' },
        { value: 'CNAME', label: 'CNAME' },
      ]} />
    </Label>
    <Label>
      Value
      <Input bind:value={form.value} placeholder="192.168.1.100" />
    </Label>
    <Label>
      TTL (seconds)
      <Input bind:value={form.ttl} type="number" placeholder="3600" />
    </Label>
    <Toggle bind:checked={form.enabled} label="Enabled" />
  </Modal.Body>
  <Modal.Footer>
    <Button onclick={() => editOpen = false}>Cancel</Button>
    <Button variant="primary" onclick={save}>Save</Button>
  </Modal.Footer>
</Modal>

<style>
  .page { max-width: 1000px; }
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
</style>
