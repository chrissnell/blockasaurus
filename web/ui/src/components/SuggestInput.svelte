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

  let {
    value = $bindable(''),
    // [{ value, hint }]. `hint` is secondary text shown greyed to the right and
    // is matched on too, so a client can be found by its IP or by its name.
    options = [],
    placeholder = '',
    label = '',
    id = undefined,
    // Enough to recognise the one you meant without turning into a list to read.
    max = 8,
    // Escape with the list closed belongs to the caller -- in the log filters it
    // clears the field.
    onEscape,
  } = $props()

  const listId = $props.id()

  let open = $state(false)
  let active = $state(-1)

  const query = $derived(value.trim().toLowerCase())

  // Prefix matches first: typing "192.168.1.1" should offer that address before
  // a name that merely contains the digits.
  const matches = $derived.by(() => {
    const prefix = []
    const substring = []

    for (const option of options) {
      if (!query) {
        prefix.push(option)
      } else {
        const at = `${option.value} ${option.hint || ''}`.toLowerCase().indexOf(query)
        if (at === 0) prefix.push(option)
        else if (at > 0) substring.push(option)
      }

      if (prefix.length >= max) break
    }

    return [...prefix, ...substring].slice(0, max)
  })

  // A single suggestion identical to what is already typed is noise, not help.
  const exhausted = $derived(matches.length === 1 && matches[0].value === value.trim())
  const showList = $derived(open && matches.length > 0 && !exhausted)

  function choose(option) {
    value = option.value
    open = false
    active = -1
  }

  function move(delta) {
    if (!showList) {
      open = true

      return
    }

    const n = matches.length
    active = active < 0 && delta < 0 ? n - 1 : (active + delta + n) % n
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
        if (showList && active >= 0) {
          e.preventDefault()
          choose(matches[active])
        }
        break
      case 'Escape':
        // One Escape dismisses the list, a second one reaches the caller. Going
        // straight to "clear the field" would make the list impossible to
        // dismiss without losing what was typed.
        if (showList) {
          e.preventDefault()
          open = false
          active = -1
        } else {
          onEscape?.()
        }
        break
      case 'Tab':
        open = false
        break
    }
  }

  // focusout rather than blur: clicking a suggestion moves focus within the
  // wrapper, and closing on the way there would unmount the click target.
  function onFocusOut(e) {
    if (e.currentTarget.contains(e.relatedTarget)) return

    open = false
    active = -1
  }
</script>

<div class="suggest" onfocusout={onFocusOut}>
  <Input
    {id}
    bind:value
    {placeholder}
    aria-label={label}
    role="combobox"
    aria-expanded={showList}
    aria-controls={listId}
    aria-autocomplete="list"
    aria-activedescendant={showList && active >= 0 ? `${listId}-${active}` : undefined}
    autocomplete="off"
    spellcheck="false"
    autocapitalize="off"
    onfocus={() => (open = true)}
    oninput={() => { open = true; active = -1 }}
    onkeydown={onKeydown}
  />

  {#if showList}
    <ul class="list" id={listId} role="listbox" aria-label={`${label} suggestions`}>
      {#each matches as option, i (option.value)}
        <li
          id={`${listId}-${i}`}
          role="option"
          aria-selected={i === active}
          class:active={i === active}
          onmousedown={(e) => { e.preventDefault(); choose(option) }}
          onmousemove={() => (active = i)}
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
