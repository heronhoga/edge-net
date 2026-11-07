import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useSession } from '../domains/auth/useSession'
import { createDevicesApi } from '../domains/devices/api'
import { createFirmwareApi } from '../domains/firmware/api'
import { createDeploymentsApi } from '../domains/deployments/api'
import type { Device, DeviceCredentials, Enrollment } from '../domains/devices/types'
import type { Firmware, FirmwareUpload } from '../domains/firmware/types'
import type { Deployment, DeploymentRequest } from '../domains/deployments/types'
import { createAdminApi } from '../shared/api/admin'
import { useNotice } from '../shared/composables/useNotice'
import type { Dialog, Page, Snapshot } from './types'

function emptySnapshot(): Snapshot {
  return { devices: [], firmware: [], deployments: [], events: [], serverTime: '' }
}

export function useWorkspace() {
  const session = useSession()
  const { token, authenticated } = session
  const { notice, toast } = useNotice()
  const state = ref<Snapshot>(emptySnapshot())
  const loading = ref(false)
  const busy = ref(false)
  const error = ref('')
  const modalError = ref('')
  const page = ref<Page>('overview')
  const dialog = ref<Dialog>('')
  const selectedId = ref('')
  const credentials = ref<DeviceCredentials>({ id: '', token: '' })
  const deploymentDefaults = ref<DeploymentRequest>({ firmwareId: '', deviceIds: [] })
  const selected = computed(() =>
    state.value.devices.find((device) => device.id === selectedId.value),
  )
  const api = createAdminApi(() => token.value, logout)
  const devicesApi = createDevicesApi(api)
  const firmwareApi = createFirmwareApi(api)
  const deploymentsApi = createDeploymentsApi(api)
  let timer: ReturnType<typeof setInterval> | undefined

  function logout() {
    session.clear()
    dialog.value = ''
    credentials.value = { id: '', token: '' }
    state.value = emptySnapshot()
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

  async function login(value: string) {
    token.value = value.trim()
    await refresh()
    if (authenticated.value) session.remember()
  }

  function open(kind: Dialog) {
    modalError.value = ''
    dialog.value = kind
  }

  function close() {
    if (busy.value) return
    dialog.value = ''
    credentials.value = { id: '', token: '' }
  }

  function openDevice(device: Device) {
    selectedId.value = device.id
    open('device')
  }

  function openDeploy(firmware?: Firmware, device?: Device) {
    deploymentDefaults.value = {
      firmwareId:
        firmware?.id ||
        state.value.firmware.find((fw) => !device || fw.target === device.target)?.id ||
        '',
      deviceIds: device ? [device.id] : [],
    }
    open('deploy')
  }

  async function mutate(action: () => Promise<void>) {
    if (busy.value) return
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

  async function enroll(input: Enrollment) {
    await mutate(async () => {
      const result = await devicesApi.enroll(input)
      credentials.value = { id: result.device.id, token: result.token }
      dialog.value = 'credentials'
      toast('Device enrolled. Connect it using its new token.')
    })
  }

  async function upload(input: FirmwareUpload) {
    await mutate(async () => {
      await firmwareApi.upload(input)
      dialog.value = ''
      toast('Firmware uploaded and SHA-256 checksum generated.')
    })
  }

  async function deploy(input: DeploymentRequest) {
    await mutate(async () => {
      await deploymentsApi.create(input)
      dialog.value = ''
      page.value = 'deployments'
      toast('Deployment queued. Devices will pick it up on their next check-in.')
    })
  }

  async function cancel(job: Deployment) {
    await mutate(async () => {
      await deploymentsApi.cancel(job.id)
      toast('Deployment cancelled.')
    })
  }

  async function rotate(device: Device) {
    if (
      !confirm(`Replace the access token for ${device.name}? Its current token will stop working.`)
    )
      return
    await mutate(async () => {
      const result = await devicesApi.rotateToken(device.id)
      credentials.value = { id: device.id, token: result.token }
      dialog.value = 'credentials'
    })
  }

  async function remove(device: Device) {
    if (!confirm(`Remove ${device.name} and revoke its access?`)) return
    await mutate(async () => {
      await devicesApi.remove(device.id)
      dialog.value = ''
      toast('Device removed.')
    })
  }

  async function copy(text: string) {
    try {
      await navigator.clipboard.writeText(text)
      toast('Copied to clipboard.')
    } catch {
      const message = 'Clipboard unavailable. Select and copy the text manually.'
      if (dialog.value) modalError.value = message
      else error.value = message
    }
  }

  onMounted(() => {
    if (token.value) void refresh()
    timer = setInterval(() => {
      if (authenticated.value) void refresh()
    }, 5000)
  })
  onUnmounted(() => clearInterval(timer))

  return {
    authenticated,
    loading,
    busy,
    error,
    modalError,
    notice,
    page,
    dialog,
    state,
    selected,
    credentials,
    deploymentDefaults,
    login,
    logout,
    refresh,
    open,
    close,
    openDevice,
    openDeploy,
    enroll,
    upload,
    deploy,
    cancel,
    rotate,
    remove,
    copy,
  }
}
