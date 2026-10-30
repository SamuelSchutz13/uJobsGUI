<template>
  <q-layout view="lHh lpR fFf">
    <q-header class="mobile-bar lt-md">
      <q-toolbar>
        <q-btn flat round dense icon="o_menu" :aria-label="t('nav.openMenu')" @click="drawer = !drawer" />
        <q-toolbar-title class="mobile-title">{{ current ? t(current.label) : 'uJobs' }}</q-toolbar-title>
        <Settings />
      </q-toolbar>
    </q-header>

    <!-- Mini (icons only) is a desktop choice; on phones the drawer is an overlay and always shows labels. -->
    <q-drawer v-model="drawer" show-if-above :width="240" :mini="isMini" :mini-width="64" :breakpoint="1023"
              class="sidebar" :class="{ 'is-mini': isMini }">
      <div class="column full-height no-wrap">
        <div class="brand">
          <svg class="mark" viewBox="0 0 28 28" role="img" aria-label="uJobs">
            <rect x="3" y="7" width="18" height="16" rx="2" fill="none" stroke="currentColor" stroke-width="2" />
            <path d="M3 9l9 7 9-7" fill="none" stroke="currentColor" stroke-width="2" />
            <circle cx="22" cy="7" r="4" fill="var(--fit)" />
          </svg>
          <div v-if="!isMini" class="brand-text">
            <div class="brand-name">uJobs</div>
            <div class="brand-scope" :title="info?.root">{{ folder }}</div>
          </div>
          <Settings class="gt-sm" />
        </div>

        <nav class="nav" :aria-label="t('nav.main')">
          <template v-for="g in groups" :key="g.label">
            <div v-if="!isMini" class="nav-group">{{ t(g.label) }}</div>
            <div v-else class="nav-gap" />
            <router-link v-for="l in g.links" :key="l.to" :to="l.to" class="nav-item" :aria-label="isMini ? t(l.label) : undefined">
              <q-icon :name="l.icon" size="16px" />
              <span v-if="!isMini">{{ t(l.label) }}</span>
              <q-tooltip v-if="isMini" anchor="center right" self="center left" :offset="[10, 0]">{{ t(l.label) }}</q-tooltip>
            </router-link>
          </template>
        </nav>

        <button type="button" class="collapse gt-sm" :aria-label="isMini ? t('nav.expand') : t('nav.collapse')"
                :aria-expanded="!isMini" @click="mini = !mini">
          <q-icon :name="isMini ? 'o_keyboard_double_arrow_right' : 'o_keyboard_double_arrow_left'" size="18px" />
          <span v-if="!isMini">{{ t('nav.collapse') }}</span>
          <q-tooltip v-if="isMini" anchor="center right" self="center left" :offset="[10, 0]">{{ t('nav.expand') }}</q-tooltip>
        </button>

        <div v-if="!isMini" class="sidebar-foot">
          {{ t('footer.for') }}
          <a href="https://github.com/ai-job-search-dev/ai-job-search" target="_blank" rel="noopener">ai-job-search</a>
        </div>
      </div>
    </q-drawer>

    <q-page-container>
      <router-view />
    </q-page-container>
  </q-layout>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useQuasar } from 'quasar'
import { get } from './api.js'
import { t } from './i18n.js'
import Settings from './Settings.vue'

const $q = useQuasar()
const drawer = ref(false)
const info = ref(null)
const route = useRoute()

// A per-browser preference: storage may be blocked, so the full sidebar is the fallback.
const mini = ref((() => { try { return localStorage.getItem('ujobs-mini') === '1' } catch { return false } })())
watch(mini, v => { try { localStorage.setItem('ujobs-mini', v ? '1' : '0') } catch {} })
const isMini = computed(() => mini.value && $q.screen.gt.sm)

const links = [
  { to: '/candidaturas', icon: 'o_assignment', label: 'nav.applications' },
  { to: '/vagas', icon: 'o_work_outline', label: 'nav.jobs' },
  { to: '/buscar', icon: 'o_search', label: 'nav.search' },
  { to: '/claude', icon: 'o_terminal', label: 'nav.commands' }
]
const groups = [
  { label: 'nav.groupSearch', links: links.slice(0, 3) },
  { label: 'nav.groupClaude', links: links.slice(3) }
]
const current = computed(() => links.find(l => route.path.startsWith(l.to)))
const folder = computed(() => info.value?.root.split(/[\\/]/).filter(Boolean).pop() || '')

onMounted(async () => { info.value = await get('/api/info') })
</script>

<style>
.sidebar.q-drawer, .sidebar .q-drawer__content { background: var(--ground); }
.q-drawer--left.q-drawer--bordered, .sidebar { border-right: 1px solid var(--border); }
.brand { display: flex; gap: 10px; align-items: center; padding: 22px 12px 18px 20px; }
.brand-text { flex: 1; min-width: 0; }
.brand .q-btn { color: var(--muted); }
.brand .q-btn:hover { color: var(--text); }
.mark { width: 28px; height: 28px; flex: none; color: var(--text); }
.brand-name { font-weight: 600; font-size: 15px; line-height: 18px; }
.brand-scope { color: var(--muted); font-size: 12px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.nav { padding: 0 10px; flex: 1; }
.nav-group { color: var(--muted); font-size: 12px; padding: 22px 10px 8px; }
.nav-group:first-child { padding-top: 6px; }
.nav-gap { height: 12px; }
.nav-gap:first-child { height: 0; }
.nav-item { display: flex; align-items: center; gap: 10px; height: 32px; margin-bottom: 2px; padding: 0 10px; border-radius: 6px; color: var(--ink-2); text-decoration: none; font-size: 14px; white-space: nowrap; }
.nav-item:hover { background: var(--hover); color: var(--text); }
.nav-item.router-link-active { background: var(--active); color: var(--text); font-weight: 500; }
.nav-item:focus-visible { outline: 2px solid var(--text); outline-offset: -2px; }
.collapse { display: flex; align-items: center; gap: 10px; margin: 0 10px; height: 32px; padding: 0 10px; border: 0; border-radius: 6px; background: transparent; color: var(--muted); font: inherit; font-size: 13px; cursor: pointer; white-space: nowrap; }
.collapse:hover { background: var(--hover); color: var(--text); }
.collapse:focus-visible { outline: 2px solid var(--text); outline-offset: -2px; }
.sidebar-foot { padding: 12px 20px 20px; font-size: 12px; line-height: 1.5; color: var(--muted); }
.sidebar-foot a { color: var(--text); }

/* Icons only: one centred column, 64px wide. */
.is-mini .brand { flex-direction: column; gap: 12px; padding: 22px 0 18px; }
.is-mini .nav { padding: 0 12px; }
.is-mini .nav-item { justify-content: center; padding: 0; width: 40px; height: 40px; margin: 0 auto 2px; }
.is-mini .nav-item .q-icon { font-size: 18px !important; }
.is-mini .collapse { justify-content: center; width: 40px; height: 40px; margin: 0 auto 20px; padding: 0; }

.mobile-bar { background: var(--ground); color: var(--text); border-bottom: 1px solid var(--border); }
.mobile-bar .q-toolbar { min-height: 48px; padding: 0 12px; }
.mobile-title { font-size: 15px; font-weight: 600; }
</style>
