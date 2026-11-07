import { onUnmounted, ref } from 'vue'

export function useNotice() {
  const notice = ref('')
  let timer: ReturnType<typeof setTimeout> | undefined

  function toast(message: string) {
    notice.value = message
    clearTimeout(timer)
    timer = setTimeout(() => (notice.value = ''), 4500)
  }

  onUnmounted(() => clearTimeout(timer))
  return { notice, toast }
}
