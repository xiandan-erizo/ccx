<template>
  <div class="conversation-dashboard">
    <!-- 过滤栏 -->
    <div class="d-flex align-center mb-4 flex-wrap ga-2">
      <!-- 手机端：下拉选择 -->
      <v-select
        v-if="xs"
        v-model="kindFilter"
        :items="kindFilterOptions"
        density="compact"
        variant="outlined"
        hide-details
        class="kind-filter-select"
      />
      <!-- 桌面端：chip 组 -->
      <v-chip-group v-else v-model="kindFilter" mandatory selected-class="text-primary">
        <v-chip value="" variant="outlined" size="small" class="filter-chip" filter>ALL</v-chip>
        <v-chip value="messages" variant="outlined" size="small" color="purple" class="filter-chip" filter>MESSAGES</v-chip>
        <v-chip value="chat" variant="outlined" size="small" color="blue" class="filter-chip" filter>CHAT</v-chip>
        <v-chip value="images" variant="outlined" size="small" color="pink" class="filter-chip" filter>IMAGES</v-chip>
        <v-chip value="responses" variant="outlined" size="small" color="teal" class="filter-chip" filter>RESPONSES</v-chip>
        <v-chip value="gemini" variant="outlined" size="small" color="orange" class="filter-chip" filter>GEMINI</v-chip>
      </v-chip-group>
      <v-spacer />
      <v-text-field
        v-model="searchQuery"
        density="compact"
        variant="outlined"
        hide-details
        clearable
        prepend-inner-icon="mdi-magnify"
        :placeholder="t('cockpit.searchPlaceholder')"
        class="conversation-search-field"
      />
      <span class="system-status-indicator" :class="'status-' + systemStore.systemStatus">
        <span class="status-dot"></span>
        {{ systemStatusText }}
      </span>
      <span class="text-caption text-medium-emphasis">
        Active: {{ visibleConversations.length }}
        <span v-if="overrideCount > 0" class="ml-2 text-warning">Override: {{ overrideCount }}</span>
      </span>
    </div>

    <!-- Loading -->
    <div v-if="loading && !conversations.length" class="d-flex justify-center py-12">
      <v-progress-circular indeterminate color="primary" />
    </div>

    <!-- Empty (no conversations at all) -->
    <v-card v-else-if="!conversations.length" variant="outlined" class="text-center pa-12">
      <v-icon size="48" color="grey">mdi-chat-outline</v-icon>
      <div class="text-body-1 mt-4 text-medium-emphasis">
        {{ t('cockpit.empty') }}
      </div>
    </v-card>

    <!-- Conversation cards -->
    <template v-else>
      <v-card v-if="!visibleConversations.length" variant="outlined" class="text-center pa-8 mb-4">
        <div class="text-body-2 text-medium-emphasis">
          {{ t('cockpit.noMatches') }}
        </div>
      </v-card>
      <div class="conversation-masonry">
        <div
          v-for="conv in visibleConversations"
          :key="conv.id"
          class="conversation-masonry-item"
        >
          <ConversationCard
            :conversation="conv"
            :override="overrides[conv.id]"
            :available-channels="getChannelsForKind(conv.kind)"
            :expanded="expandedCards.has(conv.id)"
            :now-ms="nowMs"
            @toggle-expand="toggleExpand(conv.id)"
            @set-override="handleSetOverride"
            @remove-override="handleRemoveOverride"
            @success="(msg: string) => emit('success', msg)"
            @error="(msg: string) => emit('error', msg)"
          />
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useDisplay } from 'vuetify'
import { api, type ConversationInfo, type SequenceOverrideInfo, type ChannelSequenceEntry } from '@/services/api'
import { useGlobalTick } from '@/composables/useGlobalTick'
import { useI18n } from '@/i18n'
import { useSystemStore } from '@/stores/system'
import ConversationCard from './ConversationCard.vue'

const { t } = useI18n()
const { xs } = useDisplay()
const systemStore = useSystemStore() as any

const emit = defineEmits<{
  success: [message: string]
  error: [message: string]
}>()

const systemStatusText = computed(() => {
  switch (systemStore.systemStatus) {
    case 'running': return t('system.running')
    case 'error': return t('system.error')
    case 'connecting': return t('system.connecting')
    default: return t('system.unknown')
  }
})

const loading = ref(true)
const conversations = ref<ConversationInfo[]>([])
const overrides = ref<Record<string, SequenceOverrideInfo>>({})
const kindFilter = ref('')
const searchQuery = ref('')
const nowMs = ref(Date.now())

const kindFilterOptions = [
  { title: 'ALL', value: '' },
  { title: 'MESSAGES', value: 'messages' },
  { title: 'CHAT', value: 'chat' },
  { title: 'IMAGES', value: 'images' },
  { title: 'RESPONSES', value: 'responses' },
  { title: 'GEMINI', value: 'gemini' },
]
const expandedCards = ref(new Set<string>())
type DashboardChannel = { index: number; name: string; priority: number; status: string }

const channelsByKind = ref<Record<string, DashboardChannel[]>>({})

