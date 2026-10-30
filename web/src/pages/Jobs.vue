<template>
  <q-page class="canvas">
    <div class="title-row">
      <div>
        <h1>Vagas encontradas</h1>
        <p class="finding" v-html="finding" />
      </div>
      <div class="actions">
        <q-btn unelevated color="primary" icon="o_travel_explore" label="Buscar vagas novas" :to="{ path: '/claude', query: { command: 'scrape' } }" />
        <q-btn unelevated class="btn-secondary" label="Pontuar com /rank" :to="{ path: '/claude', query: { command: 'rank' } }" />
      </div>
    </div>

    <section class="panel">
      <div class="panel-head">
        <q-tabs v-model="view" dense no-caps inline-label active-color="dark" indicator-color="dark" class="views" align="left">
          <q-tab v-for="v in views" :key="v.id" :name="v.id" :label="`${v.label} ${v.rows.length}`" />
        </q-tabs>
        <q-input v-model="filter" dense outlined debounce="200" placeholder="Filtrar" aria-label="Filtrar" style="width: 220px">
          <template #prepend><q-icon name="o_search" size="18px" /></template>
        </q-input>
      </div>
      <q-table class="ledger" flat :rows="shown" :columns="columns" row-key="key" :loading="loading"
               :filter="filter" :pagination="{ rowsPerPage: 25, sortBy: 'score', descending: true }"
               :rows-per-page-options="[25, 50, 0]" hide-bottom-space
               :no-data-label="rows.length ? 'Nada nesta visão.' : 'Nenhuma vaga ainda. Use Buscar vagas novas.'">
        <template #body-cell-title="p">
          <q-td :props="p">
            <span class="cell-main">{{ p.row.title }}</span>
            <span class="cell-sub">{{ p.row.company }}<template v-if="loc(p.row)"> · {{ loc(p.row) }}</template></span>
          </q-td>
        </template>
        <template #body-cell-status="p">
          <q-td :props="p"><span class="pill neutral">{{ statusLabel(p.value) }}</span></q-td>
        </template>
        <template #body-cell-score="p">
          <q-td :props="p">
            <span v-if="p.value !== null" class="fit"><span class="fit-bar"><i :style="{ width: p.value + '%' }" /></span><span class="fit-num">{{ p.value }}</span></span>
            <span v-else class="muted">—</span>
          </q-td>
        </template>
        <template #body-cell-deadline="p">
          <q-td :props="p">
            <span v-if="!p.value" class="muted">—</span>
            <span v-else :class="deadlineText(p.value).tone ? 'pill ' + deadlineText(p.value).tone : ''">{{ deadlineText(p.value).text }}</span>
          </q-td>
        </template>
        <template #body-cell-actions="p">
          <q-td :props="p">
            <a v-if="p.row.url" class="row-link" :href="p.row.url" target="_blank" rel="noopener noreferrer">Abrir</a>
            <q-btn v-if="p.row.url && p.row.status !== 'expired'" flat dense no-caps size="md" label="Candidatar"
                   :to="{ path: '/claude', query: { command: 'apply', args: p.row.url } }" />
          </q-td>
        </template>
      </q-table>
    </section>
  </q-page>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { get, deadlineText } from '../api.js'

const rows = ref([])
const loading = ref(true)
const filter = ref('')
const view = ref('ranked')

// rank_score is written by tools/rank_state.py; entries not yet ranked have none.
const score = r => typeof r.rank_score === 'number' ? Math.round(r.rank_score) : null
const loc = r => typeof r.location === 'string' ? r.location : ''
const LABELS = { ranked: 'Pontuada', scraped: 'Não pontuada', expired: 'Expirada', unreachable: 'Fora do ar' }
const statusLabel = s => LABELS[s] || s || '—'

const views = computed(() => [
  { id: 'ranked', label: 'Pontuadas', rows: rows.value.filter(r => r.status === 'ranked') },
  { id: 'new', label: 'Não pontuadas', rows: rows.value.filter(r => !['ranked', 'expired', 'unreachable'].includes(r.status)) },
  { id: 'all', label: 'Todas', rows: rows.value }
])
const shown = computed(() => views.value.find(v => v.id === view.value).rows)

const columns = [
  { name: 'title', label: 'Vaga', field: r => `${r.title} ${r.company} ${loc(r)}`, sortable: true, align: 'left' },
  { name: 'status', label: 'Situação', field: 'status', sortable: true, align: 'left', classes: 'gt-xs', headerClasses: 'gt-xs' },
  { name: 'score', label: 'Fit', field: score, sortable: true, align: 'right', sort: (a, b) => (a ?? -1) - (b ?? -1) },
  { name: 'deadline', label: 'Prazo', field: 'deadline', sortable: true, align: 'right' },
  { name: 'actions', label: '', field: () => '', align: 'right' }
]

const finding = computed(() => {
  if (loading.value) return 'Carregando…'
  if (!rows.value.length) return 'O /scrape guarda aqui cada vaga que encontra nos portais.'
  const ranked = views.value[0].rows
  const fresh = views.value[1].rows.length
  const strong = ranked.filter(r => score(r) >= 80).length
  const parts = [`<b>${strong}</b> de ${ranked.length} pontuadas com fit 80 ou mais`]
  if (fresh) parts.push(`${fresh} ainda sem pontuação`)
  return parts.join(', ') + '.'
})

onMounted(async () => {
  try {
    rows.value = await get('/api/jobs')
    if (!views.value[0].rows.length) view.value = 'all'
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.views :deep(.q-tab) { padding: 0 10px; min-height: 36px; color: var(--muted); }
.views :deep(.q-tab--active) { color: var(--text); }
.views :deep(.q-tab__label) { font-size: 14px; font-weight: 500; }
</style>
