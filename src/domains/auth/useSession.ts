import { computed, ref } from 'vue'
import type { SessionInfo, User } from './types'

export function useSession() {
  const token = ref(sessionStorage.getItem('edgenet-token') || '')
  const user = ref<User | null>(null)
  const csrfToken = ref('')
  const googleEnabled = ref(false)
  const adminEnabled = ref(false)
  const initializing = ref(true)
  const authenticated = computed(() => user.value !== null)

  function headers() {
    const headers = new Headers()
    if (token.value) headers.set('Authorization', `Bearer ${token.value}`)
    if (csrfToken.value) headers.set('X-CSRF-Token', csrfToken.value)
    return headers
  }

  async function load() {
    const response = await fetch('/api/auth/session', {
      headers: headers(),
      credentials: 'same-origin',
      signal: AbortSignal.timeout(15000),
    })
    if (!response.ok) throw new Error('Cannot load your session. Please try again.')
    const info: SessionInfo = await response.json()
    googleEnabled.value = info.googleEnabled
    adminEnabled.value = info.adminEnabled
    user.value = info.user
    csrfToken.value = info.csrfToken
    if (!info.user) clear()
    else if (info.user.provider === 'admin') sessionStorage.setItem('edgenet-token', token.value)
  }

  async function signOut() {
    if (token.value) {
      clear()
      return
    }
    const response = await fetch('/api/auth/logout', {
      method: 'POST',
      headers: headers(),
      credentials: 'same-origin',
      signal: AbortSignal.timeout(15000),
    })
    if (!response.ok && response.status !== 401)
      throw new Error('Could not sign out. Please try again.')
    clear()
  }

  function startGoogle() {
    clear()
    window.location.assign('/api/auth/google')
  }

  function updateProfile(profile: User, csrf: string) {
    user.value = profile
    csrfToken.value = csrf
  }

  function clear() {
    user.value = null
    csrfToken.value = ''
    token.value = ''
    sessionStorage.removeItem('edgenet-token')
  }

  return {
    token,
    user,
    csrfToken,
    authenticated,
    googleEnabled,
    adminEnabled,
    initializing,
    headers,
    load,
    signOut,
    startGoogle,
    updateProfile,
    clear,
  }
}
