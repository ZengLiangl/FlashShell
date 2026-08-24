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
      <el-button text circle class="machine-aside-close" @click="$emit('close')">
        <el-icon><Close /></el-icon>
      </el-button>
    </header>

    <div class="machine-aside-body">
      <el-form
        ref="formRef"
        class="machine-edit-form"
        :model="form"
        :rules="rules"
        label-position="top"
        require-asterisk-position="right"
      >
        <section class="machine-form-section">
          <header class="machine-form-section-head">
            <el-icon><Monitor /></el-icon>
            <span>通用</span>
          </header>
          <div class="machine-form-section-body">
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
                placeholder="选择或新建分组"
                style="width: 100%"
              >
                <el-option v-for="g in groupOptions" :key="g" :label="g" :value="g === DEFAULT_MACHINE_GROUP ? '' : g" />
              </el-select>
              <el-button class="section-link-btn" text type="primary" @click="applyGroupDefaults">
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
                placeholder="添加标签…"
                style="width: 100%"
              />
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="form.notes" type="textarea" :rows="3" maxlength="4000" show-word-limit placeholder="运维备注（可选）" />
            </el-form-item>
          </div>
        </section>

        <section class="machine-form-section">
          <header class="machine-form-section-head">
            <el-icon><Location /></el-icon>
            <span>地址</span>
          </header>
          <div class="machine-form-section-body">
            <el-form-item label="主机" prop="host">
              <el-input v-model="form.host" placeholder="IP 或主机名" />
            </el-form-item>
          </div>
        </section>

        <section class="machine-form-section">
          <header class="machine-form-section-head">
            <el-icon><Key /></el-icon>
            <span>端口与凭据</span>
          </header>
          <div class="machine-form-section-body">
            <div class="machine-form-row-2">
              <el-form-item label="用户名" prop="user">
                <el-input v-model="form.user" placeholder="SSH 用户名" />
              </el-form-item>
              <el-form-item label="端口" prop="port">
                <el-input-number v-model="form.port" :min="1" :max="65535" controls-position="right" />
              </el-form-item>
            </div>
            <el-form-item label="全局帐号">
              <el-select
                v-model="selectedAccountId"
                clearable
                placeholder="选择身份自动填充"
                style="width: 100%"
                @change="applyGlobalAccount"
              >
                <el-option v-for="account in globalAccounts" :key="account.id" :label="account.name" :value="account.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="密码">
              <el-input v-model="form.password" type="password" show-password placeholder="可选" />
            </el-form-item>
            <el-form-item label="密钥文件">
              <div class="key-file-input">
                <el-input v-model="form.key_file" placeholder="私钥路径" readonly />
                <el-button type="primary" @click="selectKeyFile">
                  <el-icon><Folder /></el-icon>
                </el-button>
              </div>
            </el-form-item>
            <el-form-item label="密钥口令">
              <el-input v-model="form.keyPassphrase" type="password" show-password placeholder="加密私钥口令" clearable />
            </el-form-item>
          </div>
        </section>

        <section class="machine-form-section">
          <header class="machine-form-section-head">
            <el-icon><Setting /></el-icon>
            <span>高级选项</span>
          </header>
          <div class="machine-form-section-body">
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
            <el-form-item label="本地回显">
              <el-switch v-model="form.localEcho" active-text="高延迟时本机立即显示输入" />
            </el-form-item>
          </div>
        </section>
      </el-form>
    </div>

    <footer class="machine-aside-footer">
      <div class="machine-aside-footer-left">
        <el-button
          v-if="!editing"
          class="machine-aside-footer-btn"
          :loading="importing"
          title="从 ~/.ssh/config 导入"
          @click="importLocalSSHConfig"
        >
          导入本地 SSH
        </el-button>
      </div>
      <div class="machine-aside-footer-right">
        <el-button class="machine-aside-footer-btn" @click="$emit('close')">取消</el-button>
        <el-button class="machine-aside-footer-btn" :loading="testing" @click="testDraft">测试连接</el-button>
        <el-button
          v-if="editing"
          class="machine-aside-footer-btn"
          type="primary"
          plain
          :loading="connecting"
          @click="saveAndConnect"
        >
          保存并连接
        </el-button>
        <el-button class="machine-aside-footer-btn" type="primary" :loading="saving" @click="save">保存</el-button>
      </div>
    </footer>
  </aside>
</template>

