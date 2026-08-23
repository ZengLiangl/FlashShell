<template>
  <div v-if="enabled" class="shell-compose-bar">
    <div class="compose-main">
      <div class="compose-head">
        <span class="compose-label">撰写</span>
        <span class="compose-hint">Ctrl/⌘+Enter 发送 · Shift+Enter 换行</span>
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

    <div class="aux-bar-actions">
      <el-tooltip content="发送 (Ctrl/⌘+Enter)" placement="top">
        <button
          type="button"
          class="aux-action-btn is-send"
          :disabled="!draft.trim() || !sessionId"
          @click="send"
        >
          <el-icon :size="14"><Promotion /></el-icon>
        </button>
      </el-tooltip>
      <el-tooltip content="关闭撰写栏" placement="top">
        <button type="button" class="aux-action-btn is-close" @click="close">
          <el-icon :size="14"><Close /></el-icon>
        </button>
      </el-tooltip>
    </div>
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
  display: flex;
  align-items: center;
  gap: 10px;
  border-bottom: 1px solid var(--term-border);
  background: oklch(24% 0.02 240);
  padding: 6px 12px;
  box-sizing: border-box;
  font-size: 12.5px;
}

.compose-label {
  color: var(--term-dim);
  font-family: var(--font-mono);
}

.compose-input {
  border: 1px solid var(--term-border);
  border-radius: 7px;
  background: var(--term-bg);
  color: var(--term-fg);
  font-family: var(--font-mono);
  min-height: 26px;
  max-height: 120px;
}

.compose-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.compose-head {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 28px;
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

.aux-bar-actions {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 2px;
  width: 60px;
  height: 28px;
  margin-top: 0;
  justify-content: flex-end;
}

.aux-action-btn {
  box-sizing: border-box;
  width: 28px;
  height: 28px;
  padding: 0;
  margin: 0;
  border: none;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--app-text-muted);
  background: transparent;
  transition: color 0.15s ease, background 0.15s ease, filter 0.15s ease;
}

.aux-action-btn.is-send {
  color: #fff;
  background: var(--app-accent-color);
}

.aux-action-btn.is-send:hover:not(:disabled) {
  filter: brightness(1.08);
  background: var(--app-accent-color);
}

.aux-action-btn.is-send:disabled {
  opacity: 0.4;
  cursor: not-allowed;
  filter: none;
}

.aux-action-btn.is-close:hover {
  color: var(--app-text);
  background: color-mix(in srgb, var(--app-text) 10%, transparent);
}
</style>
