<script setup lang="ts">
import { computed } from 'vue'
import { Check, Copy, ShieldCheck } from 'lucide-vue-next'
import type { DeviceCredentials } from '../types'

const props = defineProps<{ credentials: DeviceCredentials }>()
const emit = defineEmits<{ copy: [text: string]; close: [] }>()
const connectionHeaders = computed(
  () =>
    `Authorization: Bearer ${props.credentials.token}\nX-Device-ID: ${props.credentials.id}\nContent-Type: application/json`,
)
</script>

<template>
  <div class="modal-body">
    <div class="success-callout">
      <ShieldCheck :size="23" />
      <div>
        <strong>Save this token now</strong>
        <p>It’s only shown once. You can rotate it from device details if needed.</p>
      </div>
    </div>
    <label
      >Device ID
      <div class="copy-field">
        <code>{{ credentials.id }}</code>
        <button
          class="icon-button"
          aria-label="Copy device ID"
          @click="emit('copy', credentials.id)"
        >
          <Copy :size="16" />
        </button>
      </div>
    </label>
    <label
      >Device token
      <div class="copy-field">
        <code>{{ credentials.token }}</code>
        <button
          class="icon-button"
          aria-label="Copy device token"
          @click="emit('copy', credentials.token)"
        >
          <Copy :size="16" />
        </button>
      </div>
    </label>
    <label>Firmware HTTP headers</label>
    <pre>{{ connectionHeaders }}</pre>
    <p class="field-hint">
      Use these headers in your Arduino or ESP32 firmware’s HTTP client. Send heartbeats to your
      server at /api/agent/heartbeat. See the connection guide for the request body and update
      protocol.
    </p>
    <div class="modal-actions">
      <button class="button secondary" @click="emit('copy', connectionHeaders)">
        <Copy :size="16" />Copy headers
      </button>
      <button class="button primary" @click="emit('close')">
        I saved my token<Check :size="16" />
      </button>
    </div>
  </div>
</template>
