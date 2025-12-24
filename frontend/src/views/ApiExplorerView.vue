<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useApiClient } from '@/api/client'

interface Parameter {
  type: string
  note?: string
  required?: boolean
}

interface Endpoint {
  link: string
  note?: string | string[]
  parameters?: Record<string, Parameter>
  expirationSeconds: number
}

interface DetailedEndpoint extends Endpoint {
  // Additional fields that may appear in detailed docs
  [key: string]: unknown
}

type ApiCategory = Record<string, Endpoint>
type ApiDocs = Record<string, ApiCategory>

const apiClient = useApiClient()

const docs = ref<ApiDocs | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const expandedCategories = ref<Set<string>>(new Set())
const selectedEndpoint = ref<{ category: string; name: string; endpoint: Endpoint } | null>(null)
const detailedDocs = ref<DetailedEndpoint | null>(null)
const detailLoading = ref(false)
const detailError = ref<string | null>(null)
const searchQuery = ref('')

const categories = computed(() => {
  if (!docs.value) return []
  return Object.keys(docs.value).sort()
})

const filteredCategories = computed(() => {
  if (!searchQuery.value.trim()) return categories.value
  const query = searchQuery.value.toLowerCase()
  return categories.value.filter(cat => {
    if (cat.toLowerCase().includes(query)) return true
    const endpoints = docs.value![cat]
    return Object.keys(endpoints).some(ep => ep.toLowerCase().includes(query))
  })
})

async function loadDocs() {
  loading.value = true
  error.value = null
  try {
    docs.value = await apiClient.fetch<ApiDocs>('/developer/iracing-api/')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load API docs'
  } finally {
    loading.value = false
  }
}

function toggleCategory(category: string) {
  if (expandedCategories.value.has(category)) {
    expandedCategories.value.delete(category)
  } else {
    expandedCategories.value.add(category)
  }
}

async function selectEndpoint(category: string, name: string, endpoint: Endpoint) {
  selectedEndpoint.value = { category, name, endpoint }
  detailedDocs.value = null
  detailError.value = null
  detailLoading.value = true

  try {
    // Fetch detailed docs for this specific endpoint
    const detailed = await apiClient.fetch<DetailedEndpoint>(`/developer/iracing-api/${category}/${name}`)
    detailedDocs.value = detailed
  } catch (e) {
    detailError.value = e instanceof Error ? e.message : 'Failed to load detailed docs'
  } finally {
    detailLoading.value = false
  }
}

function formatNote(note: string | string[] | undefined): string[] {
  if (!note) return []
  return Array.isArray(note) ? note : [note]
}

function getEndpointCount(category: string): number {
  if (!docs.value) return 0
  return Object.keys(docs.value[category]).length
}

// Get the effective endpoint data (detailed if available, otherwise basic)
const effectiveEndpoint = computed(() => {
  if (detailedDocs.value) return detailedDocs.value
  return selectedEndpoint.value?.endpoint ?? null
})

onMounted(loadDocs)
</script>

