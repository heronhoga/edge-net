<script setup lang="ts">
import { ref } from 'vue'
import { ArrowRight, Code2, Cpu, LoaderCircle, Network, ShieldCheck } from 'lucide-vue-next'

defineProps<{
  loading: boolean
  initializing: boolean
  error: string
  googleEnabled: boolean
  adminEnabled: boolean
}>()
const emit = defineEmits<{ login: [token: string]; google: []; retry: [] }>()
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
      <div class="login-form">
        <h2>Welcome to Edgenet</h2>
        <p>Sign in to your own workspace for Arduino and ESP32 devices.</p>
        <button
          class="button secondary google-login"
          :disabled="initializing || loading || !googleEnabled"
          @click="emit('google')"
        >
          <LoaderCircle v-if="initializing" class="spin" :size="18" />
          <svg v-else width="18" height="18" viewBox="0 0 48 48" aria-hidden="true">
            <path
              fill="#4285F4"
              d="M43.6 20.5H24v8h11.3c-.5 2.6-2 4.8-4.2 6.2v5.2h6.8c4-3.7 6.3-9.1 6.3-15.5 0-1.3-.1-2.6-.4-3.9z"
            />
            <path
              fill="#34A853"
              d="M24 44c5.7 0 10.5-1.9 14-5.1l-6.8-5.2c-1.9 1.3-4.3 2-7.2 2-5.5 0-10.2-3.7-11.9-8.7H5.2v5.4A21.1 21.1 0 0 0 24 44z"
            />
            <path fill="#FBBC05" d="M12.1 27a12.6 12.6 0 0 1 0-8V13.6H5.2a21 21 0 0 0 0 18.8z" />
            <path
              fill="#EA4335"
              d="M24 10.3c3.1 0 5.9 1.1 8.1 3.2l6-6A20.2 20.2 0 0 0 24 2 21.1 21.1 0 0 0 5.2 13.6l6.9 5.4c1.7-5 6.4-8.7 11.9-8.7z"
            />
          </svg>
          {{ initializing ? 'Loading sign-in…' : 'Continue with Google' }}
        </button>
        <p v-if="!initializing && !googleEnabled" class="auth-help">
          Google sign-in has not been configured on this server yet.
        </p>
        <p v-if="error" class="error-box" role="alert">{{ error }}</p>
        <button v-if="error && !initializing" class="text-button auth-retry" @click="emit('retry')">
          Retry connection
        </button>
        <div class="login-hint">
          <ShieldCheck :size="18" />
          <p>
            Your devices, firmware, and deployments belong to your private workspace. Sign in with
            the same Google account to return to them.
          </p>
        </div>
        <details v-if="adminEnabled" class="admin-login">
          <summary>Local administrator</summary>
          <form @submit.prevent="emit('login', loginToken)">
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
            <button class="button primary login-submit" :disabled="loading || !loginToken.trim()">
              <LoaderCircle v-if="loading" class="spin" :size="17" />Open workspace<ArrowRight
                v-if="!loading"
                :size="17"
              />
            </button>
            <p class="auth-help">For the server owner's existing local workspace.</p>
          </form>
        </details>
      </div>
    </main>
  </div>
</template>
