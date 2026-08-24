<template>
  <div class="shell-action-toolbar" role="toolbar" aria-label="终端工具">
    <div class="tool-group tool-group-primary">
      <div class="tool-item">
        <el-tooltip content="查找" placement="top" :show-after="280">
          <button
            type="button"
            class="tool-btn"
            :class="{ active: searchVisible }"
            @click="$emit('toggle-search')"
          >
            <el-icon class="tool-icon"><Search /></el-icon>
          </button>
        </el-tooltip>
      </div>

      <div v-if="showLeftPanel" class="tool-item">
        <el-tooltip
          :content="leftPanelOpen ? `收起${leftPanelLabel}` : `展开${leftPanelLabel}`"
          placement="top"
          :show-after="280"
        >
          <button
            type="button"
            class="tool-btn"
            :class="{ active: leftPanelOpen }"
            @click="$emit('toggle-left-panel')"
          >
            <el-icon class="tool-icon">
              <Folder v-if="leftPanelLabel === '文件'" />
              <TrendCharts v-else />
            </el-icon>
          </button>
        </el-tooltip>
      </div>

      <template v-if="!activeIsLocal">
        <div class="tool-item">
          <el-tooltip content="SFTP 文件面板" placement="top" :show-after="280">
            <button
              type="button"
              class="tool-btn"
              :class="{ active: filePanelExpanded }"
              @click="$emit('toggle-files')"
            >
              <el-icon class="tool-icon"><FolderOpened /></el-icon>
            </button>
          </el-tooltip>
        </div>

        <div class="tool-item">
          <el-tooltip :content="tunnelTooltip" placement="top" :show-after="280">
            <button
              type="button"
              class="tool-btn"
              :class="{ active: tunnelOpen }"
              @click="$emit('open-tunnels')"
            >
              <el-icon class="tool-icon"><Connection /></el-icon>
            </button>
          </el-tooltip>
        </div>

        <div class="tool-item">
          <el-tooltip content="文件传输" placement="top" :show-after="280">
            <button
              type="button"
              class="tool-btn tool-btn-badge"
              :class="{ active: transferVisible }"
              @click="$emit('open-transfer')"
            >
              <el-badge :value="transferActiveCount" :hidden="!transferActiveCount" :max="99">
                <el-icon class="tool-icon"><Upload /></el-icon>
              </el-badge>
            </button>
          </el-tooltip>
        </div>
      </template>
    </div>

    <span class="tool-divider" aria-hidden="true" />

    <div class="tool-group tool-group-secondary">
      <div class="tool-item">
        <el-tooltip
          :content="composeEnabled ? '关闭撰写栏' : '开启撰写栏（多行命令）'"
          placement="top"
          :show-after="280"
        >
          <button
            type="button"
            class="tool-btn"
            :class="{ active: composeEnabled }"
            @click="$emit('toggle-compose')"
          >
            <el-icon class="tool-icon"><EditPen /></el-icon>
          </button>
        </el-tooltip>
      </div>

      <div class="tool-item">
        <el-tooltip content="命令面板 (Ctrl/⌘+Shift+P)" placement="top" :show-after="280">
          <button
            type="button"
            class="tool-btn"
            :class="{ active: commandPaletteVisible }"
            @click="$emit('open-command-palette')"
          >
            <el-icon class="tool-icon"><Memo /></el-icon>
          </button>
        </el-tooltip>
      </div>

      <div v-if="connectedCount >= 1" class="tool-item">
        <el-tooltip
          :content="broadcastEnabled ? '关闭命令广播 (Esc)' : '开启命令广播'"
          placement="top"
          :show-after="280"
        >
          <button
            type="button"
            class="tool-btn"
            :class="{ active: broadcastEnabled }"
            @click="$emit('toggle-broadcast')"
          >
            <el-icon class="tool-icon"><Promotion /></el-icon>
          </button>
        </el-tooltip>
      </div>

      <div class="tool-item">
        <el-tooltip content="清空终端" placement="top" :show-after="280">
          <button type="button" class="tool-btn tool-btn-danger" @click="$emit('clear')">
            <el-icon class="tool-icon"><Delete /></el-icon>
          </button>
        </el-tooltip>
      </div>
    </div>
  </div>
