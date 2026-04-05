// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

import '@chrissnell/chonky-ui/css'
import './styles/tokens.css'
import { mount } from 'svelte'
import App from './App.svelte'

// Polyfill crypto.randomUUID for insecure-context (plain HTTP on LAN).
// chonky-ui's toast() depends on it; crypto.getRandomValues is available even
// outside secure contexts, so synthesize a v4 UUID from it.
if (typeof crypto !== 'undefined' && typeof crypto.randomUUID !== 'function') {
  crypto.randomUUID = function () {
    const b = crypto.getRandomValues(new Uint8Array(16))
    b[6] = (b[6] & 0x0f) | 0x40
    b[8] = (b[8] & 0x3f) | 0x80
    const h = [...b].map((x) => x.toString(16).padStart(2, '0'))
    return `${h[0]}${h[1]}${h[2]}${h[3]}-${h[4]}${h[5]}-${h[6]}${h[7]}-${h[8]}${h[9]}-${h[10]}${h[11]}${h[12]}${h[13]}${h[14]}${h[15]}`
  }
}

mount(App, { target: document.getElementById('app') })
