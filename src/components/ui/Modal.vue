<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { X } from 'lucide-vue-next'
defineProps<{ title: string; wide?: boolean }>()
const emit = defineEmits<{ close: [] }>()
const root = ref<HTMLElement>()
const previous = document.activeElement as HTMLElement | null
const previousOverflow = document.body.style.overflow
function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
  if (e.key !== 'Tab') return
  const nodes = [
    ...root.value!.querySelectorAll<HTMLElement>(
      'button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), a[href], [tabindex="0"]',
    ),
  ]
  const first = nodes[0],
    last = nodes[nodes.length - 1]
  if (e.shiftKey && document.activeElement === first) {
    e.preventDefault()
    last?.focus()
  } else if (!e.shiftKey && document.activeElement === last) {
    e.preventDefault()
    first?.focus()
  }
}
onMounted(async () => {
  document.body.style.overflow = 'hidden'
  await nextTick()
  root.value?.querySelector<HTMLElement>('input, select, button')?.focus()
})
onBeforeUnmount(() => {
  document.body.style.overflow = previousOverflow
  previous?.focus()
})
</script>

<template>
  <Teleport to="body">
    <div class="modal-backdrop" @mousedown.self="emit('close')">
      <section
        ref="root"
        class="modal"
        :class="{ wide }"
        role="dialog"
        aria-modal="true"
        aria-labelledby="modal-title"
        @keydown="onKey"
      >
        <header class="modal-header">
          <h2 id="modal-title">{{ title }}</h2>
          <button class="icon-button" aria-label="Close dialog" @click="emit('close')">
            <X :size="20" />
          </button>
        </header>
        <slot />
      </section>
    </div>
  </Teleport>
</template>
