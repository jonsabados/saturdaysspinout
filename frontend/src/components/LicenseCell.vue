<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  oldLicenseLevel: number
  newLicenseLevel: number
  oldSubLevel: number
  newSubLevel: number
  oldCpi: number
  newCpi: number
}>()

const LICENSE_CLASSES = ['R', 'D', 'C', 'B', 'A'] as const
type LicenseClass = (typeof LICENSE_CLASSES)[number]

interface LicenseInfo {
  classLetter: LicenseClass
  safetyRating: string
  classIndex: number
}

function decodeLicense(licenseLevel: number, subLevel: number): LicenseInfo {
  // iRacing license_level encoding: 1-4=R, 5-8=D, 9-12=C, 13-16=B, 17-20=A
  // sub_level is SR * 100 (e.g., 381 = 3.81)
  const classIndex = Math.min(Math.floor((licenseLevel - 1) / 4), 4)
  const classLetter = LICENSE_CLASSES[classIndex]

  const sr = subLevel / 100
  const safetyRating = sr.toFixed(2)

  return { classLetter, safetyRating, classIndex }
}

const oldLicense = computed(() => decodeLicense(props.oldLicenseLevel, props.oldSubLevel))
const newLicense = computed(() => decodeLicense(props.newLicenseLevel, props.newSubLevel))

const srDelta = computed(() => {
  const oldSr = parseFloat(oldLicense.value.safetyRating)
  const newSr = parseFloat(newLicense.value.safetyRating)
  const delta = newSr - oldSr
  const sign = delta >= 0 ? '+' : ''
  return `${sign}${delta.toFixed(2)}`
})

const srDeltaClass = computed(() => {
  const oldSr = parseFloat(oldLicense.value.safetyRating)
  const newSr = parseFloat(newLicense.value.safetyRating)
  const delta = newSr - oldSr
  if (delta > 0) return 'stat-gain'
  if (delta < 0) return 'stat-loss'
  return ''
})

const licenseChanged = computed(() => {
  // License levels: 1-4=R, 5-8=D, 9-12=C, 13-16=B, 17-20=A
  const oldClass = Math.floor((props.oldLicenseLevel - 1) / 4)
  const newClass = Math.floor((props.newLicenseLevel - 1) / 4)
  return newClass > oldClass
})

const cpiDelta = computed(() => {
  const delta = props.newCpi - props.oldCpi
  const sign = delta >= 0 ? '+' : ''
  return `${sign}${delta.toFixed(2)}`
})

const tooltipText = computed(() => {
  return `CPI: ${props.oldCpi.toFixed(2)} → ${props.newCpi.toFixed(2)} (${cpiDelta.value})`
})
</script>

<template>
  <span class="license-cell" :title="tooltipText">
    <span
      class="license-badge"
      :class="`license-${newLicense.classLetter.toLowerCase()}`"
    >
      <span class="license-class">{{ newLicense.classLetter }}</span>
      <span class="license-sr">{{ newLicense.safetyRating }}</span>
    </span>
    <span v-if="licenseChanged" class="promotion-indicator" title="Promoted!">&#x2191;</span>
    <span class="sr-delta" :class="srDeltaClass">({{ srDelta }})</span>
  </span>
</template>

<style scoped>
.license-cell {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}

.license-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.125rem;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
  font-size: 0.75rem;
  font-weight: 600;
  color: white;
}

.license-class {
  font-weight: 700;
}

.license-sr {
  font-weight: 500;
}

/* iRacing license colors */
.license-r {
  background-color: #B81C1C;
}

.license-d {
  background-color: #D35400;
}

.license-c {
  background-color: #DAA520;
}

.license-b {
  background-color: #27AE60;
}

.license-a {
  background-color: #2980B9;
}

.promotion-indicator {
  color: var(--color-success, #27AE60);
  font-weight: bold;
  font-size: 0.875rem;
}

.sr-delta {
  font-size: 0.75rem;
}

.stat-gain {
  color: #22c55e;
}

.stat-loss {
  color: #ef4444;
}
</style>