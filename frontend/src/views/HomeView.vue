<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
</script>

<template>
  <div class="home">
    <div class="hero">
      <h1>Saturday's Spinout</h1>
      <p class="tagline">Race logging and analysis for iRacing</p>

      <div v-if="authStore.isLoggedIn" class="welcome">
        <p>Welcome back, {{ authStore.userName }}!</p>
        <RouterLink to="/iracing-api" class="cta-button">
          Explore iRacing API
        </RouterLink>
      </div>

      <div v-else class="guest">
        <p>Sign in with your iRacing account to get started</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.home {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: calc(100vh - 120px);
}

.hero {
  text-align: center;
  max-width: 600px;
}

h1 {
  font-size: 3rem;
  margin: 0 0 0.5rem;
  color: var(--color-accent);
}

.tagline {
  font-size: 1.25rem;
  color: var(--color-text-secondary);
  margin: 0 0 2rem;
}

.welcome p,
.guest p {
  color: var(--color-text-secondary);
  margin-bottom: 1.5rem;
}

.cta-button {
  display: inline-block;
  padding: 0.75rem 1.5rem;
  background: var(--color-accent);
  color: var(--color-bg-deep);
  text-decoration: none;
  border-radius: 6px;
  font-weight: 500;
  transition: background 0.15s;
}

.cta-button:hover {
  background: var(--color-accent-hover);
}

@media (max-width: 768px) {
  .hero {
    padding: 0 1rem;
  }

  h1 {
    font-size: 2.25rem;
  }

  .tagline {
    font-size: 1.1rem;
  }
}

@media (max-width: 480px) {
  h1 {
    font-size: 1.75rem;
  }

  .tagline {
    font-size: 1rem;
  }
}
</style>