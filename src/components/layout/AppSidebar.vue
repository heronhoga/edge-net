<script setup lang="ts">
import { ArrowUpRight, ChevronRight, CircleHelp, LogOut, Network, Radio } from 'lucide-vue-next'
import { navigation } from '../../app/navigation'
import type { Page } from '../../app/types'

defineProps<{ page: Page; deviceCount: number }>()
const emit = defineEmits<{ navigate: [page: Page]; logout: [] }>()
</script>

<template>
  <aside class="sidebar">
    <a class="brand" href="#" @click.prevent="emit('navigate', 'overview')">
      <span class="brand-symbol"><Network :size="24" /></span>
      <span class="brand-word">edgenet<span class="brand-dot">.</span></span>
    </a>
    <div class="workspace">
      <div class="workspace-avatar">E</div>
      <div><strong>My workspace</strong><span>Edge infrastructure</span></div>
      <ChevronRight :size="15" />
    </div>
    <span class="nav-label">WORKSPACE</span>
    <nav aria-label="Main navigation">
      <button
        v-for="item in navigation"
        :key="item.id"
        :class="['nav-item', { active: page === item.id }]"
        :title="item.label"
        @click="emit('navigate', item.id)"
      >
        <component :is="item.icon" :size="19" /><span>{{ item.label }}</span>
        <span v-if="item.id === 'devices'" class="nav-count">{{ deviceCount }}</span>
      </button>
    </nav>
    <div class="sidebar-bottom">
      <div class="connect-card">
        <span class="connect-icon"><Radio :size="20" /></span>
        <strong>Bring your edge online</strong>
        <p>From your first board to your next hundred devices.</p>
        <button @click="emit('navigate', 'connect')">
          Connection guide <ArrowUpRight :size="15" />
        </button>
      </div>
      <button
        :class="['nav-item', { active: page === 'connect' }]"
        title="Connection guide"
        @click="emit('navigate', 'connect')"
      >
        <CircleHelp :size="19" /><span>Connection guide</span>
      </button>
      <div class="profile">
        <span class="profile-avatar">AD</span>
        <div><strong>Administrator</strong><span>Workspace owner</span></div>
        <button class="icon-button" aria-label="Sign out" @click="emit('logout')">
          <LogOut :size="17" />
        </button>
      </div>
    </div>
  </aside>
</template>
