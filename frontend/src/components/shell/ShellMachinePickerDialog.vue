<template>
  <el-dialog
      v-model="visibleProxy"
      title="连接管理器"
      width="700px"
      class="machine-picker-dialog"
      append-to-body
  >
    <div class="toolbar">
      <el-input v-model="keyword" clearable placeholder="搜索机器" size="small" style="width: 220px"/>
      <el-button size="small" type="primary" @click="$emit('add-machine')">添加机器</el-button>
    </div>

    <div v-if="filteredGroups.length === 0" class="empty">暂无机器配置</div>
    <el-collapse v-else v-model="expanded" class="group-list">
      <el-collapse-item
          v-for="group in filteredGroups"
          :key="group.name"
          :title="`${group.name} (${group.machines.length})`"
          :name="group.name"
      >
        <div
            v-for="machine in group.machines"
            :key="machine.id || machine.name"
            class="machine-row"
        >
          <div class="machine-meta">
            <div class="name">
              {{ machine.name }}
              <el-tag v-if="isConnected(machine.name)" size="small" type="success" effect="plain">已连接</el-tag>
            </div>
            <div class="sub">{{ machine.key_file || '密码/密钥认证' }}</div>
          </div>
          <div class="row-actions">
            <el-button
                size="small"
                type="primary"
                :loading="connectingName === machine.name"
                @click="$emit('connect', machine.name)"
            >
              {{ isConnected(machine.name) ? '聚焦' : '连接' }}
            </el-button>
            <el-button size="small" text title="编辑配置" @click="$emit('edit-machine', machine)">
              <el-icon>
                <Setting/>
              </el-icon>
            </el-button>
          </div>
        </div>
      </el-collapse-item>
    </el-collapse>
  </el-dialog>
</template>

<script>
import {computed, ref, watch} from 'vue'
import {groupMachines} from '../../utils/machineGroups'

export default {
  name: 'ShellMachinePickerDialog',
  props: {
    modelValue: {type: Boolean, default: false},
    machines: {type: Array, default: () => []},
    sessions: {type: Array, default: () => []},
    connectingName: {type: String, default: ''},
  },
  emits: ['update:modelValue', 'connect', 'edit-machine', 'add-machine'],
  setup(props, {emit}) {
    const keyword = ref('')
    const expanded = ref([])

    const visibleProxy = computed({
      get: () => props.modelValue,
      set: (v) => emit('update:modelValue', v),
    })

    const filteredGroups = computed(() => {
      const kw = keyword.value.trim().toLowerCase()
      let list = props.machines || []
      if (kw) {
        list = list.filter((m) =>
            (m.name || '').toLowerCase().includes(kw) ||
            (m.group || '').toLowerCase().includes(kw)
        )
      }
      return groupMachines(list)
    })

    watch(filteredGroups, (groups) => {
      expanded.value = groups.map((g) => g.name)
    }, {immediate: true})

    const isConnected = (name) =>
        (props.sessions || []).some((s) => s.machineName === name && s.connected)

    return {visibleProxy, keyword, expanded, filteredGroups, isConnected}
  },
}
</script>

<style scoped>
.toolbar {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 12px;
}

.empty {
  text-align: center;
  color: var(--app-text-muted);
  padding: 32px 0;
}

.group-list {
  border: none;
  max-height: 480px;
  overflow: auto;
  overflow-x: hidden;
}

.machine-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 4px;
  border-bottom: 1px solid var(--app-border);
}

.machine-row:last-child {
  border-bottom: none;
}

.machine-meta {
  min-width: 0;
  flex: 1;
}

.name {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: var(--app-text);
}

.sub {
  font-size: 12px;
  color: var(--app-text-muted);
  margin-top: 2px;
}

.row-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
</style>
