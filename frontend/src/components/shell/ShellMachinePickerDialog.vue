<template>
  <el-dialog
    v-model="visibleProxy"
    title="连接管理器"
    width="560px"
    class="machine-picker-dialog"
    append-to-body
  >
    <div class="app-toolbar picker-toolbar">
      <el-input
        v-model="keyword"
        clearable
        placeholder="搜索名称 / IP"
        size="small"
        class="app-toolbar-search"
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

    <div class="picker-body">
      <MachineConnectList
        :machines="filteredMachines"
        :sessions="sessions"
        :workspace-sessions="workspaceSessions"
        :connecting-name="connectingName"
        :filter-keyword="keyword"
        show-edit
        empty-text="暂无机器配置"
        @connect="onConnect"
        @edit-machine="(m) => $emit('edit-machine', m)"
      />
    </div>
  </el-dialog>
</template>

<script>
import { computed, ref } from 'vue'
import { Monitor, Plus, Search } from '@element-plus/icons-vue'
import { machineMatchesKeyword } from '../../utils/machineGroups'
import MachineConnectList from './MachineConnectList.vue'

export default {
  name: 'ShellMachinePickerDialog',
  components: { MachineConnectList, Search },
  props: {
    modelValue: { type: Boolean, default: false },
    machines: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
    workspaceSessions: { type: Array, default: () => [] },
    connectingName: { type: String, default: '' },
  },
  emits: ['update:modelValue', 'connect', 'edit-machine', 'add-machine', 'add-local'],
  setup(props, { emit }) {
    const keyword = ref('')

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

    const onConnect = (name) => emit('connect', name)

    const onAddLocal = () => {
      emit('add-local')
      visibleProxy.value = false
    }

    return {
      Monitor,
      Plus,
      visibleProxy,
      keyword,
      filteredMachines,
      onConnect,
      onAddLocal,
    }
  },
}
</script>

<style scoped>
.picker-toolbar {
  margin-bottom: 12px;
}

.picker-body {
  max-height: 480px;
  overflow: auto;
}
</style>

<!-- append-to-body 弹窗：统一圆钮盒模型，避免 default / primary 视觉大小不一致 -->
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

/* default 若背景与面板同色，只剩描边会显得更小；补实心底色让外径与 primary 一致 */
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
</style>
