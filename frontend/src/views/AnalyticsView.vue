<script setup lang="ts">
defineOptions({ name: 'AnalyticsView' })

import { ref, computed, watch } from 'vue'
import { useRoute, useRouter, type LocationQuery } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAnalyticsStore } from '@/stores/analytics'
import { useDriverStore } from '@/stores/driver'
import { useCarsStore } from '@/stores/cars'
import { useTracksStore } from '@/stores/tracks'
import { useSeriesStore } from '@/stores/series'
import type { AnalyticsGroupBy } from '@/api/client'
import AnalyticsChart from '@/components/AnalyticsChart.vue'
import RaceFilters, {
  type RaceFiltersState,
  type RaceFiltersDimensions,
} from '@/components/RaceFilters.vue'
import { toDisplayPosition } from '@/utils/raceFormatters'
import '@/assets/page-layout.css'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const analyticsStore = useAnalyticsStore()
const driverStore = useDriverStore()
const carsStore = useCarsStore()
const tracksStore = useTracksStore()
const seriesStore = useSeriesStore()

const VALID_GROUP_BY: AnalyticsGroupBy[] = ['series', 'car', 'track']

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

function parseGroupByParam(v: LocationQuery[string]): AnalyticsGroupBy[] {
  const arr = Array.isArray(v) ? v : v != null ? [v] : []
  return arr
    .map((s) => String(s))
    .filter((s): s is AnalyticsGroupBy => VALID_GROUP_BY.includes(s as AnalyticsGroupBy))
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

// Seed groupBy from URL into store (before any fetches fire)
const initialGroupBy = parseGroupByParam(route.query.groupBy)
if (initialGroupBy.length > 0) {
  analyticsStore.setGroupBy(initialGroupBy)
}

// Filters are local source of truth; null until initialized from URL or driver
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

const dimensions = computed<RaceFiltersDimensions | null>(() => analyticsStore.dimensions)

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
  if (analyticsStore.groupBy.length) query.groupBy = analyticsStore.groupBy
  router.replace({ query })
}

// Push filter changes to store, fetch, and update URL
watch(
  filters,
  (next, prev) => {
    if (!next) return
    const dateChanged =
      !prev ||
      prev.from.getTime() !== next.from.getTime() ||
      prev.to.getTime() !== next.to.getTime()

    analyticsStore.setSeriesFilter(next.seriesIds)
    analyticsStore.setCarFilter(next.carIds)
    analyticsStore.setTrackFilter(next.trackIds)

    if (dateChanged) {
      analyticsStore.setDateRange({ startTime: next.from, endTime: next.to })
      analyticsStore.refresh()
    } else {
      analyticsStore.fetchAnalytics()
      analyticsStore.fetchTimeSeries()
    }

    syncUrl()
  },
  { immediate: true }
)

// Prune filter IDs that aren't in the latest dimensions (auto-cleanup after date change)
watch(
  () => analyticsStore.dimensions,
  (dims) => {
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
  }
)

// Sync URL whenever groupBy changes
watch(
  () => [...analyticsStore.groupBy],
  () => syncUrl()
)

// Group by options (available dimensions to add)
const groupByDimensions: { value: AnalyticsGroupBy; labelKey: string }[] = [
  { value: 'series', labelKey: 'analytics.groupBySeries' },
  { value: 'car', labelKey: 'analytics.groupByCar' },
  { value: 'track', labelKey: 'analytics.groupByTrack' },
]

const selectedGroupBy = computed(() => analyticsStore.groupBy)

const availableGroupBy = computed(() =>
  groupByDimensions.filter((d) => !selectedGroupBy.value.includes(d.value))
)

function addGroupBy(dimension: AnalyticsGroupBy) {
  analyticsStore.toggleGroupBy(dimension)
  if (analyticsStore.dateRange) {
    analyticsStore.fetchAnalytics()
  }
}

function removeGroupBy(dimension: AnalyticsGroupBy) {
  analyticsStore.toggleGroupBy(dimension)
  if (analyticsStore.dateRange) {
    analyticsStore.fetchAnalytics()
  }
}

// Drag and drop for reordering
const draggedIndex = ref<number | null>(null)
const dragOverIndex = ref<number | null>(null)

function onDragStart(index: number, event: DragEvent) {
  draggedIndex.value = index
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', index.toString())
  }
}

