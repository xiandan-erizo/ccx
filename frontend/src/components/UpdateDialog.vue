<template>
  <v-dialog
    :model-value="modelValue"
    max-width="520"
    :scrim="true"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <v-card rounded="xl">
      <v-card-title class="d-flex align-center justify-space-between pa-4">
        <div class="d-flex align-center ga-2">
          <v-icon color="primary">mdi-update</v-icon>
          <span>{{ t('update.title') }}</span>
        </div>
        <v-btn icon variant="text" @click="$emit('update:modelValue', false)">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </v-card-title>

      <v-divider />

      <v-card-text class="pa-4">
        <div v-if="loading" class="d-flex flex-column align-center py-6">
          <v-progress-circular indeterminate size="40" color="primary" />
          <p class="text-body-2 mt-3 text-medium-emphasis">{{ t('update.checking') }}</p>
        </div>

        <div v-else-if="status">
          <div class="d-flex justify-space-between align-center mb-3">
            <span class="text-body-2 text-medium-emphasis">{{ t('update.currentVersion') }}</span>
            <v-chip size="small" variant="outlined">{{ status.current_version }}</v-chip>
          </div>

          <div v-if="status.latest_version" class="d-flex justify-space-between align-center mb-4">
            <span class="text-body-2 text-medium-emphasis">{{ t('update.latestVersion') }}</span>
            <v-chip size="small" :color="status.has_update ? 'success' : 'default'" variant="outlined">
              {{ status.latest_version }}
            </v-chip>
          </div>

          <v-alert v-if="status.is_docker" type="info" variant="tonal" rounded="lg" class="mb-4">
            {{ t('update.dockerHint') }}
          </v-alert>

          <v-alert v-else-if="!status.has_update" type="success" variant="tonal" rounded="lg" class="mb-4">
            {{ t('update.upToDate') }}
          </v-alert>

          <v-alert v-else-if="status.has_update && !status.can_update" type="warning" variant="tonal" rounded="lg" class="mb-4">
            {{ status.update_disabled_reason }}
          </v-alert>

          <v-alert v-else-if="status.has_update && status.can_update" type="info" variant="tonal" rounded="lg" class="mb-4">
            {{ t('update.available') }}
          </v-alert>

          <div v-if="status.release_notes && status.has_update" class="mb-4">
            <div class="text-body-2 text-medium-emphasis mb-1">Release Notes</div>
            <v-card variant="outlined" rounded="lg" class="pa-3">
              <pre class="text-body-2 release-notes">{{ status.release_notes }}</pre>
            </v-card>
          </div>

          <v-alert v-if="applySuccess" type="success" variant="tonal" rounded="lg" class="mb-4">
            {{ t('update.restarting') }}
          </v-alert>

          <v-alert v-if="restartTimeout" type="warning" variant="tonal" rounded="lg" class="mb-4">
            {{ t('update.restartTimeout') }}
          </v-alert>
        </div>

        <v-alert v-else type="error" variant="tonal" rounded="lg">
          {{ t('update.checkFailed') }}
        </v-alert>
      </v-card-text>

      <v-divider />

      <v-card-actions class="pa-4">
        <v-btn
          variant="outlined"
          :loading="loading"
          @click="handleCheck"
        >
          {{ t('update.checkBtn') }}
        </v-btn>
        <v-spacer />
        <v-btn
          v-if="status?.can_update && !applySuccess"
          color="primary"
          variant="flat"
          :loading="systemStore.isUpdating"
          @click="handleApply"
        >
          {{ t('update.applyBtn') }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useSystemStore } from '@/stores/system'
import { useI18n } from '@/i18n'
import api from '@/services/api'
import { fetchHealth } from '@/services/api'
import type { UpdateStatusResponse } from '@/services/api'

const { t } = useI18n()

defineProps<{ modelValue: boolean }>()
defineEmits<{ 'update:modelValue': [value: boolean] }>()

const systemStore = useSystemStore() as any
const loading = ref(false)
const status = ref<UpdateStatusResponse | null>(null)
const applySuccess = ref(false)
const restartTimeout = ref(false)

onMounted(() => {
  if (systemStore.updateStatus) {
    status.value = systemStore.updateStatus
    systemStore.setIsUpdating(systemStore.isUpdating || systemStore.updateStatus.is_updating)
  } else if (!systemStore.isUpdating) {
    handleCheck()
  }
})

async function handleCheck() {
  loading.value = true
  try {
    const result = await api.checkUpdate()
    status.value = result
    systemStore.setUpdateStatus(result)
    systemStore.setIsUpdating(systemStore.isUpdating || result.is_updating)
  } catch {
    if (!systemStore.isUpdating) {
      status.value = null
    }
  } finally {
    loading.value = false
  }
}

async function handleApply() {
  systemStore.setIsUpdating(true)
  applySuccess.value = false
  restartTimeout.value = false
  try {
    await api.applyUpdate()
    applySuccess.value = true
    pollHealth()
  } catch {
    systemStore.setIsUpdating(false)
  }
}

function pollHealth() {
  const start = Date.now()
  const maxWait = 60000
  const interval = 2000
  let observedDowntime = false

  const timer = setInterval(async () => {
    const elapsed = Date.now() - start
    if (elapsed > maxWait) {
      clearInterval(timer)
      restartTimeout.value = true
      systemStore.setIsUpdating(false)
      return
    }
    try {
      await fetchHealth()
      if (observedDowntime) {
        clearInterval(timer)
        systemStore.setIsUpdating(false)
        window.location.reload()
      }
    } catch {
      observedDowntime = true
    }
  }, interval)
}
</script>

<style scoped>
.release-notes {
  white-space: pre-wrap;
  word-break: break-word;
  font-family: inherit;
  margin: 0;
}
</style>
