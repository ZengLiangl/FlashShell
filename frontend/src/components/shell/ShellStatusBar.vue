<template>
  <div class="shell-status-bar">
    <div class="status-info">
      <el-tag v-if="activeTabLabel" size="small" type="info">{{ activeTabLabel }}</el-tag>
      <template v-if="tunnels.length">
        <span
          v-for="(t, i) in tunnels"
          :key="`${t.name}-${t.localPort}-${i}`"
          class="tunnel-chip"
          :class="{ active: t.active, error: !!t.error }"
          :title="tunnelTitle(t)"
        >
          {{ tunnelShortLabel(t) }}
        </span>
      </template>
      <span v-else-if="tunnelLoading" class="tunnel-loading">隧道…</span>
    </div>
    <span class="app-info">{{ appInfo }}</span>
  </div>
</template>

<script>
export default {
  name: 'ShellStatusBar',
  props: {
    connectedCount: { type: Number, default: 0 },
    activeMachine: { type: String, default: '' },
    activeTabLabel: { type: String, default: '' },
    tunnels: { type: Array, default: () => [] },
    tunnelLoading: { type: Boolean, default: false },
    appInfo: { type: String, default: 'FlashDock · Shell' },
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
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  border-top: 1px solid var(--app-border);
  background: var(--app-panel-bg);
  color: var(--app-text);
  width: 100%;
  gap: 12px;
}

.status-info {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1;
  overflow-x: auto;
}

.tunnel-chip {
  flex-shrink: 0;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
  border: 1px solid var(--app-border);
  color: var(--app-text-muted);
  background: var(--app-card-bg);
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

.app-info {
  font-size: 12px;
  color: var(--app-text-muted);
  flex-shrink: 0;
}
</style>
