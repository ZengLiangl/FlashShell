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
          <el-tooltip content="连接" placement="top">
            <el-button size="small" circle @click="$emit('open-picker')">
              <el-icon><Monitor /></el-icon>
            </el-button>
          </el-tooltip>
        </div>
      </div>

      <div v-if="!records.length" class="app-empty">
        <p class="app-empty-title">暂无连接历史</p>
        <p class="app-empty-desc">从机器列表选择一台开始 SSH 会话</p>
        <el-button type="primary" size="small" @click="$emit('open-picker', 'machines')">打开连接</el-button>
      </div>

      <ShellHistoryList
        v-else
        :records="records"
        :sessions="sessions"
        :keyword="keyword"
        @connect="onRowConnect"
        @clear="$emit('clear')"
        @remove="(row) => $emit('remove', row)"
      />
    </div>
  </div>
</template>

<script>
import { computed, ref } from 'vue'
import { Search, ArrowLeft, Monitor } from '@element-plus/icons-vue'
import ShellHistoryList from './ShellHistoryList.vue'

function filterHistoryRecords(records, keyword) {
  const kw = String(keyword || '').trim().toLowerCase()
  const list = records || []
  if (!kw) return list
  return list.filter((row) => {
    const hay = `${row.machineName || ''} ${row.user || ''} ${row.host || ''} ${row.port || ''}`.toLowerCase()
    return hay.includes(kw)
  })
}

export default {
  name: 'ShellConnectionHistory',
  components: { Search, ArrowLeft, Monitor, ShellHistoryList },
  props: {
    records: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
  },
  emits: ['connect', 'open-picker', 'clear', 'remove', 'back'],
  setup(props, { emit }) {
    const keyword = ref('')

    const onRowConnect = (name) => emit('connect', name)

    const connectFirst = () => {
      const first = filterHistoryRecords(props.records, keyword.value)[0]
      if (first) onRowConnect(first.machineName)
    }

    return {
      keyword,
      onRowConnect,
      connectFirst,
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
  align-items: flex-start;
  padding: 28px 24px;
  background: var(--app-bg);
  overflow: auto;
}

.shell-history .app-surface-narrow {
  margin-top: min(8vh, 56px);
  padding: 18px 16px 16px;
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius-panel, 14px);
  background: var(--app-panel-bg);
  box-shadow: var(--app-surface-shadow, none);
  box-sizing: border-box;
}
</style>
