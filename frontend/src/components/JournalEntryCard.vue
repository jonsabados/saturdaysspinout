<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import type { JournalEntry } from '@/api/client'
import { useTracksStore } from '@/stores/tracks'
import { getSentiment } from '@/utils/tagHelpers'
import { toDisplayPosition } from '@/utils/raceFormatters'
import SentimentBadge from './SentimentBadge.vue'

const props = defineProps<{
  entry: JournalEntry
}>()

const { t, d } = useI18n()
const tracksStore = useTracksStore()

const sentiment = computed(() => getSentiment(props.entry.tags))

const track = computed(() => {
  return tracksStore.getTrack(props.entry.race.trackId)
})

const trackName = computed(() => {
  if (track.value) {
    return track.value.configName
      ? `${track.value.name} - ${track.value.configName}`
      : track.value.name
  }
  return `Track ${props.entry.race.trackId}`
})

const raceDate = computed(() => {
  return d(new Date(props.entry.race.startTime), 'short')
})

const relativeTime = computed(() => {
  const now = new Date()
  const updated = new Date(props.entry.updatedAt)
  const diffMs = now.getTime() - updated.getTime()
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
  const diffDays = Math.floor(diffHours / 24)

  if (diffHours < 1) return 'just now'
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays < 7) return `${diffDays}d ago`
  return d(updated, 'short')
})

const isDNF = computed(() => props.entry.race.reasonOut !== 'Running')

const iRatingChange = computed(() => {
  const diff = props.entry.race.newIrating - props.entry.race.oldIrating
  const sign = diff > 0 ? '+' : ''
  return `${sign}${diff}`
})

const srChange = computed(() => {
  const diff = (props.entry.race.newSubLevel - props.entry.race.oldSubLevel) / 100
  const sign = diff > 0 ? '+' : ''
  return `${sign}${diff.toFixed(2)}`
})

const licenseChange = computed(() => {
  const oldLevel = props.entry.race.oldLicenseLevel
  const newLevel = props.entry.race.newLicenseLevel
  if (newLevel > oldLevel) return 'promoted'
  if (newLevel < oldLevel) return 'demoted'
  return null
})

const hasNotes = computed(() => !!props.entry.notes)
</script>

<template>
  <article class="journal-card">
    <header class="card-header">
      <div class="header-left">
        <SentimentBadge :sentiment="sentiment" size="sm" />
        <span class="series-name">{{ entry.race.seriesName }}</span>
      </div>
      <span class="race-date">{{ raceDate }}</span>
    </header>

    <div class="track-name">@ {{ trackName }}</div>

    <div class="stats-row">
      <span class="stat position">
        P{{ toDisplayPosition(entry.race.startPosition) }} → P{{ toDisplayPosition(entry.race.finishPosition) }}
      </span>
      <span class="stat incidents">{{ entry.race.incidents }}x</span>
      <span class="stat irating" :class="{ positive: entry.race.newIrating > entry.race.oldIrating, negative: entry.race.newIrating < entry.race.oldIrating }">
        {{ iRatingChange }} iR
      </span>
      <span class="stat sr" :class="{ positive: entry.race.newSubLevel > entry.race.oldSubLevel, negative: entry.race.newSubLevel < entry.race.oldSubLevel }">
        {{ srChange }} SR
      </span>
    </div>

    <div v-if="isDNF || licenseChange" class="badges-row">
      <span v-if="isDNF" class="badge badge-dnf">{{ t('journal.page.dnf') }}</span>
      <span v-if="licenseChange === 'promoted'" class="badge badge-promoted">{{ t('journal.page.promoted') }}</span>
      <span v-if="licenseChange === 'demoted'" class="badge badge-demoted">{{ t('journal.page.demoted') }}</span>
    </div>

    <p v-if="hasNotes" class="notes">{{ entry.notes }}</p>

    <footer class="card-footer">
      <RouterLink :to="{ name: 'race-details', params: { subsessionId: entry.race.subsessionId } }" class="view-race-link">
        {{ t('journal.page.viewRace') }} →
      </RouterLink>
      <span class="updated-time">{{ relativeTime }}</span>
    </footer>
  </article>
</template>

<style scoped>
.journal-card {
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 0.5rem;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.series-name {
  font-weight: 600;
  color: var(--color-text-primary);
  font-size: 0.9375rem;
}

.race-date {
  font-size: 0.75rem;
  color: var(--color-text-muted);
  white-space: nowrap;
}

.track-name {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.stats-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  font-size: 0.8125rem;
}

.stat {
  color: var(--color-text-secondary);
}

.stat.position {
  font-weight: 500;
  color: var(--color-text-primary);
}

.stat.positive {
  color: #22c55e;
}

.stat.negative {
  color: #ef4444;
}

.badges-row {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.badge {
  display: inline-block;
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
  font-size: 0.6875rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.025em;
}

.badge-dnf {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.badge-promoted {
  background: rgba(34, 197, 94, 0.15);
  color: #22c55e;
}

.badge-demoted {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.notes {
  margin: 0;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  line-height: 1.5;
  white-space: pre-wrap;
}

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 0.5rem;
  border-top: 1px solid var(--color-border);
}

.view-race-link {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-accent);
  text-decoration: none;
}

.view-race-link:hover {
  text-decoration: underline;
}

.updated-time {
  font-size: 0.75rem;
  color: var(--color-text-muted);
}
</style>