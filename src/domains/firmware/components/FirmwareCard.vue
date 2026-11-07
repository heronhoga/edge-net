<script setup lang="ts">
import { ArrowUpRight, Box, Check, Copy, Cpu } from 'lucide-vue-next'
import { relative, size } from '../../../shared/lib/format'
import type { Firmware } from '../types'

defineProps<{ firmware: Firmware }>()
const emit = defineEmits<{ copy: [text: string]; deploy: [firmware: Firmware] }>()
</script>

<template>
  <article class="firmware-card">
    <div class="firmware-card-top">
      <span class="artifact-icon"><Box :size="24" /></span
      ><span class="badge online"><Check :size="12" /> Ready</span>
    </div>
    <h3>{{ firmware.name }}</h3>
    <div class="firmware-meta">
      <span class="version">{{ firmware.version }}</span
      ><span>{{ size(firmware.size) }}</span>
    </div>
    <p class="hardware-target"><Cpu :size="14" />{{ firmware.target }}</p>
    <div class="checksum">
      <span>SHA-256</span><code :title="firmware.sha256">{{ firmware.sha256.slice(0, 24) }}…</code>
      <button class="icon-button" aria-label="Copy checksum" @click="emit('copy', firmware.sha256)">
        <Copy :size="13" />
      </button>
    </div>
    <div class="firmware-footer">
      <small>{{ relative(firmware.createdAt) }}</small>
      <button class="text-button" @click="emit('deploy', firmware)">
        Deploy <ArrowUpRight :size="15" />
      </button>
    </div>
  </article>
</template>
