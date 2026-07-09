<template>
  <div class="shell-terminal-tabs">
    <div v-if="sessions.length === 0" class="empty-terminal">
      <p>请从左侧连接机器</p>
    </div>
    <template v-else>
      <el-tabs
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
      <div class="terminal-stack">
        <ShellTerminal
          v-for="session in sessions"
          :key="session.machineName"
          :ref="(el) => setTerminalRef(session.machineName, el)"
          :machine-name="session.machineName"
          :connected="session.connected"
          :active="activeTab === session.machineName"
          :class="{ 'is-active': activeTab === session.machineName }"
        />
      </div>
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
  },
  emits: ['update:activeMachine', 'disconnect', 'clear'],
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

    expose({ clearActive })

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

.empty-terminal {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--app-text-muted);
  background: #0d1117;
}

.session-tabs {
  flex-shrink: 0;
  background: var(--app-panel-bg);
  padding: 0 8px;
}

.session-tabs :deep(.el-tabs__header) {
  margin-bottom: 0;
}

.terminal-stack {
  flex: 1;
  min-height: 0;
  width: 100%;
  height: 100%;
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
