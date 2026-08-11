<template>
  <div v-if="enabled" class="shell-compose-bar">
    <div class="compose-head">
      <span class="compose-label">撰写</span>
      <span class="compose-hint">Ctrl/⌘+Enter 发送 · Shift+Enter 换行</span>
      <div class="compose-actions">
        <el-tooltip content="发送" placement="top">
          <el-button
            circle
            size="small"
            type="primary"
            :disabled="!draft.trim() || !sessionId"
            @click="send"
          >
            <el-icon><Promotion /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip content="关闭撰写栏" placement="top">
          <el-button circle size="small" text @click="close">
            <el-icon><Close /></el-icon>
          </el-button>
        </el-tooltip>
      </div>
    </div>
    <textarea
      ref="inputRef"
      v-model="draft"
      class="compose-input"
      spellcheck="false"
      placeholder="在此编辑多行命令，确认后再发送到终端…"
      @keydown="onKeydown"
    />
  </div>
</template>

<script>
import { ref, watch, nextTick } from 'vue'
import { Close, Promotion } from '@element-plus/icons-vue'
import * as App from '../../../wailsjs/go/app/App'

export default {
  name: 'ShellComposeBar',
  components: { Close, Promotion },
  props: {
    enabled: { type: Boolean, default: false },
    sessionId: { type: String, default: '' },
    broadcastEnabled: { type: Boolean, default: false },
    broadcastTargets: { type: Array, default: () => [] },
  },
  emits: ['update:enabled'],
  setup(props, { emit }) {
    const draft = ref('')
    const inputRef = ref(null)

    const close = () => emit('update:enabled', false)

    const send = async () => {
      const text = draft.value
      if (!text.trim()) return
      const payload = text.endsWith('\n') ? text : `${text}\n`
      try {
        if (props.broadcastEnabled && (props.broadcastTargets || []).length) {
          await App.BroadcastShellInput(props.broadcastTargets, payload)
        } else if (props.sessionId) {
          await App.SendShellInput(props.sessionId, payload)
        } else {
          return
        }
        draft.value = ''
      } catch (e) {
        console.warn('compose send failed', e)
      }
    }

    const onKeydown = (e) => {
      if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
        e.preventDefault()
        send()
        return
      }
      if (e.key === 'Escape') {
        e.preventDefault()
        close()
      }
    }

    watch(
      () => props.enabled,
      (on) => {
        if (on) nextTick(() => inputRef.value?.focus?.())
        else draft.value = ''
      },
    )

    return { draft, inputRef, close, send, onKeydown }
  },
}
</script>

<style scoped>
.shell-compose-bar {
  flex-shrink: 0;
  border-bottom: 1px solid var(--app-border);
  background: var(--app-card-bg);
  padding: 6px 10px 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.compose-head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.compose-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--app-text);
}

.compose-hint {
  font-size: 11px;
  color: var(--app-text-muted);
  flex: 1;
  min-width: 0;
}

.compose-actions {
  display: inline-flex;
  gap: 4px;
}

.compose-input {
  width: 100%;
  min-height: 72px;
  max-height: 180px;
  resize: vertical;
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius-md, 8px);
  background: var(--terminal-bg);
  color: var(--terminal-fg, var(--app-text));
  font-family: var(--app-font-family-mono, Consolas, monospace);
  font-size: 13px;
  line-height: 1.45;
  padding: 8px 10px;
  box-sizing: border-box;
}

.compose-input:focus {
  outline: none;
  border-color: color-mix(in srgb, var(--app-accent-color) 55%, var(--app-border));
}

.compose-input::placeholder {
  color: var(--app-text-muted);
}
</style>
