<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { retrieveCodeVerifier, clearCodeVerifier } from '@/auth/pkce'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()
const status = ref<'processing' | 'error'>('processing')
const errorMessage = ref('')

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

interface AuthCallbackData {
  token: string
  expires_at: number
  user_id: number
  user_name: string
}

interface ApiResponse<T> {
  response: T
  correlationId: string
}

onMounted(async () => {
  const params = new URLSearchParams(window.location.search)
  const code = params.get('code')
  const error = params.get('error')
  const errorDescription = params.get('error_description')

  if (error) {
    status.value = 'error'
    errorMessage.value = errorDescription || error
    return
  }

  if (!code) {
    status.value = 'error'
    errorMessage.value = 'No authorization code received'
    return
  }

  const codeVerifier = retrieveCodeVerifier()
  if (!codeVerifier) {
    status.value = 'error'
    errorMessage.value = 'No code verifier found - please try logging in again'
    return
  }

  const redirectUri = `${window.location.origin}/auth/ir/callback`

  try {
    const response = await fetch(`${apiBaseUrl}/auth/ir/callback`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        code,
        code_verifier: codeVerifier,
        redirect_uri: redirectUri,
      }),
    })

    if (!response.ok) {
      const data = await response.json().catch(() => ({}))
      throw new Error(data.error || `Token exchange failed: ${response.status}`)
    }

    const data: ApiResponse<AuthCallbackData> = await response.json()
    clearCodeVerifier()

    const { token, expires_at, user_id, user_name } = data.response
    authStore.setSession(token, expires_at, user_id, user_name)

    router.push('/')
  } catch (err) {
    status.value = 'error'
    errorMessage.value = err instanceof Error ? err.message : 'Token exchange failed'
  }
})
</script>

<template>
  <main>
    <div v-if="status === 'processing'">
      <h1>Logging in...</h1>
      <p>Please wait while we complete your login.</p>
    </div>
    <div v-else-if="status === 'error'">
      <h1>Login Failed</h1>
      <p class="error">{{ errorMessage }}</p>
      <button @click="router.push('/')">Back to Home</button>
    </div>
  </main>
</template>

<style scoped>
main {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
}

h1 {
  font-size: 2rem;
  margin-bottom: 0.5rem;
}

p {
  color: #666;
}

.error {
  color: #c00;
}

button {
  margin-top: 1rem;
  padding: 0.5rem 1rem;
  font-size: 1rem;
  cursor: pointer;
}
</style>