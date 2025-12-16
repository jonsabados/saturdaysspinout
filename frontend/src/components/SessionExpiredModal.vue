<script setup lang="ts">
import { initiateLogin } from '@/auth/iracing'
import { useAuthStore } from '@/stores/auth'
import Modal from './Modal.vue'

const authStore = useAuthStore()

async function handleLogin() {
  authStore.clearSessionExpired()
  try {
    await initiateLogin()
  } catch (error) {
    console.error('Login failed:', error)
  }
}

function handleDismiss() {
  authStore.clearSessionExpired()
}
</script>

<template>
  <Modal v-if="authStore.sessionExpired" title="Session Expired" @close="handleDismiss">
    <p class="message">
      You've been logged out. Your session has expired and couldn't be refreshed.
    </p>

    <template #actions>
      <button class="btn btn-secondary" @click="handleDismiss">
        Dismiss
      </button>
      <button class="btn btn-primary" @click="handleLogin">
        Log In
      </button>
    </template>
  </Modal>
</template>

<style scoped>
.message {
  margin: 0;
  color: var(--color-text-secondary);
  line-height: 1.5;
}

.btn {
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-secondary {
  background: transparent;
  border: 1px solid var(--color-border-light);
  color: var(--color-text-secondary);
}

.btn-secondary:hover {
  border-color: var(--color-accent);
  color: var(--color-text-primary);
}

.btn-primary {
  background: var(--color-accent);
  border: 1px solid var(--color-accent);
  color: var(--color-bg-deep);
  font-weight: 500;
}

.btn-primary:hover {
  background: var(--color-accent-hover);
  border-color: var(--color-accent-hover);
}
</style>