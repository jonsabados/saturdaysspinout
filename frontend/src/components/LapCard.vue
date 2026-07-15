<script setup lang="ts">
defineOptions({ name: 'LapCard' })

import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { LapData, Lap } from '@/api/client'
import { formatLapTime, formatLapDelta, toDisplayPosition } from '@/utils/raceFormatters'

const { t } = useI18n()

export interface ComparisonDriver {
  driverId: number
  driverName: string
  lapData: LapData
}

const props = defineProps<{
  driverName: string
  finishPosition: number
  lapData: LapData
  comparisonDrivers?: ComparisonDriver[]
  isDragOver?: boolean
}>()

const emit = defineEmits<{
  remove: []
}>()

const showDeltas = ref(false)

const comparisons = computed(() => props.comparisonDrivers ?? [])
const canShowDeltas = computed(() => comparisons.value.length > 0)

// Per-comparison-driver lookup of lapNumber -> lapTime for O(1) delta cells.
const comparisonLapTimes = computed(() =>
  comparisons.value.map(driver => {
    const byLapNumber = new Map<number, number>()
    for (const lap of driver.lapData.laps) {
      byLapNumber.set(lap.lapNumber, lap.lapTime)
    }
    return { driverId: driver.driverId, driverName: driver.driverName, byLapNumber }
  })
)

// Threshold for coloring a delta: 0.1s in iRacing's 1/10000ths units.
const DELTA_TENTH = 1000

interface DeltaCell {
  driverId: number
  text: string
  colorClass: string
}

function deltaCells(lap: Lap): DeltaCell[] {
  return comparisonLapTimes.value.map(({ driverId, byLapNumber }) => {
    const otherTime = byLapNumber.get(lap.lapNumber)
    // No comparable lap, or either lap is an out-lap / invalid (non-positive).
    if (otherTime === undefined || otherTime <= 0 || lap.lapTime <= 0) {
      return { driverId, text: '-', colorClass: '' }
    }
    const delta = lap.lapTime - otherTime
    let colorClass = 'delta-even'
    if (delta <= -DELTA_TENTH) colorClass = 'delta-faster'
    else if (delta >= DELTA_TENTH) colorClass = 'delta-slower'
    return { driverId, text: formatLapDelta(delta), colorClass }
  })
}

// Pace coloring: compare each lap's time to the previous *valid* lap (green ≥0.1s
// faster, grey within a tenth, red ≥0.1s slower). Out-laps / invalid laps (non-positive)
// are skipped as a baseline so a pit lap doesn't create a bogus swing.
const lapPaceByNumber = computed(() => {
  const result = new Map<number, string>()
  let prevValidTime: number | null = null
  for (const lap of props.lapData.laps) {
    if (lap.lapTime <= 0) {
      result.set(lap.lapNumber, '')
      continue
    }
    if (prevValidTime !== null) {
      const delta = lap.lapTime - prevValidTime
      if (delta <= -DELTA_TENTH) result.set(lap.lapNumber, 'pace-faster')
      else if (delta >= DELTA_TENTH) result.set(lap.lapNumber, 'pace-slower')
      else result.set(lap.lapNumber, 'pace-even')
    } else {
      result.set(lap.lapNumber, '') // first valid lap: no baseline
    }
    prevValidTime = lap.lapTime
  }
  return result
})

// Event categorization for styling
const offTrackEvents = ['off track']
const contactEvents = ['contact', 'car contact', 'lost control']

function getEventType(event: string): 'warning' | 'danger' | 'info' {
  const lower = event.toLowerCase()
  if (offTrackEvents.includes(lower)) return 'warning'
  if (contactEvents.includes(lower)) return 'danger'
  return 'info'
}

function getRowClass(lap: Lap): string {
  if (lap.personalBestLap) return 'best-lap'
  if (!lap.lapEvents || lap.lapEvents.length === 0) return ''

  // Contact/lost control = red (highest priority)
  const hasContact = lap.lapEvents.some(e => contactEvents.includes(e.toLowerCase()))
  if (hasContact) return 'incident-contact'

  // Off track = yellow
  const hasOffTrack = lap.lapEvents.some(e => offTrackEvents.includes(e.toLowerCase()))
  if (hasOffTrack) return 'incident-off-track'

  return ''
}

// Filter to show only incident-related events
function getDisplayEvents(lap: Lap): string[] {
  if (!lap.lapEvents) return []
  return lap.lapEvents.filter(e => {
    const lower = e.toLowerCase()
    return offTrackEvents.includes(lower) || contactEvents.includes(lower) || lower === 'black flag' || lower === 'tow'
  })
}
</script>

