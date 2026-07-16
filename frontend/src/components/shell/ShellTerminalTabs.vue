<template>
  <div class="shell-terminal-tabs">
    <div
      class="tabs-bar"
      :class="{ 'is-drop-unsplit': draggingSplitPane }"
      @dragover.prevent="onTabsBarDragOver"
      @drop.prevent="onTabsBarDrop"
    >
      <el-button
        class="home-btn"
        size="small"
        text
        title="返回首页"
        @click="$emit('back')"
      >
        <el-icon :size="14"><ArrowLeft /></el-icon>
      </el-button>
      <el-button class="folder-btn" size="small" text title="连接管理器" @click="$emit('open-picker')">
        <el-icon :size="16"><Folder /></el-icon>
      </el-button>

      <div v-if="sessions.length" class="custom-session-tabs">
        <div
          v-for="session in sessions"
          :key="session.machineName"
          class="session-tab"
          :class="{
            active: activeTab === session.machineName,
            'in-split': hasSplitGroup && splitSessionIds.includes(session.machineName),
          }"
          draggable="true"
          @click="selectTab(session.machineName)"
          @dragstart="onTabDragStart($event, session.machineName)"
          @dragend="onTabDragEnd"
        >
          <span class="session-tab-label">{{ tabLabel(session) }}</span>
          <button
            type="button"
            class="session-tab-close"
            title="关闭"
            @click.stop="onTabRemove(session.machineName)"
          >
            ×
          </button>
        </div>
      </div>

      <div class="add-session-wrap">
        <el-button class="add-session-btn" size="small" text title="新建本机" @click="$emit('add-local')">
          <el-icon :size="15"><Plus /></el-icon>
        </el-button>
        <el-dropdown
          trigger="hover"
          :show-timeout="120"
          :hide-timeout="160"
          @command="onAddCommand"
        >
          <el-button class="add-session-more" size="small" text title="更多连接方式">
            <el-icon :size="12"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="local">本机</el-dropdown-item>
              <el-dropdown-item command="remote">远程连接…</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
      <div class="tabs-bar-spacer" aria-hidden="true">
        <span v-if="draggingSplitPane" class="unsplit-hint">拖到此处移出分屏</span>
      </div>
      <el-tooltip
        v-if="sessions.length && connectedCount >= 1"
        :content="broadcastEnabled ? '关闭命令广播 (Esc)' : '开启命令广播'"
        placement="bottom"
      >
        <el-button
          class="broadcast-toggle"
          size="small"
          text
          :class="{ active: broadcastEnabled }"
          @click="toggleBroadcast"
        >
          <el-icon :size="15"><Promotion /></el-icon>
        </el-button>
      </el-tooltip>
      <el-button
        v-if="sessions.length && !isLocalSession(activeTab)"
        class="transfer-btn"
        size="small"
        text
        title="文件传输"
        @click="$emit('open-transfer')"
      >
        <el-badge :value="transferActiveCount" :hidden="!transferActiveCount" :max="99">
          <el-icon :size="15"><Upload /></el-icon>
        </el-badge>
      </el-button>
    </div>

    <ShellBroadcastBar
      v-if="sessions.length && broadcastEnabled"
      :enabled="broadcastEnabled"
      :targets="broadcastTargets"
      :sessions="sessions"
      @update:enabled="(v) => $emit('update:broadcast-enabled', v)"
      @update:targets="(v) => $emit('update:broadcast-targets', v)"
    />

    <div v-if="sessions.length === 0" class="empty-slot">
      <slot name="empty" />
    </div>
    <template v-else>
      <div
        class="terminal-stack"
        :class="{ 'is-split': splitViewVisible, 'is-drag-over': !!draggingTab && !draggingSplitPane }"
        :style="splitGridStyle"
        @dragover.prevent="onStackDragOver"
        @dragleave="onStackDragLeave"
        @drop.prevent="onStackDrop"
      >
        <div v-if="draggingTab && !draggingSplitPane" class="split-drop-overlay">
          <p class="split-drop-hint">拖到区域加入分屏（最多 4 个）</p>
          <div class="split-drop-zones">
            <div
              v-for="zone in dropZones"
              :key="zone.id"
              class="drop-zone"
              :class="[zone.id, { disabled: zone.disabled }]"
              @dragover.prevent="onZoneDragOver(zone)"
              @drop.prevent="onZoneDrop(zone)"
            >
              {{ zone.label }}
            </div>
          </div>
        </div>

        <div
          v-for="session in sessions"
          :key="session.machineName"
          class="terminal-pane"
          :class="{
            'is-active': isTerminalActive(session.machineName),
            'is-split-pane': splitViewVisible && splitSessionIds.includes(session.machineName),
            'is-split-hidden': splitViewVisible && !splitSessionIds.includes(session.machineName),
            'is-focused': activeTab === session.machineName,
          }"
        >
          <div
            v-if="splitViewVisible && splitSessionIds.includes(session.machineName)"
            class="split-pane-header"
            draggable="true"
            @dragstart="onPaneDragStart($event, session.machineName)"
            @dragend="onPaneDragEnd"
            @mousedown="activeTab = session.machineName"
            @contextmenu.prevent="onPaneContextMenu($event, session.machineName)"
          >
            <span class="split-pane-name" :title="tabLabel(session)">{{ tabLabel(session) }}</span>
            <button
              type="button"
              class="split-pane-unsplit"
              title="移出分屏"
              @click.stop="removeFromSplit(session.machineName)"
            >
              ×
            </button>
          </div>
          <ShellTerminal
            :ref="(el) => setTerminalRef(session.machineName, el)"
            :machine-name="session.machineName"
            :connected="!!session.connected"
            :active="isTerminalActive(session.machineName)"
            :view-visible="viewVisible"
            :search-query="searchQuery"
            :broadcast-enabled="broadcastEnabled"
            :in-split="splitViewVisible && splitSessionIds.includes(session.machineName)"
            @open-search="(text) => $emit('open-search', text)"
            @reconnect="(name) => $emit('reconnect', name)"
            @clear-cache="(name) => $emit('clear', name)"
            @search-result="(payload) => $emit('search-result', payload)"
            @cwd-sync="(payload) => $emit('cwd-sync', payload)"
            @remove-from-split="removeFromSplit"
            @exit-split="exitSplit"
            @focus-session="onFocusSession"
          />
        </div>
      </div>

      <ul
        v-if="paneMenu.visible"
        class="pane-ctx-menu"
        :style="{ left: paneMenu.x + 'px', top: paneMenu.y + 'px' }"
        @click.stop
      >
        <li @click="onPaneMenuRemove">移出分屏</li>
        <li @click="onPaneMenuExit">取消全部分屏</li>
      </ul>

      <slot name="footer" :active-machine="activeTab" />
    </template>
  </div>
