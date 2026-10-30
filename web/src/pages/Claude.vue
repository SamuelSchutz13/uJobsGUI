<template>
  <q-page class="inbox" :class="{ 'has-open': open }">
    <aside class="runs" aria-label="Execuções">
      <div class="runs-head">
        <h1>Comandos</h1>
        <q-btn unelevated color="primary" icon="o_add" label="Novo" :to="{ path: '/claude', query: { novo: 1 } }" />
      </div>
      <div v-if="!runs.length" class="runs-empty muted">Nenhuma execução nesta sessão do servidor.</div>
      <router-link v-for="r in runs" :key="r.id" :to="`/claude/${r.id}`" class="run" :class="{ active: run?.id === r.id }">
        <div class="run-line">
          <span class="run-cmd">/{{ r.command }}</span>
          <span class="run-time num">{{ ago(r.started) }}</span>
        </div>
        <div class="run-sub">
          <span class="run-arg">{{ firstArg(r) || 'sem argumentos' }}</span>
          <span v-if="r.running" class="pill neutral">Rodando</span>
          <span v-else-if="lastIsError(r)" class="pill negative">Erro</span>
          <span v-else-if="r.turns.length > 1" class="pill neutral">{{ (r.turns.length / 2) | 0 }} {{ r.turns.length > 3 ? 'respostas' : 'resposta' }}</span>
        </div>
      </router-link>
    </aside>

    <section class="open">
      <q-form v-if="!run" class="new-run" @submit="start">
        <div class="open-head">
          <q-btn flat round dense icon="o_arrow_back" class="lt-md" aria-label="Voltar para a lista" @click="open = false" />
          <h2>Rodar um comando do ai-job-search</h2>
        </div>
        <q-select v-model="command" :options="commands" label="Comando" outlined dense :option-label="c => '/' + c" />
        <p v-if="HINTS[command]" class="hint">{{ HINTS[command] }}</p>
        <q-input v-model="args" label="Argumentos" outlined autogrow />
        <div class="row items-center q-gutter-md">
          <q-btn type="submit" unelevated color="primary" icon="o_play_arrow" label="Rodar" :disable="!command" :loading="busy" />
          <span class="muted">Roda <code>claude -p "/{{ command || 'comando' }} …"</code> na pasta do ai-job-search, com as permissões dele.</span>
        </div>
      </q-form>

      <section v-if="!run" class="reference" aria-labelledby="ref-title">
        <h2 id="ref-title">O que cada comando faz</h2>
        <button v-for="c in reference" :key="c.id" type="button" class="ref-row" :class="{ active: command === c.id }" @click="command = c.id">
          <span class="ref-cmd">/{{ c.id }}</span>
          <span class="ref-desc">{{ c.desc }}</span>
        </button>
      </section>

      <template v-else>
        <div class="open-head">
          <q-btn flat round dense icon="o_arrow_back" class="lt-md" aria-label="Voltar para a lista" to="/claude" />
          <h2>/{{ run.command }}</h2>
          <span class="muted num">{{ new Date(run.started).toLocaleString('pt-BR') }}</span>
        </div>
        <div class="thread">
          <div v-for="(t, i) in run.turns" :key="i" class="turn" :class="t.role">
            <div class="who">{{ t.role === 'user' ? 'Você' : 'Claude' }}</div>
            <div class="text">{{ t.text }}</div>
          </div>
          <div v-if="run.running" class="turn claude working">
            <div class="who">Claude</div>
            <div class="text muted"><q-spinner-dots size="20px" /> Trabalhando. O /apply costuma levar alguns minutos.</div>
          </div>
        </div>
        <q-form v-if="!run.running" class="composer" @submit="reply">
          <q-input v-model="message" outlined autogrow dense placeholder="Responder ao Claude" aria-label="Responder ao Claude" class="col" @keydown.enter.exact.prevent="message.trim() && reply()" />
          <q-btn type="submit" unelevated color="primary" label="Enviar" :disable="!message.trim()" :loading="busy" />
        </q-form>
      </template>
    </section>
  </q-page>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useQuasar } from 'quasar'
import { get, post } from '../api.js'

const $q = useQuasar()
const route = useRoute()
const router = useRouter()
const commands = ref([])
const runs = ref([])
const run = ref(null)
const command = ref(route.query.command || null)
const args = ref(route.query.args || '')
const message = ref('')
const busy = ref(false)
const open = ref(!!(route.query.command || route.query.novo))

const HINTS = {
  apply: 'URL da vaga, ou o texto completo da vaga colado.',
  scrape: 'Opcional: um foco para a busca, como "remoto" ou "dados".',
  rank: 'Opcional: --all para pontuar tudo, ou um número para limitar.',
  interview: 'Empresa e vaga de uma candidatura registrada.',
  outcome: 'Empresa e vaga, ou "followup" para ver as que estão paradas.',
  setup: 'Sem argumentos. O Claude vai perguntar o que precisar.',
  reset: 'Apaga dados do seu perfil. Você vai confirmar antes.'
}

// Short descriptions from the ai-job-search README; commands not listed here still run.
const DESC = {
  setup: 'Monta seu perfil a partir dos documentos, de um CV colado ou de uma entrevista.',
  scrape: 'Busca nos portais as vagas que combinam com seu perfil.',
  rank: 'Pontua em lote as vagas encontradas e devolve uma lista curta.',
  apply: 'Avalia o fit e escreve, revisa e compila CV e carta sob medida.',
  interview: 'Prepara a entrevista de uma candidatura e faz uma simulação.',
  outcome: 'Registra resultados, arquiva o que foi enviado e sugere follow-ups.',
  upskill: 'Mapeia as lacunas de competência e monta um plano de estudo.',
  expand: 'Acrescenta competências achadas nos seus perfis públicos.',
  'html-report': 'Gera um painel offline das suas candidaturas.',
  'gmail-sync': 'Propõe atualizações de situação a partir do seu Gmail.',
  'notion-sync': 'Publica uma visão somente leitura no Notion.',
  'add-template': 'Registra seu próprio modelo de CV ou carta.',
  'add-portal': 'Gera um buscador para um portal de vagas do seu país.',
  reset: 'Apaga dados do perfil ou documentos.'
}
const reference = computed(() => commands.value.filter(c => DESC[c]).map(id => ({ id, desc: DESC[id] })))

