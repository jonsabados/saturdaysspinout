import { ref, computed, watch } from 'vue'
import { defineStore } from 'pinia'
import { useSessionStore } from './session'
import { useApiClient, type Driver } from '@/api/client'

export const useDriverStore = defineStore('driver', () => {
  const sessionStore = useSessionStore()

  const driver = ref<Driver | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const racesIngestedTo = computed(() => {
    if (!driver.value?.racesIngestedTo) return null
    return new Date(driver.value.racesIngestedTo)
  })

  const ingestionBlockedUntil = computed(() => {
    if (!driver.value?.ingestionBlockedUntil) return null
    return new Date(driver.value.ingestionBlockedUntil)
  })

  const isIngestionBlocked = computed(() => {
    if (!ingestionBlockedUntil.value) return false
    return ingestionBlockedUntil.value > new Date()
  })

  const syncedToFormatted = computed(() => {
    if (!racesIngestedTo.value) return null
    return racesIngestedTo.value.toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    })
  })

  const blockedUntilFormatted = computed(() => {
    if (!ingestionBlockedUntil.value) return null
    return ingestionBlockedUntil.value.toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
    })
  })

  async function fetchDriver() {
    if (!sessionStore.userId) return

    loading.value = true
    error.value = null

    try {
      const apiClient = useApiClient()
      const response = await apiClient.getDriver(sessionStore.userId)
      driver.value = response.response
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch driver info'
      console.error('Failed to fetch driver:', e)
    } finally {
      loading.value = false
    }
  }

  async function refresh() {
    await fetchDriver()
  }

  function clear() {
    driver.value = null
    error.value = null
  }

  function incrementSessionCount() {
    if (driver.value) {
      driver.value = { ...driver.value, sessionCount: driver.value.sessionCount + 1 }
    }
  }

  // Fetch driver on initial login only (when we don't have driver data yet)
  // Don't clear on temporary disconnects to avoid UI flashing
  watch(
    () => sessionStore.isReady,
    (isReady) => {
      if (isReady && !driver.value) {
        fetchDriver()
      }
    },
    { immediate: true }
  )

  // Clear driver data on logout
  watch(
    () => sessionStore.isLoggedIn,
    (isLoggedIn) => {
      if (!isLoggedIn) {
        clear()
      }
    }
  )

  return {
    driver,
    loading,
    error,
    racesIngestedTo,
    ingestionBlockedUntil,
    isIngestionBlocked,
    syncedToFormatted,
    blockedUntilFormatted,
    fetchDriver,
    refresh,
    clear,
    incrementSessionCount,
  }
})