<template>
  <el-dialog
    v-model="visibleProxy"
    title="任务流水线"
    width="92%"
    top="3vh"
    class="task-config-dialog"
    append-to-body
    destroy-on-close
    :close-on-press-escape="false"
    :before-close="handleClose"
  >
    <div class="task-config-shell">
      <div class="shell-toolbar">
        <el-radio-group v-model="mainTab" size="small">
          <el-radio-button label="flow">流水线</el-radio-button>
          <el-radio-button label="basic">基本信息</el-radio-button>
        </el-radio-group>
        <div class="toolbar-spacer" />
        <div class="icon-actions">
          <el-tooltip content="重新加载" placement="top">
            <el-button size="small" circle @click="reload">
              <el-icon><Refresh /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip content="保存到配置文件" placement="top">
            <el-button type="primary" size="small" circle :loading="saving" @click="save">
              <el-icon v-if="!saving"><Check /></el-icon>
            </el-button>
          </el-tooltip>
        </div>
      </div>

      <div class="shell-body">
        <TaskFlowNav
          :projects="root.projects"
          :selected-p="navP"
          :selected-s="navS"
          @add-project="addProject"
          @remove-project="removeProject"
          @add-sub="addSubProject"
          @remove-sub="removeSubProject"
          @select-project="selectProject"
          @select-sub="selectSub"
        />

        <div class="shell-main">
          <template v-if="mainTab === 'basic'">
            <div class="basic-pane">
              <template v-if="navP != null && root.projects[navP]">
                <h4 class="basic-title">项目信息</h4>
                <el-form label-width="90px" size="small" class="basic-form">
                  <el-form-item label="名称">
                    <el-input v-model="root.projects[navP].name" />
                  </el-form-item>
                  <el-form-item label="描述">
                    <el-input v-model="root.projects[navP].description" type="textarea" :rows="2" />
                  </el-form-item>
                  <el-form-item label="工作目录">
                    <el-input v-model="root.projects[navP].workdir" placeholder="可选" />
                  </el-form-item>
                </el-form>

                <template v-if="navS != null && root.projects[navP].subprojects?.[navS]">
                  <h4 class="basic-title">子项目信息</h4>
                  <el-form label-width="90px" size="small" class="basic-form">
                    <el-form-item label="名称">
                      <el-input v-model="root.projects[navP].subprojects[navS].name" />
                    </el-form-item>
                    <el-form-item label="描述">
                      <el-input
                        v-model="root.projects[navP].subprojects[navS].description"
                        type="textarea"
                        :rows="2"
                      />
                    </el-form-item>
                    <el-form-item label="工作目录">
                      <el-input
                        v-model="root.projects[navP].subprojects[navS].workdir"
                        placeholder="可选"
                      />
                    </el-form-item>
                  </el-form>
                </template>
                <el-alert
                  v-else
                  type="info"
                  :closable="false"
                  show-icon
                  title="在左侧展开并选择子项目可编辑其子项目基本信息"
                />
              </template>
              <el-empty v-else description="请先在左侧选择项目" :image-size="64" />
            </div>
          </template>

          <template v-else>
            <TaskFlowCanvas
              :sub-project="activeSub"
              :selected-path="flowPath"
              @select-command="selectCommand"
              @select-step="selectStep"
              @add-command="addCommand"
              @insert-command="insertCommand"
              @remove-command="removeCommand"
              @add-step="addStep"
              @remove-step="removeStep"
            />
            <TaskFlowDrawer
              :selection-kind="drawerKind"
              :draft="editDraft"
              :machines="machineOptions"
              @update:draft="onDraftUpdate"
              @close="closeDrawer"
              @delete="deleteSelected"
            />
          </template>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer icon-actions">
        <el-tooltip content="取消" placement="top">
          <el-button circle @click="handleClose">
            <el-icon><Close /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip content="保存到配置文件" placement="top">
          <el-button type="primary" circle :loading="saving" @click="save">
            <el-icon v-if="!saving"><Check /></el-icon>
          </el-button>
        </el-tooltip>
      </div>
    </template>
  </el-dialog>