function onDragOver(index: number, event: DragEvent) {
  event.preventDefault()
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'move'
  }
  dragOverIndex.value = index
}

function onDragLeave() {
  dragOverIndex.value = null
}

function onDrop(targetIndex: number, event: DragEvent) {
  event.preventDefault()
  if (draggedIndex.value === null || draggedIndex.value === targetIndex) {
    draggedIndex.value = null
    dragOverIndex.value = null
    return
  }

  const newOrder = [...selectedGroupBy.value]
  const [removed] = newOrder.splice(draggedIndex.value, 1)
  newOrder.splice(targetIndex, 0, removed)
  analyticsStore.setGroupBy(newOrder)

  draggedIndex.value = null
  dragOverIndex.value = null

  if (analyticsStore.dateRange) {
    analyticsStore.fetchAnalytics()
  }
}

function onDragEnd() {
  draggedIndex.value = null
  dragOverIndex.value = null
}

// Computed for summary display
const summary = computed(() => analyticsStore.analytics?.summary)

// Sorting state
type SortColumn = 'group' | 'races' | 'wins' | 'podiums' | 'avgFinish' | 'iRatingDelta' | 'cpiDelta' | 'incidents'
const sortColumn = ref<SortColumn>('races')
const sortDirection = ref<'asc' | 'desc'>('desc')

function toggleSort(column: SortColumn) {
  if (sortColumn.value === column) {
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortColumn.value = column
    // Default to desc for numeric columns, asc for group name
    sortDirection.value = column === 'group' ? 'asc' : 'desc'
  }
}

function getSortValue(group: typeof groupedData.value[0], column: SortColumn): string | number {
  switch (column) {
    case 'group':
      return getGroupLabel(group)
    case 'races':
      return group.summary.raceCount
    case 'wins':
      return group.summary.wins
    case 'podiums':
      return group.summary.podiums
    case 'avgFinish':
      return group.summary.avgFinishPosition
    case 'iRatingDelta':
      return group.summary.iRatingDelta
    case 'cpiDelta':
      return group.summary.cpiDelta ?? 0
    case 'incidents':
      return group.summary.totalIncidents
    default:
      return 0
  }
}

const groupedData = computed(() => {
  const data = analyticsStore.analytics?.groupedBy ?? []
  if (data.length === 0) return data

  return [...data].sort((a, b) => {
    const aVal = getSortValue(a, sortColumn.value)
    const bVal = getSortValue(b, sortColumn.value)

    let comparison = 0
    if (typeof aVal === 'string' && typeof bVal === 'string') {
      comparison = aVal.localeCompare(bVal)
    } else {
      comparison = (aVal as number) - (bVal as number)
    }

    return sortDirection.value === 'asc' ? comparison : -comparison
  })
})

// Name lookup helpers
function getSeriesName(id: number): string {
  const series = seriesStore.getSeries(id)
  return series?.name ?? `Series ${id}`
}

function getCarName(id: number): string {
  const car = carsStore.getCar(id)
  return car?.name ?? `Car ${id}`
}

function getTrackName(id: number): string {
  const track = tracksStore.getTrack(id)
  return track?.name ?? `Track ${id}`
}

function getGroupLabel(group: { seriesId?: number; carId?: number; trackId?: number }): string {
  const parts: string[] = []
  if (group.seriesId) parts.push(getSeriesName(group.seriesId))
  if (group.carId) parts.push(getCarName(group.carId))
  if (group.trackId) parts.push(getTrackName(group.trackId))
  return parts.join(' / ') || '-'
}

function formatNumber(value: number, decimals = 0): string {
  return value.toFixed(decimals)
}

function formatDelta(value: number): string {
  const sign = value > 0 ? '+' : ''
  return `${sign}${value}`
}

function getDeltaClass(value: number): string {
  if (value > 0) return 'stat-gain'
  if (value < 0) return 'stat-loss'
  return ''
}

function getGroupByLabel(dimension: AnalyticsGroupBy): string {
  const opt = groupByDimensions.find((d) => d.value === dimension)
  return opt ? t(opt.labelKey) : dimension
}
</script>

