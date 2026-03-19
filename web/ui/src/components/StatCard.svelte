<!-- Copyright 2026 Chris Snell -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script>
  let { label, value, color = 'success' } = $props()
</script>

<div class="stat-card stat-card--{color}">
  <span class="stat-card__label">{label}</span>
  <span class="stat-card__value">{value}</span>
</div>

<style>
  .stat-card {
    padding: var(--space-4) var(--space-5);
    border-radius: var(--radius);
    border-left: 3px solid;
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }
  .stat-card__label {
    font-size: var(--text-xs);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    opacity: 0.85;
  }
  .stat-card__value {
    font-size: var(--text-2xl);
    font-weight: 700;
  }
  .stat-card--success {
    background: var(--color-success-muted);
    border-color: var(--color-success);
    color: var(--color-text);
  }
  .stat-card--danger {
    background: var(--color-danger-muted);
    border-color: var(--color-danger);
    color: var(--color-text);
  }
  .stat-card--primary {
    background: var(--color-primary-muted);
    border-color: var(--color-primary);
    color: var(--color-text);
  }
  .stat-card--info {
    background: var(--color-info-muted);
    border-color: var(--color-info);
    color: var(--color-text);
  }
  .stat-card--blocked {
    background: var(--color-blocked-muted);
    border-color: var(--color-blocked);
    color: var(--color-text);
  }
</style>
