<script setup lang="ts">
import { toRef } from 'vue'
import { ArrowUpRight, ChevronRight, Cpu, Plus, RefreshCw, Search } from 'lucide-vue-next'
import StatusBadge from '../../../components/ui/StatusBadge.vue'
import { metric, relative } from '../../../shared/lib/format'
import { kindIcon, kinds } from '../hardware'
import { useDeviceFilters } from '../useDeviceFilters'
import type { Device } from '../types'

const props = defineProps<{ devices: Device[] }>()
const emit = defineEmits<{ enroll: []; deploy: []; view: [device: Device] }>()
const { query, filter, filteredDevices, clearFilters } = useDeviceFilters(toRef(props, 'devices'))
</script>

<template>
  <section class="panel device-panel">
    <div class="panel-title">
      <div>
        <h2>
          Your devices <span class="count-badge">{{ devices.length }}</span>
        </h2>
        <p>A live look at your connected infrastructure.</p>
      </div>
      <button class="button secondary small" @click="emit('deploy')">
        <ArrowUpRight :size="16" />Deploy update
      </button>
    </div>
    <div class="table-toolbar">
      <div class="tabs" aria-label="Filter device status">
        <button
          v-for="status in ['all', 'online', 'offline', 'pending']"
          :key="status"
          :class="{ selected: filter === status }"
          @click="filter = status"
        >
          {{ status === 'all' ? 'All devices' : status === 'pending' ? 'Pending' : status }}
          <span v-if="status === 'all'">{{ devices.length }}</span>
        </button>
      </div>
      <label class="search"
        ><Search :size="16" /><input
          v-model="query"
          aria-label="Search devices"
          placeholder="Search devices…"
      /></label>
    </div>
    <div v-if="filteredDevices.length" class="table-scroll">
      <table>
        <thead>
          <tr>
            <th>Device</th>
            <th>Status</th>
            <th>Firmware</th>
            <th>CPU / Memory</th>
            <th>Last seen</th>
            <th><span class="sr-only">Details</span></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="device in filteredDevices" :key="device.id">
            <td>
              <button class="device-name" @click="emit('view', device)">
                <span :class="['device-icon', device.kind]"
                  ><component :is="kindIcon(device.kind)" :size="21"
                /></span>
                <span
                  ><strong>{{ device.name }}</strong
                  ><small
                    >{{ kinds[device.kind]
                    }}<span v-if="device.location"> · {{ device.location }}</span></small
                  ></span
                >
              </button>
            </td>
            <td><StatusBadge :status="device.status" /></td>
            <td>
              <span class="version">{{ device.version }}</span
              ><small class="target-label">{{ device.target }}</small>
            </td>
            <td>
              <div class="resource-values">
                <span>{{ metric(device.metrics.cpu) }}</span
                ><span class="resource-divider">/</span
                ><span>{{ metric(device.metrics.memory) }}</span>
              </div>
              <div class="resource-meter">
                <i :style="{ width: `${device.metrics.cpu || 0}%` }"></i>
              </div>
            </td>
            <td class="muted nowrap">{{ relative(device.lastSeen) }}</td>
            <td>
              <button
                class="icon-button"
                :aria-label="`View ${device.name}`"
                @click="emit('view', device)"
              >
                <ChevronRight :size="18" />
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else class="empty-state">
      <div class="empty-icon"><Cpu :size="30" /></div>
      <h3>{{ devices.length ? 'No matching devices' : 'Your fleet starts here' }}</h3>
      <p>
        {{
          devices.length
            ? 'Try another search or status filter.'
            : 'Connect an ESP32 or Arduino and give your edge a home.'
        }}
      </p>
      <button v-if="!devices.length" class="button primary" @click="emit('enroll')">
        <Plus :size="16" />Add your first device
      </button>
      <button v-else class="button secondary" @click="clearFilters">Clear filters</button>
    </div>
    <div class="table-footer">
      <span>Showing {{ filteredDevices.length }} of {{ devices.length }} devices</span>
      <span><RefreshCw :size="12" /> Auto-refreshes every 5s</span>
    </div>
  </section>
</template>
