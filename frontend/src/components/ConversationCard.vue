<template>
  <v-card
    :class="['conversation-card', { 'override-active': hasOverride }]"
    :style="{ '--ccx-kind-color': kindCssColor }"
    elevation="0"
    role="button"
    tabindex="0"
    :aria-expanded="expanded"
    @click="$emit('toggleExpand')"
    @keydown.enter.prevent="$emit('toggleExpand')"
    @keydown.space.prevent="$emit('toggleExpand')"
  >
    <v-card-text class="pa-4">
      <!-- Row 1: LED + Kind + Title/User + Stats -->
      <div class="d-flex align-center ga-2 mb-3">
        <span :class="['status-led', `status-led--${conversation.status}`]"></span>
        <v-chip class="kind-chip" :color="kindColor" size="x-small" variant="outlined">{{ kindLabel }}</v-chip>
        <span class="display-label text-caption font-weight-mono text-medium-emphasis">
          <v-tooltip :text="tooltipText" location="top" :open-delay="150" content-class="ccx-tooltip">
            <template #activator="{ props: tp }">
              <span v-bind="tp" class="display-label-text">{{ displayLabel }}</span>
            </template>
          </v-tooltip>
        </span>
        <span class="text-caption text-medium-emphasis flex-shrink-0">{{ conversation.requestCount }}x</span>
        <span class="text-caption text-medium-emphasis flex-shrink-0">{{ duration }}</span>
      </div>

      <!-- Row 2: Model + Channel chips (collapsed) -->
      <div v-if="!expanded" class="d-flex align-center ga-2 flex-wrap">
        <span class="text-body-2 font-weight-medium mr-2">{{ conversation.lastModel }}</span>
        <v-chip
          v-for="ch in visibleChannels"
          :key="ch.index"
          :title="getChannelTooltip(ch)"
          :class="{
            'current-channel-chip': ch.index === conversation.currentChannel && !hasOverride,
            'next-channel-chip': ch.index === nextChannel,
          }"
          :color="ch.index === conversation.currentChannel ? 'primary' : ch.index === nextChannel ? 'success' : undefined"
          :variant="ch.index === conversation.currentChannel ? 'flat' : ch.index === nextChannel ? 'flat' : 'outlined'"
          size="x-small"
          @click.stop="handleQuickOverride(ch)"
        >
          {{ ch.name }}
          <template v-if="ch.index === conversation.currentChannel" #append>
            <v-icon size="10">mdi-check</v-icon>
          </template>
          <template v-else-if="ch.index === nextChannel" #append>
            <span class="next-label">| NEXT</span>
          </template>
        </v-chip>
        <v-chip v-if="hiddenCount > 0" size="x-small" variant="text" @click.stop="$emit('toggleExpand')">+{{ hiddenCount }}</v-chip>
      </div>

      <!-- Expanded: Override alert -->
      <v-alert v-if="expanded && hasOverride" type="warning" density="compact" variant="tonal" class="override-alert mb-2 mt-2">
        <div class="d-flex align-center">
          <span class="alert-bang">[!]</span>
          <span class="text-caption">{{ t('cockpit.overrideActive', { time: remainingTime }) }}</span>
          <v-spacer />
          <v-btn size="x-small" variant="text" @click.stop="$emit('removeOverride', conversation.id)">{{ t('cockpit.restoreDefault') }}</v-btn>
        </div>
      </v-alert>

      <!-- Expanded: Full channel sequence -->
      <div v-if="expanded" class="mt-3">
        <div class="text-caption text-medium-emphasis mb-1">{{ conversation.lastModel }}</div>
        <div class="channel-sequence">
          <div
            v-for="(ch, i) in channelSequence"
            :key="ch.index"
            :class="['channel-item d-flex align-center pa-1', { 'demoted': isDemoted(i) }]"
            :style="{ animationDelay: `${Math.min(i, 12) * 35}ms` }"
            class="channel-item-animated"
          >
            <span class="seq-num">{{ String(i + 1).padStart(2, '0') }}</span>
            <span class="seq-arrow">&rarr;</span>
            <span class="text-caption flex-grow-1 channel-name" @click.stop="handleMoveToTop(ch, i)">{{ ch.name }}</span>
            <v-chip v-if="ch.index === conversation.currentChannel" size="x-small" color="primary" variant="flat" class="mr-1">CURRENT</v-chip>
            <v-chip v-else-if="ch.index === nextChannel" size="x-small" color="success" variant="flat" class="mr-1">NEXT</v-chip>
            <v-chip v-if="ch.status === 'suspended'" size="x-small" variant="flat" class="fused-chip mr-1">FUSED</v-chip>
            <v-btn icon size="x-small" variant="text" :disabled="i === channelSequence.length - 1" @click.stop="handleDemote(i)">
              <v-icon size="14">mdi-arrow-down</v-icon>
            </v-btn>
          </div>
        </div>
        <div class="text-right mt-1">
          <v-btn size="x-small" variant="text" @click.stop="$emit('toggleExpand')">Collapse</v-btn>
        </div>
      </div>

      <!-- Row 3: Raw User ID -->
      <div v-if="conversation.rawUserId" class="raw-user-id mt-2 d-flex align-center">
        <span class="text-caption text-medium-emphasis font-weight-mono raw-user-id-text" @click.stop="copyRawUserId">{{ conversation.rawUserId }}</span>
        <v-btn icon size="x-small" variant="text" class="copy-btn" aria-label="Copy user ID" @click.stop="copyRawUserId">
          <v-icon size="12">mdi-content-copy</v-icon>
        </v-btn>
      </div>
    </v-card-text>
  </v-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ConversationInfo, SequenceOverrideInfo, ChannelSequenceEntry } from '@/services/api'
