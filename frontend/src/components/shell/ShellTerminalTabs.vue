<template>
  <div class="shell-terminal-tabs">
    <div class="tabs-bar">
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
          :label="session.machineName"
          :name="session.machineName"
          :closable="true"
        />
      </el-tabs>
      <div v-else class="tabs-placeholder">未连接</div>
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
          :connected="session.connected"
          :active="activeTab === session.machineName"
          :search-query="searchQuery"
          :class="{ 'is-active': activeTab === session.machineName }"
          @cd-hint="(payload) => $emit('cd-hint', payload)"
        />
      </div>
      <slot name="footer" :active-machine="activeTab" />
    </template>
  </div>
</template>

<script>
import { ref, watch } from 'vue'
import ShellTerminal from './ShellTerminal.vue'

export default {
  name: 'ShellTerminalTabs',
  components: { ShellTerminal },
  props: {
    sessions: { type: Array, default: () => [] },
    activeMachine: { type: String, default: '' },
    searchQuery: { type: String, default: '' },
  },
  emits: ['update:activeMachine', 'disconnect', 'clear', 'open-picker', 'cd-hint'],
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
    }

    const onTabRemove = (name) => {
      emit('disconnect', name)
    }

    const clearActive = () => {
      terminalRefs.value[activeTab.value]?.clear?.()
      emit('clear', activeTab.value)
    }

    const getActiveTerminal = () => terminalRefs.value[activeTab.value]

    const findNext = () => getActiveTerminal()?.findNext?.() ?? false
    const findPrevious = () => getActiveTerminal()?.findPrevious?.() ?? false
    const clearSearch = () => getActiveTerminal()?.clearSearch?.()
    const fitActive = () => getActiveTerminal()?.fitAndResize?.()

    expose({ clearActive, findNext, findPrevious, clearSearch, fitActive })

    return { activeTab, setTerminalRef, onTabRemove, clearActive }
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
  padding: 0 4px;
  min-height: 36px;
}

.folder-btn {
  flex-shrink: 0;
  margin: 0 2px;
}

.session-tabs {
  flex: 1;
  min-width: 0;
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
