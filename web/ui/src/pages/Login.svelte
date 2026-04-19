<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import { tick } from 'svelte'
  import { Box, Button, Input } from '@chrissnell/chonky-ui'
  import { login } from '../lib/auth.svelte.js'

  let username = $state('')
  let password = $state('')
  let error = $state('')
  let submitting = $state(false)

  // Rate-limit state: when truthy, submit is disabled and a countdown shows.
  let rateLimitUntil = $state(0) // epoch ms; 0 = not limited
  let now = $state(Date.now())
  let tickTimer

  let secondsRemaining = $derived(
    rateLimitUntil > now ? Math.ceil((rateLimitUntil - now) / 1000) : 0
  )
  let rateLimited = $derived(secondsRemaining > 0)

  function startCountdown(seconds) {
    rateLimitUntil = Date.now() + seconds * 1000
    if (tickTimer) clearInterval(tickTimer)
    tickTimer = setInterval(() => {
      now = Date.now()
      if (now >= rateLimitUntil) {
        clearInterval(tickTimer)
        tickTimer = null
        rateLimitUntil = 0
      }
    }, 1000)
  }

  $effect(() => {
    return () => { if (tickTimer) clearInterval(tickTimer) }
  })

  async function focusFirstInvalid() {
    await tick()
    const target = !username
      ? document.getElementById('login-username')
      : document.getElementById('login-password')
    target?.focus?.()
  }

  async function handleSubmit(e) {
    e.preventDefault()
    if (submitting || rateLimited) return
    error = ''
    submitting = true
    try {
      await login(username, password)
      // login() handles navigation via consumeReturnPath().
    } catch (err) {
      if (err?.status === 429 && err.retryAfter) {
        startCountdown(err.retryAfter)
        error = err.message || 'too many attempts'
      } else {
        error = err?.message || 'Login failed'
      }
      await focusFirstInvalid()
    } finally {
      submitting = false
    }
  }
</script>

<div class="login-page">
  <div class="login-container">
    <img src="/ui/blockasaurus-logo-face.svg" alt="" class="login-logo" />
    <h1>blockasaurus</h1>

    <Box title="Sign In">
      <form onsubmit={handleSubmit} novalidate>
          <div
            class="alert"
            role="alert"
            aria-live="polite"
            aria-atomic="true"
          >
            {#if rateLimited}
              Too many attempts. Try again in {secondsRemaining}s.
            {:else if error}
              {error}
            {/if}
          </div>

          <div class="field">
            <label for="login-username">Username</label>
            <Input
              id="login-username"
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
            <label for="login-password">Password</label>
            <Input
              id="login-password"
              type="password"
              bind:value={password}
              autocomplete="current-password"
              required
              disabled={submitting}
            />
          </div>

          <Button
            type="submit"
            disabled={submitting || rateLimited}
            aria-busy={submitting ? 'true' : 'false'}
            class="login-button"
          >
            {#if rateLimited}
              Try again in {secondsRemaining}s
            {:else if submitting}
              Signing in...
            {:else}
              Sign In
            {/if}
          </Button>
      </form>
    </Box>
  </div>
</div>

<style>
  .login-page {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 80vh;
  }

  .login-container {
    width: 100%;
    max-width: 380px;
    text-align: center;
  }

  .login-logo {
    width: 3rem;
    height: 3rem;
    margin-bottom: 0.5rem;
  }

  h1 {
    font-size: var(--text-2xl);
    font-weight: 700;
    margin-bottom: 1.5rem;
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

  /* Always-present alert region so screen readers announce transitions
     rather than the region appearing/disappearing. Empty space collapses. */
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

  :global(.login-button) {
    width: 100%;
    margin-top: 0.5rem;
  }
</style>
