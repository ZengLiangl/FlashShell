<template>
    <div class="workpath-config-container" :class="{ embedded }">
        <div class="workpath-list">
            <div class="list-header">
                <h4 v-if="!embedded">环境变量列表</h4>
                <div v-else></div>
                <el-tooltip content="添加环境变量" placement="top">
                    <el-button type="primary" circle @click="addWorkPath">
                        <el-icon><Plus /></el-icon>
                    </el-button>
                </el-tooltip>
            </div>
            <div class="workpath-table-wrap">
                <el-table :data="entries" style="width: 100%" v-loading="workPathsLoading">
                    <el-table-column prop="key" label="变量名" width="200" />
                    <el-table-column prop="value" label="变量值" overflow-tooltip />
                    <el-table-column label="操作" width="100" align="center">
                        <template #default="scope">
                            <div class="icon-actions">
                                <el-tooltip content="编辑" placement="top">
                                    <el-button size="small" text type="primary" @click="editWorkPath(scope.row.key)">
                                        <el-icon><Edit /></el-icon>
                                    </el-button>
                                </el-tooltip>
                                <el-tooltip content="删除" placement="top">
                                    <el-button size="small" text type="danger" @click="deleteWorkPath(scope.row.key)">
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
            v-model="workPathEditVisible"
            :title="editingWorkPath ? '编辑环境变量' : '添加环境变量'"
            width="500px"
            append-to-body
        >
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
                <div class="dialog-footer icon-actions">
                    <el-tooltip content="取消" placement="top">
                        <el-button circle @click="workPathEditVisible = false">
                            <el-icon><Close /></el-icon>
                        </el-button>
                    </el-tooltip>
                    <el-tooltip :content="editingWorkPath ? '更新' : '添加'" placement="top">
                        <el-button type="primary" circle :loading="savingWorkPath" @click="saveWorkPath">
                            <el-icon v-if="!savingWorkPath"><Check /></el-icon>
                        </el-button>
                    </el-tooltip>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script>
import { ref, reactive, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, Edit, Delete, Close, Check } from '@element-plus/icons-vue'
import * as App from '../../wailsjs/go/app/App'

export default {
    name: 'WorkPathConfigDialog',
    components: { Plus, Edit, Delete, Close, Check },
    props: {
        modelValue: { type: Boolean, default: false },
        embedded: { type: Boolean, default: false },
        active: { type: Boolean, default: false },
    },
    emits: ['update:modelValue'],
    setup(props, { emit }) {
        const visibleProxy = ref(props.modelValue)
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

        watch(() => props.modelValue, (v) => {
            visibleProxy.value = v
            if (!props.embedded && v) loadWorkPaths()
        })
        watch(visibleProxy, (v) => {
            if (!props.embedded) emit('update:modelValue', v)
        })
        watch(() => props.active, (v) => {
            if (props.embedded && v) loadWorkPaths()
        }, { immediate: true })

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
            embedded: computed(() => props.embedded),
            workPaths,
            workPathsLoading,
            workPathEditVisible,
            savingWorkPath,
            editingWorkPath,
            workPathFormRef,
            workPathForm,
            workPathRules,
            entries,
            addWorkPath,
            editWorkPath,
            saveWorkPath,
            deleteWorkPath
        }
    }
}
</script>

<style scoped>
.workpath-config-container {
    height: min(62vh, 560px);
    display: flex;
    flex-direction: column;
    min-height: 0;
}

.workpath-config-container.embedded {
    height: 100%;
    min-height: 360px;
}

.workpath-list {
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

.workpath-table-wrap {
    flex: 1;
    min-height: 0;
    overflow: auto;
    border: 1px solid var(--el-border-color-lighter, var(--app-border));
    border-radius: 6px;
}

.usage-info {
    font-size: 12px;
    color: var(--app-text-muted);
    line-height: 1.6;
}

.usage-info p {
    margin: 0;
}
</style>
