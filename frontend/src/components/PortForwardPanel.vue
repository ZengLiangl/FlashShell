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
      <el-table-column label="状态" width="100" align="center">
        <template #default="{ row }">
          <el-tag v-if="row._active" size="small" type="success">运行中</el-tag>
          <el-tag v-else-if="row._error" size="small" type="danger" :title="row._error">异常</el-tag>
          <el-tag v-else size="small" type="info">未启动</el-tag>
        </template>
      </el-table-column>
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

    <el-dialog v-model="editVisible" :title="editing?.id ? '编辑规则' : '添加规则'" width="560px" append-to-body>
      <div class="pf-wizard-steps">
        <button type="button" class="pf-step" :class="{ active: wizardStep === 1 }" @click="wizardStep = 1">1. 类型与机器</button>
        <button type="button" class="pf-step" :class="{ active: wizardStep === 2 }" @click="wizardStep = 2">2. 地址端口</button>
        <button type="button" class="pf-step" :class="{ active: wizardStep === 3 }" @click="wizardStep = 3">3. 选项</button>
      </div>
      <el-form :model="editForm" label-width="96px" size="small">
        <template v-if="wizardStep === 1">
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
        </template>
        <template v-else-if="wizardStep === 2">
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
        </template>
        <template v-else>
        <el-form-item label="启用">
          <el-switch v-model="editForm.enabled" />
        </el-form-item>
        <el-form-item label="自动启动">
          <el-switch v-model="editForm.autoStart" />
        </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button v-if="wizardStep > 1" @click="wizardStep -= 1">上一步</el-button>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button v-if="wizardStep < 3" type="primary" @click="wizardStep += 1">下一步</el-button>
        <el-button v-else type="primary" :loading="saving" @click="confirmEdit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, watch, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import {
  ListPortForwards,
  SavePortForwards,
  StartPortForward,
  StopPortForward,
  GetPortForwardStatus,
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
    const wizardStep = ref(1)
    const editForm = reactive(emptyRule())
    let statusTimer = null

    const refreshStatus = async () => {
      for (const row of rules.value) {
        if (!row?.id) continue
        try {
          const list = await GetPortForwardStatus(row.id)
          const active = (list || []).some((s) => s.active)
          const err = (list || []).find((s) => s.error)?.error || ''
          row._active = active
          row._error = err
        } catch {
          row._active = false
        }
      }
    }

    const load = async () => {
      const empty = rules.value.length === 0
      let spinnerTimer = null
      if (empty) {
        spinnerTimer = setTimeout(() => {
          loading.value = true
        }, 160)
      }
      try {
        rules.value = (await ListPortForwards() || []).map((r) => ({ ...r, _starting: false, _active: false, _error: '' }))
        machines.value = await GetMachines() || []
        await refreshStatus()
      } catch (error) {
        ElMessage.error('加载端口转发失败: ' + (error?.message || error))
      } finally {
        if (spinnerTimer) clearTimeout(spinnerTimer)
        loading.value = false
      }
    }

    const persist = async () => {
      saving.value = true
      try {
        const payload = rules.value.map(({ _starting, _active, _error, ...rest }) => rest)
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
      wizardStep.value = 1
      editVisible.value = true
    }

    const editRule = (row) => {
      editing.value = row
      Object.assign(editForm, { ...row })
      wizardStep.value = 1
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
        await refreshStatus()
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
        await refreshStatus()
      } catch (error) {
        ElMessage.error('停止失败: ' + (error?.message || error))
      }
    }

    watch(() => props.active, (active) => {
      if (statusTimer) {
        clearInterval(statusTimer)
        statusTimer = null
      }
      if (active) {
        load()
        statusTimer = setInterval(refreshStatus, 3000)
      }
    }, { immediate: true })

    onUnmounted(() => {
      if (statusTimer) clearInterval(statusTimer)
    })

    return {
      loading,
      saving,
      rules,
      machines,
      editVisible,
      editing,
      editForm,
      wizardStep,
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

.pf-wizard-steps {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 6px;
  margin-bottom: 14px;
}

.pf-step {
  border: 1px solid var(--app-border, #e4e7ed);
  background: var(--app-panel-bg, #f5f7fa);
  border-radius: 8px;
  padding: 8px 6px;
  font-size: 12px;
  cursor: pointer;
  color: var(--app-text-muted, #606266);
}

.pf-step.active {
  border-color: var(--app-accent-color, #409eff);
  color: var(--app-accent-color, #409eff);
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 10%, transparent);
  font-weight: 600;
}
</style>
