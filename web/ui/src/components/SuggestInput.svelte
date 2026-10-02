<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<!--
  A text input that offers matching values from a candidate list, without ever
  constraining what can be typed. Deliberately not chonky's Combobox: that one
  owns a selected value and snaps the field back to it, which would make an
  arbitrary substring -- the thing these filters are for -- impossible to enter.
  Here the suggestions are a shortcut and nothing more.
-->

<script>
  import { Input } from '@chrissnell/chonky-ui'
  import { matchSuggestions } from '../lib/loghistory.js'

  let {
    value = $bindable(''),
    // () => [{ value, hint }], called when the list opens rather than read on
    // every render -- see `openList`. `hint` is secondary text shown greyed to
    // the right, and is matched on too, so a client can be found by its IP or
    // by its name whichever of the two the candidate carries as its value.
    loadOptions,
    placeholder = '',
    label = '',
    // Enough to recognise the one you meant without turning into a list to read.
    max = 8,
    // Escape with the list closed belongs to the caller -- in the log filters it
    // clears the field.
    onEscape,
  } = $props()

  const listId = $props.id()

  let options = $state.raw([])
  let open = $state(false)
  // The highlight is held as a value, not an index. Callers build their
  // candidates from live data, so a list that is rebuilt while open would
  // otherwise move a different row under the reader's cursor.
  let activeValue = $state(null)

  const matches = $derived(matchSuggestions(options, value, max))
  const activeIndex = $derived(matches.findIndex((m) => m.value === activeValue))
  // A single suggestion identical to what is already typed is noise, not help.
  const exhausted = $derived(
    matches.length === 1
      && matches[0].value.toLowerCase() === value.trim().toLowerCase(),
  )
  const showList = $derived(open && matches.length > 0 && !exhausted)

  // Pulled once per opening, not derived. These candidates are tallied from the
  // live log buffer, which the stream rebuilds every animation frame: a derived
  // would re-tally a thousand rows per frame for a list nobody has opened.
  // Holding the snapshot while the list is open also keeps the rows from
  // reordering as traffic arrives.
  function openList() {
    options = loadOptions?.() ?? []
    open = true
  }

  function close() {
    open = false
    activeValue = null
  }

  function choose(option) {
    if (!option) return

    value = option.value
    close()
  }

  function move(delta) {
    if (!showList) {
      openList()

      return
    }

    const n = matches.length
    const next = activeIndex < 0
      ? (delta > 0 ? 0 : n - 1)
      : (activeIndex + delta + n) % n

    activeValue = matches[next].value
  }

  function onKeydown(e) {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        move(1)
        break
      case 'ArrowUp':
        e.preventDefault()
        move(-1)
        break
      case 'Enter':
        if (showList && activeIndex >= 0) {
          e.preventDefault()
          choose(matches[activeIndex])
        }
        break
      case 'Escape':
        // One Escape dismisses the list, a second one reaches the caller. Going
        // straight to "clear the field" would make the list impossible to
        // dismiss without losing what was typed.
        if (showList) {
          e.preventDefault()
          close()
        } else {
          onEscape?.()
        }
        break
      case 'Tab':
        close()
        break
    }
  }

  // focusout, not blur: blur does not bubble, so a handler on this wrapper would
  // never see the inner <input> lose focus at all.
  function onFocusOut(e) {
    if (e.currentTarget.contains(e.relatedTarget)) return

    close()
  }
</script>

<div class="suggest" onfocusout={onFocusOut}>
  <Input
    bind:value
    {placeholder}
    aria-label={label}
    role="combobox"
    aria-expanded={showList}
    aria-controls={showList ? listId : undefined}
    aria-autocomplete="list"
    aria-activedescendant={showList && activeIndex >= 0 ? `${listId}-${activeIndex}` : undefined}
    autocomplete="off"
    spellcheck="false"
    autocapitalize="off"
    onfocus={openList}
    oninput={() => { if (!open) openList(); activeValue = null }}
    onkeydown={onKeydown}
  />

  {#if showList}
    <ul class="list" id={listId} role="listbox" aria-label={`${label} suggestions`}>
      {#each matches as option, i (option.value)}
        <li
          id={`${listId}-${i}`}
          role="option"
          aria-selected={i === activeIndex}
          class:active={i === activeIndex}
          onmousedown={(e) => { e.preventDefault(); choose(option) }}
          onmousemove={() => (activeValue = option.value)}
        >
          <span class="value">{option.value}</span>
          {#if option.hint}<span class="hint">{option.hint}</span>{/if}
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .suggest {
    position: relative;
  }
  /* Inputs carry a global margin-bottom for stacked form fields; here it would
     leave the list floating away from the field it belongs to. */
  .suggest :global(input) {
    margin-bottom: 0;
  }
  .list {
    position: absolute;
    z-index: 20;
    top: calc(100% + 2px);
    left: 0;
    right: 0;
    margin: 0;
    padding: 0;
    list-style: none;
    max-height: 15rem;
    overflow-y: auto;
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
  }
  .list li {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--space-3);
    padding: 0.35rem 0.5rem;
    font-size: var(--text-sm);
    cursor: pointer;
  }
  .list li.active {
    background: var(--color-primary-muted);
  }
  .value {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .hint {
    color: var(--color-text-dim);
    font-size: var(--text-xs);
    white-space: nowrap;
  }
</style>
