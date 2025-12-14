import { useAuthStore } from '@/stores/auth'
import { useSessionStore } from '@/stores/session'

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

export interface ApiError {
  message: string
  correlationId?: string
}

export class ApiClient {
  private authStore: ReturnType<typeof useAuthStore>
  private sessionStore: ReturnType<typeof useSessionStore>

  constructor(authStore: ReturnType<typeof useAuthStore>, sessionStore: ReturnType<typeof useSessionStore>) {
    this.authStore = authStore
    this.sessionStore = sessionStore
  }

  async fetch<T>(path: string, options: RequestInit = {}): Promise<T> {
    // Proactively refresh token if it's close to expiry
    if (this.authStore.needsRefresh) {
      await this.authStore.refreshToken()
    }

    // Check if we're still logged in after potential refresh
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
}

export function useApiClient() {
  const authStore = useAuthStore()
  const sessionStore = useSessionStore()
  return new ApiClient(authStore, sessionStore)
}