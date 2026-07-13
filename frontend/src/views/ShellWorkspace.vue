<template>
  <div class="shell-workspace">
    <el-container class="shell-body main-container">
      <el-aside
        :width="leftCollapsed ? '0px' : leftPanelWidth + 'px'"
        class="left-panel shell-left-panel"
        :class="{ collapsed: leftCollapsed, resizing: isResizing }"
      >
        <div v-if="!leftCollapsed" class="resize-handle" @mousedown="$emit('start-resize', $event)"></div>
        <ShellMachineList
          v-show="!leftCollapsed"
          :machines="machines"
          :sessions="sessions"
          :active-machine="activeMachine"
          :connecting-name="connectingName"
          :testing-name="testingName"
          @back="$emit('back')"
          @connect="(name) => $emit('connect', name)"
          @disconnect="(name) => $emit('disconnect', name)"
          @test="(name) => $emit('test', name)"
          @add-machine="$emit('add-machine')"
          @select-machine="(name) => $emit('update:activeMachine', name)"
          @toggle-collapse="leftCollapsed = true"
        />
      </el-aside>

      <el-main class="terminal-container shell-terminal-container">
        <button
          v-if="leftCollapsed"
          class="panel-expand-btn"
          type="button"
          title="展开机器列表"
          @click="expandLeftPanel"
        >
          <el-icon><DArrowRight /></el-icon>
        </button>

        <TerminalHeader
          title="终端"
          :show-back="false"
          :search-visible="searchVisible"
          v-model:search-query="searchQuery"
          :match-summary="searchMatchSummary"
          @clear="clearTerminal"
          @refresh="refreshSearch"
          @toggle-search="toggleSearch"
          @search-next="findNext"
          @search-prev="findPrevious"
          @close-search="closeSearch"
        />
        <ShellTerminalTabs
          ref="tabsRef"
          class="shell-tabs-area"
          :sessions="connectedSessions"
          :active-machine="activeMachine"
          :search-query="searchQuery"
          @update:active-machine="(name) => $emit('update:activeMachine', name)"
          @disconnect="(name) => $emit('disconnect', name)"
          @clear="onClear"
        />
      </el-main>
    </el-container>

    <ShellStatusBar
      :connected-count="connectedCount"
      :active-machine="activeMachine"
      :app-info="appInfo"
    />
  </div>
</template>

<script>
import { ref, nextTick, onMounted, onUnmounted, watch } from 'vue'
import ShellMachineList from '../components/shell/ShellMachineList.vue'
import ShellTerminalTabs from '../components/shell/ShellTerminalTabs.vue'
import ShellStatusBar from '../components/shell/ShellStatusBar.vue'
import TerminalHeader from '../components/TerminalHeader.vue'
import * as App from '../../wailsjs/go/app/App'

export default {
  name: 'ShellWorkspace',
  components: { ShellMachineList, ShellTerminalTabs, ShellStatusBar, TerminalHeader },
  props: {
    leftPanelWidth: { type: Number, default: 400 },
    isResizing: { type: Boolean, default: false },
    appInfo: { type: String, default: '' },
    machines: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
    connectedSessions: { type: Array, default: () => [] },
    connectedCount: { type: Number, default: 0 },
    activeMachine: { type: String, default: '' },
    connectingName: { type: String, default: '' },
    testingName: { type: String, default: '' },
  },
  emits: ['back', 'connect', 'disconnect', 'test', 'add-machine', 'start-resize', 'update:activeMachine'],
  setup(props) {
    const tabsRef = ref(null)
    const leftCollapsed = ref(false)
    const searchVisible = ref(false)
    const searchQuery = ref('')
    const searchMatchSummary = ref('')

    const clearTerminal = () => {
      tabsRef.value?.clearActive?.()
    }

    const onClear = (machineName) => {
      App.ClearShellOutput(machineName).catch(() => {})
    }

    const toggleSearch = () => {
      searchVisible.value = !searchVisible.value
      if (!searchVisible.value) {
        searchQuery.value = ''
        tabsRef.value?.clearSearch?.()
      }
    }

    const openSearch = () => {
      searchVisible.value = true
    }

    const handleKeyDown = (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'f') {
        e.preventDefault()
        e.stopPropagation()
        openSearch()
      }
    }

    onMounted(() => {
      document.addEventListener('keydown', handleKeyDown, true)
    })

    onUnmounted(() => {
      document.removeEventListener('keydown', handleKeyDown, true)
    })

    const closeSearch = () => {
      searchVisible.value = false
      searchQuery.value = ''
      tabsRef.value?.clearSearch?.()
    }

    const findNext = () => {
      const found = tabsRef.value?.findNext?.()
      searchMatchSummary.value = found ? '已匹配' : '未找到'
    }

    const findPrevious = () => {
      const found = tabsRef.value?.findPrevious?.()
      searchMatchSummary.value = found ? '已匹配' : '未找到'
    }

    const refreshSearch = () => {
      if (searchQuery.value.trim()) {
        findNext()
      } else {
        tabsRef.value?.fitActive?.()
      }
    }

    const expandLeftPanel = async () => {
      leftCollapsed.value = false
      await nextTick()
      tabsRef.value?.fitActive?.()
    }

    watch(() => props.leftPanelWidth, async () => {
      if (leftCollapsed.value) return
      await nextTick()
      tabsRef.value?.fitActive?.()
    })

    return {
      tabsRef,
      leftCollapsed,
      searchVisible,
      searchQuery,
      searchMatchSummary,
      clearTerminal,
      onClear,
      toggleSearch,
      closeSearch,
      findNext,
      findPrevious,
      refreshSearch,
      expandLeftPanel,
    }
  },
}
</script>

<style scoped>
.shell-workspace {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.shell-body {
  flex: 1;
  min-height: 0;
  height: 100%;
  overflow: hidden;
}

.shell-left-panel {
  position: relative;
  overflow: hidden;
  transition: width 0.2s ease;
  flex-shrink: 0;
  border-right: 1px solid var(--app-border);
  background-color: var(--app-panel-bg);
}

.shell-left-panel .resize-handle {
  position: absolute;
  top: 0;
  right: -3px;
  width: 6px;
  height: 100%;
  background: transparent;
  cursor: col-resize;
  z-index: 10;
  transition: background-color 0.2s ease;
}

.shell-left-panel .resize-handle:hover {
  background: rgba(64, 158, 255, 0.3);
}

.shell-left-panel .resize-handle:active {
  background: rgba(64, 158, 255, 0.5);
}

.shell-left-panel.resizing {
  user-select: none;
}

.shell-left-panel.resizing .resize-handle {
  background: rgba(64, 158, 255, 0.5);
}

.shell-left-panel.collapsed {
  border-right: none;
  min-width: 0 !important;
}

.shell-left-panel.collapsed .resize-handle {
  display: none;
}

.shell-terminal-container {
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 0 !important;
  overflow: hidden;
}

.panel-expand-btn {
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  z-index: 10;
  width: 18px;
  height: 56px;
  border: 1px solid var(--app-border);
  border-left: none;
  border-radius: 0 8px 8px 0;
  background: var(--app-panel-bg);
  color: var(--app-text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
}

.panel-expand-btn:hover {
  color: var(--app-accent-color);
  background: var(--app-card-bg);
}

.shell-tabs-area {
  flex: 1;
  min-height: 0;
}

.shell-terminal-container :deep(.terminal-header) {
  padding: 8px 12px;
}
</style>
