<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useApiClient, type Race } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { useWebSocketStore } from '@/stores/websocket'
import GridPosition from '@/components/GridPosition.vue'

const apiClient = useApiClient()
const auth = useAuthStore()
const wsStore = useWebSocketStore()

const races = ref<Race[]>([])
const displayCount = ref(15)
const pageSize = 15
const sentinelRef = ref<HTMLElement | null>(null)
const scrollContainerRef = ref<HTMLElement | null>(null)
const observer = ref<IntersectionObserver | null>(null)
const userHasScrolled = ref(false)

const sortedRaces = computed(() =>
  [...races.value].sort((a, b) => new Date(b.startTime).getTime() - new Date(a.startTime).getTime()),
)

const visibleRaces = computed(() => sortedRaces.value.slice(0, displayCount.value))

const hasMore = computed(() => displayCount.value < sortedRaces.value.length)

function loadMore() {
  displayCount.value += pageSize
}

interface RaceIngestedPayload {
  raceId: number
}

async function handleRaceIngested(payload: unknown) {
  const { raceId } = payload as RaceIngestedPayload
  if (!auth.userId) {
    console.warn('[RaceHistory] Received race ingested event but no userId')
    return
  }

  console.log('[RaceHistory] Race ingested, fetching:', raceId)
  try {
    const response = await apiClient.getRace(auth.userId, raceId)
    if (!races.value.some((r) => r.id === response.response.id)) {
      races.value.push(response.response)
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

function formatCpiDiff(oldCpi: number, newCpi: number): string {
  const diff = newCpi - oldCpi
  const sign = diff > 0 ? '+' : ''
  return `(${sign}${diff.toFixed(2)})`
}

function getCpiDiffClass(oldCpi: number, newCpi: number): string {
  const diff = newCpi - oldCpi
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
      if (entries[0].isIntersecting && hasMore.value && userHasScrolled.value) {
        loadMore()
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
    <h1>Race History</h1>

    <div v-if="sortedRaces.length === 0" class="empty-state">
      No races yet. Races will appear here as they are ingested.
    </div>

    <div v-else ref="scrollContainerRef" class="table-container">
      <table class="races-table">
        <thead>
          <tr>
            <th>Date</th>
            <th>Car</th>
            <th>Track</th>
            <th>Start</th>
            <th>Finish</th>
            <th>Incidents</th>
            <th title="Corners Per Incident">CPI</th>
            <th>iRating</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="race in visibleRaces" :key="race.id">
            <td>{{ formatDate(race.startTime) }}</td>
            <td>{{ race.carId }}</td>
            <td>{{ race.trackId }}</td>
            <td><GridPosition :position="race.startPosition" :position-in-class="race.startPositionInClass" /></td>
            <td><GridPosition :position="race.finishPosition" :position-in-class="race.finishPositionInClass" /></td>
            <td>{{ race.incidents }}</td>
            <td>{{ race.newCpi.toFixed(2) }} <span :class="getCpiDiffClass(race.oldCpi, race.newCpi)">{{ formatCpiDiff(race.oldCpi, race.newCpi) }}</span></td>
            <td>{{ race.newIrating }} <span :class="getIRatingDiffClass(race.oldIrating, race.newIrating)">{{ formatIRatingDiff(race.oldIrating, race.newIrating) }}</span></td>
          </tr>
        </tbody>
      </table>

      <div v-if="hasMore" ref="sentinelRef" class="scroll-sentinel"></div>
    </div>
  </div>
</template>

<style scoped>
.race-history {
  padding: 2rem;
  max-width: 1000px;
  margin: 0 auto;
}

h1 {
  margin-bottom: 1.5rem;
  color: var(--color-text-primary);
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
  background: var(--color-bg-elevated);
  font-weight: 600;
  color: var(--color-text-primary);
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
  height: 1px;
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
