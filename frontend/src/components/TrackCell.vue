<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useTracksStore } from '@/stores/tracks'

const props = defineProps<{
  trackId: number
}>()

const tracksStore = useTracksStore()

const track = computed(() => tracksStore.getTrack(props.trackId))

const trackName = computed(() => {
  if (!track.value) return `Track ${props.trackId}`
  return track.value.name
})

const configName = computed(() => track.value?.configName || '')

const fullName = computed(() => {
  if (!track.value) return `Track ${props.trackId}`
  if (track.value.configName) {
    return `${track.value.name} - ${track.value.configName}`
  }
  return track.value.name
})
</script>

<template>
  <RouterLink :to="{ name: 'track-details', params: { id: trackId } }" class="track-cell" :title="fullName">
    <span class="track-text-full">
      <span class="track-name">{{ trackName }}</span>
      <span v-if="configName" class="track-config">{{ configName }}</span>
    </span>
    <span class="track-text-abbrev">{{ trackName }}</span>
  </RouterLink>
</template>

<style scoped>
.track-cell {
  display: block;
  text-decoration: none;
  color: inherit;
  transition: color 0.15s;
}

.track-cell:hover .track-name {
  color: var(--color-accent);
}

.track-text-full {
  display: flex;
  flex-direction: column;
}

.track-text-abbrev {
  display: none;
}

.track-name {
  display: block;
}

.track-config {
  display: block;
  font-size: 0.75rem;
  color: var(--color-text-muted);
}

@media (max-width: 768px) {
  .track-text-full {
    display: none;
  }

  .track-text-abbrev {
    display: block;
  }
}
</style>