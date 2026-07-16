<template>
  <div class="shell-machine-list">
    <div class="panel-header">
      <h3>机器列表</h3>
      <el-button size="small" type="primary" text @click="$emit('add-machine')">
        <el-icon><Plus /></el-icon>
        添加
      </el-button>
      <el-button size="small" type="primary" text @click="$emit('back')">
        <el-icon><ArrowLeft /></el-icon>
        返回
      </el-button>
    </div>

    <div class="list-body">
      <MachineConnectList
        :machines="machines"
        :sessions="sessions"
        :connecting-name="connectingName"
        empty-text="暂无机器，请先添加"
        @connect="(name) => $emit('connect', name)"
      />
    </div>
  </div>
</template>

<script>
import MachineConnectList from './MachineConnectList.vue'

export default {
  name: 'ShellMachineList',
  components: { MachineConnectList },
  props: {
    machines: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
    activeMachine: { type: String, default: '' },
    connectingName: { type: String, default: '' },
    testingName: { type: String, default: '' },
  },
  emits: ['back', 'connect', 'disconnect', 'test', 'add-machine', 'select-machine'],
}
</script>

<style scoped>
.shell-machine-list {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-width: 0;
  padding: 8px 10px;
  box-sizing: border-box;
  background: var(--app-panel-bg);
  color: var(--app-text);
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-bottom: 8px;
  flex-shrink: 0;
  min-width: 0;
}

.panel-header h3 {
  margin: 0;
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.panel-header :deep(.el-button) {
  padding: 4px 6px;
  font-size: 12px;
  margin-left: 0;
}

.list-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
}
</style>
