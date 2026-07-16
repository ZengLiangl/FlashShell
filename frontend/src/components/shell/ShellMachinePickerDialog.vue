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
        <el-tooltip content="添加机器" placement="top">
          <el-button size="small" type="primary" circle @click="$emit('add-machine')">
            <el-icon><Plus /></el-icon>
          </el-button>
        </el-tooltip>
      </div>
    </div>

    <div class="picker-body">
      <MachineConnectList
        :machines="filteredMachines"
        :sessions="sessions"
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
import { machineMatchesKeyword } from '../../utils/machineGroups'
import MachineConnectList from './MachineConnectList.vue'

export default {
  name: 'ShellMachinePickerDialog',
  components: { MachineConnectList },
  props: {
    modelValue: { type: Boolean, default: false },
    machines: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
    connectingName: { type: String, default: '' },
  },
  emits: ['update:modelValue', 'connect', 'edit-machine', 'add-machine'],
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

    return {
      visibleProxy,
      keyword,
      filteredMachines,
      onConnect,
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
