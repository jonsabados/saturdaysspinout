<script setup lang="ts">
import { RouterView } from 'vue-router'
import NavBar from '@/components/NavBar.vue'
import StatusBar from '@/components/StatusBar.vue'
import SessionExpiredModal from '@/components/SessionExpiredModal.vue'
import LoadingModal from '@/components/LoadingModal.vue'
</script>

<template>
  <div class="app">
    <NavBar />
    <StatusBar />
    <main class="main-content">
      <RouterView v-slot="{ Component }">
        <KeepAlive include="RaceHistoryView,JournalView">
          <component :is="Component" />
        </KeepAlive>
      </RouterView>
    </main>
    <SessionExpiredModal />
    <LoadingModal />
  </div>
</template>

<style>
* {
  box-sizing: border-box;
}

body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
  background: var(--color-bg-deep);
  color: var(--color-text-primary);
}

.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.main-content {
  flex: 1;
  padding: 1.5rem;
}

/* Global link styles */
a {
  color: var(--color-accent);
  text-decoration: none;
  transition: color 0.15s;
}

a:hover {
  text-decoration: underline;
}

/* Global button styles */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  border: 1px solid transparent;
  transition:
    background-color 0.15s,
    border-color 0.15s,
    opacity 0.15s;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.8125rem;
}

.btn-primary {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: var(--color-accent-text);
}

.btn-primary:hover:not(:disabled) {
  background: var(--color-accent-hover);
  border-color: var(--color-accent-hover);
}

.btn-secondary {
  background: var(--color-bg-surface);
  border-color: var(--color-border);
  color: var(--color-text-primary);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--color-bg-elevated);
  border-color: var(--color-border-light);
}

.btn-danger {
  background: transparent;
  border-color: var(--color-border);
  color: #ef4444;
}

.btn-danger:hover:not(:disabled) {
  background: rgba(239, 68, 68, 0.1);
  border-color: #ef4444;
}

.btn-ghost {
  background: transparent;
  border-color: transparent;
  color: var(--color-text-secondary);
}

.btn-ghost:hover:not(:disabled) {
  background: var(--color-accent-subtle);
  color: var(--color-text-primary);
}
</style>