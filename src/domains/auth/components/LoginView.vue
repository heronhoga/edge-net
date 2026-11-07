<script setup lang="ts">
import { ref } from 'vue'
import { ArrowRight, Code2, Cpu, LoaderCircle, Network, Terminal } from 'lucide-vue-next'

defineProps<{ loading: boolean; error: string }>()
const emit = defineEmits<{ login: [token: string] }>()
const loginToken = ref('')
</script>

<template>
  <div class="login-shell">
    <div class="login-art">
      <div class="brand">
        <span class="brand-symbol"><Network :size="25" /></span>edgenet<span class="brand-dot"
          >.</span
        >
      </div>
      <div class="login-story">
        <h1>Your edge.<br />All connected.</h1>
        <p>One place to monitor your hardware, ship updates, and keep your entire fleet in sync.</p>
        <div class="hardware-row">
          <Cpu /><span>ESP32</span><span class="dot-separator">/</span><Code2 /><span>Arduino</span>
        </div>
      </div>
      <div class="network-art" aria-hidden="true">
        <div class="orbit orbit-one"></div>
        <div class="orbit orbit-two"></div>
        <div class="network-center"><Network :size="48" /></div>
        <div class="network-node node-one"><Cpu /></div>
        <div class="network-node node-two"><Code2 /></div>
        <div class="network-node node-three"><Cpu /></div>
      </div>
    </div>
    <main class="login-panel">
      <form class="login-form" @submit.prevent="emit('login', loginToken)">
        <h2>Welcome to Edgenet</h2>
        <p>Connect to your device management portal.</p>
        <label for="admin-token">Admin access token</label>
        <input
          id="admin-token"
          v-model="loginToken"
          type="password"
          required
          minlength="16"
          autocomplete="current-password"
          placeholder="Enter your ADMIN_TOKEN"
        />
        <p v-if="error" class="error-box" role="alert">{{ error }}</p>
        <button class="button primary login-submit" :disabled="loading || !loginToken.trim()">
          <LoaderCircle v-if="loading" class="spin" :size="17" />Open workspace
          <ArrowRight v-if="!loading" :size="17" />
        </button>
        <div class="login-hint">
          <Terminal :size="18" />
          <p>
            Use the <code>ADMIN_TOKEN</code> configured on your Go server. Your session stays in
            this browser tab.
          </p>
        </div>
      </form>
    </main>
  </div>
</template>