<template>
  <div class="analytics-view page-view">
    <div class="page-header">
      <h1>{{ t('analytics.title') }}</h1>
    </div>

    <RaceFilters
      v-if="filters"
      :model-value="filters"
      :dimensions="dimensions"
      :disabled="analyticsStore.loading"
      @update:model-value="(v: RaceFiltersState) => (filters = v)"
    />

    <!-- Group By Section -->
    <div class="groupby-row">
      <span class="groupby-label">{{ t('analytics.groupBy') }}:</span>
      <div class="groupby-chips">
        <div
          v-for="(dim, index) in selectedGroupBy"
          :key="dim"
          class="groupby-chip selected"
          :class="{
            'dragging': draggedIndex === index,
            'drag-over': dragOverIndex === index && draggedIndex !== index
          }"
          draggable="true"
          @dragstart="onDragStart(index, $event)"
          @dragover="onDragOver(index, $event)"
          @dragleave="onDragLeave"
          @drop="onDrop(index, $event)"
          @dragend="onDragEnd"
        >
          <span class="drag-handle" :title="t('raceDetails.dragToReorder')">⠿</span>
          <span class="chip-label">{{ getGroupByLabel(dim) }}</span>
          <button
            class="chip-remove-btn"
            @click.stop="removeGroupBy(dim)"
            :aria-label="t('journal.actions.delete')"
          >×</button>
        </div>

        <!-- Available chips (clickable to add) -->
        <button
          v-for="opt in availableGroupBy"
          :key="opt.value"
          class="groupby-chip available"
          @click="addGroupBy(opt.value)"
          :disabled="analyticsStore.loading"
        >
          <span class="chip-add">+</span>
          <span class="chip-label">{{ t(opt.labelKey) }}</span>
        </button>
      </div>
      <span v-if="selectedGroupBy.length > 1" class="groupby-hint">
        {{ t('raceDetails.dragToReorder') }}
      </span>
    </div>

    <div v-if="analyticsStore.loading" class="loading-state">
      {{ t('analytics.loading') }}
    </div>

    <div v-else-if="!analyticsStore.dateRange" class="empty-state">
      {{ t('analytics.selectDateRange') }}
    </div>

    <div v-else-if="!summary || summary.raceCount === 0" class="empty-state">
      {{ t('analytics.noData') }}
    </div>

    <template v-else>
      <!-- Summary Cards -->
      <div class="summary-grid">
        <div class="stat-card">
          <div class="stat-value">{{ summary.raceCount }}</div>
          <div class="stat-label">{{ t('analytics.summary.races') }}</div>
        </div>

        <div class="stat-card">
          <div class="stat-value">{{ summary.wins }}</div>
          <div class="stat-label">{{ t('analytics.summary.wins') }}</div>
        </div>

        <div class="stat-card">
          <div class="stat-value">{{ summary.podiums }}</div>
          <div class="stat-label">{{ t('analytics.summary.podiums') }}</div>
        </div>

        <div class="stat-card">
          <div class="stat-value">{{ summary.top5Finishes }}</div>
          <div class="stat-label">{{ t('analytics.summary.top5') }}</div>
        </div>

        <div class="stat-card">
          <div class="stat-value">{{ formatNumber(toDisplayPosition(summary.avgFinishPosition), 1) }}</div>
          <div class="stat-label">{{ t('analytics.summary.avgFinish') }}</div>
        </div>

        <div class="stat-card">
          <div class="stat-value">{{ formatNumber(toDisplayPosition(summary.avgStartPosition), 1) }}</div>
          <div class="stat-label">{{ t('analytics.summary.avgStart') }}</div>
        </div>

        <div class="stat-card">
          <div class="stat-value" :class="getDeltaClass(summary.positionsGained)">
            {{ formatDelta(Math.round(summary.positionsGained * 10) / 10) }}
          </div>
          <div class="stat-label">{{ t('analytics.summary.positionsGained') }}</div>
        </div>

        <div class="stat-card">
          <div class="stat-value" :class="getDeltaClass(summary.iRatingDelta)">
            {{ formatDelta(summary.iRatingDelta) }}
          </div>
          <div class="stat-label">{{ t('analytics.summary.iRatingDelta') }}</div>
        </div>

        <div class="stat-card">
          <div class="stat-value">{{ summary.totalIncidents }}</div>
          <div class="stat-label">{{ t('analytics.summary.incidents') }}</div>
        </div>

        <div class="stat-card">
          <div class="stat-value">{{ formatNumber(summary.avgIncidents, 1) }}</div>
          <div class="stat-label">{{ t('analytics.summary.avgIncidents') }}</div>
        </div>
      </div>

      <!-- Grouped Data Table -->
      <div v-if="groupedData.length > 0" class="grouped-section">
        <div class="table-container">
          <table class="grouped-table">
            <thead>
              <tr>
                <th class="sortable" :class="{ sorted: sortColumn === 'group' }" @click="toggleSort('group')">
                  {{ t('analytics.columns.group') }}
                  <span class="sort-indicator">{{ sortColumn === 'group' ? (sortDirection === 'asc' ? '▲' : '▼') : '⇅' }}</span>
                </th>
                <th class="sortable" :class="{ sorted: sortColumn === 'races' }" @click="toggleSort('races')">
                  {{ t('analytics.columns.races') }}
                  <span class="sort-indicator">{{ sortColumn === 'races' ? (sortDirection === 'asc' ? '▲' : '▼') : '⇅' }}</span>
                </th>
                <th class="sortable" :class="{ sorted: sortColumn === 'wins' }" @click="toggleSort('wins')">
                  {{ t('analytics.columns.wins') }}
                  <span class="sort-indicator">{{ sortColumn === 'wins' ? (sortDirection === 'asc' ? '▲' : '▼') : '⇅' }}</span>
                </th>
                <th class="sortable" :class="{ sorted: sortColumn === 'podiums' }" @click="toggleSort('podiums')">
                  {{ t('analytics.columns.podiums') }}
                  <span class="sort-indicator">{{ sortColumn === 'podiums' ? (sortDirection === 'asc' ? '▲' : '▼') : '⇅' }}</span>
                </th>
                <th class="sortable" :class="{ sorted: sortColumn === 'avgFinish' }" @click="toggleSort('avgFinish')">
                  {{ t('analytics.columns.avgFinish') }}
                  <span class="sort-indicator">{{ sortColumn === 'avgFinish' ? (sortDirection === 'asc' ? '▲' : '▼') : '⇅' }}</span>
                </th>
                <th class="sortable" :class="{ sorted: sortColumn === 'iRatingDelta' }" @click="toggleSort('iRatingDelta')">
                  {{ t('analytics.columns.iRatingDelta') }}
                  <span class="sort-indicator">{{ sortColumn === 'iRatingDelta' ? (sortDirection === 'asc' ? '▲' : '▼') : '⇅' }}</span>
                </th>
                <th class="sortable" :class="{ sorted: sortColumn === 'cpiDelta' }" @click="toggleSort('cpiDelta')">
                  {{ t('analytics.columns.srDelta') }}
                  <span class="sort-indicator">{{ sortColumn === 'cpiDelta' ? (sortDirection === 'asc' ? '▲' : '▼') : '⇅' }}</span>
                </th>
                <th class="sortable" :class="{ sorted: sortColumn === 'incidents' }" @click="toggleSort('incidents')">
                  {{ t('analytics.columns.incidents') }}
                  <span class="sort-indicator">{{ sortColumn === 'incidents' ? (sortDirection === 'asc' ? '▲' : '▼') : '⇅' }}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(group, index) in groupedData" :key="index">
                <td class="group-name-cell">{{ getGroupLabel(group) }}</td>
                <td>{{ group.summary.raceCount }}</td>
                <td>{{ group.summary.wins }}</td>
                <td>{{ group.summary.podiums }}</td>
                <td>{{ formatNumber(toDisplayPosition(group.summary.avgFinishPosition), 1) }}</td>
                <td :class="getDeltaClass(group.summary.iRatingDelta)">
                  {{ formatDelta(group.summary.iRatingDelta) }}
                </td>
                <td :class="getDeltaClass(group.summary.cpiDelta ?? 0)">
                  {{ formatNumber(group.summary.cpiDelta ?? 0, 2) }}
                </td>
                <td>{{ group.summary.totalIncidents }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Time Series Chart -->
      <AnalyticsChart />
    </template>
  </div>
</template>

<style scoped>
.page-header {
  margin-bottom: 1.5rem;
}

.page-header h1 {
  margin: 0;
  color: var(--color-text-primary);
}

.loading-state,
.empty-state {
  padding: 2rem;
  text-align: center;
  color: var(--color-text-secondary);
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: 4px;
}

/* Group By Row */
.groupby-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.groupby-label {
  font-weight: 500;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
}

.groupby-chips {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  align-items: center;
}

.groupby-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.625rem;
  border-radius: 16px;
  font-size: 0.875rem;
  transition: all 0.15s ease;
}

