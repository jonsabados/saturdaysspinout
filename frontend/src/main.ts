import { createApp } from 'vue'
import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'

import App from './App.vue'
import router from './router'
import { setupWebSocketAutoConnect } from './stores/websocket'
import { setupRaceIngestionListener } from './stores/raceIngestion'
import { setupDriverListener } from './stores/driver'
import './assets/variables.css'

const pinia = createPinia()
pinia.use(piniaPluginPersistedstate)

const app = createApp(App)

app.use(pinia)
app.use(router)

app.mount('#app')

setupWebSocketAutoConnect()
setupRaceIngestionListener()
setupDriverListener()