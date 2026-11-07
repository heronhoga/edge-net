<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import {
  Activity,
  ArrowRight,
  ArrowUpRight,
  Box,
  Check,
  CheckCircle2,
  ChevronRight,
  CircleHelp,
  Code2,
  Copy,
  Cpu,
  Layers3,
  LayoutDashboard,
  LoaderCircle,
  LogOut,
  Network,
  Plus,
  Radio,
  RefreshCw,
  Search,
  ShieldCheck,
  Terminal,
  Upload,
  Wifi,
  WifiOff,
  X,
} from 'lucide-vue-next'
import Modal from './components/Modal.vue'
import type { Deployment, Device, Firmware, Snapshot } from './types'

type Page = 'overview' | 'devices' | 'firmware' | 'deployments' | 'connect'
type Dialog = '' | 'enroll' | 'upload' | 'deploy' | 'credentials' | 'device'
const token = ref(sessionStorage.getItem('edgenet-token') || '')
const loginToken = ref('')
const authenticated = ref(false)
const loading = ref(false)
const busy = ref(false)
const error = ref('')
const modalError = ref('')
const notice = ref('')
const page = ref<Page>('overview')
const dialog = ref<Dialog>('')
const state = ref<Snapshot>({
  devices: [],
  firmware: [],
  deployments: [],
  events: [],
  serverTime: '',
})
const query = ref('')
const filter = ref('all')
const selectedId = ref('')
const credentials = ref({ id: '', token: '' })
const form = ref({ name: '', kind: 'esp32', target: 'esp32-devkit-v1', location: '' })
const uploadForm = ref({ version: '', target: '' })
const uploadFile = ref<File | null>(null)
const firmwareId = ref('')
const deviceIds = ref<string[]>([])
function clearFilters() {
  query.value = ''
  filter.value = 'all'
}
let timer: ReturnType<typeof setInterval> | undefined
let noticeTimer: ReturnType<typeof setTimeout> | undefined
const nav = [
  { id: 'overview' as Page, label: 'Overview', icon: LayoutDashboard },
  { id: 'devices' as Page, label: 'Devices', icon: Cpu },
  { id: 'firmware' as Page, label: 'Firmware', icon: Layers3 },
  { id: 'deployments' as Page, label: 'Deployments', icon: ArrowUpRight },
]
const kinds: Record<string, string> = {
  esp32: 'ESP32',
  arduino: 'Arduino',
}
const targets: Record<string, string> = {
  esp32: 'esp32-devkit-v1',
  arduino: 'arduino-uno-r4-wifi',
}
const selected = computed(() => state.value.devices.find((d) => d.id === selectedId.value))
const online = computed(() => state.value.devices.filter((d) => d.status === 'online').length)
const offline = computed(() => state.value.devices.filter((d) => d.status === 'offline').length)
const pending = computed(() => state.value.devices.filter((d) => d.status === 'pending').length)
const activeJobs = computed(() => state.value.deployments.filter((j) => isActive(j.status)))
const filteredDevices = computed(() =>
  state.value.devices.filter(
    (d) =>
      (filter.value === 'all' || d.status === filter.value) &&
      `${d.name} ${d.target} ${d.location} ${kinds[d.kind]}`
        .toLowerCase()
        .includes(query.value.toLowerCase()),
  ),
)
const chosenFirmware = computed(() => state.value.firmware.find((f) => f.id === firmwareId.value))
const compatible = computed(() =>
  state.value.devices.filter((d) => d.target === chosenFirmware.value?.target),
)
const pageTitle = computed(() =>
  page.value === 'connect' ? 'Connect your devices' : nav.find((n) => n.id === page.value)?.label,
)
const connectionHeaders = computed(
  () =>
    `Authorization: Bearer ${credentials.value.token}\nX-Device-ID: ${credentials.value.id}\nContent-Type: application/json`,
)

