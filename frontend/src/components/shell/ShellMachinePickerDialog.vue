<template>
  <el-dialog
      v-model="visibleProxy"
      title="连接管理器"
      width="700px"
      class="machine-picker-dialog"
      append-to-body
  >
    <div class="toolbar">
      <el-input v-model="keyword" clearable placeholder="搜索名称 / IP" size="small" style="width: 220px">
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
      <el-tooltip content="添加机器" placement="top">
        <el-button size="small" type="primary" circle @click="$emit('add-machine')">
          <el-icon><Plus /></el-icon>
        </el-button>
      </el-tooltip>
    </div>

    <div v-if="!hasTree" class="empty">暂无机器配置</div>
    <div v-else class="machine-tree">
      <div
          v-for="group in customGroups"
          :key="group.name"
          class="tree-group"
      >
        <button
            type="button"
            class="tree-group-head"
            :aria-expanded="isGroupExpanded(group.name)"
            @click="toggleGroup(group.name)"
        >
          <el-icon class="tree-caret" :class="{ open: isGroupExpanded(group.name) }">
            <ArrowRight />
          </el-icon>
          <span class="tree-group-name">{{ group.name }}</span>
          <span class="tree-group-count">{{ group.machines.length }}</span>
        </button>
        <div v-show="isGroupExpanded(group.name)" class="tree-group-body">
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
              <div class="sub">{{ machineHost(machine) }}</div>
            </div>
            <div class="row-actions icon-actions">
              <el-tooltip :content="isConnected(machine.name) ? '聚焦' : '连接'" placement="top">
                <el-button
                  size="small"
                  :type="isConnected(machine.name) ? 'success' : 'primary'"
                  circle
                  :loading="connectingName === machine.name"
                  @click="$emit('connect', machine.name)"
                >
                  <el-icon>
                    <View v-if="isConnected(machine.name)" />
                    <Connection v-else />
                  </el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="编辑配置" placement="top">
                <el-button size="small" circle @click="$emit('edit-machine', machine)">
                  <el-icon><Setting /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </div>
        </div>
      </div>

      <div v-if="defaultMachines.length" class="tree-default">
        <div
            v-for="machine in defaultMachines"
            :key="machine.id || machine.name"
            class="machine-row"
        >
          <div class="machine-meta">
            <div class="name">
              {{ machine.name }}
              <el-tag v-if="isConnected(machine.name)" size="small" type="success" effect="plain">已连接</el-tag>
            </div>
            <div class="sub">{{ machineHost(machine) }}</div>
          </div>
          <div class="row-actions icon-actions">
            <el-tooltip :content="isConnected(machine.name) ? '聚焦' : '连接'" placement="top">
              <el-button
                size="small"
                :type="isConnected(machine.name) ? 'success' : 'primary'"
                circle
                :loading="connectingName === machine.name"
                @click="$emit('connect', machine.name)"
              >
                <el-icon>
                  <View v-if="isConnected(machine.name)" />
                  <Connection v-else />
                </el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="编辑配置" placement="top">
              <el-button size="small" circle @click="$emit('edit-machine', machine)">
                <el-icon><Setting /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<script>
import { computed, ref, watch } from 'vue'
import { ArrowRight } from '@element-plus/icons-vue'
import { splitMachineTree, machineMatchesKeyword } from '../../utils/machineGroups'

export default {
  name: 'ShellMachinePickerDialog',
  components: { ArrowRight },
  props: {
    modelValue: { type: Boolean, default: false },
    machines: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
    connectingName: { type: String, default: '' },
  },
  emits: ['update:modelValue', 'connect', 'edit-machine', 'add-machine'],
  setup(props, { emit }) {
    const keyword = ref('')
    const expandedGroups = ref([])

    const visibleProxy = computed({
      get: () => props.modelValue,
      set: (v) => emit('update:modelValue', v),
    })

    const machineTree = computed(() => {
      const kw = keyword.value
      let list = props.machines || []
      if (String(kw || '').trim()) {
        list = list.filter((m) => machineMatchesKeyword(m, kw))
      }
      return splitMachineTree(list)
    })

    const customGroups = computed(() => machineTree.value.customGroups)
    const defaultMachines = computed(() => machineTree.value.defaultMachines)
    const hasTree = computed(
      () => customGroups.value.length > 0 || defaultMachines.value.length > 0,
    )

    const isGroupExpanded = (name) => expandedGroups.value.includes(name)

    const toggleGroup = (name) => {
      if (isGroupExpanded(name)) {
        expandedGroups.value = expandedGroups.value.filter((g) => g !== name)
      } else {
        expandedGroups.value = [...expandedGroups.value, name]
      }
    }

    watch(keyword, (kw) => {
      if (String(kw || '').trim()) {
        expandedGroups.value = customGroups.value.map((g) => g.name)
      } else {
        expandedGroups.value = []
      }
    })

    // 打开时保持收起
    watch(visibleProxy, (open) => {
      if (open && !String(keyword.value || '').trim()) {
        expandedGroups.value = []
      }
    })

    const isConnected = (name) =>
        (props.sessions || []).some((s) => s.machineName === name && s.connected)

    const machineHost = (machine) => {
      const host = machine?.host || machine?.ip || ''
      if (!host) return machine?.key_file || '密码/密钥认证'
      const port = machine?.port ? `:${machine.port}` : ''
      return `${host}${port}`
    }

    return {
      visibleProxy,
      keyword,
      customGroups,
      defaultMachines,
      hasTree,
      isGroupExpanded,
      toggleGroup,
      isConnected,
      machineHost,
    }
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

.machine-tree {
  max-height: 480px;
  overflow: auto;
  overflow-x: hidden;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.tree-group {
  display: flex;
  flex-direction: column;
}

.tree-group-head {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 8px 6px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: inherit;
  cursor: pointer;
  text-align: left;
}

.tree-group-head:hover {
  background: color-mix(in srgb, var(--app-text-muted) 8%, transparent);
}

.tree-caret {
  font-size: 12px;
  color: var(--app-text-muted);
  transition: transform 0.15s ease;
  flex-shrink: 0;
}

.tree-caret.open {
  transform: rotate(90deg);
}

.tree-group-name {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 650;
  color: var(--app-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tree-group-count {
  flex-shrink: 0;
  min-width: 20px;
  height: 18px;
  padding: 0 6px;
  border-radius: 9px;
  font-size: 11px;
  line-height: 18px;
  text-align: center;
  color: var(--app-text-muted);
  background: color-mix(in srgb, var(--app-text-muted) 12%, transparent);
}

.tree-group-body {
  padding-left: 18px;
}

.tree-default {
  margin-top: 2px;
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
  flex-shrink: 0;
}
</style>
