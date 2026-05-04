<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useCarsStore } from '@/stores/cars'
import { useTracksStore } from '@/stores/tracks'
import { useSeriesStore } from '@/stores/series'

export interface RaceFiltersState {
  from: Date
  to: Date
  discipline: string | null
  seriesIds: number[]
  carIds: number[]
  trackIds: number[]
}

export interface RaceFiltersDimensions {
  series: number[]
  cars: number[]
  tracks: number[]
}

const props = defineProps<{
  modelValue: RaceFiltersState
  dimensions: RaceFiltersDimensions | null
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: RaceFiltersState]
}>()

const { t } = useI18n()
const carsStore = useCarsStore()
const tracksStore = useTracksStore()
const seriesStore = useSeriesStore()

function formatDateForInput(date: Date | null): string {
  if (!date) return ''
  return date.toISOString().split('T')[0]
}

function parseDateFromInput(value: string): Date {
  return new Date(value + 'T00:00:00')
}

const fromDateStr = computed({
  get: () => formatDateForInput(props.modelValue.from),
  set: (value: string) => {
    if (value) {
      emit('update:modelValue', { ...props.modelValue, from: parseDateFromInput(value) })
    }
  },
})

const toDateStr = computed({
  get: () => formatDateForInput(props.modelValue.to),
  set: (value: string) => {
    if (value) {
      emit('update:modelValue', { ...props.modelValue, to: parseDateFromInput(value) })
    }
  },
})

function getSeriesName(id: number): string {
  return seriesStore.getSeries(id)?.name ?? `Series ${id}`
}

function getCarName(id: number): string {
  return carsStore.getCar(id)?.name ?? `Car ${id}`
}

function getTrackName(id: number): string {
  return tracksStore.getTrack(id)?.name ?? `Track ${id}`
}

const availableDisciplines = computed(() => {
  const carIds = props.dimensions?.cars ?? []
  const disciplines = new Set<string>()
  for (const id of carIds) {
    const car = carsStore.getCar(id)
    if (car?.categories) {
      for (const category of car.categories) {
        disciplines.add(category)
      }
    }
  }
  return Array.from(disciplines).sort()
})

const sortedSeriesOptions = computed(() => {
  const ids = props.dimensions?.series ?? []
  return [...ids].sort((a, b) => getSeriesName(a).localeCompare(getSeriesName(b)))
})

const sortedCarOptions = computed(() => {
  const ids = props.dimensions?.cars ?? []
  return [...ids].sort((a, b) => getCarName(a).localeCompare(getCarName(b)))
})

const sortedTrackOptions = computed(() => {
  const ids = props.dimensions?.tracks ?? []
  return [...ids].sort((a, b) => getTrackName(a).localeCompare(getTrackName(b)))
})

function onDisciplineChange(discipline: string | null) {
  if (!discipline) {
    emit('update:modelValue', {
      ...props.modelValue,
      discipline: null,
      seriesIds: [],
      carIds: [],
      trackIds: [],
    })
    return
  }
  const carIds = props.dimensions?.cars ?? []
  const filteredCarIds = carIds.filter((id) =>
    carsStore.getCar(id)?.categories?.includes(discipline)
  )
  const seriesIds = props.dimensions?.series ?? []
  const filteredSeriesIds = seriesIds.filter(
    (id) => seriesStore.getSeries(id)?.category === discipline
  )
  emit('update:modelValue', {
    ...props.modelValue,
    discipline,
    seriesIds: filteredSeriesIds,
    carIds: filteredCarIds,
    trackIds: [],
  })
}

function toggleId(field: 'seriesIds' | 'carIds' | 'trackIds', id: number) {
  const current = props.modelValue[field]
  const next = current.includes(id) ? current.filter((v) => v !== id) : [...current, id]
  emit('update:modelValue', { ...props.modelValue, [field]: next })
}
</script>