<template>
  <div class="api-explorer">
    <header class="explorer-header">
      <h1>iRacing API Explorer</h1>
      <p class="subtitle">Browse the iRacing Data API endpoints</p>
    </header>

    <div v-if="loading" class="loading">
      Loading API documentation...
    </div>

    <div v-else-if="error" class="error">
      <p>{{ error }}</p>
      <button @click="loadDocs">Retry</button>
    </div>

    <div v-else class="explorer-layout">
      <aside class="sidebar">
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search endpoints..."
          class="search-input"
        />

        <nav class="category-list">
          <div
            v-for="category in filteredCategories"
            :key="category"
            class="category"
          >
            <button
              class="category-header"
              :class="{ expanded: expandedCategories.has(category) }"
              @click="toggleCategory(category)"
            >
              <span class="category-name">{{ category }}</span>
              <span class="endpoint-count">{{ getEndpointCount(category) }}</span>
            </button>

            <div v-if="expandedCategories.has(category)" class="endpoint-list">
              <button
                v-for="(endpoint, name) in docs![category]"
                :key="name"
                class="endpoint-item"
                :class="{ selected: selectedEndpoint?.category === category && selectedEndpoint?.name === name }"
                @click="selectEndpoint(category, String(name), endpoint)"
              >
                {{ name }}
              </button>
            </div>
          </div>
        </nav>
      </aside>

      <main class="detail-panel">
        <div v-if="!selectedEndpoint" class="empty-state">
          <p>Select an endpoint from the sidebar to view details</p>
        </div>

        <div v-else class="endpoint-detail">
          <h2>{{ selectedEndpoint.category }} / {{ selectedEndpoint.name }}</h2>

          <div v-if="detailLoading" class="detail-loading">
            Loading detailed documentation...
          </div>

          <div v-else-if="detailError" class="detail-error">
            <p>{{ detailError }}</p>
            <p class="fallback-note">Showing basic documentation:</p>
          </div>

          <template v-if="effectiveEndpoint">
            <section class="detail-section">
              <h3>Endpoint</h3>
              <code class="endpoint-url">{{ effectiveEndpoint.link }}</code>
            </section>

            <section v-if="formatNote(effectiveEndpoint.note).length" class="detail-section">
              <h3>Notes</h3>
              <ul class="notes-list">
                <li v-for="(note, i) in formatNote(effectiveEndpoint.note)" :key="i">
                  {{ note }}
                </li>
              </ul>
            </section>

            <section v-if="effectiveEndpoint.parameters" class="detail-section">
              <h3>Parameters</h3>
              <div class="params-table">
                <div
                  v-for="(param, paramName) in effectiveEndpoint.parameters"
                  :key="paramName"
                  class="param-row"
                >
                  <div class="param-header">
                    <code class="param-name">{{ paramName }}</code>
                    <span class="param-type">{{ param.type }}</span>
                    <span v-if="param.required" class="param-required">required</span>
                  </div>
                  <p v-if="param.note" class="param-note">{{ param.note }}</p>
                </div>
              </div>
            </section>

            <section class="detail-section">
              <h3>Cache</h3>
              <p class="cache-info">
                Results cached for {{ effectiveEndpoint.expirationSeconds }} seconds
              </p>
            </section>

            <section v-if="detailedDocs" class="detail-section">
              <h3>Raw Response</h3>
              <pre class="raw-json">{{ JSON.stringify(detailedDocs, null, 2) }}</pre>
            </section>
          </template>
        </div>
      </main>
    </div>
  </div>
</template>

<style scoped>
.api-explorer {
  max-width: 1400px;
  margin: 0 auto;
}

.explorer-header {
  margin-bottom: 1.5rem;
}

.explorer-header h1 {
  margin: 0 0 0.25rem;
  font-size: 1.75rem;
}

.subtitle {
  margin: 0;
  color: var(--color-text-secondary);
}

.loading, .error {
  text-align: center;
  padding: 3rem;
  color: var(--color-text-secondary);
}

.error {
  color: var(--color-error);
}

.error button {
  margin-top: 1rem;
  padding: 0.5rem 1rem;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-light);
  border-radius: 4px;
  color: var(--color-text-primary);
  cursor: pointer;
}

.error button:hover {
  border-color: var(--color-accent);
}

.explorer-layout {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 1.5rem;
  align-items: start;
}

.sidebar {
  background: var(--color-bg-surface);
  border-radius: 8px;
  padding: 1rem;
  position: sticky;
  top: 1.5rem;
  max-height: calc(100vh - 3rem);
  overflow-y: auto;
}

.search-input {
  width: 100%;
  padding: 0.625rem 0.75rem;
  background: var(--color-bg-deep);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  color: var(--color-text-primary);
  font-size: 0.875rem;
  margin-bottom: 1rem;
}

.search-input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.search-input::placeholder {
  color: var(--color-text-muted);
}

.category-list {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.category-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 0.5rem 0.75rem;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
  cursor: pointer;
  text-align: left;
  transition: background 0.15s;
}

.category-header:hover {
  background: var(--color-accent-subtle);
}

.category-header.expanded {
  color: var(--color-accent);
}

