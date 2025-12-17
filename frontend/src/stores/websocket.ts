import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import { useAuthStore } from './auth'

const wsBaseUrl = import.meta.env.VITE_WS_BASE_URL || 'ws://localhost:8081'
const HEARTBEAT_INTERVAL_MS = 120000

type ConnectionStatus = 'disconnected' | 'connecting' | 'authenticating' | 'connected' | 'error'

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

export const useWebSocketStore = defineStore('websocket', () => {
  // Public state
  const status = ref<ConnectionStatus>('disconnected')
  const error = ref<string | null>(null)
  const lastPong = ref<number | null>(null)
  const driverId = ref<number | null>(null)
  const connectionId = ref<string | null>(null)

  // Private state (not exposed)
  let socket: WebSocket | null = null
  let reconnectTimeout: ReturnType<typeof setTimeout> | null = null
  let heartbeatInterval: ReturnType<typeof setInterval> | null = null
  const listeners = new Map<string, Set<(payload: unknown) => void>>()

  // Private methods
  function clearReconnectTimeout() {
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout)
      reconnectTimeout = null
    }
  }

  function stopHeartbeat() {
    if (heartbeatInterval) {
      console.log('[WS] Stopping heartbeat')
      clearInterval(heartbeatInterval)
      heartbeatInterval = null
    }
  }

  function sendPing() {
    if (socket?.readyState === WebSocket.OPEN && driverId.value) {
      const msg = { action: 'pingRequest', driverId: driverId.value }
      console.log('[WS] Sending ping')
      socket.send(JSON.stringify(msg))
    }
  }

  function startHeartbeat() {
    stopHeartbeat()
    console.log('[WS] Starting heartbeat')
    heartbeatInterval = setInterval(() => {
      sendPing()
    }, HEARTBEAT_INTERVAL_MS)
    // Send first ping immediately
    sendPing()
  }

  function sendAuth() {
    const authStore = useAuthStore()
    if (socket?.readyState === WebSocket.OPEN && authStore.token) {
      const msg: AuthMessage = {
        action: 'auth',
        token: authStore.token,
      }
      console.log('[WS] Sending auth')
      socket.send(JSON.stringify(msg))
    }
  }

  function handleAuthResponse(response: AuthResponse) {
    if (response.success && response.userId && response.connectionId) {
      console.log('[WS] Authenticated as user:', response.userId, 'connection:', response.connectionId)
      status.value = 'connected'
      error.value = null
      driverId.value = response.userId
      connectionId.value = response.connectionId
      startHeartbeat()
    } else {
      console.error('[WS] Auth failed:', response.error)
      status.value = 'error'
      error.value = response.error || 'Authentication failed'
    }
  }

  function handleMessage(msg: Message) {
    console.log('[WS] Received:', msg.action)

    // Handle core protocol messages
    switch (msg.action) {
      case 'authResponse':
        handleAuthResponse(msg.payload as AuthResponse)
        break
      case 'pong':
        lastPong.value = Date.now()
        console.log('[WS] Pong received')
        break
      case 'error':
        console.error('[WS] Server error:', msg.payload)
        break
    }

    // Dispatch to registered listeners
    const actionListeners = listeners.get(msg.action)
    if (actionListeners) {
      actionListeners.forEach((cb) => cb(msg.payload))
    }
  }

  function scheduleReconnect() {
    clearReconnectTimeout()
    console.log('[WS] Scheduling reconnect in 3s...')
    reconnectTimeout = setTimeout(() => {
      const authStore = useAuthStore()
      if (authStore.isLoggedIn) {
        connect()
      }
    }, 3000)
  }

  // Public methods
  function on(actionType: string, callback: (payload: unknown) => void) {
    if (!listeners.has(actionType)) {
      listeners.set(actionType, new Set())
    }
    listeners.get(actionType)!.add(callback)
  }

  function off(actionType: string, callback: (payload: unknown) => void) {
    listeners.get(actionType)?.delete(callback)
  }

  function connect() {
    const authStore = useAuthStore()

    if (!authStore.token) {
      console.log('[WS] No token, skipping connection')
      return
    }

    if (socket?.readyState === WebSocket.OPEN ||
        socket?.readyState === WebSocket.CONNECTING) {
      console.log('[WS] Already connected or connecting')
      return
    }

    status.value = 'connecting'
    error.value = null

    console.log('[WS] Connecting...')

    socket = new WebSocket(wsBaseUrl)

    socket.onopen = () => {
      console.log('[WS] Socket open, sending auth...')
      status.value = 'authenticating'
      sendAuth()
    }

    socket.onclose = (event) => {
      console.log(`[WS] Closed: code=${event.code}, reason=${event.reason}`)
      socket = null

      if (status.value !== 'disconnected') {
        // Unexpected close - try to reconnect if still logged in
        status.value = 'disconnected'
        if (authStore.isLoggedIn) {
          scheduleReconnect()
        }
      }
    }

    socket.onerror = (event) => {
      console.error('[WS] Error:', event)
      status.value = 'error'
      error.value = 'WebSocket connection failed'
    }

    socket.onmessage = (event) => {
      try {
        const msg: Message = JSON.parse(event.data)
        handleMessage(msg)
      } catch (err) {
        console.error('[WS] Failed to parse message:', err, event.data)
      }
    }
  }

  function disconnect() {
    console.log('[WS] Disconnecting...')
    clearReconnectTimeout()
    stopHeartbeat()

    if (socket) {
      socket.close(1000, 'Client disconnect')
      socket = null
    }

    status.value = 'disconnected'
    error.value = null
    lastPong.value = null
    driverId.value = null
    connectionId.value = null
  }

  function reconnect() {
    console.log('[WS] Reconnecting with new token...')
    disconnect()
    connect()
  }

  return {
    // State
    status,
    error,
    lastPong,
    driverId,
    connectionId,
    // Actions
    on,
    off,
    connect,
    disconnect,
    reconnect,
  }
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