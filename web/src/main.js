import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import { Quasar, Notify, Dialog } from 'quasar'
import langPtBR from 'quasar/lang/pt-BR'
import iconSet from 'quasar/icon-set/material-icons-outlined'
import '@quasar/extras/material-icons-outlined/material-icons-outlined.css'
import 'quasar/src/css/index.sass'
import './app.css'

import App from './App.vue'
import Applications from './pages/Applications.vue'
import Jobs from './pages/Jobs.vue'
import Search from './pages/Search.vue'
import Claude from './pages/Claude.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/candidaturas' },
    { path: '/candidaturas', component: Applications },
    { path: '/vagas', component: Jobs },
    { path: '/buscar', component: Search },
    { path: '/claude/:id?', component: Claude }
  ]
})

createApp(App)
  .use(Quasar, {
    plugins: { Notify, Dialog },
    lang: langPtBR,
    iconSet,
    // The chrome is ink: the one primary action is near-black, never a brand hue.
    config: { dark: false, brand: { primary: '#141414', secondary: '#e5e5e5', negative: '#c4242a', positive: '#2e7d32' } }
  })
  .use(router)
  .mount('#app')
