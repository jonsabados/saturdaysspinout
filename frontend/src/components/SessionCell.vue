<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
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
      <RouterLink :to="{ name: 'car-details', params: { id: carId } }" class="car-name">{{ carName }}</RouterLink>
    </span>
    <RouterLink :to="{ name: 'car-details', params: { id: carId } }" class="session-text-abbrev">{{ carAbbreviated }}</RouterLink>
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
  color: var(--color-accent);
  text-decoration: none;
}

.session-text-abbrev:hover {
  text-decoration: underline;
}

.series-name {
  display: block;
}

.car-name {
  display: block;
  font-size: 0.75rem;
  color: var(--color-accent);
  text-decoration: none;
}

.car-name:hover {
  text-decoration: underline;
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