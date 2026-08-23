<template>
  <div v-if="enabled" class="shell-broadcast-bar">
    <div class="broadcast-main">
      <div class="broadcast-meta">
        <span class="broadcast-dot" aria-hidden="true"></span>
        <span class="broadcast-label">广播</span>
        <span class="broadcast-count">{{ targetCount }} 个会话</span>
      </div>

      <div class="broadcast-targets" ref="targetsContainerRef">
        <button
          v-for="s in visibleTargets"
          :key="s.machineName"
          type="button"
          class="target-chip"
          :class="{ selected: isTarget(s.machineName) }"
          @click="toggleTarget(s.machineName)"
        >
          {{ s.tabLabel || s.machineName }}
        </button>
        <el-dropdown
          v-if="overflowTargets.length"
          trigger="click"
          @command="toggleTarget"
        >
          <button type="button" class="target-chip target-more" title="更多会话">
            <el-icon :size="12"><MoreFilled /></el-icon>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item
                v-for="s in overflowTargets"
                :key="s.machineName"
                :command="s.machineName"
              >
                {{ s.tabLabel || s.machineName }}
                <span v-if="isTarget(s.machineName)" class="chip-mark"> ✓</span>
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
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
      </div>
    </div>

    <div class="aux-bar-actions">
      <el-tooltip content="发送 (Enter)" placement="top">
        <button
          type="button"
          class="aux-action-btn is-send"
          :disabled="!draft.trim() || !targetCount"
          @click="send"
        >
          <el-icon :size="14"><Promotion /></el-icon>
        </button>
      </el-tooltip>
      <el-tooltip content="关闭广播" placement="top">
        <button type="button" class="aux-action-btn is-close" @click="close">
          <el-icon :size="14"><Close /></el-icon>
        </button>
      </el-tooltip>
    </div>
  </div>
</template>

<script>
import { ref, watch, computed, nextTick } from 'vue'
import { Close, Promotion, MoreFilled } from '@element-plus/icons-vue'
import * as App from '../../../wailsjs/go/app/App'
import { useHorizontalOverflow } from '../../composables/useHorizontalOverflow'

export default {
  name: 'ShellBroadcastBar',
  components: { Close, Promotion, MoreFilled },
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

    const activeTargetKey = ref('')
    const { containerRef: targetsContainerRef, split: targetSplit } = useHorizontalOverflow(
      connectedSessions,
      activeTargetKey,
      { itemWidth: 88, moreWidth: 36 },
    )
    const visibleTargets = computed(() => targetSplit.value.visible)
    const overflowTargets = computed(() => targetSplit.value.overflow)

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
      targetsContainerRef,
      connectedSessions,
      visibleTargets,
      overflowTargets,
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
  display: flex;
  align-items: center;
  gap: 10px;
  border-bottom: 1px solid var(--term-border);
  background: oklch(24% 0.02 240);
  padding: 6px 12px;
  box-sizing: border-box;
  font-size: 12.5px;
}

.broadcast-meta {
  color: var(--term-green);
  font-family: var(--font-mono);
  border-right: none;
  padding-right: 0;
}

.broadcast-label {
  color: var(--term-green);
  font-weight: 500;
}

.broadcast-count {
  color: var(--term-dim);
}

.target-chip {
  height: 20px;
  padding: 0 9px;
  border-radius: 6px;
  border: 1px solid var(--term-border);
  color: var(--term-dim);
  font-size: 12px;
  background: transparent;
}

.target-chip.selected {
  background: oklch(30% 0.04 150);
  border-color: oklch(38% 0.06 150);
  color: var(--term-green);
}

.broadcast-command {
  flex: 1;
  background: var(--term-bg);
  border: 1px solid var(--term-border);
  border-radius: 7px;
  height: 30px;
}

.cmd-prompt {
  color: var(--term-green);
  font-family: var(--font-mono);
}

.cmd-input {
  color: var(--term-fg);
  font-family: var(--font-mono);
  font-size: 12.5px;
}

.broadcast-main {
  flex: 1;
  min-width: 0;
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
  transition: border-color 0.15s ease, color 0.15s ease, background 0.15s ease;
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
  border-radius: 4px;
  transition: color 0.15s ease, background 0.15s ease;
}

.target-action:hover {
  color: var(--app-accent-color);
  background: color-mix(in srgb, var(--app-accent-color) 10%, transparent);
}

.target-more {
  padding: 5px 8px;
}

.chip-mark {
  color: var(--app-accent-color);
}

.broadcast-command {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 140px;
  margin-left: auto;
  padding: 0 10px;
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius-md, 8px);
  background: var(--terminal-bg);
  height: 32px;
  box-sizing: border-box;
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
  height: 100%;
}

.cmd-input:focus {
  outline: none;
}

.cmd-input::placeholder {
  color: var(--app-text-muted);
}

.aux-bar-actions {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 2px;
  width: 60px;
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
  transition: color 0.15s ease, background 0.15s ease;
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
