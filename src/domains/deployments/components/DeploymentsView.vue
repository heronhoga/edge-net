<script setup lang="ts">
import { ArrowRight, ArrowUpRight, Radio } from 'lucide-vue-next'
import StatusBadge from '../../../components/ui/StatusBadge.vue'
import { relative } from '../../../shared/lib/format'
import type { Deployment } from '../types'

defineProps<{ deployments: Deployment[]; busy: boolean }>()
const emit = defineEmits<{ create: []; cancel: [job: Deployment] }>()
</script>

<template>
  <div class="info-strip">
    <Radio :size="19" /><span
      >Devices pull queued updates on their next check-in. Progress is reported by the device,
      including installation failures.</span
    >
  </div>
  <section class="panel">
    <div class="panel-title">
      <h2>
        Deployment history <span class="count-badge">{{ deployments.length }}</span>
      </h2>
      <span class="live-indicator"><i></i>Live status</span>
    </div>
    <div v-if="deployments.length" class="table-scroll">
      <table>
        <thead>
          <tr>
            <th>Device</th>
            <th>Target version</th>
            <th>Status</th>
            <th>Updated</th>
            <th>Details</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="job in deployments" :key="job.id">
            <td>
              <strong>{{ job.deviceName }}</strong
              ><small class="target-label mono">{{ job.id.slice(0, 12) }}</small>
            </td>
            <td>
              <span class="version">{{ job.version }}</span>
            </td>
            <td><StatusBadge :status="job.status" /></td>
            <td class="muted nowrap">{{ relative(job.updatedAt) }}</td>
            <td class="job-message">
              {{ job.message || (job.status === 'queued' ? 'Waiting for device' : '—') }}
            </td>
            <td>
              <button
                v-if="job.status === 'queued'"
                class="text-button"
                :disabled="busy"
                @click="emit('cancel', job)"
              >
                Cancel
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else class="empty-state">
      <div class="empty-icon"><ArrowUpRight :size="30" /></div>
      <h3>Good things are ready to ship</h3>
      <p>Select a firmware artifact and compatible devices to start your first deployment.</p>
      <button class="button primary" @click="emit('create')">
        Create deployment<ArrowRight :size="16" />
      </button>
    </div>
  </section>
</template>
