<template>
  <q-btn flat round dense icon="o_settings" :aria-label="t('settings.title')">
    <q-tooltip>{{ t('settings.title') }}</q-tooltip>
    <q-menu anchor="bottom right" self="top right" :offset="[0, 6]" class="settings-menu">
      <div class="settings">
        <div class="settings-label" id="set-theme">{{ t('settings.theme') }}</div>
        <div class="choice" role="radiogroup" aria-labelledby="set-theme">
          <button v-for="o in THEMES" :key="o.id" type="button" role="radio" :aria-checked="theme === o.id"
                  :class="{ on: theme === o.id }" @click="theme = o.id">
            <q-icon :name="o.icon" size="16px" />{{ t('theme.' + o.id) }}
          </button>
        </div>

        <div class="settings-label" id="set-lang">{{ t('settings.language') }}</div>
        <div class="choice" role="radiogroup" aria-labelledby="set-lang">
          <button v-for="l in LANGS" :key="l.id" type="button" role="radio" :aria-checked="lang === l.id"
                  :class="{ on: lang === l.id }" @click="setLang(l.id)">
            <svg v-if="l.id === 'pt'" class="flag" viewBox="0 0 20 14" aria-hidden="true">
              <rect width="20" height="14" fill="#009c3b" />
              <path d="M10 1.6 18.2 7 10 12.4 1.8 7z" fill="#ffdf00" />
              <circle cx="10" cy="7" r="3.3" fill="#002776" />
            </svg>
            <svg v-else class="flag" viewBox="0 0 20 14" aria-hidden="true">
              <rect width="20" height="14" fill="#fff" />
              <path d="M0 0h20v1.08H0zm0 2.15h20v1.08H0zm0 2.16h20v1.07H0zm0 2.15h20v1.08H0zm0 2.15h20v1.08H0zm0 2.16h20v1.07H0zm0 2.15h20V14H0z" fill="#b22234" />
              <rect width="8" height="7.54" fill="#3c3b6e" />
            </svg>
            {{ l.label }}
          </button>
        </div>
      </div>
    </q-menu>
  </q-btn>
</template>

<script>
import { ref, watch } from 'vue'
import { Dark } from 'quasar'

const THEMES = [
  { id: 'light', icon: 'o_light_mode' },
  { id: 'dark', icon: 'o_dark_mode' },
  { id: 'auto', icon: 'o_contrast' }
]

// Module scope: the sidebar and the phone bar each show this menu, and both must agree.
// A per-browser preference: storage may be blocked, so the OS setting is the fallback.
const read = () => { try { return localStorage.getItem('ujobs-theme') } catch { return null } }
const theme = ref(THEMES.some(o => o.id === read()) ? read() : 'auto')
function apply (v) {
  Dark.set(v === 'auto' ? 'auto' : v === 'dark')
  try { localStorage.setItem('ujobs-theme', v) } catch {}
}
watch(theme, apply)
</script>

<script setup>
import { onMounted } from 'vue'
import { t, lang, setLang, LANGS } from './i18n.js'

// Applied once Quasar is installed; at import time its Dark plugin is not ready yet.
onMounted(() => apply(theme.value))
</script>

<style>
.settings-menu { background: var(--surface); color: var(--text); border: 1px solid var(--border); border-radius: 8px; box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12); }
.settings { padding: 12px; width: 280px; }
.settings-label { font-size: 12px; color: var(--muted); margin: 4px 2px 6px; }
.settings-label + .choice + .settings-label { margin-top: 14px; }
.choice { display: grid; grid-auto-flow: column; grid-auto-columns: 1fr; gap: 4px; padding: 3px; background: var(--ground); border-radius: 6px; }
.choice button { display: flex; align-items: center; justify-content: center; gap: 6px; height: 32px; border: 0; border-radius: 4px; background: transparent; color: var(--ink-2); font: inherit; font-size: 13px; cursor: pointer; }
.choice button:hover { color: var(--text); }
.choice button.on { background: var(--surface); color: var(--text); font-weight: 500; box-shadow: 0 0 0 1px var(--border); }
.choice button:focus-visible { outline: 2px solid var(--text); outline-offset: 1px; }
.flag { width: 18px; height: 13px; border-radius: 2px; flex: none; box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.12); }
</style>
