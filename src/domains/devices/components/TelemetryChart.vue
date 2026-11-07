<script setup lang="ts">
import { computed } from 'vue'
import type { Device } from '../types'

const props = defineProps<{ history: Device['history'] }>()
const points = computed(() =>
  props.history
    .flatMap((sample, index) =>
      sample.cpu == null
        ? []
        : [`${(index / Math.max(props.history.length - 1, 1)) * 600},${110 - sample.cpu}`],
    )
    .join(' '),
)
</script>

<template>
  <div class="chart-title">
    <strong>CPU utilization</strong><span>Last {{ history.length }} heartbeats · 0–100%</span>
  </div>
  <div class="telemetry-chart">
    <svg
      v-if="history.some((sample) => sample.cpu != null)"
      viewBox="0 0 600 120"
      role="img"
      aria-label="CPU utilization over recent heartbeats"
      preserveAspectRatio="none"
    >
      <line
        v-for="y in [10, 60, 110]"
        :key="y"
        x1="0"
        :y1="y"
        x2="600"
        :y2="y"
        stroke="var(--line)"
        stroke-dasharray="4 4"
      />
      <polyline
        :points="points"
        fill="none"
        stroke="var(--primary)"
        stroke-width="2.5"
        vector-effect="non-scaling-stroke"
      />
    </svg>
    <span v-else>Waiting for CPU telemetry</span>
  </div>
</template>