</template>

<script>
import { computed, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check, Close, Refresh } from '@element-plus/icons-vue'
import * as App from '../../wailsjs/go/app/App'
import TaskFlowNav from './taskflow/TaskFlowNav.vue'
import TaskFlowCanvas from './taskflow/TaskFlowCanvas.vue'
import TaskFlowDrawer from './taskflow/TaskFlowDrawer.vue'
import {
  cloneNodeByPath,
  commitByPath,
  emptyCommand,
  emptyProject,
  emptyStep,
  emptySubProject,
  getSubProject,
  normalizeRoot,
  samePath,
  serializeRootForSave,
} from './taskflow/taskFlowModel'

export default {
  name: 'ConfigEditorDialog',
  components: { TaskFlowNav, TaskFlowCanvas, TaskFlowDrawer, Check, Close, Refresh },
  props: { modelValue: { type: Boolean, default: false } },
  emits: ['update:modelValue', 'saved'],
  setup(props, { emit }) {
    const root = ref({ projects: [], machines: [] })
    const machineOptions = ref([])
    const saving = ref(false)
    const mainTab = ref('flow')
    const navP = ref(null)
    const navS = ref(null)
    const flowSel = ref(null)
    const editDraft = ref(null)

    const visibleProxy = computed({
      get: () => props.modelValue,
      set: (v) => emit('update:modelValue', v),
    })

    const activeSub = computed(() => {
      if (navP.value == null || navS.value == null) return null
      return getSubProject(root.value, navP.value, navS.value)
    })

    const flowPath = computed(() => {
      if (navP.value == null || navS.value == null || !flowSel.value) return null
      return {
        p: navP.value,
        s: navS.value,
        c: flowSel.value.c,
        st: flowSel.value.st,
      }
    })

    const drawerKind = computed(() => {
      if (!flowPath.value) return ''
      if (flowPath.value.st != null) return 'step'
      if (flowPath.value.c != null) return 'command'
      return ''
    })

    const commitDraft = () => {
      if (!flowPath.value || !editDraft.value) return
      commitByPath(root.value, flowPath.value, editDraft.value)
    }

    const openPath = (path) => {
      commitDraft()
      if (!path) {
        flowSel.value = null
        editDraft.value = null
        return
      }
      flowSel.value = { c: path.c, st: path.st }
      const full = { p: navP.value, s: navS.value, c: path.c, st: path.st }
      editDraft.value = cloneNodeByPath(root.value, full)
    }

    const onDraftUpdate = (draft) => {
      editDraft.value = draft
      if (flowPath.value) {
        commitByPath(root.value, flowPath.value, draft)
      }
    }

    const closeDrawer = () => {
      commitDraft()
      flowSel.value = null
      editDraft.value = null
    }

    const selectProject = (pIndex) => {
      commitDraft()
      flowSel.value = null
      editDraft.value = null
      navP.value = pIndex
      navS.value = null
      mainTab.value = 'flow'
    }

    const selectSub = (pIndex, sIndex) => {
      commitDraft()
      flowSel.value = null
      editDraft.value = null
      navP.value = pIndex
      navS.value = sIndex
      mainTab.value = 'flow'
    }

    const selectCommand = (cIndex) => {
      const next = { c: cIndex, st: undefined }
      if (samePath(
        { p: navP.value, s: navS.value, ...next },
        flowPath.value,
      )) {
        return
      }
      openPath(next)
    }

    const selectStep = (cIndex, stIndex) => {
      openPath({ c: cIndex, st: stIndex })
    }

    const ensureSubCommands = () => {
      const sub = activeSub.value
      if (!sub) return null
      if (!Array.isArray(sub.commands)) sub.commands = []
      return sub
    }

    const addCommand = (template = 'batch') => {
      const sub = ensureSubCommands()
      if (!sub) return
      commitDraft()
      sub.commands.push(emptyCommand(template))
      openPath({ c: sub.commands.length - 1 })
    }

    const insertCommand = (atIndex) => {
      const sub = ensureSubCommands()
      if (!sub) return
      commitDraft()
      sub.commands.splice(atIndex, 0, emptyCommand('batch'))
      openPath({ c: atIndex })
    }

    const removeCommand = async (cIndex) => {
      const sub = ensureSubCommands()
      if (!sub) return
      try {
        await ElMessageBox.confirm('确定删除该命令及其全部步骤？', '删除命令', {
          type: 'warning',
          confirmButtonText: '删除',
          cancelButtonText: '取消',
        })
      } catch {
        return
      }
      const prevSel = flowSel.value ? { ...flowSel.value } : null
      if (prevSel?.c === cIndex) {
        flowSel.value = null
        editDraft.value = null
      }
      sub.commands.splice(cIndex, 1)
      if (prevSel != null && prevSel.c > cIndex) {
        flowSel.value = {
          c: prevSel.c - 1,
          st: prevSel.st,
        }
        editDraft.value = cloneNodeByPath(root.value, {
          p: navP.value,
          s: navS.value,
          ...flowSel.value,
        })
      }
    }

    const addStep = (cIndex) => {
      const sub = ensureSubCommands()
      const cmd = sub?.commands?.[cIndex]
      if (!cmd) return
      if (!Array.isArray(cmd.steps)) cmd.steps = []
      commitDraft()
      cmd.steps.push(emptyStep('shell'))
      openPath({ c: cIndex, st: cmd.steps.length - 1 })
    }

    const removeStep = (cIndex, stIndex) => {
      const cmd = ensureSubCommands()?.commands?.[cIndex]
      if (!cmd?.steps) return
      const prevSel = flowSel.value ? { ...flowSel.value } : null
      if (prevSel?.c === cIndex && prevSel?.st === stIndex) {
        flowSel.value = { c: cIndex }
        editDraft.value = null
      }
      cmd.steps.splice(stIndex, 1)
      if (prevSel?.c === cIndex && prevSel?.st === stIndex) {
        editDraft.value = cloneNodeByPath(root.value, {
          p: navP.value,
          s: navS.value,
          c: cIndex,
        })
      } else if (prevSel?.c === cIndex && prevSel?.st != null && prevSel.st > stIndex) {
        flowSel.value = { c: cIndex, st: prevSel.st - 1 }
        editDraft.value = cloneNodeByPath(root.value, {
          p: navP.value,
          s: navS.value,
          ...flowSel.value,
        })
      }
    }

    const deleteSelected = async () => {
      if (!flowSel.value) return
      if (flowSel.value.st != null) {
        removeStep(flowSel.value.c, flowSel.value.st)
        return
      }
      await removeCommand(flowSel.value.c)
    }

    const addProject = () => {
      commitDraft()
      if (!root.value.projects) root.value.projects = []
      root.value.projects.push(emptyProject())
      const p = root.value.projects.length - 1
      navP.value = p
      navS.value = null
      flowSel.value = null
      editDraft.value = null
      mainTab.value = 'basic'
    }

    const removeProject = async (pIndex) => {
      try {
        await ElMessageBox.confirm('确定删除该项目及其全部子项目？', '删除项目', {
          type: 'warning',
          confirmButtonText: '删除',
          cancelButtonText: '取消',
        })
      } catch {
        return
      }
      commitDraft()
      root.value.projects.splice(pIndex, 1)
      if (navP.value === pIndex) {
        navP.value = null
        navS.value = null
        flowSel.value = null
        editDraft.value = null
      } else if (navP.value > pIndex) {
        navP.value -= 1
      }
    }

    const addSubProject = (pIndex) => {
      const project = root.value.projects[pIndex]
      if (!project) return
      if (!Array.isArray(project.subprojects)) project.subprojects = []
      commitDraft()
      project.subprojects.push(emptySubProject())
      navP.value = pIndex
      navS.value = project.subprojects.length - 1
      flowSel.value = null
      editDraft.value = null
      mainTab.value = 'basic'
    }

    const removeSubProject = async (pIndex, sIndex) => {
      try {
        await ElMessageBox.confirm('确定删除该子项目及其流水线？', '删除子项目', {
          type: 'warning',
          confirmButtonText: '删除',
          cancelButtonText: '取消',
        })
      } catch {
        return
      }
      const project = root.value.projects[pIndex]
      if (!project?.subprojects) return
      commitDraft()
      project.subprojects.splice(sIndex, 1)
      if (navP.value === pIndex && navS.value === sIndex) {
        navS.value = null
        flowSel.value = null
        editDraft.value = null
      } else if (navP.value === pIndex && navS.value > sIndex) {
        navS.value -= 1
      }
    }

    const loadMachines = async () => {
      try {
        const list = await App.GetMachines()
        machineOptions.value = Array.isArray(list) ? list : []
      } catch {
        machineOptions.value = []
      }
    }

    const load = async () => {
      const [config] = await Promise.all([
        App.GetConfigForRefresh(),
        loadMachines(),
      ])
      root.value = normalizeRoot(config)
      flowSel.value = null
      editDraft.value = null
      navP.value = null
      navS.value = null
      mainTab.value = 'flow'
    }

    const reload = async () => {
      try {
        await ElMessageBox.confirm('重新加载将丢弃未保存的修改，是否继续？', '重新加载', {
          type: 'warning',
          confirmButtonText: '重新加载',
          cancelButtonText: '取消',
        })
      } catch {
        return
      }
      await load()
      ElMessage.success('已重新加载')
    }

    const save = async () => {
      commitDraft()
      saving.value = true
      try {
        const payload = serializeRootForSave(root.value)
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

    const handleClose = () => {
      visibleProxy.value = false
    }

    watch(
      () => props.modelValue,
      (open) => {
        if (open) load()
      },
    )

    return {
      visibleProxy,
      root,
      machineOptions,
      saving,
      mainTab,
      navP,
      navS,
      activeSub,
      flowPath,
      drawerKind,
      editDraft,
      selectProject,
      selectSub,
      selectCommand,
      selectStep,
      addCommand,
      insertCommand,
      removeCommand,
      addStep,
      removeStep,
      deleteSelected,
      addProject,
      removeProject,
      addSubProject,
      removeSubProject,
      onDraftUpdate,
      closeDrawer,
      reload,
      save,
      handleClose,
    }
  },
}
</script>

<style scoped>
.task-config-shell {
  display: flex;
  flex-direction: column;
  height: min(78vh, 720px);
  min-height: 480px;
  margin: -4px -12px 0;
  border-top: 1px solid var(--app-border);
}

.shell-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--app-border);
  background: var(--app-bg);
  flex-shrink: 0;
}

.toolbar-spacer {
  flex: 1;
}

.shell-body {
  flex: 1;
  min-height: 0;
  display: flex;
}

.shell-main {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
}

.basic-pane {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 20px 24px;
  background: var(--app-panel-bg);
}

.basic-title {
  margin: 0 0 12px;
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text);
}

.basic-title + .basic-form {
  margin-bottom: 24px;
  max-width: 520px;
  padding: 16px;
  background: var(--app-card-bg);
  border: 1px solid var(--app-card-border);
  border-radius: var(--app-radius-md, 8px);
}

.dialog-footer {
  justify-content: flex-end;
}
</style>

<style>
.task-config-dialog.el-dialog {
  overflow: hidden;
  display: flex;
  flex-direction: column;
  max-height: 92vh;
}

.task-config-dialog .el-dialog__header {
  flex-shrink: 0;
}

.task-config-dialog .el-dialog__body {
  padding-top: 4px;
  padding-bottom: 8px;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.task-config-dialog .el-dialog__footer {
  flex-shrink: 0;
  border-top: 1px solid var(--app-border);
}
</style>
