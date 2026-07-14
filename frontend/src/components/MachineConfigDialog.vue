<template>
    <div class="machine-config-container" :class="{ embedded }">
        <div class="machine-list">
            <div class="list-header">
                <h4 v-if="!embedded">机器列表</h4>
                <div v-else></div>
                <div class="header-actions">
                    <el-select
                        v-model="importAccountId"
                        clearable
                        placeholder="导入时使用全局帐号(可选)"
                        class="import-account-select"
                    >
                        <el-option
                            v-for="account in globalAccounts"
                            :key="account.id"
                            :label="account.name"
                            :value="account.id"
                        />
                    </el-select>
                    <el-dropdown split-button type="primary" @click="addMachine" @command="handleAddCommand">
                        <el-icon><Plus /></el-icon>
                        添加机器
                        <template #dropdown>
                            <el-dropdown-menu>
                                <el-dropdown-item command="import-xshell">导入 Xshell</el-dropdown-item>
                                <el-dropdown-item command="import-finalshell">导入 FinalShell</el-dropdown-item>
                            </el-dropdown-menu>
                        </template>
                    </el-dropdown>
                </div>
            </div>

            <div class="machine-table-wrap">
                <el-table
                    :data="sortedMachines"
                    row-key="id"
                    style="width: 100%"
                    v-loading="machinesLoading"
                >
                    <el-table-column prop="name" label="机器名称" width="150" />
                    <el-table-column prop="group" label="分组" width="120">
                        <template #default="scope">
                            {{ scope.row.group || '默认分组' }}
                        </template>
                    </el-table-column>
                    <el-table-column prop="key_file" label="密钥文件" overflow-tooltip />
                    <el-table-column label="操作" width="250">
                        <template #default="scope">
                            <el-button size="small" @click="editMachine(scope.row)">编辑</el-button>
                            <el-button size="small" @click="testConnection(scope.row)" :loading="scope.row.testing">
                                测试连接
                            </el-button>
                            <el-button size="small" type="danger" @click="deleteMachine(scope.row)">删除</el-button>
                        </template>
                    </el-table-column>
                </el-table>
            </div>
        </div>

        <el-dialog
            v-model="machineEditVisible"
            :title="editingMachine ? '编辑机器' : '添加机器'"
            width="600px"
            append-to-body
        >
            <el-form :model="machineForm" :rules="machineRules" ref="machineFormRef" label-width="100px">
                <el-form-item label="机器名称" prop="name">
                    <el-input v-model="machineForm.name" placeholder="请输入机器名称" />
                </el-form-item>

                <el-form-item label="分组" prop="group">
                    <el-input v-model="machineForm.group" placeholder="留空则归入默认分组" />
                </el-form-item>

                <el-form-item label="全局帐号">
                    <el-select
                        v-model="selectedAccountId"
                        clearable
                        placeholder="选择后自动填充用户名和密码"
                        style="width: 100%"
                        @change="applyGlobalAccount"
                    >
                        <el-option
                            v-for="account in globalAccounts"
                            :key="account.id"
                            :label="account.name"
                            :value="account.id"
                        />
                    </el-select>
                </el-form-item>

                <el-form-item label="密钥文件" prop="key_file">
                    <div class="key-file-input">
                        <el-input v-model="machineForm.key_file" placeholder="请选择密钥文件" readonly />
                        <el-button type="primary" @click="selectKeyFile">选择文件</el-button>
                    </div>
                </el-form-item>

                <el-divider content-position="left">连接信息</el-divider>

                <el-form-item label="主机地址" prop="host">
                    <el-input v-model="machineForm.host" placeholder="请输入主机地址" />
                </el-form-item>

                <el-form-item label="端口" prop="port">
                    <el-input-number v-model="machineForm.port" :min="1" :max="65535" placeholder="SSH端口" />
                </el-form-item>

                <el-form-item label="用户名" prop="user">
                    <el-input v-model="machineForm.user" placeholder="请输入用户名" />
                </el-form-item>

                <el-form-item label="密码" prop="password">
                    <el-input v-model="machineForm.password" type="password" placeholder="请输入密码（可选）" show-password />
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="machineEditVisible = false">取消</el-button>
                    <el-button type="primary" @click="saveMachine" :loading="savingMachine">
                        {{ editingMachine ? '更新' : '添加' }}
                    </el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script>
import { ref, reactive, watch, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
    GetMachines,
    GetGlobalAccounts,
    GetMachineSensitiveData,
    UpdateMachine,
    SetMachineSensitiveData,
    CreateMachine,
    DeleteMachine,
    TestMachineConnection,
    SelectKeyFile,
    ImportXshellPick,
    ImportFinalShellPick,
} from '../../wailsjs/go/app/App'
import { sortMachinesByName } from '../utils/machineGroups'

