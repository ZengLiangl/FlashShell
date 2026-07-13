<template>
  <div class="shell-workspace">
    <el-container class="shell-body main-container">
      <el-aside
        v-if="connectedCount > 0"
        :width="leftCollapsed ? '0px' : leftPanelWidth + 'px'"
        class="left-panel shell-left-panel"
        :class="{ collapsed: leftCollapsed, resizing: isResizing }"
      >
        <div v-if="!leftCollapsed" class="resize-handle" @mousedown="$emit('start-resize', $event)"></div>
        <ShellMonitorPanel
          v-show="!leftCollapsed"
          :active-machine="activeMachine"
        />
      </el-aside>

      <el-main class="terminal-container shell-terminal-container">
        <button
          v-if="connectedCount > 0"
          class="panel-expand-btn"
          type="button"
          :title="leftCollapsed ? '展开监控' : '收起监控'"
          @click="toggleLeftPanel"
        >
          <el-icon>
            <DArrowRight v-if="leftCollapsed" />
            <DArrowLeft v-else />
          </el-icon>
        </button>

        <ShellTerminalTabs
          ref="tabsRef"
          class="shell-tabs-area"
          :sessions="connectedSessions"
          :active-machine="activeMachine"
          :search-query="searchQuery"
          @update:active-machine="(name) => $emit('update:activeMachine', name)"
          @disconnect="(name) => $emit('disconnect', name)"
          @clear="onClear"
          @open-picker="pickerVisible = true"
          @cd-hint="onCdHint"
        >
          <template #empty>
            <ShellConnectionHistory
              :records="historyRecords"
              @connect="onHistoryConnect"
              @open-picker="pickerVisible = true"
              @clear="clearHistory"
              @remove="removeHistory"
              @back="$emit('back')"
            />
          </template>
          <template #footer="{ activeMachine: am }">
            <ShellFilePanel
              ref="filePanelRef"
              :machine-name="am"
              :cwd-hint="cwdHints[am] || ''"
              :search-visible="searchVisible"
              v-model:search-query="searchQuery"
              :match-summary="searchMatchSummary"
              @layout-change="onFilePanelLayout"
              @cwd-change="(dir) => onPanelCwdChange(am, dir)"
              @clear="clearTerminal"
              @refresh="refreshSearch"
              @toggle-search="toggleSearch"
              @search-next="findNext"
              @search-prev="findPrevious"
              @close-search="closeSearch"
            />
          </template>
        </ShellTerminalTabs>
      </el-main>
    </el-container>

    <ShellStatusBar
      :connected-count="connectedCount"
      :active-machine="activeMachine"
      :app-info="appInfo"
    />

    <ShellMachinePickerDialog
      v-model="pickerVisible"
      :machines="machines"
      :sessions="sessions"
      :connecting-name="connectingName"
      @connect="onPickerConnect"
      @edit-machine="(m) => $emit('edit-machine', m)"
      @add-machine="$emit('add-machine')"
    />
  </div>
</template>

