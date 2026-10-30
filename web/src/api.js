import { Notify } from 'quasar'

async function call (method, path, body) {
  const res = await fetch(path, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : {},
    body: body ? JSON.stringify(body) : undefined
  })
  const data = await res.json().catch(() => null)
  if (!res.ok) {
    const message = data?.error || `${res.status} ${res.statusText}`
    Notify.create({ type: 'negative', message })
    throw new Error(message)
  }
  return data
}

export const get = path => call('GET', path)
export const post = (path, body) => call('POST', path, body)

// .tex/.typ sources become their compiled PDF, which is what the user wants to open.
export const fileUrl = p => p ? '/files/' + p.replace(/\.(tex|typ)$/, '.pdf') : null

// Tracker status vocabulary from ai-job-search /outcome; legacy space spellings read the same.
const STATUS = {
  drafted: ['Rascunho', 'neutral'],
  applied: ['Enviada', 'neutral'],
  interview: ['Entrevista', 'positive'],
  offer: ['Oferta', 'positive'],
  hired: ['Contratado', 'positive'],
  rejected: ['Recusada', 'neutral'],
  no_response: ['Sem resposta', 'neutral'],
  offer_declined: ['Oferta recusada', 'neutral'],
  withdrawn: ['Desistência', 'neutral']
}
export const statusKey = s => (s || '').trim().toLowerCase().replace(/ /g, '_')
export const status = s => STATUS[statusKey(s)] || [s || '—', 'neutral']
export const isFinal = s => ['hired', 'rejected', 'no_response', 'offer_declined', 'withdrawn'].includes(statusKey(s))

// Whole days from today to an ISO date; null when there is no date.
export function daysUntil (iso) {
  if (!iso) return null
  const d = new Date(iso + 'T00:00:00')
  if (isNaN(d)) return null
  const today = new Date(); today.setHours(0, 0, 0, 0)
  return Math.round((d - today) / 86400000)
}

// Same 7-day threshold /rank and /outcome use for deadline urgency.
export function deadlineText (iso) {
  const n = daysUntil(iso)
  if (n === null) return { text: '—', tone: '' }
  if (n < 0) return { text: `venceu há ${-n} d`, tone: 'negative' }
  if (n === 0) return { text: 'hoje', tone: 'warning' }
  return { text: `em ${n} d`, tone: n <= 7 ? 'warning' : '' }
}
