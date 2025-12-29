<script setup lang="ts">
defineOptions({ name: 'RaceDetailsView' })

import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useApiClient, type Session, type SessionSimResult, type SessionDriverResult } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { useTracksStore } from '@/stores/tracks'
import GridPosition from '@/components/GridPosition.vue'
import LicenseCell from '@/components/LicenseCell.vue'
import { formatLapTime, formatInterval } from '@/utils/raceFormatters'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const apiClient = useApiClient()
const authStore = useAuthStore()
const tracksStore = useTracksStore()

const currentUserId = computed(() => authStore.userId)

const session = ref<Session | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const selectedSessionIndex = ref(0)

const subsessionId = computed(() => {
  const id = route.params.subsessionId
  return typeof id === 'string' ? parseInt(id, 10) : NaN
})

const track = computed(() => {
  if (!session.value) return undefined
  return tracksStore.getTrack(session.value.track.trackId)
})

const trackFullName = computed(() => {
  if (!session.value) return ''
  const t = session.value.track
  return t.configName ? `${t.trackName} - ${t.configName}` : t.trackName
})

const sessionTypes = computed(() => {
  if (!session.value) return []
  return session.value.sessionResults.map((sr, index) => ({
    index,
    name: sr.simsessionName,
    typeName: sr.simsessionTypeName,
    driverCount: sr.results.length,
  }))
})

const selectedSession = computed((): SessionSimResult | null => {
  if (!session.value || !session.value.sessionResults.length) return null
  return session.value.sessionResults[selectedSessionIndex.value] || null
})

const sortedResults = computed((): SessionDriverResult[] => {
  if (!selectedSession.value) return []
  return [...selectedSession.value.results].sort((a, b) => a.finishPosition - b.finishPosition)
})

// simsessionType 6 = Race in iRacing
const isRaceSession = computed(() => selectedSession.value?.simsessionType === 6)

