<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { initiateLogin } from '@/auth/iracing'
import { useSessionStore } from '@/stores/session'

const session = useSessionStore()
const router = useRouter()
const menuOpen = ref(false)

async function handleLogin() {
  try {
    await initiateLogin()
  } catch (error) {
    console.error('Login failed:', error)
  }
}

function handleLogout() {
  session.logout()
  menuOpen.value = false
}

function closeMenu() {
  menuOpen.value = false
}

router.afterEach(() => {
  menuOpen.value = false
})
</script>

<template>
  <nav class="navbar">
    <RouterLink to="/" class="brand">Saturday's Spinout</RouterLink>

    <button
      class="menu-toggle"
      :class="{ open: menuOpen }"
      @click="menuOpen = !menuOpen"
      aria-label="Toggle menu"
    >
      <span></span>
      <span></span>
      <span></span>
    </button>

    <div class="nav-links" :class="{ open: menuOpen }">
      <template v-if="session.isLoggedIn">
        <RouterLink to="/race-history" class="nav-link" @click="closeMenu">Race History</RouterLink>
        <RouterLink to="/iracing-api" class="nav-link" @click="closeMenu">iRacing API Explorer</RouterLink>
        <span class="user-name">{{ session.userName }}</span>
        <button @click="handleLogout" class="nav-button">Logout</button>
      </template>
      <template v-else>
        <button @click="handleLogin" class="nav-button primary">Login with iRacing</button>
      </template>
    </div>
  </nav>
</template>

<style scoped>
.navbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 1.5rem;
  height: 56px;
  background: var(--color-bg-surface);
  border-bottom: 1px solid var(--color-border);
}

.brand {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--color-text-primary);
  text-decoration: none;
}

.brand:hover {
  color: var(--color-accent);
}

.nav-links {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.nav-link {
  color: var(--color-text-secondary);
  text-decoration: none;
  padding: 0.5rem 0.75rem;
  border-radius: 4px;
  transition: color 0.15s, background 0.15s;
}

.nav-link:hover {
  color: var(--color-text-primary);
  background: var(--color-accent-subtle);
}

.nav-link.router-link-active {
  color: var(--color-accent);
  background: var(--color-accent-muted);
}

.user-name {
  color: var(--color-text-secondary);
  font-size: 0.875rem;
}

.nav-button {
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
  border: 1px solid var(--color-border-light);
  border-radius: 4px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.nav-button:hover {
  border-color: var(--color-accent);
  color: var(--color-text-primary);
}

.nav-button.primary {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: var(--color-bg-deep);
  font-weight: 500;
}

.nav-button.primary:hover {
  background: var(--color-accent-hover);
  border-color: var(--color-accent-hover);
}

/* Hamburger menu toggle - hidden on desktop */
.menu-toggle {
  display: none;
  flex-direction: column;
  justify-content: center;
  gap: 5px;
  width: 32px;
  height: 32px;
  padding: 4px;
  background: transparent;
  border: none;
  cursor: pointer;
}

.menu-toggle span {
  display: block;
  width: 100%;
  height: 2px;
  background: var(--color-text-secondary);
  border-radius: 1px;
  transition: transform 0.2s, opacity 0.2s;
}

.menu-toggle:hover span {
  background: var(--color-text-primary);
}

.menu-toggle.open span:nth-child(1) {
  transform: translateY(7px) rotate(45deg);
}

.menu-toggle.open span:nth-child(2) {
  opacity: 0;
}

.menu-toggle.open span:nth-child(3) {
  transform: translateY(-7px) rotate(-45deg);
}

/* Mobile styles */
@media (max-width: 768px) {
  .menu-toggle {
    display: flex;
  }

  .nav-links {
    position: absolute;
    top: 56px;
    left: 0;
    right: 0;
    flex-direction: column;
    align-items: stretch;
    gap: 0;
    padding: 0.5rem;
    background: var(--color-bg-surface);
    border-bottom: 1px solid var(--color-border);
    display: none;
  }

  .nav-links.open {
    display: flex;
  }

  .nav-link {
    padding: 0.75rem 1rem;
  }

  .user-name {
    padding: 0.75rem 1rem;
    border-top: 1px solid var(--color-border);
    margin-top: 0.5rem;
  }

  .nav-button {
    margin: 0.5rem;
  }
}
</style>