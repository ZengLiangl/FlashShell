<template>
  <div class="task-flow-canvas">
    <div v-if="!subProject" class="canvas-empty">
      <el-empty :description="emptyDescription" :image-size="72" />
    </div>
    <template v-else>
      <div class="canvas-header">
        <div class="canvas-title">
          <span class="title-text">{{ subProject.name || '(未命名子项目)' }}</span>
          <span class="title-hint">命令按从左到右顺序执行</span>
        </div>
        <div class="canvas-actions icon-actions">
          <el-dropdown trigger="click" @command="onAddCommand">
            <el-button type="primary" size="small" circle title="添加命令">
              <el-icon><Plus /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="batch">本机构建</el-dropdown-item>
                <el-dropdown-item command="remote">远程部署</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>

      <div class="flow-scroll">
        <div class="flow-row">
          <template v-for="(cmd, cIndex) in commands" :key="cIndex">
            <div v-if="cIndex > 0" class="flow-connector">
              <div class="connector-line" />
              <button
                type="button"
                class="insert-btn"
                title="在此插入命令"
                @click="$emit('insert-command', cIndex)"
              >
                <el-icon :size="12"><Plus /></el-icon>
              </button>
              <div class="connector-line" />
            </div>

            <div class="stage-col">
              <div class="stage-label">阶段 {{ cIndex + 1 }}</div>
              <div
                class="stage-card"
                :class="{ selected: isCommandSelected(cIndex) }"
                @click="$emit('select-command', cIndex)"
              >
                <div class="stage-card-head">
                  <span class="stage-name">{{ cmd.name || '(未命名)' }}</span>
                  <el-tag
                    size="small"
                    :type="cmd.type === 'remote' ? 'warning' : 'success'"
                    effect="light"
                  >
                    {{ commandTypeLabel(cmd.type) }}
                  </el-tag>
                </div>
                <div v-if="cmd.type === 'remote'" class="stage-meta">
                  {{ cmd.machine || '未选机器' }}
                </div>
                <div class="stage-card-actions icon-actions">
                  <el-tooltip content="删除命令" placement="top">
                    <el-button
                      type="danger"
                      text
                      size="small"
                      circle
                      @click.stop="$emit('remove-command', cIndex)"
                    >
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </el-tooltip>
                </div>
              </div>

              <div class="steps-stack">
                <div
                  v-for="(step, stIndex) in cmd.steps || []"
                  :key="stIndex"
                  class="step-pill"
                  :class="{ selected: isStepSelected(cIndex, stIndex) }"
                  @click.stop="$emit('select-step', cIndex, stIndex)"
                >
                  <span class="step-index">{{ stIndex + 1 }}</span>
                  <span class="step-text" :title="step.command">{{ stepSummary(step) }}</span>
                  <button
                    type="button"
                    class="step-del"
                    title="删除步骤"
                    @click.stop="$emit('remove-step', cIndex, stIndex)"
                  >
                    <el-icon :size="12"><Delete /></el-icon>
                  </button>
                </div>
                <el-tooltip content="添加步骤" placement="top">
                  <button
                    type="button"
                    class="add-step-btn"
                    @click.stop="$emit('add-step', cIndex)"
                  >
                    <el-icon :size="14"><Plus /></el-icon>
                  </button>
                </el-tooltip>
              </div>
            </div>
          </template>

          <div v-if="commands.length === 0" class="flow-empty-hint">
            点击右上角 + 添加命令，选中后在下方编辑属性
          </div>

          <div v-if="commands.length > 0" class="flow-tail">
            <button
              type="button"
              class="insert-btn tail"
              title="追加命令"
              @click="$emit('insert-command', commands.length)"
            >
              <el-icon :size="14"><Plus /></el-icon>
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script>
import { computed } from 'vue'
import { Delete, Plus } from '@element-plus/icons-vue'
import { commandTypeLabel, stepSummary } from './taskFlowModel'

