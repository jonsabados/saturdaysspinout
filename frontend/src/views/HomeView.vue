<script setup lang="ts">
import { initiateLogin } from '@/auth/iracing'
import { useAuthStore } from '@/stores/auth'
import { useApiClient } from '@/api/client'

const authStore = useAuthStore()
const apiClient = useApiClient()

async function handleLogin() {
  try {
    await initiateLogin()
  } catch (error) {
    console.error('Login failed:', error)
    alert(`Login failed: ${error}`)
  }
}

function handleLogout() {
  authStore.logout()
}

async function testDocProxy() {
  try {
    const data = await apiClient.fetch<Record<string, unknown>>('/doc/iracing-api/')
    alert(JSON.stringify(data, null, 2))
  } catch (error) {
    alert(`Error: ${error}`)
  }
}
</script>

<template>
  <main>
    <h1>Saturday's Spinout</h1>

    <template v-if="authStore.isLoggedIn">
      <p>Welcome, {{ authStore.userName }}!</p>
      <button @click="testDocProxy">Test Doc Proxy</button>
      <button @click="handleLogout">Logout</button>
    </template>

    <template v-else>
      <p>Sign in to get started</p>
      <button @click="handleLogin">Login with iRacing</button>
    </template>
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
  font-size: 3rem;
  margin-bottom: 0.5rem;
}

p {
  color: #666;
}

button {
  margin-top: 1rem;
  padding: 0.5rem 1rem;
  font-size: 1rem;
  cursor: pointer;
}
</style>