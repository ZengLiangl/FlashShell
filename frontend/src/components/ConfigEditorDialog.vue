<template>
    <el-dialog v-model="visibleProxy" title="业务配置编辑" width="90%" top="4vh" :before-close="handleClose">
        <div class="toolbar">
            <el-button type="primary" @click="addProject">新增项目</el-button>
            <el-button @click="load">重新加载</el-button>
        </div>

        <el-collapse v-model="expandedProjects" class="editor-collapse">
            <el-collapse-item v-for="(project, pIndex) in root.projects" :key="pIndex" :name="String(pIndex)">
                <template #title>
                    <span class="item-title">项目: {{ project.name || '(未命名)' }}</span>
                </template>

                <el-form label-width="90px" size="small" class="block-form">
                    <el-form-item label="名称"><el-input v-model="project.name" /></el-form-item>
                    <el-form-item label="描述"><el-input v-model="project.description" /></el-form-item>
                    <el-form-item label="工作目录"><el-input v-model="project.workdir" /></el-form-item>
                    <el-button type="danger" text @click="removeProject(pIndex)">删除项目</el-button>
                </el-form>

                <div v-for="(sub, sIndex) in project.subprojects || []" :key="sIndex" class="sub-block">
                    <div class="sub-header">
                        <strong>子项目: {{ sub.name || '(未命名)' }}</strong>
                        <el-button type="danger" size="small" text @click="removeSubProject(project, sIndex)">删除</el-button>
                    </div>
                    <el-form label-width="90px" size="small">
                        <el-form-item label="名称"><el-input v-model="sub.name" /></el-form-item>
                        <el-form-item label="描述"><el-input v-model="sub.description" /></el-form-item>
                        <el-form-item label="工作目录"><el-input v-model="sub.workdir" /></el-form-item>
                    </el-form>

                    <div v-for="(cmd, cIndex) in sub.commands || []" :key="cIndex" class="cmd-block">
                        <div class="cmd-header">
                            <span>命令: {{ cmd.name || '(未命名)' }}</span>
                            <el-button type="danger" size="small" text @click="removeCommand(sub, cIndex)">删除</el-button>
                        </div>
                        <el-form label-width="90px" size="small">
                            <el-form-item label="名称"><el-input v-model="cmd.name" /></el-form-item>
                            <el-form-item label="类型">
                                <el-select v-model="cmd.type" style="width: 160px">
                                    <el-option label="batch" value="batch" />
                                    <el-option label="remote" value="remote" />
                                </el-select>
                            </el-form-item>
                            <el-form-item v-if="cmd.type === 'remote'" label="机器"><el-input v-model="cmd.machine" /></el-form-item>
                            <el-form-item label="工作目录"><el-input v-model="cmd.workdir" /></el-form-item>
                        </el-form>

                        <div class="steps-block">
                            <div class="steps-header">
                                <span>步骤</span>
                                <el-button size="small" @click="addStep(cmd)">添加步骤</el-button>
                            </div>
                            <div v-for="(step, stIndex) in normalizedSteps(cmd)" :key="stIndex" class="step-row">
                                <el-input v-model="step.command" placeholder="cmd / shell 命令" class="step-cmd" />
                                <el-select v-model="step.onFail" placeholder="失败策略" style="width: 110px">
                                    <el-option label="abort" value="abort" />
                                    <el-option label="continue" value="continue" />
                                </el-select>
                                <el-input-number v-model="step.retry" :min="0" :max="10" controls-position="right" style="width: 100px" />
                                <el-button type="danger" text @click="removeStep(cmd, stIndex)">删</el-button>
                            </div>
                        </div>
                    </div>
                    <el-button size="small" @click="addCommand(sub)">添加命令</el-button>
                </div>
                <el-button size="small" @click="addSubProject(project)">添加子项目</el-button>
            </el-collapse-item>
        </el-collapse>

        <template #footer>
            <el-button @click="handleClose">取消</el-button>
            <el-button type="primary" :loading="saving" @click="save">保存到配置文件</el-button>
        </template>
    </el-dialog>
</template>

<script>
import { ref, watch, computed } from 'vue'
import { ElMessage } from 'element-plus'
import * as App from '../../wailsjs/go/app/App'

