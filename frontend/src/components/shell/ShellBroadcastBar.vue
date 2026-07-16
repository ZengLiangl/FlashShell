<template>
  <div v-if="enabled" class="shell-broadcast-bar">
    <div class="broadcast-toolbar">
      <div class="broadcast-meta">
        <span class="broadcast-dot" aria-hidden="true"></span>
        <span class="broadcast-label">广播</span>
        <span class="broadcast-count">{{ targetCount }} 个会话</span>
      </div>

      <div class="broadcast-targets">
        <button
          v-for="s in connectedSessions"
          :key="s.machineName"
          type="button"
          class="target-chip"
          :class="{ selected: isTarget(s.machineName) }"
          @click="toggleTarget(s.machineName)"
        >
          {{ s.tabLabel || s.machineName }}
        </button>
        <button type="button" class="target-action" @click="selectAll">全选</button>
      </div>

      <div class="broadcast-command">
        <span class="cmd-prompt" aria-hidden="true">&gt;</span>
        <input
          ref="inputRef"
          v-model="draft"
          class="cmd-input"
          type="text"
          spellcheck="false"
          placeholder="输入命令，Enter 发送"
          @keydown="onKeydown"
        />
        <el-tooltip content="发送 (Enter)" placement="top">
          <el-button
            class="send-btn"
            circle
            size="small"
            type="primary"
            :disabled="!draft.trim() || !targetCount"
            @click="send"
          >
            <el-icon><Promotion /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip content="关闭广播" placement="top">
          <el-button class="close-btn" circle size="small" text @click="close">
            <el-icon><Close /></el-icon>
          </el-button>
        </el-tooltip>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, watch, computed, nextTick } from 'vue'
import { Close, Promotion } from '@element-plus/icons-vue'
import * as App from '../../../wailsjs/go/app/App'

export default {
  name: 'ShellBroadcastBar',
  components: { Close, Promotion },
  props: {
    enabled: { type: Boolean, default: false },
    targets: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
  },
  emits: ['update:enabled', 'update:targets'],
  setup(props, { emit }) {
    const draft = ref('')
    const inputRef = ref(null)

    const connectedSessions = computed(() =>
      (props.sessions || []).filter((s) => s.connected),
    )

    const effectiveTargets = computed(() => {
      if (props.targets?.length) return props.targets
      return connectedSessions.value.map((s) => s.machineName)
    })

    const targetCount = computed(() => effectiveTargets.value.length)

    const isTarget = (id) => effectiveTargets.value.includes(id)

    const ensureTargets = () => {
      if (!props.targets?.length && connectedSessions.value.length) {
        emit(
          'update:targets',
          connectedSessions.value.map((s) => s.machineName),
        )
      }
    }

    const toggleTarget = (id) => {
      const all = connectedSessions.value.map((s) => s.machineName)
      const set = new Set(props.targets?.length ? props.targets : all)
      if (set.has(id)) set.delete(id)
      else set.add(id)
      emit('update:targets', [...set])
    }

    const selectAll = () => {
      emit(
        'update:targets',
        connectedSessions.value.map((s) => s.machineName),
      )
    }

    const close = () => {
      emit('update:enabled', false)
    }

    const send = async () => {
      const text = draft.value
      if (!text.trim() || !effectiveTargets.value.length) return
      const payload = text.endsWith('\n') ? text : `${text}\n`
      try {
        await App.BroadcastShellInput(effectiveTargets.value, payload)
        draft.value = ''
      } catch (e) {
        console.warn('broadcast failed', e)
      }
    }

    const onKeydown = (e) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        send()
      }
      if (e.key === 'Escape') {
        e.preventDefault()
        close()
      }
    }

    watch(
      () => props.enabled,
      (on) => {
        if (on) {
          ensureTargets()
          nextTick(() => inputRef.value?.focus?.())
        } else {
          draft.value = ''
        }
      },
    )

    watch(connectedSessions, () => {
      if (!props.enabled) return
      ensureTargets()
    })

    return {
      draft,
      inputRef,
      connectedSessions,
      targetCount,
      isTarget,
      toggleTarget,
      selectAll,
      close,
      send,
      onKeydown,
    }
  },
}
</script>

<style scoped>
.shell-broadcast-bar {
  flex-shrink: 0;
  border-bottom: 1px solid var(--app-border);
  background: var(--app-card-bg);
  padding: 6px 10px;
}

.broadcast-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 34px;
}

.broadcast-meta {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  padding-right: 10px;
  border-right: 1px solid var(--app-border);
}

.broadcast-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--app-accent-color);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--app-accent-color) 22%, transparent);
  animation: broadcast-pulse 1.6s ease-in-out infinite;
}

@keyframes broadcast-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.45; }
}

.broadcast-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--app-text);
}

.broadcast-count {
  font-size: 11px;
  color: var(--app-text-muted);
}

.broadcast-targets {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 0 1 auto;
  min-width: 0;
  overflow-x: auto;
  scrollbar-width: thin;
  padding: 2px 0;
}

.target-chip {
  flex-shrink: 0;
  border: 1px solid var(--app-border);
  background: var(--app-panel-bg);
  color: var(--app-text-secondary);
  font-size: 11px;
  line-height: 1;
  padding: 5px 10px;
  border-radius: 999px;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s, background 0.15s;
}

.target-chip.selected {
  border-color: color-mix(in srgb, var(--app-accent-color) 55%, var(--app-border));
  color: var(--app-accent-color);
  background: var(--app-accent-bg);
}

.target-chip:hover {
  border-color: var(--app-accent-color);
}

.target-action {
  flex-shrink: 0;
  border: none;
  background: transparent;
  color: var(--app-text-muted);
  font-size: 11px;
  cursor: pointer;
  padding: 4px 6px;
}

.target-action:hover {
  color: var(--app-accent-color);
}

.broadcast-command {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 160px;
  margin-left: auto;
  padding: 2px 4px 2px 10px;
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius-md, 8px);
  background: var(--terminal-bg);
}

.cmd-prompt {
  color: var(--terminal-success, var(--app-accent-color));
  font-family: var(--app-font-family-mono, Consolas, monospace);
  font-size: 13px;
  flex-shrink: 0;
}

.cmd-input {
  flex: 1;
  min-width: 0;
  border: none;
  background: transparent;
  color: var(--terminal-fg, var(--app-text));
  font-family: var(--app-font-family-mono, Consolas, monospace);
  font-size: 13px;
  height: 28px;
}

.cmd-input:focus {
  outline: none;
}

.cmd-input::placeholder {
  color: var(--app-text-muted);
}

.send-btn {
  flex-shrink: 0;
}

.close-btn {
  flex-shrink: 0;
  color: var(--app-text-muted);
}
</style>
