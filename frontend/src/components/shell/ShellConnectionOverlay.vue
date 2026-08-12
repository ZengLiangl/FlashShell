<template>
  <div v-if="visible" class="shell-conn-overlay" :class="statusClass">
    <div class="conn-card">
      <div class="conn-icon" aria-hidden="true">
        <el-icon v-if="status === 'connecting' || status === 'reconnecting'" class="is-loading" :size="28"><Loading /></el-icon>
        <el-icon v-else-if="status === 'disconnected'" :size="28"><WarningFilled /></el-icon>
        <el-icon v-else :size="28"><Monitor /></el-icon>
      </div>
      <div class="conn-title">{{ title }}</div>
      <div v-if="subtitle" class="conn-sub">{{ subtitle }}</div>
      <div v-if="jumpHint" class="conn-jump">{{ jumpHint }}</div>
      <div v-if="status === 'connecting' || status === 'reconnecting'" class="conn-progress">
        <div class="conn-progress-bar" />
      </div>
      <div v-if="status === 'disconnected'" class="conn-actions">
        <el-button type="primary" size="small" @click="$emit('reconnect')">重新连接</el-button>
        <span class="conn-hint">或按 Enter 重连</span>
      </div>
    </div>
  </div>
</template>

<script>
import { computed } from 'vue'
import { Loading, WarningFilled, Monitor } from '@element-plus/icons-vue'

export default {
  name: 'ShellConnectionOverlay',
  components: { Loading, WarningFilled, Monitor },
  props: {
    status: { type: String, default: '' }, // connecting | reconnecting | disconnected | ''
    machineName: { type: String, default: '' },
    host: { type: String, default: '' },
    user: { type: String, default: '' },
    jumpChain: { type: Array, default: () => [] },
    proxyJump: { type: String, default: '' },
    reconnectAttempt: { type: Number, default: 0 },
    reconnectMax: { type: Number, default: 0 },
    reconnectDelaySec: { type: Number, default: 0 },
  },
  emits: ['reconnect'],
  setup(props) {
    const visible = computed(() =>
      props.status === 'connecting' ||
      props.status === 'reconnecting' ||
      props.status === 'disconnected',
    )
    const statusClass = computed(() => ({
      'is-connecting': props.status === 'connecting' || props.status === 'reconnecting',
      'is-disconnected': props.status === 'disconnected',
    }))
    const title = computed(() => {
      if (props.status === 'reconnecting') {
        const n = props.reconnectAttempt || 1
        const max = props.reconnectMax || 3
        return `重连中 第 ${n}/${max} 次`
      }
      if (props.status === 'connecting') return `正在连接 ${props.machineName || ''}`.trim()
      if (props.status === 'disconnected') return `已断开 ${props.machineName || ''}`.trim()
      return ''
    })
    const subtitle = computed(() => {
      if (props.status === 'reconnecting' && props.reconnectDelaySec > 0) {
        return `退避等待 ${props.reconnectDelaySec} 秒后重试`
      }
      const parts = []
      if (props.user) parts.push(props.user)
      if (props.host) parts.push(props.host)
      return parts.length ? parts.join('@') : ''
    })
    const jumpHint = computed(() => {
      const chain = (props.jumpChain || []).filter(Boolean)
      if (chain.length) return `跳板链：${chain.join(' → ')} → 目标`
      if (props.proxyJump) return `跳板：${props.proxyJump}`
      return ''
    })
    return { visible, statusClass, title, subtitle, jumpHint }
  },
}
</script>

<style scoped>
.shell-conn-overlay {
  position: absolute;
  inset: 0;
  z-index: 20;
  display: flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, var(--shell-term-bg, #1e1e1e) 72%, transparent);
  backdrop-filter: blur(2px);
  pointer-events: auto;
}

.conn-card {
  min-width: 260px;
  max-width: 420px;
  padding: 28px 32px;
  border-radius: 12px;
  background: var(--app-card-bg, #fff);
  border: 1px solid var(--app-border, #e4e7ed);
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.22);
  text-align: center;
}

.conn-icon {
  margin-bottom: 12px;
  color: var(--app-accent-color, #409eff);
}

.is-disconnected .conn-icon {
  color: var(--el-color-warning, #e6a23c);
}

.conn-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--app-text, #303133);
  line-height: 1.4;
}

.conn-sub {
  margin-top: 6px;
  font-size: 13px;
  color: var(--app-text-muted, #909399);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

.conn-jump {
  margin-top: 10px;
  font-size: 12px;
  color: var(--el-color-info, #909399);
  line-height: 1.4;
}

.conn-progress {
  margin-top: 16px;
  height: 3px;
  border-radius: 2px;
  background: var(--el-fill-color, #f0f2f5);
  overflow: hidden;
}

.conn-progress-bar {
  height: 100%;
  width: 40%;
  border-radius: 2px;
  background: var(--app-accent-color, #409eff);
  animation: conn-slide 1.2s ease-in-out infinite;
}

@keyframes conn-slide {
  0% { transform: translateX(-120%); }
  100% { transform: translateX(320%); }
}

.conn-actions {
  margin-top: 18px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.conn-hint {
  font-size: 12px;
  color: var(--app-text-muted, #909399);
}
</style>
