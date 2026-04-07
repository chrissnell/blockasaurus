<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { Box, Button, Table, Modal, Input, Select, Toggle, EmptyState } from '@chrissnell/chonky-ui'
  import { domainEntries, clientGroups } from '../lib/api.js'
  import { markDirty } from '../lib/dirty.svelte.js'

  let entries = $state([])
  let groups = $state([])
  let loading = $state(true)

  // Add form
  let addDomain = $state('')
  let addComment = $state('')
  let addEntryType = $state('exact_deny')
  let addWildcard = $state(false)

  // Edit modal
  let editOpen = $state(false)
  let editId = $state(null)
  let editForm = $state({ domain: '', entry_type: 'exact_deny', comment: '', enabled: true })

  // Group assignment modal
  let assignOpen = $state(false)
  let assignEntry = $state(null)
  let assignChecked = $state({})

  // Sort state
  let sortKey = $state('')
  let sortDir = $state('asc')

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
    { key: 'entry_type', label: 'Type' },
    { key: 'enabled', label: 'Status' },
    { key: 'comment', label: 'Comment' },
    { key: 'group_name', label: 'Client Groups' },
  ]

  const sortedEntries = $derived.by(() => {
    if (!sortKey) return entries
    const arr = [...entries]
    arr.sort((a, b) => {
      const av = a[sortKey] ?? ''
      const bv = b[sortKey] ?? ''
      if (av < bv) return sortDir === 'asc' ? -1 : 1
      if (av > bv) return sortDir === 'asc' ? 1 : -1
      return 0
    })
    return arr
  })

  function handleSort(key) {
    if (sortKey === key) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc'
    } else {
      sortKey = key
      sortDir = 'asc'
    }
  }

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

  function onWildcardToggle() {
    if (addWildcard) {
      const val = addDomain.trim()
      if (val) {
        // Strip glob-style wildcards and dots from edges, then build a proper regex
        const clean = val.replace(/^\*\.?/, '').replace(/\.?\*$/, '')
        const escaped = clean.replace(/\./g, '\\.')
        addDomain = `(\\.|^)${escaped}(\\.|$)`
      }
      if (addEntryType === 'exact_deny') addEntryType = 'regex_deny'
      else if (addEntryType === 'exact_allow') addEntryType = 'regex_allow'
    }
  }

  async function addEntry() {
    if (!addDomain.trim()) return

    try {
      await domainEntries.create({
        domain: addDomain.trim(),
        entry_type: addEntryType,
        comment: addComment,
        enabled: true,
      })
    } catch (err) {
      alert(err.message)
      return
    }

    addDomain = ''
    addComment = ''
    addEntryType = 'exact_deny'
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
  <Box title="Add Entry">
    <div class="add-form">
      <div class="field">
        <label class="label" for="de-add-domain">Domain</label>
        <Input id="de-add-domain" bind:value={addDomain} placeholder="example.com" />
      </div>
      <div class="field">
        <label class="label" for="de-add-comment">Comment</label>
        <Input id="de-add-comment" bind:value={addComment} placeholder="optional" />
      </div>
      <div class="field">
        <label class="label" for="de-add-type">Type</label>
        <Select id="de-add-type" bind:value={addEntryType} options={[
          { value: 'exact_deny', label: 'Exact block' },
          { value: 'regex_deny', label: 'Regex block' },
          { value: 'exact_allow', label: 'Exact allow' },
          { value: 'regex_allow', label: 'Regex allow' },
        ]} />
      </div>
      <div class="add-btn-wrap">
        <Button onclick={addEntry}>Add</Button>
      </div>
    </div>
    <label class="wildcard-check">
      <input type="checkbox" bind:checked={addWildcard} onchange={onWildcardToggle} />
      <span>Add as wildcard</span>
    </label>
  </Box>

  <!-- List -->
  <Box>
    {#if loading}
      <EmptyState>Loading...</EmptyState>
    {:else if entries.length === 0}
      <EmptyState>
        <div>no domain entries</div>
        <p class="empty-hint">Add your first entry above</p>
      </EmptyState>
    {:else}
      <Table class="domain-table">
        <thead>
          <tr>
            {#each columns as col}
              <th
                class:sortable={col.sortable}
                onclick={() => col.sortable && handleSort(col.key)}
              >
                {col.label}
                {#if col.sortable && sortKey === col.key}
                  <span class="sort-arrow">{sortDir === 'asc' ? '▲' : '▼'}</span>
                {/if}
              </th>
            {/each}
            <th></th>
          </tr>
        </thead>
        <tbody>
          {#each sortedEntries as row}
            <tr>
              <td>{row.domain}</td>
              <td>{typeLabels[row.entry_type] ?? row.entry_type}</td>
              <td>{row.enabled ? '✓' : '—'}</td>
              <td>{row.comment ?? ''}</td>
              <td>{groupCount(row.group_name)}</td>
              <td class="actions">
                <Button size="sm" onclick={() => toggleEnabled(row)}>{row.enabled ? 'Disable' : 'Enable'}</Button>
                <Button size="sm" onclick={() => openAssign(row)}>Groups</Button>
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

<!-- Edit Modal -->
<Modal.Root bind:open={editOpen}>
  <Modal.Header>Edit Domain Entry</Modal.Header>
  <Modal.Body>
    <div class="field">
      <label class="label" for="de-edit-domain">Domain</label>
      <Input id="de-edit-domain" bind:value={editForm.domain} />
    </div>
    <div class="field">
      <label class="label" for="de-edit-type">Type</label>
      <Select id="de-edit-type" bind:value={editForm.entry_type} options={[
        { value: 'exact_deny', label: 'Exact block' },
        { value: 'regex_deny', label: 'Regex block' },
        { value: 'exact_allow', label: 'Exact allow' },
        { value: 'regex_allow', label: 'Regex allow' },
      ]} />
    </div>
    <div class="field">
      <label class="label" for="de-edit-comment">Comment</label>
      <Input id="de-edit-comment" bind:value={editForm.comment} />
    </div>
    <div class="field">
      <Toggle label="Enabled" bind:checked={editForm.enabled} />
    </div>
  </Modal.Body>
  <Modal.Footer>
    <Button onclick={() => editOpen = false}>Cancel</Button>
    <Button onclick={saveEdit}>Save</Button>
  </Modal.Footer>
</Modal.Root>

<!-- Group Assignment Modal -->
<Modal.Root bind:open={assignOpen}>
  <Modal.Header>Assign to Client Groups</Modal.Header>
  <Modal.Body>
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
  </Modal.Body>
  <Modal.Footer>
    <Button onclick={() => assignOpen = false}>Cancel</Button>
    {#if groups.length > 0}
      <Button onclick={saveAssignments}>Save</Button>
    {/if}
  </Modal.Footer>
</Modal.Root>

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

  .add-form :global(input[type="text"]) {
    margin-bottom: 0;
  }

  .add-btn-wrap {
    margin-bottom: 1rem;
  }

  .field {
    margin-bottom: 1rem;
  }

  .label {
    display: block;
    font-size: 0.85rem;
    color: var(--color-text-muted);
    margin-bottom: 0.25rem;
  }

  .wildcard-check {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin-top: 0.25rem;
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

  :global(.domain-table) {
    width: 100%;
    font-size: var(--text-sm);
    border-collapse: collapse;
  }

  :global(.domain-table th) {
    text-align: left;
    color: var(--color-text-dim);
    font-weight: 400;
    padding: 0.25rem 0.75rem 0.25rem 0;
    border-bottom: 1px solid var(--color-border);
  }

  :global(.domain-table th.sortable) {
    cursor: pointer;
    user-select: none;
  }

  :global(.domain-table th.sortable:hover) {
    color: var(--color-text);
  }

  :global(.domain-table .sort-arrow) {
    font-size: 0.6rem;
    margin-left: 0.25rem;
  }

  :global(.domain-table td) {
    padding: 0.35rem 0.75rem 0.35rem 0;
    border-bottom: 1px dotted var(--color-border-subtle);
  }

  :global(.domain-table td.actions) {
    text-align: right;
    white-space: nowrap;
  }
</style>
