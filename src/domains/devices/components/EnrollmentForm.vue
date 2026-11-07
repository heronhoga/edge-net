<script setup lang="ts">
import { ref } from 'vue'
import { LoaderCircle, Plus } from 'lucide-vue-next'
import { kinds, targets } from '../hardware'
import type { Enrollment } from '../types'

defineProps<{ busy: boolean }>()
const emit = defineEmits<{ submit: [input: Enrollment]; close: [] }>()
const form = ref<Enrollment>({ name: '', kind: 'esp32', target: targets.esp32!, location: '' })
</script>

<template>
  <form class="modal-body" @submit.prevent="emit('submit', { ...form })">
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
      >
      <label
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
      <button type="button" class="button secondary" :disabled="busy" @click="emit('close')">
        Cancel
      </button>
      <button class="button primary" :disabled="busy">
        <LoaderCircle v-if="busy" class="spin" :size="16" /><Plus v-else :size="16" />Add device
      </button>
    </div>
  </form>
</template>
