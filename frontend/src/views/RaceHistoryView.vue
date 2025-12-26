<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { VueDatePicker } from '@vuepic/vue-datepicker'
import '@vuepic/vue-datepicker/dist/main.css'
import { useI18n } from 'vue-i18n'
import { useApiClient, type Race } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { useWebSocketStore } from '@/stores/websocket'
import { useDriverStore } from '@/stores/driver'
import GridPosition from '@/components/GridPosition.vue'
import TrackCell from '@/components/TrackCell.vue'
import SessionCell from '@/components/SessionCell.vue'
import LicenseCell from '@/components/LicenseCell.vue'

const { t } = useI18n()
const apiClient = useApiClient()
const auth = useAuthStore()
const wsStore = useWebSocketStore()
const driverStore = useDriverStore()

const races = ref<Race[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const currentPage = ref(1)
const totalMatching = ref(0)
const hasMorePages = ref(false)
const pageSize = 50
const sentinelRef = ref<HTMLElement | null>(null)
const scrollContainerRef = ref<HTMLElement | null>(null)
const observer = ref<IntersectionObserver | null>(null)
const userHasScrolled = ref(false)

// Date filters
const fromDate = ref<Date | null>(null)
const toDate = ref<Date | null>(null)
const initialized = ref(false)

function isRaceInDateRange(raceStartTime: string): boolean {
  if (!fromDate.value || !toDate.value) return false
  const raceDate = new Date(raceStartTime)
  return raceDate >= fromDate.value && raceDate <= toDate.value
}

async function fetchRaces() {
  if (!auth.userId || !fromDate.value || !toDate.value) return

  loading.value = true
  currentPage.value = 1
  try {
    const response = await apiClient.getRaces(auth.userId, fromDate.value, toDate.value, 1, pageSize)
    races.value = response.items
    totalMatching.value = response.pagination.totalResults
    hasMorePages.value = response.pagination.page < response.pagination.totalPages
  } catch (err) {
    console.error('[RaceHistory] Failed to fetch races:', err)
  } finally {
    loading.value = false
  }
}

async function fetchMoreRaces() {
  if (!auth.userId || !fromDate.value || !toDate.value || loadingMore.value || !hasMorePages.value) return

  loadingMore.value = true
  const nextPage = currentPage.value + 1
  try {
    const response = await apiClient.getRaces(auth.userId, fromDate.value, toDate.value, nextPage, pageSize)
    races.value = [...races.value, ...response.items]
    currentPage.value = nextPage
    hasMorePages.value = response.pagination.page < response.pagination.totalPages
  } catch (err) {
    console.error('[RaceHistory] Failed to fetch more races:', err)
  } finally {
    loadingMore.value = false
  }
}

// Initialize date filters when driver loads (once only)
watch(
  () => driverStore.driver,
  (driver) => {
    if (driver?.memberSince && !initialized.value) {
      fromDate.value = new Date(driver.memberSince)
      toDate.value = new Date()
      initialized.value = true
      fetchRaces()
    }
  },
  { immediate: true }
)

// Refetch when date filters change (user interaction only, not initialization)
watch([fromDate, toDate], ([newFrom, newTo], [oldFrom, oldTo]) => {
  if (initialized.value && newFrom && newTo && (oldFrom !== newFrom || oldTo !== newTo)) {
    fetchRaces()
  }
})

const sortedRaces = computed(() =>
  [...races.value].sort((a, b) => new Date(b.startTime).getTime() - new Date(a.startTime).getTime()),
)

interface RaceIngestedPayload {
  raceId: number
}

async function handleRaceIngested(payload: unknown) {
  const { raceId } = payload as RaceIngestedPayload
  if (!auth.userId) {
    console.warn('[RaceHistory] Received race ingested event but no userId')
    return
  }

  try {
    const response = await apiClient.getRace(auth.userId, raceId)
    const race = response.response

    // Always increment total count
    driverStore.incrementSessionCount()

    // Only add to table if within current filter range and not already present
    if (isRaceInDateRange(race.startTime) && !races.value.some((r) => r.id === race.id)) {
      races.value.push(race)
      totalMatching.value++
    }
  } catch (err) {
    console.error('[RaceHistory] Failed to fetch ingested race:', err)
  }
}

function formatDate(isoString: string): string {
  return new Date(isoString).toLocaleDateString()
}

function formatIRatingDiff(oldRating: number, newRating: number): string {
  const diff = newRating - oldRating
  const sign = diff > 0 ? '+' : ''
  return `(${sign}${diff})`
}

function getIRatingDiffClass(oldRating: number, newRating: number): string {
  const diff = newRating - oldRating
  if (diff > 0) return 'stat-gain'
  if (diff < 0) return 'stat-loss'
  return ''
}

function handleScroll() {
  userHasScrolled.value = true
  scrollContainerRef.value?.removeEventListener('scroll', handleScroll)
}

function setupObserver() {
  if (observer.value) {
    observer.value.disconnect()
  }
  observer.value = new IntersectionObserver(
    (entries) => {
      if (entries[0].isIntersecting && hasMorePages.value && userHasScrolled.value && !loadingMore.value) {
        fetchMoreRaces()
      }
    },
    { root: scrollContainerRef.value, rootMargin: '100px' },
  )
  if (sentinelRef.value) {
    observer.value.observe(sentinelRef.value)
  }
}

watch(sentinelRef, (el) => {
  if (el && observer.value) {
    observer.value.observe(el)
  }
})

watch(scrollContainerRef, (el, oldEl) => {
  if (oldEl) {
    oldEl.removeEventListener('scroll', handleScroll)
  }
  if (el) {
    userHasScrolled.value = false
    el.addEventListener('scroll', handleScroll)
    setupObserver()
  }
})

onMounted(() => {
  wsStore.on('raceIngested', handleRaceIngested)
})

onUnmounted(() => {
  wsStore.off('raceIngested', handleRaceIngested)
  scrollContainerRef.value?.removeEventListener('scroll', handleScroll)
  if (observer.value) {
    observer.value.disconnect()
    observer.value = null
  }
})
</script>

<template>
  <div class="race-history">
    <div class="page-header">
      <h1>{{ t('raceHistory.title') }}</h1>
      <span v-if="driverStore.driver?.sessionCount" class="total-races">
        {{ t('raceHistory.racesOnFile', { count: driverStore.driver.sessionCount }) }}
      </span>
    </div>

    <div class="filters">
      <div class="filter-group">
        <label>{{ t('raceHistory.from') }}</label>
        <VueDatePicker
          v-model="fromDate"
          :disabled="loading"
          dark
          auto-apply
          :clearable="false"
          hide-input-icon
        />
      </div>
      <div class="filter-group">
        <label>{{ t('raceHistory.to') }}</label>
        <VueDatePicker
          v-model="toDate"
          :disabled="loading"
          dark
          auto-apply
          :clearable="false"
          hide-input-icon
        />
      </div>
    </div>

    <div v-if="loading" class="loading-state">
      {{ t('raceHistory.loadingRaces') }}
    </div>

    <div v-else-if="sortedRaces.length === 0" class="empty-state">
      {{ t('raceHistory.noRaces') }}
    </div>

    <div v-else ref="scrollContainerRef" class="table-container">
      <table class="races-table">
        <thead>
          <tr>
            <th>{{ t('raceHistory.columns.date') }}</th>
            <th>{{ t('raceHistory.columns.session') }}</th>
            <th>{{ t('raceHistory.columns.track') }}</th>
            <th>{{ t('raceHistory.columns.start') }}</th>
            <th>{{ t('raceHistory.columns.finish') }}</th>
            <th>{{ t('raceHistory.columns.incidents') }}</th>
            <th>{{ t('raceHistory.columns.license') }}</th>
            <th>{{ t('raceHistory.columns.irating') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="race in sortedRaces" :key="race.id">
            <td>{{ formatDate(race.startTime) }}</td>
            <td><SessionCell :series-name="race.seriesName" :car-id="race.carId" /></td>
            <td><TrackCell :track-id="race.trackId" /></td>
            <td><GridPosition :position="race.startPosition" :position-in-class="race.startPositionInClass" /></td>
            <td><GridPosition :position="race.finishPosition" :position-in-class="race.finishPositionInClass" /></td>
            <td>{{ race.incidents }}</td>
            <td><LicenseCell :old-license-level="race.oldLicenseLevel" :new-license-level="race.newLicenseLevel" :old-sub-level="race.oldSubLevel" :new-sub-level="race.newSubLevel" :old-cpi="race.oldCpi" :new-cpi="race.newCpi" /></td>
            <td>{{ race.newIrating }} <span :class="getIRatingDiffClass(race.oldIrating, race.newIrating)">{{ formatIRatingDiff(race.oldIrating, race.newIrating) }}</span></td>
          </tr>
        </tbody>
      </table>

      <div v-if="hasMorePages || loadingMore" ref="sentinelRef" class="scroll-sentinel">
        <span v-if="loadingMore" class="loading-more">{{ t('raceHistory.loadingMore') }}</span>
      </div>
    </div>

    <div v-if="sortedRaces.length > 0" class="table-footer">
      {{ t('raceHistory.showingRaces', { shown: sortedRaces.length, total: totalMatching }) }}
    </div>
  </div>
</template>

<style scoped>
.race-history {
  padding: 2rem;
}

.page-header {
  margin-bottom: 1.5rem;
}

.page-header h1 {
  margin: 0;
  color: var(--color-text-primary);
}

.total-races {
  font-size: 0.875rem;
  color: var(--color-text-muted);
}

.filters {
  display: flex;
  gap: 1rem;
  margin-bottom: 1rem;
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.filter-group label {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.loading-state {
  padding: 2rem;
  text-align: center;
  color: var(--color-text-secondary);
}

.table-footer {
  margin-top: 0.75rem;
  padding: 0.5rem 0;
  font-size: 0.875rem;
  color: var(--color-text-muted);
  text-align: center;
}

.empty-state {
  padding: 2rem;
  text-align: center;
  color: var(--color-text-secondary);
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: 4px;
}

.table-container {
  max-height: 550px;
  overflow-y: auto;
  border: 1px solid var(--color-border);
  border-radius: 4px;
}

.races-table {
  width: 100%;
  border-collapse: collapse;
  background: var(--color-bg-surface);
}

.races-table th,
.races-table td {
  padding: 0.75rem 1rem;
  text-align: left;
  border-bottom: 1px solid var(--color-border);
}

.races-table th {
  position: sticky;
  top: 0;
  background: var(--color-bg-elevated);
  font-weight: 600;
  color: var(--color-text-primary);
  z-index: 1;
}

.races-table tbody tr:nth-child(even) {
  background: var(--color-bg-elevated);
}

.races-table tbody tr:hover {
  background: var(--color-accent-subtle);
}

.races-table tbody tr:last-child td {
  border-bottom: none;
}

.scroll-sentinel {
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.loading-more {
  font-size: 0.875rem;
  color: var(--color-text-muted);
  padding: 0.5rem;
}

.stat-gain {
  color: #22c55e;
}

.stat-loss {
  color: #ef4444;
}

/* Mobile styles */
@media (max-width: 768px) {
  .race-history {
    padding: 1rem;
  }

  h1 {
    font-size: 1.5rem;
    margin-bottom: 1rem;
  }

  .table-container {
    max-height: 350px;
  }

  .races-table th,
  .races-table td {
    padding: 0.5rem 0.625rem;
    font-size: 0.875rem;
  }
}

@media (max-width: 480px) {
  .race-history {
    padding: 0.75rem;
  }

  .races-table th,
  .races-table td {
    padding: 0.5rem;
    font-size: 0.8125rem;
  }
}
</style>
