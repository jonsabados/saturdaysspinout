<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useCarsStore } from '@/stores/cars'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const carsStore = useCarsStore()

const carId = computed(() => {
  const id = route.params.id
  return typeof id === 'string' ? parseInt(id, 10) : NaN
})

const car = computed(() => {
  if (isNaN(carId.value)) return undefined
  return carsStore.getCar(carId.value)
})

const displayImage = computed(() => {
  if (!car.value) return ''
  return car.value.largeImageUrl || car.value.smallImageUrl || ''
})

const formattedWeight = computed(() => {
  if (!car.value?.weight) return ''
  return t('carDetails.lbs', { weight: car.value.weight.toLocaleString() })
})

function goBack() {
  router.back()
}
</script>

<template>
  <div class="car-details">
    <button class="back-button" @click="goBack">
      &larr; {{ t('common.back') }}
    </button>

    <div v-if="!car" class="not-found">
      {{ t('carDetails.carNotFound') }}
    </div>

    <template v-else>
      <header class="car-header">
        <div class="car-title">
          <h1>{{ car.name }}</h1>
          <span v-if="car.make || car.model" class="make-model">
            {{ [car.make, car.model].filter(Boolean).join(' ') }}
          </span>
        </div>
      </header>

      <div class="car-content">
        <div class="car-image-section">
          <img
            v-if="displayImage"
            :src="displayImage"
            :alt="car.name"
            class="car-image"
          />
          <img
            v-if="car.logoUrl"
            :src="car.logoUrl"
            :alt="`${car.name} logo`"
            class="car-logo"
          />
        </div>

        <div class="car-info">
          <div class="info-grid">
            <div v-if="car.hpUnderHood" class="info-item">
              <span class="info-label">{{ t('carDetails.horsepowerUnderHood') }}</span>
              <span class="info-value">{{ car.hpUnderHood }} hp</span>
            </div>

            <div v-if="car.hpActual" class="info-item">
              <span class="info-label">{{ t('carDetails.horsepowerActual') }}</span>
              <span class="info-value">{{ car.hpActual }} hp</span>
            </div>

            <div v-if="car.weight" class="info-item">
              <span class="info-label">{{ t('carDetails.weight') }}</span>
              <span class="info-value">{{ formattedWeight }}</span>
            </div>

            <div v-if="car.categories && car.categories.length > 0" class="info-item categories-item">
              <span class="info-label">{{ t('carDetails.categories') }}</span>
              <span class="info-value">{{ car.categories.join(', ') }}</span>
            </div>
          </div>

          <div class="features">
            <h3>{{ t('carDetails.features') }}</h3>
            <div class="feature-tags">
              <span v-if="car.hasHeadlights" class="feature-tag">{{ t('carDetails.hasHeadlights') }}</span>
              <span v-if="car.hasMultipleDryTires" class="feature-tag">{{ t('carDetails.multipleDryTires') }}</span>
              <span v-if="car.rainEnabled" class="feature-tag">{{ t('carDetails.rainEnabled') }}</span>
              <span v-if="car.freeWithSubscription" class="feature-tag free">{{ t('carDetails.freeWithSub') }}</span>
              <span v-else class="feature-tag paid">{{ t('carDetails.paidContent') }}</span>
              <span v-if="car.retired" class="feature-tag retired">{{ t('carDetails.retired') }}</span>
            </div>
          </div>

          <div v-if="car.description" class="description">
            <h3>{{ t('carDetails.description') }}</h3>
            <div class="description-content" v-html="car.description"></div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.car-details {
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

.car-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1.5rem;
  margin-bottom: 2rem;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.car-title h1 {
  margin: 0;
  font-size: 2rem;
  color: var(--color-text-primary);
}

.make-model {
  display: block;
  font-size: 1.25rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

.car-content {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 2rem;
}

.car-image-section {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.car-image {
  width: 100%;
  border-radius: 8px;
  border: 1px solid var(--color-border);
}

.car-logo {
  width: 120px;
  height: auto;
}

.car-info {
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

.info-item.categories-item {
  grid-column: span 2;
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
  .car-details {
    padding: 1rem;
  }

  .car-header {
    flex-direction: column-reverse;
    align-items: center;
    text-align: center;
  }

  .car-title h1 {
    font-size: 1.5rem;
  }

  .car-content {
    grid-template-columns: 1fr;
  }

  .info-grid {
    grid-template-columns: 1fr;
  }

  .info-item.categories-item {
    grid-column: span 1;
  }
}
</style>