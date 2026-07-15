<template>
  <div class="shell-history">
    <div class="history-header">
      <div>
        <h3>连接历史</h3>
        <p class="hint">单击记录即可连接；已连接则切换到对应终端</p>
      </div>
      <div class="history-actions icon-actions">
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
        <el-tooltip content="清空历史" placement="top">
          <el-button size="small" type="danger" plain circle :disabled="!records.length" @click="onClear">
            <el-icon><Delete /></el-icon>
          </el-button>
        </el-tooltip>
      </div>
    </div>

    <div v-if="!records.length" class="empty">暂无连接历史，点击「选择机器」开始连接</div>
    <el-table v-else :data="records" size="small" class="history-table" @row-click="onRowClick">
      <el-table-column prop="machineName" label="机器" min-width="140" />
      <el-table-column label="地址" min-width="180">
        <template #default="{ row }">
          {{ row.user }}@{{ row.host }}:{{ row.port || 22 }}
        </template>
      </el-table-column>
      <el-table-column label="上次连接" width="170">
        <template #default="{ row }">
          {{ formatTime(row.lastConnectedAt) }}
        </template>
      </el-table-column>
      <el-table-column prop="connectCount" label="次数" width="70" align="center" />
      <el-table-column label="" width="56" align="center">
        <template #default="{ row }">
          <el-tooltip content="删除" placement="top">
            <el-button size="small" text type="danger" @click.stop="$emit('remove', row)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </el-tooltip>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script>
import { ArrowLeft, Monitor, Delete } from '@element-plus/icons-vue'

export default {
  name: 'ShellConnectionHistory',
  components: { ArrowLeft, Monitor, Delete },
  props: {
    records: { type: Array, default: () => [] },
  },
  emits: ['connect', 'open-picker', 'clear', 'remove', 'back'],
  setup(_, { emit }) {
    const formatTime = (ts) => {
      if (!ts) return '-'
      const d = new Date(ts * 1000)
      const pad = (n) => String(n).padStart(2, '0')
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
    }
    const onRowClick = (row) => emit('connect', row.machineName)
    const onClear = () => emit('clear')
    return { formatTime, onRowClick, onClear }
  },
}
</script>

<style scoped>
.shell-history {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 20px 24px;
  background: var(--app-bg);
  overflow: hidden;
}

.history-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}

.history-header h3 {
  margin: 0 0 4px;
  font-size: 16px;
  color: var(--app-text);
}

.hint {
  margin: 0;
  font-size: 12px;
  color: var(--app-text-muted);
}

.history-actions {
  flex-shrink: 0;
}

.empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--app-text-muted);
  border: 1px dashed var(--app-border);
  border-radius: 10px;
}

.history-table {
  flex: 1;
  min-height: 0;
  overflow: auto;
  cursor: pointer;
}
</style>
