<template>
  <section class="task-flow-inspector" :class="`kind-${selectionKind}`">
    <header class="inspector-head">
      <div class="inspector-title-wrap">
        <span class="inspector-badge" :class="selectionKind">{{ kindBadge }}</span>
        <div class="inspector-heading">
          <span class="inspector-title">{{ title }}</span>
          <span v-if="subtitle" class="inspector-sub">{{ subtitle }}</span>
        </div>
      </div>
      <div class="inspector-actions icon-actions">
        <el-tooltip v-if="canDelete" content="删除" placement="top">
          <el-button type="danger" text size="small" circle @click="$emit('delete')">
            <el-icon><Delete /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip v-if="canClose" content="返回子项目属性" placement="top">
          <el-button text size="small" circle @click="$emit('close')">
            <el-icon><Back /></el-icon>
          </el-button>
        </el-tooltip>
      </div>
    </header>

    <div class="inspector-body">
      <!-- 项目 / 子项目 -->
      <div v-if="selectionKind === 'project' || selectionKind === 'sub'" class="field-grid meta-grid">
        <label class="field">
          <span class="field-label">名称</span>
          <el-input
            v-model="localDraft.name"
            size="small"
            :placeholder="selectionKind === 'project' ? '项目名称' : '子项目名称'"
            @input="emitChange"
          />
        </label>
        <label class="field field-span-2">
          <span class="field-label">描述</span>
          <el-input
            v-model="localDraft.description"
            size="small"
            placeholder="可选说明"
            @input="emitChange"
          />
        </label>
        <label class="field field-span-2">
          <span class="field-label">工作目录</span>
          <el-input
            v-model="localDraft.workdir"
            size="small"
            placeholder="可选，支持 ${ENV}"
            @input="emitChange"
          />
        </label>
      </div>

      <!-- 命令 -->
      <div v-else-if="selectionKind === 'command'" class="field-grid command-grid">
        <label class="field">
          <span class="field-label">任务名称</span>
          <el-input
            v-model="localDraft.name"
            size="small"
            placeholder="如：构建 / 部署"
            @input="emitChange"
          />
        </label>
        <div class="field">
          <span class="field-label">执行位置</span>
          <el-radio-group v-model="localDraft.type" size="small" @change="onTypeChange">
            <el-radio-button label="batch">本机</el-radio-button>
            <el-radio-button label="remote">远程</el-radio-button>
          </el-radio-group>
        </div>
        <label v-if="localDraft.type === 'remote'" class="field">
          <span class="field-label">目标机器</span>
          <el-select
            v-model="localDraft.machine"
            filterable
            allow-create
            default-first-option
            size="small"
            placeholder="选择或输入机器名"
            style="width: 100%"
            @change="emitChange"
          >
            <el-option v-for="m in machineNames" :key="m" :label="m" :value="m" />
          </el-select>
        </label>
        <label class="field">
          <span class="field-label">描述</span>
          <el-input
            v-model="localDraft.description"
            size="small"
            placeholder="可选"
            @input="emitChange"
          />
        </label>
        <label class="field field-span-2">
          <span class="field-label">工作目录</span>
          <el-input
            v-model="localDraft.workdir"
            size="small"
            placeholder="可选，支持 ${ENV}"
            @input="emitChange"
          />
        </label>
      </div>

      <!-- 步骤 -->
      <div v-else-if="selectionKind === 'step'" class="field-grid step-grid">
        <div class="field">
          <span class="field-label">步骤类型</span>
          <el-select v-model="localDraft.kind" size="small" style="width: 100%" @change="onKindChange">
            <el-option label="普通命令" value="shell" />
            <el-option label="上传文件 (upload)" value="upload" />
            <el-option label="切换目录 (chdir)" value="chdir" />
            <el-option label="打包 (targz)" value="targz" />
          </el-select>
        </div>
        <div class="field">
          <span class="field-label">失败策略</span>
          <el-select v-model="localDraft.onFail" size="small" style="width: 100%" @change="emitChange">
            <el-option label="中止" value="abort" />
            <el-option label="继续" value="continue" />
          </el-select>
        </div>
        <div class="field field-retry">
          <span class="field-label">重试次数</span>
          <el-input-number
            v-model="localDraft.retry"
            size="small"
            :min="0"
            :max="10"
            controls-position="right"
            @change="emitChange"
          />
        </div>

        <template v-if="localDraft.kind === 'upload'">
          <label class="field field-span-2">
            <span class="field-label">本地路径</span>
            <el-input v-model="uploadLocal" size="small" placeholder="本地文件或目录" @input="syncUpload" />
          </label>
          <label class="field field-span-2">
            <span class="field-label">远程路径</span>
            <el-input v-model="uploadRemote" size="small" placeholder="远程目标路径" @input="syncUpload" />
          </label>
        </template>
        <template v-else-if="localDraft.kind === 'chdir'">
          <label class="field field-span-full">
            <span class="field-label">远程目录</span>
            <el-input v-model="chdirPath" size="small" placeholder="/opt/app" @input="syncChdir" />
          </label>
        </template>
        <template v-else-if="localDraft.kind === 'targz'">
          <label class="field field-span-2">
            <span class="field-label">源路径</span>
            <el-input v-model="targzSrc" size="small" @input="syncTargz" />
          </label>
          <label class="field field-span-2">
            <span class="field-label">目标 .tar.gz</span>
            <el-input v-model="targzDest" size="small" @input="syncTargz" />
          </label>
        </template>
        <label v-else class="field field-span-full">
          <span class="field-label">命令</span>
          <el-input
            v-model="localDraft.command"
            type="textarea"
            :rows="2"
            size="small"
            placeholder="shell 命令"
            class="cmd-input"
            @input="emitChange"
          />
        </label>
      </div>
    </div>
  </section>
