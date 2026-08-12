<template>
  <el-drawer
    v-model="visibleProxy"
    title="全局传输中心"
    direction="rtl"
    size="420px"
    :append-to-body="true"
    class="shell-transfer-drawer"
  >
    <div class="transfer-toolbar icon-actions">
      <el-tooltip content="打开下载目录" placement="top">
        <el-button size="small" text type="primary" @click="openDownloadDir">
          <el-icon><FolderOpened /></el-icon>
        </el-button>
      </el-tooltip>
      <el-tooltip content="全部暂停" placement="top">
        <el-button size="small" text type="warning" @click="pauseAll">
          <el-icon><VideoPause /></el-icon>
        </el-button>
      </el-tooltip>
      <el-tooltip content="全部继续" placement="top">
        <el-button size="small" text type="primary" @click="resumeAll">
          <el-icon><VideoPlay /></el-icon>
        </el-button>
      </el-tooltip>
      <el-tooltip content="清除已结束" placement="top">
        <el-button size="small" text @click="clearFinished">
          <el-icon><Delete /></el-icon>
        </el-button>
      </el-tooltip>
    </div>
    <div v-if="records.length === 0" class="transfer-empty">暂无传输记录</div>
    <ul v-else class="transfer-list">
      <li v-for="item in records" :key="item.id" class="transfer-item">
        <div class="transfer-head">
          <span class="dir-tag" :class="item.direction">{{ item.direction === 'download' ? '下载' : '上传' }}</span>
          <span class="name" :title="item.name">{{ item.name }}</span>
          <span class="status" :class="item.status">{{ statusLabel(item) }}</span>
        </div>
        <div class="meta" :title="item.remotePath">
          {{ item.machineName }} · {{ item.isDir ? '目录' : '文件' }}
        </div>
        <el-progress
          v-if="item.status === 'running' || item.status === 'pending' || item.status === 'queued'"
          :percentage="Math.min(100, Math.round(item.percent || 0))"
          :stroke-width="6"
          :show-text="false"
        />
        <div class="progress-row">
          <span v-if="item.status === 'queued'">排队中（优先级 {{ item.priority || 0 }}）</span>
          <span v-else-if="item.status === 'running' || item.status === 'pending'">
            {{ formatSize(item.transferred) }}
            <template v-if="item.total > 0"> / {{ formatSize(item.total) }}</template>
            <template v-if="item.speedBps > 0"> · {{ formatSpeed(item.speedBps) }}</template>
          </span>
          <span v-else-if="item.status === 'done'">
            {{ formatSize(item.total || item.transferred) }} · 完成
            <template v-if="avgSpeedBps(item) > 0"> · 平均 {{ formatSpeed(avgSpeedBps(item)) }}</template>
          </span>
          <span v-else-if="item.status === 'paused'">已暂停 · {{ formatSize(item.transferred) }}</span>
          <span v-else-if="item.status === 'error'" class="err" :title="item.error">{{ item.error || '失败' }}</span>
        </div>
        <div class="item-actions icon-actions">
          <el-tooltip v-if="item.status === 'queued'" content="优先传输" placement="top">
            <el-button size="small" text type="primary" @click="prioritizeItem(item)">优先</el-button>
          </el-tooltip>
          <el-tooltip v-if="item.status === 'running' || item.status === 'pending' || item.status === 'queued'" content="暂停" placement="top">
            <el-button size="small" text type="warning" @click="pauseItem(item)">
              <el-icon><VideoPause /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip v-if="item.status === 'paused' || item.status === 'error'" content="继续" placement="top">
            <el-button size="small" text type="primary" @click="resumeItem(item)">
              <el-icon><VideoPlay /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip content="删除" placement="top">
            <el-button size="small" text type="danger" @click="removeItem(item)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </el-tooltip>
        </div>
      </li>
    </ul>
  </el-drawer>
</template>

<script>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { FolderOpened, Delete, VideoPause, VideoPlay } from '@element-plus/icons-vue'
import * as App from '../../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'

