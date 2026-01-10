import { ref, computed, watch } from 'vue'
import { defineStore } from 'pinia'
import { useAuthStore } from './auth'
import { useApiClient, type Series } from '@/api/client'

export const useSeriesStore = defineStore('series', () => {
  const authStore = useAuthStore()

  const series = ref<Map<number, Series>>(new Map())
  const loading = ref(false)
  const error = ref<string | null>(null)

  const isLoaded = computed(() => series.value.size > 0)

  function getSeries(seriesId: number): Series | undefined {
    return series.value.get(seriesId)
  }

  async function fetchSeries() {
    if (loading.value) return

    loading.value = true
    error.value = null

    try {
      const apiClient = useApiClient()
      const response = await apiClient.getSeries()
      const seriesMap = new Map<number, Series>()
      for (const s of response.response) {
        seriesMap.set(s.id, s)
      }
      series.value = seriesMap
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch series'
      console.error('Failed to fetch series:', e)
    } finally {
      loading.value = false
    }
  }

  function clear() {
    series.value = new Map()
    error.value = null
  }

  watch(
    () => authStore.isLoggedIn,
    (isLoggedIn) => {
      if (!isLoggedIn) {
        clear()
      }
    }
  )

  return {
    series,
    loading,
    error,
    isLoaded,
    getSeries,
    fetchSeries,
    clear,
  }
})