const firstArg = r => r.turns[0]?.text.replace(/^\/\S+\s*/, '')
const lastIsError = r => r.turns.at(-1)?.text.includes('[erro]')
function ago (iso) {
  const m = Math.round((Date.now() - new Date(iso)) / 60000)
  if (m < 1) return 'agora'
  if (m < 60) return `${m} min`
  const h = Math.round(m / 60)
  return h < 24 ? `${h} h` : new Date(iso).toLocaleDateString('pt-BR')
}

async function refresh () {
  runs.value = await get('/api/runs')
  run.value = route.params.id ? await get(`/api/runs/${route.params.id}`) : null
  if (run.value) open.value = true
}

// shortcut: polling every 2s; move to SSE if many runs are open at once.
const timer = setInterval(() => { if (runs.value.some(r => r.running)) refresh() }, 2000)
onUnmounted(() => clearInterval(timer))
onMounted(async () => {
  commands.value = await get('/api/commands')
  await refresh()
})
watch(() => route.params.id, refresh)
watch(() => route.fullPath, () => { if (route.path === '/claude') open.value = !!(route.query.command || route.query.novo) })

function confirmReset () {
  return new Promise(resolve => {
    $q.dialog({ title: '/reset', message: 'Isso apaga dados do seu perfil no ai-job-search. Continuar?', cancel: true })
      .onOk(() => resolve(true)).onCancel(() => resolve(false))
  })
}

async function start () {
  if (command.value === 'reset' && !(await confirmReset())) return
  busy.value = true
  try {
    const r = await post('/api/runs', { command: command.value, args: args.value })
    args.value = ''
    await router.push(`/claude/${r.id}`)
  } finally {
    busy.value = false
  }
}

async function reply () {
  busy.value = true
  try {
    run.value = await post(`/api/runs/${run.value.id}/reply`, { message: message.value })
    message.value = ''
    runs.value = await get('/api/runs')
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.inbox { display: grid; grid-template-columns: 340px 1fr; min-height: 100vh; }
.runs { border-right: 1px solid var(--border); padding: 32px 12px 24px; }
.runs-head { display: flex; align-items: center; justify-content: space-between; padding: 0 8px 16px; }
.runs-head h1 { font-size: 24px; line-height: 32px; font-weight: 600; margin: 0; }
.runs-empty { padding: 8px; font-size: 13px; }
.run { display: block; padding: 12px; border-radius: 6px; text-decoration: none; color: var(--text); }
.run + .run { margin-top: 2px; }
.run:hover { background: #eaedf1; }
.run.active { background: var(--active); }
.run:focus-visible { outline: 2px solid var(--text); outline-offset: -2px; }
.run-line, .run-sub { display: flex; justify-content: space-between; gap: 8px; align-items: center; }
.run-cmd { font-weight: 600; }
.run-time { color: var(--muted); font-size: 12px; }
.run-sub { margin-top: 4px; }
.run-arg { font-size: 13px; color: #3d3d3d; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; min-width: 0; }

.open { padding: 32px 40px; display: flex; flex-direction: column; min-width: 0; }
.open-head { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.open-head h2 { font-size: 18px; line-height: 28px; font-weight: 600; margin: 0; flex: 1; }
.new-run { display: grid; gap: 16px; max-width: 720px; background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-card); padding: 24px; }
.new-run .open-head { margin: 0; }
.reference { max-width: 720px; margin-top: 24px; }
.reference h2 { font-size: 16px; font-weight: 600; margin: 0 0 8px; line-height: 24px; }
.ref-row { display: grid; grid-template-columns: 140px 1fr; gap: 12px; width: 100%; text-align: left; padding: 10px 12px; border: 0; border-radius: 6px; background: transparent; font: inherit; color: var(--text); cursor: pointer; }
.ref-row + .ref-row { border-top: 1px solid var(--rule); border-radius: 0; }
.ref-row:hover, .ref-row.active { background: #eaedf1; border-radius: 6px; }
.ref-row:focus-visible { outline: 2px solid var(--text); outline-offset: -2px; }
.ref-cmd { font-weight: 500; }
.ref-desc { color: #3d3d3d; }
.hint { margin: -8px 0 0; color: var(--muted); font-size: 13px; }

.thread { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-card); flex: 1; }
.turn { padding: 16px 20px; }
.turn + .turn { border-top: 1px solid var(--rule); }
.turn.user { background: #f7f8fa; }
.who { font-size: 12px; color: var(--muted); margin-bottom: 4px; }
.text { white-space: pre-wrap; word-break: break-word; font-size: 14px; line-height: 1.55; max-width: 80ch; }
.turn.user .text { font-weight: 500; }
.working .text { display: flex; align-items: center; gap: 8px; }
.composer { display: flex; gap: var(--gap); align-items: flex-end; margin-top: var(--gap); }
.composer .q-btn { height: 40px; padding: 0 20px; }

@media (max-width: 1023px) {
  .inbox { grid-template-columns: 1fr; min-height: auto; }
  .runs { border-right: 0; padding: 20px 8px; }
  .open { padding: 20px 16px; }
  .inbox.has-open .runs, .inbox:not(.has-open) .open { display: none; }
}
</style>
