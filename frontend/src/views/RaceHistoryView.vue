<script setup lang="ts">
defineOptions({ name: 'RaceHistoryView' })

import { ref, computed, watch, onMounted, onUnmounted, onActivated } from 'vue'
import { useRoute, useRouter, type LocationQuery } from 'vue-router'
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
import RaceFilters, {
  type RaceFiltersState,
  type RaceFiltersDimensions,
} from '@/components/RaceFilters.vue'
import '@/assets/page-layout.css'

const { t } = useI18n()
const route = useRoute()
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

const dimensions = ref<RaceFiltersDimensions | null>(null)

function formatDateForInput(date: Date): string {
  return date.toISOString().split('T')[0]
}

function parseDateFromInput(value: string): Date {
  return new Date(value + 'T00:00:00')
}

function parseIdParam(v: LocationQuery[string]): number[] {
  const arr = Array.isArray(v) ? v : v != null ? [v] : []
  return arr
    .map((s) => parseInt(String(s), 10))
    .filter((n) => !isNaN(n))
}

function initFiltersFromUrl(): RaceFiltersState | null {
  const q = route.query
  const fromStr = typeof q.from === 'string' ? q.from : null
  const toStr = typeof q.to === 'string' ? q.to : null
  if (!fromStr || !toStr) return null
  return {
    from: parseDateFromInput(fromStr),
    to: parseDateFromInput(toStr),
    discipline: typeof q.discipline === 'string' ? q.discipline : null,
    seriesIds: parseIdParam(q.seriesId),
    carIds: parseIdParam(q.carId),
    trackIds: parseIdParam(q.trackId),
  }
}

const filters = ref<RaceFiltersState | null>(initFiltersFromUrl())

// Default from driver memberSince once it loads (only if URL didn't seed filters)
watch(
  () => driverStore.driver,
  (driver) => {
    if (!filters.value && driver?.memberSince) {
      filters.value = {
        from: new Date(driver.memberSince),
        to: new Date(),
        discipline: null,
        seriesIds: [],
        carIds: [],
        trackIds: [],
      }
    }
  },
  { immediate: true }
)

function syncUrl() {
  if (!filters.value) return
  const f = filters.value
  const query: Record<string, string | string[]> = {
    from: formatDateForInput(f.from),
    to: formatDateForInput(f.to),
  }
  if (f.discipline) query.discipline = f.discipline
  if (f.seriesIds.length) query.seriesId = f.seriesIds.map(String)
  if (f.carIds.length) query.carId = f.carIds.map(String)
  if (f.trackIds.length) query.trackId = f.trackIds.map(String)
  router.replace({ query })
}

function currentFilterOptions() {
  if (!filters.value) return undefined
  const f = filters.value
  return {
    seriesIds: f.seriesIds.length > 0 ? f.seriesIds : undefined,
    carIds: f.carIds.length > 0 ? f.carIds : undefined,
    trackIds: f.trackIds.length > 0 ? f.trackIds : undefined,
  }
}

async function fetchDimensions() {
  if (!auth.userId || !filters.value) return
  try {
    dimensions.value = await apiClient.getAnalyticsDimensions(
      auth.userId,
      filters.value.from,
      filters.value.to
    )
  } catch (err) {
    console.error('[RaceHistory] Failed to fetch dimensions:', err)
  }
}

async function fetchRaces() {
  if (!auth.userId || !filters.value) return

  loading.value = true
  currentPage.value = 1
  try {
    const response = await apiClient.getRaces(
      auth.userId,
      filters.value.from,
      filters.value.to,
      1,
      pageSize,
      currentFilterOptions()
    )
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
  if (!auth.userId || !filters.value || loadingMore.value || !hasMorePages.value) return

  loadingMore.value = true
  const nextPage = currentPage.value + 1
  try {
    const response = await apiClient.getRaces(
      auth.userId,
      filters.value.from,
      filters.value.to,
      nextPage,
      pageSize,
      currentFilterOptions()
    )
    races.value = [...races.value, ...response.items]
    currentPage.value = nextPage
    hasMorePages.value = response.pagination.page < response.pagination.totalPages
  } catch (err) {
    console.error('[RaceHistory] Failed to fetch more races:', err)
  } finally {
    loadingMore.value = false
  }
}

// Refetch on any filter change; refetch dimensions on date change
watch(
  filters,
  (next, prev) => {
    if (!next) return
    const dateChanged =
      !prev ||
      prev.from.getTime() !== next.from.getTime() ||
      prev.to.getTime() !== next.to.getTime()

    if (dateChanged) {
      fetchDimensions()
    }
    fetchRaces()
    syncUrl()
  },
  { immediate: true }
)

// Prune filter IDs that aren't in the latest dimensions
watch(dimensions, (dims) => {
  if (!dims || !filters.value) return
  const seriesSet = new Set(dims.series)
  const carsSet = new Set(dims.cars)
  const tracksSet = new Set(dims.tracks)

  const prunedSeries = filters.value.seriesIds.filter((id) => seriesSet.has(id))
  const prunedCars = filters.value.carIds.filter((id) => carsSet.has(id))
  const prunedTracks = filters.value.trackIds.filter((id) => tracksSet.has(id))

  const changed =
    prunedSeries.length !== filters.value.seriesIds.length ||
    prunedCars.length !== filters.value.carIds.length ||
    prunedTracks.length !== filters.value.trackIds.length

  if (changed) {
    filters.value = {
      ...filters.value,
      seriesIds: prunedSeries,
      carIds: prunedCars,
      trackIds: prunedTracks,
    }
  }
})

const sortedRaces = computed(() =>
  [...races.value].sort((a, b) => new Date(b.startTime).getTime() - new Date(a.startTime).getTime()),
)

interface RaceIngestedPayload {
  raceId: number
}

function raceMatchesActiveFilters(race: Race): boolean {
  if (!filters.value) return false
  const f = filters.value
  const raceTime = new Date(race.startTime)
  if (raceTime < f.from || raceTime > f.to) return false
  if (f.seriesIds.length > 0 && !f.seriesIds.includes(race.seriesId)) return false
  if (f.carIds.length > 0 && !f.carIds.includes(race.carId)) return false
  if (f.trackIds.length > 0 && !f.trackIds.includes(race.trackId)) return false
  return true
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

    driverStore.incrementSessionCount()

    if (raceMatchesActiveFilters(race) && !races.value.some((r) => r.id === race.id)) {
      races.value.push(race)
      totalMatching.value++
      userHasScrolled.value = true
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

    <RaceFilters
      v-if="filters"
      :model-value="filters"
      :dimensions="dimensions"
      :disabled="loading"
      @update:model-value="(v: RaceFiltersState) => (filters = v)"
    />

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
                :label="t('raceHistory.actions.view')"
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