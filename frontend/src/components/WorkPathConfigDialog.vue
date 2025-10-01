<template>
    <el-dialog v-model="visibleProxy" title="环境变量配置管理" width="80%" :before-close="handleClose">
        <div class="workpath-config-container">
            <div class="workpath-list">
                <div class="list-header">
                    <h4>环境变量列表</h4>
                    <el-button type="primary" @click="addWorkPath">
                        <el-icon>
                            <Plus />
                        </el-icon>
                        添加环境变量
                    </el-button>
                </div>
                <el-table :data="entries" style="width: 100%" v-loading="workPathsLoading">
                    <el-table-column prop="key" label="变量名" width="200" />
                    <el-table-column prop="value" label="变量值" overflow-tooltip />
                    <el-table-column label="操作" width="200">
                        <template #default="scope">
                            <el-button size="small" @click="editWorkPath(scope.row.key)">编辑</el-button>
                            <el-button size="small" type="danger" @click="deleteWorkPath(scope.row.key)">删除</el-button>
                        </template>
                    </el-table-column>
                </el-table>
            </div>
        </div>

        <el-dialog v-model="workPathEditVisible" :title="editingWorkPath ? '编辑环境变量' : '添加环境变量'" width="500px">
            <el-form :model="workPathForm" :rules="workPathRules" ref="workPathFormRef" label-width="100px">
                <el-form-item label="变量名" prop="key">
                    <el-input v-model="workPathForm.key" placeholder="请输入变量名（如：PROJECT_HOME）" />
                </el-form-item>
                <el-form-item label="变量值" prop="value">
                    <el-input v-model="workPathForm.value" placeholder="请输入变量值（如：/home/user/projects）" />
                </el-form-item>
                <el-form-item label="使用说明">
                    <div class="usage-info">
                        <p>• 变量名只能包含大写字母、数字和下划线</p>
                        <p>• 变量名必须以字母或下划线开头</p>
                        <p>• 在配置文件中可以使用 ${变量名} 来引用这些环境变量</p>
                        <p>• 例如：workdir: "${PROJECT_HOME}/my-project"</p>
                    </div>
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="workPathEditVisible = false">取消</el-button>
                    <el-button type="primary" @click="saveWorkPath" :loading="savingWorkPath">
                        {{ editingWorkPath ? '更新' : '添加' }}
                    </el-button>
                </div>
            </template>
        </el-dialog>
    </el-dialog>
</template>

<script>
import { ref, reactive, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import * as App from '../../wailsjs/go/app/App'

export default {
    name: 'WorkPathConfigDialog',
    props: {
        modelValue: { type: Boolean, required: true }
    },
    emits: ['update:modelValue'],
    setup(props, { emit }) {
        const visibleProxy = ref(props.modelValue)
        watch(() => props.modelValue, v => {
            visibleProxy.value = v
            if (v) loadWorkPaths()
        })
        watch(visibleProxy, v => emit('update:modelValue', v))

        const workPaths = ref({})
        const workPathsLoading = ref(false)
        const workPathEditVisible = ref(false)
        const savingWorkPath = ref(false)
        const editingWorkPath = ref(null)
        const workPathFormRef = ref(null)

        const workPathForm = reactive({ key: '', value: '' })
        const workPathRules = {
            key: [{ required: true, message: '请输入变量名', trigger: 'blur' }],
            value: [{ required: true, message: '请输入变量值', trigger: 'blur' }]
        }

        const entries = computed(() => Object.entries(workPaths.value).map(([key, value]) => ({ key, value })))

        const handleClose = () => { visibleProxy.value = false }

        const loadWorkPaths = async () => {
            try {
                workPathsLoading.value = true
                const workPathsData = await App.GetWorkPaths()
                workPaths.value = workPathsData || {}
            } catch (error) {
                console.error('加载环境变量配置失败:', error)
                ElMessage.error('加载环境变量配置失败: ' + error.message)
            } finally {
                workPathsLoading.value = false
            }
        }

        const addWorkPath = () => {
            editingWorkPath.value = null
            resetWorkPathForm()
            workPathEditVisible.value = true
        }

        const editWorkPath = (key) => {
            editingWorkPath.value = key
            workPathForm.key = key
            workPathForm.value = workPaths.value[key] || ''
            workPathEditVisible.value = true
        }

        const resetWorkPathForm = () => {
            workPathForm.key = ''
            workPathForm.value = ''
        }

        const saveWorkPath = async () => {
            if (!workPathFormRef.value) return
            try {
                await workPathFormRef.value.validate()
                savingWorkPath.value = true
                if (editingWorkPath.value) {
                    await App.UpdateWorkPath(workPathForm.key, workPathForm.value)
                    ElMessage.success('环境变量更新成功')
                } else {
                    await App.AddWorkPath(workPathForm.key, workPathForm.value)
                    ElMessage.success('环境变量添加成功')
                }
                workPathEditVisible.value = false
                await loadWorkPaths()
            } catch (error) {
                console.error('保存环境变量失败:', error)
                ElMessage.error('保存环境变量失败: ' + error.message)
            } finally {
                savingWorkPath.value = false
            }
        }

        const deleteWorkPath = async (key) => {
            try {
                await App.DeleteWorkPath(key)
                ElMessage.success('环境变量删除成功')
                await loadWorkPaths()
            } catch (error) {
                console.error('删除环境变量失败:', error)
                ElMessage.error('删除环境变量失败: ' + error.message)
            }
        }

        return {
            visibleProxy,
            workPaths,
            workPathsLoading,
            workPathEditVisible,
            savingWorkPath,
            editingWorkPath,
            workPathFormRef,
            workPathForm,
            workPathRules,
            entries,
            handleClose,
            addWorkPath,
            editWorkPath,
            saveWorkPath,
            deleteWorkPath
        }
    }
}
</script>

<style scoped>
.workpath-config-container {}

.workpath-list {}

.list-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
}
</style>