export default {
  name: 'TaskFlowCanvas',
  components: { Delete, Plus },
  props: {
    subProject: { type: Object, default: null },
    selectedPath: { type: Object, default: null },
    emptyDescription: { type: String, default: '请在左侧选择子项目以编辑流水线' },
  },
  emits: [
    'select-command',
    'select-step',
    'add-command',
    'insert-command',
    'remove-command',
    'add-step',
    'remove-step',
  ],
  setup(props, { emit }) {
    const commands = computed(() => props.subProject?.commands || [])

    const isCommandSelected = (cIndex) => {
      const sp = props.selectedPath
      return sp && sp.c === cIndex && sp.st == null
    }

    const isStepSelected = (cIndex, stIndex) => {
      const sp = props.selectedPath
      return sp && sp.c === cIndex && sp.st === stIndex
    }

    const onAddCommand = (template) => {
      emit('add-command', template)
    }

    return {
      commands,
      isCommandSelected,
      isStepSelected,
      onAddCommand,
      commandTypeLabel,
      stepSummary,
    }
  },
}
</script>

<style scoped>
.task-flow-canvas {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--app-panel-bg);
}

.canvas-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.canvas-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 16px;
  border-bottom: 1px solid var(--app-border);
  background: var(--app-bg);
  flex-shrink: 0;
  min-height: 44px;
  box-sizing: border-box;
}

.canvas-title {
  display: flex;
  align-items: baseline;
  gap: 10px;
  min-width: 0;
}

.title-text {
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.title-hint {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--app-text-muted);
}

.flow-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 20px 18px 28px;
  background:
    radial-gradient(circle at 1px 1px, color-mix(in srgb, var(--app-border) 70%, transparent) 1px, transparent 0)
      0 0 / 18px 18px;
}

.flow-row {
  display: flex;
  align-items: flex-start;
  gap: 0;
  min-width: max-content;
}

.stage-col {
  width: 220px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.stage-label {
  font-size: 12px;
  color: var(--app-text-muted);
  font-weight: 500;
  padding-left: 4px;
}

.stage-card {
  padding: 12px;
  border: 1px solid var(--app-card-border);
  border-radius: var(--app-radius-lg, 10px);
  background: var(--app-card-bg);
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.stage-card:hover {
  border-color: var(--app-accent-color);
}

.stage-card.selected {
  border-color: var(--app-accent-color);
  background: var(--app-card-active-bg);
  box-shadow: 0 0 0 1px var(--app-accent-color);
}

.stage-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 4px;
}

.stage-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stage-meta {
  font-size: 12px;
  color: var(--app-text-muted);
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stage-card-actions {
  display: flex;
  justify-content: flex-end;
}

.steps-stack {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 0 4px;
}

.step-pill {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: var(--app-radius-md, 8px);
  border: 1px solid var(--step-border, var(--app-border));
  background: var(--step-bg, var(--app-card-bg));
  cursor: pointer;
  font-size: 12px;
  color: var(--app-text-secondary);
  transition: border-color 0.15s, background 0.15s;
}

.step-pill:hover {
  border-color: var(--app-accent-color);
  color: var(--app-accent-color);
}

.step-pill.selected {
  border-color: var(--app-accent-color);
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
  font-weight: 500;
}

.step-index {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--app-border);
  color: var(--app-text-muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
}

.step-pill.selected .step-index {
  background: var(--app-accent-color);
  color: #fff;
}

.step-text {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

.step-del {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--app-text-muted);
  cursor: pointer;
  opacity: 0;
  font-size: 14px;
  line-height: 1;
}

.step-pill:hover .step-del {
  opacity: 1;
}

.step-del:hover {
  background: var(--terminal-error);
  color: #fff;
}

.add-step-btn {
  width: 100%;
  height: 28px;
  border: 1px dashed var(--app-border);
  border-radius: var(--app-radius-md, 8px);
  background: transparent;
  color: var(--app-text-muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  padding: 0;
}

.add-step-btn:hover {
  border-color: var(--app-accent-color);
  color: var(--app-accent-color);
}

.flow-connector {
  display: flex;
  align-items: center;
  width: 48px;
  flex-shrink: 0;
  padding-top: 36px;
  gap: 0;
}

.connector-line {
  flex: 1;
  height: 1px;
  background: var(--app-border);
}

.insert-btn {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 1px solid var(--app-border);
  background: var(--app-card-bg);
  color: var(--app-text-muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  flex-shrink: 0;
  padding: 0;
}

.insert-btn:hover {
  border-color: var(--app-accent-color);
  color: var(--app-accent-color);
  background: var(--app-accent-bg);
}

.flow-tail {
  display: flex;
  align-items: flex-start;
  padding: 36px 0 0 8px;
}

.insert-btn.tail {
  width: 28px;
  height: 28px;
}

.flow-empty-hint {
  padding: 40px 24px;
  color: var(--app-text-muted);
  font-size: 13px;
}
</style>
