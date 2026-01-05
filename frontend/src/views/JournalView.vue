<script setup lang="ts">
defineOptions({ name: 'JournalView' })

import { ref, computed, onUnmounted, onActivated, onDeactivated, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useApiClient, type JournalEntry } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { useDriverStore } from '@/stores/driver'
import { getSentiment } from '@/utils/tagHelpers'
import JournalFilters, { type JournalFiltersState } from '@/components/JournalFilters.vue'
import JournalEntryCard from '@/components/JournalEntryCard.vue'
import '@/assets/page-layout.css'

const { t } = useI18n()
const apiClient = useApiClient()
const authStore = useAuthStore()
const driverStore = useDriverStore()

// State
const entries = ref<JournalEntry[]>([])
const isLoading = ref(false)
const isLoadingMore = ref(false)
const hasMore = ref(true)
const page = ref(1)
const totalResults = ref(0)
const error = ref<string | null>(null)

// Sentinel element for infinite scroll
const sentinelRef = ref<HTMLElement | null>(null)

// Scroll position preservation
const savedScrollY = ref(0)

// Track if we've initialized dates from driver data
const initialized = ref(false)

// Filters - dates will be set when driver data loads
const filters = ref<JournalFiltersState>({
  from: new Date(), // Placeholder, will be set to memberSince
  to: new Date(),
  sentiment: null,
  showDNFOnly: false,
})

const driverId = computed(() => authStore.userId)

// Filter entries client-side for sentiment and DNF
// (API returns all entries, we filter locally)
const filteredEntries = computed(() => {
  let result = entries.value

  // Filter by sentiment
  if (filters.value.sentiment !== null && filters.value.sentiment.length > 0) {
    result = result.filter(entry => {
      const sentiment = getSentiment(entry.tags)
      return sentiment !== null && filters.value.sentiment!.includes(sentiment)
    })
  }

  // Filter by DNF only
  if (filters.value.showDNFOnly) {
    result = result.filter(entry => entry.race.reasonOut !== 'Running')
  }

  return result
})

const hasEntries = computed(() => entries.value.length > 0)
const hasFilteredEntries = computed(() => filteredEntries.value.length > 0)
const isFiltering = computed(() =>
  filters.value.sentiment !== null || filters.value.showDNFOnly
)

async function loadEntries(reset = false) {
  if (!driverId.value) return

  if (reset) {
    page.value = 1
    entries.value = []
    hasMore.value = true
    isLoading.value = true
  } else {
    isLoadingMore.value = true
  }

  error.value = null

  try {
    const response = await apiClient.getJournalEntries(
      driverId.value,
      filters.value.from,
      filters.value.to,
      page.value,
      20
    )

    if (reset) {
      entries.value = response.items
    } else {
      entries.value = [...entries.value, ...response.items]
    }

    totalResults.value = response.pagination.totalResults
    hasMore.value = page.value < response.pagination.totalPages
    page.value++
  } catch (err) {
    console.error('[JournalView] Failed to load entries:', err)
    error.value = err instanceof Error ? err.message : 'Failed to load journal entries'
  } finally {
    isLoading.value = false
    isLoadingMore.value = false
  }
}

function loadMore() {
  if (!isLoadingMore.value && hasMore.value) {
    loadEntries(false)
  }
}

// Set up Intersection Observer for infinite scroll
let observer: IntersectionObserver | null = null

// Watch for sentinel element to become available (it's inside v-else block)
watch(sentinelRef, (el) => {
  // Clean up previous observer
  if (observer) {
    observer.disconnect()
  }

  if (el) {
    observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && !isLoading.value && !isLoadingMore.value && hasMore.value) {
          loadMore()
        }
      },
      { rootMargin: '100px' }
    )
    observer.observe(el)
  }
})

// Initialize date filters when driver loads (once only)
watch(
  () => driverStore.driver,
  (driver) => {
    if (driver?.memberSince && !initialized.value) {
      filters.value.from = new Date(driver.memberSince)
      filters.value.to = new Date()
      initialized.value = true
      loadEntries(true)
    }
  },
  { immediate: true }
)

// Reload when date filters change (user interaction only, not initialization)
watch(
  () => [filters.value.from, filters.value.to],
  ([newFrom, newTo], [oldFrom, oldTo]) => {
    if (initialized.value && (oldFrom !== newFrom || oldTo !== newTo)) {
      loadEntries(true)
    }
  }
)

onUnmounted(() => {
  if (observer) {
    observer.disconnect()
  }
})

// Save scroll position when leaving, restore when returning
onDeactivated(() => {
  savedScrollY.value = window.scrollY
})

onActivated(() => {
  requestAnimationFrame(() => {
    if (savedScrollY.value > 0) {
      window.scrollTo(0, savedScrollY.value)
    }
  })
})
</script>

<template>
  <div class="journal-page page-view">
    <header class="page-header">
      <h1>{{ t('journal.page.title') }}</h1>
    </header>

    <JournalFilters v-model="filters" />

    <!-- Loading state -->
    <div v-if="isLoading" class="loading-state">
      {{ t('journal.page.loading') }}
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="error-state">
      {{ error }}
    </div>

    <!-- No entries at all -->
    <div v-else-if="!hasEntries" class="empty-state">
      <h2>{{ t('journal.page.noEntries') }}</h2>
      <p>{{ t('journal.page.noEntriesPrompt') }}</p>
    </div>

    <!-- Has entries but none match filters -->
    <div v-else-if="isFiltering && !hasFilteredEntries" class="empty-state">
      <h2>{{ t('journal.page.noMatchingEntries') }}</h2>
      <p>{{ t('journal.page.noMatchingPrompt') }}</p>
    </div>

    <!-- Timeline of entries -->
    <div v-else class="journal-timeline">
      <JournalEntryCard
        v-for="entry in filteredEntries"
        :key="entry.raceId"
        :entry="entry"
      />

      <!-- Sentinel for infinite scroll -->
      <div ref="sentinelRef" class="sentinel">
        <span v-if="isLoadingMore" class="loading-more">
          {{ t('journal.page.loadingMore') }}
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* View-specific styles (shared styles from page-layout.css) */
.journal-timeline {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.sentinel {
  padding: 1rem;
  text-align: center;
  min-height: 50px;
}

.loading-more {
  color: var(--color-text-muted);
  font-size: 0.875rem;
}
</style>