<script setup lang="ts">
import { Layers3, ShieldCheck, Upload } from 'lucide-vue-next'
import FirmwareCard from './FirmwareCard.vue'
import type { Firmware } from '../types'

defineProps<{ firmware: Firmware[] }>()
const emit = defineEmits<{ upload: []; copy: [text: string]; deploy: [firmware: Firmware] }>()
</script>

<template>
  <div class="info-strip">
    <ShieldCheck :size="19" /><span
      >Artifacts are SHA-256 verified and can only be deployed to devices with a matching hardware
      target.</span
    >
  </div>
  <section class="panel">
    <div class="panel-title">
      <h2>
        Firmware library <span class="count-badge">{{ firmware.length }}</span>
      </h2>
      <span class="subtle-label">64 MiB MAX PER ARTIFACT</span>
    </div>
    <div v-if="firmware.length" class="firmware-grid">
      <FirmwareCard
        v-for="artifact in firmware"
        :key="artifact.id"
        :firmware="artifact"
        @copy="emit('copy', $event)"
        @deploy="emit('deploy', $event)"
      />
    </div>
    <div v-else class="empty-state">
      <div class="empty-icon"><Layers3 :size="30" /></div>
      <h3>Make your next release a little easier</h3>
      <p>
        Upload a compiled Arduino or ESP32 firmware image. Give it a version and an exact hardware
        target.
      </p>
      <button class="button primary" @click="emit('upload')">
        <Upload :size="16" />Upload first artifact
      </button>
    </div>
  </section>
</template>
