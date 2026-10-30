<template>
  <q-page class="canvas column no-wrap">
    <div class="title-row">
      <div>
        <h1>Buscar nos portais</h1>
        <p class="finding">
          Pesquisa direto em um portal, pelos buscadores do ai-job-search. Para buscar em todos e guardar o histórico, use
          <router-link :to="{ path: '/claude', query: { command: 'scrape' } }">/scrape</router-link>.
        </p>
      </div>
    </div>

    <section class="panel results">
      <q-form class="search-form" @submit="search">
        <q-select v-model="portal" :options="portals" :option-label="portalName" label="Portal" outlined dense emit-value map-options class="portal" />
        <q-input v-model="query" label="Cargo, área ou palavra-chave" outlined dense class="query" autofocus />
        <q-btn type="submit" unelevated color="primary" label="Buscar" :loading="loading" :disable="!portal || !query.trim()" />
      </q-form>

      <div v-if="!searched" class="empty">
        <p>Escolha um dos {{ portals.length }} portais instalados no ai-job-search e busque.</p>
        <div class="portal-list">
          <q-btn v-for="p in portals" :key="p" unelevated no-caps :class="p === portal ? 'bg-primary text-white' : 'btn-secondary'"
                 :label="portalName(p)" :aria-pressed="p === portal" @click="portal = p" />
        </div>
      </div>
      <q-table v-else class="ledger" flat :rows="rows" :columns="columns" row-key="_i" :pagination="{ rowsPerPage: 20 }"
               :rows-per-page-options="[20, 50, 0]" hide-bottom-space
               :no-data-label="searched ? 'Nenhuma vaga para essa busca.' : 'Os resultados aparecem aqui.'">
        <template #body-cell-title="p">
          <q-td :props="p">
            <span class="cell-main">{{ p.value || '—' }}</span>
            <span class="cell-sub">{{ [pick(p.row, 'company', 'companyName', 'employer'), pick(p.row, 'location', 'area', 'city', 'workplace')].filter(Boolean).join(' · ') }}</span>
          </q-td>
        </template>
        <template #body-cell-actions="p">
          <q-td :props="p">
            <a v-if="p.row._url" class="row-link" :href="p.row._url" target="_blank" rel="noopener noreferrer">Abrir</a>
            <q-btn v-if="p.row._url" flat dense no-caps label="Candidatar" :to="{ path: '/claude', query: { command: 'apply', args: p.row._url } }" />
          </q-td>
        </template>
      </q-table>
    </section>
  </q-page>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { get } from '../api.js'

const portals = ref([])
const portal = ref(null)
const query = ref('')
const rows = ref([])
const loading = ref(false)
const searched = ref(false)

const portalName = p => p ? p.replace(/-search$/, '').replace(/^./, c => c.toUpperCase()) : ''
// Each portal CLI shapes its JSON a little differently; take the first field that exists.
const pick = (o, ...keys) => keys.map(k => o[k]).find(v => typeof v === 'string' && v) || ''
const columns = [
  { name: 'title', label: 'Vaga', field: r => pick(r, 'title', 'headline', 'jobTitle', 'name'), align: 'left' },
  { name: 'date', label: 'Publicada', field: r => pick(r, 'date', 'published', 'postedAt', 'publishedDate').slice(0, 10) || '—', align: 'right', classes: 'num muted' },
  { name: 'actions', label: '', field: () => '', align: 'right' }
]

onMounted(async () => {
  portals.value = await get('/api/portals')
  portal.value = portals.value[0] ?? null
})

async function search () {
  loading.value = true
  try {
    const data = await get(`/api/search?portal=${encodeURIComponent(portal.value)}&q=${encodeURIComponent(query.value)}`)
    const list = Array.isArray(data) ? data : data.results || data.jobs || data.items || []
    rows.value = list.map((r, i) => ({ ...r, _i: i, _url: pick(r, 'url', 'link', 'jobUrl', 'href') }))
    searched.value = true
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.results { flex: 1; display: flex; flex-direction: column; }
.empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 16px; padding: 48px 16px; text-align: center; color: var(--muted); }
.empty p { margin: 0; font-size: 15px; }
.portal-list { display: flex; flex-wrap: wrap; justify-content: center; gap: 8px; max-width: 560px; }
.search-form { display: grid; grid-template-columns: 220px 1fr auto; gap: var(--gap); padding: 16px; border-bottom: 1px solid var(--rule); }
.search-form .q-btn { height: 40px; padding: 0 20px; }
@media (max-width: 599px) {
  .search-form { grid-template-columns: 1fr; }
}
</style>