.category-name {
  font-weight: 500;
}

.endpoint-count {
  font-size: 0.75rem;
  color: var(--color-text-muted);
  background: var(--color-border);
  padding: 0.125rem 0.5rem;
  border-radius: 10px;
}

.endpoint-list {
  display: flex;
  flex-direction: column;
  margin-left: 0.75rem;
  padding-left: 0.75rem;
  border-left: 1px solid var(--color-border);
}

.endpoint-item {
  padding: 0.375rem 0.75rem;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: var(--color-text-secondary);
  font-size: 0.8125rem;
  cursor: pointer;
  text-align: left;
  transition: all 0.15s;
}

.endpoint-item:hover {
  color: var(--color-text-primary);
  background: var(--color-accent-subtle);
}

.endpoint-item.selected {
  color: var(--color-accent);
  background: var(--color-accent-muted);
}

.detail-panel {
  background: var(--color-bg-surface);
  border-radius: 8px;
  padding: 1.5rem;
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--color-text-muted);
}

.endpoint-detail h2 {
  margin: 0 0 1.5rem;
  font-size: 1.25rem;
  color: var(--color-text-primary);
}

.detail-section {
  margin-bottom: 1.5rem;
}

.detail-section h3 {
  margin: 0 0 0.5rem;
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-secondary);
}

.endpoint-url {
  display: block;
  padding: 0.75rem 1rem;
  background: var(--color-bg-deep);
  border-radius: 4px;
  font-size: 0.875rem;
  color: var(--color-accent);
  word-break: break-all;
}

.notes-list {
  margin: 0;
  padding-left: 1.25rem;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
  line-height: 1.5;
}

.notes-list li {
  margin-bottom: 0.25rem;
}

.params-table {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.param-row {
  padding: 0.75rem;
  background: var(--color-bg-deep);
  border-radius: 4px;
}

.param-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.param-name {
  font-size: 0.875rem;
  color: var(--color-text-primary);
}

.param-type {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  background: var(--color-border);
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.param-required {
  font-size: 0.6875rem;
  color: var(--color-warning);
  background: rgba(255, 170, 0, 0.15);
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.param-note {
  margin: 0.5rem 0 0;
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  line-height: 1.4;
}

.cache-info {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
}

.detail-loading {
  color: var(--color-text-secondary);
  font-size: 0.875rem;
  margin-bottom: 1rem;
}

.detail-error {
  color: var(--color-warning);
  font-size: 0.875rem;
  margin-bottom: 1rem;
}

.detail-error p {
  margin: 0 0 0.25rem;
}

.fallback-note {
  color: var(--color-text-secondary);
  font-style: italic;
}

.raw-json {
  margin: 0;
  padding: 1rem;
  background: var(--color-bg-deep);
  border-radius: 4px;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 400px;
  overflow-y: auto;
}

/* Tablet - narrower sidebar */
@media (max-width: 1024px) {
  .explorer-layout {
    grid-template-columns: 240px 1fr;
    gap: 1rem;
  }
}

/* Mobile - stacked layout */
@media (max-width: 768px) {
  .explorer-header h1 {
    font-size: 1.5rem;
  }

  .explorer-layout {
    grid-template-columns: 1fr;
    gap: 1rem;
  }

  .sidebar {
    position: static;
    max-height: none;
  }

  .detail-panel {
    max-height: none;
    min-height: 50vh;
  }

  .empty-state {
    min-height: 200px;
  }

  .endpoint-detail h2 {
    font-size: 1.1rem;
    word-break: break-word;
  }

  .endpoint-url {
    font-size: 0.75rem;
    padding: 0.625rem 0.75rem;
  }

  .param-row {
    padding: 0.625rem;
  }

  .raw-json {
    font-size: 0.625rem;
    max-height: 300px;
  }
}

/* Small mobile */
@media (max-width: 480px) {
  .explorer-header h1 {
    font-size: 1.25rem;
  }

  .subtitle {
    font-size: 0.875rem;
  }

  .sidebar {
    padding: 0.75rem;
  }

  .detail-panel {
    padding: 1rem;
  }
}
</style>