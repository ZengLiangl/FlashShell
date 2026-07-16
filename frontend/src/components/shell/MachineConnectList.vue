<template>
  <div v-if="!hasTree" class="mcl-empty">{{ emptyText }}</div>
  <div v-else class="mcl-root">
    <div
      v-for="group in customGroups"
      :key="group.name"
      class="mcl-group"
    >
      <button
        type="button"
        class="mcl-group-head"
        :aria-expanded="isGroupExpanded(group.name)"
        @click="toggleGroup(group.name)"
      >
        <el-icon class="mcl-caret" :class="{ open: isGroupExpanded(group.name) }">
          <ArrowRight />
        </el-icon>
        <span class="mcl-group-name">{{ group.name }}</span>
        <span class="mcl-group-count">{{ group.machines.length }}</span>
      </button>
      <ul
        v-show="isGroupExpanded(group.name)"
        class="mcl-list"
        role="listbox"
      >
        <li
          v-for="machine in group.machines"
          :key="machine.id || machine.name"
          class="mcl-item"
          :class="{
            connected: isConnected(machine.name),
            connecting: connectingName === machine.name,
          }"
          role="option"
          tabindex="0"
          @click="onConnect(machine)"
          @keydown.enter.prevent="onConnect(machine)"
        >
          <div class="mcl-dot" aria-hidden="true" />
          <div class="mcl-body">
            <div class="mcl-line">
              <span class="mcl-name">{{ machine.name }}</span>
              <span v-if="isConnected(machine.name)" class="mcl-badge">已连接</span>
              <span v-else-if="connectingName === machine.name" class="mcl-badge connecting">连接中</span>
            </div>
            <div class="mcl-addr">{{ formatMachineAddr(machine) }}</div>
          </div>
          <div v-if="showEdit" class="mcl-side" @click.stop>
            <el-tooltip content="编辑配置" placement="top">
              <button type="button" class="mcl-edit" @click="$emit('edit-machine', machine)">
                <el-icon :size="14"><Setting /></el-icon>
              </button>
            </el-tooltip>
          </div>
        </li>
      </ul>
    </div>

    <ul v-if="defaultMachines.length" class="mcl-list" role="listbox">
      <li
        v-for="machine in defaultMachines"
        :key="machine.id || machine.name"
        class="mcl-item"
        :class="{
          connected: isConnected(machine.name),
          connecting: connectingName === machine.name,
        }"
        role="option"
        tabindex="0"
        @click="onConnect(machine)"
        @keydown.enter.prevent="onConnect(machine)"
      >
        <div class="mcl-dot" aria-hidden="true" />
        <div class="mcl-body">
          <div class="mcl-line">
            <span class="mcl-name">{{ machine.name }}</span>
            <span v-if="isConnected(machine.name)" class="mcl-badge">已连接</span>
            <span v-else-if="connectingName === machine.name" class="mcl-badge connecting">连接中</span>
          </div>
          <div class="mcl-addr">{{ formatMachineAddr(machine) }}</div>
        </div>
        <div v-if="showEdit" class="mcl-side" @click.stop>
          <el-tooltip content="编辑配置" placement="top">
            <button type="button" class="mcl-edit" @click="$emit('edit-machine', machine)">
              <el-icon :size="14"><Setting /></el-icon>
            </button>
          </el-tooltip>
        </div>
      </li>
    </ul>
  </div>
</template>

<script>
import { computed, ref, watch } from 'vue'
import { ArrowRight, Setting } from '@element-plus/icons-vue'
import {
  splitMachineTree,
  formatMachineAddr,
  isMachineConnected,
} from '../../utils/machineGroups'

