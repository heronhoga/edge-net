<script setup lang="ts">
import { CheckCircle2, X } from 'lucide-vue-next'
defineProps<{ message: string }>()
const emit = defineEmits<{ dismiss: [] }>()
</script>

<template>
  <Transition name="toast">
    <div v-if="message" class="toast-message" role="status">
      <CheckCircle2 :size="19" /><span>{{ message }}</span>
      <button class="icon-button" aria-label="Dismiss notification" @click="emit('dismiss')">
        <X :size="15" />
      </button>
    </div>
  </Transition>
</template>
