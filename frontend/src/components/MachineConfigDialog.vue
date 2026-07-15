<template>
    <div class="machine-config-container" :class="{ embedded }">
        <div class="machine-list">
            <div class="list-header">
                <h4 v-if="!embedded">机器列表</h4>
                <div v-else class="list-header-spacer"></div>
                <div class="header-actions">
                    <div class="filter-bar">
                        <el-input
                            v-model="machineKeyword"
                            clearable
                            size="small"
                            placeholder="搜索名称 / IP"
                            class="list-search"
                        >
                            <template #prefix>
                                <el-icon class="filter-icon"><Search /></el-icon>
                            </template>
                        </el-input>
                        <el-select
                            v-model="importGroup"
                            clearable
                            filterable
                            allow-create
                            default-first-option
                            size="small"
                            placeholder="导入分组"
                            class="import-group-select"
                        >
                            <el-option
                                v-for="g in groupOptions"
                                :key="g"
                                :label="g"
                                :value="g === DEFAULT_MACHINE_GROUP ? '' : g"
                            />
                        </el-select>
                        <el-select
                            v-model="importAccountId"
                            clearable
                            size="small"
                            placeholder="导入帐号"
                            class="import-account-select"
                        >
                            <el-option
                                v-for="account in globalAccounts"
                                :key="account.id"
                                :label="account.name"
                                :value="account.id"
                            />
                        </el-select>
                    </div>
                    <div class="toolbar-ops icon-actions">
                        <el-tooltip content="分组管理" placement="top">
                            <el-button size="small" circle @click="groupManageVisible = true">
                                <el-icon><FolderOpened /></el-icon>
                            </el-button>
                        </el-tooltip>
                        <el-tooltip content="添加机器" placement="top">
                            <el-button size="small" type="primary" circle @click="addMachine">
                                <el-icon><Plus /></el-icon>
                            </el-button>
                        </el-tooltip>
                        <el-dropdown trigger="click" @command="handleAddCommand">
                            <el-button size="small" circle title="导入机器">
                                <el-icon><Upload /></el-icon>
                            </el-button>
                            <template #dropdown>
                                <el-dropdown-menu>
                                    <el-dropdown-item command="import-xshell">导入 Xshell</el-dropdown-item>
                                    <el-dropdown-item command="import-finalshell">导入 FinalShell</el-dropdown-item>
                                </el-dropdown-menu>
                            </template>
                        </el-dropdown>
                    </div>
                </div>
            </div>

            <div class="machine-table-wrap">
                <el-table
                    :data="filteredMachines"
                    row-key="id"
                    style="width: 100%"
                    v-loading="machinesLoading"
                >
                    <el-table-column prop="name" label="机器名称" width="150" />
                    <el-table-column prop="host" label="IP" min-width="140" show-overflow-tooltip>
                        <template #default="scope">
                            {{ scope.row.host || '-' }}
                        </template>
                    </el-table-column>
                    <el-table-column prop="group" label="分组" width="120">
                        <template #default="scope">
                            {{ scope.row.group || DEFAULT_MACHINE_GROUP }}
                        </template>
                    </el-table-column>
                    <el-table-column prop="key_file" label="密钥文件" overflow-tooltip />
                    <el-table-column label="操作" width="130" align="center">
                        <template #default="scope">
                            <div class="icon-actions">
                                <el-tooltip content="编辑" placement="top">
                                    <el-button size="small" text type="primary" @click="editMachine(scope.row)">
                                        <el-icon><Edit /></el-icon>
                                    </el-button>
                                </el-tooltip>
                                <el-tooltip content="测试连接" placement="top">
                                    <el-button size="small" text type="success" :loading="scope.row.testing" @click="testConnection(scope.row)">
                                        <el-icon><Connection /></el-icon>
                                    </el-button>
                                </el-tooltip>
                                <el-tooltip content="删除" placement="top">
                                    <el-button size="small" text type="danger" @click="deleteMachine(scope.row)">
                                        <el-icon><Delete /></el-icon>
                                    </el-button>
                                </el-tooltip>
                            </div>
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
                    <el-select
                        v-model="machineForm.group"
                        clearable
                        filterable
                        allow-create
                        default-first-option
                        placeholder="选择或输入分组，留空为默认分组"
                        style="width: 100%"
                    >
                        <el-option
                            v-for="g in groupOptions"
                            :key="g"
                            :label="g"
                            :value="g === DEFAULT_MACHINE_GROUP ? '' : g"
                        />
                    </el-select>
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
                        <el-tooltip content="选择文件" placement="top">
                            <el-button type="primary" circle @click="selectKeyFile">
                                <el-icon><Folder /></el-icon>
                            </el-button>
                        </el-tooltip>
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
                    <el-tooltip content="测试连接" placement="top">
                        <el-button :loading="testingDraft" circle @click="testDraftConnection">
                            <el-icon><Connection /></el-icon>
                        </el-button>
                    </el-tooltip>
                    <el-button type="primary" @click="saveMachine" :loading="savingMachine">
                        {{ editingMachine ? '更新' : '添加' }}
                    </el-button>
                </div>
            </template>
        </el-dialog>

        <el-dialog v-model="groupManageVisible" title="分组管理" width="480px" append-to-body @open="loadGroups">
            <div class="group-add-row">
                <el-input v-model="newGroupName" placeholder="新分组名称" clearable @keydown.enter.exact.prevent="addGroup" />
                <el-tooltip content="添加分组" placement="top">
                    <el-button type="primary" circle @click="addGroup">
                        <el-icon><Plus /></el-icon>
                    </el-button>
                </el-tooltip>
            </div>
            <el-table :data="managedGroups" size="small" empty-text="暂无自定义分组">
                <el-table-column prop="name" label="分组名称" />
                <el-table-column label="操作" width="100" align="center">
                    <template #default="{ row }">
                        <div class="icon-actions">
                            <el-tooltip content="重命名" placement="top">
                                <el-button size="small" text type="primary" @click="renameGroup(row.name)">
                                    <el-icon><Edit /></el-icon>
                                </el-button>
                            </el-tooltip>
                            <el-tooltip content="删除" placement="top">
                                <el-button size="small" text type="danger" @click="deleteGroup(row.name)">
                                    <el-icon><Delete /></el-icon>
                                </el-button>
                            </el-tooltip>
                        </div>
                    </template>
                </el-table-column>
            </el-table>
            <template #footer>
                <el-button @click="groupManageVisible = false">关闭</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script>
