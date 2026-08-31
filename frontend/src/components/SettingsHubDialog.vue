<template>
  <el-dialog
    v-model="visibleProxy"
    title="设置"
    width="1000px"
    top="4vh"
    class="settings-hub-dialog"
    append-to-body
    destroy-on-close
    :close-on-press-escape="false"
    :before-close="handleClose"
  >
    <div class="settings-hub">
      <aside class="hub-nav">
        <template v-for="(group, gi) in navGroups" :key="group.id">
          <div v-if="gi > 0" class="hub-nav-sep" role="separator" />
          <button
            v-for="item in group.items"
            :key="item.id"
            type="button"
            class="hub-nav-item"
            :class="{ active: section === item.id }"
            @click="section = item.id"
          >
            <el-icon :size="16"><component :is="item.icon" /></el-icon>
            <span>{{ item.label }}</span>
          </button>
        </template>
      </aside>

      <main class="hub-main">
        <div v-if="section !== 'mcp'" class="hub-main-title">{{ currentTitle }}</div>
        <div class="hub-main-body" :class="{ 'hub-main-body--scroll': section === 'mcp', 'hub-main-body--mcp': section === 'mcp' }">
          <SystemSettingsDialog
            v-if="mountedPanels.system"
            v-show="isSystemSection"
            embedded
            :active="isSystemSection"
            :panel="isSystemSection ? section : panelCache.system"
            @saved="visibleProxy = false"
          />
          <MachineConfigDialog
            v-if="mountedPanels.machines"
            v-show="section === 'machines'"
            embedded
            :active="section === 'machines'"
            :edit-machine-id="editMachineId"
            @changed="$emit('machines-changed')"
            @closed="$emit('machines-closed')"
            @connect="onConnectMachine"
          />
          <WorkPathConfigDialog
            v-if="mountedPanels.env"
            v-show="section === 'env'"
            embedded
            :active="section === 'env'"
          />
          <ProxySettingsPanel
            v-if="mountedPanels.proxy"
            v-show="section === 'proxy'"
            :active="section === 'proxy'"
          />
          <PortForwardPanel
            v-if="mountedPanels.portforwards"
            v-show="section === 'portforwards'"
            :active="section === 'portforwards'"
          />
          <ShortcutSettingsPanel
            v-if="mountedPanels.shortcuts"
            v-show="section === 'shortcuts'"
            :active="section === 'shortcuts'"
          />
          <McpAccessView
            v-if="mountedPanels.mcp"
            v-show="section === 'mcp'"
            embedded
            :active="section === 'mcp'"
          />
        </div>
      </main>
    </div>
  </el-dialog>
</template>

<script>
import { computed, reactive, ref, watch } from 'vue'
import {
  Monitor,
  Key,
  Operation,
  Connection,
  Switch,
  Brush,
  Cpu,
  FolderOpened,
  Document,
  Lock,
  InfoFilled,
  Box,
  Link,
} from '@element-plus/icons-vue'
import MachineConfigDialog from './MachineConfigDialog.vue'
import WorkPathConfigDialog from './WorkPathConfigDialog.vue'
import SystemSettingsDialog from './SystemSettingsDialog.vue'
import ShortcutSettingsPanel from './ShortcutSettingsPanel.vue'
import ProxySettingsPanel from './ProxySettingsPanel.vue'
import PortForwardPanel from './PortForwardPanel.vue'
import McpAccessView from './McpAccessView.vue'

/** 偏好设置（对齐 Netcatty：应用 / 外观 / 终端 / …） */
const PREFS_ITEMS = [
  { id: 'app', label: '应用', icon: Monitor },
  { id: 'appearance', label: '外观', icon: Brush },
  { id: 'terminal', label: '终端', icon: Cpu },
  { id: 'sftp', label: 'SFTP', icon: FolderOpened },
  { id: 'accounts', label: '密钥库', icon: Key },
  { id: 'credentials', label: '凭据安全', icon: Lock },
  { id: 'security', label: '主机密钥', icon: Document },
]

/** 运维资源 */
const OPS_ITEMS = [
  { id: 'machines', label: '机器配置', icon: Box },
  { id: 'env', label: '环境变量', icon: Document },
  { id: 'portforwards', label: '端口转发', icon: Switch },
  { id: 'proxy', label: 'HTTP 代理', icon: Connection },
  { id: 'shortcuts', label: '快捷键', icon: Operation },
  { id: 'mcp', label: 'MCP 接入', icon: Link },
]

const META_ITEMS = [
  { id: 'about', label: '关于', icon: InfoFilled },
]

const ALL_ITEMS = [...PREFS_ITEMS, ...OPS_ITEMS, ...META_ITEMS]
const SYSTEM_SECTION_IDS = new Set(['app', 'appearance', 'terminal', 'sftp', 'accounts', 'security', 'credentials', 'about', 'general'])

