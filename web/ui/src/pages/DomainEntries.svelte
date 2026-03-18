<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import Card from '../components/Card.svelte'
  import Button from '../components/Button.svelte'
  import DataTable from '../components/DataTable.svelte'
  import Modal from '../components/Modal.svelte'
  import FormField from '../components/FormField.svelte'
  import TextInput from '../components/TextInput.svelte'
  import Select from '../components/Select.svelte'
  import Toggle from '../components/Toggle.svelte'
  import EmptyState from '../components/EmptyState.svelte'
  import { domainEntries, clientGroups } from '../lib/api.js'
  import { markDirty } from '../lib/dirty.svelte.js'

  let entries = $state([])
  let groups = $state([])
  let loading = $state(true)

  // Add form
  let addDomain = $state('')
  let addComment = $state('')
  let addListType = $state('deny')
  let addWildcard = $state(false)

  // Edit modal
  let editOpen = $state(false)
  let editId = $state(null)
  let editForm = $state({ domain: '', entry_type: 'exact_deny', comment: '', enabled: true })

  // Group assignment modal
  let assignOpen = $state(false)
  let assignEntry = $state(null)
  let assignChecked = $state({})

  const typeLabels = {
    exact_deny: 'Exact block',
    regex_deny: 'Regex block',
    exact_allow: 'Exact allow',
    regex_allow: 'Regex allow',
  }

  // Count how many client groups reference a given group_name
  function groupCount(groupName) {
    return groups.filter(g => g.groups?.includes(groupName)).length
  }

  const columns = [
    { key: 'domain', label: 'Domain/RegEx', sortable: true },
    { key: 'entry_type', label: 'Type', render: (r) => typeLabels[r.entry_type] ?? r.entry_type },
    { key: 'enabled', label: 'Status', render: (r) => r.enabled ? '✓' : '—' },
    { key: 'comment', label: 'Comment' },
    { key: 'group_name', label: 'Client Groups', render: (r) => `${groupCount(r.group_name)}` },
  ]

  async function load() {
    loading = true
    try {
      ;[entries, groups] = await Promise.all([
        domainEntries.list().then(r => r ?? []),
        clientGroups.list().then(r => r ?? []),
      ])
    } catch {
      entries = []
      groups = []
    }
    loading = false
  }

  async function addEntry() {
    if (!addDomain.trim()) return

    let domain = addDomain.trim()
    let entryType

    if (addWildcard) {
      // Convert to regex like Pi-hole: example.com -> (\.|^)example\.com$
      const escaped = domain.replace(/\./g, '\\.')
      domain = `(\\.|^)${escaped}$`
      entryType = addListType === 'deny' ? 'regex_deny' : 'regex_allow'
    } else {
      entryType = addListType === 'deny' ? 'exact_deny' : 'exact_allow'
    }

    await domainEntries.create({
      domain,
      entry_type: entryType,
      comment: addComment,
      enabled: true,
    })

    addDomain = ''
    addComment = ''
    addWildcard = false
    markDirty()
    await load()
  }

  async function toggleEnabled(entry) {
    await domainEntries.update(entry.id, {
      domain: entry.domain,
      entry_type: entry.entry_type,
      comment: entry.comment,
      enabled: !entry.enabled,
    })
    markDirty()
    await load()
  }

  function openEdit(row) {
    editId = row.id
    editForm = {
      domain: row.domain,
      entry_type: row.entry_type,
      comment: row.comment ?? '',
      enabled: row.enabled,
    }
    editOpen = true
  }

  async function saveEdit() {
    await domainEntries.update(editId, editForm)
    editOpen = false
    markDirty()
    await load()
  }

  async function remove(id) {
    await domainEntries.delete(id)
    markDirty()
    await load()
  }

  // --- Group assignment modal (same pattern as Blocklists page) ---
  function openAssign(row) {
    assignEntry = row
    assignChecked = {}
    for (const g of groups) {
      assignChecked[g.name] = g.groups?.includes(row.group_name) ?? false
    }
    assignOpen = true
  }

  function selectAll() {
    for (const g of groups) assignChecked[g.name] = true
  }

  function selectNone() {
    for (const g of groups) assignChecked[g.name] = false
  }

  async function saveAssignments() {
    const groupName = assignEntry.group_name
    const updates = []

    for (const g of groups) {
      const hasIt = g.groups?.includes(groupName) ?? false
      const wantIt = assignChecked[g.name] ?? false

      if (hasIt === wantIt) continue

      let newGroups
      if (wantIt) {
        newGroups = [...(g.groups ?? []), groupName]
      } else {
        newGroups = (g.groups ?? []).filter(n => n !== groupName)
      }
      updates.push(clientGroups.put(g.name, { clients: g.clients, groups: newGroups }))
    }

    if (updates.length > 0) {
      await Promise.all(updates)
      markDirty()
      await load()
    }
    assignOpen = false
  }

  load()
