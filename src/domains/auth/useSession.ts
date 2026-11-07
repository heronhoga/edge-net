import { ref } from 'vue'

export function useSession() {
  const token = ref(sessionStorage.getItem('edgenet-token') || '')
  const authenticated = ref(false)

  function remember() {
    sessionStorage.setItem('edgenet-token', token.value)
  }

  function clear() {
    authenticated.value = false
    token.value = ''
    sessionStorage.removeItem('edgenet-token')
  }

  return { token, authenticated, remember, clear }
}