function isActive(s: string) {
  return ['queued', 'downloading', 'installing'].includes(s)
}
function kindIcon(kind: string) {
  return kind === 'arduino' ? Code2 : Cpu
}
function relative(value: string | null) {
  if (!value) return 'Never connected'
  const seconds = Math.max(0, Math.floor((Date.now() - new Date(value).getTime()) / 1000))
  return seconds < 10
    ? 'Just now'
    : seconds < 60
      ? `${seconds}s ago`
      : seconds < 3600
        ? `${Math.floor(seconds / 60)}m ago`
        : seconds < 86400
          ? `${Math.floor(seconds / 3600)}h ago`
          : `${Math.floor(seconds / 86400)}d ago`
}
function size(bytes: number) {
  return bytes < 1048576 ? `${(bytes / 1024).toFixed(1)} KB` : `${(bytes / 1048576).toFixed(1)} MB`
}
function metric(value?: number) {
  return value == null ? '—' : `${Math.round(value)}%`
}
function uptime(seconds: number) {
  return seconds >= 86400
    ? `${Math.floor(seconds / 86400)}d ${Math.floor((seconds % 86400) / 3600)}h`
    : `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`
}
function toast(message: string) {
  notice.value = message
  clearTimeout(noticeTimer)
  noticeTimer = setTimeout(() => (notice.value = ''), 4500)
}
function logout() {
  authenticated.value = false
  token.value = ''
  sessionStorage.removeItem('edgenet-token')
  dialog.value = ''
  state.value = { devices: [], firmware: [], deployments: [], events: [], serverTime: '' }
}
async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Authorization', `Bearer ${token.value}`)
  if (init.body && !(init.body instanceof FormData)) headers.set('Content-Type', 'application/json')
  const res = await fetch(`/api/admin${path}`, {
    ...init,
    headers,
    signal: AbortSignal.timeout(60000),
  })
  if (!res.ok) {
    if (res.status === 401) logout()
    const body = await res.json().catch(() => ({ error: `Request failed (${res.status})` }))
    throw new Error(body.error || 'Request failed')
  }
  return res.status === 204 ? (undefined as T) : res.json()
}
async function refresh() {
  if (loading.value || !token.value) return
  loading.value = true
  const requestToken = token.value
  try {
    const snapshot = await api<Snapshot>('/state')
    if (token.value !== requestToken) return
    state.value = snapshot
    authenticated.value = true
    error.value = ''
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Cannot reach the API'
  } finally {
    loading.value = false
  }
}
async function login() {
  token.value = loginToken.value.trim()
  await refresh()
  if (authenticated.value) {
    sessionStorage.setItem('edgenet-token', token.value)
    loginToken.value = ''
  }
}
function open(kind: Dialog) {
  modalError.value = ''
  if (kind === 'enroll')
    form.value = { name: '', kind: 'esp32', target: targets.esp32!, location: '' }
  if (kind === 'upload') {
    uploadForm.value = { version: '', target: '' }
    uploadFile.value = null
  }
  dialog.value = kind
}
function close() {
  if (!busy.value) {
    dialog.value = ''
    credentials.value = { id: '', token: '' }
  }
}
function openDevice(d: Device) {
  selectedId.value = d.id
  open('device')
}
function openDeploy(f?: Firmware, d?: Device) {
  firmwareId.value =
    f?.id || state.value.firmware.find((fw) => !d || fw.target === d.target)?.id || ''
  deviceIds.value = d ? [d.id] : []
  open('deploy')
}
async function mutate(action: () => Promise<void>) {
  busy.value = true
  modalError.value = ''
  try {
    await action()
    await refresh()
  } catch (e) {
    const message = e instanceof Error ? e.message : 'Request failed'
    if (dialog.value) modalError.value = message
    else error.value = message
  } finally {
    busy.value = false
  }
}
async function enroll() {
  await mutate(async () => {
    const result = await api<{ device: Device; token: string }>('/devices', {
      method: 'POST',
      body: JSON.stringify(form.value),
    })
    credentials.value = { id: result.device.id, token: result.token }
    dialog.value = 'credentials'
    toast('Device enrolled. Connect it using its new token.')
  })
}
async function upload() {
  if (!uploadFile.value) {
    modalError.value = 'Choose an artifact to upload.'
    return
  }
  if (uploadFile.value.size > 64 * 1024 * 1024) {
    modalError.value = 'The artifact must be 64 MiB or smaller.'
    return
  }
  await mutate(async () => {
    const body = new FormData()
    body.set('file', uploadFile.value!)
    body.set('version', uploadForm.value.version)
    body.set('target', uploadForm.value.target)
    await api('/firmware', { method: 'POST', body })
    dialog.value = ''
    toast('Firmware uploaded and SHA-256 checksum generated.')
  })
}
async function deploy() {
  await mutate(async () => {
    await api('/deployments', {
      method: 'POST',
      body: JSON.stringify({ firmwareId: firmwareId.value, deviceIds: deviceIds.value }),
    })
    dialog.value = ''
    page.value = 'deployments'
    toast('Deployment queued. Devices will pick it up on their next check-in.')
  })
}
async function cancel(j: Deployment) {
  await mutate(async () => {
    await api(`/deployments/${j.id}/cancel`, { method: 'POST' })
    toast('Deployment cancelled.')
  })
}
async function rotate(d: Device) {
  if (!confirm(`Replace the access token for ${d.name}? Its current token will stop working.`))
    return
  await mutate(async () => {
    const result = await api<{ token: string }>(`/devices/${d.id}/rotate-token`, { method: 'POST' })
    credentials.value = { id: d.id, token: result.token }
    dialog.value = 'credentials'
  })
}
async function remove(d: Device) {
  if (!confirm(`Remove ${d.name} and revoke its access?`)) return
  await mutate(async () => {
    await api(`/devices/${d.id}`, { method: 'DELETE' })
    dialog.value = ''
    toast('Device removed.')
  })
}
async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    toast('Copied to clipboard.')
  } catch {
    modalError.value = 'Clipboard unavailable. Select and copy the text manually.'
  }
}
function fileChanged(event: Event) {
  uploadFile.value = (event.target as HTMLInputElement).files?.[0] || null
}
function cpuPoints(d: Device) {
  return d.history
    .flatMap((h, i) =>
      h.cpu == null ? [] : [`${(i / Math.max(d.history.length - 1, 1)) * 600},${110 - h.cpu}`],
    )
    .join(' ')
}
onMounted(() => {
  if (token.value) void refresh()
  timer = setInterval(() => {
    if (authenticated.value) void refresh()
  }, 5000)
})
onUnmounted(() => {
  clearInterval(timer)
  clearTimeout(noticeTimer)
})
</script>

