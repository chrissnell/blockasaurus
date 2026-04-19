<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { Box, Button, Input, Modal } from '@chrissnell/chonky-ui'
  import { authAPI } from '../lib/api.js'
  import { getUser } from '../lib/auth.svelte.js'

  let users = $state([])
  let loading = $state(true)
  let error = $state('')

  // Add user dialog
  let showAdd = $state(false)
  let newUsername = $state('')
  let newPassword = $state('')
  let newRole = $state('viewer')
  let addError = $state('')
  let addSubmitting = $state(false)

  // Delete confirm dialog
  let showDelete = $state(false)
  let deleteTarget = $state(null)
  let deleteError = $state('')
  let deleteSubmitting = $state(false)

  const currentUser = $derived(getUser())
  const adminCount = $derived(users.filter((u) => u.role === 'admin').length)

  const MIN_PW = 12

  async function loadUsers(signal) {
    loading = true
    error = ''
    try {
      const resp = await authAPI.listUsers(signal)
      if (signal?.aborted) return
      users = (resp && resp.data) || resp || []
    } catch (e) {
      // Abort on unmount is expected; swallow silently.
      if (e?.name === 'AbortError') return
      // 401 already redirects; don't overwrite with a misleading message.
      if (e?.message !== 'unauthorized') error = e?.message || 'failed to load users'
    } finally {
      if (!signal?.aborted) loading = false
    }
  }

  async function handleAdd(e) {
    e.preventDefault()
    addError = ''
    if (newPassword.length < MIN_PW) {
      addError = `Password must be at least ${MIN_PW} characters`
      return
    }
    addSubmitting = true
    try {
      await authAPI.createUser({
        username: newUsername,
        password: newPassword,
        role: newRole,
      })
      showAdd = false
      newUsername = ''
      newPassword = ''
      newRole = 'viewer'
      await loadUsers()
    } catch (err) {
      if (err?.message === 'unauthorized') return
      addError = err?.message || 'failed to create user'
    }
    addSubmitting = false
  }

  async function handleDelete() {
    if (!deleteTarget) return
    deleteError = ''
    deleteSubmitting = true
    try {
      await authAPI.deleteUser(deleteTarget.id)
      showDelete = false
      deleteTarget = null
      await loadUsers()
    } catch (err) {
      if (err?.message === 'unauthorized') {
        deleteSubmitting = false
        return
      }
      deleteError = err?.message || 'failed to delete user'
    }
    deleteSubmitting = false
  }

  function confirmDelete(u) {
    deleteTarget = u
    deleteError = ''
    showDelete = true
  }

  // Client-side guard. Server remains authoritative — any mismatch surfaces
  // as a 409 which we render via deleteError.
  function deleteGuard(row) {
    if (!row || !currentUser) return { disabled: true, reason: 'loading' }
    if (row.id === currentUser.id) {
      return { disabled: true, reason: 'You cannot delete your own account.' }
    }
    if (row.role === 'admin' && adminCount <= 1) {
      return { disabled: true, reason: 'At least one admin must exist.' }
    }
    return { disabled: false, reason: '' }
  }

  function formatCreated(ts) {
    if (!ts) return ''
    const d = new Date(ts)
    if (Number.isNaN(d.getTime())) return ''
    return d.toLocaleString()
  }

  $effect(() => {
    const controller = new AbortController()
    loadUsers(controller.signal)
    return () => controller.abort()
  })
</script>