function normalizeChannel(ch: any): DashboardChannel {
  const index = ch.index ?? ch.Index ?? 0
  return {
    index,
    name: ch.name ?? ch.Name ?? `Channel ${index}`,
    priority: ch.priority ?? ch.Priority ?? index,
    status: ch.status ?? ch.Status ?? 'active',
  }
}

function normalizeChannelsByKind(value: Record<string, any[]>): Record<string, DashboardChannel[]> {
  return Object.fromEntries(
    Object.entries(value).map(([kind, channels]) => [
      kind,
      (channels || [])
        .map(normalizeChannel)
        .sort((a, b) => (a.priority - b.priority) || (a.index - b.index)),
    ]),
  )
}

const sortedConversations = computed(() => {
  return [...conversations.value].sort((a, b) => new Date(b.lastActiveAt).getTime() - new Date(a.lastActiveAt).getTime())
})

const visibleConversations = computed(() => {
  let list = sortedConversations.value
  const kind = kindFilter.value
  if (kind) list = list.filter(c => c.kind === kind)
  const q = (searchQuery.value || '').trim().toLowerCase()
  if (q) {
    list = list.filter(c =>
      (c.title || '').toLowerCase().includes(q) ||
      (c.userId || '').toLowerCase().includes(q) ||
      (c.rawUserId || '').toLowerCase().includes(q) ||
      (c.lastModel || '').toLowerCase().includes(q) ||
      (c.channelName || '').toLowerCase().includes(q),
    )
  }
  return list
})

const overrideCount = computed(() => Object.keys(overrides.value).length)

function getChannelsForKind(kind: string): DashboardChannel[] {
  return channelsByKind.value[kind] || []
}

async function fetchAllChannels() {
  const kinds = ['messages', 'chat', 'responses', 'gemini', 'images'] as const
  for (const kind of kinds) {
    try {
      const dashboard = await api.getChannelDashboard(kind)
      if (!channelsByKind.value[kind]?.length) {
        channelsByKind.value[kind] = (dashboard.channels || [])
          .map(normalizeChannel)
          .sort((a, b) => (a.priority - b.priority) || (a.index - b.index))
      }
    } catch (e) {
      console.error(`[ConversationDashboard] fetch ${kind} channels error:`, e)
    }
  }
}

async function fetchConversations() {
  try {
    const resp = await api.getConversations(undefined)
    conversations.value = resp.conversations || []
    overrides.value = resp.overrides || {}
    if (resp.channelsByKind) {
      channelsByKind.value = normalizeChannelsByKind(resp.channelsByKind)
    }
  } catch (e) {
    console.error('[ConversationDashboard] fetch error:', e)
  } finally {
    loading.value = false
  }
}

function toggleExpand(id: string) {
  const next = new Set(expandedCards.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  expandedCards.value = next
}

async function handleSetOverride(convId: string, sequence: ChannelSequenceEntry[]) {
  try {
    await api.setConversationOverride(convId, sequence)
    await fetchConversations()
  } catch (e) {
    console.error('[ConversationDashboard] set override error:', e)
    emit('error', e instanceof Error ? e.message : 'Override failed')
  }
}

async function handleRemoveOverride(convId: string) {
  try {
    await api.removeConversationOverride(convId)
    await fetchConversations()
  } catch (e) {
    console.error('[ConversationDashboard] remove override error:', e)
    emit('error', e instanceof Error ? e.message : 'Remove override failed')
  }
}

// Polling (3s for data, 1s for clock)
const tick = useGlobalTick(3000, 'ConversationDashboard')
tick.onTick(() => fetchConversations())
const clockTick = useGlobalTick(1000, 'ConversationDashboardClock')
clockTick.onTick(() => { nowMs.value = Date.now() })
fetchConversations()
fetchAllChannels()
</script>

<style scoped>
.conversation-dashboard {
  max-width: 1400px;
  margin: 0 auto;
}
.filter-chip {
  border-radius: 0 !important;
  font-size: 10px !important;
  font-weight: 700;
  letter-spacing: 0.06em;
}
.kind-filter-select {
  max-width: 160px;
  flex: 0 0 auto;
}
.conversation-search-field {
  max-width: 320px;
  min-width: 180px;
  flex: 1 1 auto;
}
.conversation-masonry {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(420px, 100%), 1fr));
  gap: 16px;
  align-items: start;
  box-sizing: border-box;
  padding-right: 6px;
  padding-bottom: 6px;
}
.system-status-indicator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  padding: 4px 10px;
  margin-right: 12px;
  border: 1px solid rgb(var(--v-theme-on-surface));
  background: rgb(var(--v-theme-surface));
}
.system-status-indicator .status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #9ca3af;
}
.system-status-indicator.status-running .status-dot {
  background: #10b981;
  animation: dot-pulse 2s ease-in-out infinite;
}
.system-status-indicator.status-error .status-dot {
  background: #ef4444;
}
.system-status-indicator.status-connecting .status-dot {
  background: #f59e0b;
}
@keyframes dot-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
</style>