<template>
  <aside v-if="open" class="machine-aside-panel">
    <header class="machine-aside-header">
      <div class="machine-aside-header-main">
        <el-button v-if="showBack" text circle class="machine-aside-back" @click="$emit('back')">
          <el-icon><ArrowLeft /></el-icon>
        </el-button>
        <div class="machine-aside-titles">
          <h3 class="machine-aside-title">{{ title }}</h3>
          <p v-if="subtitle" class="machine-aside-subtitle">{{ subtitle }}</p>
        </div>
      </div>
      <el-button text circle @click="$emit('close')">
        <el-icon><Close /></el-icon>
      </el-button>
    </header>

    <div class="machine-aside-body">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="88px" size="small">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="机器名称" />
        </el-form-item>
        <el-form-item label="分组">
          <el-select
            v-model="form.group"
            clearable
            filterable
            allow-create
            default-first-option
            placeholder="选择分组"
            style="width: 100%"
          >
            <el-option v-for="g in groupOptions" :key="g" :label="g" :value="g === DEFAULT_MACHINE_GROUP ? '' : g" />
          </el-select>
          <el-button class="group-default-btn" text type="primary" size="small" @click="applyGroupDefaults">
            应用分组默认
          </el-button>
        </el-form-item>
        <el-form-item label="标签">
          <el-select
            v-model="form.tags"
            multiple
            filterable
            allow-create
            default-first-option
            collapse-tags
            collapse-tags-tooltip
            placeholder="标签"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.notes" type="textarea" :rows="2" maxlength="4000" show-word-limit />
        </el-form-item>
        <el-form-item label="全局帐号">
          <el-select
            v-model="selectedAccountId"
            clearable
            placeholder="选择身份"
            style="width: 100%"
            @change="applyGlobalAccount"
          >
            <el-option v-for="account in globalAccounts" :key="account.id" :label="account.name" :value="account.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="密钥文件">
          <div class="key-file-input">
            <el-input v-model="form.key_file" placeholder="私钥路径" readonly />
            <el-button type="primary" circle @click="selectKeyFile">
              <el-icon><Folder /></el-icon>
            </el-button>
          </div>
        </el-form-item>
        <el-divider content-position="left">连接</el-divider>
        <el-form-item label="主机" prop="host">
          <el-input v-model="form.host" placeholder="主机地址" />
        </el-form-item>
        <el-form-item label="端口" prop="port">
          <el-input-number v-model="form.port" :min="1" :max="65535" style="width: 100%" />
        </el-form-item>
        <el-form-item label="用户名" prop="user">
          <el-input v-model="form.user" placeholder="SSH 用户名" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" show-password placeholder="可选" />
        </el-form-item>
        <el-form-item label="跳板机">
          <el-input v-model="form.proxyJump" placeholder="机器名或 host[:port]" clearable />
        </el-form-item>
        <el-form-item label="启动命令">
          <el-input v-model="form.startupCommand" placeholder="连接后执行" clearable />
        </el-form-item>
        <el-form-item label="终端配色">
          <el-select v-model="form.terminalPreset" clearable placeholder="跟随全局主题" style="width: 100%">
            <el-option label="跟随全局" value="" />
            <el-option v-for="preset in terminalPresetOptions" :key="preset.id" :label="preset.label" :value="preset.id" />
          </el-select>
        </el-form-item>
      </el-form>
    </div>

    <footer class="machine-aside-footer">
      <el-button @click="$emit('close')">取消</el-button>
      <el-button :loading="testing" @click="testDraft">测试连接</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      <el-button v-if="editing" type="success" :loading="connecting" @click="saveAndConnect">保存并连接</el-button>
    </footer>
  </aside>
</template>

<script>
import { ref, reactive, watch, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Close, Folder } from '@element-plus/icons-vue'
import {
  GetMachineGroups,
  GetMachineGroupDefaults,
  GetGlobalAccounts,
  GetMachineSensitiveData,
  CreateMachine,
  UpdateMachine,
  SetMachineSensitiveData,
  TestMachineDraftConnection,
  SelectKeyFile,
} from '../../wailsjs/go/app/App'
import {
  DEFAULT_MACHINE_GROUP,
  normalizeMachineTags,
  collectMachineTags,
} from '../utils/machineGroups'
import { TERMINAL_PRESETS } from '../utils/themePresets'

