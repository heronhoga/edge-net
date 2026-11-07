<script setup lang="ts">
import { ref } from 'vue'
import { LoaderCircle, Upload } from 'lucide-vue-next'
import { size } from '../../../shared/lib/format'
import { MAX_ARTIFACT_SIZE } from '../api'
import type { FirmwareUpload } from '../types'

defineProps<{ busy: boolean; targets: string[] }>()
const emit = defineEmits<{ submit: [input: FirmwareUpload]; close: [] }>()
const form = ref({ version: '', target: '' })
const file = ref<File | null>(null)
const error = ref('')

function fileChanged(event: Event) {
  file.value = (event.target as HTMLInputElement).files?.[0] || null
  error.value = ''
}

function submit() {
  if (!file.value) {
    error.value = 'Choose an artifact to upload.'
    return
  }
  if (file.value.size > MAX_ARTIFACT_SIZE) {
    error.value = 'The artifact must be 64 MiB or smaller.'
    return
  }
  error.value = ''
  emit('submit', { ...form.value, file: file.value })
}
</script>

<template>
  <form class="modal-body" @submit.prevent="submit">
    <p v-if="error" class="error-box" role="alert">{{ error }}</p>
    <p class="form-description">
      Upload a compiled Arduino or ESP32 firmware image for one hardware target.
    </p>
    <label class="upload-zone">
      <Upload :size="28" /><strong>{{ file?.name || 'Choose a firmware artifact' }}</strong>
      <span>{{ file ? size(file.size) : 'Compiled firmware · Up to 64 MiB' }}</span>
      <input type="file" required aria-label="Firmware artifact" @change="fileChanged" />
    </label>
    <div class="form-row">
      <label
        >Version<input v-model="form.version" required maxlength="80" placeholder="1.0.0"
      /></label>
      <label
        >Hardware target
        <input
          v-model="form.target"
          required
          maxlength="100"
          placeholder="esp32-devkit-v1"
          list="hardware-targets"
        />
        <datalist id="hardware-targets">
          <option v-for="target in targets" :key="target" :value="target" />
        </datalist>
      </label>
    </div>
    <p class="field-hint">
      The server computes a SHA-256 checksum. The device verifies it before installation.
    </p>
    <div class="modal-actions">
      <button type="button" class="button secondary" :disabled="busy" @click="emit('close')">
        Cancel
      </button>
      <button class="button primary" :disabled="busy">
        <LoaderCircle v-if="busy" class="spin" :size="16" /><Upload v-else :size="16" />{{
          busy ? 'Uploading…' : 'Upload artifact'
        }}
      </button>
    </div>
  </form>
</template>
