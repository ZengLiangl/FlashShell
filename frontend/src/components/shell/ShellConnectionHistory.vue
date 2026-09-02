<template>
  <div class="shell-landing">
    <div class="shell-landing-inner">
      <header class="shell-landing-hero">
        <h1 class="shell-landing-title">开始会话</h1>
        <p class="shell-landing-lead">从最近连接快速重连，或打开本机终端 / 机器列表</p>
      </header>

      <div class="app-toolbar shell-landing-toolbar">
        <el-input
          ref="searchInputRef"
          v-model="keyword"
          clearable
          size="large"
          class="app-toolbar-search"
          placeholder="搜索机器名 / 地址，回车连接第一条"
          @keydown.enter.prevent="connectFirst"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <div class="shell-landing-actions">
        <button type="button" class="shell-landing-action" @click="$emit('add-local')">
          <span class="shell-landing-action-icon" aria-hidden="true">
            <el-icon :size="18"><Monitor /></el-icon>
          </span>
          <span class="shell-landing-action-copy">
            <b>本机终端</b>
            <span>打开本地 Shell</span>
          </span>
        </button>
        <button type="button" class="shell-landing-action" @click="$emit('open-picker', 'machines')">
          <span class="shell-landing-action-icon" aria-hidden="true">
            <el-icon :size="18"><Connection /></el-icon>
          </span>
          <span class="shell-landing-action-copy">
            <b>连接机器</b>
            <span>从主机列表选择</span>
          </span>
        </button>
        <button type="button" class="shell-landing-action" @click="$emit('add-machine')">
          <span class="shell-landing-action-icon" aria-hidden="true">
            <el-icon :size="18"><Plus /></el-icon>
          </span>
          <span class="shell-landing-action-copy">
            <b>新增机器</b>
            <span>添加 SSH 主机</span>
          </span>
        </button>
      </div>

      <div v-if="!records.length" class="app-empty shell-landing-empty">
        <p class="app-empty-title">暂无连接历史</p>
        <p class="app-empty-desc">连过的主机会出现在这里，方便下次一键重连</p>
        <el-button type="primary" @click="$emit('open-picker', 'machines')">打开连接</el-button>
      </div>

      <div v-else class="shell-landing-list">
        <ShellHistoryList
          embedded
          :records="records"
          :sessions="sessions"
          :workspace-sessions="workspaceSessions"
          :keyword="keyword"
          @connect="onRowConnect"
          @clear="$emit('clear')"
          @remove="(row) => $emit('remove', row)"
        />
      </div>
    </div>
  </div>
</template>

<script>
import { nextTick, onMounted, ref, watch } from 'vue'
import { Search, Plus, Monitor, Connection } from '@element-plus/icons-vue'
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
  components: { Search, Plus, Monitor, Connection, ShellHistoryList },
  props: {
    records: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
    workspaceSessions: { type: Array, default: () => [] },
    active: { type: Boolean, default: true },
  },
  emits: ['connect', 'open-picker', 'clear', 'remove', 'add-machine', 'add-local'],
  setup(props, { emit }) {
    const keyword = ref('')
    const searchInputRef = ref(null)

    const focusSearch = () => {
      const run = () => {
        const inst = searchInputRef.value
        if (inst && typeof inst.focus === 'function') {
          inst.focus()
          return
        }
        const el = inst?.$el?.querySelector?.('input') || inst?.input
        if (el && typeof el.focus === 'function') el.focus()
      }
      nextTick(() => {
        run()
        requestAnimationFrame(run)
      })
    }

    onMounted(focusSearch)
    watch(
      () => props.active,
      (v) => {
        if (v) focusSearch()
      },
    )

    const onRowConnect = (name) => emit('connect', name)

    const connectFirst = () => {
      const first = filterHistoryRecords(props.records, keyword.value)[0]
      if (first) onRowConnect(first.machineName)
    }

    return {
      keyword,
      searchInputRef,
      onRowConnect,
      connectFirst,
    }
  },
}
</script>

<style scoped>
.shell-landing {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--app-bg);
  overflow: hidden;
}

.shell-landing-inner {
  flex: 1;
  min-height: 0;
  width: 100%;
  max-width: 760px;
  margin: 0 auto;
  padding: 36px 32px 28px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  box-sizing: border-box;
}

.shell-landing-hero {
  flex-shrink: 0;
}

.shell-landing-title {
  margin: 0;
  font-size: 22px;
  font-weight: 650;
  letter-spacing: -0.03em;
  color: var(--app-text);
}

.shell-landing-lead {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.45;
  color: var(--app-text-muted);
}

.shell-landing-toolbar {
  flex-shrink: 0;
}

.shell-landing-toolbar :deep(.el-input__wrapper) {
  min-height: 40px;
  border-radius: var(--app-radius-lg);
}

.shell-landing-actions {
  flex-shrink: 0;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.shell-landing-action {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 64px;
  padding: 10px 12px;
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius-lg);
  background: var(--app-panel-bg);
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition: background 0.12s ease, border-color 0.12s ease;
}

.shell-landing-action:hover {
  border-color: color-mix(in srgb, var(--app-accent-color) 45%, var(--app-border));
  background: color-mix(in srgb, var(--app-accent-color) 8%, var(--app-panel-bg));
}

.shell-landing-action-icon {
  width: 32px;
  height: 32px;
  border-radius: var(--app-radius-md);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--app-accent-color);
  background: color-mix(in srgb, var(--app-accent-color) 14%, transparent);
}

.shell-landing-action-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.shell-landing-action-copy b {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
}

.shell-landing-action-copy span {
  font-size: 12px;
  color: var(--app-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.shell-landing-empty {
  flex: 1;
  min-height: 180px;
}

.shell-landing-list {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.shell-landing-list :deep(.history-list-wrap) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.shell-landing-list :deep(.history-scroll) {
  flex: 1;
}

@media (max-width: 640px) {
  .shell-landing-inner {
    padding: 24px 16px 20px;
  }

  .shell-landing-actions {
    grid-template-columns: 1fr;
  }
}
</style>
