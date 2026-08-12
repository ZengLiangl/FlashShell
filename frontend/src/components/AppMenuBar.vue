<template>
  <div class="app-menu-bar">
    <div class="menu-side menu-left" aria-hidden="true" />

    <div class="menu-center" />

    <div class="menu-side menu-right">
      <ModeSwitcher
        v-if="activeView === 'task' && (hasProjects || hasTask)"
        compact
        float-align="end"
        :model-value="activeView"
        :has-projects="hasProjects"
        :has-machines="hasMachines"
        :has-task="hasTask"
        :task-running="taskRunning"
        :connected-count="connectedCount"
        :projects="projects"
        :selected-project-name="selectedProjectName"
        :sessions="sessions"
        :active-session-id="activeSessionId"
        @change="$emit('change-view', $event)"
        @select-project="$emit('select-project', $event)"
        @focus-session="$emit('focus-session', $event)"
      />
      <AppChromeIcons />
    </div>
  </div>
</template>

<script>
import ModeSwitcher from './ModeSwitcher.vue'
import AppChromeIcons from './AppChromeIcons.vue'

export default {
  name: 'AppMenuBar',
  components: { ModeSwitcher, AppChromeIcons },
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
    sessions: { type: Array, default: () => [] },
    activeSessionId: { type: String, default: '' },
  },
  emits: ['change-view', 'select-project', 'focus-session'],
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