import { useI18n } from '@/i18n'

const { t } = useI18n()

interface ChannelInfo {
  index: number
  name: string
  status: string
}

const props = defineProps<{
  conversation: ConversationInfo
  override?: SequenceOverrideInfo
  availableChannels: ChannelInfo[]
  expanded: boolean
  nowMs: number
}>()

const emit = defineEmits<{
  toggleExpand: []
  setOverride: [convId: string, sequence: ChannelSequenceEntry[]]
  removeOverride: [convId: string]
  success: [message: string]
  error: [message: string]
}>()

const MAX_VISIBLE = 6

const conversation = computed(() => props.conversation)
const hasOverride = computed(() => !!props.override)
const kindLabel = computed(() => `[ ${props.conversation.kind.toUpperCase()} ]`)

const kindColor = computed(() => {
  switch (props.conversation.kind) {
    case 'messages': return 'purple'
    case 'chat': return 'blue'
    case 'responses': return 'teal'
    case 'gemini': return 'orange'
    case 'images': return 'pink'
    default: return 'grey'
  }
})

const kindCssColor = computed(() => {
  const map: Record<string, string> = {
    messages: 'var(--ccx-kind-messages)',
    chat: 'var(--ccx-kind-chat)',
    responses: 'var(--ccx-kind-responses)',
    gemini: 'var(--ccx-kind-gemini)',
    images: 'var(--ccx-kind-images)',
  }
  return map[props.conversation.kind] ?? 'rgb(var(--v-theme-on-surface))'
})

const displayLabel = computed(() => props.conversation.title || props.conversation.userId)

const tooltipText = computed(() => {
  if (props.conversation.title) return props.conversation.title
  return props.conversation.userId
})

const duration = computed(() => {
  const start = new Date(props.conversation.createdAt).getTime()
  const mins = Math.floor((props.nowMs - start) / 60000)
  if (mins < 1) return '<1m'
  if (mins < 60) return `${mins}m`
  return `${Math.floor(mins / 60)}h${mins % 60}m`
})

