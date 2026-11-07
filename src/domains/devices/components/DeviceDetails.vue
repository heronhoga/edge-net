<script setup lang="ts">
import { ArrowUpRight } from 'lucide-vue-next'
import StatusBadge from '../../../components/ui/StatusBadge.vue'
import { metric, uptime } from '../../../shared/lib/format'
import { kindIcon, kinds } from '../hardware'
import type { Device } from '../types'
import TelemetryChart from './TelemetryChart.vue'

defineProps<{ device: Device; busy: boolean }>()
const emit = defineEmits<{
  remove: [device: Device]
  rotate: [device: Device]
  deploy: [device: Device]
}>()
</script>

<template>
  <div class="modal-body">
    <div class="detail-summary">
      <span :class="['device-icon large', device.kind]"
        ><component :is="kindIcon(device.kind)" :size="30"
      /></span>
      <div>
        <h3>{{ kinds[device.kind] }}</h3>
        <p>
          {{ device.target }}<span v-if="device.location"> · {{ device.location }}</span>
        </p>
      </div>
      <StatusBadge :status="device.status" />
    </div>
    <div class="detail-stats">
      <div>
        <span>CPU</span><strong>{{ metric(device.metrics.cpu) }}</strong>
      </div>
      <div>
        <span>Memory</span><strong>{{ metric(device.metrics.memory) }}</strong>
      </div>
      <div>
        <span>Temperature</span
        ><strong>{{
          device.metrics.temperature == null ? '—' : `${device.metrics.temperature.toFixed(1)}°C`
        }}</strong>
      </div>
      <div>
        <span>Uptime</span
        ><strong>{{ device.lastSeen ? uptime(device.metrics.uptime) : '—' }}</strong>
      </div>
    </div>
    <TelemetryChart :history="device.history" />
    <dl class="device-properties">
      <div>
        <dt>Device ID</dt>
        <dd class="mono">{{ device.id }}</dd>
      </div>
      <div>
        <dt>Firmware</dt>
        <dd>{{ device.version }}</dd>
      </div>
      <div>
        <dt>Last heartbeat</dt>
        <dd>
          {{ device.lastSeen ? new Date(device.lastSeen).toLocaleString() : 'Never connected' }}
        </dd>
      </div>
    </dl>
    <div class="modal-actions detail-actions">
      <button class="text-button danger" :disabled="busy" @click="emit('remove', device)">
        Remove device
      </button>
      <div>
        <button class="button secondary" :disabled="busy" @click="emit('rotate', device)">
          Rotate token
        </button>
        <button class="button primary" @click="emit('deploy', device)">
          <ArrowUpRight :size="16" />Update
        </button>
      </div>
    </div>
  </div>
</template>
