import { defineStore } from 'pinia'

interface AuthState {
  token: string | null
  expiresAt: number | null
  userId: number | null
  userName: string | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    token: null,
    expiresAt: null,
    userId: null,
    userName: null,
  }),

  getters: {
    isLoggedIn(): boolean {
      if (!this.token || !this.expiresAt) {
        return false
      }
      // Check if token is expired (expiresAt is Unix timestamp in seconds)
      return this.expiresAt > Date.now() / 1000
    },
  },

  actions: {
    setSession(token: string, expiresAt: number, userId?: number, userName?: string) {
      this.token = token
      this.expiresAt = expiresAt
      this.userId = userId ?? null
      this.userName = userName ?? null
    },

    logout() {
      this.token = null
      this.expiresAt = null
      this.userId = null
      this.userName = null
    },
  },

  persist: true,
})