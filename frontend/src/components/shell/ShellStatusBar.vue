<template>
  <div class="shell-status-bar">
    <div class="status-info">
      <el-tag v-if="connectedCount > 0" type="success" size="small">
        {{ connectedCount }} 个会话
      </el-tag>
      <el-tag v-else type="info" size="small">无活动会话</el-tag>
      <el-tag v-if="activeMachine" size="small" type="info">{{ activeMachine }}</el-tag>
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
    appInfo: { type: String, default: 'Quick Cmd · Shell' },
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
}

.status-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.app-info {
  font-size: 12px;
  color: var(--app-text-muted);
}
</style>
