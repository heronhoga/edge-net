<script setup lang="ts">
import { ChevronRight, RefreshCw } from 'lucide-vue-next'
import AppSidebar from './AppSidebar.vue'
import { pageDescriptions, pageTitle } from '../../app/navigation'
import type { Page } from '../../app/types'
import { initials, type User } from '../../domains/auth/types'

defineProps<{ page: Page; deviceCount: number; loading: boolean; error: string; user: User }>()
const emit = defineEmits<{ navigate: [page: Page]; logout: []; refresh: [] }>()
</script>

<template>
  <div class="app-shell">
    <AppSidebar
      :page="page"
      :device-count="deviceCount"
      :user="user"
      @navigate="emit('navigate', $event)"
      @logout="emit('logout')"
    />
    <div class="main-shell">
      <header class="topbar">
        <div class="breadcrumb">
          Workspace <ChevronRight :size="14" /><strong>{{ pageTitle(page) }}</strong>
        </div>
        <div class="topbar-right">
          <span class="live-indicator" :class="{ disconnected: error }">
            <i></i>{{ error ? 'Connection issue' : 'Live updates' }}
          </span>
          <button
            class="icon-button"
            :disabled="loading"
            aria-label="Refresh dashboard"
            @click="emit('refresh')"
          >
            <RefreshCw :size="17" :class="{ spin: loading }" />
          </button>
          <span class="topbar-avatar" :title="user.name">{{ initials(user) }}</span>
        </div>
      </header>
      <main class="content">
        <div v-if="error" class="error-box global-error" role="alert">
          {{ error }} <button class="text-button" @click="emit('refresh')">Retry</button>
        </div>
        <section class="page-heading">
          <div>
            <div class="eyebrow">YOUR EDGE, AT A GLANCE</div>
            <h1>{{ page === 'overview' ? 'Fleet overview' : pageTitle(page) }}</h1>
            <p>{{ pageDescriptions[page] }}</p>
          </div>
          <slot name="actions" />
        </section>
        <slot />
        <footer class="page-footer">
          <span><span class="tiny-dot"></span> Edgenet · Your edge, connected.</span>
        </footer>
      </main>
    </div>
  </div>
</template>
