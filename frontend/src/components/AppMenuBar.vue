<template>
  <header
    class="topbar"
    @dblclick="onChromeTitleDblActivate"
    @mousedown="onChromeTitlePointerDown"
  >
    <div class="brand">
      <span class="brand-mark" aria-hidden="true">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M13.2 2 3 14h6.5l-1 8L20 10h-6.5l-.3-8z" />
        </svg>
      </span>
      FlashShell
    </div>

    <template v-if="activeView !== 'home'">
      <span class="tb-sep" aria-hidden="true" />
      <div class="tb-context">{{ contextText }}</div>
    </template>

    <div class="tb-spacer" />

    <div class="tb-right">
      <AppSegmented
        v-if="showModeSeg"
        :model-value="segValue"
        :options="segOptions"
        aria-label="任务与终端"
        @update:model-value="onSegChange"
      />
      <span v-if="showModeSeg" class="tb-sep" aria-hidden="true" />

      <AppIconBtn title="新建窗口" :aria-label="`新建窗口 (${newWindowLabel})`" @click="onNewWindow">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" aria-hidden="true">
          <rect x="5" y="4" width="14" height="16" rx="2" /><path d="M3 9h2M3 15h2M19 9h2M19 15h2" />
        </svg>
      </AppIconBtn>

      <AppIconBtn title="配置文件" aria-label="配置文件" @click.stop="toggleConfigMenu">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" aria-hidden="true">
          <path d="M6 3h9l4 4v14H6z" /><path d="M14 3v5h5" />
        </svg>
      </AppIconBtn>

      <AppIconBtn title="系统设置" aria-label="系统设置" @click="openSettings">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <circle cx="12" cy="12" r="3" /><path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M19.1 4.9L17 7M7 17l-2.1 2.1" />
        </svg>
      </AppIconBtn>

      <AppIconBtn title="关于 FlashShell" aria-label="关于" @click="openAbout">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" aria-hidden="true">
          <circle cx="12" cy="12" r="9" /><path d="M12 11v5M12 8h.01" />
        </svg>
      </AppIconBtn>

      <WindowControls v-if="!isMac" />
    </div>

    <div v-show="configMenuOpen" class="dropdown config-dropdown" @click.stop>
      <div class="dd-label">业务配置文件</div>
      <template v-if="configFiles.length">
        <button
          v-for="file in configFiles"
          :key="file"
          type="button"
          class="dd-item"
          :class="{ active: file === currentConfig }"
          @click="onConfigCommand(`switch:${file}`)"
        >
          <svg
            v-if="file === currentConfig"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.4"
            aria-hidden="true"
          >
            <path d="M5 13l4 4L19 7" />
          </svg>
          {{ basename(file) }}
        </button>
      </template>
      <button v-else type="button" class="dd-item" disabled>无法加载配置文件</button>
      <div class="dd-sep" />
      <button type="button" class="dd-item" @click="onConfigCommand('edit-pipeline')">编辑任务流水线</button>
      <button type="button" class="dd-item" @click="onConfigCommand('reload')">刷新</button>
      <button type="button" class="dd-item" @click="onConfigCommand('refresh')">刷新配置列表</button>
      <div class="dd-sep" />
      <button type="button" class="dd-item" @click="onConfigCommand('open-global')">打开全局配置</button>
      <button type="button" class="dd-item" @click="onConfigCommand('open-current')">打开当前配置</button>
    </div>
  </header>
</template>

<script>
import { computed, onMounted, onUnmounted, ref, h } from 'vue'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { mergeShortcuts, formatShortcut } from '../utils/shortcuts'
import { isMacPlatform } from '../utils/platform'
import { onChromeTitleDblActivate, onChromeTitlePointerDown } from '../utils/windowChrome'
import { useConfigFileMenu } from '../composables/useConfigFileMenu'
import WindowControls from './WindowControls.vue'
import { AppIconBtn } from './ui'
import AppSegmented from './ui/AppSegmented.vue'

const TaskIcon = {
  render() {
    return h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }, [
      h('rect', { x: '3', y: '4', width: '18', height: '16', rx: '3' }),
      h('path', { d: 'M3 9h18M8 4v5' }),
    ])
  },
}

const ShellIcon = {
  render() {
    return h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }, [
      h('path', { d: 'M5 7l4 5-4 5M11 17h6' }),
    ])
  },
}

