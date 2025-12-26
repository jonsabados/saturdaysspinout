import { ref, computed, watch } from 'vue'
import { defineStore } from 'pinia'
import { useAuthStore } from './auth'
import { useApiClient, type Car } from '@/api/client'

export const useCarsStore = defineStore('cars', () => {
  const authStore = useAuthStore()

  const cars = ref<Map<number, Car>>(new Map())
  const loading = ref(false)
  const error = ref<string | null>(null)

  const isLoaded = computed(() => cars.value.size > 0)

  function getCar(carId: number): Car | undefined {
    return cars.value.get(carId)
  }

  async function fetchCars() {
    if (loading.value) return

    loading.value = true
    error.value = null

    try {
      const apiClient = useApiClient()
      const response = await apiClient.getCars()
      const carMap = new Map<number, Car>()
      for (const car of response.response) {
        carMap.set(car.id, car)
      }
      cars.value = carMap
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch cars'
      console.error('Failed to fetch cars:', e)
    } finally {
      loading.value = false
    }
  }

  function clear() {
    cars.value = new Map()
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
    cars,
    loading,
    error,
    isLoaded,
    getCar,
    fetchCars,
    clear,
  }
})