<template>
  <div
    class="lap-card"
    draggable="true"
    :class="{ 'drag-over': isDragOver }"
  >
    <div class="lap-card-header">
      <span class="drag-handle" :title="t('raceDetails.dragToReorder')">⋮⋮</span>
      <span class="lap-card-title">
        P{{ toDisplayPosition(finishPosition) }} - {{ driverName }}
      </span>
      <button
        v-if="canShowDeltas"
        type="button"
        class="lap-card-deltas-toggle"
        :class="{ active: showDeltas }"
        :aria-label="showDeltas ? t('raceDetails.hideDeltas') : t('raceDetails.showDeltas')"
        :aria-pressed="showDeltas"
        :title="showDeltas ? t('raceDetails.hideDeltas') : t('raceDetails.showDeltas')"
        @click="showDeltas = !showDeltas"
      >
        Δ
      </button>
      <button
        class="lap-card-close"
        :title="t('common.dismiss')"
        @click="emit('remove')"
      >
        &times;
      </button>
    </div>
    <div class="lap-card-summary">
      {{ lapData.laps.length }} {{ t('raceDetails.lapsAbbr') }} &middot;
      {{ t('raceDetails.columns.bestLap') }}: {{ formatLapTime(lapData.bestLapTime) }}
    </div>
    <div class="lap-card-content">
      <table class="laps-table">
        <thead>
          <tr>
            <th class="col-lap-num">#</th>
            <th class="col-lap-time">{{ t('raceDetails.lapColumns.time') }}</th>
            <th
              v-for="driver in comparisonLapTimes"
              v-show="showDeltas"
              :key="driver.driverId"
              class="col-lap-delta"
              :title="driver.driverName"
            >Δ {{ driver.driverName }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="lap in lapData.laps"
            :key="lap.lapNumber"
            :class="getRowClass(lap)"
          >
            <td class="col-lap-num">{{ lap.lapNumber }}</td>
            <td class="col-lap-time" :class="lapPaceByNumber.get(lap.lapNumber)">
              {{ formatLapTime(lap.lapTime) }}
              <span v-if="lap.personalBestLap" class="best-lap-badge">PB</span>
              <span
                v-for="event in getDisplayEvents(lap)"
                :key="event"
                class="event-badge"
                :class="`event-${getEventType(event)}`"
              >{{ event }}</span>
            </td>
            <td
              v-for="cell in deltaCells(lap)"
              v-show="showDeltas"
              :key="cell.driverId"
              class="col-lap-delta"
              :class="cell.colorClass"
            >{{ cell.text }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.lap-card {
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  overflow: hidden;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.lap-card:hover {
  border-color: var(--color-border-light);
}

.lap-card.drag-over {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 2px rgba(147, 51, 234, 0.2);
}

.lap-card-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  background: var(--color-bg-elevated);
  border-bottom: 1px solid var(--color-border);
}

.drag-handle {
  cursor: grab;
  color: var(--color-text-muted);
  font-size: 0.875rem;
  letter-spacing: -0.1em;
  user-select: none;
  padding: 0.25rem;
}

.drag-handle:active {
  cursor: grabbing;
}

.lap-card-title {
  font-weight: 600;
  font-size: 0.8125rem;
  color: var(--color-text-primary);
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.lap-card-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: var(--color-text-muted);
  font-size: 1rem;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
  flex-shrink: 0;
}

.lap-card-close:hover {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.lap-card-deltas-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  color: var(--color-text-muted);
  font-size: 0.75rem;
  line-height: 1;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
  flex-shrink: 0;
}

.lap-card-deltas-toggle:hover {
  color: var(--color-text-primary);
  border-color: var(--color-border-light);
}

.lap-card-deltas-toggle.active {
  background: var(--color-accent-subtle);
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.lap-card-summary {
  padding: 0.375rem 0.75rem;
  font-size: 0.6875rem;
  color: var(--color-text-muted);
  border-bottom: 1px solid var(--color-border);
}

.lap-card-content {
  /* Let tables be their natural height */
}

/* Laps Table */
.laps-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.8125rem;
}

.laps-table th,
.laps-table td {
  padding: 0.5rem 0.75rem;
  text-align: left;
  border-bottom: 1px solid var(--color-border);
}

.laps-table th {
  background: var(--color-bg-elevated);
  font-weight: 600;
  color: var(--color-text-primary);
  font-size: 0.6875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.laps-table tbody tr:nth-child(even) {
  background: var(--color-bg-elevated);
}

.laps-table tbody tr:last-child td {
  border-bottom: none;
}

.laps-table tbody tr.best-lap {
  background: rgba(34, 197, 94, 0.1);
}

.laps-table tbody tr.incident-contact {
  background: rgba(239, 68, 68, 0.1);
}

.laps-table tbody tr.incident-off-track {
  background: rgba(234, 179, 8, 0.15);
}

.col-lap-num {
  width: 50px;
  text-align: center;
  font-weight: 500;
}

.col-lap-time {
  font-variant-numeric: tabular-nums;
}

/* Pace vs. previous valid lap (colors the time text only; badges keep their own colors) */
.col-lap-time.pace-faster {
  color: #22c55e;
}

.col-lap-time.pace-slower {
  color: #ef4444;
}

.col-lap-time.pace-even {
  color: var(--color-text-muted);
}

.col-lap-delta {
  text-align: right;
  font-variant-numeric: tabular-nums;
  max-width: 90px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

th.col-lap-delta {
  color: var(--color-text-muted);
}

.col-lap-delta.delta-faster {
  color: #22c55e;
}

.col-lap-delta.delta-slower {
  color: #ef4444;
}

.col-lap-delta.delta-even {
  color: var(--color-text-muted);
}

.best-lap-badge,
.event-badge {
  display: inline-block;
  padding: 0.125rem 0.375rem;
  margin-left: 0.5rem;
  border-radius: 4px;
  font-size: 0.625rem;
  font-weight: 600;
  text-transform: capitalize;
}

.best-lap-badge {
  background: rgba(34, 197, 94, 0.2);
  color: #22c55e;
}

.event-badge.event-danger {
  background: rgba(239, 68, 68, 0.2);
  color: #ef4444;
}

.event-badge.event-warning {
  background: rgba(234, 179, 8, 0.2);
  color: #ca8a04;
}

.event-badge.event-info {
  background: rgba(100, 116, 139, 0.2);
  color: #64748b;
}
</style>