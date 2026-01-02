<script setup lang="ts">
defineOptions({ name: 'JournalView' })

import { ref, computed, onMounted, onUnmounted, onActivated, onDeactivated, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useApiClient, type JournalEntry } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { getSentiment } from '@/utils/tagHelpers'
import JournalFilters, { type JournalFiltersState } from '@/components/JournalFilters.vue'
import JournalEntryCard from '@/components/JournalEntryCard.vue'

const { t } = useI18n()
const apiClient = useApiClient()
const authStore = useAuthStore()

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

// Filters - default to last 90 days
const filters = ref<JournalFiltersState>({
  from: new Date(Date.now() - 90 * 24 * 60 * 60 * 1000),
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

onMounted(() => {
  loadEntries(true)
})

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

// Reload when date filters change
watch(
  () => [filters.value.from, filters.value.to],
  () => {
    loadEntries(true)
  }
)
</script>

<template>
  <div class="journal-page">
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
.journal-page {
  max-width: 800px;
  margin: 0 auto;
  padding: 2rem;
}

.page-header {
  margin-bottom: 1.5rem;
}

.page-header h1 {
  margin: 0;
  font-size: 1.75rem;
  font-weight: 600;
  color: var(--color-text-primary);
}

.loading-state,
.error-state,
.empty-state {
  text-align: center;
  padding: 3rem 2rem;
  background: var(--color-bg-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
}

.loading-state {
  color: var(--color-text-muted);
}

.error-state {
  color: #ef4444;
}

.empty-state h2 {
  margin: 0 0 0.5rem 0;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--color-text-primary);
}

.empty-state p {
  margin: 0;
  color: var(--color-text-muted);
}

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

@media (max-width: 768px) {
  .journal-page {
    padding: 1rem;
  }

  .page-header h1 {
    font-size: 1.5rem;
  }
}
</style>