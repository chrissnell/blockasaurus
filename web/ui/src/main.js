// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

import '@chrissnell/chonky-ui/css'
import './styles/tokens.css'
import { mount } from 'svelte'
import App from './App.svelte'

mount(App, { target: document.getElementById('app') })
