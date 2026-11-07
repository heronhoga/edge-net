<script setup lang="ts">
import { computed, ref } from 'vue'
import { ArrowUpRight, LoaderCircle } from 'lucide-vue-next'
import StatusBadge from '../../../components/ui/StatusBadge.vue'
import type { Device } from '../../devices/types'
import type { Firmware } from '../../firmware/types'
import type { Deployment, DeploymentRequest } from '../types'
import { isActive } from '../status'

const props = defineProps<{
  devices: Device[]
  firmware: Firmware[]
  deployments: Deployment[]
  defaults: DeploymentRequest
  busy: boolean
}>()
const emit = defineEmits<{ submit: [input: DeploymentRequest]; close: []; upload: [] }>()
const firmwareId = ref(props.defaults.firmwareId)
const deviceIds = ref([...props.defaults.deviceIds])
const chosenFirmware = computed(() =>
  props.firmware.find((artifact) => artifact.id === firmwareId.value),
)
const compatible = computed(() =>
  props.devices.filter((device) => device.target === chosenFirmware.value?.target),
)
const activeDeviceIds = computed(
  () => new Set(props.deployments.filter((job) => isActive(job.status)).map((job) => job.deviceId)),
)
</script>

<template>
  <form
    class="modal-body"
    @submit.prevent="emit('submit', { firmwareId, deviceIds: [...deviceIds] })"
  >
    <p class="form-description">
      Updates are picked up by the device agent. Offline devices will wait in the queue.
    </p>
    <label
      >Firmware artifact<select v-model="firmwareId" required @change="deviceIds = []">
        <option value="" disabled>Select an artifact</option>
        <option v-for="artifact in firmware" :key="artifact.id" :value="artifact.id">
          {{ artifact.version }} · {{ artifact.target }} · {{ artifact.name }}
        </option>
      </select></label
    >
    <div v-if="!firmware.length" class="inline-empty">
      No firmware yet.
      <button type="button" class="text-button" @click="emit('upload')">
        Upload an artifact first
      </button>
    </div>
    <label class="device-selection-label"
      >Compatible devices <span class="optional">{{ deviceIds.length }} selected</span></label
    >
    <div class="device-selection">
      <label v-for="device in compatible" :key="device.id" class="selection-row">
        <input
          v-model="deviceIds"
          type="checkbox"
          :value="device.id"
          :disabled="activeDeviceIds.has(device.id)"
        />
        <span
          ><strong>{{ device.name }}</strong
          ><small>{{
            activeDeviceIds.has(device.id)
              ? 'Already has an active deployment'
              : `${device.target} · ${device.version}`
          }}</small></span
        >
        <StatusBadge :status="device.status" />
      </label>
      <p v-if="!compatible.length" class="inline-empty">
        {{
          firmwareId
            ? 'No devices match this hardware target.'
            : 'Select an artifact to see compatible devices.'
        }}
      </p>
    </div>
    <div class="modal-actions">
      <button type="button" class="button secondary" :disabled="busy" @click="emit('close')">
        Cancel
      </button>
      <button class="button primary" :disabled="busy || !firmwareId || !deviceIds.length">
        <LoaderCircle v-if="busy" class="spin" :size="16" /><ArrowUpRight v-else :size="16" />Deploy
        to {{ deviceIds.length }} device{{ deviceIds.length === 1 ? '' : 's' }}
      </button>
    </div>
  </form>
</template>
