<script setup lang="ts">
defineOptions({ name: 'RowActionButton' })

defineProps<{
  direction?: 'right' | 'down'
  loading?: boolean
  title?: string
  label?: string
}>()
</script>

<template>
  <button
    class="row-action-btn"
    :class="{ loading, 'has-label': !!label }"
    :disabled="loading"
    :title="title"
  >
    <span v-if="loading" class="spinner"></span>
    <template v-else>
      <span v-if="label" class="btn-label">{{ label }}</span>
      <span class="chevron" :class="`chevron-${direction ?? 'right'}`"></span>
    </template>
  </button>
</template>

<style scoped>
.row-action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.375rem;
  min-width: 40px;
  height: 32px;
  padding: 0 0.5rem;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, box-shadow 0.15s;
}

.row-action-btn:hover:not(:disabled) {
  background: var(--color-accent-subtle);
  border-color: var(--color-accent);
  box-shadow: 0 0 0 1px var(--color-accent-subtle);
}

.row-action-btn:active:not(:disabled) {
  background: var(--color-accent-muted);
}

.row-action-btn:disabled {
  cursor: wait;
  opacity: 0.7;
}

.btn-label {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  white-space: nowrap;
  transition: color 0.15s;
}

.row-action-btn:hover:not(:disabled) .btn-label {
  color: var(--color-text-primary);
}

.chevron {
  display: block;
  width: 7px;
  height: 7px;
  border-right: 2px solid var(--color-text-muted);
  border-bottom: 2px solid var(--color-text-muted);
  transition: transform 0.2s, border-color 0.15s;
  flex-shrink: 0;
}

.row-action-btn:hover:not(:disabled) .chevron {
  border-color: var(--color-accent);
}

.chevron-right {
  transform: rotate(-45deg);
}

.chevron-down {
  transform: rotate(45deg);
}

.spinner {
  display: block;
  width: 14px;
  height: 14px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Hide label on small screens, keep icon */
@media (max-width: 600px) {
  .btn-label {
    display: none;
  }

  .row-action-btn.has-label {
    padding: 0 0.375rem;
  }
}
</style>