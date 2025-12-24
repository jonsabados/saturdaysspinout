<script setup lang="ts">
import { computed } from 'vue'
import { useTracksStore } from '@/stores/tracks'

const props = defineProps<{
  trackId: number
}>()

const tracksStore = useTracksStore()

const track = computed(() => tracksStore.getTrack(props.trackId))

const displayName = computed(() => {
  if (!track.value) return `Track ${props.trackId}`
  if (track.value.configName) {
    return `${track.value.name} - ${track.value.configName}`
  }
  return track.value.name
})
</script>

<template>
  <span class="track-cell" :title="displayName">{{ displayName }}</span>
</template>

<style scoped>
.track-cell {
  display: block;
  max-width: 200px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>