<script>
import { ref, reactive, watch, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Close, Folder, Monitor, Location, Key, Setting } from '@element-plus/icons-vue'
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
  ImportOpenSSHConfigDefault,
} from '../../wailsjs/go/app/App'
import {
  DEFAULT_MACHINE_GROUP,
  normalizeMachineTags,
  collectMachineTags,
} from '../utils/machineGroups'
import { TERMINAL_PRESETS } from '../utils/themePresets'

export default {
  name: 'MachineAsidePanel',
  components: { ArrowLeft, Close, Folder, Monitor, Location, Key, Setting },
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
    const importing = ref(false)
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
      keyPassphrase: '',
      proxyJump: '',
      startupCommand: '',
      terminalPreset: '',
      localEcho: false,
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
      form.keyPassphrase = ''
      form.proxyJump = ''
      form.startupCommand = ''
      form.terminalPreset = ''
      form.localEcho = false
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
      form.localEcho = !!machine.localEcho
      selectedAccountId.value = form.identityId || ''
      if (machine.id) {
        try {
          const sensitive = await GetMachineSensitiveData(machine.id)
          if (sensitive) {
            form.host = sensitive.host || ''
            form.port = sensitive.port || 22
            form.user = sensitive.user || ''
            form.password = sensitive.password || ''
            form.keyPassphrase = sensitive.keyPassphrase || ''
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
      if (defaults.localEcho) form.localEcho = true
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
        localEcho: !!form.localEcho,
      }
      const sensitiveData = {
        host: form.host,
        port: form.port,
        user: form.user,
        password: form.password,
        keyPassphrase: form.keyPassphrase || '',
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
      } catch {
        // 表单校验失败时字段下方已有红字提示，勿再弹系统错误
        return
      }
      testing.value = true
      try {
        const { machineData, sensitiveData } = buildPayload()
        await TestMachineDraftConnection(machineData, sensitiveData)
        ElMessage.success('连接测试成功')
      } catch (error) {
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

    const importLocalSSHConfig = async () => {
      importing.value = true
      try {
        const result = await ImportOpenSSHConfigDefault('', normalizeGroup(form.group))
        if (!result) return
        const errors = result?.errors?.length ? `\n失败: ${result.errors.join('\n')}` : ''
        ElMessage.success(`导入完成：新增 ${result?.added || 0}，更新 ${result?.updated || 0}，跳过 ${result?.skipped || 0}${errors}`)
        emit('saved')
        emit('close')
      } catch (error) {
        ElMessage.error('导入失败: ' + (error?.message || error))
      } finally {
        importing.value = false
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
      importing,
      globalAccounts,
      groupOptions,
      knownTagOptions,
      selectedAccountId,
      applyGlobalAccount,
      applyGroupDefaults,
      selectKeyFile,
      importLocalSSHConfig,
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
  width: min(420px, 100%);
  z-index: 40;
  display: flex;
  flex-direction: column;
  background: var(--app-inset-bg, var(--app-panel-bg));
  border-left: 1px solid color-mix(in srgb, var(--app-border) 80%, transparent);
  box-shadow: -12px 0 32px rgba(0, 0, 0, 0.22);
}

.machine-aside-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 16px 18px 14px;
  border-bottom: 1px solid color-mix(in srgb, var(--app-border) 70%, transparent);
  background: color-mix(in srgb, var(--app-panel-bg) 88%, transparent);
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
  font-size: 16px;
  font-weight: 650;
  letter-spacing: 0.01em;
  color: var(--app-text);
}

.machine-aside-subtitle {
  margin: 2px 0 0;
  font-size: 12px;
  color: var(--app-text-muted);
}

.machine-aside-close {
  color: var(--app-text-muted);
}

.machine-aside-body {
  flex: 1;
  overflow: auto;
  padding: 16px;
}

.machine-aside-footer {
  display: flex;
  flex-wrap: nowrap;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
  padding: 12px 16px 14px;
  border-top: 1px solid color-mix(in srgb, var(--app-border) 70%, transparent);
  background: color-mix(in srgb, var(--app-panel-bg) 92%, transparent);
  flex-shrink: 0;
}

.machine-aside-footer-left,
.machine-aside-footer-right {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 8px;
}

.machine-aside-footer-left {
  flex: 1 1 auto;
  min-width: 0;
  justify-content: flex-start;
}

.machine-aside-footer-right {
  flex: 0 0 auto;
  justify-content: flex-end;
}

.machine-aside-footer-btn {
  margin: 0 !important;
  height: 32px;
  padding: 0 14px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
}

.machine-aside-footer :deep(.el-button + .el-button) {
  margin-left: 0;
}
</style>