function formatDateTime(isoString: string): string {
  const date = new Date(isoString)
  return date.toLocaleDateString(undefined, {
    weekday: 'short',
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
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

function goBack() {
  router.back()
}

function selectSession(index: number) {
  selectedSessionIndex.value = index
}

onMounted(async () => {
  if (isNaN(subsessionId.value)) {
    error.value = t('raceDetails.invalidSubsessionId')
    loading.value = false
    return
  }

  try {
    const response = await apiClient.getSession(subsessionId.value)
    session.value = response.response
    // Default to the last session (usually the race)
    if (session.value.sessionResults.length > 0) {
      selectedSessionIndex.value = session.value.sessionResults.length - 1
    }
  } catch (err) {
    console.error('[RaceDetails] Failed to fetch session:', err)
    error.value = err instanceof Error ? err.message : t('raceDetails.fetchError')
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="race-details">
    <button class="back-button" @click="goBack">
      &larr; {{ t('common.back') }}
    </button>

    <div v-if="loading" class="loading-state">
      {{ t('common.loading') }}
    </div>

    <div v-else-if="error" class="error-state">
      {{ error }}
    </div>

    <template v-else-if="session">
      <!-- Header -->
      <header class="race-header">
        <div class="race-title">
          <h1>{{ session.seriesName }}</h1>
          <RouterLink :to="{ name: 'track-details', params: { id: session.track.trackId } }" class="track-name">
            {{ trackFullName }}
          </RouterLink>
          <span class="race-date">{{ formatDateTime(session.startTime) }}</span>
        </div>
        <div v-if="track?.smallImageUrl" class="track-thumbnail">
          <img :src="track.smallImageUrl" :alt="trackFullName" />
        </div>
      </header>

      <!-- Stats Bar -->
      <div class="stats-bar">
        <div class="stat-item">
          <span class="stat-label">{{ t('raceDetails.sof') }}</span>
          <span class="stat-value">{{ session.eventStrengthOfField.toLocaleString() }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">{{ t('raceDetails.drivers') }}</span>
          <span class="stat-value">{{ session.numDrivers }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">{{ t('raceDetails.laps') }}</span>
          <span class="stat-value">{{ session.eventLapsComplete }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">{{ t('raceDetails.leadChanges') }}</span>
          <span class="stat-value">{{ session.numLeadChanges }}</span>
        </div>
        <div v-if="session.numCautions > 0" class="stat-item">
          <span class="stat-label">{{ t('raceDetails.cautions') }}</span>
          <span class="stat-value">{{ session.numCautions }} ({{ session.numCautionLaps }} {{ t('raceDetails.lapsAbbr') }})</span>
        </div>
      </div>

      <!-- Weather Info (collapsible) -->
      <details class="weather-section">
        <summary>{{ t('raceDetails.weather') }}</summary>
        <div class="weather-grid">
          <div class="weather-item">
            <span class="weather-label">{{ t('raceDetails.temperature') }}</span>
            <span class="weather-value">{{ session.weather.tempValue }}°C</span>
          </div>
          <div class="weather-item">
            <span class="weather-label">{{ t('raceDetails.humidity') }}</span>
            <span class="weather-value">{{ session.weather.relHumidity }}%</span>
          </div>
          <div class="weather-item">
            <span class="weather-label">{{ t('raceDetails.wind') }}</span>
            <span class="weather-value">{{ session.weather.windValue }} km/h</span>
          </div>
        </div>
      </details>

      <!-- Session Type Tabs -->
      <div class="session-tabs">
        <button
          v-for="st in sessionTypes"
          :key="st.index"
          :class="['session-tab', { active: selectedSessionIndex === st.index }]"
          @click="selectSession(st.index)"
        >
          {{ st.name || st.typeName }}
          <span class="driver-count">({{ st.driverCount }})</span>
        </button>
      </div>

      <!-- Results Table -->
      <div v-if="selectedSession" class="results-section">
        <div class="table-container">
          <table class="results-table">
            <thead>
              <tr>
                <th class="col-position">{{ t('raceDetails.columns.pos') }}</th>
                <th class="col-driver">{{ t('raceDetails.columns.driver') }}</th>
                <th class="col-car">{{ t('raceDetails.columns.car') }}</th>
                <th v-if="isRaceSession" class="col-start">{{ t('raceDetails.columns.start') }}</th>
                <th v-if="isRaceSession" class="col-interval">{{ t('raceDetails.columns.interval') }}</th>
                <th class="col-laps">{{ t('raceDetails.columns.laps') }}</th>
                <th v-if="isRaceSession" class="col-led">{{ t('raceDetails.columns.led') }}</th>
                <th class="col-best-lap">{{ t('raceDetails.columns.bestLap') }}</th>
                <th class="col-avg-lap">{{ t('raceDetails.columns.avgLap') }}</th>
                <th class="col-incidents">{{ t('raceDetails.columns.incidents') }}</th>
                <th v-if="isRaceSession" class="col-license">{{ t('columns.license') }}</th>
                <th v-if="isRaceSession" class="col-irating">{{ t('columns.irating') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(driver, index) in sortedResults"
                :key="driver.custId"
                :class="{ 'current-user': driver.custId === currentUserId }"
              >
                <td class="col-position">
                  <GridPosition
                    :position="driver.finishPosition"
                    :position-in-class="driver.finishPositionInClass"
                  />
                </td>
                <td class="col-driver">
                  <span class="driver-name">{{ driver.displayName }}</span>
                  <span v-if="driver.ai" class="ai-badge">AI</span>
                </td>
                <td class="col-car">
                  <span class="car-number">#{{ driver.livery.carNumber }}</span>
                  <RouterLink :to="{ name: 'car-details', params: { id: driver.carId } }" class="car-link">
                    {{ driver.carName }}
                  </RouterLink>
                </td>
                <td v-if="isRaceSession" class="col-start">
                  <GridPosition
                    :position="driver.startingPosition"
                    :position-in-class="driver.startingPositionInClass"
                  />
                </td>
                <td v-if="isRaceSession" class="col-interval">{{ formatInterval(driver.interval, index === 0, driver.lapsComplete, sortedResults[0]?.lapsComplete ?? 0) }}</td>
                <td class="col-laps">{{ driver.lapsComplete }}</td>
                <td v-if="isRaceSession" class="col-led">{{ driver.lapsLead || '-' }}</td>
                <td class="col-best-lap">{{ formatLapTime(driver.bestLapTime) }}</td>
                <td class="col-avg-lap">{{ formatLapTime(driver.averageLap) }}</td>
                <td class="col-incidents">{{ driver.incidents }}</td>
                <td v-if="isRaceSession" class="col-license">
                  <LicenseCell
                    :old-license-level="driver.oldLicenseLevel"
                    :new-license-level="driver.newLicenseLevel"
                    :old-sub-level="driver.oldSubLevel"
                    :new-sub-level="driver.newSubLevel"
                    :old-cpi="driver.oldCpi"
                    :new-cpi="driver.newCpi"
                  />
                </td>
                <td v-if="isRaceSession" class="col-irating">
                  {{ driver.newIrating }}
                  <span :class="getIRatingDiffClass(driver.oldIrating, driver.newIrating)">
                    {{ formatIRatingDiff(driver.oldIrating, driver.newIrating) }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.race-details {
  padding: 2rem;
  max-width: 1400px;
  margin: 0 auto;
}

.back-button {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.5rem 1rem;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s, background 0.15s;
  margin-bottom: 1.5rem;
}

.back-button:hover {
  color: var(--color-text-primary);
  border-color: var(--color-border-light);
  background: var(--color-accent-subtle);
}

.loading-state,
.error-state {
  padding: 3rem;
  text-align: center;
  color: var(--color-text-secondary);
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
}

.error-state {
  color: #ef4444;
}

/* Header */
.race-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1.5rem;
  margin-bottom: 1.5rem;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.race-title h1 {
  margin: 0;
  font-size: 1.75rem;
  color: var(--color-text-primary);
}

.track-name {
  display: block;
  font-size: 1.125rem;
  margin-top: 0.25rem;
}

.race-date {
  display: block;
  font-size: 0.875rem;
  color: var(--color-text-muted);
  margin-top: 0.5rem;
}

.track-thumbnail img {
  width: 120px;
  border-radius: 4px;
  border: 1px solid var(--color-border);
}

/* Stats Bar */
.stats-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  margin-bottom: 1.5rem;
  padding: 1rem;
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  padding: 0.5rem 1rem;
  min-width: 100px;
}

.stat-label {
  font-size: 0.75rem;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.stat-value {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--color-text-primary);
}

/* Session Tabs */
.session-tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1rem;
  border-bottom: 1px solid var(--color-border);
  padding-bottom: 0;
}

.session-tab {
  padding: 0.75rem 1.25rem;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
}

.session-tab:hover {
  color: var(--color-text-primary);
}

.session-tab.active {
  color: var(--color-text-primary);
  border-bottom-color: var(--color-accent);
}

.driver-count {
  font-size: 0.75rem;
  color: var(--color-text-muted);
  margin-left: 0.25rem;
}

/* Results Table */
.results-section {
  margin-bottom: 1.5rem;
}

.table-container {
  overflow-x: auto;
  border: 1px solid var(--color-border);
  border-radius: 8px;
}

.results-table {
  width: 100%;
  border-collapse: collapse;
  background: var(--color-bg-surface);
  font-size: 0.875rem;
}

.results-table th,
.results-table td {
  padding: 0.625rem 0.75rem;
  text-align: left;
  border-bottom: 1px solid var(--color-border);
  white-space: nowrap;
}

.results-table th {
  position: sticky;
  top: 0;
  background: var(--color-bg-elevated);
  font-weight: 600;
  color: var(--color-text-primary);
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.results-table tbody tr:nth-child(even) {
  background: var(--color-bg-elevated);
}

.results-table tbody tr:hover {
  background: var(--color-accent-subtle);
}

.results-table tbody tr.current-user {
  background: rgba(59, 130, 246, 0.15);
}

.results-table tbody tr.current-user:hover {
  background: rgba(59, 130, 246, 0.25);
}

.results-table tbody tr:last-child td {
  border-bottom: none;
}

.col-position,
.col-start,
.col-laps,
.col-led,
.col-incidents {
  text-align: center;
}

.col-interval,
.col-best-lap,
.col-avg-lap,
.col-irating {
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.driver-name {
  font-weight: 500;
}

.ai-badge {
  display: inline-block;
  padding: 0.125rem 0.375rem;
  margin-left: 0.5rem;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  font-size: 0.625rem;
  color: var(--color-text-muted);
  text-transform: uppercase;
}

.car-number {
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-right: 0.5rem;
}

.stat-gain {
  color: #22c55e;
}

.stat-loss {
  color: #ef4444;
}

/* Weather Section */
.weather-section {
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 1rem;
  margin-bottom: 1.5rem;
}

.weather-section summary {
  cursor: pointer;
  font-weight: 500;
  color: var(--color-text-primary);
}

.weather-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 2rem;
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--color-border);
}

.weather-item {
  display: flex;
  flex-direction: column;
}

.weather-label {
  font-size: 0.75rem;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.weather-value {
  font-size: 1rem;
  color: var(--color-text-primary);
}

/* Mobile */
@media (max-width: 768px) {
  .race-details {
    padding: 1rem;
  }

  .race-header {
    flex-direction: column-reverse;
    align-items: center;
    text-align: center;
  }

  .race-title h1 {
    font-size: 1.25rem;
  }

  .stats-bar {
    justify-content: center;
  }

  .session-tabs {
    overflow-x: auto;
  }

  .results-table {
    font-size: 0.8125rem;
  }

  .results-table th,
  .results-table td {
    padding: 0.5rem;
  }
}
</style>