<div class="page">
  <div class="page-header">
    <h2>Users</h2>
    <Button onclick={() => { showAdd = true; addError = '' }}>Add User</Button>
  </div>

  {#if error}
    <div class="error" role="alert">{error}</div>
  {/if}

  <p class="role-note" id="role-note">
    Roles cannot be changed. To change a role, delete the user and recreate them.
  </p>

  {#if loading}
    <p class="dim">Loading...</p>
  {:else}
    <Box>
        <table class="users-table" aria-describedby="role-note">
          <thead>
            <tr>
              <th>Username</th>
              <th>Role</th>
              <th>Created</th>
              <th class="actions-col"><span class="sr-only">Actions</span></th>
            </tr>
          </thead>
          <tbody>
            {#each users as u (u.id)}
              {@const guard = deleteGuard(u)}
              <tr>
                <td>{u.username}</td>
                <td>
                  <span
                    class="badge"
                    class:admin={u.role === 'admin'}
                    title="Role is immutable. Delete and recreate to change."
                  >
                    {u.role}
                  </span>
                </td>
                <td class="dim">{formatCreated(u.created_at)}</td>
                <td class="actions">
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={guard.disabled}
                    title={guard.reason || 'Delete user'}
                    onclick={() => confirmDelete(u)}
                  >
                    Delete
                  </Button>
                </td>
              </tr>
            {/each}
            {#if users.length === 0}
              <tr><td colspan="4" class="dim">No users.</td></tr>
            {/if}
          </tbody>
        </table>
    </Box>
  {/if}
</div>

<!-- Add User Modal -->
<Modal bind:open={showAdd}>
  <Modal.Header>
    <h2>Add User</h2>
  </Modal.Header>
  <form onsubmit={handleAdd}>
    <Modal.Body>
      <div
        class="alert"
        role="alert"
        aria-live="polite"
        aria-atomic="true"
      >
        {#if addError}{addError}{/if}
      </div>

      <div class="field">
        <label for="new-username">Username</label>
        <Input
          id="new-username"
          type="text"
          bind:value={newUsername}
          autocomplete="off"
          autocapitalize="off"
          spellcheck="false"
          required
        />
      </div>

      <div class="field">
        <label for="new-password">Password</label>
        <Input
          id="new-password"
          type="password"
          bind:value={newPassword}
          autocomplete="new-password"
          aria-describedby="new-password-hint"
          required
        />
        <span id="new-password-hint" class="hint">Minimum {MIN_PW} characters</span>
      </div>

      <div class="field">
        <label for="new-role">Role</label>
        <select id="new-role" bind:value={newRole}>
          <option value="viewer">Viewer (read-only)</option>
          <option value="admin">Admin (full access)</option>
        </select>
        <span class="hint">Roles cannot be changed later.</span>
      </div>
    </Modal.Body>
    <Modal.Footer>
      <Button variant="ghost" type="button" onclick={() => { showAdd = false }}>
        Cancel
      </Button>
      <Button
        type="submit"
        disabled={addSubmitting || newPassword.length < MIN_PW || !newUsername.trim()}
        aria-busy={addSubmitting ? 'true' : 'false'}
      >
        {addSubmitting ? 'Creating...' : 'Create User'}
      </Button>
    </Modal.Footer>
  </form>
</Modal>

<!-- Delete Confirm Modal -->
<Modal bind:open={showDelete}>
  <Modal.Header>
    <h2>Delete User</h2>
  </Modal.Header>
  <Modal.Body>
    {#if deleteError}
      <div class="alert" role="alert" aria-live="polite">{deleteError}</div>
    {/if}
    <p>
      Delete user <strong>{deleteTarget?.username}</strong>?
      This will immediately log them out and cannot be undone.
    </p>
  </Modal.Body>
  <Modal.Footer>
    <Button variant="ghost" onclick={() => { showDelete = false }}>Cancel</Button>
    <Button
      variant="destructive"
      onclick={handleDelete}
      disabled={deleteSubmitting}
      aria-busy={deleteSubmitting ? 'true' : 'false'}
    >
      {deleteSubmitting ? 'Deleting...' : 'Delete'}
    </Button>
  </Modal.Footer>
</Modal>

<style>
  .page { max-width: 900px; }

  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    margin-bottom: 1rem;
  }

  h2 {
    font-size: var(--text-xl);
    font-weight: 600;
  }

  .role-note {
    font-size: var(--text-sm);
    color: var(--color-text-dim);
    margin-bottom: 0.75rem;
  }

  .users-table {
    width: 100%;
    border-collapse: collapse;
  }

  .users-table th, .users-table td {
    padding: 0.5rem 0.75rem;
    text-align: left;
    border-bottom: 1px solid var(--color-border);
  }

  .users-table th {
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--color-text-dim);
  }

  .actions-col {
    width: 1%;
    white-space: nowrap;
  }

  .badge {
    display: inline-block;
    padding: 0.125rem 0.5rem;
    border-radius: var(--radius);
    font-size: var(--text-xs);
    font-weight: 500;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    cursor: help;
  }

  .badge.admin {
    background: var(--color-primary-subtle, #eff6ff);
    border-color: var(--color-primary, #3b82f6);
    color: var(--color-primary, #3b82f6);
  }

  .actions {
    text-align: right;
  }

  .dim {
    color: var(--color-text-dim);
    font-size: var(--text-sm);
  }

  .error, .alert:not(:empty) {
    padding: 0.5rem 0.75rem;
    background: var(--color-danger-subtle, #fef2f2);
    color: var(--color-danger, #dc2626);
    border-radius: var(--radius);
    font-size: var(--text-sm);
    margin-bottom: 1rem;
    text-align: left;
  }

  .alert:empty { display: none; }

  .field {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    margin-bottom: 0.75rem;
  }

  .field label {
    font-size: var(--text-sm);
    font-weight: 500;
  }

  .field select {
    padding: 0.5rem;
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    background: var(--color-bg);
    color: var(--color-text);
    font-size: var(--text-sm);
  }

  .hint {
    font-size: var(--text-xs);
    color: var(--color-text-dim);
  }

  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }
</style>
