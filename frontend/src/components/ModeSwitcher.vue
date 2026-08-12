<template>
  <div
    ref="hostRef"
    class="mode-switcher-host"
    :class="{
      'is-compact': compact,
      'is-expanded': showPanel,
      'is-float-end': compact && floatAlign === 'end',
    }"
    @mouseenter="onHostEnter"
    @mouseleave="onHostLeave"
  >
    <button
      v-if="compact"
      type="button"
      class="mode-trigger"
      :title="triggerTitle"
      :aria-expanded="showPanel"
      aria-label="切换任务或 Shell"
    >
      <el-icon class="mode-trigger-icon" :size="13">
        <component :is="activeItem.icon" />
      </el-icon>
      <span
        v-if="activeItem.badge"
        class="mode-trigger-badge"
        aria-hidden="true"
      >{{ activeItem.badge }}</span>
      <span
        v-else-if="activeItem.dot"
        class="mode-trigger-dot"
        aria-hidden="true"
      />
    </button>

    <Teleport to="body" :disabled="!compact">
      <div
        v-show="showPanel"
        class="mode-picker"
        :class="{ 'is-floating': compact }"
        :style="compact ? floatStyle : undefined"
        role="menu"
        aria-label="任务与 Shell"
        @mouseenter="onHostEnter"
        @mouseleave="onHostLeave"
      >
        <div class="mode-picker-section">
          <div class="mode-picker-head">
            <el-icon :size="13"><Folder /></el-icon>
            <span>任务项目</span>
            <span v-if="projects.length" class="mode-picker-count">{{ projects.length }}</span>
          </div>
          <div v-if="!projects.length" class="mode-picker-empty">暂无任务项目</div>
          <button
            v-for="project in projects"
            :key="project.name"
            type="button"
            class="mode-picker-item"
            :class="{ active: project.name === selectedProjectName && modelValue === 'task' }"
            role="menuitem"
            @click="onPickProject(project.name)"
          >
            <span class="mode-picker-title">{{ project.name || '(未命名项目)' }}</span>
            <span v-if="project.name === selectedProjectName && modelValue === 'task'" class="mode-picker-check">✓</span>
          </button>
        </div>

        <div class="mode-picker-section">
          <div class="mode-picker-head">
            <el-icon :size="13"><Monitor /></el-icon>
            <span>已连接 Shell</span>
            <span v-if="connectedSessions.length" class="mode-picker-count">{{ connectedSessions.length }}</span>
          </div>
          <div v-if="!connectedSessions.length" class="mode-picker-empty">暂无已连接会话</div>
          <button
            v-for="session in connectedSessions"
            :key="session.machineName"
            type="button"
            class="mode-picker-item"
            :class="{ active: session.machineName === activeSessionId && modelValue === 'shell' }"
            role="menuitem"
            @click="onPickSession(session.machineName)"
          >
            <span class="mode-picker-dot" :class="session.connected ? 'is-on' : 'is-off'" aria-hidden="true" />
            <span class="mode-picker-main">
              <span class="mode-picker-title">{{ sessionLabel(session) }}</span>
              <span v-if="sessionSub(session)" class="mode-picker-sub">{{ sessionSub(session) }}</span>
            </span>
            <span
              v-if="session.machineName === activeSessionId && modelValue === 'shell'"
              class="mode-picker-check"
            >✓</span>
          </button>
          <button
            v-if="!connectedSessions.length"
            type="button"
            class="mode-picker-item mode-picker-item--action"
            role="menuitem"
            @click="onOpenShell"
          >
            打开 Shell…
          </button>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script>
import { computed, ref, onUnmounted, watch, nextTick } from 'vue'
import { Folder, Monitor } from '@element-plus/icons-vue'

const isLocalSession = (s) =>
  s?.kind === 'local' || String(s?.machineName || '').startsWith('local')