export default {
    name: 'MachineConfigDialog',
    props: {
        modelValue: { type: Boolean, default: false },
        embedded: { type: Boolean, default: false },
        active: { type: Boolean, default: false },
        editMachineId: { type: String, default: '' },
    },
    emits: ['update:modelValue', 'closed', 'changed'],
    setup(props, { emit }) {
        const visibleProxy = ref(props.modelValue)
        const machines = ref([])
        const sortedMachines = computed(() => sortMachinesByName(machines.value))
        const globalAccounts = ref([])
        const machinesLoading = ref(false)
        const machineEditVisible = ref(false)
        const savingMachine = ref(false)
        const editingMachine = ref(null)
        const machineFormRef = ref(null)
        const selectedAccountId = ref('')
        const importAccountId = ref('')

        const machineForm = reactive({
            name: '',
            group: '',
            key_file: '',
            host: '',
            port: 22,
            user: '',
            password: ''
        })

        const machineRules = {
            name: [{ required: true, message: '请输入机器名称', trigger: 'blur' }],
            host: [{ required: true, message: '请输入主机地址', trigger: 'blur' }],
            port: [{ required: true, message: '请输入端口', trigger: 'blur' }],
            user: [{ required: true, message: '请输入用户名', trigger: 'blur' }]
        }

        const handleClose = () => {
            visibleProxy.value = false
        }

        const loadMachines = async () => {
            try {
                machinesLoading.value = true
                const machinesData = await GetMachines()
                machines.value = machinesData || []
            } catch (error) {
                console.error('加载机器配置失败:', error)
                ElMessage.error('加载机器配置失败: ' + error.message)
            } finally {
                machinesLoading.value = false
            }
        }

        const loadGlobalAccounts = async () => {
            try {
                globalAccounts.value = await GetGlobalAccounts() || []
            } catch {
                globalAccounts.value = []
            }
        }

        const applyGlobalAccount = (accountId) => {
            if (!accountId) return
            const account = globalAccounts.value.find((item) => item.id === accountId)
            if (!account) return
            machineForm.user = account.user || ''
            machineForm.password = account.password || ''
        }

        const addMachine = () => {
            editingMachine.value = null
            selectedAccountId.value = ''
            resetMachineForm()
            machineEditVisible.value = true
        }

        const editMachine = async (machine) => {
            editingMachine.value = machine
            selectedAccountId.value = ''
            machineForm.name = machine.name
            machineForm.group = machine.group || ''
            machineForm.key_file = machine.key_file || ''
            try {
                const sensitiveData = await GetMachineSensitiveData(machine.id)
                if (sensitiveData) {
                    machineForm.host = sensitiveData.host || ''
                    machineForm.port = sensitiveData.port || 22
                    machineForm.user = sensitiveData.user || ''
                    machineForm.password = sensitiveData.password || ''
                }
            } catch (error) {
                console.error('获取敏感数据失败:', error)
                ElMessage.warning('获取敏感数据失败，请重新输入')
            }
            machineEditVisible.value = true
        }

        const activate = async () => {
            await loadMachines()
            await loadGlobalAccounts()
            if (props.editMachineId) {
                const target = machines.value.find((m) => m.id === props.editMachineId)
                if (target) await editMachine(target)
            }
        }

        watch(() => props.modelValue, async (v) => {
            visibleProxy.value = v
            if (props.embedded) return
            if (v) await activate()
            else emit('closed')
        })
        watch(visibleProxy, (v) => {
            if (!props.embedded) emit('update:modelValue', v)
        })
        watch(() => props.active, async (v) => {
            if (!props.embedded) return
            if (v) await activate()
            else emit('closed')
        }, { immediate: true })

        const resetMachineForm = () => {
            machineForm.name = ''
            machineForm.group = ''
            machineForm.key_file = ''
            machineForm.host = ''
            machineForm.port = 22
            machineForm.user = ''
            machineForm.password = ''
        }

        const saveMachine = async () => {
            if (!machineFormRef.value) return
            try {
                await machineFormRef.value.validate()
                savingMachine.value = true
                const machineData = {
                    name: machineForm.name,
                    group: machineForm.group,
                    key_file: machineForm.key_file
                }
                const sensitiveData = {
                    host: machineForm.host,
                    port: machineForm.port,
                    user: machineForm.user,
                    password: machineForm.password
                }
                if (editingMachine.value) {
                    machineData.id = editingMachine.value.id
                    await UpdateMachine(editingMachine.value.id, machineData)
                    await SetMachineSensitiveData(editingMachine.value.id, sensitiveData)
                    ElMessage.success('机器配置更新成功')
                } else {
                    await CreateMachine(machineData, sensitiveData)
                    ElMessage.success('机器配置添加成功')
                }
                machineEditVisible.value = false
                await loadMachines()
                emit('changed')
            } catch (error) {
                console.error('保存机器配置失败:', error)
                ElMessage.error('保存机器配置失败: ' + error.message)
            } finally {
                savingMachine.value = false
            }
        }

        const deleteMachine = async (machine) => {
            try {
                await ElMessageBox.confirm(`确定删除机器「${machine.name}」吗？`, '确认删除', { type: 'warning' })
                await DeleteMachine(machine.id)
                ElMessage.success('机器配置删除成功')
                await loadMachines()
                emit('changed')
            } catch (error) {
                if (error === 'cancel') return
                console.error('删除机器配置失败:', error)
                ElMessage.error('删除机器配置失败: ' + error.message)
            }
        }

        const testConnection = async (machine) => {
            try {
                machine.testing = true
                await TestMachineConnection(machine.id)
                ElMessage.success('连接测试成功')
            } catch (error) {
                console.error('连接测试失败:', error)
                ElMessage.error('连接测试失败: ' + error.message)
            } finally {
                machine.testing = false
            }
        }

        const selectKeyFile = async () => {
            try {
                const filePath = await SelectKeyFile()
                if (filePath) {
                    machineForm.key_file = filePath
                }
            } catch (error) {
                console.error('选择密钥文件失败:', error)
                ElMessage.error('选择密钥文件失败: ' + error.message)
            }
        }

        const showImportResult = (result) => {
            const errors = result?.errors?.length ? `\n失败: ${result.errors.join('\n')}` : ''
            ElMessage.success(`导入完成：成功 ${result?.imported || 0}，跳过 ${result?.skipped || 0}${errors}`)
        }

        const ensureImportApi = (fn, label) => {
            if (typeof fn !== 'function') {
                ElMessage.error(`${label} 不可用，请停止后重新运行 wails dev`)
                return false
            }
            return true
        }

        const importXshell = async () => {
            if (!ensureImportApi(ImportXshellPick, 'Xshell 导入')) return
            try {
                const result = await ImportXshellPick(importAccountId.value || '')
                if (!result) return
                showImportResult(result)
                await loadMachines()
                emit('changed')
            } catch (error) {
                ElMessage.error('导入失败: ' + error)
            }
        }

        const importFinalShell = async () => {
            if (!ensureImportApi(ImportFinalShellPick, 'FinalShell 导入')) return
            try {
                const result = await ImportFinalShellPick(importAccountId.value || '')
                if (!result) return
                showImportResult(result)
                await loadMachines()
                emit('changed')
            } catch (error) {
                ElMessage.error('导入失败: ' + error)
            }
        }

        const handleAddCommand = (command) => {
            if (command === 'import-finalshell') importFinalShell()
            else if (command === 'import-xshell') importXshell()
        }

        return {
            embedded: computed(() => props.embedded),
            visibleProxy,
            machines,
            sortedMachines,
            globalAccounts,
            machinesLoading,
            machineEditVisible,
            savingMachine,
            editingMachine,
            machineFormRef,
            machineForm,
            machineRules,
            selectedAccountId,
            handleClose,
            addMachine,
            editMachine,
            saveMachine,
            deleteMachine,
            testConnection,
            selectKeyFile,
            applyGlobalAccount,
            handleAddCommand,
            importAccountId
        }
    }
}
</script>

<style scoped>
.machine-config-container {
    height: min(62vh, 560px);
    display: flex;
    flex-direction: column;
    min-height: 0;
}

.machine-config-container.embedded {
    height: 100%;
    min-height: 360px;
}

.machine-config-container.embedded .machine-table-wrap {
    min-height: 280px;
}

.machine-list {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
}

.list-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
    flex-shrink: 0;
}

.header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
}

.import-account-select {
    width: 220px;
}

.import-account-select :deep(.el-select__wrapper) {
    min-height: 32px;
}

.machine-table-wrap {
    flex: 1;
    min-height: 0;
    max-height: 100%;
    overflow: auto;
    border: 1px solid var(--el-border-color-lighter, var(--app-border));
    border-radius: 6px;
}

.key-file-input {
    display: flex;
    gap: 8px;
    width: 100%;
}
</style>