function emptyProject() {
    return { name: '', description: '', workdir: '', subprojects: [] }
}
function emptySubProject() {
    return { name: '', description: '', workdir: '', commands: [] }
}
function emptyCommand() {
    return { name: '', description: '', type: 'batch', machine: '', workdir: '', steps: [] }
}
function emptyStep() {
    return { command: '', onFail: 'abort', retry: 0 }
}

export default {
    name: 'ConfigEditorDialog',
    props: { modelValue: { type: Boolean, default: false } },
    emits: ['update:modelValue', 'saved'],
    setup(props, { emit }) {
        const root = ref({ projects: [], machines: [] })
        const saving = ref(false)
        const expandedProjects = ref(['0'])

        const visibleProxy = computed({
            get: () => props.modelValue,
            set: (v) => emit('update:modelValue', v)
        })

        const normalizeSteps = (cmd) => {
            if (!cmd.steps) cmd.steps = []
            cmd.steps = cmd.steps.map((s) => {
                if (typeof s === 'string') {
                    return { command: s, onFail: 'abort', retry: 0 }
                }
                return {
                    command: s.command || '',
                    onFail: s.onFail || 'abort',
                    retry: s.retry || 0
                }
            })
            return cmd.steps
        }

        const load = async () => {
            const config = await App.GetConfigForRefresh()
            root.value = JSON.parse(JSON.stringify(config))
            root.value.projects?.forEach((p) => {
                p.subprojects?.forEach((sp) => {
                    sp.commands?.forEach((cmd) => normalizeSteps(cmd))
                })
            })
        }

        watch(() => props.modelValue, (open) => { if (open) load() })

        const addProject = () => {
            root.value.projects.push(emptyProject())
            expandedProjects.value = [String(root.value.projects.length - 1)]
        }
        const removeProject = (index) => root.value.projects.splice(index, 1)
        const addSubProject = (project) => {
            if (!project.subprojects) project.subprojects = []
            project.subprojects.push(emptySubProject())
        }
        const removeSubProject = (project, index) => project.subprojects.splice(index, 1)
        const addCommand = (sub) => {
            if (!sub.commands) sub.commands = []
            sub.commands.push(emptyCommand())
        }
        const removeCommand = (sub, index) => sub.commands.splice(index, 1)
        const addStep = (cmd) => {
            normalizeSteps(cmd).push(emptyStep())
        }
        const removeStep = (cmd, index) => normalizeSteps(cmd).splice(index, 1)

        const serializeSteps = (steps) => steps.map((s) => {
            const hasExtra = (s.onFail && s.onFail !== 'abort') || (s.retry && s.retry > 0)
            if (!hasExtra) return s.command
            const obj = { cmd: s.command }
            if (s.onFail && s.onFail !== 'abort') obj.on_fail = s.onFail
            if (s.retry > 0) obj.retry = s.retry
            return obj
        })

        const save = async () => {
            saving.value = true
            try {
                const payload = JSON.parse(JSON.stringify(root.value))
                payload.projects?.forEach((p) => {
                    p.subprojects?.forEach((sp) => {
                        sp.commands?.forEach((cmd) => {
                            cmd.steps = serializeSteps(cmd.steps || [])
                        })
                    })
                })
                await App.SaveConfig(payload)
                ElMessage.success('配置已保存')
                emit('saved')
                visibleProxy.value = false
            } catch (e) {
                ElMessage.error(`保存失败: ${e}`)
            } finally {
                saving.value = false
            }
        }

        const handleClose = () => { visibleProxy.value = false }

        return {
            visibleProxy, root, saving, expandedProjects,
            addProject, removeProject, addSubProject, removeSubProject,
            addCommand, removeCommand, addStep, removeStep,
            normalizedSteps: normalizeSteps, load, save, handleClose
        }
    }
}
</script>

<style scoped>
.toolbar { margin-bottom: 12px; }
.editor-collapse { max-height: 65vh; overflow: auto; }
.block-form { margin-bottom: 8px; }
.sub-block, .cmd-block { margin: 12px 0 12px 16px; padding: 12px; border: 1px solid var(--app-border, #e4e7ed); border-radius: 8px; }
.sub-header, .cmd-header, .steps-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.step-row { display: flex; gap: 8px; margin-bottom: 8px; align-items: center; }
.step-cmd { flex: 1; }
.item-title { font-weight: 600; }
</style>
