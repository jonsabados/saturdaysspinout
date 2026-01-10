import { createApp } from 'vue'
import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'

import App from './App.vue'
import router from './router'
import { i18n } from './i18n'
import { setupWebSocketAutoConnect } from './stores/websocket'
import { setupRaceIngestionListener } from './stores/raceIngestion'
import { setupDriverListener } from './stores/driver'
import { setupAnalyticsListener } from './stores/analytics'
import './assets/variables.css'

const pinia = createPinia()
pinia.use(piniaPluginPersistedstate)

const app = createApp(App)

app.use(pinia)
app.use(router)
app.use(i18n)

app.mount('#app')

setupWebSocketAutoConnect()
setupRaceIngestionListener()
setupDriverListener()
setupAnalyticsListener()