export default {
  name: 'ShellTransferPanel',
  components: { FolderOpened, Delete, VideoPause, VideoPlay },
  props: {
    modelValue: { type: Boolean, default: false },
  },
  emits: ['update:modelValue', 'active-change'],
  setup(props, { emit, expose }) {
    const visibleProxy = ref(props.modelValue)
    const records = ref([])
    /** 用户手动关闭后，同一批传输进度不再自动打开；新任务会清掉此标记 */
    const suppressAutoOpen = ref(false)
    const EVENT = 'shell:transfer'

    watch(() => props.modelValue, (v) => { visibleProxy.value = v })
    watch(visibleProxy, (v, prev) => {
      emit('update:modelValue', v)
      if (!v && prev) {
        // 用户关掉抽屉
        suppressAutoOpen.value = true
      }
      if (v) {
        // 手动打开则恢复自动打开能力（针对后续新任务）
        // 注意：关掉后 suppress 为 true，进度事件不会再打开
      }
    })

    const activeCount = computed(() =>
      records.value.filter((r) => r.status === 'running' || r.status === 'pending' || r.status === 'queued').length,
    )

    watch(activeCount, (n) => emit('active-change', n), { immediate: true })

    const upsert = (rec) => {
      if (!rec?.id) return
      if (rec.status === 'removed') {
        records.value = records.value.filter((r) => r.id !== rec.id)
        return
      }
      const idx = records.value.findIndex((r) => r.id === rec.id)
      if (idx >= 0) {
        const cur = records.value[idx]
        // 忽略完成/失败后迟到的 running 进度事件，避免卡在「解压中」
        if ((cur.status === 'done' || cur.status === 'error') &&
            (rec.status === 'running' || rec.status === 'pending' || rec.status === 'queued')) {
          return
        }
        records.value.splice(idx, 1, { ...records.value[idx], ...rec })
      } else {
        records.value.unshift(rec)
      }
    }

    const callGo = async (name, ...args) => {
      const fromModule = App?.[name]
      const fromRuntime = typeof window !== 'undefined'
        ? window?.go?.app?.App?.[name]
        : null
      const fn = typeof fromModule === 'function' ? fromModule : fromRuntime
      if (typeof fn !== 'function') {
        throw new Error(`${name} 不可用：请完全退出并重新运行 wails dev（仅热更新前端不够）`)
      }
      return fn(...args)
    }

    const load = async () => {
      try {
        const list = await App.ListShellTransfers()
        records.value = list || []
      } catch {
        records.value = []
      }
    }

    const open = () => {
      suppressAutoOpen.value = false
      visibleProxy.value = true
    }
    const close = () => { visibleProxy.value = false }

    const openDownloadDir = async () => {
      try {
        await App.OpenShellDownloadDir()
      } catch (e) {
        ElMessage.error('打开下载目录失败: ' + e)
      }
    }

    const clearFinished = async () => {
      try {
        await App.ClearFinishedShellTransfers()
        records.value = records.value.filter((r) =>
          r.status === 'running' || r.status === 'pending' || r.status === 'queued',
        )
      } catch (e) {
        ElMessage.error('清除失败: ' + e)
      }
    }

    const pauseAll = async () => {
      try {
        await callGo('PauseAllShellTransfers')
        await load()
      } catch (e) {
        ElMessage.error('全部暂停失败: ' + e)
      }
    }

    const resumeAll = async () => {
      try {
        await callGo('ResumeAllShellTransfers')
        await load()
      } catch (e) {
        ElMessage.error('全部继续失败: ' + e)
      }
    }

    const prioritizeItem = async (item) => {
      if (!item?.id) return
      try {
        await callGo('PrioritizeShellTransfer', item.id)
      } catch (e) {
        ElMessage.error('优先失败: ' + e)
      }
    }

    const pauseItem = async (item) => {
      if (!item?.id) return
      try {
        await callGo('PauseShellTransfer', item.id)
      } catch (e) {
        ElMessage.error('暂停失败: ' + e)
      }
    }

    const resumeItem = async (item) => {
      if (!item?.id) return
      try {
        await callGo('ResumeShellTransfer', item.id)
      } catch (e) {
        ElMessage.error('继续失败: ' + e)
      }
    }

    const removeItem = async (item) => {
      if (!item?.id) return
      try {
        await callGo('RemoveShellTransfer', item.id)
        records.value = records.value.filter((r) => r.id !== item.id)
      } catch (e) {
        ElMessage.error('删除失败: ' + e)
      }
    }

    const formatSize = (n) => {
      if (n == null || n <= 0) return '0 B'
      const units = ['B', 'KB', 'MB', 'GB', 'TB']
      let v = n
      let i = 0
      while (v >= 1024 && i < units.length - 1) {
        v /= 1024
        i++
      }
      return i === 0 ? `${v} ${units[i]}` : `${v.toFixed(2)} ${units[i]}`
    }

    const formatSpeed = (bps) => {
      if (!bps || bps <= 0) return ''
      const units = ['B/s', 'KB/s', 'MB/s', 'GB/s', 'TB/s']
      let v = bps
      let i = 0
      while (v >= 1024 && i < units.length - 1) {
        v /= 1024
        i++
      }
      return i === 0 ? `${Math.round(v)} ${units[i]}` : `${v.toFixed(2)} ${units[i]}`
    }

    const avgSpeedBps = (item) => {
      if (!item) return 0
      const bytes = item.total || item.transferred || 0
      const start = item.startedAt || 0
      const end = item.finishedAt || item.updatedAt || 0
      if (bytes > 0 && start > 0 && end > start) {
        return bytes / (end - start)
      }
      return item.speedBps > 0 ? item.speedBps : 0
    }

    const statusLabel = (item) => {
      const s = typeof item === 'string' ? item : item?.status
      if (s === 'queued') return '排队'
      if (s === 'pending') return '等待'
      if (s === 'running') {
        const phase = typeof item === 'object' ? item?.phase : ''
        if (phase === 'compressing') return '压缩中'
        if (phase === 'uploading') return '上传中'
        if (phase === 'extracting') return '解压中'
        return '进行中'
      }
      if (s === 'done') return '完成'
      if (s === 'error') return '失败'
      if (s === 'paused') return '已暂停'
      return s || ''
    }

    onMounted(async () => {
      await load()
      EventsOn(EVENT, (payload) => {
        if (!payload?.id) return
        const isNewTask = (payload.status === 'pending' || payload.status === 'queued')
          && !records.value.some((r) => r.id === payload.id)
        upsert(payload)
        if (isNewTask && !suppressAutoOpen.value) {
          visibleProxy.value = true
        } else if (isNewTask) {
          suppressAutoOpen.value = false
          visibleProxy.value = true
        }
      })
    })

    onUnmounted(() => {
      EventsOff(EVENT)
    })

    expose({ open, close, activeCount })

    return {
      visibleProxy,
      records,
      openDownloadDir,
      clearFinished,
      pauseAll,
      resumeAll,
      prioritizeItem,
      pauseItem,
      resumeItem,
      removeItem,
      formatSize,
      formatSpeed,
      avgSpeedBps,
      statusLabel,
    }
  },
}
</script>