</template>

<script>
import { ref, reactive, watch, computed, onMounted, onUnmounted } from 'vue'
import { ArrowLeft, ArrowDown, Folder, Upload, Plus, Promotion } from '@element-plus/icons-vue'
import ShellTerminal from './ShellTerminal.vue'
import ShellBroadcastBar from './ShellBroadcastBar.vue'

const MAX_SPLIT = 4

const isLocalSession = (name) => {
  const n = String(name || '')
  return n === 'local' || n.startsWith('local-')
}

const localTabLabel = (name) => {
  if (name === 'local') return '本机'
  const n = String(name || '').replace(/^local-/, '')
  return n ? `本机 ${n}` : '本机'
}

export default {
  name: 'ShellTerminalTabs',
  components: { ShellTerminal, ShellBroadcastBar, ArrowLeft, ArrowDown, Folder, Upload, Plus, Promotion },
  props: {
    sessions: { type: Array, default: () => [] },
    activeMachine: { type: String, default: '' },
    searchQuery: { type: String, default: '' },
    viewVisible: { type: Boolean, default: true },
    transferActiveCount: { type: Number, default: 0 },
    broadcastEnabled: { type: Boolean, default: false },
    broadcastTargets: { type: Array, default: () => [] },
    splitSessionIds: { type: Array, default: () => [] },
  },
  emits: [
    'update:activeMachine', 'close-session', 'clear', 'open-picker', 'add-local',
    'back', 'open-search', 'reconnect', 'search-result', 'open-transfer', 'cwd-sync',
    'update:broadcast-enabled', 'update:broadcast-targets', 'update:split-session-ids',
  ],
  setup(props, { emit, expose }) {
    const activeTab = ref(props.activeMachine)
    const terminalRefs = ref({})
    const draggingTab = ref('')
    const draggingSplitPane = ref('')
    const dropTargetZone = ref('')
    const paneMenu = reactive({ visible: false, x: 0, y: 0, sessionId: '' })

    const hasSplitGroup = computed(() => props.splitSessionIds.length >= 2)
    /** 当前激活 Tab 属于分屏组时才展示分屏；切到组外 Tab 则临时全屏，分屏配置保留 */
    const splitViewVisible = computed(
      () => hasSplitGroup.value && props.splitSessionIds.includes(activeTab.value),
    )

    watch(() => props.activeMachine, (val) => {
      if (val) activeTab.value = val
    })

    watch(activeTab, (val) => {
      emit('update:activeMachine', val)
      nextTickFit(val)
    })

    watch(splitViewVisible, (visible) => {
      if (visible) {
        setTimeout(() => {
          props.splitSessionIds.forEach((id) => terminalRefs.value[id]?.fitAndResize?.())
        }, 80)
      } else if (activeTab.value) {
        setTimeout(() => terminalRefs.value[activeTab.value]?.fitAndResize?.(), 80)
      }
    })

    const nextTickFit = (name) => {
      setTimeout(() => {
        if (splitViewVisible.value) {
          props.splitSessionIds.forEach((id) => terminalRefs.value[id]?.fitAndResize?.())
        } else {
          terminalRefs.value[name]?.fitAndResize?.()
        }
      }, 40)
    }

    watch(
      () => props.splitSessionIds,
      (ids) => {
        if (ids.length === 1) {
          emit('update:split-session-ids', [])
        }
      },
    )

    const hidePaneMenu = () => {
      paneMenu.visible = false
      paneMenu.sessionId = ''
    }

    const onDocClick = () => hidePaneMenu()
    onMounted(() => document.addEventListener('click', onDocClick))
    onUnmounted(() => document.removeEventListener('click', onDocClick))

    const setTerminalRef = (name, el) => {
      if (el) terminalRefs.value[name] = el
      else delete terminalRefs.value[name]
    }

    const tabLabel = (session) => {
      const base = session?.tabLabel
        || (session?.kind === 'local' || isLocalSession(session?.machineName)
          ? localTabLabel(session.machineName)
          : (session?.configName || session?.machineName))
      if (!session?.connected) return `${base} (未连接)`
      return base
    }

    const exitSplit = () => {
      emit('update:split-session-ids', [])
      hidePaneMenu()
      setTimeout(() => terminalRefs.value[activeTab.value]?.fitAndResize?.(), 80)
    }

    const removeFromSplit = (sessionId) => {
      hidePaneMenu()
      const id = sessionId || paneMenu.sessionId
      if (!id) return
      const next = props.splitSessionIds.filter((x) => x !== id)
      if (next.length < 2) {
        emit('update:split-session-ids', [])
        if (id) activeTab.value = id
      } else {
        emit('update:split-session-ids', next)
        if (activeTab.value === id) activeTab.value = next[0]
      }
      setTimeout(() => {
        const ids = next.length >= 2 ? next : [activeTab.value]
        ids.forEach((sid) => terminalRefs.value[sid]?.fitAndResize?.())
      }, 80)
    }

    /** 点击 Tab / 聚焦分屏窗格：同步左侧监控与底部栏 */
    const selectTab = (name) => {
      activeTab.value = name
    }

    const onFocusSession = (name) => {
      if (!name || activeTab.value === name) return
      activeTab.value = name
    }

    const onTabRemove = (name) => {
      emit('close-session', name)
    }

    const onAddCommand = (cmd) => {
      if (cmd === 'remote') emit('open-picker')
      else emit('add-local')
    }

    const isTerminalActive = (name) => {
      if (splitViewVisible.value) {
        return props.splitSessionIds.includes(name)
      }
      return activeTab.value === name
    }

    const connectedCount = computed(() =>
      (props.sessions || []).filter((s) => s.connected).length,
    )

    const toggleBroadcast = () => {
      const next = !props.broadcastEnabled
      emit('update:broadcast-enabled', next)
      if (next && !props.broadcastTargets.length) {
        const ids = (props.sessions || []).filter((s) => s.connected).map((s) => s.machineName)
        emit('update:broadcast-targets', ids)
      }
    }

    const splitGridStyle = computed(() => {
      if (!splitViewVisible.value) return {}
      const n = props.splitSessionIds.length
      if (n === 2) return { gridTemplateColumns: '1fr 1fr', gridTemplateRows: '1fr' }
      return { gridTemplateColumns: '1fr 1fr', gridTemplateRows: '1fr 1fr' }
    })

    const dropZones = computed(() => {
      const full = props.splitSessionIds.length >= MAX_SPLIT
      return [
        { id: 'left', label: '左侧', disabled: full },
        { id: 'right', label: '右侧', disabled: full },
        { id: 'top', label: '上方', disabled: full },
        { id: 'bottom', label: '下方', disabled: full },
      ]
    })

    const onTabDragStart = (e, sessionId) => {
      draggingTab.value = sessionId
      draggingSplitPane.value = ''
      e.dataTransfer.effectAllowed = 'move'
      e.dataTransfer.setData('text/plain', sessionId)
      e.dataTransfer.setData('application/x-flashdock-tab', sessionId)
    }

    const onTabDragEnd = () => {
      draggingTab.value = ''
      dropTargetZone.value = ''
    }

    const onPaneDragStart = (e, sessionId) => {
      draggingSplitPane.value = sessionId
      draggingTab.value = ''
      e.dataTransfer.effectAllowed = 'move'
      e.dataTransfer.setData('text/plain', sessionId)
      e.dataTransfer.setData('application/x-flashdock-unsplit', sessionId)
    }

    const onPaneDragEnd = () => {
      draggingSplitPane.value = ''
    }

    const onTabsBarDragOver = (e) => {
      if (draggingSplitPane.value || e.dataTransfer?.types?.includes?.('application/x-flashdock-unsplit')) {
        e.dataTransfer.dropEffect = 'move'
      }
    }

    const onTabsBarDrop = (e) => {
      const id = draggingSplitPane.value
        || e.dataTransfer.getData('application/x-flashdock-unsplit')
        || e.dataTransfer.getData('text/plain')
      if (id && props.splitSessionIds.includes(id)) {
        removeFromSplit(id)
      }
      draggingSplitPane.value = ''
    }

    const onStackDragOver = () => {
      if (draggingTab.value) dropTargetZone.value = 'stack'
    }

    const onStackDragLeave = (e) => {
      if (!e.currentTarget.contains(e.relatedTarget)) {
        dropTargetZone.value = ''
      }
    }

    const buildSplitIds = (draggedId, zoneId) => {
      const dragged = draggedId || draggingTab.value
      if (!dragged) return null

      let ids = [...props.splitSessionIds]
      const inSplit = ids.includes(dragged)

      if (ids.length >= MAX_SPLIT && !inSplit) return null

      if (ids.length === 0) {
        const anchor = activeTab.value && activeTab.value !== dragged ? activeTab.value : ''
        if (zoneId === 'left' || zoneId === 'top') {
          ids = anchor ? [dragged, anchor] : [dragged]
        } else {
          ids = anchor ? [anchor, dragged] : [dragged]
        }
      } else if (!inSplit) {
        if (zoneId === 'left' || zoneId === 'top') ids.unshift(dragged)
        else ids.push(dragged)
      }

      ids = [...new Set(ids)].slice(0, MAX_SPLIT)
      if (ids.length < 2) {
        const other = props.sessions.find((s) => s.machineName !== dragged)?.machineName
        if (other && !ids.includes(other)) ids.push(other)
      }
      return ids.length >= 2 ? ids : null
    }

    const applySplit = (ids) => {
      if (!ids || ids.length < 2) return
      emit('update:split-session-ids', ids)
      if (!ids.includes(activeTab.value)) activeTab.value = ids[0]
      draggingTab.value = ''
      dropTargetZone.value = ''
      setTimeout(() => {
        ids.forEach((id) => terminalRefs.value[id]?.fitAndResize?.())
      }, 80)
    }

    const onZoneDragOver = (zone) => {
      if (!zone.disabled) dropTargetZone.value = zone.id
    }

    const onZoneDrop = (zone) => {
      if (zone.disabled) return
      const ids = buildSplitIds(draggingTab.value, zone.id)
      applySplit(ids)
    }

    const onStackDrop = () => {
      if (draggingSplitPane.value) return
      const ids = buildSplitIds(draggingTab.value, 'right')
      applySplit(ids)
    }

    const onPaneContextMenu = (e, sessionId) => {
      if (!splitViewVisible.value || !props.splitSessionIds.includes(sessionId)) return
      paneMenu.sessionId = sessionId
      paneMenu.x = e.clientX
      paneMenu.y = e.clientY
      paneMenu.visible = true
      activeTab.value = sessionId
    }

    const onPaneMenuRemove = () => removeFromSplit(paneMenu.sessionId)
    const onPaneMenuExit = () => exitSplit()

    const clearActive = () => {
      terminalRefs.value[activeTab.value]?.clear?.()
      emit('clear', activeTab.value)
    }

    const getActiveTerminal = () => terminalRefs.value[activeTab.value]

    const emptyResult = () => ({ found: false, resultIndex: -1, resultCount: 0 })
    const findNext = () => getActiveTerminal()?.findNext?.() ?? emptyResult()
    const findPrevious = () => getActiveTerminal()?.findPrevious?.() ?? emptyResult()
    const clearSearch = () => getActiveTerminal()?.clearSearch?.()
    const fitActive = () => {
      if (splitViewVisible.value) {
        props.splitSessionIds.forEach((id) => terminalRefs.value[id]?.fitAndResize?.())
        return
      }
      getActiveTerminal()?.fitAndResize?.()
    }
    const getSelection = () => getActiveTerminal()?.getSelection?.() || ''

    expose({ clearActive, findNext, findPrevious, clearSearch, fitActive, getSelection })

    return {
      activeTab,
      draggingTab,
      draggingSplitPane,
      dropZones,
      paneMenu,
      hasSplitGroup,
      splitViewVisible,
      setTerminalRef,
      onTabRemove,
      clearActive,
      tabLabel,
      onAddCommand,
      isLocalSession,
      isTerminalActive,
      splitGridStyle,
      selectTab,
      onFocusSession,
      onTabDragStart,
      onTabDragEnd,
      onPaneDragStart,
      onPaneDragEnd,
      onTabsBarDragOver,
      onTabsBarDrop,
      onStackDragOver,
      onStackDragLeave,
      onStackDrop,
      onZoneDragOver,
      onZoneDrop,
      onPaneContextMenu,
      onPaneMenuRemove,
      onPaneMenuExit,
      removeFromSplit,
      exitSplit,
      toggleBroadcast,
      connectedCount,
    }
  },
}
</script>