<template>
  <div class="filter-bar">
    <div v-if="availableDisciplines.length > 1" class="filter-group">
      <label class="filter-label">
        <span class="label-text">{{ t('raceFilters.discipline') }}</span>
        <select
          class="select-input discipline-select"
          :value="modelValue.discipline ?? ''"
          :disabled="disabled"
          @change="(e) => onDisciplineChange((e.target as HTMLSelectElement).value || null)"
        >
          <option value="">{{ t('raceFilters.allDisciplines') }}</option>
          <option v-for="d in availableDisciplines" :key="d" :value="d">
            {{ t(`raceFilters.disciplines.${d}`) }}
          </option>
        </select>
      </label>
    </div>

    <div class="filter-group date-filters">
      <label class="filter-label">
        <span class="label-text">{{ t('raceFilters.from') }}</span>
        <input
          type="date"
          v-model="fromDateStr"
          :disabled="disabled"
          class="date-input"
        />
      </label>
      <label class="filter-label">
        <span class="label-text">{{ t('raceFilters.to') }}</span>
        <input
          type="date"
          v-model="toDateStr"
          :disabled="disabled"
          class="date-input"
        />
      </label>
    </div>

    <div v-if="sortedSeriesOptions.length > 0" class="filter-group">
      <label class="filter-label">
        <span class="label-text">{{ t('raceFilters.series') }}</span>
        <select
          class="select-input"
          :disabled="disabled"
          @change="(e) => {
            const val = parseInt((e.target as HTMLSelectElement).value)
            if (!isNaN(val)) toggleId('seriesIds', val)
            ;(e.target as HTMLSelectElement).value = ''
          }"
        >
          <option value="">{{ t('raceFilters.filterAll') }}</option>
          <option v-for="id in sortedSeriesOptions" :key="id" :value="id">
            {{ modelValue.seriesIds.includes(id) ? '✓ ' : '' }}{{ getSeriesName(id) }}
          </option>
        </select>
      </label>
      <div v-if="modelValue.seriesIds.length > 0" class="active-filters">
        <span
          v-for="id in modelValue.seriesIds"
          :key="id"
          class="filter-chip"
          @click="toggleId('seriesIds', id)"
        >
          {{ getSeriesName(id) }}
          <span class="chip-remove">×</span>
        </span>
      </div>
    </div>

    <div v-if="sortedCarOptions.length > 0" class="filter-group">
      <label class="filter-label">
        <span class="label-text">{{ t('raceFilters.car') }}</span>
        <select
          class="select-input"
          :disabled="disabled"
          @change="(e) => {
            const val = parseInt((e.target as HTMLSelectElement).value)
            if (!isNaN(val)) toggleId('carIds', val)
            ;(e.target as HTMLSelectElement).value = ''
          }"
        >
          <option value="">{{ t('raceFilters.filterAll') }}</option>
          <option v-for="id in sortedCarOptions" :key="id" :value="id">
            {{ modelValue.carIds.includes(id) ? '✓ ' : '' }}{{ getCarName(id) }}
          </option>
        </select>
      </label>
      <div v-if="modelValue.carIds.length > 0" class="active-filters">
        <span
          v-for="id in modelValue.carIds"
          :key="id"
          class="filter-chip"
          @click="toggleId('carIds', id)"
        >
          {{ getCarName(id) }}
          <span class="chip-remove">×</span>
        </span>
      </div>
    </div>

    <div v-if="sortedTrackOptions.length > 0" class="filter-group">
      <label class="filter-label">
        <span class="label-text">{{ t('raceFilters.track') }}</span>
        <select
          class="select-input"
          :disabled="disabled"
          @change="(e) => {
            const val = parseInt((e.target as HTMLSelectElement).value)
            if (!isNaN(val)) toggleId('trackIds', val)
            ;(e.target as HTMLSelectElement).value = ''
          }"
        >
          <option value="">{{ t('raceFilters.filterAll') }}</option>
          <option v-for="id in sortedTrackOptions" :key="id" :value="id">
            {{ modelValue.trackIds.includes(id) ? '✓ ' : '' }}{{ getTrackName(id) }}
          </option>
        </select>
      </label>
      <div v-if="modelValue.trackIds.length > 0" class="active-filters">
        <span
          v-for="id in modelValue.trackIds"
          :key="id"
          class="filter-chip"
          @click="toggleId('trackIds', id)"
        >
          {{ getTrackName(id) }}
          <span class="chip-remove">×</span>
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.filter-group {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.select-input {
  padding: 0.5rem;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-bg-surface);
  color: var(--color-text-primary);
  font-size: 0.875rem;
  min-width: 150px;
}

.select-input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.active-filters {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  margin-top: 0.375rem;
  max-height: 150px;
  overflow-y: auto;
  padding-right: 0.25rem;
}

.filter-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.25rem 0.5rem;
  background: var(--color-accent-subtle);
  color: var(--color-accent);
  border-radius: 12px;
  font-size: 0.75rem;
  cursor: pointer;
  transition: background 0.15s ease;
}

.filter-chip:hover {
  background: var(--color-accent);
  color: var(--color-accent-text);
}

.chip-remove {
  font-weight: 600;
  opacity: 0.7;
}

.filter-chip:hover .chip-remove {
  opacity: 1;
}

@media (max-width: 768px) {
  .filter-group {
    width: 100%;
    max-width: 100%;
  }

  .select-input {
    width: 100%;
    max-width: 100%;
    min-width: 0;
  }

  .active-filters {
    max-width: 100%;
  }

  .filter-chip {
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
</style>