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
      <RouterView />
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
</style>