</template>

<script>
import { computed } from 'vue'
import {
  Promotion,
  EditPen,
  TrendCharts,
  FolderOpened,
  Connection,
  Search,
  Memo,
  Upload,
  Delete,
  Folder,
} from '@element-plus/icons-vue'

export default {
  name: 'ShellActionToolbar',
  components: {
    Promotion,
    EditPen,
    TrendCharts,
    FolderOpened,
    Connection,
    Search,
    Memo,
    Upload,
    Delete,
    Folder,
  },
  props: {
    connectedCount: { type: Number, default: 0 },
    broadcastEnabled: { type: Boolean, default: false },
    composeEnabled: { type: Boolean, default: false },
    showLeftPanel: { type: Boolean, default: false },
    leftPanelOpen: { type: Boolean, default: false },
    leftPanelLabel: { type: String, default: '监控' },
    filePanelExpanded: { type: Boolean, default: false },
    activeIsLocal: { type: Boolean, default: false },
    tunnelDialogVisible: { type: Boolean, default: false },
    hasActiveTunnel: { type: Boolean, default: false },
    transferActiveCount: { type: Number, default: 0 },
    searchVisible: { type: Boolean, default: false },
    transferVisible: { type: Boolean, default: false },
    commandPaletteVisible: { type: Boolean, default: false },
  },
  emits: [
    'toggle-broadcast',
    'toggle-compose',
    'toggle-left-panel',
    'toggle-files',
    'open-tunnels',
    'toggle-search',
    'open-command-palette',
    'open-transfer',
    'clear',
  ],
  setup(props) {
    const tunnelOpen = computed(() => props.tunnelDialogVisible || props.hasActiveTunnel)
    const tunnelTooltip = computed(() => (
      props.hasActiveTunnel ? 'SSH 隧道（已连接）' : 'SSH 隧道'
    ))
    return { tunnelOpen, tunnelTooltip }
  },
}
</script>

<style scoped>
.shell-action-toolbar {
  --tool-size: 28px;
  --tool-icon: 15px;
  display: inline-flex;
  align-items: center;
  gap: 0;
  flex-shrink: 0;
  height: var(--tool-size);
  padding: 0 4px;
  border-radius: 8px;
  background: var(--surface, var(--app-panel-bg));
  border: 1px solid var(--shell-chrome-border);
  box-sizing: border-box;
}

.tool-group {
  display: inline-flex;
  align-items: center;
  height: 100%;
}

.tool-item {
  width: var(--tool-size);
  height: var(--tool-size);
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tool-item :deep(.el-tooltip__trigger) {
  display: flex;
  align-items: center;
  justify-content: center;
  width: var(--tool-size);
  height: var(--tool-size);
}

.tool-btn {
  position: relative;
  box-sizing: border-box;
  width: var(--tool-size);
  height: var(--tool-size);
  margin: 0;
  padding: 0;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--shell-chrome-muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: color 0.12s ease, background 0.12s ease;
}

.tool-icon {
  width: var(--tool-icon);
  height: var(--tool-icon);
  font-size: var(--tool-icon);
  display: block;
}

.tool-btn:hover {
  color: var(--shell-chrome-fg);
  background: var(--shell-chrome-hover);
}

.tool-btn.active {
  color: var(--accent);
  background: transparent;
}

.tool-btn.active::after {
  content: '';
  position: absolute;
  left: 6px;
  right: 6px;
  bottom: 3px;
  height: 2px;
  border-radius: 2px;
  background: var(--accent);
  pointer-events: none;
}

.tool-divider {
  width: 1px;
  height: 16px;
  margin: 0 2px;
  background: var(--shell-chrome-border);
  flex-shrink: 0;
}

.tool-btn-badge :deep(.el-badge) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
}

.tool-btn-badge :deep(.el-badge__content) {
  top: 2px;
  right: 4px;
  transform: scale(0.82);
  border: none;
}

.tool-btn-danger:hover {
  color: var(--danger, var(--app-danger-color, #f56c6c));
  background: var(--danger-soft, color-mix(in srgb, #f56c6c 12%, transparent));
}
</style>
