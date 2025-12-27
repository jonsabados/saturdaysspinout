<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TrackMapLayers } from '@/api/client'

const { t } = useI18n()

const props = defineProps<{
  baseUrl: string
  layers: TrackMapLayers
}>()

const svgContents = ref<Record<string, string>>({})
const loading = ref(true)
const error = ref(false)
const showBackground = ref(false)

// Layer order from bottom to top
const layerOrder = ['background', 'inactive', 'active', 'pitroad', 'startFinish', 'turns'] as const

const hasLayers = computed(() => {
  return props.baseUrl && props.layers && Object.values(props.layers).some((v) => v)
})

const orderedLayers = computed(() => {
  return layerOrder
    .filter((key) => {
      if (key === 'background' && !showBackground.value) return false
      return props.layers[key] && svgContents.value[key]
    })
    .map((key) => ({
      name: key,
      content: svgContents.value[key],
    }))
})

async function fetchLayer(name: string, filename: string): Promise<void> {
  if (!filename) return

  try {
    const url = `${props.baseUrl}${filename}`
    const response = await fetch(url)
    if (response.ok) {
      const text = await response.text()
      svgContents.value[name] = text
    }
  } catch (err) {
    console.warn(`[TrackMap] Failed to fetch layer ${name}:`, err)
  }
}

async function fetchAllLayers() {
  if (!hasLayers.value) {
    loading.value = false
    return
  }

  loading.value = true
  error.value = false
  svgContents.value = {}

  try {
    await Promise.all([
      fetchLayer('background', props.layers.background),
      fetchLayer('inactive', props.layers.inactive),
      fetchLayer('active', props.layers.active),
      fetchLayer('pitroad', props.layers.pitroad),
      fetchLayer('startFinish', props.layers.startFinish),
      fetchLayer('turns', props.layers.turns),
    ])

    if (Object.keys(svgContents.value).length === 0) {
      error.value = true
    }
  } catch (err) {
    console.error('[TrackMap] Failed to fetch layers:', err)
    error.value = true
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchAllLayers()
})

watch(
  () => props.baseUrl,
  () => {
    fetchAllLayers()
  }
)
</script>

<template>
  <div v-if="hasLayers" class="track-map">
    <div v-if="loading" class="track-map-loading">
      <span class="spinner"></span>
    </div>
    <div v-else-if="error" class="track-map-error">
      Failed to load track map
    </div>
    <template v-else>
      <div class="track-map-container">
        <div
          v-for="layer in orderedLayers"
          :key="layer.name"
          class="track-map-layer"
          :class="`layer-${layer.name}`"
          v-html="layer.content"
        ></div>
      </div>
      <div class="track-map-controls">
        <label class="background-toggle">
          <input type="checkbox" v-model="showBackground" />
          <span>{{ t('trackDetails.showBackground') }}</span>
        </label>
      </div>
    </template>
  </div>
</template>

<style scoped>
.track-map {
  width: 100%;
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  overflow: hidden;
}

.track-map-loading,
.track-map-error {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
  color: var(--color-text-muted);
  font-size: 0.875rem;
}

.spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-accent);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.track-map-container {
  position: relative;
  width: 100%;
  padding-top: 75%; /* 4:3 aspect ratio fallback */
}

.track-map-layer {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
}

.track-map-layer :deep(svg) {
  width: 100%;
  height: 100%;
  display: block;
}

/* Layer-specific styling */
.layer-background :deep(svg) {
  z-index: 1;
}

.layer-inactive :deep(svg) {
  z-index: 2;
}

.layer-active :deep(svg) {
  z-index: 3;
}

.layer-pitroad :deep(svg) {
  z-index: 4;
}

.layer-startFinish :deep(svg) {
  z-index: 5;
}

.layer-turns :deep(svg) {
  z-index: 6;
}

.track-map-controls {
  padding: 0.5rem 0.75rem;
  border-top: 1px solid var(--color-border);
  background: var(--color-bg-elevated);
}

.background-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  cursor: pointer;
}

.background-toggle input {
  cursor: pointer;
}
</style>