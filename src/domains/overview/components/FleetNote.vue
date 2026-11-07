<script setup lang="ts">
import { ArrowRight, Code2, Cpu, Network } from 'lucide-vue-next'
const emit = defineEmits<{ guide: [] }>()
</script>

<template>
  <section class="fleet-note">
    <span class="mini-label"><Network :size="16" /> BUILT FOR MICROCONTROLLERS</span>
    <h2>Different devices.<br />One connected fleet.</h2>
    <p>
      Arduino and ESP32. Monitor your boards together and deploy firmware built for each target.
    </p>
    <div class="supported-icons">
      <span><Cpu :size="19" />ESP32</span><span><Code2 :size="19" />Arduino</span>
    </div>
    <button @click="emit('guide')">Explore the connection guide <ArrowRight :size="17" /></button>
  </section>
</template>