<style scoped>
.shell-terminal-tabs {
  flex: 1;
  min-height: 0;
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.tabs-bar {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  background: var(--app-panel-bg);
  border-bottom: 1px solid var(--app-border);
  padding: 0 4px 0 4px;
  min-height: 36px;
}

.tabs-bar.is-drop-unsplit {
  outline: 1px dashed var(--app-accent-color);
  outline-offset: -2px;
  background: color-mix(in srgb, var(--app-accent-color) 8%, var(--app-panel-bg));
}

.unsplit-hint {
  font-size: 11px;
  color: var(--app-accent-color);
  white-space: nowrap;
}

.custom-session-tabs {
  display: flex;
  align-items: center;
  gap: 2px;
  flex: 0 1 auto;
  max-width: calc(100% - 160px);
  min-width: 0;
  overflow-x: auto;
  scrollbar-width: thin;
}

.session-tab {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 32px;
  padding: 0 8px 0 12px;
  font-size: 12px;
  color: var(--app-text-secondary);
  border-radius: 6px 6px 0 0;
  cursor: grab;
  user-select: none;
  flex-shrink: 0;
  max-width: 180px;
}

.session-tab:active {
  cursor: grabbing;
}

.session-tab.active {
  color: var(--app-accent-color);
  background: var(--app-card-bg);
}

.session-tab.in-split {
  box-shadow: inset 0 -2px 0 var(--app-accent-color);
}

.session-tab-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-tab-close {
  border: none;
  background: transparent;
  color: var(--app-text-muted);
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  padding: 0 2px;
}

.session-tab-close:hover {
  color: var(--app-danger-color, #f56c6c);
}

.folder-btn {
  flex-shrink: 0;
  margin: 0 2px;
}

.add-session-wrap {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  margin-left: 2px;
}

.add-session-btn {
  color: var(--app-text-secondary);
  padding: 4px 6px;
}

.add-session-more {
  color: var(--app-text-secondary);
  padding: 4px 4px;
  margin-left: -2px;
}

.add-session-btn:hover,
.add-session-more:hover {
  color: var(--app-accent-color);
}

.home-btn {
  flex-shrink: 0;
  margin-left: 0;
  color: var(--app-text-secondary);
  padding: 4px 10px;
}

.broadcast-toggle {
  flex-shrink: 0;
  color: var(--app-text-secondary);
  padding: 4px 8px;
}

.broadcast-toggle:hover,
.broadcast-toggle.active {
  color: var(--app-accent-color);
}

.broadcast-toggle.active {
  background: var(--app-accent-bg);
  border-radius: var(--app-radius-sm, 6px);
}

.transfer-btn {
  flex-shrink: 0;
  margin-left: 0;
  color: var(--app-text-secondary);
  padding: 4px 10px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.transfer-btn:hover {
  color: var(--app-accent-color);
}

.transfer-btn :deep(.el-badge__content) {
  transform: translateY(-2px) translateX(4px);
}

.home-btn:hover {
  color: var(--app-accent-color);
}

.tabs-bar-spacer {
  flex: 1;
  min-width: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.empty-slot {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.terminal-stack {
  flex: 1;
  min-height: 0;
  width: 100%;
  position: relative;
}

.terminal-stack.is-split {
  display: grid;
  gap: 2px;
  background: var(--app-border);
  position: relative;
}

.terminal-pane {
  position: absolute;
  inset: 0;
  visibility: hidden;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  background: var(--terminal-bg, #1e1e1e);
}

.terminal-pane.is-active {
  visibility: visible;
  z-index: 1;
}

.terminal-stack.is-split .terminal-pane.is-split-pane {
  position: relative;
  inset: auto;
  visibility: visible;
  z-index: 1;
}

.terminal-stack.is-split .terminal-pane.is-split-hidden {
  display: none;
}

.terminal-pane.is-focused.is-split-pane {
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--app-accent-color) 55%, transparent);
}

.split-pane-header {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  height: 26px;
  padding: 0 6px 0 10px;
  background: var(--app-panel-bg);
  border-bottom: 1px solid var(--app-border);
  cursor: grab;
  user-select: none;
}

.split-pane-header:active {
  cursor: grabbing;
}

.split-pane-name {
  flex: 1;
  min-width: 0;
  font-size: 11px;
  font-weight: 600;
  color: var(--app-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.terminal-pane.is-focused .split-pane-name {
  color: var(--app-accent-color);
}

.split-pane-unsplit {
  flex-shrink: 0;
  border: none;
  background: transparent;
  color: var(--app-text-muted);
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  padding: 0 4px;
  border-radius: 4px;
}

.split-pane-unsplit:hover {
  color: var(--app-danger-color, #f56c6c);
  background: color-mix(in srgb, var(--app-danger-color, #f56c6c) 12%, transparent);
}

.terminal-pane :deep(.shell-terminal) {
  flex: 1;
  min-height: 0;
  min-width: 0;
  position: relative;
}

.pane-ctx-menu {
  position: fixed;
  z-index: 3000;
  margin: 0;
  padding: 4px 0;
  list-style: none;
  min-width: 140px;
  background: var(--app-card-bg);
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius-md, 8px);
  box-shadow: 0 8px 24px color-mix(in srgb, #000 18%, transparent);
}

.pane-ctx-menu li {
  padding: 8px 14px;
  font-size: 12px;
  color: var(--app-text);
  cursor: pointer;
}

.pane-ctx-menu li:hover {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}

.split-drop-overlay {
  position: absolute;
  inset: 0;
  z-index: 20;
  background: color-mix(in srgb, var(--app-accent-color) 12%, transparent);
  backdrop-filter: blur(1px);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  pointer-events: none;
}

.split-drop-hint {
  margin: 0;
  font-size: 13px;
  color: var(--app-text);
  font-weight: 500;
}

.split-drop-zones {
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-template-rows: 1fr 1fr;
  gap: 8px;
  width: min(420px, 80%);
  height: min(220px, 60%);
  pointer-events: auto;
}

.drop-zone {
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px dashed var(--app-accent-color);
  border-radius: 8px;
  background: color-mix(in srgb, var(--app-card-bg) 85%, transparent);
  color: var(--app-accent-color);
  font-size: 13px;
  font-weight: 500;
}

.drop-zone:hover:not(.disabled) {
  background: color-mix(in srgb, var(--app-accent-color) 18%, var(--app-card-bg));
}

.drop-zone.disabled {
  opacity: 0.35;
  pointer-events: none;
}
</style>
