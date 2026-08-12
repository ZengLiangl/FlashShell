<template>
  <div class="history-list-wrap">
    <div v-if="showHead && records.length" class="app-section-head">
      <span class="app-section-label">{{ headLabel }}</span>
      <button
        type="button"
        class="clear-link"
        :disabled="!records.length"
        @click="$emit('clear')"
      >
        清空
      </button>
    </div>

    <div v-if="!embedded && !records.length" class="app-empty compact">
      <p class="app-empty-desc">暂无连接历史</p>
    </div>

    <div v-else-if="!embedded && records.length && !filtered.length" class="app-empty compact">
      <p class="app-empty-desc">没有匹配「{{ keyword }}」的记录</p>
    </div>

    <ul v-else-if="filtered.length" class="ml-list history-scroll" :class="{ embedded }" role="listbox">
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
            <TextOverflowTooltip :text="row.machineName" text-class="ml-name" />
            <span v-if="isConnected(row.machineName)" class="ml-badge">已连接</span>
            <span v-else-if="isOpen(row.machineName)" class="ml-badge is-muted">已打开</span>
          </div>
          <TextOverflowTooltip :text="formatAddr(row)" text-class="ml-addr" />
        </div>
        <div class="ml-side-meta">
          <span class="ml-meta-time" :title="formatAbsolute(row.lastConnectedAt)">{{ formatRelative(row.lastConnectedAt) }}</span>
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
  </div>
</template>

<script>
import { computed } from 'vue'
import { Close, Monitor } from '@element-plus/icons-vue'
import { formatMachineAddr } from '../../utils/machineGroups'
import TextOverflowTooltip from '../TextOverflowTooltip.vue'

export default {
  name: 'ShellHistoryList',
  components: { Close, Monitor, TextOverflowTooltip },
  props: {
    records: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
    workspaceSessions: { type: Array, default: () => [] },
    keyword: { type: String, default: '' },
    showHead: { type: Boolean, default: true },
    headLabel: { type: String, default: '最近连接' },
    embedded: { type: Boolean, default: false },
  },
  emits: ['connect', 'clear', 'remove'],
  setup(props, { emit }) {
    const rowKey = (row) => `${row.machineId || ''}:${row.machineName || ''}:${row.host || ''}`

    const formatAddr = (row) => formatMachineAddr(row)

    const isConnected = (name) =>
      (props.sessions || []).some((s) => s.machineName === name && s.connected)

    const isOpen = (name) =>
      (props.workspaceSessions || []).some((s) => s.machineName === name)

    const filtered = computed(() => {
      const kw = String(props.keyword || '').trim().toLowerCase()
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

    const formatAbsolute = (ts) => {
      if (!ts) return ''
      const d = new Date(Number(ts) * 1000)
      const pad = (n) => String(n).padStart(2, '0')
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
    }

    const onRowClick = (row) => emit('connect', row.machineName)

    return {
      filtered,
      rowKey,
      formatAddr,
      isConnected,
      isOpen,
      formatRelative,
      formatAbsolute,
      onRowClick,
    }
  },
}
</script>

<style scoped>
.history-list-wrap {
  min-height: 0;
}

.history-scroll {
  max-height: min(52vh, 420px);
  overflow: auto;
}

.history-list-wrap .history-scroll {
  max-height: min(46vh, 400px);
}

.history-scroll.embedded {
  max-height: none;
}

.app-empty.compact {
  min-height: 120px;
  padding: 28px 16px;
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

.ml-line :deep(.el-tooltip) {
  flex: 1;
  min-width: 0;
}
</style>