.groupby-chip.selected {
  background: var(--color-accent);
  color: var(--color-accent-text);
  cursor: grab;
  user-select: none;
  border: 1px solid var(--color-accent-hover);
  font-weight: 500;
}

.groupby-chip.selected:active {
  cursor: grabbing;
}

.groupby-chip.selected.dragging {
  opacity: 0.5;
  transform: scale(0.95);
}

.groupby-chip.selected.drag-over {
  transform: scale(1.05);
  box-shadow: 0 0 0 2px var(--color-accent-muted);
}

.groupby-chip.available {
  background: var(--color-bg-surface);
  color: var(--color-text-primary);
  border: 1px dashed var(--color-text-muted);
  cursor: pointer;
}

.groupby-chip.available:hover:not(:disabled) {
  background: var(--color-bg-elevated);
  border-color: var(--color-accent);
  border-style: solid;
  color: var(--color-accent);
}

.groupby-chip.available:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.drag-handle {
  cursor: grab;
  opacity: 0.7;
  font-size: 0.75rem;
  letter-spacing: -1px;
}

.drag-handle:hover {
  opacity: 1;
}

.chip-label {
  font-weight: 500;
}

.chip-add {
  font-weight: 600;
  font-size: 1rem;
  line-height: 1;
}

.chip-remove-btn {
  background: none;
  border: none;
  color: inherit;
  opacity: 0.7;
  cursor: pointer;
  padding: 0;
  font-size: 1rem;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.chip-remove-btn:hover {
  opacity: 1;
}

.groupby-hint {
  font-size: 0.75rem;
  color: var(--color-text-muted);
  font-style: italic;
}

/* Summary Grid */
.summary-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}

