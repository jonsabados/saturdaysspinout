<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  position: number
  positionInClass: number
}>()

const displayPosition = computed(() => props.position + 1)
const displayPositionInClass = computed(() => props.positionInClass + 1)
const isMultiClass = computed(() => props.position !== props.positionInClass)

const tooltipText = computed(() =>
  `Overall: P${displayPosition.value}, Class: P${displayPositionInClass.value}`,
)
</script>

<template>
  <span class="grid-position" :class="{ 'multi-class': isMultiClass }" :data-tooltip="tooltipText">
    <template v-if="isMultiClass">
      {{ displayPositionInClass }}<span class="separator">/</span>{{ displayPosition }}
    </template>
    <template v-else>
      {{ displayPosition }}
    </template>
  </span>
</template>

<style scoped>
.grid-position {
  position: relative;
  cursor: default;
}

.separator {
  opacity: 0.5;
  margin: 0 1px;
}

.grid-position.multi-class:hover::after {
  content: attr(data-tooltip);
  position: absolute;
  bottom: calc(100% + 6px);
  left: 50%;
  transform: translateX(-50%);
  padding: 0.375rem 0.625rem;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  font-size: 0.75rem;
  white-space: nowrap;
  color: var(--color-text-primary);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  z-index: 10;
}

.grid-position.multi-class:hover::before {
  content: '';
  position: absolute;
  bottom: calc(100% + 2px);
  left: 50%;
  transform: translateX(-50%);
  border: 4px solid transparent;
  border-top-color: var(--color-border);
  z-index: 10;
}
</style>