export default {
  name: 'SettingsHubDialog',
  components: {
    MachineConfigDialog,
    WorkPathConfigDialog,
    SystemSettingsDialog,
    ShortcutSettingsPanel,
    ProxySettingsPanel,
    PortForwardPanel,
    McpAccessView,
    Monitor,
    Key,
    Operation,
    Connection,
    Switch,
    Brush,
    Cpu,
    FolderOpened,
    Document,
    Lock,
    InfoFilled,
    Box,
  },
  props: {
    modelValue: { type: Boolean, default: false },
    initialSection: { type: String, default: 'app' },
    editMachineId: { type: String, default: '' },
  },
  emits: ['update:modelValue', 'machines-changed', 'machines-closed', 'connect-machine'],
  setup(props, { emit }) {
    const navGroups = [
      { id: 'prefs', items: PREFS_ITEMS },
      { id: 'ops', items: OPS_ITEMS },
      { id: 'meta', items: META_ITEMS },
    ]

    const resolveSection = (id) => {
      const raw = String(id || '').trim()
      if (raw === 'general') return 'app'
      return ALL_ITEMS.some((i) => i.id === raw) ? raw : 'app'
    }

    const section = ref(resolveSection(props.initialSection))
    const mountedPanels = reactive({
      system: false,
      machines: false,
      env: false,
      proxy: false,
      portforwards: false,
      shortcuts: false,
      mcp: false,
    })
    /** 离开 system 分区时保留上次 panel，避免 v-show 隐藏期间被误路由 */
    const panelCache = reactive({ system: 'app' })

    const markMounted = (id) => {
      if (SYSTEM_SECTION_IDS.has(id)) {
        mountedPanels.system = true
        panelCache.system = id === 'general' ? 'app' : id
        return
      }
      if (Object.prototype.hasOwnProperty.call(mountedPanels, id)) {
        mountedPanels[id] = true
      }
    }

    const visibleProxy = computed({
      get: () => props.modelValue,
      set: (v) => emit('update:modelValue', v),
    })

    const isSystemSection = computed(() => SYSTEM_SECTION_IDS.has(section.value))

    const currentTitle = computed(() => {
      return ALL_ITEMS.find((i) => i.id === section.value)?.label || '设置'
    })

    const onConnectMachine = (machineName) => {
      visibleProxy.value = false
      emit('connect-machine', machineName)
    }

    watch(
      () => [props.modelValue, props.initialSection],
      ([open, initial]) => {
        if (open) {
          section.value = resolveSection(initial)
          markMounted(section.value)
        } else {
          mountedPanels.system = false
          mountedPanels.machines = false
          mountedPanels.env = false
          mountedPanels.proxy = false
          mountedPanels.portforwards = false
          mountedPanels.shortcuts = false
          mountedPanels.mcp = false
        }
      },
      { immediate: true },
    )

    watch(section, (id) => markMounted(id))

    const handleClose = () => {
      visibleProxy.value = false
      emit('machines-closed')
    }

    return {
      visibleProxy,
      section,
      navGroups,
      currentTitle,
      isSystemSection,
      mountedPanels,
      panelCache,
      handleClose,
      onConnectMachine,
    }
  },
}
</script>

<style scoped>
.settings-hub {
  display: flex;
  height: min(72vh, 680px);
  min-height: 440px;
}

.hub-nav {
  width: 200px;
  flex-shrink: 0;
  padding: 12px 8px;
  border-right: 1px solid var(--border);
  background: var(--surface-2);
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow-y: auto;
}

.hub-nav-sep {
  height: 1px;
  margin: 8px 8px;
  background: var(--border);
  flex-shrink: 0;
}

.hub-nav-item {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 7px 10px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--fg-2);
  font-size: 13.5px;
  cursor: pointer;
  text-align: left;
  transition: background 0.12s ease, color 0.12s ease;
}

.hub-nav-item:hover {
  background: color-mix(in oklch, var(--fg) 6%, transparent);
  color: var(--fg);
}

.hub-nav-item.active {
  background: var(--accent-soft);
  color: var(--accent);
  font-weight: 500;
}

.hub-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--surface);
}

.hub-main-title {
  flex-shrink: 0;
  padding: 16px 20px 14px;
  font-size: 16px;
  font-weight: 600;
  color: var(--fg);
  border-bottom: 1px solid var(--border);
}

.hub-main-body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding: 4px 20px 12px;
  display: flex;
  flex-direction: column;
}

.hub-main-body > * {
  flex: 1;
  min-height: 0;
}

.hub-main-body--mcp {
  padding-top: 10px;
}

.hub-main-body--scroll {
  overflow: hidden;
  padding-right: 12px;
}

.hub-main-body--scroll > .mcp-hub {
  overflow-y: auto;
  overflow-x: hidden;
}
</style>

<style>
.settings-hub-dialog.el-dialog {
  overflow: hidden;
  display: flex;
  flex-direction: column;
  max-height: 92vh;
  /* 外框仅装饰，暗色下易在侧栏底角露出异色描边；靠阴影区分即可 */
  border: none;
  background: var(--surface);
  --el-dialog-bg-color: var(--surface);
  box-shadow: var(--shadow-lg);
}

.settings-hub-dialog .el-dialog__header {
  flex-shrink: 0;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
}

.settings-hub-dialog .el-dialog__body {
  padding: 0;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  background: var(--surface);
}
</style>
