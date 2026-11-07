<script setup lang="ts">
defineProps<{ status: string }>()
</script>

<template>
  <span :class="['badge', status]"><i></i>{{ status }}</span>
</template>
