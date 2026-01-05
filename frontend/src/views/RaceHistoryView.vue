<script setup lang="ts">
defineOptions({ name: 'RaceHistoryView' })

import { ref, computed, watch, onMounted, onUnmounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useApiClient, type Race } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { useWebSocketStore } from '@/stores/websocket'
import { useDriverStore } from '@/stores/driver'
import GridPosition from '@/components/GridPosition.vue'
import TrackCell from '@/components/TrackCell.vue'
import SessionCell from '@/components/SessionCell.vue'
import LicenseCell from '@/components/LicenseCell.vue'
import RowActionButton from '@/components/RowActionButton.vue'
import '@/assets/page-layout.css'

const { t } = useI18n()
const router = useRouter()
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
const savedScrollTop = ref(0)
const lastScrollSave = ref(0)

// Date filters
const fromDate = ref<Date | null>(null)
const toDate = ref<Date | null>(null)
const initialized = ref(false)

// Format date for input[type="date"]
function formatDateForInput(date: Date | null): string {
  if (!date) return ''
  return date.toISOString().split('T')[0]
}

// Parse date from input[type="date"]
function parseDateFromInput(value: string): Date {
  return new Date(value + 'T00:00:00')
}

// Computed properties for native date input binding
const fromDateStr = computed({
  get: () => formatDateForInput(fromDate.value),
  set: (value: string) => {
    if (value) {
      fromDate.value = parseDateFromInput(value)
    }
  },
})

const toDateStr = computed({
  get: () => formatDateForInput(toDate.value),
  set: (value: string) => {
    if (value) {
      toDate.value = parseDateFromInput(value)
    }
  },
})

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
      // Enable infinite scroll so table auto-fills during ingestion
      userHasScrolled.value = true
      // Refresh from API to get accurate pagination (other races may have been ingested)
      if (races.value.length < pageSize && !loading.value) {
        fetchRaces()
      }
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

function viewRaceDetails(subsessionId: number) {
  router.push({ name: 'race-details', params: { subsessionId } })
}

// Note: Unlike the original implementation, we keep this listener attached (not removed after first scroll)
// because we need continuous scroll position saves for restoration after navigation.
// Throttling mitigates the performance impact.
function handleScroll() {
  userHasScrolled.value = true
  const now = Date.now()
  if (now - lastScrollSave.value > 100 && scrollContainerRef.value) {
    savedScrollTop.value = scrollContainerRef.value.scrollTop
    lastScrollSave.value = now
  }
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

onActivated(() => {
  requestAnimationFrame(() => {
    if (scrollContainerRef.value && savedScrollTop.value > 0) {
      scrollContainerRef.value.scrollTop = savedScrollTop.value
    }
  })
})
</script>

<template>
  <div class="race-history page-view">
    <div class="page-header">
      <h1>{{ t('raceHistory.title') }}</h1>
      <span v-if="driverStore.driver?.sessionCount" class="total-races">
        {{ t('raceHistory.racesOnFile', { count: driverStore.driver.sessionCount }) }}
      </span>
    </div>

    <div class="filter-bar">
      <div class="filter-group date-filters">
        <label class="filter-label">
          <span class="label-text">{{ t('raceHistory.from') }}</span>
          <input
            type="date"
            v-model="fromDateStr"
            :disabled="loading"
            class="date-input"
          />
        </label>
        <label class="filter-label">
          <span class="label-text">{{ t('raceHistory.to') }}</span>
          <input
            type="date"
            v-model="toDateStr"
            :disabled="loading"
            class="date-input"
          />
        </label>
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
            <th class="col-actions"></th>
            <th>{{ t('raceHistory.columns.date') }}</th>
            <th>{{ t('raceHistory.columns.session') }}</th>
            <th>{{ t('raceHistory.columns.track') }}</th>
            <th>{{ t('columns.start') }}</th>
            <th>{{ t('raceHistory.columns.finish') }}</th>
            <th>{{ t('raceHistory.columns.incidents') }}</th>
            <th>{{ t('columns.license') }}</th>
            <th>{{ t('columns.irating') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="race in sortedRaces" :key="race.id">
            <td class="col-actions">
              <RowActionButton
                direction="right"
                :title="t('raceHistory.viewDetails')"
                @click="viewRaceDetails(race.subsessionId)"
              />
            </td>
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
/* View-specific styles (shared layout from page-layout.css) */
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

.col-actions {
  width: 32px;
  padding: 0.25rem 0.5rem !important;
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
