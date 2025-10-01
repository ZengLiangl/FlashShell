<template>
    <el-dialog v-model="visibleProxy" title="机器配置管理" width="80%" :before-close="handleClose">
        <div class="machine-config-container">
            <div class="machine-list">
                <div class="list-header">
                    <h4>机器列表</h4>
                    <el-button type="primary" @click="addMachine">
                        <el-icon>
                            <Plus />
                        </el-icon>
                        添加机器
                    </el-button>
                </div>

                <el-table :data="machines" style="width: 100%" v-loading="machinesLoading">
                    <el-table-column prop="name" label="机器名称" width="150" />
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

        <el-dialog v-model="machineEditVisible" :title="editingMachine ? '编辑机器' : '添加机器'" width="600px">
            <el-form :model="machineForm" :rules="machineRules" ref="machineFormRef" label-width="100px">
                <el-form-item label="机器名称" prop="name">
                    <el-input v-model="machineForm.name" placeholder="请输入机器名称" />
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
    </el-dialog>
</template>

<script>
import { ref, reactive, watch } from 'vue'
import { ElMessage } from 'element-plus'
import * as App from '../../wailsjs/go/app/App'

export default {
    name: 'MachineConfigDialog',
    props: {
        modelValue: { type: Boolean, required: true }
    },
    emits: ['update:modelValue'],
    setup(props, { emit }) {
        const visibleProxy = ref(props.modelValue)
        watch(() => props.modelValue, v => {
            visibleProxy.value = v
            if (v) loadMachines()
        })
        watch(visibleProxy, v => emit('update:modelValue', v))

        const machines = ref([])
        const machinesLoading = ref(false)
        const machineEditVisible = ref(false)
        const savingMachine = ref(false)
        const editingMachine = ref(null)
        const machineFormRef = ref(null)

        const machineForm = reactive({
            name: '',
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
                const machinesData = await App.GetMachines()
                machines.value = machinesData || []
            } catch (error) {
                console.error('加载机器配置失败:', error)
                ElMessage.error('加载机器配置失败: ' + error.message)
            } finally {
                machinesLoading.value = false
            }
        }

        const addMachine = () => {
            editingMachine.value = null
            resetMachineForm()
            machineEditVisible.value = true
        }

        const editMachine = async (machine) => {
            editingMachine.value = machine
            machineForm.name = machine.name
            machineForm.key_file = machine.key_file || ''
            try {
                const sensitiveData = await App.GetMachineSensitiveData(machine.name)
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

        const resetMachineForm = () => {
            machineForm.name = ''
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
                const machineData = { name: machineForm.name, key_file: machineForm.key_file }
                const sensitiveData = { host: machineForm.host, port: machineForm.port, user: machineForm.user, password: machineForm.password }
                if (editingMachine.value) {
                    await App.UpdateMachine(editingMachine.value.name, machineData)
                    await App.SetMachineSensitiveData(machineForm.name, sensitiveData)
                    ElMessage.success('机器配置更新成功')
                } else {
                    await App.AddMachine(machineData)
                    await App.SetMachineSensitiveData(machineForm.name, sensitiveData)
                    ElMessage.success('机器配置添加成功')
                }
                machineEditVisible.value = false
                await loadMachines()
            } catch (error) {
                console.error('保存机器配置失败:', error)
                ElMessage.error('保存机器配置失败: ' + error.message)
            } finally {
                savingMachine.value = false
            }
        }

        const deleteMachine = async (machine) => {
            try {
                await App.DeleteMachine(machine.name)
                ElMessage.success('机器配置删除成功')
                await loadMachines()
            } catch (error) {
                console.error('删除机器配置失败:', error)
                ElMessage.error('删除机器配置失败: ' + error.message)
            }
        }

        const testConnection = async (machine) => {
            try {
                machine.testing = true
                await App.TestMachineConnection(machine.name)
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
                const filePath = await App.SelectKeyFile()
                if (filePath) {
                    machineForm.key_file = filePath
                }
            } catch (error) {
                console.error('选择密钥文件失败:', error)
                ElMessage.error('选择密钥文件失败: ' + error.message)
            }
        }

        return {
            visibleProxy,
            machines,
            machinesLoading,
            machineEditVisible,
            savingMachine,
            editingMachine,
            machineFormRef,
            machineForm,
            machineRules,
            handleClose,
            addMachine,
            editMachine,
            saveMachine,
            deleteMachine,
            testConnection,
            selectKeyFile
        }
    }
}
</script>

<style scoped>
.machine-config-container {}

.machine-list {}

.list-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
}
</style>
