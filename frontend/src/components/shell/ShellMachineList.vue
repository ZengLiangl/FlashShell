<template>
  <div class="shell-machine-list">
    <div class="panel-header">
      <h3>机器列表</h3>
      <el-button size="small" type="primary" text @click="$emit('add-machine')">
        <el-icon>
          <Plus />
        </el-icon>
        添加
      </el-button>
      <el-button size="small" type="primary" text @click="$emit('back')">
        <el-icon>
          <ArrowLeft />
        </el-icon>
        返回
      </el-button>
    </div>

    <div v-if="machineGroups.length === 0" class="empty-hint">暂无机器，请先添加</div>
    <el-collapse v-else v-model="expandedGroups" class="group-collapse">
      <el-collapse-item v-for="group in machineGroups" :key="group.name"
        :title="`${group.name} (${group.machines.length})`" :name="group.name">
        <div class="machine-list">
          <div v-for="machine in group.machines" :key="machine.id" class="machine-item"
            :class="{ active: isConnected(machine.name) }">
            <div class="machine-info" @click="onSelectMachine(machine.name)">
              <div class="machine-name">{{ machine.name }}</div>
            </div>
            <div class="machine-actions">
              <el-button v-if="!isConnected(machine.name)" size="small" type="primary"
                :loading="connectingName === machine.name" @click="$emit('connect', machine.name)">
                连接
              </el-button>
              <el-button v-else size="small" type="danger" @click="$emit('disconnect', machine.name)">
                断开
              </el-button>
              <el-button size="small" text :loading="testingName === machine.name" @click="$emit('test', machine.name)">
                测试
              </el-button>
            </div>
          </div>
        </div>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script>
import { computed, ref, watch } from 'vue'
import { groupMachines } from '../../utils/machineGroups'

export default {
  name: 'ShellMachineList',
  props: {
    machines: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
    activeMachine: { type: String, default: '' },
    connectingName: { type: String, default: '' },
    testingName: { type: String, default: '' },
  },
  emits: ['back', 'connect', 'disconnect', 'test', 'add-machine', 'select-machine'],
  setup(props, { emit }) {
    const expandedGroups = ref([])

    const machineGroups = computed(() => groupMachines(props.machines))

    watch(machineGroups, (groups) => {
      expandedGroups.value = groups.map((g) => g.name)
    }, { immediate: true })

    const isConnected = (name) =>
      props.sessions.some((s) => s.machineName === name && s.connected)

    const onSelectMachine = (name) => {
      if (isConnected(name)) {
        emit('select-machine', name)
      }
    }

    return { machineGroups, expandedGroups, isConnected, onSelectMachine }
  },
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
  margin-left: 0px;
}

.empty-hint {
  text-align: center;
  color: var(--app-text-muted);
  padding: 20px 0;
  font-size: 13px;
}

.group-collapse {
  flex: 1;
  min-height: 0;
  min-width: 0;
  overflow-x: hidden;
  overflow-y: auto;
  border: none;
}

.group-collapse :deep(.el-collapse-item__content) {
  padding-bottom: 0;
  overflow: hidden;
}

.group-collapse :deep(.el-collapse-item__header) {
  background: transparent;
  color: var(--app-text);
  border-bottom-color: var(--app-border);
  font-size: 12px;
  font-weight: 500;
  height: 32px;
  line-height: 32px;
  min-height: 32px;
  padding-top: 0;
  padding-bottom: 0;
}

.group-collapse :deep(.el-collapse-item__wrap) {
  background: transparent;
  border-bottom-color: var(--app-border);
}

.machine-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-bottom: 4px;
}

.machine-item {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  min-height: 32px;
  border: 1px solid var(--app-border);
  border-radius: 6px;
  background: var(--app-card-bg);
  min-width: 0;
}

.machine-item.active {
  border-color: #67c23a;
  background: rgba(103, 194, 58, 0.08);
}

.machine-info {
  min-width: 0;
  flex: 1;
  cursor: pointer;
}

.machine-name {
  font-weight: 500;
  font-size: 12px;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.machine-actions {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
  align-items: center;
}

.machine-actions :deep(.el-button) {
  padding: 2px 6px;
  height: 24px;
  font-size: 12px;
}
</style>
