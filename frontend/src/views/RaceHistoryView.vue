<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useApiClient, type Race } from '@/api/client'
import { useSessionStore } from '@/stores/session'
import { useAuthStore } from '@/stores/auth'
import { useWebSocketStore } from '@/stores/websocket'
import { useRaceIngestionStore } from '@/stores/raceIngestion'

const apiClient = useApiClient()
const session = useSessionStore()
const auth = useAuthStore()
const wsStore = useWebSocketStore()
const ingestionStore = useRaceIngestionStore()

const races = ref<Race[]>([])

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
    // Prepend to show newest first, avoid duplicates
    if (!races.value.some((r) => r.id === response.response.id)) {
      races.value.unshift(response.response)
    }
  } catch (err) {
    console.error('[RaceHistory] Failed to fetch ingested race:', err)
  }
}

function formatDate(isoString: string): string {
  return new Date(isoString).toLocaleDateString()
}

function formatIRatingChange(oldRating: number, newRating: number): string {
  const diff = newRating - oldRating
  const sign = diff >= 0 ? '+' : ''
  return `${newRating} (${sign}${diff})`
}

onMounted(() => {
  wsStore.on('raceIngested', handleRaceIngested)

  if (session.isReady) {
    ingestionStore.triggerIngestion()
  }
})

onUnmounted(() => {
  wsStore.off('raceIngested', handleRaceIngested)
})

watch(
  () => session.isReady,
  (ready) => {
    if (ready && ingestionStore.status === 'idle') {
      ingestionStore.triggerIngestion()
    }
  },
)
</script>

<template>
  <div class="race-history">
    <h1>Race History</h1>

    <div v-if="races.length === 0" class="empty-state">
      No races yet. Races will appear here as they are ingested.
    </div>

    <table v-else class="races-table">
      <thead>
        <tr>
          <th>Date</th>
          <th>Track</th>
          <th>Start</th>
          <th>Finish</th>
          <th>Incidents</th>
          <th>iRating</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="race in races" :key="race.id">
          <td>{{ formatDate(race.startTime) }}</td>
          <td>{{ race.trackId }}</td>
          <td>{{ race.startPosition }}</td>
          <td>{{ race.finishPosition }}</td>
          <td>{{ race.incidents }}</td>
          <td>{{ formatIRatingChange(race.oldIrating, race.newIrating) }}</td>
        </tr>
      </tbody>
    </table>
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

.races-table {
  width: 100%;
  border-collapse: collapse;
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: 4px;
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

.races-table tbody tr:hover {
  background: var(--color-bg-elevated);
}

.races-table tbody tr:last-child td {
  border-bottom: none;
}
</style>
