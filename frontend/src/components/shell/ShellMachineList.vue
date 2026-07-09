<template>
  <div class="shell-machine-list">
    <div class="panel-header">
      <el-button size="small" text @click="$emit('back')">
        <el-icon><ArrowLeft /></el-icon>
        返回首页
      </el-button>
      <h3>机器列表</h3>
      <el-button size="small" type="primary" text @click="$emit('add-machine')">
        <el-icon><Plus /></el-icon>
        添加
      </el-button>
    </div>

    <div v-if="machineGroups.length === 0" class="empty-hint">暂无机器，请先添加</div>
    <el-collapse v-else v-model="expandedGroups" class="group-collapse">
      <el-collapse-item
        v-for="group in machineGroups"
        :key="group.name"
        :title="`${group.name} (${group.machines.length})`"
        :name="group.name"
      >
        <div class="machine-list">
          <div
            v-for="machine in group.machines"
            :key="machine.name"
            class="machine-item"
            :class="{ active: isConnected(machine.name) }"
          >
            <div class="machine-info" @click="onSelectMachine(machine.name)">
              <div class="machine-name">{{ machine.name }}</div>
              <div class="machine-meta">{{ machine.key_file || '密钥/密码' }}</div>
            </div>
            <div class="machine-actions">
              <el-button
                v-if="!isConnected(machine.name)"
                size="small"
                type="primary"
                :loading="connectingName === machine.name"
                @click="$emit('connect', machine.name)"
              >
                连接
              </el-button>
              <el-button
                v-else
                size="small"
                type="danger"
                @click="$emit('disconnect', machine.name)"
              >
                断开
              </el-button>
              <el-button
                size="small"
                text
                :loading="testingName === machine.name"
                @click="$emit('test', machine.name)"
              >
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
import { ref, computed, watch } from 'vue'
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
  padding: 12px 16px;
  background: var(--app-panel-bg);
  color: var(--app-text);
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.panel-header h3 {
  margin: 0;
  flex: 1;
  font-size: 14px;
  font-weight: 600;
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
  overflow-y: auto;
  border: none;
}

.group-collapse :deep(.el-collapse-item__header) {
  background: transparent;
  color: var(--app-text);
  border-bottom-color: var(--app-border);
  font-size: 13px;
  font-weight: 500;
}

.group-collapse :deep(.el-collapse-item__wrap) {
  background: transparent;
  border-bottom-color: var(--app-border);
}

.machine-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-bottom: 8px;
}

.machine-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-card-bg);
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
  font-weight: 600;
  font-size: 13px;
}

.machine-meta {
  font-size: 11px;
  color: var(--app-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.machine-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}
</style>
