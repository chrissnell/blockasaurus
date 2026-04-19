// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

// WebSocket log stream with auth-aware auto-reconnect.
//
// A 401 on the WS upgrade closes the socket with a CloseEvent (no status
// surfaced); without a probe we'd reconnect forever against a dead session.
// On any unexpected close we probe /api/auth/session:
//   - 401 → stop reconnecting, trigger the same 401 flow as api.js
//           (stash return path, redirect to #/login)
//   - 2xx → fall through to normal reconnect backoff
export function connectLogStream(onEntry, onStatus) {
  let ws
  let reconnectTimer
  let alive = true

  function stashReturnPathForLogin() {
    const current = window.location.hash.slice(1) || '/'
    if (current !== '/login' && current !== '/') {
      sessionStorage.setItem('authReturnTo', current)
    }
    window.location.hash = '#/login'
  }

  async function probeSessionAndMaybeReconnect() {
    if (!alive) return
    try {
      const resp = await fetch('/api/auth/session', {
        credentials: 'include',
        headers: { 'X-Requested-With': 'fetch' },
      })
      if (resp.status === 401) {
        alive = false
        stashReturnPathForLogin()
        return
      }
    } catch {
      // Network error — fall through to reconnect; the backoff will
      // naturally throttle retries.
    }
    if (alive) reconnectTimer = setTimeout(connect, 3000)
  }

  function connect() {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    ws = new WebSocket(`${proto}//${location.host}/api/ws/logs`)

    ws.onopen = () => onStatus(true)

    ws.onmessage = (evt) => {
      try {
        onEntry(JSON.parse(evt.data))
      } catch { /* ignore malformed */ }
    }

    ws.onclose = () => {
      onStatus(false)
      if (alive) probeSessionAndMaybeReconnect()
    }

    ws.onerror = () => ws.close()
  }

  connect()

  return function disconnect() {
    alive = false
    clearTimeout(reconnectTimer)
    if (ws) ws.close()
  }
}
