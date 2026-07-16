<template>
  <div v-if="!hasTree" class="app-empty">
    <p class="app-empty-desc">{{ emptyText }}</p>
  </div>
  <div v-else class="ml-stack">
    <div
      v-for="group in customGroups"
      :key="group.name"
      class="ml-group"
    >
      <button
        type="button"
        class="ml-group-head"
        :aria-expanded="isGroupExpanded(group.name)"
        @click="toggleGroup(group.name)"
      >
        <el-icon class="ml-group-caret" :class="{ open: isGroupExpanded(group.name) }">
          <ArrowRight />
        </el-icon>
        <span class="ml-group-name">{{ group.name }}</span>
        <span class="ml-group-count">{{ group.machines.length }}</span>
      </button>
      <ul
        v-show="isGroupExpanded(group.name)"
        class="ml-list"
        role="listbox"
      >
        <li
          v-for="machine in group.machines"
          :key="machine.id || machine.name"
          class="ml-item"
          :class="{
            connected: isConnected(machine.name),
            connecting: machineConnecting(machine.name),
          }"
          role="option"
          tabindex="0"
          @click="onConnect(machine)"
          @keydown.enter.prevent="onConnect(machine)"
        >
          <div class="ml-machine-icon" aria-hidden="true">
            <el-icon :size="16"><Monitor /></el-icon>
          </div>
          <div class="ml-body">
            <div class="ml-line">
              <span class="ml-name">{{ machine.name }}</span>
              <span v-if="sessionCount(machine.name) > 0" class="ml-badge">{{ sessionCount(machine.name) }} 会话</span>
              <span v-else-if="machineConnecting(machine.name)" class="ml-badge connecting">连接中</span>
            </div>
            <div class="ml-addr">{{ formatMachineAddr(machine) }}</div>
          </div>
          <div v-if="showEdit" class="ml-side" @click.stop>
            <button type="button" class="ml-icon-btn" title="编辑配置" @click="$emit('edit-machine', machine)">
              <el-icon :size="14"><Setting /></el-icon>
            </button>
          </div>
        </li>
      </ul>
    </div>

    <ul v-if="defaultMachines.length" class="ml-list" role="listbox">
      <li
        v-for="machine in defaultMachines"
        :key="machine.id || machine.name"
        class="ml-item"
        :class="{
          connected: isConnected(machine.name),
          connecting: machineConnecting(machine.name),
        }"
        role="option"
        tabindex="0"
        @click="onConnect(machine)"
        @keydown.enter.prevent="onConnect(machine)"
      >
          <div class="ml-machine-icon" aria-hidden="true">
            <el-icon :size="16"><Monitor /></el-icon>
          </div>
        <div class="ml-body">
          <div class="ml-line">
            <span class="ml-name">{{ machine.name }}</span>
            <span v-if="sessionCount(machine.name) > 0" class="ml-badge">{{ sessionCount(machine.name) }} 会话</span>
            <span v-else-if="machineConnecting(machine.name)" class="ml-badge connecting">连接中</span>
          </div>
          <div class="ml-addr">{{ formatMachineAddr(machine) }}</div>
        </div>
        <div v-if="showEdit" class="ml-side" @click.stop>
          <button type="button" class="ml-icon-btn" title="编辑配置" @click="$emit('edit-machine', machine)">
            <el-icon :size="14"><Setting /></el-icon>
          </button>
        </div>
      </li>
    </ul>
  </div>
</template>

<script>
import { computed, ref, watch } from 'vue'
import { ArrowRight, Setting, Monitor } from '@element-plus/icons-vue'
import {
  splitMachineTree,
  formatMachineAddr,
  isMachineConnected,
  isMachineConnecting,
  countMachineSessions,
} from '../../utils/machineGroups'

export default {
  name: 'MachineConnectList',
  components: { ArrowRight, Setting, Monitor },
  props: {
    machines: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
    workspaceSessions: { type: Array, default: () => [] },
    connectingName: { type: String, default: '' },
    showEdit: { type: Boolean, default: false },
    emptyText: { type: String, default: '暂无机器' },
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
    const sessionCount = (name) => countMachineSessions(name, props.sessions)
    const machineConnecting = (name) =>
      isMachineConnecting(name, props.workspaceSessions.length ? props.workspaceSessions : props.sessions)

    const onConnect = (machine) => {
      if (machineConnecting(machine.name)) return
      emit('connect', machine.name)
    }

    return {
      customGroups,
      defaultMachines,
      hasTree,
      isGroupExpanded,
      toggleGroup,
      isConnected,
      sessionCount,
      machineConnecting,
      formatMachineAddr,
      onConnect,
    }
  },
}
</script>
