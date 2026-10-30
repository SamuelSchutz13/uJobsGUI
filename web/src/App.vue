<template>
  <q-layout view="lHh lpR fFf">
    <q-header class="mobile-bar lt-md">
      <q-toolbar>
        <q-btn flat round dense icon="o_menu" :aria-label="t('nav.openMenu')" @click="drawer = !drawer" />
        <q-toolbar-title class="mobile-title">{{ current ? t(current.label) : 'uJobs' }}</q-toolbar-title>
        <Settings />
      </q-toolbar>
    </q-header>

    <q-drawer v-model="drawer" show-if-above :width="240" :breakpoint="1023" class="sidebar">
      <div class="column full-height no-wrap">
        <div class="brand">
          <svg class="mark" viewBox="0 0 28 28" aria-hidden="true">
            <rect x="3" y="7" width="18" height="16" rx="2" fill="none" stroke="currentColor" stroke-width="2" />
            <path d="M3 9l9 7 9-7" fill="none" stroke="currentColor" stroke-width="2" />
            <circle cx="22" cy="7" r="4" fill="var(--fit)" />
          </svg>
          <div class="brand-text">
            <div class="brand-name">uJobs</div>
            <div class="brand-scope" :title="info?.root">{{ folder }}</div>
          </div>
          <Settings class="gt-sm" />
        </div>

        <nav class="nav" :aria-label="t('nav.main')">
          <div class="nav-group">{{ t('nav.groupSearch') }}</div>
          <router-link v-for="l in links.slice(0, 3)" :key="l.to" :to="l.to" class="nav-item">
            <q-icon :name="l.icon" size="16px" />{{ t(l.label) }}
          </router-link>
          <div class="nav-group">Claude Code</div>
          <router-link :to="links[3].to" class="nav-item">
            <q-icon :name="links[3].icon" size="16px" />{{ t(links[3].label) }}
          </router-link>
        </nav>

        <div class="sidebar-foot">
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
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { get } from './api.js'
import { t } from './i18n.js'
import Settings from './Settings.vue'

const drawer = ref(false)
const info = ref(null)
const route = useRoute()
const links = [
  { to: '/candidaturas', icon: 'o_assignment', label: 'nav.applications' },
  { to: '/vagas', icon: 'o_work_outline', label: 'nav.jobs' },
  { to: '/buscar', icon: 'o_search', label: 'nav.search' },
  { to: '/claude', icon: 'o_terminal', label: 'nav.commands' }
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
.nav-item { display: flex; align-items: center; gap: 10px; height: 32px; margin-bottom: 2px; padding: 0 10px; border-radius: 6px; color: var(--ink-2); text-decoration: none; font-size: 14px; }
.nav-item:hover { background: var(--hover); color: var(--text); }
.nav-item.router-link-active { background: var(--active); color: var(--text); font-weight: 500; }
.nav-item:focus-visible { outline: 2px solid var(--text); outline-offset: -2px; }
.sidebar-foot { padding: 16px 20px 20px; font-size: 12px; line-height: 1.5; color: var(--muted); }
.sidebar-foot a { color: var(--text); }
.mobile-bar { background: var(--ground); color: var(--text); border-bottom: 1px solid var(--border); }
.mobile-bar .q-toolbar { min-height: 48px; padding: 0 12px; }
.mobile-title { font-size: 15px; font-weight: 600; }
</style>