import { ref, reactive, watch, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
    Plus, Search, FolderOpened, Upload, Edit, Delete, Connection, Folder,
} from '@element-plus/icons-vue'
import {
    GetMachines,
    GetMachineGroups,
    AddMachineGroup,
    RenameMachineGroup,
    DeleteMachineGroup,
    GetGlobalAccounts,
    GetMachineSensitiveData,
    UpdateMachine,
    SetMachineSensitiveData,
    CreateMachine,
    DeleteMachine,
    TestMachineConnection,
    TestMachineDraftConnection,
    SelectKeyFile,
    ImportXshellPick,
    ImportFinalShellPick,
} from '../../wailsjs/go/app/App'
import { DEFAULT_MACHINE_GROUP, sortMachinesByName, machineMatchesKeyword } from '../utils/machineGroups'

export default {
    name: 'MachineConfigDialog',
    components: {
        Plus, Search, FolderOpened, Upload, Edit, Delete, Connection, Folder,
    },
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
        const machineGroups = ref([])
        const machineKeyword = ref('')
        const sortedMachines = computed(() => sortMachinesByName(machines.value))
        const filteredMachines = computed(() => {
            const kw = machineKeyword.value
            const list = sortedMachines.value
            if (!String(kw || '').trim()) return list
            return list.filter((m) => machineMatchesKeyword(m, kw))
        })
        const groupOptions = computed(() => {
            const set = new Set([DEFAULT_MACHINE_GROUP])
            for (const g of machineGroups.value || []) {
                if (g) set.add(g)
            }
            for (const m of machines.value || []) {
                if (m.group) set.add(m.group)
            }
            return Array.from(set).sort((a, b) => {
                if (a === DEFAULT_MACHINE_GROUP) return -1
                if (b === DEFAULT_MACHINE_GROUP) return 1
                return a.localeCompare(b, 'zh-CN')
            })
        })
        const managedGroups = computed(() =>
            (machineGroups.value || [])
                .filter((g) => g && g !== DEFAULT_MACHINE_GROUP)
                .map((name) => ({ name })),
        )
        const globalAccounts = ref([])
        const machinesLoading = ref(false)
        const machineEditVisible = ref(false)
        const groupManageVisible = ref(false)
        const newGroupName = ref('')
        const savingMachine = ref(false)
        const testingDraft = ref(false)
        const editingMachine = ref(null)
        const machineFormRef = ref(null)
        const selectedAccountId = ref('')
        const importAccountId = ref('')
        const importGroup = ref('')

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

        const loadGroups = async () => {
            try {
                machineGroups.value = await GetMachineGroups() || []
            } catch {
                machineGroups.value = []
            }
        }

        const loadMachines = async () => {
            try {
                machinesLoading.value = true
                const machinesData = await GetMachines()
                machines.value = machinesData || []
                await loadGroups()
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

        const normalizeGroup = (g) => {
            const s = String(g || '').trim()
            if (!s || s === DEFAULT_MACHINE_GROUP) return ''
            return s
        }

        const saveMachine = async () => {
            if (!machineFormRef.value) return
            try {
                await machineFormRef.value.validate()
                savingMachine.value = true
                const machineData = {
                    name: machineForm.name,
                    group: normalizeGroup(machineForm.group),
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

        const testDraftConnection = async () => {
            if (!machineFormRef.value) return
            try {
                await machineFormRef.value.validate()
                testingDraft.value = true
                await TestMachineDraftConnection(
                    {
                        name: machineForm.name || 'draft-test',
                        group: normalizeGroup(machineForm.group),
                        key_file: machineForm.key_file,
                    },
                    {
                        host: machineForm.host,
                        port: machineForm.port,
                        user: machineForm.user,
                        password: machineForm.password,
                    }
                )
                ElMessage.success('连接测试成功')
            } catch (error) {
                if (error === false || error?.fields) return
                console.error('连接测试失败:', error)
                ElMessage.error('连接测试失败: ' + (error.message || error))
            } finally {
                testingDraft.value = false
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
                const result = await ImportXshellPick(importAccountId.value || '', normalizeGroup(importGroup.value))
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
                const result = await ImportFinalShellPick(importAccountId.value || '', normalizeGroup(importGroup.value))
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

        const addGroup = async () => {
            const name = newGroupName.value.trim()
            if (!name) {
                ElMessage.warning('请输入分组名称')
                return
            }
            try {
                await AddMachineGroup(name)
                newGroupName.value = ''
                ElMessage.success('分组已添加')
                await loadGroups()
                emit('changed')
            } catch (e) {
                ElMessage.error('添加分组失败: ' + e)
            }
        }

        const renameGroup = async (oldName) => {
            try {
                const { value } = await ElMessageBox.prompt('请输入新的分组名称', '重命名分组', {
                    inputValue: oldName,
                    confirmButtonText: '确定',
                    cancelButtonText: '取消',
                    inputValidator: (v) => {
                        const s = String(v || '').trim()
                        if (!s) return '名称不能为空'
                        if (s === DEFAULT_MACHINE_GROUP) return '不能使用默认分组名称'
                        return true
                    },
                })
                await RenameMachineGroup(oldName, String(value).trim())
                ElMessage.success('分组已重命名')
                await loadMachines()
                emit('changed')
            } catch (e) {
                if (e === 'cancel') return
                ElMessage.error('重命名失败: ' + e)
            }
        }

        const deleteGroup = async (name) => {
            try {
                await ElMessageBox.confirm(
                    `删除分组「${name}」后，该分组下的机器将归入「${DEFAULT_MACHINE_GROUP}」。确定删除？`,
                    '删除分组',
                    { type: 'warning' },
                )
                await DeleteMachineGroup(name)
                ElMessage.success('分组已删除')
                await loadMachines()
                emit('changed')
            } catch (e) {
                if (e === 'cancel') return
                ElMessage.error('删除失败: ' + e)
            }
        }

        return {
            embedded: computed(() => props.embedded),
            DEFAULT_MACHINE_GROUP,
            visibleProxy,
            machines,
            sortedMachines,
            filteredMachines,
            groupOptions,
            managedGroups,
            machineKeyword,
            globalAccounts,
            machinesLoading,
            machineEditVisible,
            groupManageVisible,
            newGroupName,
            savingMachine,
            testingDraft,
            editingMachine,
            machineFormRef,
            machineForm,
            machineRules,
            selectedAccountId,
            importAccountId,
            importGroup,
            handleClose,
            addMachine,
            editMachine,
            saveMachine,
            deleteMachine,
            testConnection,
            testDraftConnection,
            selectKeyFile,
            applyGlobalAccount,
            handleAddCommand,
            loadGroups,
            addGroup,
            renameGroup,
            deleteGroup,
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
    gap: 12px;
    margin-bottom: 14px;
    flex-shrink: 0;
}

.list-header-spacer {
    flex: 0;
}

.header-actions {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
    justify-content: flex-end;
    min-width: 0;
}

.filter-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
    padding: 6px 10px;
    border-radius: 10px;
    background: color-mix(in srgb, var(--el-fill-color-light, #f5f7fa) 88%, transparent);
    border: 1px solid color-mix(in srgb, var(--el-border-color-lighter, #ebeef5) 80%, transparent);
}

.filter-icon {
    color: var(--el-text-color-secondary);
}

.list-search {
    width: 168px;
}

.import-group-select {
    width: 128px;
}

.import-account-select {
    width: 140px;
}

.filter-bar :deep(.el-input__wrapper),
.filter-bar :deep(.el-select__wrapper) {
    box-shadow: none !important;
    background: transparent;
    border-radius: 6px;
}

.filter-bar :deep(.el-input__wrapper:hover),
.filter-bar :deep(.el-select__wrapper:hover),
.filter-bar :deep(.el-input__wrapper.is-focus),
.filter-bar :deep(.el-select__wrapper.is-focused) {
    background: var(--el-bg-color, #fff);
    box-shadow: 0 0 0 1px var(--el-border-color, #dcdfe6) inset !important;
}

.toolbar-ops {
    flex-shrink: 0;
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

.group-add-row {
    display: flex;
    gap: 8px;
    margin-bottom: 12px;
}
</style>
