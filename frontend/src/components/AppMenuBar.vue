<template>
  <div
    class="app-menu-bar"
    :class="{
      'is-shell-top': activeView === 'shell',
      'is-task-top': activeView === 'task',
    }"
  >
    <div v-if="activeView !== 'shell'" class="menu-side menu-left" aria-hidden="true" />

    <div v-if="activeView !== 'shell'" class="menu-center">
      <ModeSwitcher
        v-if="showHomeSwitcher"
        :model-value="activeView"
        :has-projects="hasProjects"
        :has-machines="hasMachines"
        :has-task="hasTask"
        :task-running="taskRunning"
        :connected-count="connectedCount"
        :projects="projects"
        :selected-project-name="selectedProjectName"
        @change="$emit('change-view', $event)"
        @select-project="$emit('select-project', $event)"
      />
    </div>

    <div class="menu-side menu-right">
      <div class="menu-icons">
        <div v-if="showCompactSwitcher" class="mode-compact-slot">
          <ModeSwitcher
            compact
            float-align="end"
            :model-value="activeView"
            :has-projects="hasProjects"
            :has-machines="hasMachines"
            :has-task="hasTask"
            :task-running="taskRunning"
            :connected-count="connectedCount"
            :projects="projects"
            :selected-project-name="selectedProjectName"
            @change="$emit('change-view', $event)"
            @select-project="$emit('select-project', $event)"
          />
        </div>

        <button
          type="button"
          class="icon-btn"
          :title="`新建窗口 ${labelOf('newWindow')}`"
          @click="newWindow"
        >
          <el-icon :size="16"><DocumentAdd /></el-icon>
        </button>

        <button
          type="button"
          class="icon-btn"
          title="系统设置"
          @click="openSettings"
        >
          <el-icon :size="16"><Setting /></el-icon>
        </button>

        <button type="button" class="icon-btn" title="帮助" @click="onHelpCommand('about')">
          <el-icon :size="16"><QuestionFilled /></el-icon>
        </button>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { DocumentAdd, Setting, QuestionFilled } from '@element-plus/icons-vue'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { mergeShortcuts, formatShortcut } from '../utils/shortcuts'
import ModeSwitcher from './ModeSwitcher.vue'

export default {
  name: 'AppMenuBar',
  components: { DocumentAdd, Setting, QuestionFilled, ModeSwitcher },
  props: {
    activeView: {
      type: String,
      default: 'home',
      validator: (v) => ['home', 'task', 'shell'].includes(v),
    },
    hasProjects: { type: Boolean, default: false },
    hasMachines: { type: Boolean, default: false },
    hasTask: { type: Boolean, default: false },
    taskRunning: { type: Boolean, default: false },
    connectedCount: { type: Number, default: 0 },
    projects: { type: Array, default: () => [] },
    selectedProjectName: { type: String, default: '' },
  },
  emits: ['change-view', 'select-project'],
  setup(props) {
    const canSwitchModes = computed(() =>
      props.hasProjects || props.hasMachines || props.hasTask || props.connectedCount > 0
    )

    const showHomeSwitcher = computed(() =>
      props.activeView === 'home' && canSwitchModes.value
    )

    const showCompactSwitcher = computed(() =>
      (props.activeView === 'task' || props.activeView === 'shell') && canSwitchModes.value
    )

    const shortcuts = ref(mergeShortcuts())

    const labelOf = (id) => formatShortcut(shortcuts.value[id])

    const loadShortcuts = async () => {
      try {
        shortcuts.value = mergeShortcuts(await App.GetShortcutSettings())
      } catch {
        shortcuts.value = mergeShortcuts()
      }
    }

    const newWindow = () => {
      App.NewWindow()
    }

    const openSettings = () => {
      App.OpenSystemSettings()
    }

    const onHelpCommand = (cmd) => {
      if (cmd === 'about') App.OpenAbout()
    }

    onMounted(() => {
      loadShortcuts()
      EventsOn('shortcuts:changed', (data) => {
        shortcuts.value = mergeShortcuts(data)
      })
    })

    onUnmounted(() => {
      EventsOff('shortcuts:changed')
    })

    return {
      showHomeSwitcher,
      showCompactSwitcher,
      labelOf,
      newWindow,
      openSettings,
      onHelpCommand,
    }
  },
}
</script>

<style scoped>
.app-menu-bar {
  position: relative;
  flex-shrink: 0;
  border-bottom: 1px solid var(--app-border);
  background: var(--app-panel-bg);
  color: var(--app-text);
  height: 40px;
  display: grid;
  grid-template-columns: minmax(96px, 1fr) auto minmax(96px, 1fr);
  align-items: center;
  padding: 0 10px;
  gap: 12px;
  z-index: 30;
}

/* Shell：顶栏改为悬浮叠层，不占高度，会话 Tab 回到监控栏右侧 */
.app-menu-bar.is-shell-top {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 40px;
  margin: 0;
  padding: 0 8px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  grid-template-columns: unset;
  background: transparent;
  border-bottom: none;
  pointer-events: none;
}

.app-menu-bar.is-shell-top .menu-right {
  flex: 0 0 auto;
  pointer-events: auto;
  z-index: 2;
  gap: 6px;
}

.app-menu-bar.is-task-top {
  grid-template-columns: minmax(0, 1fr) auto;
}

.menu-side {
  display: flex;
  align-items: center;
  min-width: 0;
}

.menu-left {
  justify-content: flex-start;
}

.menu-right {
  justify-content: flex-end;
  gap: 6px;
}

.menu-center {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
}

.mode-compact-slot {
  position: relative;
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

/* 并入设置组后，触发器尺寸与旁侧 icon-btn 对齐 */
.menu-icons .mode-compact-slot :deep(.mode-trigger) {
  width: 30px;
  height: 28px;
  min-width: 30px;
  border: none;
  border-radius: 6px;
  background: transparent;
  box-shadow: none;
  opacity: 1;
  color: var(--app-text-secondary, var(--app-text));
}

.menu-icons .mode-compact-slot :deep(.mode-trigger:hover),
.menu-icons .mode-compact-slot :deep(.mode-switcher-host.is-expanded .mode-trigger) {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}

.menu-icons .mode-compact-slot :deep(.mode-trigger-icon) {
  font-size: 16px;
}

.menu-icons {
  display: flex;
  align-items: center;
  gap: 2px;
}

.app-menu-bar.is-shell-top .menu-icons {
  padding: 2px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--app-panel-bg) 92%, transparent);
  border: 1px solid color-mix(in srgb, var(--app-border) 70%, transparent);
  box-shadow: 0 1px 3px color-mix(in srgb, var(--app-text) 6%, transparent);
}

.icon-btn {
  width: 30px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--app-text-secondary, var(--app-text));
  cursor: pointer;
  padding: 0;
  transition: background 0.15s ease, color 0.15s ease;
}

.icon-btn:hover {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}
</style>
