<template>
  <q-page class="canvas">
    <div class="title-row">
      <div>
        <h1>{{ t('apps.title') }}</h1>
        <p class="finding" v-html="finding" />
      </div>
      <div class="actions">
        <q-btn unelevated color="primary" icon="o_send" :label="t('apps.new')" :to="{ path: '/claude', query: { command: 'apply' } }" />
        <q-btn unelevated class="btn-secondary" :label="t('apps.outcome')" :to="{ path: '/claude', query: { command: 'outcome' } }" />
      </div>
    </div>

    <section class="panel stages" :aria-label="t('apps.stages')">
      <div v-for="s in stages" :key="s.label" class="stage">
        <div class="stage-label">{{ s.label }}</div>
        <div class="stage-value num" :class="{ muted: !s.count }">{{ s.count }}</div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h2>{{ t('all') }} <span class="muted num">{{ rows.length }}</span></h2>
        <q-input v-model="filter" dense outlined debounce="200" :placeholder="t('apps.filter')" :aria-label="t('filter')" style="width: 260px">
          <template #prepend><q-icon name="o_search" size="18px" /></template>
        </q-input>
      </div>
      <q-table class="ledger" flat :rows="rows" :columns="columns" row-key="_i" :loading="loading"
               :filter="filter" :pagination="{ rowsPerPage: 25, sortBy: 'date', descending: true }"
               :rows-per-page-options="[25, 50, 0]" hide-bottom-space
               :no-data-label="t('apps.empty')">
        <template #body-cell-company="p">
          <q-td :props="p">
            <span class="cell-main">{{ p.row.company }}</span>
            <span class="cell-sub">{{ p.row.role }}</span>
            <span class="pill lt-sm q-mt-xs" :class="status(p.row.status)[1]">{{ status(p.row.status)[0] }}</span>
          </q-td>
        </template>
        <template #body-cell-status="p">
          <q-td :props="p"><span class="pill" :class="status(p.value)[1]">{{ status(p.value)[0] }}</span></q-td>
        </template>
        <template #body-cell-fit="p">
          <q-td :props="p">
            <span v-if="p.value !== null" class="fit"><span class="fit-bar"><i :style="{ width: p.value + '%' }" /></span><span class="fit-num">{{ p.value }}</span></span>
            <span v-else class="muted">—</span>
          </q-td>
        </template>
        <template #body-cell-deadline="p">
          <q-td :props="p">
            <span v-if="isFinal(p.row.status) || !p.value" class="muted">—</span>
            <span v-else :class="deadlineText(p.value).tone ? 'pill ' + deadlineText(p.value).tone : ''">{{ deadlineText(p.value).text }}</span>
          </q-td>
        </template>
        <template #body-cell-docs="p">
          <q-td :props="p">
            <a v-if="p.row.cv_file" class="row-link" :href="fileUrl(p.row.cv_file)" target="_blank">{{ t('apps.cv') }}</a>
            <a v-if="p.row.cover_letter_file" class="row-link" :href="fileUrl(p.row.cover_letter_file)" target="_blank">{{ t('apps.letter') }}</a>
            <a v-if="p.row.source" class="row-link" :href="p.row.source" target="_blank" rel="noopener noreferrer">{{ t('apps.posting') }}</a>
            <q-btn flat round dense size="sm" icon="o_more_horiz" :aria-label="t('apps.more', { name: p.row.company })">
              <q-menu auto-close>
                <q-list dense style="min-width: 200px">
                  <q-item clickable :to="{ path: '/claude', query: { command: 'interview', args: `${p.row.company} ${p.row.role}` } }">
                    <q-item-section>{{ t('apps.interview') }}</q-item-section>
                  </q-item>
                  <q-item clickable :to="{ path: '/claude', query: { command: 'outcome', args: `${p.row.company} ${p.row.role}` } }">
                    <q-item-section>{{ t('apps.outcome') }}</q-item-section>
                  </q-item>
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
import { get, fileUrl, status, statusKey, isFinal, daysUntil, deadlineText } from '../api.js'
import { t } from '../i18n.js'

const rows = ref([])
const loading = ref(true)
const filter = ref('')
const fit = r => (r.fit_rating === '' || r.fit_rating == null || isNaN(+r.fit_rating)) ? null : +r.fit_rating

const columns = computed(() => [
  { name: 'date', label: t('apps.colDate'), field: 'date', sortable: true, align: 'left', classes: 'num muted gt-xs', headerClasses: 'gt-xs', style: 'width: 120px' },
  { name: 'company', label: t('apps.colCompany'), field: r => `${r.company} ${r.role}`, sortable: true, align: 'left' },
  { name: 'status', label: t('apps.colStatus'), field: 'status', sortable: true, align: 'left', classes: 'gt-xs', headerClasses: 'gt-xs' },
  { name: 'fit', label: t('apps.colFit'), field: fit, sortable: true, align: 'right', sort: (a, b) => (a ?? -1) - (b ?? -1) },
  { name: 'deadline', label: t('apps.colDeadline'), field: 'deadline', sortable: true, align: 'right' },
  { name: 'docs', label: '', field: () => '', align: 'right' }
])

// Each row lands in exactly one stage, so the stages always sum to the table's total.
const stages = computed(() => {
  const by = k => rows.value.filter(r => k.includes(statusKey(r.status))).length
  const out = [
    { label: t('apps.stageDrafted'), count: by(['drafted']) },
    { label: t('apps.stageApplied'), count: by(['applied']) },
    { label: t('apps.stageProgress'), count: by(['interview', 'offer']) },
    { label: t('apps.stageClosed'), count: rows.value.filter(r => isFinal(r.status)).length }
  ]
  const other = rows.value.length - out.reduce((n, s) => n + s.count, 0)
  if (other) out.push({ label: t('apps.stageOther'), count: other })
  return out
})

const esc = s => String(s).replace(/[&<>"']/g, c => `&#${c.charCodeAt(0)};`)
const finding = computed(() => {
  if (loading.value) return t('loading')
  if (!rows.value.length) return t('apps.findingEmpty')
  const due = rows.value
    .filter(r => statusKey(r.status) === 'drafted' && daysUntil(r.deadline) !== null)
    .sort((a, b) => daysUntil(a.deadline) - daysUntil(b.deadline))[0]
  if (due) {
    const n = daysUntil(due.deadline)
    const when = n < 0 ? t('apps.whenOverdue', { n: -n }) : n === 0 ? t('apps.whenToday') : t('apps.whenIn', { n })
    return t('apps.findingDue', { company: `<b>${esc(due.company)}</b>`, when })
  }
  const open = rows.value.filter(r => !isFinal(r.status)).length
  return t('apps.findingOpen', { open: `<b>${open}</b>`, total: rows.value.length })
})

onMounted(async () => {
  try {
    rows.value = (await get('/api/tracker')).map((r, i) => ({ ...r, _i: i }))
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.stages { display: grid; grid-auto-flow: column; grid-auto-columns: 1fr; margin-bottom: var(--gap); }
.stage { padding: 16px 20px 18px; }
.stage + .stage { border-left: 1px solid var(--rule); }
.stage-label { color: var(--muted); font-size: 13px; }
.stage-value { font-size: 30px; line-height: 38px; font-weight: 400; margin-top: 4px; }
@media (max-width: 599px) {
  .stages { grid-auto-flow: row; grid-template-columns: repeat(2, 1fr); }
  .stage:nth-child(3) { border-left: 0; }
  .stage:nth-child(n + 3) { border-top: 1px solid var(--rule); }
}
</style>
