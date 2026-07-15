<template>
  <div class="shell-terminal-tabs">
    <div class="tabs-bar">
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
      <el-tabs
        v-if="sessions.length"
        v-model="activeTab"
        type="card"
        class="session-tabs"
        @tab-remove="onTabRemove"
      >
        <el-tab-pane
          v-for="session in sessions"
          :key="session.machineName"
          :label="tabLabel(session)"
          :name="session.machineName"
          :closable="true"
        />
      </el-tabs>
      <div v-else class="tabs-placeholder">未连接</div>
      <el-button
        v-if="sessions.length"
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

    <div v-if="sessions.length === 0" class="empty-slot">
      <slot name="empty" />
    </div>
    <template v-else>
      <div class="terminal-stack">
        <ShellTerminal
          v-for="session in sessions"
          :key="session.machineName"
          :ref="(el) => setTerminalRef(session.machineName, el)"
          :machine-name="session.machineName"
          :connected="!!session.connected"
          :active="activeTab === session.machineName"
          :view-visible="viewVisible"
          :search-query="searchQuery"
          :class="{ 'is-active': activeTab === session.machineName, 'is-disconnected': !session.connected }"
          @cd-hint="(payload) => $emit('cd-hint', payload)"
          @open-search="(text) => $emit('open-search', text)"
          @reconnect="(name) => $emit('reconnect', name)"
          @clear-cache="(name) => $emit('clear', name)"
          @search-result="(payload) => $emit('search-result', payload)"
        />
      </div>
      <slot name="footer" :active-machine="activeTab" />
    </template>
  </div>
</template>

<script>
import { ref, watch } from 'vue'
import { ArrowLeft, Folder, Upload } from '@element-plus/icons-vue'
import ShellTerminal from './ShellTerminal.vue'

export default {
  name: 'ShellTerminalTabs',
  components: { ShellTerminal, ArrowLeft, Folder, Upload },
  props: {
    sessions: { type: Array, default: () => [] },
    activeMachine: { type: String, default: '' },
    searchQuery: { type: String, default: '' },
    viewVisible: { type: Boolean, default: true },
    transferActiveCount: { type: Number, default: 0 },
  },
  emits: [
    'update:activeMachine', 'close-session', 'clear', 'open-picker',
    'cd-hint', 'back', 'open-search', 'reconnect', 'search-result', 'open-transfer',
  ],
  setup(props, { emit, expose }) {
    const activeTab = ref(props.activeMachine)
    const terminalRefs = ref({})

    watch(() => props.activeMachine, (val) => {
      if (val) activeTab.value = val
    })

    watch(activeTab, (val) => {
      emit('update:activeMachine', val)
      const ref = terminalRefs.value[val]
      ref?.fitAndResize?.()
    })

    const setTerminalRef = (name, el) => {
      if (el) terminalRefs.value[name] = el
      else delete terminalRefs.value[name]
    }

    const tabLabel = (session) => {
      if (!session?.connected) return `${session.machineName} (未连接)`
      return session.machineName
    }

    const onTabRemove = (name) => {
      emit('close-session', name)
    }

    const clearActive = () => {
      terminalRefs.value[activeTab.value]?.clear?.()
      emit('clear', activeTab.value)
    }

    const getActiveTerminal = () => terminalRefs.value[activeTab.value]

    const emptyResult = () => ({ found: false, resultIndex: -1, resultCount: 0 })
    const findNext = () => getActiveTerminal()?.findNext?.() ?? emptyResult()
    const findPrevious = () => getActiveTerminal()?.findPrevious?.() ?? emptyResult()
    const clearSearch = () => getActiveTerminal()?.clearSearch?.()
    const fitActive = () => getActiveTerminal()?.fitAndResize?.()
    const getSelection = () => getActiveTerminal()?.getSelection?.() || ''

    expose({ clearActive, findNext, findPrevious, clearSearch, fitActive, getSelection })

    return { activeTab, setTerminalRef, onTabRemove, clearActive, tabLabel }
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

.folder-btn {
  flex-shrink: 0;
  margin: 0 2px;
}

.home-btn {
  flex-shrink: 0;
  margin-left: 0;
  color: var(--app-text-secondary);
  padding: 4px 10px;
}

.transfer-btn {
  flex-shrink: 0;
  margin-left: auto;
  color: var(--app-text-secondary);
  padding: 4px 10px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.transfer-btn:hover {
  color: var(--app-accent-color);
}

.transfer-btn-text {
  margin-left: 2px;
}

.transfer-btn :deep(.el-badge__content) {
  transform: translateY(-2px) translateX(4px);
}

.home-btn:hover {
  color: var(--app-accent-color);
}

.home-btn-text {
  margin-left: 2px;
}

.session-tabs {
  flex: 1;
  min-width: 0;
  overflow: hidden;
}

.session-tabs :deep(.el-tabs__header) {
  margin-bottom: 0;
}

.session-tabs :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.tabs-placeholder {
  flex: 1;
  font-size: 12px;
  color: var(--app-text-muted);
  padding-left: 8px;
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

.terminal-stack :deep(.shell-terminal) {
  position: absolute;
  inset: 0;
  visibility: hidden;
}

.terminal-stack :deep(.shell-terminal.is-active) {
  visibility: visible;
  z-index: 1;
}
</style>
