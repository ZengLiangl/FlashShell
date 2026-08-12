<template>
  <div v-if="!hasTree" class="app-empty">
    <p class="app-empty-desc">{{ emptyText }}</p>
  </div>
  <div v-else class="ml-stack" :class="{ 'is-grid': layout === 'grid', 'ml-stack--cards': variant === 'cards' }">
    <div
      v-for="group in customGroups"
      :key="group.name"
      class="ml-group"
      :class="{ 'is-open': isGroupExpanded(group.name) }"
    >
      <button
        type="button"
        class="ml-group-head"
        :aria-expanded="isGroupExpanded(group.name)"
        @click="toggleGroup(group.name)"
      >
        <span class="ml-group-icon" aria-hidden="true">
          <el-icon :size="variant === 'cards' ? 16 : 14"><Folder /></el-icon>
        </span>
        <span class="ml-group-meta">
          <span class="ml-group-name">{{ group.name }}</span>
          <span class="ml-group-sub">{{ group.machines.length }} 台机器</span>
        </span>
        <span class="ml-group-count">{{ group.machines.length }}</span>
        <el-icon class="ml-group-caret" :class="{ open: isGroupExpanded(group.name) }">
          <ArrowRight />
        </el-icon>
      </button>
      <div
        v-show="isGroupExpanded(group.name)"
        class="ml-list-scroll"
        :class="{ 'is-virtual': needsVirtual(group.machines) }"
        @scroll="onListScroll($event, group.name)"
      >
        <ul
          class="ml-list"
          :class="{ 'ml-list--grid': layout === 'grid' }"
          role="listbox"
          :style="listPadStyle(group.name, group.machines)"
        >
          <li
            v-for="machine in visibleMachines(group.name, group.machines)"
            :key="machine.id || machine.name"
            class="ml-item"
            :class="{
              connected: isConnected(machine.name),
              connecting: machineConnecting(machine.name),
              'is-context-target': showContextMenu && isContextTarget(machine),
            }"
            role="option"
            tabindex="0"
            @click="onConnect(machine)"
            @keydown.enter.prevent="onConnect(machine)"
            @contextmenu.prevent="onItemContextMenu($event, machine)"
          >
            <div class="ml-machine-icon" aria-hidden="true">
              <el-icon :size="variant === 'cards' ? 18 : 16"><Monitor /></el-icon>
            </div>
            <div class="ml-body">
              <div class="ml-line">
                <span class="ml-name">{{ machine.name }}</span>
                <el-icon v-if="machine.pinned" class="ml-pin" :size="12" title="已置顶"><StarFilled /></el-icon>
                <template v-if="variant !== 'cards'">
                  <span v-if="sessionCount(machine.name) > 0" class="ml-badge">{{ sessionCount(machine.name) }} 会话</span>
                  <span v-else-if="machineConnecting(machine.name)" class="ml-badge connecting">连接中</span>
                </template>
              </div>
              <div class="ml-addr">{{ formatMachineAddr(machine) }}</div>
              <div v-if="machineTags(machine).length" class="ml-tags">
                <span v-for="t in machineTags(machine)" :key="t" class="ml-tag">{{ t }}</span>
              </div>
            </div>
            <div class="ml-side" :class="{ 'ml-side--meta': variant === 'cards' }" @click.stop>
              <template v-if="variant === 'cards'">
                <span v-if="sessionCount(machine.name) > 0" class="ml-card-badge">{{ sessionCount(machine.name) }} 会话</span>
                <span v-else-if="machineConnecting(machine.name)" class="ml-card-badge is-accent">连接中</span>
                <span v-else class="ml-card-badge is-muted">未连接</span>
              </template>
              <button v-if="showEdit" type="button" class="ml-icon-btn" title="编辑配置" @click="$emit('edit-machine', machine)">
                <el-icon :size="14"><Setting /></el-icon>
              </button>
            </div>
          </li>
        </ul>
      </div>
    </div>

    <div
      v-if="defaultMachines.length"
      class="ml-list-scroll"
      :class="{ 'is-virtual': needsVirtual(defaultMachines) }"
      @scroll="onListScroll($event, '__default__')"
    >
      <ul
        class="ml-list"
        :class="{ 'ml-list--grid': layout === 'grid' }"
        role="listbox"
        :style="listPadStyle('__default__', defaultMachines)"
      >
        <li
          v-for="machine in visibleMachines('__default__', defaultMachines)"
          :key="machine.id || machine.name"
          class="ml-item"
          :class="{
            connected: isConnected(machine.name),
            connecting: machineConnecting(machine.name),
            'is-context-target': showContextMenu && isContextTarget(machine),
          }"
          role="option"
          tabindex="0"
          @click="onConnect(machine)"
          @keydown.enter.prevent="onConnect(machine)"
          @contextmenu.prevent="onItemContextMenu($event, machine)"
        >
          <div class="ml-machine-icon" aria-hidden="true">
            <el-icon :size="variant === 'cards' ? 18 : 16"><Monitor /></el-icon>
          </div>
          <div class="ml-body">
            <div class="ml-line">
              <span class="ml-name">{{ machine.name }}</span>
              <el-icon v-if="machine.pinned" class="ml-pin" :size="12" title="已置顶"><StarFilled /></el-icon>
              <template v-if="variant !== 'cards'">
                <span v-if="sessionCount(machine.name) > 0" class="ml-badge">{{ sessionCount(machine.name) }} 会话</span>
                <span v-else-if="machineConnecting(machine.name)" class="ml-badge connecting">连接中</span>
              </template>
            </div>
            <div class="ml-addr">{{ formatMachineAddr(machine) }}</div>
            <div v-if="machineTags(machine).length" class="ml-tags">
              <span v-for="t in machineTags(machine)" :key="t" class="ml-tag">{{ t }}</span>
            </div>
          </div>
          <div class="ml-side" :class="{ 'ml-side--meta': variant === 'cards' }" @click.stop>
            <template v-if="variant === 'cards'">
              <span v-if="sessionCount(machine.name) > 0" class="ml-card-badge">{{ sessionCount(machine.name) }} 会话</span>
              <span v-else-if="machineConnecting(machine.name)" class="ml-card-badge is-accent">连接中</span>
              <span v-else class="ml-card-badge is-muted">未连接</span>
            </template>
            <button v-if="showEdit" type="button" class="ml-icon-btn" title="编辑配置" @click="$emit('edit-machine', machine)">
              <el-icon :size="14"><Setting /></el-icon>
            </button>
          </div>
        </li>
      </ul>
    </div>
  </div>

  <MachineContextMenu
    v-if="showContextMenu"
    :ctx="ctx"
    @connect="onConnect"
    @copy="onCopy"
    @edit="onEdit"
    @delete="onDelete"
    @toggle-pin="onTogglePin"
    @hide="hideContextMenu"
  />
