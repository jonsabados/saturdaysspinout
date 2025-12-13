import { defineStore } from 'pinia'
import { watch } from 'vue'
import { useAuthStore } from './auth'

const wsBaseUrl = import.meta.env.VITE_WS_BASE_URL || 'ws://localhost:8081'
const HEARTBEAT_INTERVAL_MS = 30000

type ConnectionStatus = 'disconnected' | 'connecting' | 'authenticating' | 'connected' | 'error'

interface WebSocketState {
  status: ConnectionStatus
  error: string | null
  lastPong: number | null
  driverId: number | null
  connectionId: string | null
}

interface Message {
  action: string
  payload?: unknown
}

interface AuthMessage {
  action: 'auth'
  token: string
}

interface AuthResponse {
  success: boolean
  userId?: number
  connectionId?: string
  error?: string
}

export const useWebSocketStore = defineStore('websocket', {
  state: (): WebSocketState => ({
    status: 'disconnected',
    error: null,
    lastPong: null,
    driverId: null,
    connectionId: null,
  }),

  actions: {
    _socket: null as WebSocket | null,
    _reconnectTimeout: null as ReturnType<typeof setTimeout> | null,
    _heartbeatInterval: null as ReturnType<typeof setInterval> | null,

    connect() {
      const authStore = useAuthStore()

      if (!authStore.token) {
        console.log('[WS] No token, skipping connection')
        return
      }

      if (this._socket?.readyState === WebSocket.OPEN ||
          this._socket?.readyState === WebSocket.CONNECTING) {
        console.log('[WS] Already connected or connecting')
        return
      }

      this.status = 'connecting'
      this.error = null

      console.log('[WS] Connecting...')

      this._socket = new WebSocket(wsBaseUrl)

      this._socket.onopen = () => {
        console.log('[WS] Socket open, sending auth...')
        this.status = 'authenticating'
        this._sendAuth()
      }

      this._socket.onclose = (event) => {
        console.log(`[WS] Closed: code=${event.code}, reason=${event.reason}`)
        this._socket = null

        if (this.status !== 'disconnected') {
          // Unexpected close - try to reconnect if still logged in
          this.status = 'disconnected'
          if (authStore.isLoggedIn) {
            this._scheduleReconnect()
          }
        }
      }

      this._socket.onerror = (event) => {
        console.error('[WS] Error:', event)
        this.status = 'error'
        this.error = 'WebSocket connection failed'
      }

      this._socket.onmessage = (event) => {
        try {
          const msg: Message = JSON.parse(event.data)
          this._handleMessage(msg)
        } catch (err) {
          console.error('[WS] Failed to parse message:', err, event.data)
        }
      }
    },

    disconnect() {
      console.log('[WS] Disconnecting...')
      this._clearReconnectTimeout()
      this._stopHeartbeat()

      if (this._socket) {
        this._socket.close(1000, 'Client disconnect')
        this._socket = null
      }

      this.status = 'disconnected'
      this.error = null
      this.lastPong = null
      this.driverId = null
      this.connectionId = null
    },

    reconnect() {
      console.log('[WS] Reconnecting with new token...')
      this.disconnect()
      this.connect()
    },

    _scheduleReconnect() {
      this._clearReconnectTimeout()
      console.log('[WS] Scheduling reconnect in 3s...')
      this._reconnectTimeout = setTimeout(() => {
        const authStore = useAuthStore()
        if (authStore.isLoggedIn) {
          this.connect()
        }
      }, 3000)
    },

    _clearReconnectTimeout() {
      if (this._reconnectTimeout) {
        clearTimeout(this._reconnectTimeout)
        this._reconnectTimeout = null
      }
    },

    _startHeartbeat() {
      this._stopHeartbeat()
      console.log('[WS] Starting heartbeat')
      this._heartbeatInterval = setInterval(() => {
        this._sendPing()
      }, HEARTBEAT_INTERVAL_MS)
      // Send first ping immediately
      this._sendPing()
    },

    _stopHeartbeat() {
      if (this._heartbeatInterval) {
        console.log('[WS] Stopping heartbeat')
        clearInterval(this._heartbeatInterval)
        this._heartbeatInterval = null
      }
    },

    _sendPing() {
      if (this._socket?.readyState === WebSocket.OPEN && this.driverId) {
        const msg = { action: 'pingRequest', driverId: this.driverId }
        console.log('[WS] Sending ping')
        this._socket.send(JSON.stringify(msg))
      }
    },

    _sendAuth() {
      const authStore = useAuthStore()
      if (this._socket?.readyState === WebSocket.OPEN && authStore.token) {
        const msg: AuthMessage = {
          action: 'auth',
          token: authStore.token,
        }
        console.log('[WS] Sending auth')
        this._socket.send(JSON.stringify(msg))
      }
    },

    _handleMessage(msg: Message) {
      console.log('[WS] Received:', msg.action)
      switch (msg.action) {
        case 'authResponse':
          this._handleAuthResponse(msg.payload as AuthResponse)
          break
        case 'pong':
          this.lastPong = Date.now()
          console.log('[WS] Pong received')
          break
        case 'error':
          console.error('[WS] Server error:', msg.payload)
          break
        default:
          console.log('[WS] Unknown action:', msg.action)
      }
    },

    _handleAuthResponse(response: AuthResponse) {
      if (response.success && response.userId && response.connectionId) {
        console.log('[WS] Authenticated as user:', response.userId, 'connection:', response.connectionId)
        this.status = 'connected'
        this.error = null
        this.driverId = response.userId
        this.connectionId = response.connectionId
        this._startHeartbeat()
      } else {
        console.error('[WS] Auth failed:', response.error)
        this.status = 'error'
        this.error = response.error || 'Authentication failed'
      }
    },
  },
})

export function setupWebSocketAutoConnect() {
  const authStore = useAuthStore()
  const wsStore = useWebSocketStore()

  // Watch for login/logout
  watch(
    () => authStore.isLoggedIn,
    (isLoggedIn) => {
      if (isLoggedIn) {
        wsStore.connect()
      } else {
        wsStore.disconnect()
      }
    },
    { immediate: true }
  )

  // Watch for token changes - handles both refresh and late hydration
  watch(
    () => authStore.token,
    (newToken, oldToken) => {
      if (newToken && oldToken && newToken !== oldToken) {
        // Token was refreshed, reconnect with new token
        wsStore.reconnect()
      } else if (newToken && !oldToken && wsStore.status === 'disconnected') {
        // Token appeared (late hydration), connect
        wsStore.connect()
      }
    }
  )
}
