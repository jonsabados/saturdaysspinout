import { useAuthStore } from '@/stores/auth'
import { useSessionStore } from '@/stores/session'

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

export interface ApiError {
  message: string
  correlationId?: string
}

export interface Race {
  id: number
  subsessionId: number
  trackId: number
  carId: number
  startTime: string
  startPosition: number
  startPositionInClass: number
  finishPosition: number
  finishPositionInClass: number
  incidents: number
  oldCpi: number
  newCpi: number
  oldIrating: number
  newIrating: number
  reasonOut: string
}

export interface Pagination {
  page: number
  resultsPerPage: number
  totalResults: number
  totalPages: number
}

export interface RacesResponse {
  items: Race[]
  pagination: Pagination
  correlationId: string
}

export interface RaceResponse {
  response: Race
  correlationId: string
}

export class ApiClient {
  private authStore: ReturnType<typeof useAuthStore>
  private sessionStore: ReturnType<typeof useSessionStore>

  constructor(authStore: ReturnType<typeof useAuthStore>, sessionStore: ReturnType<typeof useSessionStore>) {
    this.authStore = authStore
    this.sessionStore = sessionStore
  }

  async fetch<T>(path: string, options: RequestInit = {}): Promise<T> {
    if (!this.authStore.isLoggedIn) {
      throw new Error('Not authenticated')
    }

    const headers = new Headers(options.headers)
    headers.set('Authorization', `Bearer ${this.authStore.token}`)

    const response = await fetch(`${apiBaseUrl}${path}`, {
      ...options,
      headers,
    })

    if (response.status === 401) {
      // Token might have expired between check and request - try refresh once
      const refreshed = await this.authStore.refreshToken()
      if (refreshed) {
        // Retry the request with new token
        headers.set('Authorization', `Bearer ${this.authStore.token}`)
        const retryResponse = await fetch(`${apiBaseUrl}${path}`, {
          ...options,
          headers,
        })
        if (!retryResponse.ok) {
          throw await this.parseError(retryResponse)
        }
        return retryResponse.json()
      }
      throw new Error('Session expired - please log in again')
    }

    if (!response.ok) {
      throw await this.parseError(response)
    }

    return response.json()
  }

  private async parseError(response: Response): Promise<Error> {
    try {
      const data = await response.json() as ApiError
      let message = data.message || `Request failed: ${response.status}`
      if (data.correlationId) {
        message += ` (Correlation ID: ${data.correlationId})`
      }
      return new Error(message)
    } catch {
      return new Error(`Request failed: ${response.status}`)
    }
  }

  async triggerRaceIngestion(): Promise<void> {
    if (!this.sessionStore.isReady) {
      throw new Error('Session not ready')
    }

    await this.fetch('/ingestion/race', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ notifyConnectionId: this.sessionStore.connectionId }),
    })
  }

  async getRaces(
    driverId: number,
    startTime: Date,
    endTime: Date,
    page = 1,
    resultsPerPage = 10
  ): Promise<RacesResponse> {
    const params = new URLSearchParams({
      startTime: startTime.toISOString(),
      endTime: endTime.toISOString(),
      page: page.toString(),
      resultsPerPage: resultsPerPage.toString(),
    })
    return this.fetch<RacesResponse>(`/driver/${driverId}/races?${params}`)
  }

  async getRace(driverId: number, driverRaceId: number): Promise<RaceResponse> {
    return this.fetch<RaceResponse>(`/driver/${driverId}/races/${driverRaceId}`)
  }
}

export function useApiClient() {
  const authStore = useAuthStore()
  const sessionStore = useSessionStore()
  return new ApiClient(authStore, sessionStore)
}