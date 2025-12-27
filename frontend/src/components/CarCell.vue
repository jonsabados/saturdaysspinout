<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useCarsStore } from '@/stores/cars'

const props = defineProps<{
  carId: number
}>()

const carsStore = useCarsStore()

const car = computed(() => carsStore.getCar(props.carId))

const carName = computed(() => {
  if (!car.value) return `Car ${props.carId}`
  return car.value.name
})

const makeModel = computed(() => {
  if (!car.value) return ''
  return [car.value.make, car.value.model].filter(Boolean).join(' ')
})

const abbreviatedName = computed(() => {
  if (!car.value) return `Car ${props.carId}`
  return car.value.nameAbbreviated || car.value.name
})
</script>

<template>
  <RouterLink :to="{ name: 'car-details', params: { id: carId } }" class="car-cell" :title="carName">
    <span class="car-text-full">
      <span class="car-name">{{ carName }}</span>
      <span v-if="makeModel" class="car-make-model">{{ makeModel }}</span>
    </span>
    <span class="car-text-abbrev">{{ abbreviatedName }}</span>
  </RouterLink>
</template>

<style scoped>
.car-cell {
  display: block;
  text-decoration: none;
  color: inherit;
  transition: color 0.15s;
}

.car-cell:hover .car-name {
  text-decoration: underline;
}

.car-text-full {
  display: flex;
  flex-direction: column;
}

.car-text-abbrev {
  display: none;
  color: var(--color-accent);
}

.car-name {
  display: block;
  color: var(--color-accent);
}

.car-make-model {
  display: block;
  font-size: 0.75rem;
  color: var(--color-text-muted);
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