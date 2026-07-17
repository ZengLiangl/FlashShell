<template>
  <el-dialog
    v-model="visibleProxy"
    title="连接"
    width="600px"
    class="machine-picker-dialog"
    append-to-body
  >
    <div class="picker-shell">
      <div class="app-toolbar picker-toolbar">
        <el-input
          v-model="keyword"
          clearable
          placeholder="搜索机器名 / 地址"
          size="small"
          class="app-toolbar-search"
          @keydown.enter.prevent="connectFirst"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <div class="icon-actions">
          <el-tooltip content="本机终端" placement="top">
            <el-button class="picker-tool-btn" size="small" circle :icon="Monitor" @click="onAddLocal" />
          </el-tooltip>
          <el-tooltip content="添加机器" placement="top">
            <el-button class="picker-tool-btn" size="small" circle :icon="Plus" @click="$emit('add-machine')" />
          </el-tooltip>
        </div>
      </div>

      <div class="picker-segment" role="tablist">
        <button
          type="button"
          class="picker-segment-btn"
          :class="{ active: activeTab === 'history' }"
          role="tab"
          :aria-selected="activeTab === 'history'"
          @click="activeTab = 'history'"
        >
          <el-icon :size="14"><Clock /></el-icon>
          <span class="picker-segment-label">最近连接</span>
          <span class="picker-segment-count">{{ historyRecords.length }}</span>
        </button>
        <button
          type="button"
          class="picker-segment-btn"
          :class="{ active: activeTab === 'machines' }"
          role="tab"
          :aria-selected="activeTab === 'machines'"
          @click="activeTab = 'machines'"
        >
          <el-icon :size="14"><List /></el-icon>
          <span class="picker-segment-label">全部机器</span>
          <span class="picker-segment-count">{{ machines.length }}</span>
        </button>
      </div>

      <div class="picker-panel">
        <div v-show="activeTab === 'history'" class="picker-pane">
          <div v-if="historyRecords.length" class="picker-pane-head">
            <span class="picker-pane-hint">{{ historyPaneHint }}</span>
            <button type="button" class="picker-clear-btn" @click="$emit('clear-history')">清空历史</button>
          </div>

          <div v-if="!historyRecords.length" class="picker-empty">
            <el-icon :size="36" class="picker-empty-icon"><Clock /></el-icon>
            <p class="picker-empty-title">暂无连接历史</p>
            <p class="picker-empty-desc">连接过的机器会出现在这里，方便快速重连</p>
            <el-button size="small" type="primary" plain @click="activeTab = 'machines'">浏览全部机器</el-button>
          </div>

          <ShellHistoryList
            v-else-if="historyFiltered.length"
            :records="historyRecords"
            :sessions="sessions"
            :workspace-sessions="workspaceSessions"
            :keyword="keyword"
            :show-head="false"
            embedded
            @connect="onConnect"
            @remove="(row) => $emit('remove-history', row)"
          />

          <div v-else class="picker-empty compact">
            <p class="picker-empty-desc">没有匹配「{{ keyword }}」的记录</p>
          </div>
        </div>

        <div v-show="activeTab === 'machines'" class="picker-pane">
          <div v-if="machines.length" class="picker-pane-head">
            <span class="picker-pane-hint">{{ machinesPaneHint }}</span>
          </div>

          <div v-if="!machines.length" class="picker-empty">
            <el-icon :size="36" class="picker-empty-icon"><Monitor /></el-icon>
            <p class="picker-empty-title">暂无机器配置</p>
            <p class="picker-empty-desc">添加机器后即可在此发起 SSH 连接</p>
            <el-button size="small" type="primary" @click="$emit('add-machine')">添加机器</el-button>
          </div>

          <div v-else class="picker-pane-body">
            <MachineConnectList
              :machines="filteredMachines"
              :sessions="sessions"
              :workspace-sessions="workspaceSessions"
              :connecting-name="connectingName"
              :filter-keyword="keyword"
              show-edit
              show-context-menu
              :empty-text="keyword ? `没有匹配「${keyword}」的机器` : '暂无机器配置'"
              @connect="onConnect"
              @edit-machine="(m) => $emit('edit-machine', m)"
              @copy-machine="(m) => $emit('copy-machine', m)"
              @delete-machine="(m) => $emit('delete-machine', m)"
            />
          </div>
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<script>
import { computed, ref, watch } from 'vue'
import { Monitor, Plus, Search, Clock, List } from '@element-plus/icons-vue'
import { machineMatchesKeyword } from '../../utils/machineGroups'
import MachineConnectList from './MachineConnectList.vue'
import ShellHistoryList from './ShellHistoryList.vue'

