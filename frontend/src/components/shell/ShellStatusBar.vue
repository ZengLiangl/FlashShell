<template>
  <div class="shell-status-bar">
    <div class="status-left">
      <el-tag v-if="activeTabLabel" size="small" type="info" class="session-tag">{{ activeTabLabel }}</el-tag>
      <span v-if="showToolbar && activeTabLabel" class="status-left-sep" aria-hidden="true" />
      <ShellActionToolbar
        v-if="showToolbar"
        :connected-count="connectedCount"
        :broadcast-enabled="broadcastEnabled"
        :compose-enabled="composeEnabled"
        :show-left-panel="showLeftPanel"
        :left-panel-open="leftPanelOpen"
        :left-panel-label="leftPanelLabel"
        :file-panel-expanded="filesExpanded"
        :active-is-local="activeIsLocal"
        :tunnel-dialog-visible="tunnelDialogVisible"
        :has-active-tunnel="hasActiveTunnel"
        :transfer-active-count="transferActiveCount"
        :search-visible="searchVisible"
        :transfer-visible="transferVisible"
        :command-palette-visible="commandPaletteVisible"
        @toggle-broadcast="$emit('toggle-broadcast')"
        @toggle-compose="$emit('toggle-compose')"
        @toggle-left-panel="$emit('toggle-left-panel')"
        @toggle-files="$emit('toggle-files')"
        @open-tunnels="$emit('open-tunnels')"
        @toggle-search="$emit('toggle-search')"
        @open-command-palette="$emit('open-command-palette')"
        @open-transfer="$emit('open-transfer')"
        @clear="$emit('clear')"
      />
    </div>

    <div class="status-right">
      <span class="app-info">{{ appInfo }}</span>
    </div>
  </div>
</template>

<script>
import { computed } from 'vue'
import ShellActionToolbar from './ShellActionToolbar.vue'

export default {
  name: 'ShellStatusBar',
  components: { ShellActionToolbar },
  emits: [
    'open-tunnels',
    'toggle-files',
    'toggle-search',
    'clear',
    'toggle-broadcast',
    'toggle-compose',
    'toggle-left-panel',
    'open-command-palette',
    'open-transfer',
  ],
  props: {
    connectedCount: { type: Number, default: 0 },
    activeTabLabel: { type: String, default: '' },
    tunnels: { type: Array, default: () => [] },
    appInfo: { type: String, default: 'FlashShell · Shell' },
    showToolbar: { type: Boolean, default: false },
    broadcastEnabled: { type: Boolean, default: false },
    composeEnabled: { type: Boolean, default: false },
    showLeftPanel: { type: Boolean, default: false },
    leftPanelOpen: { type: Boolean, default: false },
    leftPanelLabel: { type: String, default: '监控' },
    filesExpanded: { type: Boolean, default: false },
    activeIsLocal: { type: Boolean, default: false },
    activeConnected: { type: Boolean, default: false },
    transferActiveCount: { type: Number, default: 0 },
    searchVisible: { type: Boolean, default: false },
    transferVisible: { type: Boolean, default: false },
    commandPaletteVisible: { type: Boolean, default: false },
    tunnelDialogVisible: { type: Boolean, default: false },
  },
  setup(props) {
    const hasActiveTunnel = computed(() => (props.tunnels || []).some((t) => t.active))

    return { hasActiveTunnel }
  },
}
</script>

<style scoped>
.shell-status-bar {
  flex-shrink: 0;
  min-height: 36px;
  height: auto;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 12px 4px 14px;
  border-top: 1px solid var(--shell-chrome-border);
  background: var(--shell-chrome-bg-2);
  color: var(--shell-chrome-muted);
  width: 100%;
  gap: 12px;
  overflow: hidden;
  font-size: 12px;
}

.status-left {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
  min-height: 28px;
}

.status-left-sep {
  width: 1px;
  height: 16px;
  background: var(--shell-chrome-border);
  flex-shrink: 0;
}

.status-left :deep(.session-tag.el-tag) {
  --el-tag-bg-color: var(--surface, var(--app-panel-bg));
  --el-tag-border-color: var(--shell-chrome-border, var(--app-border));
  --el-tag-text-color: var(--shell-chrome-fg, var(--app-text-secondary));
  box-sizing: border-box;
  height: 28px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 500;
  line-height: 26px;
  border-radius: 8px;
  flex-shrink: 0;
}

.status-right {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-shrink: 0;
  height: 100%;
}

.app-info {
  font-size: 12px;
  color: var(--shell-chrome-muted);
  flex-shrink: 0;
}
</style>
