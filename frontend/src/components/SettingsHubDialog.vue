<template>
  <el-dialog
    v-model="visibleProxy"
    title="系统设置"
    width="960px"
    top="5vh"
    class="settings-hub-dialog"
    append-to-body
    destroy-on-close
    :before-close="handleClose"
  >
    <div class="settings-hub">
      <aside class="hub-nav">
        <button
          v-for="item in navItems"
          :key="item.id"
          type="button"
          class="hub-nav-item"
          :class="{ active: section === item.id }"
          @click="section = item.id"
        >
          <el-icon :size="16"><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
        </button>
      </aside>

      <main class="hub-main">
        <div class="hub-main-title">{{ currentTitle }}</div>
        <div class="hub-main-body">
          <MachineConfigDialog
            v-if="visibleProxy && section === 'machines'"
            embedded
            :active="true"
            :edit-machine-id="editMachineId"
            @changed="$emit('machines-changed')"
            @closed="$emit('machines-closed')"
          />
          <WorkPathConfigDialog
            v-if="visibleProxy && section === 'env'"
            embedded
            :active="true"
          />
          <SystemSettingsDialog
            v-if="visibleProxy && section === 'general'"
            embedded
            :active="true"
            @saved="visibleProxy = false"
          />
          <ShortcutSettingsPanel
            v-if="visibleProxy && section === 'shortcuts'"
            :active="true"
          />
          <ExecutionHistoryDialog
            v-if="visibleProxy && section === 'history'"
            embedded
            :active="true"
          />
        </div>
      </main>
    </div>
  </el-dialog>
</template>

<script>
import { computed, ref, watch } from 'vue'
import { Monitor, Key, Setting, Clock, Operation } from '@element-plus/icons-vue'
import MachineConfigDialog from './MachineConfigDialog.vue'
import WorkPathConfigDialog from './WorkPathConfigDialog.vue'
import SystemSettingsDialog from './SystemSettingsDialog.vue'
import ShortcutSettingsPanel from './ShortcutSettingsPanel.vue'
import ExecutionHistoryDialog from './ExecutionHistoryDialog.vue'

const NAV_ITEMS = [
  { id: 'machines', label: '机器配置', icon: Monitor },
  { id: 'env', label: '环境变量', icon: Key },
  { id: 'general', label: '系统设置', icon: Setting },
  { id: 'shortcuts', label: '快捷键', icon: Operation },
  { id: 'history', label: '执行历史', icon: Clock },
]

export default {
  name: 'SettingsHubDialog',
  components: {
    MachineConfigDialog,
    WorkPathConfigDialog,
    SystemSettingsDialog,
    ShortcutSettingsPanel,
    ExecutionHistoryDialog,
    Monitor,
    Key,
    Setting,
    Operation,
    Clock,
  },
  props: {
    modelValue: { type: Boolean, default: false },
    initialSection: { type: String, default: 'general' },
    editMachineId: { type: String, default: '' },
  },
  emits: ['update:modelValue', 'machines-changed', 'machines-closed'],
  setup(props, { emit }) {
    const navItems = NAV_ITEMS
    const resolveSection = (id) => {
      return navItems.some((i) => i.id === id) ? id : 'general'
    }

    const section = ref(resolveSection(props.initialSection))

    const visibleProxy = computed({
      get: () => props.modelValue,
      set: (v) => emit('update:modelValue', v),
    })

    const currentTitle = computed(() => {
      return navItems.find((i) => i.id === section.value)?.label || '系统设置'
    })

    watch(
      () => [props.modelValue, props.initialSection],
      ([open, initial]) => {
        if (open) section.value = resolveSection(initial)
      },
      { immediate: true },
    )

    const handleClose = () => {
      visibleProxy.value = false
      emit('machines-closed')
    }

    return {
      visibleProxy,
      section,
      navItems,
      currentTitle,
      handleClose,
    }
  },
}
</script>

<style scoped>
.settings-hub {
  display: flex;
  height: min(68vh, 640px);
  min-height: 420px;
  margin: -8px -12px -4px;
  border-top: 1px solid var(--app-border);
}

.hub-nav {
  width: 180px;
  flex-shrink: 0;
  padding: 12px 8px;
  border-right: 1px solid var(--app-border);
  background: var(--app-bg);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.hub-nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 10px 12px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--app-text-secondary, var(--app-text));
  font-size: 13px;
  cursor: pointer;
  text-align: left;
}

.hub-nav-item:hover {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}

.hub-nav-item.active {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
  font-weight: 600;
}

.hub-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--app-panel-bg);
}

.hub-main-title {
  flex-shrink: 0;
  padding: 14px 18px 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--app-text);
}

.hub-main-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 4px 18px 16px;
}
</style>

<style>
.settings-hub-dialog .el-dialog__body {
  padding-top: 8px;
  padding-bottom: 12px;
}
</style>
