<template>
  <ConversationDashboard @success="(msg: string) => emit('success', msg)" @error="(msg: string) => emit('error', msg)" />
</template>

<script setup lang="ts">
import ConversationDashboard from '@/components/ConversationDashboard.vue'

const emit = defineEmits<{
  success: [message: string]
  error: [message: string]
}>()
</script>
