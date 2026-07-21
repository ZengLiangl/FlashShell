<template>
  <el-dialog
    v-model="visibleProxy"
    title="任务流水线"
    width="1280px"
    top="4vh"
    class="task-config-dialog"
    append-to-body
    destroy-on-close
    :close-on-press-escape="true"
    :before-close="handleClose"
  >
    <div class="task-config-shell">
      <div class="shell-toolbar">
        <span class="toolbar-title">编排</span>
        <div v-if="contextLabel" class="shell-context" :title="contextLabel">
          <span class="context-label">{{ contextLabel }}</span>
        </div>
        <div v-else class="toolbar-spacer" />
        <div class="icon-actions">
          <el-tooltip content="重新加载" placement="top">
            <el-button size="small" circle @click="reload">
              <el-icon><Refresh /></el-icon>
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
          <div class="flow-workspace">
            <TaskFlowCanvas
              :sub-project="activeSub"
              :selected-path="flowPath"
              :empty-description="canvasEmptyHint"
              @select-command="selectCommand"
              @select-step="selectStep"
              @add-command="addCommand"
              @insert-command="insertCommand"
              @remove-command="removeCommand"
              @add-step="addStep"
              @remove-step="removeStep"
            />
            <TaskFlowDrawer
              v-if="drawerKind"
              :selection-kind="drawerKind"
              :draft="editDraft"
              :machines="machineOptions"
              @update:draft="onDraftUpdate"
              @close="closeDrawer"
              @delete="deleteSelected"
            />
          </div>
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
    const navP = ref(null)
    const navS = ref(null)
    const flowSel = ref(null)
    const editDraft = ref(null)
    /** 打开/保存/重新加载后的配置快照，用于判断是否有未保存修改 */
    const baselineJson = ref('')

    const snapshotForCompare = () => JSON.stringify(serializeRootForSave(root.value))

    const markClean = () => {
      baselineJson.value = snapshotForCompare()
    }

    const hasUnsavedChanges = () => {
      commitDraft()
      return snapshotForCompare() !== baselineJson.value
    }

    const visibleProxy = computed({
      get: () => props.modelValue,
      set: (v) => emit('update:modelValue', v),
    })

    const activeSub = computed(() => {
      if (navP.value == null || navS.value == null) return null
      return getSubProject(root.value, navP.value, navS.value)
    })

    const contextLabel = computed(() => {
      if (navP.value == null || !root.value.projects?.[navP.value]) return ''
      const project = root.value.projects[navP.value]
      const projectName = project.name || '(未命名项目)'
      if (navS.value == null || !project.subprojects?.[navS.value]) return projectName
      const subName = project.subprojects[navS.value].name || '(未命名子项目)'
      return `${projectName} / ${subName}`
    })

    const canvasEmptyHint = computed(() => {
      if (navP.value == null) return '请在左侧选择或新建项目'
      if (navS.value == null) return '请展开项目并选择子项目，下方可编辑项目属性'
      return '请在左侧选择子项目以编辑流水线'
    })

    /** 画布高亮：仅命令 / 步骤 */
    const flowPath = computed(() => {
      if (navP.value == null || navS.value == null || !flowSel.value) return null
      return {
        p: navP.value,
        s: navS.value,
        c: flowSel.value.c,
        st: flowSel.value.st,
      }
    })

    /** 属性面板路径：命令/步骤优先，否则子项目/项目 */
    const editPath = computed(() => {
      if (navP.value == null) return null
      if (flowSel.value && navS.value != null) {
        return {
          p: navP.value,
          s: navS.value,
          c: flowSel.value.c,
          st: flowSel.value.st,
        }
      }
      if (navS.value != null) return { p: navP.value, s: navS.value }
      return { p: navP.value }
    })

    const drawerKind = computed(() => {
      const path = editPath.value
      if (!path) return ''
      if (path.st != null) return 'step'
      if (path.c != null) return 'command'
      if (path.s != null) return 'sub'
      return 'project'
    })

    const commitDraft = () => {
      if (!editPath.value || !editDraft.value) return
      commitByPath(root.value, editPath.value, editDraft.value)
    }

    const openEditPath = (path) => {
      commitDraft()
      if (!path || path.p == null) {
        flowSel.value = null
        editDraft.value = null
        return
      }
      if (path.c != null) {
        flowSel.value = { c: path.c, st: path.st }
      } else {
        flowSel.value = null
      }
      editDraft.value = cloneNodeByPath(root.value, path)
    }

    const onDraftUpdate = (draft) => {
      editDraft.value = draft
      if (editPath.value) {
        commitByPath(root.value, editPath.value, draft)
      }
    }

    /** 关闭命令/步骤编辑，回到子项目或项目属性 */
    const closeDrawer = () => {
      if (!flowSel.value) return
      commitDraft()
      flowSel.value = null
      if (navP.value == null) {
        editDraft.value = null
        return
      }
      if (navS.value != null) {
        editDraft.value = cloneNodeByPath(root.value, { p: navP.value, s: navS.value })
      } else {
        editDraft.value = cloneNodeByPath(root.value, { p: navP.value })
      }
    }

    const selectProject = (pIndex) => {
      commitDraft()
      flowSel.value = null
      navP.value = pIndex
      navS.value = null
      editDraft.value = cloneNodeByPath(root.value, { p: pIndex })
    }

    const selectSub = (pIndex, sIndex) => {
      commitDraft()
      flowSel.value = null
      navP.value = pIndex
      navS.value = sIndex
      editDraft.value = cloneNodeByPath(root.value, { p: pIndex, s: sIndex })
    }

    const selectCommand = (cIndex) => {
      const next = { p: navP.value, s: navS.value, c: cIndex }
      if (samePath(next, editPath.value)) return
      openEditPath(next)
    }

    const selectStep = (cIndex, stIndex) => {
      openEditPath({ p: navP.value, s: navS.value, c: cIndex, st: stIndex })
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
      openEditPath({
        p: navP.value,
        s: navS.value,
        c: sub.commands.length - 1,
      })
    }

    const insertCommand = (atIndex) => {
      const sub = ensureSubCommands()
      if (!sub) return
      commitDraft()
      sub.commands.splice(atIndex, 0, emptyCommand('batch'))
      openEditPath({ p: navP.value, s: navS.value, c: atIndex })
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
        openEditPath({
          p: navP.value,
          s: navS.value,
          c: prevSel.c - 1,
          st: prevSel.st,
        })
      } else if (prevSel?.c === cIndex) {
        flowSel.value = null
        editDraft.value = cloneNodeByPath(root.value, {
          p: navP.value,
          s: navS.value,
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
      openEditPath({
        p: navP.value,
        s: navS.value,
        c: cIndex,
        st: cmd.steps.length - 1,
      })
    }

    const removeStep = (cIndex, stIndex) => {
      const cmd = ensureSubCommands()?.commands?.[cIndex]
      if (!cmd?.steps) return
      const prevSel = flowSel.value ? { ...flowSel.value } : null
      cmd.steps.splice(stIndex, 1)
      if (prevSel?.c === cIndex && prevSel?.st === stIndex) {
        openEditPath({ p: navP.value, s: navS.value, c: cIndex })
      } else if (prevSel?.c === cIndex && prevSel?.st != null && prevSel.st > stIndex) {
        openEditPath({
          p: navP.value,
          s: navS.value,
          c: cIndex,
          st: prevSel.st - 1,
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
      flowSel.value = null
      navP.value = p
      navS.value = null
      editDraft.value = cloneNodeByPath(root.value, { p })
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
      const sIndex = project.subprojects.length - 1
      flowSel.value = null
      navP.value = pIndex
      navS.value = sIndex
      editDraft.value = cloneNodeByPath(root.value, { p: pIndex, s: sIndex })
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
        editDraft.value = cloneNodeByPath(root.value, { p: pIndex })
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
      markClean()
    }

    const reload = async () => {
      if (hasUnsavedChanges()) {
        try {
          await ElMessageBox.confirm('重新加载将丢弃未保存的修改，是否继续？', '重新加载', {
            type: 'warning',
            confirmButtonText: '重新加载',
            cancelButtonText: '取消',
          })
        } catch {
          return
        }
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
        markClean()
        ElMessage.success('配置已保存')
        emit('saved')
        visibleProxy.value = false
      } catch (e) {
        ElMessage.error(`保存失败: ${e}`)
      } finally {
        saving.value = false
      }
    }

    /** 关闭弹框；有未保存修改时二次确认。done 来自 el-dialog before-close */
    const handleClose = async (done) => {
      if (hasUnsavedChanges()) {
        try {
          await ElMessageBox.confirm('有未保存的修改，关闭后将丢失，是否继续？', '关闭确认', {
            type: 'warning',
            confirmButtonText: '放弃修改',
            cancelButtonText: '继续编辑',
            distinguishCancelAndClose: true,
          })
        } catch {
          return
        }
      }
      if (typeof done === 'function') done()
      else visibleProxy.value = false
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
      navP,
      navS,
      activeSub,
      contextLabel,
      canvasEmptyHint,
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
  height: min(78vh, 740px);
  min-height: 520px;
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius-lg, 10px);
  overflow: hidden;
  background: var(--app-bg);
}

.shell-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--app-border);
  background: var(--app-panel-bg);
  flex-shrink: 0;
  min-height: 48px;
  box-sizing: border-box;
}

.toolbar-title {
  flex-shrink: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
}

.shell-context {
  flex: 1;
  min-width: 0;
  display: flex;
  justify-content: center;
}

.context-label {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: var(--app-text-muted);
  padding: 4px 10px;
  border-radius: var(--app-radius-sm, 6px);
  background: color-mix(in srgb, var(--app-border) 45%, transparent);
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
  flex-direction: column;
  background: var(--app-panel-bg);
}

.flow-workspace {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
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
  max-width: calc(100vw - 48px);
  max-height: 92vh;
}

.task-config-dialog .el-dialog__header {
  flex-shrink: 0;
  margin-right: 0;
  padding: 16px 20px 12px;
}

.task-config-dialog .el-dialog__title {
  font-size: 16px;
  font-weight: 600;
}

.task-config-dialog .el-dialog__body {
  padding: 0 16px 12px;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.task-config-dialog .el-dialog__footer {
  flex-shrink: 0;
  padding: 10px 16px 14px;
  border-top: 1px solid var(--app-border);
}
</style>
