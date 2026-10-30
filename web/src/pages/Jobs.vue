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
        <q-tabs v-model="view" dense no-caps inline-label class="views" align="left">
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
            <span class="gt-xs">
              <a v-if="p.row.url" class="row-link" :href="p.row.url" target="_blank" rel="noopener noreferrer">Abrir</a>
              <q-btn v-if="p.row.url && p.row.status !== 'expired'" flat dense no-caps size="md" label="Candidatar"
                     :to="{ path: '/claude', query: { command: 'apply', args: p.row.url } }" />
              <q-btn v-if="p.row.status === 'skipped'" flat dense no-caps label="Restaurar" @click="restore(p.row)" />
              <q-btn v-else flat round dense size="sm" icon="o_close" :aria-label="`Remover ${p.row.title} do painel`" @click="remove(p.row)">
                <q-tooltip>Remover do painel</q-tooltip>
              </q-btn>
            </span>
            <q-btn class="lt-sm" flat round dense icon="o_more_horiz" :aria-label="`Ações para ${p.row.title}`">
              <q-menu auto-close>
                <q-list style="min-width: 180px">
                  <q-item v-if="p.row.url" clickable tag="a" :href="p.row.url" target="_blank" rel="noopener noreferrer"><q-item-section>Abrir vaga</q-item-section></q-item>
                  <q-item v-if="p.row.url && p.row.status !== 'expired'" clickable :to="{ path: '/claude', query: { command: 'apply', args: p.row.url } }"><q-item-section>Candidatar</q-item-section></q-item>
                  <q-item v-if="p.row.status === 'skipped'" clickable @click="restore(p.row)"><q-item-section>Restaurar</q-item-section></q-item>
                  <q-item v-else clickable @click="remove(p.row)"><q-item-section>Remover do painel</q-item-section></q-item>
                </q-list>
              </q-menu>
            </q-btn>
          </q-td>
        </template>
      </q-table>
    </section>
  </q-page>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { Notify } from 'quasar'
import { get, post, deadlineText } from '../api.js'

const rows = ref([])
const loading = ref(true)
const filter = ref('')
const view = ref('ranked')

// rank_score is written by tools/rank_state.py; entries not yet ranked have none.
const score = r => typeof r.rank_score === 'number' ? Math.round(r.rank_score) : null
const loc = r => typeof r.location === 'string' ? r.location : ''
const LABELS = { ranked: 'Pontuada', new: 'Nova', skipped: 'Removida', expired: 'Expirada', unreachable: 'Fora do ar' }
const statusLabel = s => LABELS[s] || s || '—'

const views = computed(() => [
  { id: 'ranked', label: 'Pontuadas', rows: rows.value.filter(r => r.status === 'ranked') },
  // /rank picks up "new" and "unreachable"; "skipped" is what the scraper (and Remover) set aside.
  { id: 'new', label: 'Não pontuadas', rows: rows.value.filter(r => ['new', 'unreachable'].includes(r.status)) },
  { id: 'all', label: 'Todas', rows: rows.value.filter(r => r.status !== 'skipped') },
  { id: 'skipped', label: 'Removidas', rows: rows.value.filter(r => r.status === 'skipped') }
])
const shown = computed(() => views.value.find(v => v.id === view.value).rows)

async function setStatus (row, status) {
  await post('/api/jobs/status', { key: row.key, status })
  row.status = status
}

// Removing is reversible: the entry stays in seen_jobs.json as "skipped", so /scrape won't bring it back.
async function remove (row) {
  const before = row.status
  await setStatus(row, 'skipped')
  Notify.create({
    message: `${row.title} removida do painel.`,
    color: 'dark',
    actions: [{ label: 'Desfazer', color: 'white', handler: () => setStatus(row, before) }]
  })
}

// The status it had before is not stored, so a restored posting goes back to the /rank queue.
const restore = row => setStatus(row, typeof row.rank_score === 'number' ? 'ranked' : 'new')

const columns = [
  { name: 'title', label: 'Vaga', field: r => `${r.title} ${r.company} ${loc(r)}`, sortable: true, align: 'left' },
  { name: 'status', label: 'Situação', field: 'status', sortable: true, align: 'left', classes: 'gt-xs', headerClasses: 'gt-xs' },
  { name: 'score', label: 'Fit', field: score, sortable: true, align: 'right', sort: (a, b) => (a ?? -1) - (b ?? -1) },
  { name: 'deadline', label: 'Prazo', field: 'deadline', sortable: true, align: 'right', classes: 'gt-xs', headerClasses: 'gt-xs' },
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
.views :deep(.q-tab__indicator) { background: var(--text); }
.views :deep(.q-tab__label) { font-size: 14px; font-weight: 500; }
@media (max-width: 1023px) {
  .views { width: 100%; }
}
</style>
