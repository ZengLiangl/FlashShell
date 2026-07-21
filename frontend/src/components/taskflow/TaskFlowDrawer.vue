<template>
  <aside class="task-flow-drawer">
    <template v-if="selectionKind">
      <div class="drawer-header">
        <span class="drawer-title">{{ title }}</span>
        <div class="drawer-header-actions icon-actions">
          <el-tooltip v-if="canDelete" content="删除" placement="top">
            <el-button type="danger" text size="small" circle @click="$emit('delete')">
              <el-icon><Delete /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip content="关闭" placement="top">
            <el-button text size="small" circle @click="$emit('close')">
              <el-icon><Close /></el-icon>
            </el-button>
          </el-tooltip>
        </div>
      </div>

      <div class="drawer-body">
        <!-- 项目 / 子项目基本信息 -->
        <el-form
          v-if="selectionKind === 'project' || selectionKind === 'sub'"
          label-position="top"
          size="small"
          class="drawer-form"
        >
          <el-form-item label="名称">
            <el-input v-model="localDraft.name" placeholder="名称" @change="emitChange" />
          </el-form-item>
          <el-form-item label="描述">
            <el-input
              v-model="localDraft.description"
              type="textarea"
              :rows="2"
              placeholder="可选"
              @change="emitChange"
            />
          </el-form-item>
          <el-form-item label="工作目录">
            <el-input
              v-model="localDraft.workdir"
              placeholder="可选，支持 ${ENV}"
              @change="emitChange"
            />
          </el-form-item>
        </el-form>

        <!-- 命令 Stage -->
        <el-form
          v-else-if="selectionKind === 'command'"
          label-position="top"
          size="small"
          class="drawer-form"
        >
          <el-form-item label="任务名称" required>
            <el-input v-model="localDraft.name" placeholder="如：构建 / 部署" @change="emitChange" />
          </el-form-item>
          <el-form-item label="执行位置" required>
            <el-radio-group v-model="localDraft.type" @change="onTypeChange">
              <el-radio-button label="batch">本机</el-radio-button>
              <el-radio-button label="remote">远程</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="localDraft.type === 'remote'" label="目标机器" required>
            <el-select
              v-model="localDraft.machine"
              filterable
              allow-create
              default-first-option
              placeholder="选择或输入机器名"
              style="width: 100%"
              @change="emitChange"
            >
              <el-option
                v-for="m in machineNames"
                :key="m"
                :label="m"
                :value="m"
              />
            </el-select>
          </el-form-item>

          <el-collapse v-model="advancedOpen" class="adv-collapse">
            <el-collapse-item title="高级设置" name="adv">
              <el-form-item label="描述">
                <el-input
                  v-model="localDraft.description"
                  type="textarea"
                  :rows="2"
                  @change="emitChange"
                />
              </el-form-item>
              <el-form-item label="工作目录">
                <el-input
                  v-model="localDraft.workdir"
                  placeholder="可选"
                  @change="emitChange"
                />
              </el-form-item>
            </el-collapse-item>
          </el-collapse>
        </el-form>

        <!-- 步骤 -->
        <el-form
          v-else-if="selectionKind === 'step'"
          label-position="top"
          size="small"
          class="drawer-form"
        >
          <el-form-item label="步骤类型">
            <el-select
              v-model="localDraft.kind"
              style="width: 100%"
              @change="onKindChange"
            >
              <el-option label="普通命令" value="shell" />
              <el-option label="上传文件 (upload)" value="upload" />
              <el-option label="切换目录 (chdir)" value="chdir" />
              <el-option label="打包 (targz)" value="targz" />
            </el-select>
          </el-form-item>

          <template v-if="localDraft.kind === 'upload'">
            <el-form-item label="本地路径" required>
              <el-input v-model="uploadLocal" placeholder="本地文件或目录" @change="syncUpload" />
            </el-form-item>
            <el-form-item label="远程路径" required>
              <el-input v-model="uploadRemote" placeholder="远程目标路径" @change="syncUpload" />
            </el-form-item>
          </template>
          <template v-else-if="localDraft.kind === 'chdir'">
            <el-form-item label="远程目录" required>
              <el-input v-model="chdirPath" placeholder="/opt/app" @change="syncChdir" />
            </el-form-item>
          </template>
          <template v-else-if="localDraft.kind === 'targz'">
            <el-form-item label="源路径" required>
              <el-input v-model="targzSrc" @change="syncTargz" />
            </el-form-item>
            <el-form-item label="目标 .tar.gz" required>
              <el-input v-model="targzDest" @change="syncTargz" />
            </el-form-item>
          </template>
          <el-form-item v-else label="命令" required>
            <el-input
              v-model="localDraft.command"
              type="textarea"
              :rows="3"
              placeholder="shell 命令"
              @change="emitChange"
            />
          </el-form-item>

          <el-collapse v-model="advancedOpen" class="adv-collapse">
            <el-collapse-item title="高级设置" name="adv">
              <el-form-item label="失败策略">
                <el-select v-model="localDraft.onFail" style="width: 100%" @change="emitChange">
                  <el-option label="中止 (abort)" value="abort" />
                  <el-option label="继续 (continue)" value="continue" />
                </el-select>
              </el-form-item>
              <el-form-item label="失败重试次数">
                <el-input-number
                  v-model="localDraft.retry"
                  :min="0"
                  :max="10"
                  controls-position="right"
                  @change="emitChange"
                />
              </el-form-item>
            </el-collapse-item>
          </el-collapse>
        </el-form>
      </div>
    </template>
    <div v-else class="drawer-placeholder">
      <p>选中流水线中的命令或步骤进行配置</p>
      <p class="hint">每次只编辑当前节点，不会影响其它节点</p>
    </div>
  </aside>
