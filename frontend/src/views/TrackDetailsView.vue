<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useTracksStore } from '@/stores/tracks'
import TrackMap from '@/components/TrackMap.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const tracksStore = useTracksStore()

const trackId = computed(() => {
  const id = route.params.id
  return typeof id === 'string' ? parseInt(id, 10) : NaN
})

const track = computed(() => {
  if (isNaN(trackId.value)) return undefined
  return tracksStore.getTrack(trackId.value)
})

const fullName = computed(() => {
  if (!track.value) return ''
  if (track.value.configName) {
    return `${track.value.name} - ${track.value.configName}`
  }
  return track.value.name
})

const trackType = computed(() => {
  if (!track.value) return ''
  return track.value.isOval ? t('trackDetails.oval') : t('trackDetails.roadCourse')
})

const surfaceType = computed(() => {
  if (!track.value) return ''
  return track.value.isDirt ? t('trackDetails.dirt') : t('trackDetails.paved')
})

const displayImage = computed(() => {
  if (!track.value) return ''
  return track.value.largeImageUrl || track.value.smallImageUrl || ''
})

function goBack() {
  router.push({ name: 'race-history' })
}
</script>

<template>
  <div class="track-details">
    <button class="back-button" @click="goBack">
      &larr; {{ t('trackDetails.backToRaces') }}
    </button>

    <div v-if="!track" class="not-found">
      {{ t('trackDetails.trackNotFound') }}
    </div>

    <template v-else>
      <header class="track-header">
        <div class="track-title">
          <h1>{{ track.name }}</h1>
          <span v-if="track.configName" class="config-name">{{ track.configName }}</span>
        </div>
      </header>

      <div class="track-content">
        <div class="track-image-section">
          <img
            v-if="displayImage"
            :src="displayImage"
            :alt="fullName"
            class="track-image"
          />
          <TrackMap
            v-if="track.trackMapUrl && track.trackMapLayers"
            :base-url="track.trackMapUrl"
            :layers="track.trackMapLayers"
          />
        </div>

        <div class="track-info">
          <div class="info-grid">
            <div v-if="track.location" class="info-item">
              <span class="info-label">{{ t('trackDetails.location') }}</span>
              <span class="info-value">{{ track.location }}</span>
            </div>

            <div class="info-item">
              <span class="info-label">{{ t('trackDetails.type') }}</span>
              <span class="info-value">{{ trackType }} / {{ surfaceType }}</span>
            </div>

            <div v-if="track.lengthMiles" class="info-item">
              <span class="info-label">{{ t('trackDetails.length') }}</span>
              <span class="info-value">{{ t('trackDetails.miles', { length: track.lengthMiles.toFixed(2) }) }}</span>
            </div>

            <div v-if="track.cornersPerLap" class="info-item">
              <span class="info-label">{{ t('trackDetails.corners') }}</span>
              <span class="info-value">{{ track.cornersPerLap }}</span>
            </div>

            <div v-if="track.pitRoadSpeedLimit" class="info-item">
              <span class="info-label">{{ t('trackDetails.pitSpeed') }}</span>
              <span class="info-value">{{ t('trackDetails.mph', { speed: track.pitRoadSpeedLimit }) }}</span>
            </div>

            <div v-if="track.category" class="info-item">
              <span class="info-label">{{ t('trackDetails.category') }}</span>
              <span class="info-value">{{ track.category }}</span>
            </div>
          </div>

          <div class="features">
            <h3>{{ t('trackDetails.features') }}</h3>
            <div class="feature-tags">
              <span v-if="track.hasNightLighting" class="feature-tag">{{ t('trackDetails.nightLighting') }}</span>
              <span v-if="track.rainEnabled" class="feature-tag">{{ t('trackDetails.rainEnabled') }}</span>
              <span v-if="track.freeWithSubscription" class="feature-tag free">{{ t('trackDetails.freeWithSub') }}</span>
              <span v-else class="feature-tag paid">{{ t('trackDetails.paidContent') }}</span>
              <span v-if="track.retired" class="feature-tag retired">{{ t('trackDetails.retired') }}</span>
            </div>
          </div>

          <div v-if="track.description" class="description">
            <h3>{{ t('trackDetails.description') }}</h3>
            <div class="description-content" v-html="track.description"></div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.track-details {
  padding: 2rem;
  max-width: 1200px;
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

.not-found {
  padding: 3rem;
  text-align: center;
  color: var(--color-text-secondary);
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
}

.track-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1.5rem;
  margin-bottom: 2rem;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.track-title h1 {
  margin: 0;
  font-size: 2rem;
  color: var(--color-text-primary);
}

.config-name {
  display: block;
  font-size: 1.25rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

.track-content {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 2rem;
}

.track-image-section {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.track-image {
  width: 100%;
  border-radius: 8px;
  border: 1px solid var(--color-border);
}

.track-info {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
}

.info-item {
  background: var(--color-bg-surface);
  padding: 1rem;
  border-radius: 8px;
  border: 1px solid var(--color-border);
}

.info-label {
  display: block;
  font-size: 0.75rem;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.25rem;
}

.info-value {
  display: block;
  font-size: 1rem;
  color: var(--color-text-primary);
  font-weight: 500;
}

.features h3,
.description h3 {
  margin: 0 0 0.75rem 0;
  font-size: 1rem;
  color: var(--color-text-primary);
}

.feature-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.feature-tag {
  display: inline-block;
  padding: 0.375rem 0.75rem;
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: 16px;
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
}

.feature-tag.free {
  background: rgba(34, 197, 94, 0.1);
  border-color: rgba(34, 197, 94, 0.3);
  color: #22c55e;
}

.feature-tag.paid {
  background: rgba(234, 179, 8, 0.1);
  border-color: rgba(234, 179, 8, 0.3);
  color: #eab308;
}

.feature-tag.retired {
  background: rgba(239, 68, 68, 0.1);
  border-color: rgba(239, 68, 68, 0.3);
  color: #ef4444;
}

.description-content {
  color: var(--color-text-secondary);
  line-height: 1.6;
}

.description-content :deep(p) {
  margin: 0 0 0.75rem 0;
}

.description-content :deep(p:last-child) {
  margin-bottom: 0;
}

.description-content :deep(a) {
  color: var(--color-accent);
}

@media (max-width: 768px) {
  .track-details {
    padding: 1rem;
  }

  .track-header {
    flex-direction: column-reverse;
    align-items: center;
    text-align: center;
  }

  .track-title h1 {
    font-size: 1.5rem;
  }

  .track-content {
    grid-template-columns: 1fr;
  }

  .info-grid {
    grid-template-columns: 1fr;
  }
}
</style>