export default {
  name: 'ShellMachinePickerDialog',
  components: { MachineConnectList, ShellHistoryList, Search, Clock, List, Monitor },
  props: {
    modelValue: { type: Boolean, default: false },
    machines: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
    workspaceSessions: { type: Array, default: () => [] },
    connectingName: { type: String, default: '' },
    historyRecords: { type: Array, default: () => [] },
    initialTab: { type: String, default: '' },
  },
  emits: [
    'update:modelValue',
    'connect',
    'edit-machine',
    'copy-machine',
    'delete-machine',
    'add-machine',
    'add-local',
    'clear-history',
    'remove-history',
    'open',
  ],
  setup(props, { emit }) {
    const keyword = ref('')
    const activeTab = ref('history')

    const visibleProxy = computed({
      get: () => props.modelValue,
      set: (v) => emit('update:modelValue', v),
    })

    const filteredMachines = computed(() => {
      const kw = keyword.value
      let list = props.machines || []
      if (String(kw || '').trim()) {
        list = list.filter((m) => machineMatchesKeyword(m, kw))
      }
      return list
    })

    const historyFiltered = computed(() => {
      const kw = String(keyword.value || '').trim().toLowerCase()
      const list = props.historyRecords || []
      if (!kw) return list
      return list.filter((row) => {
        const hay = `${row.machineName || ''} ${row.user || ''} ${row.host || ''} ${row.port || ''}`.toLowerCase()
        return hay.includes(kw)
      })
    })

    const historyPaneHint = computed(() => {
      const kw = String(keyword.value || '').trim()
      const total = props.historyRecords.length
      const matched = historyFiltered.value.length
      if (kw) return `匹配 ${matched} / ${total} 条，Enter 连接首条`
      return `共 ${total} 条，点击或 Enter 快速连接`
    })

    const machinesPaneHint = computed(() => {
      const kw = String(keyword.value || '').trim()
      const total = (props.machines || []).length
      const matched = filteredMachines.value.length
      if (kw) return `匹配 ${matched} / ${total} 台，Enter 连接首条`
      return `共 ${total} 台，按分组浏览或搜索`
    })

    const resolveDefaultTab = () => {
      if (props.initialTab === 'history' || props.initialTab === 'machines') {
        return props.initialTab
      }
      return (props.historyRecords || []).length ? 'history' : 'machines'
    }

    const onDialogOpen = () => {
      activeTab.value = resolveDefaultTab()
      emit('open')
    }

    watch(
      () => props.modelValue,
      (visible) => {
        if (!visible) {
          keyword.value = ''
          return
        }
        onDialogOpen()
      },
    )

    const onConnect = (name) => emit('connect', name)

    const connectFirst = () => {
      if (activeTab.value === 'history') {
        const first = historyFiltered.value[0]
        if (first) onConnect(first.machineName)
        return
      }
      const first = filteredMachines.value[0]
      if (first) onConnect(first.name)
    }

    const onAddLocal = () => {
      emit('add-local')
      visibleProxy.value = false
    }

    return {
      Monitor,
      Plus,
      visibleProxy,
      keyword,
      activeTab,
      filteredMachines,
      historyFiltered,
      historyPaneHint,
      machinesPaneHint,
      onConnect,
      onAddLocal,
      connectFirst,
    }
  },
}
</script>