const remainingTime = computed(() => {
  if (!props.override) return ''
  const expires = new Date(props.override.expiresAt).getTime()
  const remaining = Math.max(0, expires - props.nowMs)
  const mins = Math.floor(remaining / 60000)
  const secs = Math.floor((remaining % 60000) / 1000)
  return `${mins}:${secs.toString().padStart(2, '0')}`
})

const fallbackChannels = computed((): ChannelInfo[] => {
  const channels: ChannelInfo[] = []
  const pushUnique = (channel: ChannelInfo) => {
    if (!channels.some(ch => ch.index === channel.index)) channels.push(channel)
  }
  if (props.override?.sequence) {
    for (const entry of props.override.sequence) {
      pushUnique({ index: entry.channelIndex, name: entry.channelName || `Channel ${entry.channelIndex}`, status: 'active' })
    }
  }
  pushUnique({ index: props.conversation.currentChannel, name: props.conversation.channelName || `Channel ${props.conversation.currentChannel}`, status: 'active' })
  return channels
})

const channelSequence = computed((): ChannelInfo[] => {
  if (props.override?.sequence) {
    return props.override.sequence.map(entry => {
      const ch = props.availableChannels.find(c => c.index === entry.channelIndex)
      return { index: entry.channelIndex, name: entry.channelName || ch?.name || `Channel ${entry.channelIndex}`, status: ch?.status || 'active' }
    })
  }
  const channels = props.availableChannels.filter(ch => ch.status !== 'disabled')
  return channels.length > 0 ? channels : fallbackChannels.value
})

const currentChannelInfo = computed(() => {
  const existing = channelSequence.value.find(ch => ch.index === props.conversation.currentChannel)
    ?? props.availableChannels.find(ch => ch.index === props.conversation.currentChannel)
  if (existing) return existing
  return { index: props.conversation.currentChannel, name: props.conversation.channelName || `Channel ${props.conversation.currentChannel}`, status: 'active' }
})

const nextChannel = computed(() => {
  const candidate = props.override?.sequence?.[0]?.channelIndex
  return candidate !== undefined && candidate !== props.conversation.currentChannel ? candidate : undefined
})

const nextChannelInfo = computed(() => {
  if (nextChannel.value === undefined) return undefined
  const existing = channelSequence.value.find(ch => ch.index === nextChannel.value)
    ?? props.availableChannels.find(ch => ch.index === nextChannel.value)
  if (existing) return existing
  const entry = props.override?.sequence?.[0]
  return { index: nextChannel.value!, name: entry?.channelName || `Channel ${nextChannel.value}`, status: 'active' }
})

const visibleChannels = computed(() => {
  const result: ChannelInfo[] = []
  const required = [currentChannelInfo.value, nextChannelInfo.value].filter((ch): ch is ChannelInfo => !!ch)
  const requiredIndexes = new Set(required.map(ch => ch.index))
  const pushUnique = (channel?: ChannelInfo) => {
    if (!channel || result.some(ch => ch.index === channel.index)) return
    result.push(channel)
  }
  for (const ch of channelSequence.value) {
    if (result.length >= MAX_VISIBLE) break
    pushUnique(ch)
  }
  for (const channel of required) {
    if (result.some(ch => ch.index === channel.index)) continue
    if (result.length >= MAX_VISIBLE) {
      let removeIndex = result.length - 1
      for (let i = result.length - 1; i >= 0; i--) {
        if (!requiredIndexes.has(result[i].index)) { removeIndex = i; break }
      }
      result.splice(removeIndex, 1)
    }
    pushUnique(channel)
  }
  return result
})

const hiddenCount = computed(() => Math.max(0, channelSequence.value.length - visibleChannels.value.length))

function isDemoted(index: number): boolean {
  if (!props.override) return false
  return index >= channelSequence.value.length - 1
}

