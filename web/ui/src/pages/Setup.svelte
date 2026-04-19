<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { tick } from 'svelte'
  import { Box, Button, Input } from '@chrissnell/chonky-ui'
  import { setup } from '../lib/auth.svelte.js'

  const MIN_LENGTH = 12

  let username = $state('admin')
  let password = $state('')
  let confirm = $state('')
  let error = $state('')
  let submitting = $state(false)

  let passwordLength = $derived(password.length)
  let passwordLongEnough = $derived(passwordLength >= MIN_LENGTH)
  let passwordsMatch = $derived(password === confirm && confirm.length > 0)
  let canSubmit = $derived(
    !submitting && username.trim().length > 0 && passwordLongEnough && passwordsMatch
  )

  async function focusField(id) {
    await tick()
    document.getElementById(id)?.focus?.()
  }

  async function handleSubmit(e) {
    e.preventDefault()
    if (!canSubmit) return
    error = ''
    submitting = true
    try {
      await setup(username, password)
      // setup() handles navigation via consumeReturnPath().
    } catch (err) {
      error = err?.message || 'Setup failed'
      // Focus the most likely culprit: username if short, else password.
      await focusField(username.trim().length === 0 ? 'setup-username' : 'setup-password')
    } finally {
      submitting = false
    }
  }
</script>

<div class="setup-page">
  <div class="setup-container">
    <img src="/ui/blockasaurus-logo-face.svg" alt="" class="setup-logo" />
    <h1>Welcome to Blockasaurus</h1>
    <p class="subtitle">Create your admin account to get started.</p>

    <Box title="Initial Setup">
      <form onsubmit={handleSubmit} novalidate>
          <div
            class="alert"
            role="alert"
            aria-live="polite"
            aria-atomic="true"
          >
            {#if error}{error}{/if}
          </div>

          <div class="field">
            <label for="setup-username">Username</label>
            <Input
              id="setup-username"
              type="text"
              bind:value={username}
              autocomplete="username"
              autocapitalize="off"
              spellcheck="false"
              required
              disabled={submitting}
            />
          </div>

          <div class="field">
            <label for="setup-password">Password</label>
            <Input
              id="setup-password"
              type="password"
              bind:value={password}
              autocomplete="new-password"
              aria-describedby="setup-password-help"
              required
              disabled={submitting}
            />
            <span
              id="setup-password-help"
              class="hint"
              class:ok={passwordLongEnough}
              aria-live="polite"
            >
              {passwordLength} / {MIN_LENGTH} characters
            </span>
          </div>

          <div class="field">
            <label for="setup-confirm">Confirm Password</label>
            <Input
              id="setup-confirm"
              type="password"
              bind:value={confirm}
              autocomplete="new-password"
              aria-describedby="setup-confirm-help"
              required
              disabled={submitting}
            />
            <span
              id="setup-confirm-help"
              class="hint"
              class:ok={passwordsMatch}
              aria-live="polite"
            >
              {#if confirm.length === 0}
                Re-enter the password above.
              {:else if passwordsMatch}
                Passwords match.
              {:else}
                Passwords do not match.
              {/if}
            </span>
          </div>

          <Button
            type="submit"
            disabled={!canSubmit}
            aria-busy={submitting ? 'true' : 'false'}
            class="setup-button"
          >
            {submitting ? 'Creating account...' : 'Create Admin Account'}
          </Button>
      </form>
    </Box>
  </div>
</div>

<style>
  .setup-page {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 80vh;
  }

  .setup-container {
    width: 100%;
    max-width: 420px;
    text-align: center;
  }

  .setup-logo {
    width: 3rem;
    height: 3rem;
    margin-bottom: 0.5rem;
  }

  h1 {
    font-size: var(--text-2xl);
    font-weight: 700;
    margin-bottom: 0.25rem;
  }

  .subtitle {
    color: var(--color-text-dim);
    margin-bottom: 1.5rem;
    font-size: var(--text-sm);
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    text-align: left;
  }

  .field label {
    font-size: var(--text-sm);
    font-weight: 500;
  }

  .hint {
    font-size: var(--text-xs);
    color: var(--color-text-dim);
  }

  .hint.ok {
    color: var(--color-success, #16a34a);
  }

  .alert {
    min-height: 0;
  }
  .alert:empty {
    display: none;
  }
  .alert:not(:empty) {
    padding: 0.5rem 0.75rem;
    background: var(--color-danger-subtle, #fef2f2);
    color: var(--color-danger, #dc2626);
    border-radius: var(--radius);
    font-size: var(--text-sm);
    text-align: left;
  }

  :global(.setup-button) {
    width: 100%;
    margin-top: 0.5rem;
  }
</style>
