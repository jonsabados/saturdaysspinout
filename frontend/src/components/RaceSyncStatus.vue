<script setup lang="ts">
import { computed, ref, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRaceIngestionStore } from '@/stores/raceIngestion'
import { useSessionStore } from '@/stores/session'
import { useDriverStore } from '@/stores/driver'

const { t } = useI18n()

const COOLDOWN_MINUTES = 15
const COOLDOWN_MS = COOLDOWN_MINUTES * 60 * 1000

const ingestionStore = useRaceIngestionStore()
const sessionStore = useSessionStore()
const driverStore = useDriverStore()

const cooldownStartTime = ref<number | null>(null)
const now = ref(Date.now())

// Update 'now' every second to keep the countdown accurate
const interval = setInterval(() => {
  now.value = Date.now()
}, 1000)
onUnmounted(() => clearInterval(interval))

const cooldownRemaining = computed(() => {
  if (!cooldownStartTime.value) return 0
  const elapsed = now.value - cooldownStartTime.value
  return Math.max(0, COOLDOWN_MS - elapsed)
})

const isOnCooldown = computed(() => cooldownRemaining.value > 0)

const isLoading = computed(() => ingestionStore.status === 'loading')
const isDisabled = computed(() => !sessionStore.isReady || isLoading.value || isOnCooldown.value)

const syncStatusText = computed(() => {
  if (driverStore.syncedToFormatted) {
    return t('sync.syncedTo', { date: driverStore.syncedToFormatted })
  }
  return t('sync.notSynced')
})

const showCta = computed(() => {
  return sessionStore.isReady && !driverStore.syncedToFormatted
})

const buttonTitle = computed(() => {
  if (isOnCooldown.value) {
    const minutes = Math.ceil(cooldownRemaining.value / 60000)
    return t('sync.cooldownMessage', { minutes })
  }
  return t('sync.syncRaceHistory')
})

function handleClick() {
  if (!isDisabled.value) {
    cooldownStartTime.value = Date.now()
    ingestionStore.triggerIngestion()
  }
}
</script>

<template>
  <div class="sync-status">
    <span v-if="sessionStore.isReady" class="sync-text">{{ syncStatusText }}</span>
    <button
      class="sync-button"
      :class="{ loading: isLoading, disabled: isDisabled, blocked: isOnCooldown }"
      :disabled="isDisabled"
      @click="handleClick"
      :title="buttonTitle"
    >
      <span class="sync-icon" :class="{ spinning: isLoading }">&#x21bb;</span>
      <span class="sync-label">{{ t('sync.sync') }}</span>
    </button>
    <span v-if="showCta" class="sync-cta">
      {{ t('sync.getStarted') }}
    </span>
  </div>
</template>

<style scoped>
.sync-status {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.sync-text {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  white-space: nowrap;
}

.sync-button {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.25rem 0.5rem;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  color: var(--color-text-secondary);
  font-size: 0.75rem;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s, background 0.15s;
}

.sync-button:hover:not(.disabled) {
  color: var(--color-text-primary);
  border-color: var(--color-border-light);
  background: var(--color-accent-subtle);
}

.sync-button.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.sync-button.blocked {
  opacity: 0.5;
  cursor: help;
}

.sync-icon {
  font-size: 0.875rem;
  line-height: 1;
}

.sync-icon.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.sync-label {
  font-weight: 500;
}

.sync-cta {
  font-size: 0.75rem;
  color: var(--color-accent);
  white-space: nowrap;
  animation: pulse 1.5s ease-in-out 3;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

/* Hide button label on smaller screens, keep icon */
@media (max-width: 480px) {
  .sync-label {
    display: none;
  }

  .sync-button {
    padding: 0.25rem;
  }
}
</style>