<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  import {
    Button,
    Badge,
    Box,
    BoxHeader,
    StatCard,
    Table,
    Input,
    Select,
    Toggle,
    Checkbox,
    Radio,
    RadioGroup,
    Combobox,
    Modal,
    Spinner,
    EmptyState,
    Label,
    Separator,
    Dot,
    Tabs,
    Tooltip,
    Popover,
    ApplyBanner,
    StatusBar,
    ThemeToggle,
    Toaster,
    toast,
  } from '@chrissnell/chonky-ui'

  // Local state for interactive examples
  let modalOpen = $state(false)
  let toggleChecked = $state(true)
  let checkboxChecked = $state(false)
  let inputValue = $state('')
  let selectValue = $state('dns')
  let radioValue = $state('allow')
  let comboboxOpen = $state(false)
  let comboboxValue = $state('')
  let tabsValue = $state('one')
  let theme = $state(
    typeof document !== 'undefined'
      ? /** @type {'light' | 'dark'} */ (
          document.documentElement.getAttribute('data-theme') || 'dark'
        )
      : 'dark'
  )

  const selectOptions = [
    { value: 'dns', label: 'DNS' },
    { value: 'http', label: 'HTTP' },
    { value: 'tls', label: 'TLS' },
  ]

  const fruits = [
    { value: 'apple', label: 'Apple' },
    { value: 'banana', label: 'Banana' },
    { value: 'cherry', label: 'Cherry' },
    { value: 'date', label: 'Date' },
  ]
</script>

<Toaster />

