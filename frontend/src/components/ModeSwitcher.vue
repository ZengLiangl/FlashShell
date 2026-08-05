<template>
  <div
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
      <el-icon class="mode-trigger-icon" :size="12"><Grid /></el-icon>
    </button>

    <div
      v-show="showSwitcher"
      class="mode-switcher"
      :class="{ 'is-floating': compact }"
      role="tablist"
      aria-label="工作模式"
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
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item
                v-for="project in projects"
                :key="project.name"
                :command="project.name"
                :class="{ 'is-current': project.name === selectedProjectName }"
              >
                <span class="project-item">
                  <span class="project-name">{{ project.name || '(未命名项目)' }}</span>
                  <span v-if="project.description" class="project-desc">{{ project.description }}</span>
                </span>
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
  </div>
</template>

<script>
import { computed, ref, onUnmounted, watch } from 'vue'
import { HomeFilled, Folder, Monitor, Grid } from '@element-plus/icons-vue'

export default {
  name: 'ModeSwitcher',
  components: { HomeFilled, Folder, Monitor, Grid },
  props: {
    modelValue: {
      type: String,
      default: 'home',
      validator: (v) => ['home', 'task', 'shell'].includes(v),
    },
    /** 收起为触发器，悬停展开（任务 / Shell 顶栏） */
    compact: { type: Boolean, default: false },
    /** compact 展开时对齐：center | end（靠右，避免冲出窗口） */
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
    const hoverOpen = ref(false)
    const taskMenuOpen = ref(false)
    const pointerInside = ref(false)
    let leaveTimer = null

    const showSwitcher = computed(() => !props.compact || hoverOpen.value)

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
      if (props.hasProjects || props.hasTask || props.modelValue === 'task' || canPickProject) {
        list.push({
          id: 'task',
          label: '任务',
          icon: Folder,
          title: props.hasTask
            ? (props.taskRunning
              ? '点击返回任务（运行中），悬停可切换项目'
              : '点击返回任务，悬停可切换项目')
            : (canPickProject ? '悬停选择项目' : '请先在首页选择项目'),
          disabled: !props.hasTask && !canPickProject,
          showProjectPicker: canPickProject,
          dot: props.hasTask && props.taskRunning,
        })
      }
      if (props.hasMachines || props.connectedCount > 0 || props.modelValue === 'shell') {
        list.push({
          id: 'shell',
          label: 'Shell',
          icon: Monitor,
          title: props.connectedCount > 0
            ? `进入 Shell（${props.connectedCount} 会话）`
            : '进入 Shell',
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

    /** 悬浮展开时：当前模块居中（如 Shell 在中间） */
    const displayItems = computed(() => {
      const list = items.value
      if (!props.compact || list.length < 3) return list
      const current = list.find((i) => i.id === props.modelValue)
      if (!current) return list
      const others = list.filter((i) => i.id !== props.modelValue)
      const mid = Math.floor(others.length / 2)
      return [...others.slice(0, mid), current, ...others.slice(mid)]
    })

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
      if (props.compact) hoverOpen.value = true
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
        return
      }
      if (props.compact) scheduleClose(120)
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

    onUnmounted(() => {
      clearLeaveTimer()
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
      items,
      displayItems,
      activeItem,
      triggerTitle,
      showSwitcher,
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
  min-height: 22px;
}

.mode-switcher-host.is-compact.is-expanded {
  z-index: 50;
}

.mode-switcher-host.is-compact.is-float-end.is-expanded {
  /* 保持占位，面板向下展开，避免横向盖住旁侧按钮 */
  position: relative;
}

.mode-trigger {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  min-width: 22px;
  padding: 0;
  border: 1px solid color-mix(in srgb, var(--app-border) 65%, transparent);
  border-radius: 6px;
  background: color-mix(in srgb, var(--app-panel-bg) 82%, transparent);
  color: var(--app-text-muted);
  cursor: pointer;
  opacity: 0.72;
  box-shadow: 0 1px 2px color-mix(in srgb, var(--app-text) 5%, transparent);
  transition:
    background 0.16s ease,
    border-color 0.16s ease,
    color 0.16s ease,
    opacity 0.12s ease,
    box-shadow 0.16s ease;
}

.mode-trigger:hover,
.mode-switcher-host.is-expanded .mode-trigger {
  opacity: 1;
  color: var(--app-accent-color);
  background: var(--app-accent-bg);
  border-color: color-mix(in srgb, var(--app-accent-color) 35%, var(--app-border));
}

.mode-switcher-host.is-compact.is-expanded .mode-trigger {
  opacity: 1;
  pointer-events: auto;
  position: relative;
  color: var(--app-accent-color);
  background: var(--app-accent-bg);
  border-color: color-mix(in srgb, var(--app-accent-color) 35%, var(--app-border));
}

.mode-trigger-icon {
  flex-shrink: 0;
}

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
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  z-index: 60;
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

/* 触发器与下拉面板之间的悬停桥，避免移入时提前收起 */
.mode-switcher.is-floating::before {
  content: '';
  position: absolute;
  top: -10px;
  left: 0;
  right: 0;
  height: 10px;
}

@keyframes mode-float-in {
  from {
    opacity: 0;
    transform: translateY(-4px) scale(0.98);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.mode-switcher :deep(.el-dropdown) {
  min-width: 0;
  width: 100%;
}

.mode-btn {
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
    box-shadow 0.16s ease,
    transform 0.16s ease;
}

.mode-icon {
  flex-shrink: 0;
  opacity: 0.85;
}

.mode-label {
  flex: 0 1 auto;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mode-btn:hover:not(.disabled):not(.active) {
  color: var(--app-accent-color);
  background: var(--app-accent-bg);
}

.mode-btn.active {
  background: var(--app-card-bg, #fff);
  color: var(--app-text);
  font-weight: 600;
  box-shadow:
    0 1px 2px color-mix(in srgb, var(--app-text) 8%, transparent),
    0 0 0 1px color-mix(in srgb, var(--app-border) 75%, transparent);
}

.mode-btn.mode-task.active,
.mode-btn.mode-shell.active {
  color: var(--app-accent-color);
  background: var(--app-card-bg, #fff);
}

.mode-btn.active .mode-icon {
  opacity: 1;
}

.mode-btn.disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.mode-badge {
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

.mode-btn.active .mode-badge {
  background: color-mix(in srgb, var(--mode-shell) 20%, transparent);
}

.mode-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
  background: var(--mode-task);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--mode-task) 24%, transparent);
}

.project-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  max-width: 148px;
}

.project-name {
  font-size: 12px;
  color: var(--app-text);
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-desc {
  font-size: 11px;
  color: var(--app-text-muted);
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:deep(.el-dropdown-menu__item.is-current) {
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

html.dark .mode-btn.active {
  background: color-mix(in srgb, var(--app-card-bg) 92%, #fff);
  box-shadow:
    0 1px 3px rgba(0, 0, 0, 0.35),
    0 0 0 1px color-mix(in srgb, var(--app-border) 80%, transparent);
}

html.dark .mode-trigger {
  background: color-mix(in srgb, var(--app-panel-bg) 70%, transparent);
  opacity: 0.8;
}
</style>

<style>
/* 下拉挂到 body，需非 scoped；收窄任务项目菜单 */
.mode-task-project-popper.el-popper,
.mode-task-project-popper {
  min-width: 0 !important;
  width: max-content;
  max-width: 168px;
}

.mode-task-project-popper .el-dropdown-menu {
  min-width: 0 !important;
  padding: 4px;
}

.mode-task-project-popper .el-dropdown-menu__item {
  padding: 6px 10px;
  line-height: 1.3;
}
</style>