<script>
import { ref, reactive, nextTick, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import ShellMonitorPanel from '../components/shell/ShellMonitorPanel.vue'
import ShellTerminalTabs from '../components/shell/ShellTerminalTabs.vue'
import ShellStatusBar from '../components/shell/ShellStatusBar.vue'
import ShellConnectionHistory from '../components/shell/ShellConnectionHistory.vue'
import ShellMachinePickerDialog from '../components/shell/ShellMachinePickerDialog.vue'
import ShellFilePanel from '../components/shell/ShellFilePanel.vue'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

export default {
  name: 'ShellWorkspace',
  components: {
    ShellMonitorPanel,
    ShellTerminalTabs,
    ShellStatusBar,
    ShellConnectionHistory,
    ShellMachinePickerDialog,
    ShellFilePanel,
  },
  props: {
    leftPanelWidth: { type: Number, default: 280 },
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
  emits: [
    'back', 'connect', 'disconnect', 'test', 'add-machine', 'edit-machine',
    'start-resize', 'update:activeMachine', 'history-changed',
  ],
  setup(props, { emit }) {
    const tabsRef = ref(null)
    const filePanelRef = ref(null)
    const leftCollapsed = ref(false)
    const searchVisible = ref(false)
    const searchQuery = ref('')
    const searchMatchSummary = ref('')
    const pickerVisible = ref(false)
    const historyRecords = ref([])
    const cwdHints = reactive({})
    const ptyCwds = reactive({})

    const ensurePtyCwd = async (machineName) => {
      if (ptyCwds[machineName]) return ptyCwds[machineName]
      try {
        const home = await App.GetShellRemoteHome(machineName)
        ptyCwds[machineName] = home || '/'
      } catch {
        ptyCwds[machineName] = '/'
      }
      return ptyCwds[machineName]
    }

    const loadHistory = async () => {
      try {
        historyRecords.value = await App.GetShellHistory() || []
      } catch {
        historyRecords.value = []
      }
    }

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
      loadHistory()
      // PTY 真实 cwd（OSC）—— SFTP 跟踪的权威来源
      EventsOn('shell:cwd', onShellCwd)
    })

    onUnmounted(() => {
      document.removeEventListener('keydown', handleKeyDown, true)
      EventsOff('shell:cwd')
    })

    /** 终端真实工作目录变化 → 同步 SFTP */
    const onShellCwd = async (payload) => {
      const machineName = payload?.machineName
      let cwd = payload?.cwd
      if (!machineName || !cwd) return
      if (!String(cwd).startsWith('/')) cwd = `/${cwd}`
      if (cwd.length > 1) cwd = cwd.replace(/\/+$/, '')
      ptyCwds[machineName] = cwd
      cwdHints[machineName] = cwd
      // 只驱动当前活动会话的 SFTP 面板
      if (machineName !== props.activeMachine) return
      await nextTick()
      filePanelRef.value?.applyCwdHint?.(cwd)
    }

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

    const toggleLeftPanel = async () => {
      leftCollapsed.value = !leftCollapsed.value
      await nextTick()
      tabsRef.value?.fitActive?.()
    }

    const onHistoryConnect = async (name) => {
      emit('connect', name)
      await nextTick()
      await loadHistory()
    }

    const onPickerConnect = async (name) => {
      emit('connect', name)
      pickerVisible.value = false
      await nextTick()
      await loadHistory()
    }

    const clearHistory = async () => {
      try {
        await ElMessageBox.confirm('确定清空连接历史？', '确认', { type: 'warning' })
        await App.ClearShellHistory()
        await loadHistory()
      } catch {
        // cancel
      }
    }

    const removeHistory = async (row) => {
      try {
        await App.RemoveShellHistory(row.machineId || '', row.machineName || '')
        await loadHistory()
      } catch (e) {
        ElMessage.error(String(e))
      }
    }

    const onCdHint = async ({ machineName, target }) => {
      // 乐观更新；权威路径以 shell:cwd（每个 prompt 的 OSC）为准
      if (!machineName || target == null) return
      try {
        const base = cwdHints[machineName] || await ensurePtyCwd(machineName)
        const next = await App.ApplyShellCd(machineName, base, target)
        if (next === base) return
        ptyCwds[machineName] = next
        cwdHints[machineName] = next
        await nextTick()
        filePanelRef.value?.applyCwdHint?.(next)
      } catch (e) {
        console.warn('[shell-cd] 乐观同步失败，等待 OSC:', e)
      }
    }

    const onPanelCwdChange = (machineName, dir) => {
      if (!machineName || !dir) return
      const abs = String(dir).startsWith('/') ? dir : `/${dir}`
      ptyCwds[machineName] = abs
      cwdHints[machineName] = abs
    }

    const onFilePanelLayout = async () => {
      await nextTick()
      tabsRef.value?.fitActive?.()
    }

    watch(() => props.connectedSessions, async (sessions) => {
      const alive = new Set((sessions || []).map((s) => s?.machineName).filter(Boolean))
      for (const name of Object.keys(ptyCwds)) {
        if (!alive.has(name)) {
          delete ptyCwds[name]
          delete cwdHints[name]
        }
      }
      for (const s of sessions || []) {
        if (s?.machineName && !ptyCwds[s.machineName]) {
          const home = await ensurePtyCwd(s.machineName)
          // 初始 hint = login home，供 SFTP 面板打开时使用
          if (!cwdHints[s.machineName]) {
            cwdHints[s.machineName] = home
          }
        }
      }
    }, { immediate: true, deep: true })

    watch(() => props.connectedCount, async () => {
      await loadHistory()
      await nextTick()
      tabsRef.value?.fitActive?.()
    })

    watch(() => props.leftPanelWidth, async () => {
      if (leftCollapsed.value) return
      await nextTick()
      tabsRef.value?.fitActive?.()
    })

    return {
      tabsRef,
      filePanelRef,
      leftCollapsed,
      searchVisible,
      searchQuery,
      searchMatchSummary,
      pickerVisible,
      historyRecords,
      cwdHints,
      clearTerminal,
      onClear,
      toggleSearch,
      closeSearch,
      findNext,
      findPrevious,
      refreshSearch,
      toggleLeftPanel,
      onHistoryConnect,
      onPickerConnect,
      clearHistory,
      removeHistory,
      onCdHint,
      onPanelCwdChange,
      onFilePanelLayout,
      loadHistory,
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
  background: var(--app-bg);
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
}

.shell-left-panel .resize-handle:hover,
.shell-left-panel.resizing .resize-handle {
  background: rgba(64, 158, 255, 0.35);
}

.shell-left-panel.collapsed {
  border-right: none;
  min-width: 0 !important;
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
</style>
