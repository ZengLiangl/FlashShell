<template>
  <div class="status-bar task-status-style">
    <span class="st-label">
      <StatusDot :state="status.isRunning ? 'on' : 'off'" />
      <span class="status-text">
        <template v-if="status.isRunning">
          执行中<template v-if="status.currentCommand"> · {{ status.currentCommand }}</template>
        </template>
        <template v-else>就绪</template>
      </span>
    </span>

    <AppProgress v-if="status.isRunning" :value="progressValue" />

    <span v-if="stepProgressLabel" class="st-label">
      {{ stepProgressLabel }}
    </span>

    <span v-if="selectedProject && !status.isRunning" class="st-label project-tag">
      项目: {{ selectedProject.name }}
    </span>

    <div class="tb-spacer" />

    <div class="status-actions">
      <AppIconBtn
        v-if="remoteFailure && !status.isRunning"
        title="进入失败机器 Shell"
        @click="$emit('open-failure-shell')"
      >
        <el-icon :size="14"><Monitor /></el-icon>
      </AppIconBtn>

      <AppButton
        v-if="status.isRunning"
        variant="danger"
        size="sm"
        @click="$emit('stop-all')"
      >
        <el-icon :size="12"><VideoPause /></el-icon>
        停止
      </AppButton>

      <span v-if="appInfo" class="app-info">{{ appInfo }}</span>
    </div>
  </div>
</template>

<script>
import { computed } from 'vue'
import { AppIconBtn, AppButton, AppProgress, StatusDot } from './ui'
import { calcTaskProgressPercentage, formatTaskStepProgress } from '../utils/taskProgress'

export default {
  name: 'StatusBar',
  components: { AppIconBtn, AppButton, AppProgress, StatusDot },
  props: {
    status: { type: Object, required: true },
    selectedProject: { type: Object, default: null },
    remoteFailure: { type: Object, default: null },
    appInfo: { type: String, default: '' },
    progressPercentage: { type: Number, default: 0 },
  },
  emits: ['stop-all', 'open-failure-shell'],
  setup(props) {
    const stepProgressLabel = computed(() => {
      if (!props.status?.isRunning) return ''
      return formatTaskStepProgress(props.status)
    })
    const progressValue = computed(() => {
      if (!props.status?.isRunning) return 0
      if (props.progressPercentage > 0) return props.progressPercentage
      return calcTaskProgressPercentage(props.status)
    })
    return { progressValue, stepProgressLabel }
  },
}
</script>

<style scoped>
.tb-spacer {
  flex: 1;
}

.st-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.status-text {
  max-width: 240px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.project-tag {
  color: var(--muted);
  font-size: 12px;
}

.status-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.app-info {
  color: var(--muted);
  font-size: 12px;
}

@media (max-width: 720px) {
  .project-tag,
  .app-info {
    display: none;
  }
}
</style>