export default {
  name: 'ModeSwitcher',
  components: { Folder, Monitor },
  props: {
    modelValue: {
      type: String,
      default: 'home',
      validator: (v) => ['home', 'task', 'shell'].includes(v),
    },
    compact: { type: Boolean, default: false },
    floatAlign: {
      type: String,
      default: 'center',
      validator: (v) => ['center', 'end'].includes(v),
    },
    hasProjects: { type: Boolean, default: false },
    hasMachines: { type: Boolean, default: false },
    hasTask: { type: Boolean, default: false },
    taskRunning: { type: Boolean, default: false },
    connectedCount: { type: Number, default: 0 },
    projects: { type: Array, default: () => [] },
    selectedProjectName: { type: String, default: '' },
    /** 工作区会话（含未连接），用于挑已连接 Shell */
    sessions: { type: Array, default: () => [] },
    activeSessionId: { type: String, default: '' },
  },
  emits: ['update:modelValue', 'change', 'select-project', 'focus-session'],
  setup(props, { emit }) {
    const hostRef = ref(null)
    const hoverOpen = ref(false)
    const pointerInside = ref(false)
    const floatStyle = ref({})
    let leaveTimer = null

    const showPanel = computed(() => !props.compact || hoverOpen.value)

    const connectedSessions = computed(() =>
      (props.sessions || []).filter((s) => s?.machineName && s.connected),
    )

    const sessionLabel = (session) => {
      if (session?.tabLabel) return session.tabLabel
      if (isLocalSession(session)) {
        const n = String(session.machineName || '').replace(/^local-?/, '')
        return n ? `本机-${n}` : '本机'
      }
      return session?.configName || session?.machineName || ''
    }

    const sessionSub = (session) => {
      const host = String(session?.host || '').trim()
      const label = sessionLabel(session)
      if (host && label && !label.includes(host)) return host
      return ''
    }

    const activeItem = computed(() => {
      if (props.modelValue === 'task') {
        return {
          icon: Folder,
          badge: '',
          dot: !!props.taskRunning,
          label: '任务',
        }
      }
      return {
        icon: Monitor,
        badge: props.connectedCount > 0 ? String(props.connectedCount) : '',
        dot: false,
        label: 'Shell',
      }
    })

    const triggerTitle = computed(() => {
      const label = activeItem.value.label
      return `当前：${label}（悬停选择项目或已连接 Shell）`
    })

    const updateFloatPosition = () => {
      if (!props.compact || !hostRef.value) return
      const rect = hostRef.value.getBoundingClientRect()
      const gap = 6
      const style = {
        position: 'fixed',
        top: `${Math.round(rect.bottom + gap)}px`,
        zIndex: 4000,
      }
      if (props.floatAlign === 'end') {
        style.right = `${Math.round(Math.max(8, window.innerWidth - rect.right))}px`
        style.left = 'auto'
        style.transform = 'none'
      } else {
        style.left = `${Math.round(rect.left + rect.width / 2)}px`
        style.right = 'auto'
        style.transform = 'translateX(-50%)'
      }
      floatStyle.value = style
    }

    const clearLeaveTimer = () => {
      if (leaveTimer) {
        clearTimeout(leaveTimer)
        leaveTimer = null
      }
    }

    const scheduleClose = (delay = 160) => {
      clearLeaveTimer()
      leaveTimer = setTimeout(() => {
        if (!pointerInside.value) hoverOpen.value = false
      }, delay)
    }

    const onHostEnter = () => {
      pointerInside.value = true
      clearLeaveTimer()
      if (props.compact) {
        hoverOpen.value = true
        nextTick(() => updateFloatPosition())
      }
    }

    const onHostLeave = () => {
      pointerInside.value = false
      if (props.compact) scheduleClose(180)
    }

    const onWinChange = () => {
      if (hoverOpen.value) updateFloatPosition()
    }

    watch(() => props.modelValue, () => {
      hoverOpen.value = false
      pointerInside.value = false
      clearLeaveTimer()
    })

    watch(hoverOpen, (open) => {
      if (open && props.compact) {
        nextTick(() => updateFloatPosition())
        window.addEventListener('resize', onWinChange)
        window.addEventListener('scroll', onWinChange, true)
      } else {
        window.removeEventListener('resize', onWinChange)
        window.removeEventListener('scroll', onWinChange, true)
      }
    })

    onUnmounted(() => {
      clearLeaveTimer()
      window.removeEventListener('resize', onWinChange)
      window.removeEventListener('scroll', onWinChange, true)
    })

    const onPickProject = (name) => {
      const project = props.projects.find((p) => p.name === name)
      if (!project) return
      emit('select-project', project)
      hoverOpen.value = false
    }

    const onPickSession = (sessionId) => {
      if (!sessionId) return
      emit('focus-session', sessionId)
      hoverOpen.value = false
    }

    const onOpenShell = () => {
      emit('update:modelValue', 'shell')
      emit('change', 'shell')
      hoverOpen.value = false
    }

    return {
      hostRef,
      showPanel,
      floatStyle,
      activeItem,
      triggerTitle,
      connectedSessions,
      sessionLabel,
      sessionSub,
      onHostEnter,
      onHostLeave,
      onPickProject,
      onPickSession,
      onOpenShell,
    }
  },
}
</script>