function buildSequence(channels: ChannelInfo[]): ChannelSequenceEntry[] {
  return channels.map(ch => ({ channelIndex: ch.index, channelName: ch.name }))
}

function getChannelTooltip(ch: ChannelInfo): string {
  if (ch.index === props.conversation.currentChannel && !hasOverride.value) return 'Current channel'
  if (ch.index === nextChannel.value) return 'Next override target'
  return 'Click to set as next'
}

function handleQuickOverride(ch: ChannelInfo) {
  if (!hasOverride.value && ch.index === props.conversation.currentChannel) return
  const rest = channelSequence.value.filter(c => c.index !== ch.index)
  emit('setOverride', props.conversation.id, buildSequence([ch, ...rest]))
}

function handleMoveToTop(ch: ChannelInfo, currentIdx: number) {
  if (currentIdx === 0) return
  const current = [...channelSequence.value]
  const [item] = current.splice(currentIdx, 1)
  current.unshift(item)
  emit('setOverride', props.conversation.id, buildSequence(current))
}

function handleDemote(index: number) {
  const current = [...channelSequence.value]
  if (index >= current.length - 1) return
  const [item] = current.splice(index, 1)
  current.push(item)
  emit('setOverride', props.conversation.id, buildSequence(current))
}



async function copyRawUserId() {
  if (!props.conversation.rawUserId) return
  try {
    await navigator.clipboard.writeText(props.conversation.rawUserId)
    emit('success', t('cockpit.rawUserIdCopied'))
  } catch {
    emit('error', t('cockpit.rawUserIdCopyFailed'))
  }
}

</script>

<style scoped>
.conversation-card {
  cursor: pointer;
  position: relative;
  transition: all 0.1s ease;
  border: 2px solid rgb(var(--v-theme-on-surface));
  box-shadow: 4px 4px 0 0 rgb(var(--v-theme-on-surface));
  background:
    radial-gradient(circle, var(--ccx-dot-grid-color) 1px, transparent 1px) 0 0 / var(--ccx-dot-grid-size) var(--ccx-dot-grid-size),
    rgb(var(--v-theme-surface));
}
.conversation-card::before {
  content: '';
  position: absolute;
  top: 6px; left: 6px;
  width: var(--ccx-hud-corner-size);
  height: var(--ccx-hud-corner-size);
  border-top: var(--ccx-hud-corner-width) solid var(--ccx-hud-corner-color);
  border-left: var(--ccx-hud-corner-width) solid var(--ccx-hud-corner-color);
  pointer-events: none;
  z-index: 1;
  opacity: 0.4;
}
.conversation-card::after {
  content: '';
  position: absolute;
  bottom: 6px; right: 6px;
  width: var(--ccx-hud-corner-size);
  height: var(--ccx-hud-corner-size);
  border-bottom: var(--ccx-hud-corner-width) solid var(--ccx-hud-corner-color);
  border-right: var(--ccx-hud-corner-width) solid var(--ccx-hud-corner-color);
  pointer-events: none;
  z-index: 1;
  opacity: 0.4;
}
.conversation-card:hover {
  transform: translate(-1px, -1px);
  border-color: var(--ccx-kind-color);
  box-shadow: 5px 5px 0 0 var(--ccx-kind-color);
}
.conversation-card:active {
  transform: translate(1px, 1px);
  box-shadow: 2px 2px 0 0 rgb(var(--v-theme-on-surface));
}
.conversation-card.override-active {
  border-color: rgb(var(--v-theme-warning));
  box-shadow: 4px 4px 0 0 rgb(var(--v-theme-warning));
}
.conversation-card.override-active:hover {
  border-color: rgb(var(--v-theme-warning));
  box-shadow: 5px 5px 0 0 rgb(var(--v-theme-warning));
}
.v-theme--dark .conversation-card {
  border-color: rgba(255, 255, 255, 0.8);
  box-shadow: 4px 4px 0 0 rgba(255, 255, 255, 0.8);
}
.v-theme--dark .conversation-card:hover {
  border-color: var(--ccx-kind-color);
  box-shadow: 5px 5px 0 0 var(--ccx-kind-color);
}
.v-theme--dark .conversation-card:active {
  box-shadow: 2px 2px 0 0 rgba(255, 255, 255, 0.8);
}

