<template>
  <div
    ref="hostRef"
    class="mode-switcher-host"
    :class="{
      'is-compact': compact,
      'is-expanded': showSwitcher,
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
      :aria-expanded="showSwitcher"
      aria-label="展开模块切换"
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
        v-show="showSwitcher"
        class="mode-switcher"
        :class="{ 'is-floating': compact }"
        :style="compact ? floatStyle : undefined"
        role="tablist"
        aria-label="工作模式"
        @mouseenter="onHostEnter"
        @mouseleave="onHostLeave"
      >
        <template v-for="item in displayItems" :key="item.id">
          <el-dropdown
            v-if="item.showProjectPicker"
            trigger="hover"
            placement="bottom-end"
            :show-timeout="compact ? 220 : 160"
            :hide-timeout="compact ? 180 : 160"
            :popper-options="taskPopperOptions"
            popper-class="mode-task-project-popper"
            @visible-change="onTaskMenuVisible"
            @command="onPickProject"
          >
            <div class="mode-dropdown-trigger">
              <button
                type="button"
                role="tab"
                class="mode-btn mode-task mode-btn--picker"
                :class="{ active: modelValue === 'task' }"
                :aria-selected="modelValue === 'task'"
                :title="item.title"
                @click="onTaskClick"
              >
                <el-icon class="mode-icon" :size="13">
                  <component :is="item.icon" />
                </el-icon>
                <span class="mode-label">{{ item.label }}</span>
                <span
                  v-if="item.dot"
                  class="mode-dot"
                  aria-hidden="true"
                />
              </button>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item
                  v-for="project in projects"
                  :key="project.name"
                  :command="project.name"
                  :class="{ 'is-current': project.name === selectedProjectName }"
                >
                  {{ project.name || '(未命名项目)' }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>

          <button
            v-else
            type="button"
            role="tab"
            class="mode-btn"
            :class="[
              `mode-${item.id}`,
              { active: modelValue === item.id, disabled: item.disabled },
            ]"
            :aria-selected="modelValue === item.id"
            :disabled="item.disabled"
            :title="item.title"
            @click="onSelect(item)"
          >
            <el-icon class="mode-icon" :size="13">
              <component :is="item.icon" />
            </el-icon>
            <span class="mode-label">{{ item.label }}</span>
            <span
              v-if="item.badge"
              class="mode-badge"
              aria-hidden="true"
            >{{ item.badge }}</span>
            <span
              v-else-if="item.dot"
              class="mode-dot"
              aria-hidden="true"
            />
          </button>
        </template>
      </div>
    </Teleport>
  </div>
</template>

<script>
import { computed, ref, onUnmounted, watch, nextTick } from 'vue'
import { HomeFilled, Folder, Monitor } from '@element-plus/icons-vue'

