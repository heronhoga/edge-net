<script setup lang="ts">
import { ArrowRight, ChevronRight, Cpu, ShieldCheck } from 'lucide-vue-next'
const emit = defineEmits<{ enroll: [] }>()
</script>

<template>
  <div class="guide-grid">
    <section class="panel guide-main">
      <div class="panel-title">
        <h2>From hardware to dashboard</h2>
        <span class="badge online">HTTP protocol</span>
      </div>
      <div class="guide-step">
        <span>01</span>
        <div>
          <h3>Enroll your device</h3>
          <p>
            Add a name, device type, and hardware target. Copy the generated device ID and token;
            the token is shown only once.
          </p>
          <button class="text-button" @click="emit('enroll')">
            Add a device <ArrowRight :size="15" />
          </button>
        </div>
      </div>
      <div class="guide-step">
        <span>02</span>
        <div>
          <h3>Send a heartbeat every 30 seconds</h3>
          <p>
            Use the server’s reachable address. Authenticate with your device token and ID. A device
            goes offline after 90 seconds without a heartbeat.
          </p>
          <pre>
POST /api/agent/heartbeat
Authorization: Bearer YOUR_DEVICE_TOKEN
X-Device-ID: YOUR_DEVICE_ID
Content-Type: application/json

{
  "version": "1.0.0",
  "metrics": {
    "cpu": 12.5,
    "memory": 38,
    "temperature": 42,
    "uptime": 3600
  }
}</pre>
          <p>
            Only report metrics your hardware supports. CPU, memory, and temperature are optional.
          </p>
        </div>
      </div>
      <div class="guide-step">
        <span>03</span>
        <div>
          <h3>Poll, verify, and apply updates</h3>
          <p>
            Poll <code>GET /api/agent/jobs</code>. Download the assigned artifact with the same
            credentials, verify its SHA-256 and hardware target, then report each installation
            state.
          </p>
          <div class="flow">
            <span>Queued</span><ChevronRight :size="13" /><span>Downloading</span
            ><ChevronRight :size="13" /><span>Installing</span><ChevronRight :size="13" /><span
              >Success</span
            >
          </div>
          <p>
            Report to <code>POST /api/agent/jobs/:id/status</code> with
            <code>{"status":"downloading"}</code>. Use <code>failed</code> with a message when an
            update cannot complete.
          </p>
        </div>
      </div>
    </section>
    <div class="guide-side">
      <section class="panel guide-card">
        <Cpu :size="25" />
        <h3>ESP32 & Arduino</h3>
        <p>
          Implement the HTTP protocol in your sketch. ESP32 can use an OTA partition and the Update
          library. Arduino support depends on the board and bootloader.
        </p>
        <p>
          USB-only boards need a network-connected gateway to report telemetry and perform flashing.
        </p>
      </section>
      <div class="guide-tip">
        <ShieldCheck :size="20" />
        <p>Use HTTPS when connecting devices outside your local development network.</p>
      </div>
    </div>
  </div>
</template>
