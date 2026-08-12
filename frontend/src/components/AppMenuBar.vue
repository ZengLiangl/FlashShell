<template>
  <div class="app-menu-bar">
    <div class="menu-side menu-left" aria-hidden="true" />

    <div class="menu-center">
      <!-- 首页已有左侧「任务/主机」分区，顶栏 ModeSwitcher 暂隐藏 -->
      <!--
      <ModeSwitcher
        v-if="showFullSwitcher"
        :model-value="activeView"
        :has-projects="hasProjects"
        :has-machines="hasMachines"
        :has-task="hasTask"
        :task-running="taskRunning"
        :connected-count="connectedCount"
        :projects="projects"
        :selected-project-name="selectedProjectName"
        @change="$emit('change-view', $event)"
        @select-project="$emit('select-project', $event)"
      />
      -->
    </div>

    <div class="menu-side menu-right">
      <AppChromeIcons />
    </div>
  </div>
</template>

<script>
import { computed } from 'vue'
// import ModeSwitcher from './ModeSwitcher.vue'
import AppChromeIcons from './AppChromeIcons.vue'

export default {
  name: 'AppMenuBar',
  components: { AppChromeIcons },
  // components: { ModeSwitcher, AppChromeIcons },
  props: {
    activeView: {
      type: String,
      default: 'home',
      validator: (v) => ['home', 'task', 'shell'].includes(v),
    },
    hasProjects: { type: Boolean, default: false },
    hasMachines: { type: Boolean, default: false },
    hasTask: { type: Boolean, default: false },
    taskRunning: { type: Boolean, default: false },
    connectedCount: { type: Number, default: 0 },
    projects: { type: Array, default: () => [] },
    selectedProjectName: { type: String, default: '' },
  },
  emits: ['change-view', 'select-project'],
  setup(props) {
    const canSwitchModes = computed(() =>
      props.hasProjects || props.hasMachines || props.hasTask || props.connectedCount > 0
        || props.activeView === 'task' || props.activeView === 'shell'
    )

    const showFullSwitcher = computed(() => canSwitchModes.value)

    return { showFullSwitcher }
  },
}
</script>

<style scoped>
.app-menu-bar {
  position: relative;
  flex-shrink: 0;
  border-bottom: 1px solid var(--app-border);
  background: var(--app-panel-bg);
  color: var(--app-text);
  height: 36px;
  display: grid;
  grid-template-columns: minmax(96px, 1fr) auto minmax(96px, 1fr);
  align-items: center;
  padding: 0 8px;
  gap: 8px;
  z-index: 30;
  box-sizing: border-box;
}

.menu-side {
  display: flex;
  align-items: center;
  min-width: 0;
}

.menu-left {
  justify-content: flex-start;
}

.menu-right {
  justify-content: flex-end;
  gap: 6px;
}

.menu-center {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
}
</style>
