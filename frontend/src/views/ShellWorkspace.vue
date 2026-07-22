<template>
  <div class="shell-workspace">
    <el-container class="shell-body main-container">
      <el-aside
        v-if="showLeftPanel"
        :width="leftCollapsed ? '0px' : leftPanelWidth + 'px'"
        class="left-panel shell-left-panel"
        :class="{ collapsed: leftCollapsed, resizing: isResizing }"
      >
        <ShellMonitorPanel
          v-if="monitorSession && !leftCollapsed"
          :active-machine="monitorMachineName"
          :active-connected="monitorConnected"
          :connecting="monitorConnecting"
          @toggle-connection="onToggleConnection"
        />
        <LocalFileTreePanel
          v-else-if="localFileSession && !leftCollapsed"
        />
        <!-- 右边框：拖拽改宽 + 悬停显示收起 -->
        <div
          v-if="!leftCollapsed"
          class="panel-edge-wrap"
          @mouseenter="edgeHover = true"
          @mouseleave="edgeHover = false"
        >
          <div class="resize-handle" @mousedown="$emit('start-resize', $event)"></div>
          <button
            v-show="edgeHover"
            type="button"
            class="panel-edge-btn"
            title="收起侧栏"
            @mousedown.stop
            @click.stop="toggleLeftPanel"
          >
            <el-icon><DArrowLeft /></el-icon>
          </button>
        </div>
      </el-aside>

      <!-- 收起后：贴左边悬停出现展开按钮 -->
      <div
        v-if="showLeftPanel && leftCollapsed"
        class="left-edge-hotzone"
        @mouseenter="edgeHover = true"
        @mouseleave="edgeHover = false"
      >
        <button
          v-show="edgeHover"
          type="button"
          class="panel-edge-btn panel-edge-btn--expand"
          title="展开侧栏"
          @click="toggleLeftPanel"
        >
          <el-icon><DArrowRight /></el-icon>
        </button>
      </div>

      <el-main class="terminal-container shell-terminal-container">
        <ShellTerminalTabs
          ref="tabsRef"
          class="shell-tabs-area"
          :sessions="workspaceSessions"
          :active-machine="activeMachine"
          :search-query="searchQuery"
          :view-visible="active"
          :transfer-active-count="transferActiveCount"
          :broadcast-enabled="broadcastEnabled"
          :broadcast-targets="broadcastTargets"
          :split-session-ids="splitSessionIds"
          @update:active-machine="(name) => $emit('update:activeMachine', name)"
          @update:broadcast-enabled="(v) => $emit('update:broadcast-enabled', v)"
          @update:broadcast-targets="(v) => $emit('update:broadcast-targets', v)"
          @update:split-session-ids="(v) => $emit('update:split-session-ids', v)"
          @close-session="(name) => $emit('close-session', name)"
          @close-sessions="(names) => $emit('close-sessions', names)"
          @reconnect="onReconnect"
          @clear="onClear"
          @open-picker="() => openPicker()"
          @add-local="onAddLocal"
          @back="$emit('back')"
          @open-search="openSearch"
          @search-result="onSearchResult"
          @open-transfer="transferVisible = true"
          @open-command-palette="commandPaletteVisible = true"
          @cwd-sync="onCwdSync"
          @reorder-tabs="(payload) => $emit('reorder-tabs', payload)"
        >
          <template #empty>
            <div v-if="connectingName" class="app-empty shell-connecting">
              <el-icon class="is-loading" :size="28"><Loading /></el-icon>
              <p class="app-empty-title">正在连接 {{ connectingName }}…</p>
            </div>
            <ShellConnectionHistory
              v-else
              :records="historyRecords"
              :sessions="sessions"
              @connect="onHistoryConnect"
              @open-picker="(tab) => openPicker(tab)"
              @clear="clearHistory"
              @remove="removeHistory"
              @back="$emit('back')"
            />
          </template>
          <template #footer="{ activeMachine: am }">
            <ShellFilePanel
              v-if="am && !isLocalSessionName(am)"
              ref="filePanelRef"
              :machine-name="am"
              :cwd-hint="cwdHints[am] || ''"
              :search-visible="searchVisible"
              v-model:search-query="searchQuery"
              :match-summary="searchMatchSummary"
              @layout-change="onFilePanelLayout"
              @cwd-change="(dir) => onPanelCwdChange(am, dir)"
              @clear="clearTerminal"
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
      :active-tab-label="activeTabLabel"
      :tunnels="tunnelStatuses"
      :tunnel-loading="tunnelLoading"
      :app-info="appInfo"
      @open-tunnels="tunnelDialogVisible = true"
    />

    <ShellTunnelDialog
      v-model="tunnelDialogVisible"
      :session-id="activeMachine"
      :config-name="activeConfigName"
      @changed="loadTunnels"
    />

    <ShellCommandPalette
      v-model="commandPaletteVisible"
      :session-id="activeMachine"
      :config-name="activeConfigName"
      @insert="onCommandPaletteInsert"
    />

    <ShellTransferPanel
      v-model="transferVisible"
      @active-change="(n) => { transferActiveCount = n }"
    />

    <ShellMachinePickerDialog
      v-model="pickerVisible"
      :machines="machines"
      :sessions="sessions"
      :workspace-sessions="workspaceSessions"
      :connecting-name="connectingName"
      :history-records="historyRecords"
      :initial-tab="pickerInitialTab"
      @connect="onPickerConnect"
      @edit-machine="(m) => $emit('edit-machine', m)"
      @copy-machine="(m) => $emit('copy-machine', m)"
      @delete-machine="(m) => $emit('delete-machine', m)"
      @add-machine="$emit('add-machine')"
      @add-local="onPickerAddLocal"
      @clear-history="clearHistory"
      @remove-history="removeHistory"
      @open="loadHistory"
    />
  </div>
