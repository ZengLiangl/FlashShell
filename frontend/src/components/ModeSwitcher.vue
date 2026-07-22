<template>
  <div class="mode-switcher" role="tablist" aria-label="工作模式">
    <button
      v-for="item in items"
      :key="item.id"
      type="button"
      role="tab"
      class="mode-btn"
      :class="[
        `mode-${item.id}`,
        { active: modelValue === item.id, disabled: item.disabled },
      ]"
      :aria-selected="modelValue === item.id"
      :disabled="item.disabled"
      :title="item.title"
      @click="onSelect(item)"
    >
      <el-icon class="mode-icon" :size="13">
        <component :is="item.icon" />
      </el-icon>
      <span class="mode-label">{{ item.label }}</span>
      <span
        v-if="item.badge"
        class="mode-badge"
        aria-hidden="true"
      >{{ item.badge }}</span>
      <span
        v-else-if="item.dot"
        class="mode-dot"
        aria-hidden="true"
      />
    </button>
  </div>
</template>

<script>
import { computed } from 'vue'
import { HomeFilled, Folder, Monitor } from '@element-plus/icons-vue'

export default {
  name: 'ModeSwitcher',
  components: { HomeFilled, Folder, Monitor },
  props: {
    modelValue: {
      type: String,
      default: 'home',
      validator: (v) => ['home', 'task', 'shell'].includes(v),
    },
    hasProjects: { type: Boolean, default: false },
    hasMachines: { type: Boolean, default: false },
    hasTask: { type: Boolean, default: false },
    taskRunning: { type: Boolean, default: false },
    connectedCount: { type: Number, default: 0 },
  },
  emits: ['update:modelValue', 'change'],
  setup(props, { emit }) {
    const items = computed(() => {
      const list = [
        {
          id: 'home',
          label: '首页',
          icon: HomeFilled,
          title: '返回首页',
          disabled: false,
        },
      ]
      if (props.hasProjects || props.hasTask || props.modelValue === 'task') {
        list.push({
          id: 'task',
          label: '任务',
          icon: Folder,
          title: props.hasTask
            ? (props.taskRunning ? '返回任务（运行中）' : '返回任务')
            : '请先在首页选择项目',
          disabled: !props.hasTask,
          dot: props.hasTask && props.taskRunning,
        })
      }
      if (props.hasMachines || props.connectedCount > 0 || props.modelValue === 'shell') {
        list.push({
          id: 'shell',
          label: 'Shell',
          icon: Monitor,
          title: props.connectedCount > 0
            ? `进入 Shell（${props.connectedCount} 会话）`
            : '进入 Shell',
          disabled: false,
          badge: props.connectedCount > 0 ? String(props.connectedCount) : '',
        })
      }
      return list
    })

    const onSelect = (item) => {
      if (item.disabled || item.id === props.modelValue) return
      emit('update:modelValue', item.id)
      emit('change', item.id)
    }

    return { items, onSelect }
  },
}
</script>

<style scoped>
.mode-switcher {
  --mode-track: color-mix(in srgb, var(--app-text) 6%, var(--app-panel-bg));
  --mode-home: var(--app-text);
  --mode-task: var(--app-accent-color);
  --mode-shell: var(--app-accent-color);

  display: inline-grid;
  grid-auto-flow: column;
  grid-auto-columns: 1fr;
  align-items: stretch;
  gap: 2px;
  min-width: min(280px, 42vw);
  padding: 3px;
  border-radius: 9px;
  background: var(--mode-track);
  border: 1px solid color-mix(in srgb, var(--app-border) 80%, transparent);
  box-shadow: inset 0 1px 0 color-mix(in srgb, #fff 35%, transparent);
}

.mode-btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  min-width: 0;
  height: 26px;
  padding: 0 12px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--app-text-muted, #909399);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.01em;
  line-height: 1;
  cursor: pointer;
  transition:
    background 0.16s ease,
    color 0.16s ease,
    box-shadow 0.16s ease,
    transform 0.16s ease;
}

.mode-icon {
  flex-shrink: 0;
  opacity: 0.85;
}

.mode-label {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mode-btn:hover:not(.disabled):not(.active) {
  color: var(--app-accent-color);
  background: var(--app-accent-bg);
}

.mode-btn.active {
  background: var(--app-card-bg, #fff);
  color: var(--app-text);
  font-weight: 600;
  box-shadow:
    0 1px 2px color-mix(in srgb, var(--app-text) 8%, transparent),
    0 0 0 1px color-mix(in srgb, var(--app-border) 75%, transparent);
}

.mode-btn.mode-task.active,
.mode-btn.mode-shell.active {
  color: var(--app-accent-color);
  background: var(--app-card-bg, #fff);
}

.mode-btn.active .mode-icon {
  opacity: 1;
}

.mode-btn.disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.mode-badge {
  min-width: 15px;
  height: 15px;
  padding: 0 4px;
  border-radius: 8px;
  font-size: 10px;
  font-weight: 700;
  line-height: 15px;
  text-align: center;
  color: var(--mode-shell);
  background: color-mix(in srgb, var(--mode-shell) 16%, transparent);
}

.mode-btn.active .mode-badge {
  background: color-mix(in srgb, var(--mode-shell) 20%, transparent);
}

.mode-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
  background: var(--mode-task);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--mode-task) 24%, transparent);
}

html.dark .mode-switcher {
  --mode-track: color-mix(in srgb, #000 28%, var(--app-panel-bg));
  box-shadow: inset 0 1px 0 color-mix(in srgb, #fff 4%, transparent);
}

html.dark .mode-btn.active {
  background: color-mix(in srgb, var(--app-card-bg) 92%, #fff);
  box-shadow:
    0 1px 3px rgba(0, 0, 0, 0.35),
    0 0 0 1px color-mix(in srgb, var(--app-border) 80%, transparent);
}
</style>
