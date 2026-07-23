<template>
  <div class="app-menu-bar">
    <div class="menu-side menu-left" aria-hidden="true" />

    <div class="menu-center">
      <ModeSwitcher
        v-if="showModeSwitcher"
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
    const showModeSwitcher = computed(() =>
      props.hasProjects
      || props.hasMachines
      || props.activeView === 'task'
      || props.activeView === 'shell'
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
      showModeSwitcher,
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
}

.menu-center {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
}

.menu-icons {
  display: flex;
  align-items: center;
  gap: 2px;
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
