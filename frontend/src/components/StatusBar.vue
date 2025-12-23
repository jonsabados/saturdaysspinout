<script setup lang="ts">
import { useSessionStore } from '@/stores/session'
import RaceSyncStatus from './RaceSyncStatus.vue'

const sessionStore = useSessionStore()
</script>

<template>
  <div v-if="sessionStore.isLoggedIn" class="status-bar">
    <div class="status-bar-content">
      <RaceSyncStatus />
      <div class="status-stats">
        <!-- Future stats go here -->
      </div>
    </div>
  </div>
</template>

<style scoped>
.status-bar {
  height: 32px;
  background: var(--color-bg-elevated);
  border-bottom: 1px solid var(--color-border);
}

.status-bar-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 100%;
  padding: 0 1.5rem;
}

.status-stats {
  display: flex;
  align-items: center;
  gap: 1rem;
}
</style>