<template>
  <div v-if="!authenticated" class="login-shell">
    <div class="login-art">
      <div class="brand">
        <span class="brand-symbol"><Network :size="25" /></span>edgenet<span class="brand-dot"
          >.</span
        >
      </div>
      <div class="login-story">
        <h1>Your edge.<br />All connected.</h1>
        <p>One place to monitor your hardware, ship updates, and keep your entire fleet in sync.</p>
        <div class="hardware-row">
          <Cpu /><span>ESP32</span><span class="dot-separator">/</span><Code2 /><span
            >Arduino</span
          >
        </div>
      </div>
      <div class="network-art" aria-hidden="true">
        <div class="orbit orbit-one"></div>
        <div class="orbit orbit-two"></div>
        <div class="network-center"><Network :size="48" /></div>
        <div class="network-node node-one"><Cpu /></div>
        <div class="network-node node-two"><Code2 /></div>
        <div class="network-node node-three"><Cpu /></div>
      </div>
    </div>
    <main class="login-panel">
      <form class="login-form" @submit.prevent="login">
        <h2>Welcome to Edgenet</h2>
        <p>Connect to your device management portal.</p>
        <label for="admin-token">Admin access token</label
        ><input
          id="admin-token"
          v-model="loginToken"
          type="password"
          required
          minlength="16"
          autocomplete="current-password"
          placeholder="Enter your ADMIN_TOKEN"
        />
        <p v-if="error" class="error-box" role="alert">{{ error }}</p>
        <button class="button primary login-submit" :disabled="loading || !loginToken.trim()">
          <LoaderCircle v-if="loading" class="spin" :size="17" />Open workspace<ArrowRight
            v-if="!loading"
            :size="17"
          />
        </button>
        <div class="login-hint">
          <Terminal :size="18" />
          <p>
            Use the <code>ADMIN_TOKEN</code> configured on your Go server. Your session stays in
            this browser tab.
          </p>
        </div>
      </form>
    </main>
  </div>

  <div v-else class="app-shell">
    <aside class="sidebar">
      <a class="brand" href="#" @click.prevent="page = 'overview'"
        ><span class="brand-symbol"><Network :size="24" /></span
        ><span class="brand-word">edgenet<span class="brand-dot">.</span></span></a
      >
      <div class="workspace">
        <div class="workspace-avatar">E</div>
        <div><strong>My workspace</strong><span>Edge infrastructure</span></div>
        <ChevronRight :size="15" />
      </div>
      <span class="nav-label">WORKSPACE</span>
      <nav aria-label="Main navigation">
        <button
          v-for="item in nav"
          :key="item.id"
          :class="['nav-item', { active: page === item.id }]"
          :title="item.label"
          @click="page = item.id"
        >
          <component :is="item.icon" :size="19" /><span>{{ item.label }}</span
          ><span v-if="item.id === 'devices'" class="nav-count">{{ state.devices.length }}</span>
        </button>
      </nav>
      <div class="sidebar-bottom">
        <div class="connect-card">
          <span class="connect-icon"><Radio :size="20" /></span
          ><strong>Bring your edge online</strong>
          <p>From your first board to your next hundred devices.</p>
          <button @click="page = 'connect'">Connection guide <ArrowUpRight :size="15" /></button>
        </div>
        <button
          :class="['nav-item', { active: page === 'connect' }]"
          title="Connection guide"
          @click="page = 'connect'"
        >
          <CircleHelp :size="19" /><span>Connection guide</span>
        </button>
        <div class="profile">
          <span class="profile-avatar">AD</span>
          <div><strong>Administrator</strong><span>Workspace owner</span></div>
          <button class="icon-button" aria-label="Sign out" @click="logout">
            <LogOut :size="17" />
          </button>
        </div>
      </div>
    </aside>
    <div class="main-shell">
      <header class="topbar">
        <div class="breadcrumb">
          Workspace <ChevronRight :size="14" /><strong>{{ pageTitle }}</strong>
        </div>
        <div class="topbar-right">
          <span class="live-indicator" :class="{ disconnected: error }"
            ><i></i>{{ error ? 'Connection issue' : 'Live updates' }}</span
          ><button
            class="icon-button"
            :disabled="loading"
            aria-label="Refresh dashboard"
            @click="refresh"
          >
            <RefreshCw :size="17" :class="{ spin: loading }" /></button
          ><span class="topbar-avatar">AD</span>
        </div>
      </header>
      <main class="content">
        <div v-if="error" class="error-box global-error" role="alert">
          {{ error }} <button class="text-button" @click="refresh">Retry</button>
        </div>
        <section class="page-heading">
          <div>
            <div class="eyebrow">YOUR EDGE, AT A GLANCE</div>
            <h1>{{ page === 'overview' ? 'Fleet overview' : pageTitle }}</h1>
            <p>
              {{
                page === 'overview'
                  ? 'A little visibility goes a long way. Here’s how your devices are doing.'
                  : page === 'devices'
                    ? 'Every Arduino and ESP32 board. All in one place.'
                    : page === 'firmware'
                      ? 'Versioned artifacts, ready for their next deployment.'
                      : page === 'deployments'
                        ? 'Follow every update from queued to completed.'
                        : 'Connect your Arduino and ESP32 boards with a simple HTTP protocol.'
              }}
            </p>
          </div>
          <button
            v-if="page === 'overview' || page === 'devices'"
            class="button primary"
            @click="open('enroll')"
          >
            <Plus :size="17" />Add device</button
          ><button v-else-if="page === 'firmware'" class="button primary" @click="open('upload')">
            <Upload :size="17" />Upload firmware</button
          ><button v-else-if="page === 'deployments'" class="button primary" @click="openDeploy()">
            <Plus :size="17" />New deployment
          </button>
        </section>

        <template v-if="page === 'overview' || page === 'devices'">
          <section class="stats-grid" aria-label="Fleet statistics">
            <article class="stat-card">
              <div class="stat-top">
                <span>Total devices</span><span class="stat-icon"><Cpu :size="18" /></span>
              </div>
              <div class="stat-value">
                {{ state.devices.length }}<span class="stat-caption">in your fleet</span>
              </div>
              <div class="stat-bottom">
                <span class="tiny-dot dark"></span
                >{{ new Set(state.devices.map((d) => d.kind)).size }} hardware types
              </div>
            </article>
            <article class="stat-card">
              <div class="stat-top">
                <span>Online now</span><span class="stat-icon green"><Wifi :size="18" /></span>
              </div>
              <div class="stat-value">
                {{ online
                }}<span class="stat-pill"
                  >{{
                    state.devices.length ? Math.round((online / state.devices.length) * 100) : 0
                  }}%</span
                >
              </div>
              <div class="stat-bottom">
                <span class="tiny-dot"></span>Heartbeat within 90 seconds
              </div>
            </article>
            <article class="stat-card">
              <div class="stat-top">
                <span>Offline devices</span
                ><span class="stat-icon amber"><WifiOff :size="18" /></span>
              </div>
              <div class="stat-value">
                {{ offline
                }}<span class="stat-caption">{{
                  offline ? 'need a check-in' : 'looking good'
                }}</span>
              </div>
              <div class="stat-bottom">
                <span class="tiny-dot amber"></span>{{ pending }} awaiting first connection
              </div>
            </article>
            <article class="stat-card">
              <div class="stat-top">
                <span>Active deployments</span
                ><span class="stat-icon blue"><ArrowUpRight :size="18" /></span>
              </div>
              <div class="stat-value">
                {{ activeJobs.length }}<span class="stat-caption">in progress</span>
              </div>
              <div class="stat-bottom">
                <span class="tiny-dot blue"></span>{{ state.firmware.length }} firmware artifacts
                available
              </div>
            </article>
          </section>
          <section class="panel device-panel">
            <div class="panel-title">
              <div>
                <h2>
                  Your devices <span class="count-badge">{{ state.devices.length }}</span>
                </h2>
                <p>A live look at your connected infrastructure.</p>
              </div>
              <button class="button secondary small" @click="openDeploy()">
                <ArrowUpRight :size="16" />Deploy update
              </button>
            </div>
            <div class="table-toolbar">
              <div class="tabs" aria-label="Filter device status">
                <button
                  v-for="s in ['all', 'online', 'offline', 'pending']"
                  :key="s"
                  :class="{ selected: filter === s }"
                  @click="filter = s"
                >
                  {{ s === 'all' ? 'All devices' : s === 'pending' ? 'Pending' : s
                  }}<span v-if="s === 'all'">{{ state.devices.length }}</span>
                </button>
              </div>
              <label class="search"
                ><Search :size="16" /><input
                  v-model="query"
                  aria-label="Search devices"
                  placeholder="Search devices…"
              /></label>
            </div>
            <div v-if="filteredDevices.length" class="table-scroll">
              <table>
                <thead>
                  <tr>
                    <th>Device</th>
                    <th>Status</th>
                    <th>Firmware</th>
                    <th>CPU / Memory</th>
                    <th>Last seen</th>
                    <th><span class="sr-only">Details</span></th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="d in filteredDevices" :key="d.id">
                    <td>
                      <button class="device-name" @click="openDevice(d)">
                        <span :class="['device-icon', d.kind]"
                          ><component :is="kindIcon(d.kind)" :size="21" /></span
                        ><span
                          ><strong>{{ d.name }}</strong
                          ><small
                            >{{ kinds[d.kind]
                            }}<span v-if="d.location"> · {{ d.location }}</span></small
                          ></span
                        >
                      </button>
                    </td>
                    <td>
                      <span :class="['badge', d.status]"><i></i>{{ d.status }}</span>
                    </td>
                    <td>
                      <span class="version">{{ d.version }}</span
                      ><small class="target-label">{{ d.target }}</small>
                    </td>
                    <td>
                      <div class="resource-values">
                        <span>{{ metric(d.metrics.cpu) }}</span
                        ><span class="resource-divider">/</span
                        ><span>{{ metric(d.metrics.memory) }}</span>
                      </div>
                      <div class="resource-meter">
                        <i :style="{ width: `${d.metrics.cpu || 0}%` }"></i>
                      </div>
                    </td>
                    <td class="muted nowrap">{{ relative(d.lastSeen) }}</td>
                    <td>
                      <button
                        class="icon-button"
                        :aria-label="`View ${d.name}`"
                        @click="openDevice(d)"
                      >
                        <ChevronRight :size="18" />
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-else class="empty-state">
              <div class="empty-icon"><Cpu :size="30" /></div>
              <h3>{{ state.devices.length ? 'No matching devices' : 'Your fleet starts here' }}</h3>
              <p>
                {{
                  state.devices.length
                    ? 'Try another search or status filter.'
                    : 'Connect an ESP32 or Arduino and give your edge a home.'
                }}
              </p>
              <button v-if="!state.devices.length" class="button primary" @click="open('enroll')">
                <Plus :size="16" />Add your first device</button
              ><button
                v-else
                class="button secondary"
                @click="clearFilters"
              >
                Clear filters
              </button>
            </div>
            <div class="table-footer">
              <span>Showing {{ filteredDevices.length }} of {{ state.devices.length }} devices</span
              ><span><RefreshCw :size="12" /> Auto-refreshes every 5s</span>
            </div>
          </section>
          <div v-if="page === 'overview'" class="bottom-grid">
            <section class="panel activity-panel">
              <div class="panel-title">
                <h2>Recent activity</h2>
                <span class="subtle-label">LATEST EVENTS</span>
              </div>
              <div v-if="!state.events.length" class="quiet-empty">
                <Activity :size="23" />
                <p>All quiet for now. Your fleet’s activity will appear here.</p>
              </div>
              <div v-for="(event, index) in state.events.slice(0, 5)" :key="event.id" class="event">
                <span class="event-icon" :class="{ first: index === 0 }"
                  ><Activity :size="15"
                /></span>
                <div>
                  <p>{{ event.message }}</p>
                  <small>{{ relative(event.at) }}</small>
                </div>
              </div>
            </section>
            <section class="fleet-note">
              <span class="mini-label"><Network :size="16" /> BUILT FOR MICROCONTROLLERS</span>
              <h2>Different devices.<br />One connected fleet.</h2>
              <p>
                Arduino and ESP32. Monitor your boards together and deploy firmware built for each target.
              </p>
              <div class="supported-icons">
                <span><Cpu :size="19" />ESP32</span><span><Code2 :size="19" />Arduino</span>
              </div>
              <button @click="page = 'connect'">
                Explore the connection guide <ArrowRight :size="17" />
              </button>
            </section>
          </div>
        </template>

        <template v-else-if="page === 'firmware'">
          <div class="info-strip">
            <ShieldCheck :size="19" /><span
              >Artifacts are SHA-256 verified and can only be deployed to devices with a matching
              hardware target.</span
            >
          </div>
          <section class="panel">
            <div class="panel-title">
              <h2>
                Firmware library <span class="count-badge">{{ state.firmware.length }}</span>
              </h2>
              <span class="subtle-label">64 MiB MAX PER ARTIFACT</span>
            </div>
            <div v-if="state.firmware.length" class="firmware-grid">
              <article v-for="fw in state.firmware" :key="fw.id" class="firmware-card">
                <div class="firmware-card-top">
                  <span class="artifact-icon"><Box :size="24" /></span
                  ><span class="badge online"><Check :size="12" /> Ready</span>
                </div>
                <h3>{{ fw.name }}</h3>
                <div class="firmware-meta">
                  <span class="version">{{ fw.version }}</span
                  ><span>{{ size(fw.size) }}</span>
                </div>
                <p class="hardware-target"><Cpu :size="14" />{{ fw.target }}</p>
                <div class="checksum">
                  <span>SHA-256</span><code :title="fw.sha256">{{ fw.sha256.slice(0, 24) }}…</code
                  ><button class="icon-button" aria-label="Copy checksum" @click="copy(fw.sha256)">
                    <Copy :size="13" />
                  </button>
                </div>
                <div class="firmware-footer">
                  <small>{{ relative(fw.createdAt) }}</small
                  ><button class="text-button" @click="openDeploy(fw)">
                    Deploy <ArrowUpRight :size="15" />
                  </button>
                </div>
              </article>
            </div>
            <div v-else class="empty-state">
              <div class="empty-icon"><Layers3 :size="30" /></div>
              <h3>Make your next release a little easier</h3>
              <p>
                Upload a compiled Arduino or ESP32 firmware image. Give it a version and an
                exact hardware target.
              </p>
              <button class="button primary" @click="open('upload')">
                <Upload :size="16" />Upload first artifact
              </button>
            </div>
          </section>
        </template>

        <template v-else-if="page === 'deployments'">
          <div class="info-strip">
            <Radio :size="19" /><span
              >Devices pull queued updates on their next check-in. Progress is reported by the
              device, including installation failures.</span
            >
          </div>
          <section class="panel">
            <div class="panel-title">
              <h2>
                Deployment history <span class="count-badge">{{ state.deployments.length }}</span>
              </h2>
              <span class="live-indicator"><i></i>Live status</span>
            </div>
            <div v-if="state.deployments.length" class="table-scroll">
              <table>
                <thead>
                  <tr>
                    <th>Device</th>
                    <th>Target version</th>
                    <th>Status</th>
                    <th>Updated</th>
                    <th>Details</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="j in state.deployments" :key="j.id">
                    <td>
                      <strong>{{ j.deviceName }}</strong
                      ><small class="target-label mono">{{ j.id.slice(0, 12) }}</small>
                    </td>
                    <td>
                      <span class="version">{{ j.version }}</span>
                    </td>
                    <td>
                      <span :class="['badge', j.status]"><i></i>{{ j.status }}</span>
                    </td>
                    <td class="muted nowrap">{{ relative(j.updatedAt) }}</td>
                    <td class="job-message">
                      {{ j.message || (j.status === 'queued' ? 'Waiting for device' : '—') }}
                    </td>
                    <td>
                      <button
                        v-if="j.status === 'queued'"
                        class="text-button"
                        :disabled="busy"
                        @click="cancel(j)"
                      >
                        Cancel
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-else class="empty-state">
              <div class="empty-icon"><ArrowUpRight :size="30" /></div>
              <h3>Good things are ready to ship</h3>
              <p>
                Select a firmware artifact and compatible devices to start your first deployment.
              </p>
              <button class="button primary" @click="openDeploy()">
                Create deployment<ArrowRight :size="16" />
              </button>
            </div>
          </section>
        </template>

        <template v-else-if="page === 'connect'">
          <div class="guide-grid">
            <section class="panel guide-main">
              <div class="panel-title">
                <h2>From hardware to dashboard</h2>
                <span class="badge online">HTTP protocol</span>
              </div>
              <div class="guide-step">
                <span>01</span>
                <div>
                  <h3>Enroll your device</h3>
                  <p>
                    Add a name, device type, and hardware target. Copy the generated device ID and
                    token; the token is shown only once.
                  </p>
                  <button class="text-button" @click="open('enroll')">
                    Add a device <ArrowRight :size="15" />
                  </button>
                </div>
              </div>
              <div class="guide-step">
                <span>02</span>
                <div>
                  <h3>Send a heartbeat every 30 seconds</h3>
                  <p>
                    Use the server’s reachable address. Authenticate with your device token and ID.
                    A device goes offline after 90 seconds without a heartbeat.
                  </p>
                  <pre>
