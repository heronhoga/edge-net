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
import type { Dialog, Page, Snapshot, WorkspaceSnapshot } from './types'

function emptySnapshot(): Snapshot {
  return { devices: [], firmware: [], deployments: [], events: [], serverTime: '' }
}

export function useWorkspace() {
  const session = useSession()
  const { token, user, authenticated, googleEnabled, adminEnabled, initializing } = session
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
  const api = createAdminApi(session.headers, expireSession)
  const devicesApi = createDevicesApi(api)
  const firmwareApi = createFirmwareApi(api)
  const deploymentsApi = createDeploymentsApi(api)
  let timer: ReturnType<typeof setInterval> | undefined

  let generation = 0

  function resetWorkspace() {
    generation++
    dialog.value = ''
    page.value = 'overview'
    notice.value = ''
    modalError.value = ''
    credentials.value = { id: '', token: '' }
    state.value = emptySnapshot()
  }

  function expireSession() {
    session.clear()
    resetWorkspace()
  }

  async function logout() {
    try {
      await session.signOut()
      resetWorkspace()
      error.value = ''
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Could not sign out'
    }
  }

  async function refresh() {
    if (loading.value || !authenticated.value) return
    loading.value = true
    const requestGeneration = generation
    try {
      const snapshot = await api<WorkspaceSnapshot>('/state')
      if (requestGeneration !== generation) return
      // A Google account can change in another tab. Drop old dialogs and cached
      // views before displaying the account that owns this response.
      if (snapshot.user.id !== user.value?.id) resetWorkspace()
      session.updateProfile(snapshot.user, snapshot.csrfToken)
      state.value = snapshot
      error.value = ''
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Cannot reach the API'
    } finally {
      loading.value = false
    }
  }

  async function login(value: string) {
    token.value = value.trim()
    loading.value = true
    try {
      await session.load()
      if (!authenticated.value) throw new Error('Invalid admin access token')
      error.value = ''
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Sign-in failed'
    } finally {
      loading.value = false
    }
    if (authenticated.value) await refresh()
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

  async function mutate(action: (isCurrent: () => boolean) => Promise<void>) {
    if (busy.value) return
    busy.value = true
    modalError.value = ''
    const requestGeneration = generation
    const isCurrent = () => requestGeneration === generation
    try {
      await action(isCurrent)
      if (isCurrent()) await refresh()
    } catch (e) {
      if (!isCurrent()) return
      const message = e instanceof Error ? e.message : 'Request failed'
      if (dialog.value) modalError.value = message
      else error.value = message
    } finally {
      busy.value = false
    }
  }

  async function enroll(input: Enrollment) {
    await mutate(async (isCurrent) => {
      const result = await devicesApi.enroll(input)
      if (!isCurrent()) return
      credentials.value = { id: result.device.id, token: result.token }
      dialog.value = 'credentials'
      toast('Device enrolled. Connect it using its new token.')
    })
  }

  async function upload(input: FirmwareUpload) {
    await mutate(async (isCurrent) => {
      await firmwareApi.upload(input)
      if (!isCurrent()) return
      dialog.value = ''
      toast('Firmware uploaded and SHA-256 checksum generated.')
    })
  }

  async function deploy(input: DeploymentRequest) {
    await mutate(async (isCurrent) => {
      await deploymentsApi.create(input)
      if (!isCurrent()) return
      dialog.value = ''
      page.value = 'deployments'
      toast('Deployment queued. Devices will pick it up on their next check-in.')
    })
  }

  async function cancel(job: Deployment) {
    await mutate(async (isCurrent) => {
      await deploymentsApi.cancel(job.id)
      if (!isCurrent()) return
      toast('Deployment cancelled.')
    })
  }

  async function rotate(device: Device) {
    if (
      !confirm(`Replace the access token for ${device.name}? Its current token will stop working.`)
    )
      return
    await mutate(async (isCurrent) => {
      const result = await devicesApi.rotateToken(device.id)
      if (!isCurrent()) return
      credentials.value = { id: device.id, token: result.token }
      dialog.value = 'credentials'
    })
  }

  async function remove(device: Device) {
    if (!confirm(`Remove ${device.name} and revoke its access?`)) return
    await mutate(async (isCurrent) => {
      await devicesApi.remove(device.id)
      if (!isCurrent()) return
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

  async function restoreSession() {
    initializing.value = true
    try {
      const previousUser = user.value?.id
      await session.load()
      if (user.value?.id !== previousUser) resetWorkspace()
      if (authenticated.value) await refresh()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Could not restore your session'
    } finally {
      initializing.value = false
    }
  }

  onMounted(() => {
    const url = new URL(window.location.href)
    const authError = url.searchParams.get('auth_error')
    if (authError) {
      error.value =
        authError === 'cancelled'
          ? 'Google sign-in was cancelled. Please try again.'
          : authError === 'expired'
            ? 'Your sign-in request expired. Please try again.'
            : 'Google sign-in failed. Please try again.'
      url.searchParams.delete('auth_error')
      window.history.replaceState({}, '', url)
    }
    void restoreSession()
    timer = setInterval(() => {
      if (authenticated.value) void refresh()
    }, 5000)
  })
  onUnmounted(() => clearInterval(timer))

  return {
    authenticated,
    user,
    googleEnabled,
    adminEnabled,
    initializing,
    startGoogle: session.startGoogle,
    restoreSession,
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
