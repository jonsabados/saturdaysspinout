<script setup lang="ts">
defineOptions({ name: 'LapCard' })

import { useI18n } from 'vue-i18n'
import type { LapData, Lap } from '@/api/client'
import { formatLapTime, toDisplayPosition } from '@/utils/raceFormatters'

const { t } = useI18n()

const props = defineProps<{
  driverName: string
  finishPosition: number
  lapData: LapData
  isDragOver?: boolean
}>()

const emit = defineEmits<{
  remove: []
}>()

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
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="lap in lapData.laps"
            :key="lap.lapNumber"
            :class="getRowClass(lap)"
          >
            <td class="col-lap-num">{{ lap.lapNumber }}</td>
            <td class="col-lap-time">
              {{ formatLapTime(lap.lapTime) }}
              <span v-if="lap.personalBestLap" class="best-lap-badge">PB</span>
              <span
                v-for="event in getDisplayEvents(lap)"
                :key="event"
                class="event-badge"
                :class="`event-${getEventType(event)}`"
              >{{ event }}</span>
            </td>
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