</template>

<script>
import { computed, ref, watch } from 'vue'
import { Back, Delete } from '@element-plus/icons-vue'
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
  components: { Back, Delete },
  props: {
    selectionKind: { type: String, default: '' },
    draft: { type: Object, default: null },
    machines: { type: Array, default: () => [] },
  },
  emits: ['update:draft', 'close', 'delete'],
  setup(props, { emit }) {
    const localDraft = ref({})
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
          return '项目'
        case 'sub':
          return '子项目'
        case 'command':
          return '命令'
        case 'step':
          return '步骤'
        default:
          return '属性'
      }
    })

    const kindBadge = computed(() => title.value)

    const subtitle = computed(() => {
      if (props.selectionKind === 'command') {
        return localDraft.value?.name || '未命名'
      }
      if (props.selectionKind === 'step') {
        const kind = localDraft.value?.kind || 'shell'
        const map = {
          shell: '普通命令',
          upload: '上传文件',
          chdir: '切换目录',
          targz: '打包',
        }
        return map[kind] || kind
      }
      return localDraft.value?.name || '未命名'
    })

    const canDelete = computed(
      () => props.selectionKind === 'command' || props.selectionKind === 'step',
    )

    const canClose = computed(
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
      machineNames,
      title,
      kindBadge,
      subtitle,
      canDelete,
      canClose,
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
.task-flow-inspector {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  border-top: 1px solid var(--app-border);
  background: var(--app-card-bg);
  max-height: 38%;
  min-height: 120px;
}

.inspector-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 16px;
  border-bottom: 1px solid color-mix(in srgb, var(--app-border) 70%, transparent);
  flex-shrink: 0;
}

.inspector-title-wrap {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.inspector-badge {
  flex-shrink: 0;
  font-size: 11px;
  font-weight: 600;
  line-height: 1;
  padding: 5px 8px;
  border-radius: var(--app-radius-sm, 6px);
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}

.inspector-badge.step {
  background: color-mix(in srgb, var(--app-success-color, #67c23a) 14%, transparent);
  color: var(--app-success-color, #67c23a);
}

.inspector-badge.project,
.inspector-badge.sub {
  background: color-mix(in srgb, var(--app-warning-color, #e6a23c) 16%, transparent);
  color: var(--app-warning-color, #e6a23c);
}

.inspector-heading {
  display: flex;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
}

.inspector-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
  flex-shrink: 0;
}

.inspector-sub {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: var(--app-text-muted);
}

.inspector-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 12px 16px 14px;
}

.field-grid {
  display: grid;
  gap: 10px 14px;
  align-items: end;
}

.meta-grid {
  grid-template-columns: minmax(180px, 1fr) minmax(180px, 1.2fr) minmax(180px, 1.2fr);
}

.command-grid {
  grid-template-columns: minmax(160px, 1fr) auto minmax(160px, 1fr) minmax(160px, 1fr);
}

.step-grid {
  grid-template-columns: minmax(140px, 0.9fr) minmax(110px, 0.7fr) minmax(110px, 0.6fr) minmax(160px, 1fr);
}

.field {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.field-label {
  font-size: 12px;
  line-height: 1.2;
  color: var(--app-text-secondary);
}

.field-span-2 {
  grid-column: span 2;
}

.field-span-full {
  grid-column: 1 / -1;
}

.field-retry :deep(.el-input-number) {
  width: 100%;
}

.cmd-input :deep(.el-textarea__inner) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.45;
}

@media (max-width: 1100px) {
  .meta-grid,
  .command-grid,
  .step-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .field-span-2,
  .field-span-full {
    grid-column: 1 / -1;
  }
}
</style>