</template>

<script>
import { ref, reactive, computed, nextTick, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { DArrowLeft, DArrowRight } from '@element-plus/icons-vue'
import ShellMonitorPanel from '../components/shell/ShellMonitorPanel.vue'
import ShellTerminalTabs from '../components/shell/ShellTerminalTabs.vue'
import ShellStatusBar from '../components/shell/ShellStatusBar.vue'
import ShellConnectionHistory from '../components/shell/ShellConnectionHistory.vue'
import ShellMachinePickerDialog from '../components/shell/ShellMachinePickerDialog.vue'
import ShellFilePanel from '../components/shell/ShellFilePanel.vue'
import LocalFileTreePanel from '../components/shell/LocalFileTreePanel.vue'
import ShellTransferPanel from '../components/shell/ShellTransferPanel.vue'
import ShellTunnelDialog from '../components/shell/ShellTunnelDialog.vue'
import ShellCommandPalette from '../components/shell/ShellCommandPalette.vue'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { remoteConfigName, buildKnownMachineNames } from '../utils/sessionId'
import {
  normalizeSnippets,
  findMatchingSnippet,
  buildSnippetPayload,
  isFormFieldTarget,
} from '../utils/shortcuts'

export default {
  name: 'ShellWorkspace',
  components: {
    DArrowLeft,
    DArrowRight,
    ShellMonitorPanel,
    ShellTerminalTabs,
    ShellStatusBar,
    ShellConnectionHistory,
    ShellMachinePickerDialog,
    ShellFilePanel,
    LocalFileTreePanel,
    ShellTransferPanel,
    ShellTunnelDialog,
    ShellCommandPalette,
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
    broadcastEnabled: { type: Boolean, default: false },
    broadcastTargets: { type: Array, default: () => [] },
    splitSessionIds: { type: Array, default: () => [] },
    /** 系统设置等上层弹层打开时，禁用片段快捷键 */
    blockShortcuts: { type: Boolean, default: false },
  },
  emits: [
    'back', 'connect', 'disconnect', 'close-session', 'close-sessions', 'reconnect', 'test', 'add-machine', 'edit-machine',
    'copy-machine', 'delete-machine',
    'add-local', 'start-resize', 'update:activeMachine', 'history-changed',
    'update:broadcast-enabled', 'update:broadcast-targets', 'update:split-session-ids',
    'reorder-tabs', 'machines-changed',
  ],
  setup(props, { emit }) {
    const tabsRef = ref(null)
    const filePanelRef = ref(null)
    const leftCollapsed = ref(false)
    const edgeHover = ref(false)
    const searchVisible = ref(false)
    const searchQuery = ref('')
    const searchMatchSummary = ref('')
    const pickerVisible = ref(false)
    const pickerInitialTab = ref('')
    const transferVisible = ref(false)
    const transferActiveCount = ref(0)
    const historyRecords = ref([])
    const cwdHints = reactive({})
    const ptyCwds = reactive({})
    const tunnelStatuses = ref([])
    const tunnelLoading = ref(false)
    const tunnelDialogVisible = ref(false)
    const commandPaletteVisible = ref(false)
    const snippetList = ref([])
    let tunnelTimer = null

    const loadSnippetList = async () => {
      try {
        const data = await App.GetShortcutSettings()
        snippetList.value = normalizeSnippets(data?.snippets)
      } catch {
        snippetList.value = []
      }
    }

    const onShortcutsChanged = (data) => {
      if (Array.isArray(data?.snippets)) {
        snippetList.value = normalizeSnippets(data.snippets)
      }
      loadSnippetList()
    }

    const knownMachineNames = computed(() => buildKnownMachineNames(props.machines))

    const resolveRemoteConfigName = (sessionID) =>
      remoteConfigName(sessionID, knownMachineNames.value)

    const activeSession = computed(() =>
      (props.workspaceSessions || []).find((s) => s.machineName === props.activeMachine),
    )

    const activeTabLabel = computed(() =>
      activeSession.value?.tabLabel || props.activeMachine || '',
    )

    const activeConfigName = computed(() => {
      const s = activeSession.value
      if (!s || isLocalSessionName(s.machineName)) return ''
      return s.configName || resolveRemoteConfigName(s.machineName)
    })

    const loadTunnels = async () => {
      const cfg = activeConfigName.value
      if (!cfg || !activeConnected.value) {
        tunnelStatuses.value = []
        tunnelLoading.value = false
        return
      }
      tunnelLoading.value = true
      try {
        tunnelStatuses.value = (await App.GetShellTunnelStatus(cfg)) || []
      } catch {
        tunnelStatuses.value = []
      } finally {
        tunnelLoading.value = false
      }
    }

    const isLocalSessionName = (name) => {
      const n = String(name || '')
      return n === 'local' || n.startsWith('local-')
    }

    const showMonitorPanel = computed(() => !!monitorSession.value)

    const showLeftPanel = computed(() => !!monitorSession.value || !!localFileSession.value)

    const localFileSession = computed(() => {
      const sessions = props.workspaceSessions || []
      const active = String(props.activeMachine || '')
      if (!active) return null
      const tab = sessions.find((s) => s.machineName === active)
      if (!tab) return null
      if (tab.kind !== 'local' && !isLocalSessionName(tab.machineName)) return null
      if (isPendingSessionName(tab.machineName)) return null
      return tab
    })

    const isPendingSessionName = (name) => String(name || '').startsWith('__pending__')

    const monitorSession = computed(() => {
      const sessions = props.workspaceSessions || []
      const active = String(props.activeMachine || '')
      if (!active) return null
      const tab = sessions.find((s) => s.machineName === active)
      if (!tab) return null
      if (tab.kind === 'local' || isLocalSessionName(tab.machineName)) return null
      return tab
    })

    const monitorMachineName = computed(() => {
      const tab = monitorSession.value
      if (!tab) return ''
      if (String(tab.machineName).startsWith('__pending__')) {
        return tab.configName || tab.tabLabel || ''
      }
      return tab.machineName
    })

    const monitorConnected = computed(() => {
      const tab = monitorSession.value
      return !!(tab?.connected && !tab?.connecting)
    })

    const monitorConnecting = computed(() => !!monitorSession.value?.connecting)

    const activeConnected = computed(() => {
      const name = props.activeMachine
      if (!name) return false
      return (props.workspaceSessions || []).some((s) => s.machineName === name && s.connected)
    })

    watch(showLeftPanel, async () => {
      await nextTick()
      tabsRef.value?.fitActive?.()
    })

    const findMonitorMachineRecord = () => {
      const key = String(monitorMachineName.value || activeConfigName.value || '').trim()
      if (!key) return null
      return (props.machines || []).find((m) => m?.name === key || m?.id === key) || null
    }

    const isMachineMonitorOpen = (m) => m?.shellMonitorOpen !== false

    const syncLeftCollapsedFromMachine = () => {
      if (!monitorSession.value) return
      leftCollapsed.value = !isMachineMonitorOpen(findMonitorMachineRecord())
    }

    watch(
      [
        monitorMachineName,
        activeConfigName,
        monitorSession,
        () => {
          const m = findMonitorMachineRecord()
          return m ? isMachineMonitorOpen(m) : true
        },
      ],
      () => {
        syncLeftCollapsedFromMachine()
      },
      { immediate: true },
    )

    watch(localFileSession, (cur, prev) => {
      if (cur && !prev && !monitorSession.value) {
        leftCollapsed.value = false
      }
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

    watch([activeConfigName, activeConnected], () => {
      loadTunnels()
    })

    onMounted(() => {
      loadHistory()
      loadSnippetList()
      EventsOn('shell:cwd', onShellCwd)
      EventsOn('shortcuts:changed', onShortcutsChanged)
      loadTunnels()
      tunnelTimer = setInterval(loadTunnels, 5000)
    })

    onUnmounted(() => {
      EventsOff('shell:cwd')
      EventsOff('shortcuts:changed')
      if (tunnelTimer) clearInterval(tunnelTimer)
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

    const toggleLeftPanel = async () => {
      leftCollapsed.value = !leftCollapsed.value
      edgeHover.value = false
      const open = !leftCollapsed.value
      const m = findMonitorMachineRecord()
      if (m?.id || m?.name) {
        try {
          await App.SetMachineShellMonitorOpen(m.id || m.name, open)
          emit('machines-changed')
        } catch (e) {
          console.warn('保存监控栏展开状态失败:', e)
        }
      }
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
      emit('reconnect', name || props.activeMachine)
    }

    const onAddLocal = () => {
      emit('add-local')
    }

    const onPickerAddLocal = () => {
      pickerVisible.value = false
      emit('add-local')
    }

    const openPicker = (tab = '') => {
      pickerInitialTab.value = tab === 'history' || tab === 'machines' ? tab : ''
      pickerVisible.value = true
    }

    const onToggleConnection = () => {
      const name = props.activeMachine
      if (!name) return
      if (activeConnected.value) {
        emit('disconnect', name)
      } else {
        emit('reconnect', name)
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

    const onCommandPaletteInsert = async (text) => {
      if (!props.activeMachine || !text) return
      try {
        await App.SendShellInput(props.activeMachine, text)
      } catch (e) {
        ElMessage.error('发送失败: ' + e)
      }
    }

    const recordInputHistory = async (text) => {
      const cmd = String(text || '').replace(/\r?\n/g, '').trim()
      if (!cmd) return
      const scope = activeConfigName.value || 'global'
      try {
        await App.RecordShellCommandHistory(scope, cmd)
      } catch {
        // ignore
      }
    }

    /** 代码片段快捷键：向当前会话（或广播目标）写入字符串 */
    const sendMappedInput = async (text) => {
      if (!text) return false
      try {
        // 未换行执行的插入也记入历史
        if (!String(text).includes('\n')) {
          await recordInputHistory(text)
        }
        if (props.broadcastEnabled) {
          const targets = (props.broadcastTargets || []).filter(Boolean)
          if (!targets.length) return false
          await App.BroadcastShellInput(targets, text)
          return true
        }
        if (!props.activeMachine) return false
        await App.SendShellInput(props.activeMachine, text)
        return true
      } catch (e) {
        ElMessage.error('发送失败: ' + e)
        return false
      }
    }

    /** 片段快捷键：Shell 视图内处理；shortcuts:changed 后立即用最新列表 */
    const onSnippetHotkey = (e) => {
      if (!props.active || props.blockShortcuts) return
      if (commandPaletteVisible.value || tunnelDialogVisible.value || pickerVisible.value) return
      if (isFormFieldTarget(e.target)) return
      if (!snippetList.value.length) return
      const matched = findMatchingSnippet(e, snippetList.value)
      if (!matched) return
      const payload = buildSnippetPayload(matched)
      if (payload == null || payload === '') return
      e.preventDefault()
      e.stopImmediatePropagation()
      void sendMappedInput(payload)
    }

    onMounted(() => {
      document.addEventListener('keydown', onSnippetHotkey, true)
    })
    onUnmounted(() => {
      document.removeEventListener('keydown', onSnippetHotkey, true)
    })

    const openCommandPalette = () => {
      if (!props.workspaceSessions?.length) return
      commandPaletteVisible.value = true
    }

    const pasteClipboard = async () => {
      await tabsRef.value?.pasteClipboard?.()
    }

    return {
      tabsRef,
      filePanelRef,
      leftCollapsed,
      edgeHover,
      searchVisible,
      searchQuery,
      searchMatchSummary,
      pickerVisible,
      pickerInitialTab,
      openPicker,
      transferVisible,
      transferActiveCount,
      historyRecords,
      cwdHints,
      activeConnected,
      showMonitorPanel,
      showLeftPanel,
      localFileSession,
      monitorSession,
      monitorMachineName,
      monitorConnected,
      monitorConnecting,
      isLocalSessionName,
      clearTerminal,
      onClear,
      toggleSearch,
      openSearch,
      closeSearch,
      findNext,
      findPrevious,
      toggleLeftPanel,
      onHistoryConnect,
      onPickerConnect,
      onReconnect,
      onAddLocal,
      onPickerAddLocal,
      onToggleConnection,
      onSearchResult,
      clearHistory,
      removeHistory,
      onPanelCwdChange,
      onFilePanelLayout,
      onCwdSync,
      loadHistory,
      activeTabLabel,
      activeConfigName,
      tunnelStatuses,
      tunnelLoading,
      tunnelDialogVisible,
      commandPaletteVisible,
      openCommandPalette,
      pasteClipboard,
      onCommandPaletteInsert,
      sendMappedInput,
      loadTunnels,
    }
  },
}
</script>

<style scoped>
.shell-workspace {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--app-bg);
}

.shell-connecting {
  min-height: 240px;
  gap: 14px;
}

.shell-body {
  flex: 1;
  min-height: 0;
  height: 100%;
  overflow: hidden;
}

.left-edge-hotzone {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 28px; /* 避开底部状态栏 */
  width: 10px;
  z-index: 20;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  pointer-events: auto;
}

.left-edge-hotzone:hover {
  width: 28px;
}

.panel-edge-wrap {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  width: 10px;
  z-index: 20;
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.panel-edge-wrap .resize-handle {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: auto;
  width: 5px; /* 与 SFTP .height-handle 同宽感 */
  height: auto;
  background: transparent;
  cursor: col-resize;
  z-index: 1;
}

/* 拖动手柄轻微高亮 */
.shell-left-panel.resizing .panel-edge-wrap .resize-handle,
.panel-edge-wrap:hover .resize-handle,
.panel-edge-wrap .resize-handle:hover,
.panel-edge-wrap .resize-handle:active {
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 8%, transparent);
}

.panel-edge-btn {
  position: relative;
  z-index: 2;
  width: 18px;
  height: 52px;
  border: 1px solid var(--app-border);
  border-radius: 8px 0 0 8px;
  background: var(--app-panel-bg);
  color: var(--app-text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  box-shadow: -2px 0 8px color-mix(in srgb, #000 12%, transparent);
}

.panel-edge-btn--expand {
  border-radius: 0 8px 8px 0;
  border-left: none;
  box-shadow: 2px 0 8px color-mix(in srgb, #000 12%, transparent);
}

.panel-edge-btn:hover {
  color: var(--app-accent-color);
  background: var(--app-card-bg);
  border-color: var(--app-accent-color);
}

.shell-left-panel {
  position: relative;
  overflow: visible;
  transition: width 0.2s ease;
  flex-shrink: 0;
  border-right: 1px solid var(--app-border);
  background-color: var(--app-panel-bg);
}

.shell-left-panel.resizing {
  transition: none;
}

.shell-left-panel :deep(.shell-monitor) {
  overflow: auto;
  height: 100%;
}

.shell-left-panel.collapsed {
  border-right: none;
  min-width: 0 !important;
  overflow: hidden;
}

.shell-terminal-container {
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 0 !important;
  overflow: hidden;
}

.shell-tabs-area {
  flex: 1;
  min-height: 0;
}
</style>
