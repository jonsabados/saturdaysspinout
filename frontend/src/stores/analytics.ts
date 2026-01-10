import { ref, computed, watch } from 'vue'
import { defineStore } from 'pinia'
import { useSessionStore } from './session'
import { useWebSocketStore } from './websocket'
import {
  useApiClient,
  type Analytics,
  type AnalyticsDimensions,
  type AnalyticsGroupBy,
  type AnalyticsGranularity,
  type AnalyticsPeriod,
} from '@/api/client'

export interface DateRange {
  startTime: Date
  endTime: Date
}

export const useAnalyticsStore = defineStore('analytics', () => {
  const sessionStore = useSessionStore()

  // Core data
  const analytics = ref<Analytics | null>(null)
  const dimensions = ref<AnalyticsDimensions | null>(null)
  const timeSeries = ref<AnalyticsPeriod[] | null>(null)

  // Loading states
  const loading = ref(false)
  const loadingDimensions = ref(false)
  const loadingTimeSeries = ref(false)
  const error = ref<string | null>(null)

  // Filters and options
  const dateRange = ref<DateRange | null>(null)
  const groupBy = ref<AnalyticsGroupBy[]>([])
  const granularity = ref<AnalyticsGranularity>('week')
  const selectedSeriesIds = ref<number[]>([])
  const selectedCarIds = ref<number[]>([])
  const selectedTrackIds = ref<number[]>([])

  // Computed
  const hasData = computed(() => analytics.value !== null)
  const hasDimensions = computed(() => dimensions.value !== null)
  const hasTimeSeries = computed(() => timeSeries.value !== null && timeSeries.value.length > 0)
  const hasFilters = computed(
    () =>
      selectedSeriesIds.value.length > 0 ||
      selectedCarIds.value.length > 0 ||
      selectedTrackIds.value.length > 0
  )

  async function fetchDimensions() {
    if (!sessionStore.userId || !dateRange.value) return

    loadingDimensions.value = true
    error.value = null

    try {
      const apiClient = useApiClient()
      dimensions.value = await apiClient.getAnalyticsDimensions(
        sessionStore.userId,
        dateRange.value.startTime,
        dateRange.value.endTime
      )
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch dimensions'
      console.error('Failed to fetch analytics dimensions:', e)
    } finally {
      loadingDimensions.value = false
    }
  }

  async function fetchAnalytics() {
    if (!sessionStore.userId || !dateRange.value) return

    loading.value = true
    error.value = null

    try {
      const apiClient = useApiClient()
      analytics.value = await apiClient.getAnalytics(
        sessionStore.userId,
        dateRange.value.startTime,
        dateRange.value.endTime,
        {
          groupBy: groupBy.value ?? undefined,
          seriesIds: selectedSeriesIds.value.length > 0 ? selectedSeriesIds.value : undefined,
          carIds: selectedCarIds.value.length > 0 ? selectedCarIds.value : undefined,
          trackIds: selectedTrackIds.value.length > 0 ? selectedTrackIds.value : undefined,
        }
      )
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch analytics'
      console.error('Failed to fetch analytics:', e)
    } finally {
      loading.value = false
    }
  }

  async function fetchTimeSeries() {
    if (!sessionStore.userId || !dateRange.value) return

    loadingTimeSeries.value = true

    try {
      const apiClient = useApiClient()
      const result = await apiClient.getAnalyticsTimeSeries(
        sessionStore.userId,
        dateRange.value.startTime,
        dateRange.value.endTime,
        granularity.value,
        {
          seriesIds: selectedSeriesIds.value.length > 0 ? selectedSeriesIds.value : undefined,
          carIds: selectedCarIds.value.length > 0 ? selectedCarIds.value : undefined,
          trackIds: selectedTrackIds.value.length > 0 ? selectedTrackIds.value : undefined,
        }
      )
      timeSeries.value = result.timeSeries ?? null
    } catch (e) {
      console.error('Failed to fetch time series:', e)
      // Don't set error - time series is supplementary data
    } finally {
      loadingTimeSeries.value = false
    }
  }

  async function refresh() {
    await Promise.all([fetchDimensions(), fetchAnalytics(), fetchTimeSeries()])
  }

  function setDateRange(range: DateRange) {
    dateRange.value = range
    // Clear filters when date range changes since dimensions may be different
    clearFilters()
  }

  function setGroupBy(value: AnalyticsGroupBy[]) {
    groupBy.value = value
  }

  function toggleGroupBy(dimension: AnalyticsGroupBy) {
    const index = groupBy.value.indexOf(dimension)
    if (index === -1) {
      groupBy.value = [...groupBy.value, dimension]
    } else {
      groupBy.value = groupBy.value.filter((g) => g !== dimension)
    }
  }

  function setSeriesFilter(ids: number[]) {
    selectedSeriesIds.value = ids
  }

  function setCarFilter(ids: number[]) {
    selectedCarIds.value = ids
  }

  function setTrackFilter(ids: number[]) {
    selectedTrackIds.value = ids
  }

  function setGranularity(value: AnalyticsGranularity) {
    granularity.value = value
  }

  function clearFilters() {
    selectedSeriesIds.value = []
    selectedCarIds.value = []
    selectedTrackIds.value = []
  }

  function clear() {
    analytics.value = null
    dimensions.value = null
    timeSeries.value = null
    dateRange.value = null
    groupBy.value = []
    granularity.value = 'week'
    error.value = null
    clearFilters()
  }

  // Clear data on logout
  watch(
    () => sessionStore.isLoggedIn,
    (isLoggedIn) => {
      if (!isLoggedIn) {
        clear()
      }
    }
  )

  function setupListener() {
    const wsStore = useWebSocketStore()
    wsStore.on('ingestionChunkComplete', () => {
      console.log('[Analytics] Received ingestionChunkComplete, refreshing analytics')
      if (dateRange.value) {
        refresh()
      }
    })
  }

  return {
    // State
    analytics,
    dimensions,
    timeSeries,
    loading,
    loadingDimensions,
    loadingTimeSeries,
    error,
    dateRange,
    groupBy,
    granularity,
    selectedSeriesIds,
    selectedCarIds,
    selectedTrackIds,

    // Computed
    hasData,
    hasDimensions,
    hasTimeSeries,
    hasFilters,

    // Methods
    fetchDimensions,
    fetchAnalytics,
    fetchTimeSeries,
    refresh,
    setDateRange,
    setGroupBy,
    toggleGroupBy,
    setGranularity,
    setSeriesFilter,
    setCarFilter,
    setTrackFilter,
    clearFilters,
    clear,
    setupListener,
  }
})

export function setupAnalyticsListener() {
  const analyticsStore = useAnalyticsStore()
  analyticsStore.setupListener()
}