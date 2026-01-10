<script setup lang="ts">
import { computed, ref } from 'vue'
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  type ChartData,
  type ChartOptions,
} from 'chart.js'
import { useAnalyticsStore } from '@/stores/analytics'
import type { AnalyticsGranularity } from '@/api/client'
import { useI18n } from 'vue-i18n'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend)

const { t } = useI18n()
const analyticsStore = useAnalyticsStore()

// Metric toggles
const showIRating = ref(true)
const showCPI = ref(true)
const showIncidents = ref(false)

// Granularity options
const granularityOptions: { value: AnalyticsGranularity; labelKey: string }[] = [
  { value: 'day', labelKey: 'analytics.chart.day' },
  { value: 'week', labelKey: 'analytics.chart.week' },
  { value: 'month', labelKey: 'analytics.chart.month' },
  { value: 'year', labelKey: 'analytics.chart.year' },
]

function onGranularityChange(event: Event) {
  const value = (event.target as HTMLSelectElement).value as AnalyticsGranularity
  analyticsStore.setGranularity(value)
  analyticsStore.fetchTimeSeries()
}

// Chart colors from design system
const colors = {
  iRating: '#3b82f6', // blue
  cpi: '#22c55e', // green
  incidents: '#ef4444', // red
}

const chartData = computed<ChartData<'line'>>(() => {
  const timeSeries = analyticsStore.timeSeries
  if (!timeSeries || timeSeries.length === 0) {
    return { labels: [], datasets: [] }
  }

  const labels = timeSeries.map((p) => formatPeriodLabel(p.period))
  const datasets = []

  if (showIRating.value) {
    datasets.push({
      label: t('analytics.chart.iRating'),
      data: timeSeries.map((p) => p.summary.iRatingEnd ?? 0),
      borderColor: colors.iRating,
      backgroundColor: colors.iRating + '20',
      tension: 0.3,
      yAxisID: 'y',
    })
  }

  if (showCPI.value) {
    datasets.push({
      label: t('analytics.chart.cpi'),
      data: timeSeries.map((p) => p.summary.cpiEnd ?? 0),
      borderColor: colors.cpi,
      backgroundColor: colors.cpi + '20',
      tension: 0.3,
      yAxisID: 'y1',
    })
  }

  if (showIncidents.value) {
    datasets.push({
      label: t('analytics.chart.incidents'),
      data: timeSeries.map((p) => p.summary.avgIncidents ?? 0),
      borderColor: colors.incidents,
      backgroundColor: colors.incidents + '20',
      tension: 0.3,
      yAxisID: 'y2',
    })
  }

  return { labels, datasets }
})

// Chart.js doesn't support CSS variables, so we need to read computed styles
function getChartColors() {
  const root = document.documentElement
  const styles = getComputedStyle(root)
  return {
    textPrimary: styles.getPropertyValue('--color-text-primary').trim() || '#e5e5e5',
    textSecondary: styles.getPropertyValue('--color-text-secondary').trim() || '#a3a3a3',
    textMuted: styles.getPropertyValue('--color-text-muted').trim() || '#737373',
    border: styles.getPropertyValue('--color-border').trim() || '#404040',
    bgElevated: styles.getPropertyValue('--color-bg-elevated').trim() || '#262626',
  }
}

const chartOptions = computed<ChartOptions<'line'>>(() => {
  const hasIRating = showIRating.value
  const hasCPI = showCPI.value
  const hasIncidents = showIncidents.value
  const chartColors = getChartColors()

  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: {
      mode: 'index',
      intersect: false,
    },
    plugins: {
      legend: {
        position: 'bottom',
        labels: {
          color: chartColors.textPrimary,
          usePointStyle: true,
        },
      },
      tooltip: {
        backgroundColor: chartColors.bgElevated,
        titleColor: chartColors.textPrimary,
        bodyColor: chartColors.textSecondary,
        borderColor: chartColors.border,
        borderWidth: 1,
      },
    },
    scales: {
      x: {
        grid: {
          color: chartColors.border,
        },
        ticks: {
          color: chartColors.textSecondary,
        },
      },
      y: {
        type: 'linear',
        display: hasIRating,
        position: 'left',
        title: {
          display: true,
          text: t('analytics.chart.iRatingAxis'),
          color: chartColors.textSecondary,
        },
        grid: {
          color: chartColors.border,
        },
        ticks: {
          color: chartColors.textSecondary,
        },
      },
      y1: {
        type: 'linear',
        display: hasCPI,
        position: 'right',
        title: {
          display: true,
          text: t('analytics.chart.cpiAxis'),
          color: chartColors.textSecondary,
        },
        grid: {
          drawOnChartArea: false,
        },
        ticks: {
          color: chartColors.textSecondary,
        },
      },
      y2: {
        type: 'linear',
        display: hasIncidents,
        position: 'right',
        title: {
          display: true,
          text: t('analytics.chart.incidentsAxis'),
          color: chartColors.textSecondary,
        },
        grid: {
          drawOnChartArea: false,
        },
        ticks: {
          color: chartColors.textSecondary,
        },
      },
    },
  }
})