/* Status LED */
.status-led {
  display: inline-block;
  width: 8px; height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.status-led--streaming {
  background: var(--ccx-led-streaming-color);
  animation: ccx-led-pulse 1.4s ease-in-out infinite;
}
.status-led--active {
  background: var(--ccx-status-breaker-half-open-dot-bg);
  box-shadow: 0 0 4px 1px var(--ccx-status-breaker-half-open-dot-glow);
}
.status-led--idle {
  background: var(--ccx-status-disabled-dot-bg);
}

/* Kind chip */
.kind-chip {
  border-radius: 0 !important;
  font-size: 9px !important;
  font-weight: 700;
  letter-spacing: 0.08em;
}



/* Display label (title/userId) */
.display-label {
  min-width: 0;
  flex: 1;
}
.display-label-text {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Next label */
.next-label {
  display: inline-block;
  margin-left: 6px;
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.05em;
}

/* Channel sequence (expanded) */
.channel-sequence {
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 0;
  overflow: hidden;
}
.channel-item {
  border-bottom: 1px solid rgba(var(--v-border-color), calc(var(--v-border-opacity) * 0.6));
}
.channel-item:last-child {
  border-bottom: none;
}
.channel-item-animated {
  animation: ccx-slide-in 0.18s ease both;
}
.channel-item.demoted {
  opacity: 0.5;
}
.seq-num {
  font-size: 10px;
  font-weight: 700;
  opacity: 0.5;
  min-width: 2.5ch;
  font-variant-numeric: tabular-nums;
}
.seq-arrow {
  font-size: 10px;
  opacity: 0.35;
  margin: 0 4px;
}
.channel-name {
  cursor: pointer;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.channel-name:hover {
  text-decoration: underline;
  color: rgb(var(--v-theme-primary));
}

/* Fused chip */
.fused-chip {
  background: repeating-linear-gradient(
    -45deg,
    var(--ccx-fused-stripe-b) 0px,
    var(--ccx-fused-stripe-b) 4px,
    var(--ccx-fused-stripe-a) 4px,
    var(--ccx-fused-stripe-a) 8px
  ) !important;
  color: #fff !important;
  border-radius: 0 !important;
  border: none !important;
  font-weight: 700;
  font-size: 9px !important;
  letter-spacing: 0.05em;
}

/* Override alert */
.override-alert {
  border: 2px solid rgb(var(--v-theme-warning)) !important;
  border-radius: 0 !important;
}
.alert-bang {
  font-weight: 900;
  font-size: 11px;
  letter-spacing: 0.1em;
  margin-right: 6px;
  animation: ccx-alert-blink 0.8s step-end infinite;
  color: rgb(var(--v-theme-warning));
}

.current-channel-chip {
  cursor: default !important;
  opacity: 0.85;
}

.next-channel-chip {
  font-weight: 700;
}
.next-channel-chip :deep(.v-chip__content),
.next-channel-chip :deep(.v-chip__append) {
  color: #fff !important;
}

.font-weight-mono {
  font-family: monospace;
}

/* Raw User ID */
.raw-user-id {
  border-top: 1px dashed rgba(var(--v-border-color), var(--v-border-opacity));
  padding-top: 6px;
  opacity: 0.6;
  cursor: pointer;
}
.raw-user-id:hover {
  opacity: 0.9;
}
.raw-user-id-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.raw-user-id .copy-btn {
  flex-shrink: 0;
  opacity: 0.5;
}
.raw-user-id:hover .copy-btn {
  opacity: 1;
}
</style>