<style scoped>
.picker-shell {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.picker-toolbar {
  margin-bottom: 0;
}

.picker-segment {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6px;
  padding: 4px;
  border-radius: var(--app-radius-lg, 10px);
  background: color-mix(in srgb, var(--app-panel-bg, #f5f7fa) 88%, transparent);
  border: 1px solid var(--app-border, #e4e7ed);
}

.picker-segment-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 36px;
  padding: 0 12px;
  border: none;
  border-radius: var(--app-radius-md, 8px);
  background: transparent;
  color: var(--app-text-secondary, #606266);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease, box-shadow 0.15s ease;
}

.picker-segment-btn:hover {
  color: var(--app-text, #303133);
  background: color-mix(in srgb, var(--app-card-bg, #fff) 70%, transparent);
}

.picker-segment-btn.active {
  color: var(--app-accent-color, #409eff);
  background: var(--app-card-bg, #fff);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}

.picker-segment-label {
  line-height: 1;
}

.picker-segment-count {
  min-width: 18px;
  padding: 2px 6px;
  border-radius: 999px;
  font-size: 11px;
  line-height: 1.2;
  font-weight: 600;
  color: var(--app-text-muted, #909399);
  background: color-mix(in srgb, var(--app-text-muted, #909399) 12%, transparent);
}

.picker-segment-btn.active .picker-segment-count {
  color: var(--app-accent-color, #409eff);
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 14%, transparent);
}

.picker-panel {
  min-height: 360px;
  max-height: min(52vh, 440px);
  display: flex;
  flex-direction: column;
  border: 1px solid var(--app-border, #e4e7ed);
  border-radius: var(--app-radius-xl, 12px);
  background: var(--app-card-bg, #fff);
  overflow: hidden;
}

.picker-pane {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.picker-pane-head {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 14px 8px;
  border-bottom: 1px solid color-mix(in srgb, var(--app-border, #e4e7ed) 70%, transparent);
}

.picker-pane-hint {
  font-size: 12px;
  color: var(--app-text-muted, #909399);
  line-height: 1.4;
}

.picker-clear-btn {
  flex-shrink: 0;
  border: none;
  background: transparent;
  padding: 0;
  font-size: 12px;
  color: var(--app-text-muted, #909399);
  cursor: pointer;
  transition: color 0.12s ease;
}

.picker-clear-btn:hover {
  color: #f56c6c;
}

.picker-pane-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 8px;
}

.picker-pane :deep(.history-list-wrap) {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 8px;
}

.picker-pane :deep(.history-scroll) {
  max-height: none;
  border: none;
  background: transparent;
  padding: 0;
}

.picker-pane :deep(.ml-list) {
  border: none;
  background: transparent;
  padding: 0;
  gap: 4px;
}

.picker-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 36px 24px;
  text-align: center;
}

.picker-empty.compact {
  padding: 28px 16px;
}

.picker-empty-icon {
  color: var(--app-text-muted, #909399);
  opacity: 0.65;
}

.picker-empty-title {
  margin: 4px 0 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text, #303133);
}

.picker-empty-desc {
  margin: 0 0 6px;
  font-size: 12px;
  color: var(--app-text-muted, #909399);
  line-height: 1.5;
  max-width: 280px;
}
</style>

<!-- append-to-body 弹窗：统一圆钮盒模型 -->
<style>
.machine-picker-dialog .picker-tool-btn.el-button.is-circle {
  width: 28px !important;
  height: 28px !important;
  min-width: 28px !important;
  max-width: 28px !important;
  padding: 0 !important;
  margin: 0 !important;
  box-sizing: border-box !important;
  border-style: solid !important;
  border-width: 1px !important;
  font-size: 14px !important;
  line-height: 1 !important;
}

.machine-picker-dialog .picker-tool-btn.el-button--default.is-circle {
  background-color: var(--el-fill-color-light) !important;
  border-color: var(--el-border-color) !important;
}

.machine-picker-dialog .picker-tool-btn.el-button.is-circle .el-icon {
  width: 14px !important;
  height: 14px !important;
  font-size: 14px !important;
  margin: 0 !important;
}

.machine-picker-dialog .el-dialog__body {
  padding-top: 12px;
}
</style>