POST /api/agent/heartbeat
Authorization: Bearer YOUR_DEVICE_TOKEN
X-Device-ID: YOUR_DEVICE_ID
Content-Type: application/json

{
  "version": "1.0.0",
  "metrics": {
    "cpu": 12.5,
    "memory": 38,
    "temperature": 42,
    "uptime": 3600
  }
}</pre>
                  <p>
                    Only report metrics your hardware supports. CPU, memory, and temperature are
                    optional.
                  </p>
                </div>
              </div>
              <div class="guide-step">
                <span>03</span>
                <div>
                  <h3>Poll, verify, and apply updates</h3>
                  <p>
                    Poll <code>GET /api/agent/jobs</code>. Download the assigned artifact with the
                    same credentials, verify its SHA-256 and hardware target, then report each
                    installation state.
                  </p>
                  <div class="flow">
                    <span>Queued</span><ChevronRight :size="13" /><span>Downloading</span
                    ><ChevronRight :size="13" /><span>Installing</span
                    ><ChevronRight :size="13" /><span>Success</span>
                  </div>
                  <p>
                    Report to <code>POST /api/agent/jobs/:id/status</code> with
                    <code>{"status":"downloading"}</code>. Use <code>failed</code> with a message
                    when an update cannot complete.
                  </p>
                </div>
              </div>
            </section>
            <div class="guide-side">
              <section class="panel guide-card">
                <Cpu :size="25" />
                <h3>ESP32 & Arduino</h3>
                <p>
                  Implement the HTTP protocol in your sketch. ESP32 can use an OTA partition and the
                  Update library. Arduino support depends on the board and bootloader.
                </p>
                <p>
                  USB-only boards need a network-connected gateway to report telemetry and perform
                  flashing.
                </p>
              </section>
              <div class="guide-tip">
                <ShieldCheck :size="20" />
                <p>Use HTTPS when connecting devices outside your local development network.</p>
              </div>
            </div>
          </div>
        </template>
        <footer class="page-footer">
          <span><span class="tiny-dot"></span> Edgenet · Your edge, connected.</span
          >
        </footer>
      </main>
    </div>
  </div>

  <Modal
    v-if="dialog"
    :title="
      dialog === 'enroll'
        ? 'Add a device'
        : dialog === 'upload'
          ? 'Upload firmware'
          : dialog === 'deploy'
            ? 'Deploy an update'
            : dialog === 'credentials'
              ? 'Your device is ready to connect'
              : selected?.name || 'Device details'
    "
    :wide="dialog === 'device'"
    @close="close"
  >
    <p v-if="modalError" class="error-box modal-error" role="alert">{{ modalError }}</p>
    <form v-if="dialog === 'enroll'" class="modal-body" @submit.prevent="enroll">
      <p class="form-description">
        Give your hardware a home. You’ll get a unique access token to connect it.
      </p>
      <label
        >Device name<input
          v-model="form.name"
          required
          maxlength="80"
          placeholder="e.g. Greenhouse sensor"
      /></label>
      <div class="form-row">
        <label
          >Device type<select v-model="form.kind" @change="form.target = targets[form.kind] || ''">
            <option v-for="(label, value) in kinds" :key="value" :value="value">{{ label }}</option>
          </select></label
        ><label
          >Location <span class="optional">optional</span
          ><input v-model="form.location" maxlength="100" placeholder="e.g. Workshop"
        /></label>
      </div>
      <label
        >Hardware target<input
          v-model="form.target"
          required
          maxlength="100"
          placeholder="esp32-devkit-v1"
      /></label>
      <p class="field-hint">
        Use an exact board / architecture identifier. Firmware must match this target.
      </p>
      <div class="modal-actions">
        <button type="button" class="button secondary" :disabled="busy" @click="close">
          Cancel</button
        ><button class="button primary" :disabled="busy">
          <LoaderCircle v-if="busy" class="spin" :size="16" /><Plus v-else :size="16" />Add device
        </button>
      </div>
    </form>
    <div v-else-if="dialog === 'credentials'" class="modal-body">
      <div class="success-callout">
        <ShieldCheck :size="23" />
        <div>
          <strong>Save this token now</strong>
          <p>It’s only shown once. You can rotate it from device details if needed.</p>
        </div>
      </div>
      <label
        >Device ID
        <div class="copy-field">
          <code>{{ credentials.id }}</code
          ><button class="icon-button" aria-label="Copy device ID" @click="copy(credentials.id)">
            <Copy :size="16" />
          </button></div></label
      ><label
        >Device token
        <div class="copy-field">
          <code>{{ credentials.token }}</code
          ><button
            class="icon-button"
            aria-label="Copy device token"
            @click="copy(credentials.token)"
          >
            <Copy :size="16" />
          </button></div></label
      ><label>Firmware HTTP headers</label>
      <pre>{{ connectionHeaders }}</pre>
      <p class="field-hint">
        Use these headers in your Arduino or ESP32 firmware’s HTTP client. Send heartbeats to
        your server at /api/agent/heartbeat. See the connection guide for the request body and update protocol.
      </p>
      <div class="modal-actions">
        <button class="button secondary" @click="copy(connectionHeaders)">
          <Copy :size="16" />Copy headers</button
        ><button class="button primary" @click="close">I saved my token<Check :size="16" /></button>
      </div>
    </div>
    <form v-else-if="dialog === 'upload'" class="modal-body" @submit.prevent="upload">
      <p class="form-description">
        Upload a compiled Arduino or ESP32 firmware image for one hardware target.
      </p>
      <label class="upload-zone"
        ><Upload :size="28" /><strong>{{ uploadFile?.name || 'Choose a firmware artifact' }}</strong
        ><span>{{
          uploadFile ? size(uploadFile.size) : 'Compiled firmware · Up to 64 MiB'
        }}</span
        ><input type="file" required aria-label="Firmware artifact" @change="fileChanged"
      /></label>
      <div class="form-row">
        <label
          >Version<input
            v-model="uploadForm.version"
            required
            maxlength="80"
            placeholder="1.0.0" /></label
        ><label
          >Hardware target<input
            v-model="uploadForm.target"
            required
            maxlength="100"
            placeholder="esp32-devkit-v1"
            list="hardware-targets" /><datalist id="hardware-targets">
            <option
              v-for="target in [...new Set(state.devices.map((d) => d.target))]"
              :key="target"
              :value="target"
            /></datalist
        ></label>
      </div>
      <p class="field-hint">
        The server computes a SHA-256 checksum. The device verifies it before installation.
      </p>
      <div class="modal-actions">
        <button type="button" class="button secondary" :disabled="busy" @click="close">
          Cancel</button
        ><button class="button primary" :disabled="busy">
          <LoaderCircle v-if="busy" class="spin" :size="16" /><Upload v-else :size="16" />{{
            busy ? 'Uploading…' : 'Upload artifact'
          }}
        </button>
      </div>
    </form>
    <form v-else-if="dialog === 'deploy'" class="modal-body" @submit.prevent="deploy">
      <p class="form-description">
        Updates are picked up by the device agent. Offline devices will wait in the queue.
      </p>
      <label
        >Firmware artifact<select v-model="firmwareId" required @change="deviceIds = []">
          <option value="" disabled>Select an artifact</option>
          <option v-for="fw in state.firmware" :key="fw.id" :value="fw.id">
            {{ fw.version }} · {{ fw.target }} · {{ fw.name }}
          </option>
        </select></label
      >
      <div v-if="!state.firmware.length" class="inline-empty">
        No firmware yet.
        <button type="button" class="text-button" @click="open('upload')">
          Upload an artifact first
        </button>
      </div>
      <label class="device-selection-label"
        >Compatible devices <span class="optional">{{ deviceIds.length }} selected</span></label
      >
      <div class="device-selection">
        <label v-for="d in compatible" :key="d.id" class="selection-row"
          ><input
            v-model="deviceIds"
            type="checkbox"
            :value="d.id"
            :disabled="activeJobs.some((j) => j.deviceId === d.id)"
          /><span
            ><strong>{{ d.name }}</strong
            ><small>{{
              activeJobs.some((j) => j.deviceId === d.id)
                ? 'Already has an active deployment'
                : `${d.target} · ${d.version}`
            }}</small></span
          ><span :class="['badge', d.status]"><i></i>{{ d.status }}</span></label
        >
        <p v-if="!compatible.length" class="inline-empty">
          {{
            firmwareId
              ? 'No devices match this hardware target.'
              : 'Select an artifact to see compatible devices.'
          }}
        </p>
      </div>
      <div class="modal-actions">
        <button type="button" class="button secondary" :disabled="busy" @click="close">
          Cancel</button
        ><button class="button primary" :disabled="busy || !firmwareId || !deviceIds.length">
          <LoaderCircle v-if="busy" class="spin" :size="16" /><ArrowUpRight
            v-else
            :size="16"
          />Deploy to {{ deviceIds.length }} device{{ deviceIds.length === 1 ? '' : 's' }}
        </button>
      </div>
    </form>
    <div v-else-if="dialog === 'device' && selected" class="modal-body">
      <div class="detail-summary">
        <span :class="['device-icon large', selected.kind]"
          ><component :is="kindIcon(selected.kind)" :size="30"
        /></span>
        <div>
          <h3>{{ kinds[selected.kind] }}</h3>
          <p>
            {{ selected.target }}<span v-if="selected.location"> · {{ selected.location }}</span>
          </p>
        </div>
        <span :class="['badge', selected.status]"><i></i>{{ selected.status }}</span>
      </div>
      <div class="detail-stats">
        <div>
          <span>CPU</span><strong>{{ metric(selected.metrics.cpu) }}</strong>
        </div>
        <div>
          <span>Memory</span><strong>{{ metric(selected.metrics.memory) }}</strong>
        </div>
        <div>
          <span>Temperature</span
          ><strong>{{
            selected.metrics.temperature == null
              ? '—'
              : `${selected.metrics.temperature.toFixed(1)}°C`
          }}</strong>
        </div>
        <div>
          <span>Uptime</span
          ><strong>{{ selected.lastSeen ? uptime(selected.metrics.uptime) : '—' }}</strong>
        </div>
      </div>
      <div class="chart-title">
        <strong>CPU utilization</strong
        ><span>Last {{ selected.history.length }} heartbeats · 0–100%</span>
      </div>
      <div class="telemetry-chart">
        <template v-if="selected.history.some((h) => h.cpu != null)"
          ><svg
            viewBox="0 0 600 120"
            role="img"
            aria-label="CPU utilization over recent heartbeats"
            preserveAspectRatio="none"
          >
            <line
              v-for="y in [10, 60, 110]"
              :key="y"
              x1="0"
              :y1="y"
              x2="600"
              :y2="y"
              stroke="#e8ecea"
              stroke-dasharray="4 4"
            />
            <polyline
              :points="cpuPoints(selected)"
              fill="none"
              stroke="#13866c"
              stroke-width="2.5"
              vector-effect="non-scaling-stroke"
            /></svg></template
        ><span v-else>Waiting for CPU telemetry</span>
      </div>
      <dl class="device-properties">
        <div>
          <dt>Device ID</dt>
          <dd class="mono">{{ selected.id }}</dd>
        </div>
        <div>
          <dt>Firmware</dt>
          <dd>{{ selected.version }}</dd>
        </div>
        <div>
          <dt>Last heartbeat</dt>
          <dd>
            {{
              selected.lastSeen ? new Date(selected.lastSeen).toLocaleString() : 'Never connected'
            }}
          </dd>
        </div>
      </dl>
      <div class="modal-actions detail-actions">
        <button class="text-button danger" :disabled="busy" @click="remove(selected)">
          Remove device
        </button>
        <div>
          <button class="button secondary" :disabled="busy" @click="rotate(selected)">
            Rotate token</button
          ><button class="button primary" @click="openDeploy(undefined, selected)">
            <ArrowUpRight :size="16" />Update
          </button>
        </div>
      </div>
    </div>
  </Modal>
  <Transition name="toast"
    ><div v-if="notice" class="toast-message" role="status">
      <CheckCircle2 :size="19" /><span>{{ notice }}</span
      ><button class="icon-button" aria-label="Dismiss notification" @click="notice = ''">
        <X :size="15" />
      </button></div
  ></Transition>
</template>
