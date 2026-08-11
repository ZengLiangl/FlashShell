<template>
  <div class="port-forward-panel">
    <div class="panel-head">
      <div>
        <h4>独立端口转发</h4>
        <p class="panel-desc">不依赖 Shell 会话的 SSH 隧道；可设置开机自动启动</p>
      </div>
      <el-button type="primary" size="small" @click="addRule">
        <el-icon><Plus /></el-icon>
        添加规则
      </el-button>
    </div>

    <el-table :data="rules" size="small" empty-text="暂无端口转发规则" v-loading="loading">
      <el-table-column prop="name" label="名称" min-width="120" />
      <el-table-column prop="type" label="类型" width="80" />
      <el-table-column label="本地" min-width="120">
        <template #default="{ row }">{{ row.localHost || '127.0.0.1' }}:{{ row.localPort }}</template>
      </el-table-column>
      <el-table-column label="远端" min-width="120">
        <template #default="{ row }">
          <span v-if="row.type === 'dynamic'">SOCKS5</span>
          <span v-else>{{ row.remoteHost || '-' }}:{{ row.remotePort || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="machineName" label="关联机器" min-width="100" />
      <el-table-column label="自动启动" width="88" align="center">
        <template #default="{ row }">
          <el-switch v-model="row.autoStart" size="small" @change="persist" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" align="center">
        <template #default="{ row }">
          <el-button size="small" text type="primary" @click="editRule(row)">编辑</el-button>
          <el-button size="small" text type="success" :loading="row._starting" @click="startRule(row)">启动</el-button>
          <el-button size="small" text type="warning" @click="stopRule(row)">停止</el-button>
          <el-button size="small" text type="danger" @click="removeRule(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="editVisible" :title="editing?.id ? '编辑规则' : '添加规则'" width="520px" append-to-body>
      <el-form :model="editForm" label-width="96px" size="small">
        <el-form-item label="名称">
          <el-input v-model="editForm.name" placeholder="规则名称" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="editForm.type" style="width: 100%">
            <el-option label="本地转发" value="local" />
            <el-option label="远程转发" value="remote" />
            <el-option label="动态 SOCKS" value="dynamic" />
          </el-select>
        </el-form-item>
        <el-form-item label="关联机器">
          <el-select v-model="editForm.machineName" filterable placeholder="选择机器" style="width: 100%">
            <el-option v-for="m in machines" :key="m.id || m.name" :label="m.name" :value="m.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="本地地址">
          <el-input v-model="editForm.localHost" placeholder="127.0.0.1" />
        </el-form-item>
        <el-form-item label="本地端口">
          <el-input-number v-model="editForm.localPort" :min="1" :max="65535" style="width: 100%" />
        </el-form-item>
        <template v-if="editForm.type !== 'dynamic'">
          <el-form-item label="远端地址">
            <el-input v-model="editForm.remoteHost" placeholder="127.0.0.1" />
          </el-form-item>
          <el-form-item label="远端端口">
            <el-input-number v-model="editForm.remotePort" :min="1" :max="65535" style="width: 100%" />
          </el-form-item>
        </template>
        <el-form-item label="启用">
          <el-switch v-model="editForm.enabled" />
        </el-form-item>
        <el-form-item label="自动启动">
          <el-switch v-model="editForm.autoStart" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="confirmEdit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import {
  ListPortForwards,
  SavePortForwards,
  StartPortForward,
  StopPortForward,
  GetMachines,
} from '../../wailsjs/go/app/App'

const emptyRule = () => ({
  id: crypto.randomUUID(),
  name: '',
  type: 'local',
  localHost: '127.0.0.1',
  localPort: 0,
  remoteHost: '127.0.0.1',
  remotePort: 0,
  machineName: '',
  enabled: true,
  autoStart: false,
})

export default {
  name: 'PortForwardPanel',
  components: { Plus },
  props: {
    active: { type: Boolean, default: false },
  },
  setup(props) {
    const loading = ref(false)
    const saving = ref(false)
    const rules = ref([])
    const machines = ref([])
    const editVisible = ref(false)
    const editing = ref(null)
    const editForm = reactive(emptyRule())

    const load = async () => {
      loading.value = true
      try {
        rules.value = (await ListPortForwards() || []).map((r) => ({ ...r, _starting: false }))
        machines.value = await GetMachines() || []
      } catch (error) {
        ElMessage.error('加载端口转发失败: ' + (error?.message || error))
      } finally {
        loading.value = false
      }
    }

    const persist = async () => {
      saving.value = true
      try {
        const payload = rules.value.map(({ _starting, ...rest }) => rest)
        await SavePortForwards(payload)
      } catch (error) {
        ElMessage.error('保存失败: ' + (error?.message || error))
        await load()
      } finally {
        saving.value = false
      }
    }

    const addRule = () => {
      editing.value = null
      Object.assign(editForm, emptyRule())
      editVisible.value = true
    }

    const editRule = (row) => {
      editing.value = row
      Object.assign(editForm, { ...row })
      editVisible.value = true
    }

    const confirmEdit = async () => {
      if (!editForm.name?.trim()) {
        ElMessage.warning('请填写规则名称')
        return
      }
      if (!editForm.machineName?.trim()) {
        ElMessage.warning('请选择关联机器')
        return
      }
      if (!editForm.localPort) {
        ElMessage.warning('请填写本地端口')
        return
      }
      const payload = { ...editForm, name: editForm.name.trim(), machineName: editForm.machineName.trim() }
      if (editing.value?.id) {
        const idx = rules.value.findIndex((r) => r.id === editing.value.id)
        if (idx >= 0) rules.value[idx] = { ...rules.value[idx], ...payload }
      } else {
        rules.value.push({ ...payload, _starting: false })
      }
      saving.value = true
      try {
        await persist()
        editVisible.value = false
        ElMessage.success('规则已保存')
      } finally {
        saving.value = false
      }
    }

    const removeRule = async (row) => {
      try {
        await ElMessageBox.confirm(`确定删除规则「${row.name}」吗？`, '确认删除', { type: 'warning' })
        await StopPortForward(row.id)
        rules.value = rules.value.filter((r) => r.id !== row.id)
        await persist()
      } catch (error) {
        if (error !== 'cancel') ElMessage.error('删除失败: ' + (error?.message || error))
      }
    }

    const startRule = async (row) => {
      row._starting = true
      try {
        await StartPortForward(row.id)
        ElMessage.success('端口转发已启动')
      } catch (error) {
        ElMessage.error('启动失败: ' + (error?.message || error))
      } finally {
        row._starting = false
      }
    }

    const stopRule = async (row) => {
      try {
        await StopPortForward(row.id)
        ElMessage.success('端口转发已停止')
      } catch (error) {
        ElMessage.error('停止失败: ' + (error?.message || error))
      }
    }

    watch(() => props.active, (active) => {
      if (active) load()
    }, { immediate: true })

    return {
      loading,
      saving,
      rules,
      machines,
      editVisible,
      editing,
      editForm,
      addRule,
      editRule,
      confirmEdit,
      removeRule,
      startRule,
      stopRule,
      persist,
    }
  },
}
</script>

<style scoped>
.port-forward-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.panel-head h4 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}

.panel-desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--app-text-muted);
}
</style>