.stat-card {
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 1rem;
  text-align: center;
}

.stat-value {
  font-size: 1.75rem;
  font-weight: 600;
  color: var(--color-text-primary);
  line-height: 1.2;
}

.stat-label {
  font-size: 0.75rem;
  color: var(--color-text-muted);
  margin-top: 0.25rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.stat-gain {
  color: #22c55e;
}

.stat-loss {
  color: #ef4444;
}

/* Grouped Section */
.grouped-section {
  margin-top: 2rem;
}

.table-container {
  overflow-x: auto;
  border: 1px solid var(--color-border);
  border-radius: 4px;
}

.grouped-table {
  width: 100%;
  border-collapse: collapse;
  background: var(--color-bg-surface);
}

.grouped-table th,
.grouped-table td {
  padding: 0.75rem 1rem;
  text-align: left;
  border-bottom: 1px solid var(--color-border);
}

.grouped-table th {
  background: var(--color-bg-elevated);
  font-weight: 600;
  color: var(--color-text-primary);
}

.grouped-table th.sortable {
  cursor: pointer;
  user-select: none;
  white-space: nowrap;
}

.grouped-table th.sortable:hover {
  background: var(--color-bg-surface);
}

.grouped-table th.sortable.sorted {
  color: var(--color-accent);
}

.sort-indicator {
  margin-left: 0.375rem;
  font-size: 0.75rem;
  opacity: 0.5;
}

.grouped-table th.sortable:hover .sort-indicator,
.grouped-table th.sorted .sort-indicator {
  opacity: 1;
}

.grouped-table tbody tr:nth-child(even) {
  background: var(--color-bg-elevated);
}

.grouped-table tbody tr:hover {
  background: var(--color-accent-subtle);
}

.grouped-table tbody tr:last-child td {
  border-bottom: none;
}

.group-name-cell {
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Mobile styles */
@media (max-width: 768px) {
  .analytics-view {
    padding: 1rem;
  }

  .summary-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .stat-value {
    font-size: 1.5rem;
  }

  .grouped-table th,
  .grouped-table td {
    padding: 0.5rem 0.625rem;
    font-size: 0.875rem;
  }

  .groupby-row {
    flex-direction: column;
    align-items: flex-start;
  }
}

@media (max-width: 480px) {
  .summary-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 0.5rem;
  }

  .stat-card {
    padding: 0.75rem;
  }

  .stat-value {
    font-size: 1.25rem;
  }
}
</style>