export default {
  name: 'MachineConnectList',
  components: { ArrowRight, Setting },
  props: {
    machines: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
    connectingName: { type: String, default: '' },
    showEdit: { type: Boolean, default: false },
    emptyText: { type: String, default: '暂无机器' },
    /** 有关键词时自动展开全部分组 */
    autoExpandOnFilter: { type: Boolean, default: true },
    filterKeyword: { type: String, default: '' },
  },
  emits: ['connect', 'edit-machine'],
  setup(props, { emit }) {
    const expandedGroups = ref([])

    const machineTree = computed(() => splitMachineTree(props.machines || []))
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

    watch(
      () => [props.filterKeyword, customGroups.value.map((g) => g.name).join('\0')],
      ([kw]) => {
        if (!props.autoExpandOnFilter) return
        if (String(kw || '').trim()) {
          expandedGroups.value = customGroups.value.map((g) => g.name)
        } else {
          expandedGroups.value = []
        }
      },
    )

    const isConnected = (name) => isMachineConnected(name, props.sessions)

    const onConnect = (machine) => {
      if (props.connectingName) return
      emit('connect', machine.name)
    }

    return {
      customGroups,
      defaultMachines,
      hasTree,
      isGroupExpanded,
      toggleGroup,
      isConnected,
      formatMachineAddr,
      onConnect,
    }
  },
}
</script>

<style scoped>
.mcl-empty {
  text-align: center;
  color: var(--app-text-muted);
  font-size: 13px;
  padding: 32px 12px;
}

.mcl-root {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.mcl-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.mcl-group-head {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 6px 4px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: inherit;
  cursor: pointer;
  text-align: left;
}

.mcl-group-head:hover {
  background: color-mix(in srgb, var(--app-text-muted) 8%, transparent);
}

.mcl-caret {
  font-size: 12px;
  color: var(--app-text-muted);
  transition: transform 0.15s ease;
  flex-shrink: 0;
}

.mcl-caret.open {
  transform: rotate(90deg);
}

.mcl-group-name {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 650;
  color: var(--app-text-secondary, var(--app-text));
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mcl-group-count {
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

.mcl-list {
  list-style: none;
  margin: 0;
  padding: 6px;
  border: 1px solid var(--app-border);
  border-radius: 12px;
  background: var(--app-card-bg, var(--app-panel-bg));
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.mcl-item {
  display: grid;
  grid-template-columns: 8px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  padding: 10px 8px 10px 10px;
  border-radius: 8px;
  cursor: pointer;
  outline: none;
  transition: background 0.12s ease;
}

.mcl-item:hover,
.mcl-item:focus-visible {
  background: var(--app-accent-bg);
}

.mcl-item.connecting {
  background: color-mix(in srgb, var(--app-accent-color) 10%, transparent);
}

.mcl-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--app-border);
  justify-self: center;
}

.mcl-item.connected .mcl-dot {
  background: #67c23a;
}

.mcl-item.connecting .mcl-dot,
.mcl-item:hover .mcl-dot,
.mcl-item:focus-visible .mcl-dot {
  background: var(--app-accent-color);
}

.mcl-item.connected:hover .mcl-dot,
.mcl-item.connected:focus-visible .mcl-dot {
  background: #67c23a;
}

.mcl-body {
  min-width: 0;
}

.mcl-line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.mcl-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mcl-badge {
  flex-shrink: 0;
  font-size: 10px;
  line-height: 1;
  padding: 3px 6px;
  border-radius: 4px;
  color: #67c23a;
  background: color-mix(in srgb, #67c23a 14%, transparent);
}

.mcl-badge.connecting {
  color: var(--app-accent-color);
  background: var(--app-accent-bg);
}

.mcl-addr {
  margin-top: 3px;
  font-size: 12px;
  color: var(--app-text-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mcl-side {
  flex-shrink: 0;
  display: flex;
  align-items: center;
}

.mcl-edit {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--app-text-muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  padding: 0;
  opacity: 0;
}

.mcl-item:hover .mcl-edit,
.mcl-item:focus-within .mcl-edit,
.mcl-edit:focus-visible {
  opacity: 1;
}

.mcl-edit:hover {
  color: var(--app-accent-color);
  background: var(--app-accent-bg);
}
</style>
