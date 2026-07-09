<template>
  <div class="shell-workspace">
    <el-container class="shell-body main-container">
      <el-aside :width="leftPanelWidth + 'px'" class="left-panel" :class="{ resizing: isResizing }">
        <div class="resize-handle" @mousedown="$emit('start-resize', $event)"></div>
        <ShellMachineList
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
        />
      </el-aside>

      <el-main class="terminal-container shell-terminal-container">
        <div class="shell-main-header">
          <h3>终端</h3>
          <el-button size="small" :disabled="!activeMachine" @click="clearTerminal">
            <el-icon><Delete /></el-icon>
            清屏
          </el-button>
        </div>
        <ShellTerminalTabs
          ref="tabsRef"
          class="shell-tabs-area"
          :sessions="connectedSessions"
          :active-machine="activeMachine"
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
import { ref } from 'vue'
import ShellMachineList from '../components/shell/ShellMachineList.vue'
import ShellTerminalTabs from '../components/shell/ShellTerminalTabs.vue'
import ShellStatusBar from '../components/shell/ShellStatusBar.vue'
import * as App from '../../wailsjs/go/app/App'

export default {
  name: 'ShellWorkspace',
  components: { ShellMachineList, ShellTerminalTabs, ShellStatusBar },
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
  setup(props, { emit }) {
    const tabsRef = ref(null)

    const clearTerminal = () => {
      tabsRef.value?.clearActive?.()
    }

    const onClear = (machineName) => {
      App.ClearShellOutput(machineName).catch(() => {})
    }

    return { tabsRef, clearTerminal, onClear }
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

.shell-main-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid var(--app-border);
  background: var(--app-panel-bg);
}

.shell-main-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}

.shell-terminal-container {
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
