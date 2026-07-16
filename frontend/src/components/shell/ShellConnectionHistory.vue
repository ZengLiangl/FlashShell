<template>
  <div class="shell-history">
    <div class="history-shell">
      <div class="history-top">
        <el-input
          v-model="keyword"
          clearable
          size="default"
          class="history-search"
          placeholder="搜索机器名 / 地址"
          @keydown.enter.prevent="connectFirst"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <div class="history-actions">
          <el-tooltip content="返回首页" placement="top">
            <button type="button" class="icon-btn" @click="$emit('back')">
              <el-icon :size="16"><ArrowLeft /></el-icon>
            </button>
          </el-tooltip>
          <el-tooltip content="选择机器" placement="top">
            <button type="button" class="icon-btn primary" @click="$emit('open-picker')">
              <el-icon :size="16"><Monitor /></el-icon>
            </button>
          </el-tooltip>
        </div>
      </div>

      <div v-if="!records.length" class="empty">
        <p class="empty-title">暂无连接历史</p>
        <p class="empty-desc">从机器列表选择一台开始 SSH 会话</p>
        <el-button type="primary" size="small" @click="$emit('open-picker')">打开连接管理器</el-button>
      </div>

      <template v-else>
        <div class="list-head">
          <span class="list-label">最近连接</span>
          <button
            type="button"
            class="clear-link"
            :disabled="!records.length"
            @click="onClear"
          >
            清空
          </button>
        </div>

        <div v-if="!filtered.length" class="empty empty-filter">
          没有匹配「{{ keyword }}」的记录
        </div>

        <ul v-else class="history-list" role="listbox">
          <li
            v-for="row in filtered"
            :key="rowKey(row)"
            class="history-item"
            :class="{ connected: isConnected(row.machineName) }"
            role="option"
            tabindex="0"
            @click="onRowClick(row)"
            @keydown.enter.prevent="onRowClick(row)"
          >
            <div class="item-dot" aria-hidden="true" />
            <div class="item-body">
              <div class="item-line">
                <span class="item-name">{{ row.machineName }}</span>
                <span v-if="isConnected(row.machineName)" class="item-badge">已连接</span>
              </div>
              <div class="item-addr">{{ formatAddr(row) }}</div>
            </div>
            <div class="item-side">
              <span class="item-time">{{ formatRelative(row.lastConnectedAt) }}</span>
              <span class="item-count">{{ row.connectCount || 1 }} 次</span>
              <button
                type="button"
                class="item-remove"
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

.history-shell {
  width: 100%;
  max-width: 500px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.history-top {
  display: flex;
  align-items: center;
  gap: 8px;
}

.history-search {
  flex: 1;
  min-width: 0;
}

.history-search :deep(.el-input__wrapper) {
  border-radius: 8px;
  box-shadow: 0 0 0 1px var(--app-border) inset;
  background: var(--app-card-bg, var(--app-panel-bg));
}

.history-search :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px var(--app-accent-color) inset;
}

.history-actions {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
}

.icon-btn {
  width: 34px;
  height: 34px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-card-bg, var(--app-panel-bg));
  color: var(--app-text-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  padding: 0;
  transition: color 0.12s ease, border-color 0.12s ease, background 0.12s ease;
}

.icon-btn:hover {
  color: var(--app-accent-color);
  border-color: var(--app-accent-color);
  background: var(--app-accent-bg);
}

.icon-btn.primary {
  color: #fff;
  border-color: var(--app-accent-color);
  background: var(--app-accent-color);
}

.icon-btn.primary:hover {
  filter: brightness(1.06);
  color: #fff;
  background: var(--app-accent-color);
  border-color: var(--app-accent-color);
}

.list-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  padding: 0 4px;
}

.list-label {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--app-text-muted);
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

.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 40px 12px 28px;
  gap: 8px;
}

.empty-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text);
}

.empty-desc {
  margin: 0 0 10px;
  font-size: 13px;
  color: var(--app-text-muted);
}

.empty-filter {
  color: var(--app-text-muted);
  font-size: 13px;
  padding: 28px 12px;
}

.history-list {
  list-style: none;
  margin: 0;
  padding: 6px;
  max-height: min(52vh, 420px);
  overflow: auto;
  border: 1px solid var(--app-border);
  border-radius: 12px;
  background: var(--app-card-bg, var(--app-panel-bg));
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.history-item {
  display: grid;
  grid-template-columns: 8px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  padding: 10px 8px 10px 10px;
  border-radius: 8px;
  cursor: pointer;
  outline: none;
  transition: background 0.12s ease;
}

.history-item:hover,
.history-item:focus-visible {
  background: var(--app-accent-bg);
}

.item-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--app-border);
  justify-self: center;
}

.history-item.connected .item-dot {
  background: #67c23a;
}

.history-item:hover .item-dot,
.history-item:focus-visible .item-dot {
  background: var(--app-accent-color);
}

.history-item.connected:hover .item-dot,
.history-item.connected:focus-visible .item-dot {
  background: #67c23a;
}

.item-body {
  min-width: 0;
}

.item-line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.item-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-badge {
  flex-shrink: 0;
  font-size: 10px;
  line-height: 1;
  padding: 3px 6px;
  border-radius: 4px;
  color: #67c23a;
  background: color-mix(in srgb, #67c23a 14%, transparent);
}

.item-addr {
  margin-top: 3px;
  font-size: 12px;
  color: var(--app-text-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  position: relative;
  padding-right: 22px;
  flex-shrink: 0;
}

.item-time {
  font-size: 12px;
  color: var(--app-text-secondary);
  white-space: nowrap;
}

.item-count {
  font-size: 11px;
  color: var(--app-text-muted);
  white-space: nowrap;
}

.item-remove {
  position: absolute;
  right: -2px;
  top: 50%;
  transform: translateY(-50%);
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--app-text-muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  padding: 0;
  opacity: 0;
}

.history-item:hover .item-remove,
.history-item:focus-within .item-remove,
.item-remove:focus-visible {
  opacity: 1;
}

.item-remove:hover {
  color: #f56c6c;
  background: color-mix(in srgb, #f56c6c 12%, transparent);
}
</style>
