<script setup lang="ts">
defineOptions({ name: 'RowActionButton' })

defineProps<{
  direction?: 'right' | 'down'
  loading?: boolean
  title?: string
}>()
</script>

<template>
  <button
    class="row-action-btn"
    :class="{ loading }"
    :disabled="loading"
    :title="title"
  >
    <span v-if="loading" class="spinner"></span>
    <span v-else class="chevron" :class="`chevron-${direction ?? 'right'}`"></span>
  </button>
</template>

<style scoped>
.row-action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}

.row-action-btn:hover:not(:disabled) {
  background: var(--color-accent-subtle);
  border-color: var(--color-border-light);
}

.row-action-btn:disabled {
  cursor: wait;
}

.chevron {
  display: block;
  width: 8px;
  height: 8px;
  border-right: 2px solid var(--color-text-secondary);
  border-bottom: 2px solid var(--color-text-secondary);
  transition: transform 0.2s;
}

.chevron-right {
  transform: rotate(-45deg);
  margin-left: -2px;
}

.chevron-down {
  transform: rotate(45deg);
  margin-top: -2px;
}

.spinner {
  display: block;
  width: 12px;
  height: 12px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>