export default {
  name: 'AppMenuBar',
  components: { AppIconBtn, AppSegmented, WindowControls },
  props: {
    activeView: {
      type: String,
      default: 'home',
      validator: (v) => ['home', 'task', 'shell'].includes(v),
    },
    hasProjects: { type: Boolean, default: false },
    hasTask: { type: Boolean, default: false },
    taskRunning: { type: Boolean, default: false },
    connectedCount: { type: Number, default: 0 },
    openSessionCount: { type: Number, default: 0 },
    selectedProjectName: { type: String, default: '' },
    currentConfigLabel: { type: String, default: '' },
  },
  emits: ['change-view', 'open-config-editor', 'refresh'],
  setup(props, { emit }) {
    const isMac = isMacPlatform()
    const shortcuts = ref(mergeShortcuts())

    const {
      configFiles,
      currentConfig,
      configMenuOpen,
      basename,
      toggleConfigMenu,
      closeConfigMenu,
      onConfigCommand: runConfigCommand,
    } = useConfigFileMenu({
      onEditPipeline: () => emit('open-config-editor'),
      onReload: () => emit('refresh'),
    })

    const loadShortcuts = async () => {
      try {
        shortcuts.value = mergeShortcuts(await App.GetShortcutSettings())
      } catch {
        shortcuts.value = mergeShortcuts()
      }
    }

    const newWindowLabel = computed(() => formatShortcut(shortcuts.value.newWindow))

    const showModeSeg = computed(() => props.hasProjects || props.hasTask || props.openSessionCount > 0 || props.connectedCount > 0)

    const segValue = computed(() => {
      if (props.activeView === 'task' || props.activeView === 'shell') return props.activeView
      return ''
    })

    const segOptions = computed(() => [
      { value: 'task', label: '任务', icon: TaskIcon },
      {
        value: 'shell',
        label: '终端',
        icon: ShellIcon,
        dot: true,
        dotActive: (props.connectedCount || props.openSessionCount) > 0,
      },
    ])

    const contextText = computed(() => {
      if (props.activeView === 'task' && props.selectedProjectName) {
        return `${props.selectedProjectName} · 任务输出`
      }
      if (props.activeView === 'shell') {
        const n = props.connectedCount || props.openSessionCount
        if (n > 0) return `Shell 模式 · ${n} 个会话已连接`
        return 'Shell 模式'
      }
      const cfg = props.currentConfigLabel || basename(currentConfig.value)
      if (cfg) return `${cfg} · 任务与 Shell 并行`
      return '任务与 Shell 并行'
    })

    const onSegChange = (view) => {
      if (view === 'task' || view === 'shell') {
        emit('change-view', view)
      }
    }

    const onConfigCommand = (cmd) => runConfigCommand(cmd)

    const onNewWindow = () => App.NewWindow()
    const openSettings = () => App.OpenSystemSettings()
    const openAbout = () => App.OpenAbout()

    let offShortcutsChanged = null

    onMounted(() => {
      loadShortcuts()
      offShortcutsChanged = EventsOn('shortcuts:changed', (data) => {
        shortcuts.value = mergeShortcuts(data)
      })
      document.addEventListener('click', closeConfigMenu)
    })

    onUnmounted(() => {
      offShortcutsChanged?.()
      offShortcutsChanged = null
      document.removeEventListener('click', closeConfigMenu)
    })

    return {
      isMac,
      contextText,
      showModeSeg,
      segValue,
      segOptions,
      configFiles,
      currentConfig,
      configMenuOpen,
      basename,
      newWindowLabel,
      onChromeTitleDblActivate,
      onChromeTitlePointerDown,
      toggleConfigMenu,
      onConfigCommand,
      onSegChange,
      onNewWindow,
      openSettings,
      openAbout,
    }
  },
}
</script>

<style scoped>
.topbar {
  -webkit-app-region: drag;
  position: relative;
}

.topbar button,
.topbar .tb-right,
.topbar .dropdown {
  -webkit-app-region: no-drag;
}

.config-dropdown {
  position: absolute;
  top: 44px;
  right: 96px;
  min-width: 200px;
  z-index: 200;
}

.dd-item svg {
  width: 12px;
  height: 12px;
  flex-shrink: 0;
}

.dd-item:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

@media (max-width: 720px) {
  .tb-context {
    display: none;
  }

  .topbar .tb-sep:first-of-type {
    display: none;
  }
}
</style>