export default {
  name: 'MachineAsidePanel',
  components: { ArrowLeft, Close, Folder },
  props: {
    open: { type: Boolean, default: false },
    machine: { type: Object, default: null },
    machines: { type: Array, default: () => [] },
    showBack: { type: Boolean, default: false },
    subtitle: { type: String, default: '' },
  },
  emits: ['close', 'back', 'saved', 'connect'],
  setup(props, { emit }) {
    const formRef = ref(null)
    const saving = ref(false)
    const testing = ref(false)
    const connecting = ref(false)
    const globalAccounts = ref([])
    const machineGroups = ref([])
    const groupDefaults = ref([])
    const selectedAccountId = ref('')
    const editing = computed(() => !!props.machine?.id)

    const title = computed(() => (editing.value ? '编辑机器' : '添加机器'))

    const form = reactive({
      name: '',
      group: '',
      tags: [],
      notes: '',
      identityId: '',
      key_file: '',
      host: '',
      port: 22,
      user: '',
      password: '',
      proxyJump: '',
      startupCommand: '',
      terminalPreset: '',
    })

    const rules = {
      name: [{ required: true, message: '请输入机器名称', trigger: 'blur' }],
      host: [{ required: true, message: '请输入主机地址', trigger: 'blur' }],
      port: [{ required: true, message: '请输入端口', trigger: 'blur' }],
      user: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
    }

    const groupOptions = computed(() => {
      const set = new Set([DEFAULT_MACHINE_GROUP])
      for (const g of machineGroups.value || []) if (g) set.add(g)
      for (const m of props.machines || []) if (m.group) set.add(m.group)
      return Array.from(set).sort((a, b) => {
        if (a === DEFAULT_MACHINE_GROUP) return -1
        if (b === DEFAULT_MACHINE_GROUP) return 1
        return a.localeCompare(b, 'zh-CN')
      })
    })

    const knownTagOptions = computed(() => collectMachineTags(props.machines))

    const normalizeGroup = (g) => {
      const s = String(g || '').trim()
      if (!s || s === DEFAULT_MACHINE_GROUP) return ''
      return s
    }

    const resetForm = () => {
      form.name = ''
      form.group = ''
      form.tags = []
      form.notes = ''
      form.identityId = ''
      form.key_file = ''
      form.host = ''
      form.port = 22
      form.user = ''
      form.password = ''
      form.proxyJump = ''
      form.startupCommand = ''
      form.terminalPreset = ''
      selectedAccountId.value = ''
    }

    const loadMeta = async () => {
      try {
        machineGroups.value = await GetMachineGroups() || []
      } catch {
        machineGroups.value = []
      }
      try {
        groupDefaults.value = await GetMachineGroupDefaults() || []
      } catch {
        groupDefaults.value = []
      }
      try {
        globalAccounts.value = await GetGlobalAccounts() || []
      } catch {
        globalAccounts.value = []
      }
    }

    const fillForm = async (machine) => {
      resetForm()
      if (!machine) return
      form.name = machine.name || ''
      form.group = machine.group || ''
      form.tags = normalizeMachineTags(machine.tags)
      form.notes = machine.notes || ''
      form.identityId = machine.identityId || ''
      form.key_file = machine.key_file || ''
      form.proxyJump = machine.proxyJump || ''
      form.startupCommand = machine.startupCommand || ''
      form.terminalPreset = machine.terminalPreset || ''
      selectedAccountId.value = form.identityId || ''
      if (machine.id) {
        try {
          const sensitive = await GetMachineSensitiveData(machine.id)
          if (sensitive) {
            form.host = sensitive.host || ''
            form.port = sensitive.port || 22
            form.user = sensitive.user || ''
            form.password = sensitive.password || ''
          }
        } catch {
          ElMessage.warning('获取敏感数据失败，请重新输入')
        }
      }
    }

    const applyGlobalAccount = (accountId) => {
      form.identityId = accountId || ''
      if (!accountId) return
      const account = globalAccounts.value.find((item) => item.id === accountId)
      if (!account) return
      form.user = account.user || ''
      form.password = account.password || ''
      if (account.keyFile) form.key_file = account.keyFile
    }

    const applyGroupDefaults = () => {
      const groupName = normalizeGroup(form.group)
      const defaults = groupDefaults.value.find((item) => normalizeGroup(item.name) === groupName)
      if (!defaults) {
        ElMessage.info('当前分组暂无默认配置')
        return
      }
      if (defaults.user) form.user = defaults.user
      if (defaults.keyFile) form.key_file = defaults.keyFile
      if (defaults.proxyJump) form.proxyJump = defaults.proxyJump
      if (defaults.startupCommand) form.startupCommand = defaults.startupCommand
      if (defaults.sftpEncoding) {
        // aside 表单未展示编码，但保留应用入口一致性
      }
      if (defaults.tags?.length) form.tags = normalizeMachineTags(defaults.tags)
      ElMessage.success('已应用分组默认')
    }

    const buildPayload = () => {
      const machineData = {
        name: form.name,
        group: normalizeGroup(form.group),
        tags: normalizeMachineTags(form.tags),
        notes: String(form.notes || '').trim(),
        identityId: form.identityId || '',
        key_file: form.key_file,
        proxyJump: form.proxyJump?.trim() || '',
        startupCommand: form.startupCommand?.trim() || '',
        terminalPreset: form.terminalPreset || '',
      }
      const sensitiveData = {
        host: form.host,
        port: form.port,
        user: form.user,
        password: form.password,
      }
      return { machineData, sensitiveData }
    }

    const save = async () => {
      if (!formRef.value) return false
      try {
        await formRef.value.validate()
      } catch {
        return false
      }
      saving.value = true
      try {
        const { machineData, sensitiveData } = buildPayload()
        if (editing.value) {
          machineData.id = props.machine.id
          await UpdateMachine(props.machine.id, machineData)
          await SetMachineSensitiveData(props.machine.id, sensitiveData)
          ElMessage.success('机器配置已更新')
        } else {
          await CreateMachine(machineData, sensitiveData)
          ElMessage.success('机器配置已添加')
        }
        emit('saved')
        emit('close')
        return true
      } catch (error) {
        ElMessage.error('保存失败: ' + (error?.message || error))
        return false
      } finally {
        saving.value = false
      }
    }

    const testDraft = async () => {
      if (!formRef.value) return
      try {
        await formRef.value.validate()
        testing.value = true
        const { machineData, sensitiveData } = buildPayload()
        await TestMachineDraftConnection(machineData, sensitiveData)
        ElMessage.success('连接测试成功')
      } catch (error) {
        if (error === false || error?.fields) return
        ElMessage.error('连接测试失败: ' + (error?.message || error))
      } finally {
        testing.value = false
      }
    }

    const saveAndConnect = async () => {
      const name = String(form.name || '').trim()
      if (!name) {
        ElMessage.warning('请先填写机器名称')
        return
      }
      connecting.value = true
      try {
        const ok = await save()
        if (ok) emit('connect', name)
      } finally {
        connecting.value = false
      }
    }

    const selectKeyFile = async () => {
      try {
        const filePath = await SelectKeyFile()
        if (filePath) form.key_file = filePath
      } catch (error) {
        ElMessage.error('选择密钥文件失败: ' + (error?.message || error))
      }
    }

    watch(() => props.open, async (open) => {
      if (!open) return
      await loadMeta()
      await fillForm(props.machine)
    })

    watch(() => props.machine, async (machine) => {
      if (!props.open) return
      await fillForm(machine)
    })

    return {
      DEFAULT_MACHINE_GROUP,
      formRef,
      form,
      rules,
      title,
      editing,
      saving,
      testing,
      connecting,
      globalAccounts,
      groupOptions,
      knownTagOptions,
      selectedAccountId,
      applyGlobalAccount,
      applyGroupDefaults,
      selectKeyFile,
      terminalPresetOptions: TERMINAL_PRESETS,
      save,
      testDraft,
      saveAndConnect,
    }
  },
}
</script>

<style scoped>
.machine-aside-panel {
  position: absolute;
  top: 40px;
  right: 0;
  bottom: 0;
  width: 400px;
  z-index: 40;
  display: flex;
  flex-direction: column;
  background: var(--app-panel-bg);
  border-left: 1px solid var(--app-border);
  box-shadow: var(--app-surface-shadow);
}

.machine-aside-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
}

.machine-aside-header-main {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.machine-aside-titles {
  min-width: 0;
}

.machine-aside-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--app-text);
}

.machine-aside-subtitle {
  margin: 2px 0 0;
  font-size: 12px;
  color: var(--app-text-muted);
}

.machine-aside-body {
  flex: 1;
  overflow: auto;
  padding: 16px;
}

.machine-aside-footer {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--app-border);
  flex-shrink: 0;
}

.key-file-input {
  display: flex;
  gap: 8px;
  width: 100%;
}

.key-file-input .el-input {
  flex: 1;
}

.group-default-btn {
  margin-top: 4px;
  padding-left: 0;
}
</style>