export default {
  name: 'ModeSwitcher',
  components: { HomeFilled, Folder, Monitor },
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
  },
  emits: ['update:modelValue', 'change', 'select-project'],
  setup(props, { emit }) {
    const hostRef = ref(null)
    const hoverOpen = ref(false)
    const taskMenuOpen = ref(false)
    const pointerInside = ref(false)
    const floatStyle = ref({})
    let leaveTimer = null

    const showSwitcher = computed(() => !props.compact || hoverOpen.value)

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

    const taskPopperOptions = {
      modifiers: [
        {
          name: 'offset',
          options: { offset: [0, 6] },
        },
      ],
    }

    const items = computed(() => {
      const list = [
        {
          id: 'home',
          label: '首页',
          icon: HomeFilled,
          title: '返回首页',
          disabled: false,
        },
      ]
      const canPickProject = props.projects.length > 0
      if (props.hasTask || props.modelValue === 'task') {
        list.push({
          id: 'task',
          label: '任务',
          icon: Folder,
          title: props.taskRunning
            ? '切换到任务工作台（运行中）· 悬停可换项目'
            : '切换到任务工作台 · 悬停可换项目',
          disabled: false,
          showProjectPicker: canPickProject,
          dot: props.taskRunning,
        })
      }
      if (props.hasMachines || props.connectedCount > 0 || props.modelValue === 'shell' || props.hasTask || props.modelValue === 'task') {
        list.push({
          id: 'shell',
          label: 'Shell',
          icon: Monitor,
          title: props.connectedCount > 0
            ? `切换到 Shell（${props.connectedCount} 个会话）`
            : '切换到 Shell 终端',
          disabled: false,
          badge: props.connectedCount > 0 ? String(props.connectedCount) : '',
        })
      }
      return list
    })

    const activeItem = computed(() => {
      const found = items.value.find((i) => i.id === props.modelValue)
      return found || items.value[0] || { id: 'home', icon: HomeFilled, label: '首页' }
    })

    const displayItems = computed(() => items.value)

    const triggerTitle = computed(() => {
      const label = activeItem.value.label || ''
      return `当前：${label}（悬停切换模块）`
    })

    const clearLeaveTimer = () => {
      if (leaveTimer) {
        clearTimeout(leaveTimer)
        leaveTimer = null
      }
    }

    const scheduleClose = (delay = 160) => {
      clearLeaveTimer()
      leaveTimer = setTimeout(() => {
        if (!taskMenuOpen.value && !pointerInside.value) {
          hoverOpen.value = false
        }
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

    const onTaskMenuVisible = (visible) => {
      taskMenuOpen.value = visible
      if (visible) {
        clearLeaveTimer()
        hoverOpen.value = true
        nextTick(() => updateFloatPosition())
        return
      }
      if (props.compact) scheduleClose(120)
    }

    const onWinChange = () => {
      if (hoverOpen.value) updateFloatPosition()
    }

    watch(() => props.modelValue, () => {
      hoverOpen.value = false
      taskMenuOpen.value = false
      pointerInside.value = false
      clearLeaveTimer()
    })

    watch(() => props.compact, () => {
      hoverOpen.value = false
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

    const onSelect = (item) => {
      if (item.disabled || item.id === props.modelValue) return
      emit('update:modelValue', item.id)
      emit('change', item.id)
      hoverOpen.value = false
    }

    const onTaskClick = () => {
      if (!props.hasTask) return
      if (props.modelValue === 'task') return
      emit('update:modelValue', 'task')
      emit('change', 'task')
      if (props.compact) hoverOpen.value = false
    }

    const onPickProject = (name) => {
      const project = props.projects.find((p) => p.name === name)
      if (!project) return
      emit('select-project', project)
      hoverOpen.value = false
    }

    return {
      hostRef,
      displayItems,
      activeItem,
      triggerTitle,
      showSwitcher,
      floatStyle,
      floatAlign: computed(() => props.floatAlign),
      taskPopperOptions,
      onHostEnter,
      onHostLeave,
      onTaskMenuVisible,
      onSelect,
      onTaskClick,
      onPickProject,
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
/* Teleport 到 body 后需非 scoped，否则面板样式丢失 */
.mode-switcher {
  --mode-track: color-mix(in srgb, var(--app-text) 6%, var(--app-panel-bg));
  --mode-home: var(--app-text);
  --mode-task: var(--app-accent-color);
  --mode-shell: var(--app-accent-color);

  display: inline-grid;
  grid-auto-flow: column;
  grid-auto-columns: minmax(108px, 1fr);
  align-items: stretch;
  gap: 2px;
  min-width: min(336px, 52vw);
  padding: 3px;
  border-radius: 9px;
  background: var(--mode-track);
  border: 1px solid color-mix(in srgb, var(--app-border) 80%, transparent);
  box-shadow: inset 0 1px 0 color-mix(in srgb, #fff 35%, transparent);
}

.mode-switcher.is-floating {
  width: max-content;
  min-width: 336px;
  background: var(--app-panel-bg);
  border: 1px solid var(--app-border);
  box-shadow:
    0 8px 24px color-mix(in srgb, var(--app-text) 12%, transparent),
    0 2px 6px color-mix(in srgb, var(--app-text) 8%, transparent),
    inset 0 1px 0 color-mix(in srgb, #fff 40%, transparent);
  animation: mode-float-in 0.14s ease-out;
}

.mode-switcher.is-floating::before {
  content: '';
  position: absolute;
  top: -10px;
  left: 0;
  right: 0;
  height: 10px;
}

@keyframes mode-float-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

.mode-switcher .el-dropdown {
  display: block;
  min-width: 0;
  width: 100%;
  height: 100%;
}

.mode-switcher .el-dropdown > .el-tooltip__trigger,
.mode-switcher .el-dropdown > span {
  display: block !important;
  width: 100%;
  height: 100%;
  min-width: 0;
}

.mode-switcher .mode-dropdown-trigger {
  display: block;
  width: 100%;
  height: 100%;
  min-width: 0;
}

.mode-switcher .mode-btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-width: 0;
  width: 100%;
  height: 26px;
  padding: 0 8px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--app-text-muted, #909399);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.01em;
  line-height: 1;
  cursor: pointer;
  transition:
    background 0.16s ease,
    color 0.16s ease,
    box-shadow 0.16s ease;
}

.mode-switcher .mode-icon {
  flex-shrink: 0;
  opacity: 0.85;
}

.mode-switcher .mode-label {
  flex: 0 1 auto;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mode-switcher .mode-btn:hover:not(.disabled):not(.active) {
  color: var(--app-accent-color);
  background: var(--app-accent-bg);
}

.mode-switcher .mode-btn.active {
  background: var(--app-card-bg, #fff);
  color: var(--app-text);
  font-weight: 600;
  box-shadow:
    0 1px 2px color-mix(in srgb, var(--app-text) 8%, transparent),
    0 0 0 1px color-mix(in srgb, var(--app-border) 75%, transparent);
}

.mode-switcher .mode-btn.mode-task.active,
.mode-switcher .mode-btn.mode-shell.active {
  color: var(--app-accent-color);
  background: var(--app-card-bg, #fff);
}

.mode-switcher .mode-btn.active .mode-icon {
  opacity: 1;
}

.mode-switcher .mode-btn.disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.mode-switcher .mode-badge {
  flex-shrink: 0;
  min-width: 15px;
  height: 15px;
  padding: 0 4px;
  border-radius: 8px;
  font-size: 10px;
  font-weight: 700;
  line-height: 15px;
  text-align: center;
  color: var(--mode-shell);
  background: color-mix(in srgb, var(--mode-shell) 16%, transparent);
}

.mode-switcher .mode-btn.active .mode-badge {
  background: color-mix(in srgb, var(--mode-shell) 20%, transparent);
}

.mode-switcher .mode-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
  background: var(--mode-task);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--mode-task) 24%, transparent);
}

.mode-switcher .el-dropdown-menu__item.is-current {
  color: var(--app-accent-color);
  background: var(--app-accent-bg);
  font-weight: 600;
}

html.dark .mode-switcher {
  --mode-track: color-mix(in srgb, #000 28%, var(--app-panel-bg));
  box-shadow: inset 0 1px 0 color-mix(in srgb, #fff 4%, transparent);
}

html.dark .mode-switcher.is-floating {
  background: var(--app-panel-bg);
  box-shadow:
    0 10px 28px rgba(0, 0, 0, 0.45),
    0 2px 8px rgba(0, 0, 0, 0.3),
    inset 0 1px 0 color-mix(in srgb, #fff 5%, transparent);
}

html.dark .mode-switcher .mode-btn.active {
  background: color-mix(in srgb, var(--app-card-bg) 92%, #fff);
  box-shadow:
    0 1px 3px rgba(0, 0, 0, 0.35),
    0 0 0 1px color-mix(in srgb, var(--app-border) 80%, transparent);
}

.mode-task-project-popper.el-popper,
.mode-task-project-popper {
  min-width: 0 !important;
  width: max-content;
  max-width: 220px;
}

.mode-task-project-popper .el-dropdown-menu {
  min-width: 0 !important;
  padding: 4px;
}

.mode-task-project-popper .el-dropdown-menu__item {
  padding: 6px 12px;
  line-height: 1.3;
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