function formatPeriodLabel(period: string): string {
  // Period comes in formats like "2024-01-15" for day, "2024-W03" for week, "2024-01" for month, "2024" for year
  if (period.includes('-W')) {
    // Week format: 2024-W03
    return period
  }
  const parts = period.split('-')
  if (parts.length === 3) {
    // Day format: 2024-01-15
    const date = new Date(period)
    return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  }
  if (parts.length === 2) {
    // Month format: 2024-01
    const date = new Date(period + '-01')
    return date.toLocaleDateString(undefined, { month: 'short', year: '2-digit' })
  }
  // Year format: 2024
  return period
}
</script>

<template>
  <div class="analytics-chart">
    <div class="chart-header">
      <h3>{{ t('analytics.chart.title') }}</h3>
      <div class="chart-controls">
        <div class="metric-toggles">
          <label class="toggle" :style="{ '--toggle-color': colors.iRating }">
            <input type="checkbox" v-model="showIRating" />
            <span>{{ t('analytics.chart.iRating') }}</span>
          </label>
          <label class="toggle" :style="{ '--toggle-color': colors.cpi }">
            <input type="checkbox" v-model="showCPI" />
            <span>{{ t('analytics.chart.cpi') }}</span>
          </label>
          <label class="toggle" :style="{ '--toggle-color': colors.incidents }">
            <input type="checkbox" v-model="showIncidents" />
            <span>{{ t('analytics.chart.incidents') }}</span>
          </label>
        </div>
        <select
          class="granularity-select"
          :value="analyticsStore.granularity"
          @change="onGranularityChange"
        >
          <option v-for="opt in granularityOptions" :key="opt.value" :value="opt.value">
            {{ t(opt.labelKey) }}
          </option>
        </select>
      </div>
    </div>

    <div class="chart-container" v-if="analyticsStore.hasTimeSeries">
      <Line :data="chartData" :options="chartOptions" />
    </div>

    <div class="chart-loading" v-else-if="analyticsStore.loadingTimeSeries">
      {{ t('analytics.chart.loading') }}
    </div>

    <div class="chart-empty" v-else>
      {{ t('analytics.chart.noData') }}
    </div>

    <p class="chart-disclaimer">
      {{ t('analytics.chart.disclaimer') }}
    </p>
  </div>
</template>

<style scoped>
.analytics-chart {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 1rem;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 1rem;
  margin-bottom: 1rem;
}

.chart-header h3 {
  margin: 0;
  font-size: 1rem;
  color: var(--color-text-primary);
}

.chart-controls {
  display: flex;
  gap: 1rem;
  align-items: center;
  flex-wrap: wrap;
}

.metric-toggles {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.toggle {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.toggle input {
  accent-color: var(--toggle-color);
}

.toggle:has(input:checked) span {
  color: var(--toggle-color);
  font-weight: 500;
}

.granularity-select {
  padding: 0.375rem 0.75rem;
  background: var(--color-bg-deep);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  color: var(--color-text-primary);
  font-size: 0.875rem;
  cursor: pointer;
}

.granularity-select option {
  background: var(--color-bg-deep);
  color: var(--color-text-primary);
}

.granularity-select:focus {
  outline: none;
  border-color: var(--color-accent);
}

.chart-container {
  height: 300px;
  position: relative;
}

.chart-loading,
.chart-empty {
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
  font-size: 0.875rem;
}

.chart-disclaimer {
  margin: 0.75rem 0 0 0;
  font-size: 0.75rem;
  color: var(--color-text-muted);
  font-style: italic;
  text-align: center;
}
</style>