</script>

<div class="page">
  <h1 class="page-title">Domains</h1>

  <!-- Add form -->
  <Card title="Add Entry">
    <div class="add-form">
      <FormField label="Domain">
        <TextInput bind:value={addDomain} placeholder="example.com" />
      </FormField>
      <FormField label="Comment">
        <TextInput bind:value={addComment} placeholder="optional" />
      </FormField>
      <FormField label="Action">
        <Select bind:value={addListType} options={[
          { value: 'deny', label: 'Block' },
          { value: 'allow', label: 'Allow' },
        ]} />
      </FormField>
      <div class="add-btn-wrap">
        <Button onclick={addEntry}>Add</Button>
      </div>
    </div>
    <div class="add-form-row2">
      <label class="wildcard-check">
        <input type="checkbox" bind:checked={addWildcard} />
        <span>Add as wildcard</span>
      </label>
    </div>
  </Card>

  <!-- List -->
  <Card>
    {#if loading}
      <EmptyState message="Loading..." />
    {:else if entries.length === 0}
      <EmptyState message="no domain entries">
        <p class="empty-hint">Add your first entry above</p>
      </EmptyState>
    {:else}
      <DataTable {columns} rows={entries}>
        {#snippet rowActions(row)}
          <Button size="sm" onclick={() => toggleEnabled(row)}>{row.enabled ? 'Disable' : 'Enable'}</Button>
          <Button size="sm" onclick={() => openAssign(row)}>Groups</Button>
          <Button size="sm" onclick={() => openEdit(row)}>Edit</Button>
          <Button size="sm" variant="danger" onclick={() => remove(row.id)}>Delete</Button>
        {/snippet}
      </DataTable>
    {/if}
  </Card>
</div>

<!-- Edit Modal -->
<Modal bind:open={editOpen} title="Edit Domain Entry">
  <FormField label="Domain">
    <TextInput bind:value={editForm.domain} />
  </FormField>
  <FormField label="Type">
    <Select bind:value={editForm.entry_type} options={[
      { value: 'exact_deny', label: 'Exact block' },
      { value: 'regex_deny', label: 'Regex block' },
      { value: 'exact_allow', label: 'Exact allow' },
      { value: 'regex_allow', label: 'Regex allow' },
    ]} />
  </FormField>
  <FormField label="Comment">
    <TextInput bind:value={editForm.comment} />
  </FormField>
  <FormField label="Enabled">
    <Toggle bind:checked={editForm.enabled} />
  </FormField>
  {#snippet actions()}
    <Button onclick={() => editOpen = false}>Cancel</Button>
    <Button onclick={saveEdit}>Save</Button>
  {/snippet}
</Modal>

<!-- Group Assignment Modal -->
<Modal bind:open={assignOpen} title="Assign to Client Groups">
  {#if groups.length === 0}
    <p class="empty-hint">No client groups exist yet. Create one from the Client Groups page.</p>
  {:else}
    <div class="assign-actions">
      <Button size="sm" onclick={selectAll}>All</Button>
      <Button size="sm" onclick={selectNone}>None</Button>
    </div>
    <div class="assign-list">
      {#each groups as g}
        <label class="assign-row">
          <input type="checkbox" bind:checked={assignChecked[g.name]} />
          <span>{g.name}</span>
        </label>
      {/each}
    </div>
  {/if}
  {#snippet actions()}
    <Button onclick={() => assignOpen = false}>Cancel</Button>
    {#if groups.length > 0}
      <Button onclick={saveAssignments}>Save</Button>
    {/if}
  {/snippet}
</Modal>

<style>
  .page { max-width: 1000px; }
  .page-title {
    font-size: var(--text-2xl);
    font-weight: 700;
    margin-bottom: var(--space-6);
  }

  .add-form {
    display: flex;
    align-items: flex-end;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .add-btn-wrap {
    align-self: flex-end;
  }

  .add-form-row2 {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-top: 0.35rem;
  }

  .wildcard-check {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: var(--text-sm);
    cursor: pointer;
  }

  .wildcard-check input[type="checkbox"] {
    accent-color: var(--color-accent, currentColor);
  }

  .empty-hint {
    color: var(--color-text-dim);
    font-size: var(--text-sm);
    font-style: italic;
  }

  .assign-actions {
    display: flex;
    gap: 0.4rem;
    margin-bottom: 0.75rem;
  }

  .assign-list {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  .assign-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.3rem 0.2rem;
    font-size: var(--text-sm);
    cursor: pointer;
    border-radius: var(--radius);
  }

  .assign-row:hover {
    background: var(--color-btn-bg);
  }

  .assign-row input[type="checkbox"] {
    accent-color: var(--color-accent, currentColor);
  }
</style>
