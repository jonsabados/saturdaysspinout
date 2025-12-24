<script setup lang="ts">
import { computed } from 'vue'
import { useCarsStore } from '@/stores/cars'

const props = defineProps<{
  carId: number
}>()

const carsStore = useCarsStore()

const car = computed(() => carsStore.getCar(props.carId))

const displayName = computed(() => {
  if (!car.value) return `Car ${props.carId}`
  return car.value.name
})

const abbreviatedName = computed(() => {
  if (!car.value) return `Car ${props.carId}`
  return car.value.nameAbbreviated || car.value.name
})
</script>

<template>
  <span class="car-cell" :title="displayName">
    <span class="car-text-full">{{ displayName }}</span>
    <span class="car-text-abbrev">{{ abbreviatedName }}</span>
  </span>
</template>

<style scoped>
.car-cell {
  display: block;
}

.car-text-full {
  display: block;
}

.car-text-abbrev {
  display: none;
}

@media (max-width: 768px) {
  .car-text-full {
    display: none;
  }

  .car-text-abbrev {
    display: block;
  }
}
</style>