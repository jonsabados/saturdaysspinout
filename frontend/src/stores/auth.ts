import { defineStore } from 'pinia'

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

// Refresh when token has less than 5 minutes remaining
const REFRESH_THRESHOLD_SECONDS = 5 * 60

interface AuthState {
  token: string | null
  expiresAt: number | null
  userId: number | null
  userName: string | null
  refreshInProgress: boolean
  sessionExpired: boolean
}

interface RefreshResponse {
  response: {
    token: string
    expires_at: number
    user_id: number
    user_name: string
  }
  correlationId: string
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    token: null,
    expiresAt: null,
    userId: null,
    userName: null,
    refreshInProgress: false,
    sessionExpired: false,
  }),

  getters: {
    isLoggedIn(): boolean {
      return this.token !== null
    },

    needsRefresh(): boolean {
      if (!this.token || !this.expiresAt) {
        return false
      }
      // expiresAt is the iRacing token expiry - refresh before it expires
      const secondsUntilExpiry = this.expiresAt - Date.now() / 1000
      return secondsUntilExpiry < REFRESH_THRESHOLD_SECONDS
    },
  },

  actions: {
    setSession(token: string, expiresAt: number, userId?: number, userName?: string) {
      this.token = token
      this.expiresAt = expiresAt
      this.userId = userId ?? null
      this.userName = userName ?? null
      this.sessionExpired = false
    },

    logout() {
      this.token = null
      this.expiresAt = null
      this.userId = null
      this.userName = null
    },

    clearSessionExpired() {
      this.sessionExpired = false
    },

    async refreshToken(): Promise<boolean> {
      if (!this.token || this.refreshInProgress) {
        return false
      }

      this.refreshInProgress = true
      try {
        const response = await fetch(`${apiBaseUrl}/auth/refresh`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${this.token}`,
          },
        })

        if (!response.ok) {
          // Token refresh failed - user needs to log in again
          this.sessionExpired = true
          this.logout()
          return false
        }

        const data: RefreshResponse = await response.json()
        const { token, expires_at, user_id, user_name } = data.response
        this.setSession(token, expires_at, user_id, user_name)
        return true
      } catch {
        this.sessionExpired = true
        this.logout()
        return false
      } finally {
        this.refreshInProgress = false
      }
    },
  },

  // Only persist auth credentials, not transient state.
  // Excluded: refreshInProgress (would appear stuck if page closed mid-refresh),
  //           sessionExpired (should reset on page reload)
  persist: {
    paths: ['token', 'expiresAt', 'userId', 'userName'],
  },
})