<style scoped>
.transfer-toolbar {
  justify-content: flex-end;
  width: 100%;
  margin-bottom: 10px;
  flex-wrap: wrap;
  gap: 4px;
}

.status.queued {
  color: var(--el-color-info);
}

.transfer-empty {
  padding: 32px 12px;
  text-align: center;
  color: var(--app-text-muted);
  font-size: 13px;
}

.transfer-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.transfer-item {
  padding: 10px 12px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-panel-bg);
}

.transfer-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.dir-tag {
  flex-shrink: 0;
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 4px;
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 12%, transparent);
  color: var(--app-accent-color, #409eff);
}

.dir-tag.upload {
  background: rgba(103, 194, 58, 0.12);
  color: #67c23a;
}

.name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 500;
}

.status {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--app-text-muted);
}

.status.running,
.status.pending {
  color: var(--app-accent-color, #409eff);
}

.status.done {
  color: #67c23a;
}

.status.error {
  color: #f56c6c;
}

.status.paused {
  color: #e6a23c;
}

.meta {
  font-size: 11px;
  color: var(--app-text-muted);
  margin-bottom: 6px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.progress-row {
  margin-top: 4px;
  font-size: 11px;
  color: var(--app-text-secondary);
}

.progress-row .err {
  color: #f56c6c;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.item-actions {
  justify-content: flex-end;
  margin-top: 4px;
  width: 100%;
}
</style>
