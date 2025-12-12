<script setup lang="ts">
import { RouterView } from 'vue-router'
import NavBar from '@/components/NavBar.vue'
</script>

<template>
  <div class="app">
    <NavBar />
    <main class="main-content">
      <RouterView />
    </main>
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