</template>

<script>
import { computed, reactive, ref, watch } from 'vue'
import { ArrowRight, Setting, Monitor, StarFilled, Folder } from '@element-plus/icons-vue'
import {
  splitMachineTree,
  formatMachineAddr,
  isMachineConnected,
  isMachineConnecting,
  countMachineSessions,
  normalizeMachineTags,
} from '../../utils/machineGroups'
import { windowMachineList, MACHINE_LIST_VIRTUALIZE_AT, MACHINE_LIST_ROW_H } from '../../utils/machineListWindow'
import { useMachineContextMenu } from '../../composables/useMachineContextMenu'
import MachineContextMenu from './MachineContextMenu.vue'

export default {
  name: 'MachineConnectList',
  components: { ArrowRight, Setting, Monitor, StarFilled, Folder, MachineContextMenu },
  props: {
    machines: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
    workspaceSessions: { type: Array, default: () => [] },
    connectingName: { type: String, default: '' },
    showEdit: { type: Boolean, default: false },
    showContextMenu: { type: Boolean, default: false },
    emptyText: { type: String, default: '暂无机器' },
    autoExpandOnFilter: { type: Boolean, default: true },
    filterKeyword: { type: String, default: '' },
    /** list：单列；grid：多列紧凑，便于一眼看到更多机器 */
    layout: { type: String, default: 'list' },
    /** default：紧凑列表；cards：与首页任务卡片同尺寸的卡片行 */
    variant: { type: String, default: 'default' },
    /** 不按分组名折叠，按传入顺序直接展示机器（如首页最近连接） */
    flat: { type: Boolean, default: false },
  },
  emits: ['connect', 'edit-machine', 'copy-machine', 'delete-machine', 'toggle-pin'],
  setup(props, { emit }) {
    const expandedGroups = ref([])
    const scrollTops = reactive({})
    const { ctx, hideContextMenu, onMachineContextMenu, isContextTarget } = useMachineContextMenu()

    const machineTree = computed(() => {
      const list = props.machines || []
      if (props.flat) {
        return { customGroups: [], defaultMachines: list }
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

    const needsVirtual = (list) => (list || []).length >= MACHINE_LIST_VIRTUALIZE_AT

    const windowFor = (key, list) => {
      const scrollTop = scrollTops[key] || 0
      const cols = props.layout === 'grid' ? 2 : 1
      return windowMachineList(list, scrollTop, 520, {
        rowH: props.variant === 'cards' ? MACHINE_LIST_ROW_H : 64,
        cols,
      })
    }

    const visibleMachines = (key, list) => windowFor(key, list).items

    const listPadStyle = (key, list) => {
      const win = windowFor(key, list)
      if (!win.virtual) return undefined
      return {
        paddingTop: `${win.padTop}px`,
        paddingBottom: `${win.padBottom}px`,
      }
    }

    const onListScroll = (event, key) => {
      scrollTops[key] = event?.target?.scrollTop || 0
    }

    const isConnected = (name) => isMachineConnected(name, props.sessions)
    const sessionCount = (name) => countMachineSessions(name, props.sessions)
    const machineConnecting = (name) =>
      isMachineConnecting(name, props.workspaceSessions.length ? props.workspaceSessions : props.sessions)
    const machineTags = (machine) => normalizeMachineTags(machine?.tags)

    const onConnect = (machine) => {
      if (machineConnecting(machine.name)) return
      emit('connect', machine.name)
    }

    const onItemContextMenu = (event, machine) => {
      if (!props.showContextMenu) return
      onMachineContextMenu(event, machine)
    }

    const onCopy = (machine) => {
      hideContextMenu()
      if (machine) emit('copy-machine', machine)
    }

    const onEdit = (machine) => {
      hideContextMenu()
      if (machine) emit('edit-machine', machine)
    }

    const onDelete = (machine) => {
      hideContextMenu()
      if (machine) emit('delete-machine', machine)
    }

    const onTogglePin = (machine) => {
      hideContextMenu()
      if (machine) emit('toggle-pin', machine)
    }

    return {
      ctx,
      hideContextMenu,
      customGroups,
      defaultMachines,
      hasTree,
      isGroupExpanded,
      toggleGroup,
      isConnected,
      sessionCount,
      machineConnecting,
      machineTags,
      formatMachineAddr,
      needsVirtual,
      visibleMachines,
      listPadStyle,
      onListScroll,
      onConnect,
      onItemContextMenu,
      isContextTarget,
      onCopy,
      onEdit,
      onDelete,
      onTogglePin,
    }
  },
}
</script>

<style scoped>
.ml-list-scroll.is-virtual {
  max-height: min(70vh, 720px);
  overflow: auto;
}
</style>
