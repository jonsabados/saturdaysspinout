import { ref, computed, watch } from 'vue'
import { defineStore } from 'pinia'
import { useAuthStore } from './auth'
import { useApiClient, type Track } from '@/api/client'

export const useTracksStore = defineStore('tracks', () => {
  const authStore = useAuthStore()

  const tracks = ref<Map<number, Track>>(new Map())
  const loading = ref(false)
  const error = ref<string | null>(null)

  const isLoaded = computed(() => tracks.value.size > 0)

  function getTrack(trackId: number): Track | undefined {
    return tracks.value.get(trackId)
  }

  async function fetchTracks() {
    if (loading.value) return

    loading.value = true
    error.value = null

    try {
      const apiClient = useApiClient()
      const response = await apiClient.getTracks()
      const trackMap = new Map<number, Track>()
      for (const track of response.response) {
        trackMap.set(track.id, track)
      }
      tracks.value = trackMap
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch tracks'
      console.error('Failed to fetch tracks:', e)
    } finally {
      loading.value = false
    }
  }

  function clear() {
    tracks.value = new Map()
    error.value = null
  }

  watch(
    () => authStore.isLoggedIn,
    (isLoggedIn) => {
      if (!isLoggedIn) {
        clear()
      }
    }
  )

  return {
    tracks,
    loading,
    error,
    isLoaded,
    getTrack,
    fetchTracks,
    clear,
  }
})