</template>

<script>
import { computed, ref, watch } from 'vue'
import { Close, Delete } from '@element-plus/icons-vue'
import {
  buildChdirCommand,
  buildTargzCommand,
  buildUploadCommand,
  deepClone,
  parseChdirPath,
  parseTargzPaths,
  parseUploadPaths,
} from './taskFlowModel'

export default {
  name: 'TaskFlowDrawer',
  components: { Close, Delete },
  props: {
    selectionKind: { type: String, default: '' }, // project | sub | command | step
    draft: { type: Object, default: null },
    machines: { type: Array, default: () => [] },
  },
  emits: ['update:draft', 'close', 'delete'],
  setup(props, { emit }) {
    const localDraft = ref({})
    const advancedOpen = ref([])
    const uploadLocal = ref('')
    const uploadRemote = ref('')
    const chdirPath = ref('')
    const targzSrc = ref('')
    const targzDest = ref('')
    let syncing = false

    const machineNames = computed(() =>
      (props.machines || []).map((m) => m.name).filter(Boolean),
    )

    const title = computed(() => {
      switch (props.selectionKind) {
        case 'project':
          return '编辑项目'
        case 'sub':
          return '编辑子项目'
        case 'command':
          return '编辑命令'
        case 'step':
          return '编辑步骤'
        default:
          return '编辑'
      }
    })

    const canDelete = computed(
      () => props.selectionKind === 'command' || props.selectionKind === 'step',
    )

    const hydrateHelpers = (draft) => {
      if (!draft || props.selectionKind !== 'step') return
      const kind = draft.kind || 'shell'
      if (kind === 'upload') {
        const { local, remote } = parseUploadPaths(draft.command)
        uploadLocal.value = local
        uploadRemote.value = remote
      } else if (kind === 'chdir') {
        chdirPath.value = parseChdirPath(draft.command)
      } else if (kind === 'targz') {
        const { src, dest } = parseTargzPaths(draft.command)
        targzSrc.value = src
        targzDest.value = dest
      }
    }

    watch(
      () => [props.draft, props.selectionKind],
      () => {
        syncing = true
        localDraft.value = props.draft ? deepClone(props.draft) : {}
        advancedOpen.value = []
        hydrateHelpers(localDraft.value)
        syncing = false
      },
      { immediate: true, deep: true },
    )

    const emitChange = () => {
      if (syncing) return
      emit('update:draft', deepClone(localDraft.value))
    }

    const onTypeChange = () => {
      if (localDraft.value.type !== 'remote') {
        localDraft.value.machine = ''
      }
      emitChange()
    }

    const onKindChange = (kind) => {
      if (kind === 'upload') {
        uploadLocal.value = ''
        uploadRemote.value = ''
        localDraft.value.command = 'upload '
      } else if (kind === 'chdir') {
        chdirPath.value = ''
        localDraft.value.command = 'chdir '
      } else if (kind === 'targz') {
        targzSrc.value = ''
        targzDest.value = ''
        localDraft.value.command = 'targz '
      } else {
        localDraft.value.command = ''
      }
      localDraft.value.kind = kind
      emitChange()
    }

    const syncUpload = () => {
      localDraft.value.command = buildUploadCommand(uploadLocal.value, uploadRemote.value)
      localDraft.value.kind = 'upload'
      emitChange()
    }

    const syncChdir = () => {
      localDraft.value.command = buildChdirCommand(chdirPath.value)
      localDraft.value.kind = 'chdir'
      emitChange()
    }

    const syncTargz = () => {
      localDraft.value.command = buildTargzCommand(targzSrc.value, targzDest.value)
      localDraft.value.kind = 'targz'
      emitChange()
    }

    return {
      localDraft,
      advancedOpen,
      machineNames,
      title,
      canDelete,
      uploadLocal,
      uploadRemote,
      chdirPath,
      targzSrc,
      targzDest,
      emitChange,
      onTypeChange,
      onKindChange,
      syncUpload,
      syncChdir,
      syncTargz,
    }
  },
}
</script>

<style scoped>
.task-flow-drawer {
  width: 300px;
  flex-shrink: 0;
  overflow: hidden;
  border-left: 1px solid var(--app-border);
  background: var(--app-bg);
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
}

.drawer-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text);
}

.drawer-header-actions {
  display: flex;
  align-items: center;
  gap: 0;
}

.drawer-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 12px 14px 20px;
}

.drawer-form :deep(.el-form-item) {
  margin-bottom: 14px;
}

.adv-collapse {
  border: none;
  margin-top: 4px;
}

.adv-collapse :deep(.el-collapse-item__header) {
  font-size: 13px;
  color: var(--app-text-secondary);
  height: 36px;
  line-height: 36px;
  border-bottom: none;
  background: transparent;
}

.adv-collapse :deep(.el-collapse-item__wrap) {
  border-bottom: none;
  background: transparent;
}

.adv-collapse :deep(.el-collapse-item__content) {
  padding-bottom: 0;
}

.drawer-placeholder {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px;
  text-align: center;
  color: var(--app-text-muted);
  font-size: 13px;
  gap: 8px;
}

.drawer-placeholder .hint {
  font-size: 12px;
  opacity: 0.85;
}
</style>
