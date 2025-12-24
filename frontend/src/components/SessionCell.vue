<script setup lang="ts">
import { computed } from 'vue'
import { useCarsStore } from '@/stores/cars'

const props = defineProps<{
  seriesName: string
  carId: number
}>()

const carsStore = useCarsStore()

const car = computed(() => carsStore.getCar(props.carId))

const carName = computed(() => {
  if (!car.value) return `Car ${props.carId}`
  return car.value.name
})

const carAbbreviated = computed(() => {
  if (!car.value) return `Car ${props.carId}`
  return car.value.nameAbbreviated || car.value.name
})

const fullDescription = computed(() => {
  return `${props.seriesName} - ${carName.value}`
})
</script>

<template>
  <span class="session-cell" :title="fullDescription">
    <span class="session-text-full">
      <span class="series-name">{{ seriesName }}</span>
      <span class="car-name">{{ carName }}</span>
    </span>
    <span class="session-text-abbrev">{{ carAbbreviated }}</span>
  </span>
</template>

<style scoped>
.session-cell {
  display: block;
}

.session-text-full {
  display: flex;
  flex-direction: column;
}

.session-text-abbrev {
  display: none;
}

.series-name {
  display: block;
}

.car-name {
  display: block;
  font-size: 0.75rem;
  color: var(--color-text-muted);
}

@media (max-width: 768px) {
  .session-text-full {
    display: none;
  }

  .session-text-abbrev {
    display: block;
  }
}
</style>