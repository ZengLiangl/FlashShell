<template>
  <div class="shell-history">
    <div class="app-surface-narrow">
      <div class="app-toolbar">
        <el-input
          v-model="keyword"
          clearable
          size="small"
          class="app-toolbar-search"
          placeholder="搜索机器名 / 地址"
          @keydown.enter.prevent="connectFirst"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <div class="icon-actions">
          <el-tooltip content="返回首页" placement="top">
            <el-button size="small" circle @click="$emit('back')">
              <el-icon><ArrowLeft /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip content="选择机器" placement="top">
            <el-button size="small" type="primary" circle @click="$emit('open-picker')">
              <el-icon><Monitor /></el-icon>
            </el-button>
          </el-tooltip>
        </div>
      </div>

      <div v-if="!records.length" class="app-empty">
        <p class="app-empty-title">暂无连接历史</p>
        <p class="app-empty-desc">从机器列表选择一台开始 SSH 会话</p>
        <el-button type="primary" size="small" @click="$emit('open-picker')">打开连接管理器</el-button>
      </div>

      <template v-else>
        <div class="app-section-head">
          <span class="app-section-label">最近连接</span>
          <button
            type="button"
            class="clear-link"
            :disabled="!records.length"
            @click="onClear"
          >
            清空
          </button>
        </div>

        <div v-if="!filtered.length" class="app-empty" style="min-height: 120px; padding: 28px 16px">
          <p class="app-empty-desc">没有匹配「{{ keyword }}」的记录</p>
        </div>

        <ul v-else class="ml-list history-scroll" role="listbox">
          <li
            v-for="row in filtered"
            :key="rowKey(row)"
            class="ml-item"
            :class="{ connected: isConnected(row.machineName) }"
            role="option"
            tabindex="0"
            @click="onRowClick(row)"
            @keydown.enter.prevent="onRowClick(row)"
          >
            <div class="ml-machine-icon" aria-hidden="true">
              <el-icon :size="16"><Monitor /></el-icon>
            </div>
            <div class="ml-body">
              <div class="ml-line">
                <span class="ml-name">{{ row.machineName }}</span>
                <span v-if="isConnected(row.machineName)" class="ml-badge">已连接</span>
              </div>
              <div class="ml-addr">{{ formatAddr(row) }}</div>
            </div>
            <div class="ml-side-meta">
              <span class="ml-meta-time">{{ formatRelative(row.lastConnectedAt) }}</span>
              <span class="ml-meta-count">{{ row.connectCount || 1 }} 次</span>
              <button
                type="button"
                class="ml-icon-btn is-danger"
                title="删除"
                @click.stop="$emit('remove', row)"
              >
                <el-icon :size="14"><Close /></el-icon>
              </button>
            </div>
          </li>
        </ul>
      </template>
    </div>
  </div>
</template>

<script>
import { computed, ref } from 'vue'
import { Search, Close, ArrowLeft, Monitor } from '@element-plus/icons-vue'
import { formatMachineAddr } from '../../utils/machineGroups'

export default {
  name: 'ShellConnectionHistory',
  components: { Search, Close, ArrowLeft, Monitor },
  props: {
    records: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
  },
  emits: ['connect', 'open-picker', 'clear', 'remove', 'back'],
  setup(props, { emit }) {
    const keyword = ref('')

    const rowKey = (row) => `${row.machineId || ''}:${row.machineName || ''}:${row.host || ''}`

    const formatAddr = (row) => formatMachineAddr(row)

    const isConnected = (name) =>
      (props.sessions || []).some((s) => s.machineName === name && s.connected)

    const filtered = computed(() => {
      const kw = String(keyword.value || '').trim().toLowerCase()
      const list = props.records || []
      if (!kw) return list
      return list.filter((row) => {
        const hay = `${row.machineName || ''} ${row.user || ''} ${row.host || ''} ${row.port || ''}`.toLowerCase()
        return hay.includes(kw)
      })
    })

    const formatRelative = (ts) => {
      if (!ts) return ''
      const now = Date.now()
      const t = Number(ts) * 1000
      const diff = Math.max(0, now - t)
      const min = Math.floor(diff / 60000)
      if (min < 1) return '刚刚'
      if (min < 60) return `${min} 分钟前`
      const hour = Math.floor(min / 60)
      if (hour < 24) return `${hour} 小时前`
      const day = Math.floor(hour / 24)
      if (day < 30) return `${day} 天前`
      const d = new Date(t)
      const pad = (n) => String(n).padStart(2, '0')
      return `${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
    }

    const onRowClick = (row) => emit('connect', row.machineName)
    const connectFirst = () => {
      const first = filtered.value[0]
      if (first) onRowClick(first)
    }
    const onClear = () => emit('clear')

    return {
      keyword,
      filtered,
      rowKey,
      formatAddr,
      isConnected,
      formatRelative,
      onRowClick,
      connectFirst,
      onClear,
    }
  },
}
</script>

<style scoped>
.shell-history {
  flex: 1;
  min-height: 0;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 24px 20px;
  background: var(--app-bg);
  overflow: auto;
}

.clear-link {
  border: none;
  background: transparent;
  padding: 0;
  font-size: 12px;
  color: var(--app-text-muted);
  cursor: pointer;
}

.clear-link:hover:not(:disabled) {
  color: #f56c6c;
}

.clear-link:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.history-scroll {
  max-height: min(52vh, 420px);
  overflow: auto;
}
</style>