<div class="dev-page">
  <header class="page-header">
    <h1>Chonky UI Showcase</h1>
    <p class="subtitle">Component gallery for @chrissnell/chonky-ui</p>
  </header>

  <!-- ============ CORE ============ -->
  <section>
    <h2>Core</h2>
    <div class="grid">
      <Box title="Button — variants">
        <div class="row">
          <Button>default</Button>
          <Button variant="primary">primary</Button>
          <Button variant="accent">accent</Button>
          <Button variant="danger">danger</Button>
          <Button variant="ghost">ghost</Button>
        </div>
      </Box>

      <Box title="Button — sizes">
        <div class="row">
          <Button size="sm">small</Button>
          <Button size="md">medium</Button>
          <Button size="lg">large</Button>
        </div>
      </Box>

      <Box title="Badge">
        <div class="row">
          <Badge>default</Badge>
          <Badge variant="success">success</Badge>
          <Badge variant="warning">warning</Badge>
          <Badge variant="danger">danger</Badge>
          <Badge variant="info">info</Badge>
        </div>
      </Box>

      <Box title="Box + BoxHeader">
        <BoxHeader>
          <strong>Header row</strong>
          <Badge variant="info">12</Badge>
        </BoxHeader>
        <p class="muted">A titled box may also include a BoxHeader row.</p>
      </Box>

      <Box title="StatCard">
        <div class="row">
          <StatCard label="queries" value="1,284" />
          <StatCard label="blocked" value="73" variant="danger" />
          <StatCard label="uptime" value="99.9%" variant="success" />
        </div>
      </Box>

      <Box title="Table">
        <Table>
          <thead>
            <tr><th>name</th><th>type</th><th>status</th></tr>
          </thead>
          <tbody>
            <tr><td>ads</td><td>deny</td><td><Badge variant="success">active</Badge></td></tr>
            <tr><td>malware</td><td>deny</td><td><Badge variant="success">active</Badge></td></tr>
            <tr><td>whitelist</td><td>allow</td><td><Badge>inactive</Badge></td></tr>
          </tbody>
        </Table>
      </Box>

      <Box title="Label, Separator, Dot">
        <Label for="demo-input">A label</Label>
        <Separator />
        <div class="row">
          <span><Dot /> off</span>
          <span><Dot on /> on</span>
        </div>
      </Box>
    </div>
  </section>

  <!-- ============ FORM ============ -->
  <section>
    <h2>Form</h2>
    <div class="grid">
      <Box title="Input">
        <Label for="demo-input">Name</Label>
        <Input id="demo-input" bind:value={inputValue} placeholder="type something…" />
        <p class="muted">value: {inputValue || '(empty)'}</p>
      </Box>

      <Box title="Select">
        <Select options={selectOptions} bind:value={selectValue} placeholder="Pick a protocol" />
        <p class="muted">selected: {selectValue}</p>
      </Box>

      <Box title="Toggle">
        <Toggle bind:checked={toggleChecked} label="enabled" />
      </Box>

      <Box title="Checkbox">
        <Checkbox bind:checked={checkboxChecked} label="I agree" />
      </Box>

      <Box title="RadioGroup">
        <RadioGroup bind:value={radioValue} name="demo-radio">
          <Radio value="allow" label="allow" />
          <Radio value="deny" label="deny" />
          <Radio value="log" label="log only" />
        </RadioGroup>
        <p class="muted">selected: {radioValue}</p>
      </Box>

      <Box title="Combobox">
        <Combobox.Root bind:open={comboboxOpen} bind:value={comboboxValue}>
          <Combobox.Input placeholder="Pick a fruit…" />
          <Combobox.Trigger>▾</Combobox.Trigger>
          <Combobox.Content>
            {#each fruits as f}
              <Combobox.Item value={f.value} label={f.label}>{f.label}</Combobox.Item>
            {/each}
          </Combobox.Content>
        </Combobox.Root>
        <p class="muted">selected: {comboboxValue || '(none)'}</p>
      </Box>
    </div>
  </section>

  <!-- ============ OVERLAY ============ -->
  <section>
    <h2>Overlay</h2>
    <div class="grid">
      <Box title="Modal">
        <Button variant="primary" onclick={() => (modalOpen = true)}>open modal</Button>
        <Modal bind:open={modalOpen}>
          <Modal.Header>
            <strong>Example modal</strong>
            <Modal.Close />
          </Modal.Header>
          <Modal.Body>
            <p>This is a Chonky modal dialog. Click outside or press escape to close.</p>
          </Modal.Body>
          <Modal.Footer>
            <Button onclick={() => (modalOpen = false)}>cancel</Button>
            <Button variant="primary" onclick={() => (modalOpen = false)}>ok</Button>
          </Modal.Footer>
        </Modal>
      </Box>

      <Box title="Tooltip">
        <Tooltip.Root>
          <Tooltip.Trigger>
            <Button>hover me</Button>
          </Tooltip.Trigger>
          <Tooltip.Content>A helpful hint</Tooltip.Content>
        </Tooltip.Root>
      </Box>

      <Box title="Popover">
        <Popover.Root>
          <Popover.Trigger>
            <Button>open popover</Button>
          </Popover.Trigger>
          <Popover.Content>
            <div class="popover-body">
              <strong>Popover content</strong>
              <p class="muted">Any markup you like.</p>
            </div>
          </Popover.Content>
        </Popover.Root>
      </Box>
    </div>
  </section>

  <!-- ============ NAVIGATION ============ -->
  <section>
    <h2>Navigation</h2>
    <div class="grid">
      <Box title="Tabs">
        <Tabs.Root bind:value={tabsValue}>
          <Tabs.List>
            <Tabs.Trigger value="one">One</Tabs.Trigger>
            <Tabs.Trigger value="two">Two</Tabs.Trigger>
            <Tabs.Trigger value="three">Three</Tabs.Trigger>
          </Tabs.List>
          <Tabs.Content value="one">First panel content.</Tabs.Content>
          <Tabs.Content value="two">Second panel content.</Tabs.Content>
          <Tabs.Content value="three">Third panel content.</Tabs.Content>
        </Tabs.Root>
      </Box>

      <Box title="ThemeToggle">
        <ThemeToggle bind:theme />
        <p class="muted">current: {theme}</p>
      </Box>
    </div>
  </section>

  <!-- ============ FEEDBACK ============ -->
  <section>
    <h2>Feedback</h2>
    <div class="grid">
      <Box title="Spinner">
        <div class="row">
          <Spinner size={16} />
          <Spinner size={24} />
          <Spinner size={32} />
        </div>
      </Box>

      <Box title="EmptyState">
        <EmptyState>
          <p>Nothing to see here.</p>
          <Button size="sm">create one</Button>
        </EmptyState>
      </Box>

      <Box title="ApplyBanner">
        <ApplyBanner count={3} onApply={() => toast('applied!')} />
      </Box>

      <Box title="StatusBar">
        <StatusBar>
          <span><Dot on /> connected</span>
          <span>v0.33.40</span>
          <span>ready</span>
        </StatusBar>
      </Box>

      <Box title="Toast">
        <div class="row">
          <Button onclick={() => toast('hello')}>fire toast</Button>
          <Button variant="primary" onclick={() => toast('saved successfully', 'success')}>
            success toast
          </Button>
          <Button variant="danger" onclick={() => toast('something broke', 'danger')}>
            danger toast
          </Button>
        </div>
      </Box>
    </div>
  </section>
</div>

<style>
  .dev-page {
    padding: 1.5rem;
    max-width: 1200px;
    margin: 0 auto;
  }

  .page-header {
    margin-bottom: 2rem;
  }

  .page-header h1 {
    margin: 0 0 0.25rem 0;
  }

  .subtitle {
    margin: 0;
    opacity: 0.7;
  }

  section {
    margin-bottom: 2.5rem;
  }

  section h2 {
    margin: 0 0 1rem 0;
    font-size: 1.1rem;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    opacity: 0.8;
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    gap: 1rem;
  }

  .row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
  }

  .muted {
    margin: 0.5rem 0 0 0;
    opacity: 0.6;
    font-size: 0.85rem;
  }

  .popover-body {
    padding: 0.5rem 0.75rem;
    min-width: 180px;
  }
</style>
