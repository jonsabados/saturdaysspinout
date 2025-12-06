<script setup lang="ts">
const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

async function makeTestRequest() {
  console.log('Making test request to', `${apiBaseUrl}/health/ping`)
  try {
    const response = await fetch(`${apiBaseUrl}/health/ping`)
    console.log('Response status:', response.status)
    console.log('Response headers:', Object.fromEntries(response.headers.entries()))
    const data = await response.json()
    console.log('Response data:', data)
  } catch (error) {
    console.error('Request failed:', error)
  }
}
</script>

<template>
  <main>
    <h1>Hello World</h1>
    <p>Saturday's Racelog</p>
    <button @click="makeTestRequest">Make Test Request</button>
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