<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { initiateLogin } from '@/auth/iracing'
import { useSessionStore } from '@/stores/session'

const session = useSessionStore()
const router = useRouter()
const menuOpen = ref(false)
const moreOpen = ref(false)
const moreDropdown = ref<HTMLElement | null>(null)

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
  moreOpen.value = false
}

function toggleMore() {
  moreOpen.value = !moreOpen.value
}

function handleClickOutside(event: MouseEvent) {
  if (moreDropdown.value && !moreDropdown.value.contains(event.target as Node)) {
    moreOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

router.afterEach(() => {
  menuOpen.value = false
  moreOpen.value = false
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

        <!-- Desktop "More" dropdown -->
        <div ref="moreDropdown" class="more-dropdown desktop-only">
          <button class="nav-link more-toggle" @click.stop="toggleMore" :class="{ active: moreOpen }">
            Tools
            <span class="more-arrow" :class="{ open: moreOpen }">▾</span>
          </button>
          <div class="more-menu" v-show="moreOpen">
            <RouterLink to="/iracing-api" class="more-item" @click="closeMenu">iRacing API Explorer</RouterLink>
          </div>
        </div>

        <!-- Mobile tools section -->
        <div class="mobile-tools mobile-only">
          <div class="tools-divider"></div>
          <span class="tools-header">Tools</span>
          <RouterLink to="/iracing-api" class="nav-link" @click="closeMenu">iRacing API Explorer</RouterLink>
        </div>

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

/* Desktop "More" dropdown */
.more-dropdown {
  position: relative;
}

.more-toggle {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  background: transparent;
  border: none;
  cursor: pointer;
  font-size: inherit;
}

.more-toggle.active {
  color: var(--color-text-primary);
  background: var(--color-accent-subtle);
}

.more-arrow {
  font-size: 0.75rem;
  transition: transform 0.15s;
}

.more-arrow.open {
  transform: rotate(180deg);
}

.more-menu {
  position: absolute;
  top: calc(100% + 0.5rem);
  right: 0;
  min-width: 180px;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  padding: 0.5rem 0;
  z-index: 100;
}

.more-item {
  display: block;
  padding: 0.625rem 1rem;
  color: var(--color-text-secondary);
  text-decoration: none;
  transition: color 0.15s, background 0.15s;
}

.more-item:hover {
  color: var(--color-text-primary);
  background: var(--color-accent-subtle);
}

.more-item.router-link-active {
  color: var(--color-accent);
}

/* Responsive visibility */
.desktop-only {
  display: block;
}

.mobile-only {
  display: none;
}

/* Mobile tools section - hidden on desktop via .mobile-only */
.mobile-tools {
  /* display: contents applied in mobile media query */
}

.tools-divider {
  height: 1px;
  background: var(--color-border);
  margin: 0.5rem 0;
}

.tools-header {
  padding: 0.5rem 1rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
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

  /* Hide desktop dropdown, show mobile tools */
  .desktop-only {
    display: none;
  }

  .mobile-only {
    display: contents;
  }
}
</style>