<style scoped>
.mode-switcher-host {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 26px;
  overflow: visible;
}

.mode-trigger {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  min-width: 26px;
  padding: 0;
  border: 1px solid color-mix(in srgb, var(--app-border) 65%, transparent);
  border-radius: 7px;
  background: color-mix(in srgb, var(--app-panel-bg) 82%, transparent);
  color: var(--app-accent-color);
  cursor: pointer;
  opacity: 0.88;
  box-shadow: 0 1px 2px color-mix(in srgb, var(--app-text) 5%, transparent);
  transition:
    background 0.16s ease,
    border-color 0.16s ease,
    color 0.16s ease,
    opacity 0.12s ease,
    box-shadow 0.16s ease;
}

.mode-trigger-badge {
  position: absolute;
  top: -4px;
  right: -5px;
  min-width: 14px;
  height: 14px;
  padding: 0 3px;
  border-radius: 7px;
  font-size: 9px;
  font-weight: 700;
  line-height: 14px;
  text-align: center;
  color: var(--app-accent-color);
  background: color-mix(in srgb, var(--app-accent-color) 18%, var(--app-panel-bg));
  border: 1px solid color-mix(in srgb, var(--app-accent-color) 28%, var(--app-border));
}

.mode-trigger-dot {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--app-accent-color);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--app-accent-color) 20%, transparent);
}

.mode-trigger:hover,
.mode-switcher-host.is-expanded .mode-trigger {
  opacity: 1;
  color: var(--app-accent-color);
  background: var(--app-accent-bg);
  border-color: color-mix(in srgb, var(--app-accent-color) 35%, var(--app-border));
}

.mode-trigger-icon {
  flex-shrink: 0;
}
</style>

<style>
/* Teleport 到 body 后需非 scoped */
.mode-picker {
  min-width: 220px;
  max-width: 280px;
  max-height: min(420px, 70vh);
  overflow: auto;
  padding: 6px;
  border-radius: 10px;
  border: 1px solid var(--app-border);
  background: var(--app-card-bg, var(--app-panel-bg));
  box-shadow: 0 10px 28px color-mix(in srgb, #000 28%, transparent);
  box-sizing: border-box;
}

.mode-picker.is-floating {
  position: fixed;
}

.mode-picker-section + .mode-picker-section {
  margin-top: 6px;
  padding-top: 6px;
  border-top: 1px solid color-mix(in srgb, var(--app-border) 80%, transparent);
}

.mode-picker-head {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px 6px;
  font-size: 11px;
  font-weight: 600;
  color: var(--app-text-muted);
}

.mode-picker-count {
  margin-left: auto;
  min-width: 16px;
  padding: 0 5px;
  border-radius: 8px;
  font-size: 10px;
  line-height: 16px;
  text-align: center;
  color: var(--app-text-secondary);
  background: color-mix(in srgb, var(--app-text) 8%, transparent);
}

.mode-picker-empty {
  padding: 8px 10px;
  font-size: 12px;
  color: var(--app-text-muted);
}

.mode-picker-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  margin: 0;
  padding: 7px 10px;
  border: none;
  border-radius: 7px;
  background: transparent;
  color: var(--app-text);
  font-size: 12px;
  text-align: left;
  cursor: pointer;
  box-sizing: border-box;
}

.mode-picker-item:hover {
  background: color-mix(in srgb, var(--app-text) 7%, transparent);
}

.mode-picker-item.active {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}

.mode-picker-item--action {
  color: var(--app-accent-color);
  font-weight: 600;
}

.mode-picker-dot {
  flex-shrink: 0;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--app-text-muted);
}

.mode-picker-dot.is-on {
  background: var(--app-success-color, #67c23a);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--app-success-color, #67c23a) 22%, transparent);
}

.mode-picker-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.mode-picker-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 500;
}

.mode-picker-sub {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
  color: var(--app-text-muted);
}

.mode-picker-item.active .mode-picker-sub {
  color: color-mix(in srgb, var(--app-accent-color) 70%, var(--app-text-muted));
}

.mode-picker-check {
  flex-shrink: 0;
  font-size: 12px;
  font-weight: 700;
  color: var(--app-accent-color);
}
</style>
