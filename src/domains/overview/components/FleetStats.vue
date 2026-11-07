<script setup lang="ts">
import { computed } from 'vue'
import { ArrowUpRight, Cpu, Wifi, WifiOff } from 'lucide-vue-next'
import type { Device } from '../../devices/types'

const props = defineProps<{ devices: Device[]; activeJobCount: number; firmwareCount: number }>()
const online = computed(() => props.devices.filter((device) => device.status === 'online').length)
const offline = computed(() => props.devices.filter((device) => device.status === 'offline').length)
const pending = computed(() => props.devices.filter((device) => device.status === 'pending').length)
</script>

<template>
  <section class="stats-grid" aria-label="Fleet statistics">
    <article class="stat-card">
      <div class="stat-top">
        <span>Total devices</span><span class="stat-icon"><Cpu :size="18" /></span>
      </div>
      <div class="stat-value">
        {{ devices.length }}<span class="stat-caption">in your fleet</span>
      </div>
      <div class="stat-bottom">
        <span class="tiny-dot dark"></span
        >{{ new Set(devices.map((device) => device.kind)).size }} hardware types
      </div>
    </article>
    <article class="stat-card">
      <div class="stat-top">
        <span>Online now</span><span class="stat-icon green"><Wifi :size="18" /></span>
      </div>
      <div class="stat-value">
        {{ online
        }}<span class="stat-pill"
          >{{ devices.length ? Math.round((online / devices.length) * 100) : 0 }}%</span
        >
      </div>
      <div class="stat-bottom"><span class="tiny-dot"></span>Heartbeat within 90 seconds</div>
    </article>
    <article class="stat-card">
      <div class="stat-top">
        <span>Offline devices</span><span class="stat-icon amber"><WifiOff :size="18" /></span>
      </div>
      <div class="stat-value">
        {{ offline
        }}<span class="stat-caption">{{ offline ? 'need a check-in' : 'looking good' }}</span>
      </div>
      <div class="stat-bottom">
        <span class="tiny-dot amber"></span>{{ pending }} awaiting first connection
      </div>
    </article>
    <article class="stat-card">
      <div class="stat-top">
        <span>Active deployments</span
        ><span class="stat-icon blue"><ArrowUpRight :size="18" /></span>
      </div>
      <div class="stat-value">
        {{ activeJobCount }}<span class="stat-caption">in progress</span>
      </div>
      <div class="stat-bottom">
        <span class="tiny-dot blue"></span>{{ firmwareCount }} firmware artifacts available
      </div>
    </article>
  </section>
</template>
