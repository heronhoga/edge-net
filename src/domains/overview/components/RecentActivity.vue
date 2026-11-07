<script setup lang="ts">
import { Activity } from 'lucide-vue-next'
import { relative } from '../../../shared/lib/format'
import type { ActivityEvent } from '../types'

defineProps<{ events: ActivityEvent[] }>()
</script>

<template>
  <section class="panel activity-panel">
    <div class="panel-title">
      <h2>Recent activity</h2>
      <span class="subtle-label">LATEST EVENTS</span>
    </div>
    <div v-if="!events.length" class="quiet-empty">
      <Activity :size="23" />
      <p>All quiet for now. Your fleet’s activity will appear here.</p>
    </div>
    <div v-for="(event, index) in events.slice(0, 5)" :key="event.id" class="event">
      <span class="event-icon" :class="{ first: index === 0 }"><Activity :size="15" /></span>
      <div>
        <p>{{ event.message }}</p>
        <small>{{ relative(event.at) }}</small>
      </div>
    </div>
  </section>
</template>
