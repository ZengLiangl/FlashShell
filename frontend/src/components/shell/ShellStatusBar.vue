<template>
  <div class="shell-status-bar">
    <div class="status-info">
      <el-tag v-if="activeTabLabel" size="small" type="info">{{ activeTabLabel }}</el-tag>
      <template v-if="tunnels.length">
        <span
          v-for="(t, i) in tunnels"
          :key="`${t.name}-${t.localPort}-${i}`"
          class="tunnel-chip clickable"
          :class="{ active: t.active, error: !!t.error }"
          :title="tunnelTitle(t)"
          @click="$emit('open-tunnels')"
        >
          {{ tunnelShortLabel(t) }}
        </span>
      </template>
      <span v-else-if="tunnelLoading" class="tunnel-loading">隧道…</span>
    </div>
    <div class="status-right">
      <span class="app-info">{{ appInfo }}</span>
      <div v-if="showChromeActions" class="chrome-actions">
        <el-tooltip content="文件" placement="top">
          <button
            type="button"
            class="chrome-icon-btn"
            :class="{ 'is-active': filesExpanded }"
            @click="$emit('toggle-files')"
          >
            <el-icon :size="14"><FolderOpened /></el-icon>
          </button>
        </el-tooltip>
        <el-tooltip content="搜索" placement="top">
          <button type="button" class="chrome-icon-btn" @click="$emit('toggle-search')">
            <el-icon :size="14"><Search /></el-icon>
          </button>
        </el-tooltip>
        <el-tooltip content="清空" placement="top">
          <button type="button" class="chrome-icon-btn" @click="$emit('clear')">
            <el-icon :size="14"><Delete /></el-icon>
          </button>
        </el-tooltip>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'ShellStatusBar',
  emits: ['open-tunnels', 'toggle-files', 'toggle-search', 'clear'],
  props: {
    connectedCount: { type: Number, default: 0 },
    activeMachine: { type: String, default: '' },
    activeTabLabel: { type: String, default: '' },
    tunnels: { type: Array, default: () => [] },
    tunnelLoading: { type: Boolean, default: false },
    appInfo: { type: String, default: 'FlashShell · Shell' },
    showChromeActions: { type: Boolean, default: false },
    filesExpanded: { type: Boolean, default: false },
  },
  setup() {
    const typeLabel = (type) => {
      if (type === 'remote') return '远程'
      if (type === 'dynamic') return 'SOCKS'
      return '本地'
    }

    const tunnelShortLabel = (t) => {
      const name = t.name || typeLabel(t.type)
      if (t.type === 'dynamic') {
        return `${name}:${t.localPort}${t.active ? '' : ' ✕'}`
      }
      const remote = t.remoteHost ? `${t.remoteHost}:${t.remotePort}` : ''
      return `${name} ${t.localPort}→${remote}${t.active ? '' : ' ✕'}`
    }

    const tunnelTitle = (t) => {
      if (t.error) return t.error
      if (t.active) return '隧道运行中'
      return '隧道未建立'
    }

    return { tunnelShortLabel, tunnelTitle }
  },
}
</script>

<style scoped>
.shell-status-bar {
  flex-shrink: 0;
  height: 28px;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 8px 0 12px;
  border-top: 1px solid var(--shell-chrome-border, var(--app-border));
  background: var(--shell-chrome-bg, var(--app-panel-bg));
  color: var(--app-text);
  width: 100%;
  gap: 12px;
  overflow: hidden;
  box-shadow: inset 0 1px 0 var(--shell-chrome-highlight, transparent);
}

.status-info {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1;
  overflow-x: auto;
  height: 100%;
}

.status-info :deep(.el-tag) {
  --el-tag-bg-color: color-mix(in srgb, var(--app-card-bg) 65%, transparent);
  --el-tag-border-color: var(--shell-chrome-divider, var(--app-border));
  --el-tag-text-color: var(--app-text-muted);
  height: 18px;
  padding: 0 7px;
  font-weight: 500;
  line-height: 16px;
}

.status-right {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  height: 100%;
}

.chrome-actions {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
  height: 22px;
}

.chrome-icon-btn {
  box-sizing: border-box;
  width: 22px;
  height: 22px;
  padding: 0;
  margin: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--app-text-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  line-height: 1;
  flex-shrink: 0;
}

.chrome-icon-btn:hover {
  color: var(--app-accent-color, #409eff);
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 12%, transparent);
}

.chrome-icon-btn.is-active {
  color: var(--app-accent-color, #409eff);
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 16%, transparent);
}

.tunnel-chip {
  flex-shrink: 0;
  font-size: 11px;
  padding: 1px 7px;
  border-radius: 999px;
  border: 1px solid var(--shell-chrome-divider, var(--app-border));
  color: var(--app-text-muted);
  background: color-mix(in srgb, var(--app-card-bg) 70%, transparent);
}

.tunnel-chip.active {
  border-color: color-mix(in srgb, var(--app-success-color, #67c23a) 50%, var(--app-border));
  color: var(--app-success-color, #67c23a);
}

.tunnel-chip.error {
  border-color: color-mix(in srgb, var(--app-danger-color, #f56c6c) 50%, var(--app-border));
  color: var(--app-danger-color, #f56c6c);
}

.tunnel-loading {
  font-size: 11px;
  color: var(--app-text-muted);
}

.tunnel-chip.clickable {
  cursor: pointer;
}
.tunnel-chip.clickable:hover {
  opacity: 0.85;
}

.app-info {
  font-size: 12px;
  color: var(--app-text-muted);
  flex-shrink: 0;
}
</style>
