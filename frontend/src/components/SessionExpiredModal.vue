<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { initiateLogin } from '@/auth/iracing'
import { useAuthStore } from '@/stores/auth'
import Modal from './Modal.vue'

const { t } = useI18n()
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
  <Modal v-if="authStore.sessionExpired" :title="t('sessionExpired.title')" @close="handleDismiss">
    <p class="message">
      {{ t('sessionExpired.message') }}
    </p>

    <template #actions>
      <button class="btn btn-secondary" @click="handleDismiss">
        {{ t('common.dismiss') }}
      </button>
      <button class="btn btn-primary" @click="handleLogin">
        {{ t('common.login') }}
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
</style>