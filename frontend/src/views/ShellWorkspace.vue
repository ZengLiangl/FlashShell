<template>
  <div class="shell-workspace">
    <el-container class="shell-body main-container">
      <el-aside
        v-if="openSessionCount > 0"
        :width="leftCollapsed ? '0px' : leftPanelWidth + 'px'"
        class="left-panel shell-left-panel"
        :class="{ collapsed: leftCollapsed, resizing: isResizing }"
      >
        <div v-if="!leftCollapsed" class="resize-handle" @mousedown="$emit('start-resize', $event)"></div>
        <ShellMonitorPanel
          v-show="!leftCollapsed"
          :active-machine="activeMachine"
          :active-connected="activeConnected"
          :connecting="connectingName === activeMachine"
          @toggle-connection="onToggleConnection"
        />
      </el-aside>

      <el-main class="terminal-container shell-terminal-container">
        <button
          v-if="openSessionCount > 0"
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
          :sessions="workspaceSessions"
          :active-machine="activeMachine"
          :search-query="searchQuery"
          :view-visible="active"
          :transfer-active-count="transferActiveCount"
          @update:active-machine="(name) => $emit('update:activeMachine', name)"
          @close-session="(name) => $emit('close-session', name)"
          @reconnect="onReconnect"
          @clear="onClear"
          @open-picker="pickerVisible = true"
          @back="$emit('back')"
          @open-search="openSearch"
          @search-result="onSearchResult"
          @open-transfer="transferVisible = true"
          @cwd-sync="onCwdSync"
        >
          <template #empty>
            <div v-if="connectingName" class="shell-connecting">
              <el-icon class="is-loading" :size="28"><Loading /></el-icon>
              <p>正在连接 {{ connectingName }}…</p>
            </div>
            <ShellConnectionHistory
              v-else
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

    <ShellTransferPanel
      v-model="transferVisible"
      @active-change="(n) => { transferActiveCount = n }"
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
import { ref, reactive, computed, nextTick, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import ShellMonitorPanel from '../components/shell/ShellMonitorPanel.vue'
import ShellTerminalTabs from '../components/shell/ShellTerminalTabs.vue'
import ShellStatusBar from '../components/shell/ShellStatusBar.vue'
import ShellConnectionHistory from '../components/shell/ShellConnectionHistory.vue'
import ShellMachinePickerDialog from '../components/shell/ShellMachinePickerDialog.vue'
import ShellFilePanel from '../components/shell/ShellFilePanel.vue'
import ShellTransferPanel from '../components/shell/ShellTransferPanel.vue'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { mergeShortcuts, matchesShortcut } from '../utils/shortcuts'

export default {
  name: 'ShellWorkspace',
  components: {
    ShellMonitorPanel,
    ShellTerminalTabs,
    ShellStatusBar,
    ShellConnectionHistory,
    ShellMachinePickerDialog,
    ShellFilePanel,
    ShellTransferPanel,
  },
  props: {
    active: { type: Boolean, default: true },
    leftPanelWidth: { type: Number, default: 280 },
    isResizing: { type: Boolean, default: false },
    appInfo: { type: String, default: '' },
    machines: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
    workspaceSessions: { type: Array, default: () => [] },
    connectedCount: { type: Number, default: 0 },
    openSessionCount: { type: Number, default: 0 },
    activeMachine: { type: String, default: '' },
    connectingName: { type: String, default: '' },
    testingName: { type: String, default: '' },
  },
  emits: [
    'back', 'connect', 'disconnect', 'close-session', 'test', 'add-machine', 'edit-machine',
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
    const transferVisible = ref(false)
    const transferActiveCount = ref(0)
    const historyRecords = ref([])
    const cwdHints = reactive({})
    const ptyCwds = reactive({})

    const activeConnected = computed(() => {
      const name = props.activeMachine
      if (!name) return false
      return (props.workspaceSessions || []).some((s) => s.machineName === name && s.connected)
    })

    const formatSearchSummary = (result) => {
      if (!result) return '未找到'
      const total = Number(result.resultCount) || 0
      if (!total) return '未找到'
      const idx = Number(result.resultIndex)
      if (idx < 0) return `${total}+`
      return `${idx + 1}/${total}`
    }

    const onSearchResult = (result) => {
      if (!searchVisible.value) return
      searchMatchSummary.value = formatSearchSummary(result)
    }

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
      if (searchVisible.value) {
        closeSearch()
        return
      }
      openSearch()
    }

    const openSearch = (prefill) => {
      searchVisible.value = true
      // Vue 事件参数：右键查找会传入选中文本；快捷键无参时从终端读取选区
      let text = ''
      if (typeof prefill === 'string') {
        text = prefill
      } else if (Array.isArray(prefill)) {
        text = prefill[0] || ''
      }
      if (!text) {
        text = tabsRef.value?.getSelection?.() || ''
      }
      if (text) {
        searchQuery.value = text
      }
      nextTick(() => {
        filePanelRef.value?.focusSearch?.()
        if (searchQuery.value.trim()) {
          const result = tabsRef.value?.findNext?.()
          searchMatchSummary.value = formatSearchSummary(result)
        }
      })
    }

    const findShortcut = ref(mergeShortcuts().find)

    const handleKeyDown = (e) => {
      if (!props.active) return
      if (matchesShortcut(e, findShortcut.value)) {
        e.preventDefault()
        e.stopPropagation()
        openSearch()
      }
    }

    const loadFindShortcut = async () => {
      try {
        findShortcut.value = mergeShortcuts(await App.GetShortcutSettings()).find
      } catch {
        findShortcut.value = mergeShortcuts().find
      }
    }

    onMounted(() => {
      document.addEventListener('keydown', handleKeyDown, true)
      loadHistory()
      loadFindShortcut()
      EventsOn('shell:cwd', onShellCwd)
      EventsOn('shortcuts:changed', (data) => {
        findShortcut.value = mergeShortcuts(data).find
      })
    })

    onUnmounted(() => {
      document.removeEventListener('keydown', handleKeyDown, true)
      EventsOff('shell:cwd')
      EventsOff('shortcuts:changed')
    })

    /** 终端 cd 后同步 SFTP（直接驱动面板，不依赖 shell:cwd 事件） */
    const onCwdSync = async (payload) => {
      const machineName = payload?.machineName
      let cwd = payload?.cwd
      if (!machineName || !cwd) return
      cwd = String(cwd).trim()
      if (!cwd.startsWith('/')) return
      if (cwd.length > 1) cwd = cwd.replace(/\/+$/, '')
      ptyCwds[machineName] = cwd
      cwdHints[machineName] = cwd
      if (machineName !== props.activeMachine) return
      await nextTick()
      filePanelRef.value?.applyCwdHint?.(cwd)
    }

    /** 终端真实工作目录变化（OSC 777/7）→ 同步 SFTP */
    const onShellCwd = async (payload) => {
      const machineName = payload?.machineName
      let cwd = payload?.cwd
      if (!machineName || !cwd) return
      cwd = String(cwd).trim()
      const marker = '777;cwd;'
      const idx = cwd.indexOf(marker)
      if (idx >= 0) cwd = cwd.slice(idx + marker.length)
      cwd = cwd.replace(/\x07/g, '')
      const esc = cwd.indexOf('\x1b')
      if (esc >= 0) cwd = cwd.slice(0, esc)
      if (!cwd.startsWith('/') || cwd.includes('777;cwd') || cwd.includes(']')) return
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
      searchMatchSummary.value = ''
    }

    const findNext = () => {
      const result = tabsRef.value?.findNext?.()
      searchMatchSummary.value = formatSearchSummary(result)
    }

    const findPrevious = () => {
      const result = tabsRef.value?.findPrevious?.()
      searchMatchSummary.value = formatSearchSummary(result)
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

    const onReconnect = (name) => {
      emit('connect', name || props.activeMachine)
    }

    const onToggleConnection = () => {
      const name = props.activeMachine
      if (!name) return
      if (activeConnected.value) {
        emit('disconnect', name)
      } else {
        emit('connect', name)
      }
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

    const onPanelCwdChange = (_machineName, _dir) => {
      // SFTP 面板手动浏览不应覆盖终端 cwd 缓存，避免与 shell:cwd 同步冲突
    }

    const onFilePanelLayout = async () => {
      await nextTick()
      tabsRef.value?.fitActive?.()
    }

    watch(() => props.activeMachine, async (name) => {
      if (!name) return
      const hint = cwdHints[name] || ptyCwds[name]
      if (!hint) return
      await nextTick()
      filePanelRef.value?.applyCwdHint?.(hint)
    })

    watch(() => props.workspaceSessions, async (sessions) => {
      const alive = new Set((sessions || []).map((s) => s?.machineName).filter(Boolean))
      for (const name of Object.keys(ptyCwds)) {
        if (!alive.has(name)) {
          delete ptyCwds[name]
          delete cwdHints[name]
        }
      }
      for (const s of sessions || []) {
        if (s?.machineName && s.connected && !ptyCwds[s.machineName]) {
          const home = await ensurePtyCwd(s.machineName)
          if (!cwdHints[s.machineName]) {
            cwdHints[s.machineName] = home
          }
        }
      }
    }, { immediate: true, deep: true })

    watch(() => props.openSessionCount, async () => {
      await loadHistory()
      await nextTick()
      tabsRef.value?.fitActive?.()
    })

    watch(() => props.leftPanelWidth, async () => {
      if (leftCollapsed.value) return
      await nextTick()
      tabsRef.value?.fitActive?.()
    })

    watch(() => props.active, async (visible) => {
      if (!visible) return
      await nextTick()
      tabsRef.value?.fitActive?.()
    })

    watch(searchQuery, (q) => {
      if (!searchVisible.value) return
      if (!q.trim()) {
        searchMatchSummary.value = ''
        tabsRef.value?.clearSearch?.()
        return
      }
      // 输入时由终端 watch 触发 find；summary 在下一帧由 findNext 结果更新
      nextTick(() => {
        const result = tabsRef.value?.findNext?.()
        searchMatchSummary.value = formatSearchSummary(result)
      })
    })

    return {
      tabsRef,
      filePanelRef,
      leftCollapsed,
      searchVisible,
      searchQuery,
      searchMatchSummary,
      pickerVisible,
      transferVisible,
      transferActiveCount,
      historyRecords,
      cwdHints,
      activeConnected,
      clearTerminal,
      onClear,
      toggleSearch,
      openSearch,
      closeSearch,
      findNext,
      findPrevious,
      refreshSearch,
      toggleLeftPanel,
      onHistoryConnect,
      onPickerConnect,
      onReconnect,
      onToggleConnection,
      onSearchResult,
      clearHistory,
      removeHistory,
      onPanelCwdChange,
      onFilePanelLayout,
      onCwdSync,
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

.shell-connecting {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  min-height: 240px;
  color: var(--app-text-secondary);
  font-size: 14px;
}

.shell-connecting p {
  margin: 0;
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
