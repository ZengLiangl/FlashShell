<template>
  <div class="task-flow-canvas">
    <div v-if="!subProject" class="canvas-empty">
      <el-empty :description="emptyDescription" :image-size="72" />
    </div>
    <template v-else>
      <div class="canvas-header">
        <div class="canvas-title">
          <span class="title-text">{{ subProject.name || '(未命名子项目)' }}</span>
          <span class="title-hint">命令从左到右执行，拖动卡片可调整顺序</span>
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

            <div
              class="stage-col"
              :class="{
                'is-cmd-drop': dragKind === 'command' && dropCmdIndex === cIndex && dragCmdIndex !== cIndex,
              }"
              @dragover.prevent="onCommandDragOver(cIndex, $event)"
              @dragleave="onCommandDragLeave(cIndex, $event)"
              @drop.prevent="onCommandDrop(cIndex, $event)"
            >
              <div class="stage-label">阶段 {{ cIndex + 1 }}</div>
              <div
                class="stage-card"
                :class="{
                  selected: isCommandSelected(cIndex),
                  'is-dragging': dragKind === 'command' && dragCmdIndex === cIndex,
                }"
                draggable="true"
                title="拖动调整命令顺序"
                @click="$emit('select-command', cIndex)"
                @dragstart="onCommandDragStart(cIndex, $event)"
                @dragend="onDragEnd"
              >
                <div class="stage-card-head">
                  <span class="drag-handle" aria-hidden="true">
                    <el-icon :size="14"><Rank /></el-icon>
                  </span>
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
                <div class="stage-card-actions icon-actions" @mousedown.stop @dragstart.stop.prevent>
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

              <div
                class="steps-stack"
                :class="{
                  'is-step-drop-host':
                    dragKind === 'step' && dropStep?.c === cIndex && dragStep?.c !== cIndex,
                }"
                @dragover.prevent="onStepStackDragOver(cIndex, $event)"
                @drop.prevent="onStepStackDrop(cIndex, $event)"
              >
                <div
                  v-for="(step, stIndex) in cmd.steps || []"
                  :key="stIndex"
                  class="step-pill"
                  :class="{
                    selected: isStepSelected(cIndex, stIndex),
                    'is-dragging':
                      dragKind === 'step' && dragStep?.c === cIndex && dragStep?.st === stIndex,
                    'is-drop-target':
                      dragKind === 'step' &&
                      dropStep?.c === cIndex &&
                      dropStep?.st === stIndex &&
                      !(dragStep?.c === cIndex && dragStep?.st === stIndex),
                  }"
                  draggable="true"
                  title="拖动调整步骤顺序"
                  @click.stop="$emit('select-step', cIndex, stIndex)"
                  @dragstart="onStepDragStart(cIndex, stIndex, $event)"
                  @dragend="onDragEnd"
                  @dragover.prevent="onStepDragOver(cIndex, stIndex, $event)"
                  @dragleave="onStepDragLeave(cIndex, stIndex, $event)"
                  @drop.prevent="onStepDrop(cIndex, stIndex, $event)"
                >
                  <span class="drag-handle step-handle" aria-hidden="true">
                    <el-icon :size="12"><Rank /></el-icon>
                  </span>
                  <span class="step-index">{{ stIndex + 1 }}</span>
                  <span class="step-text" :title="step.command">{{ stepSummary(step) }}</span>
                  <div
                    class="step-del-wrap icon-actions"
                    @click.stop
                    @mousedown.stop
                    @dragstart.stop.prevent
                  >
                    <el-tooltip content="删除步骤" placement="top">
                      <el-button
                        class="step-del-btn"
                        type="danger"
                        text
                        size="small"
                        circle
                        @click="$emit('remove-step', cIndex, stIndex)"
                      >
                        <el-icon><Delete /></el-icon>
                      </el-button>
                    </el-tooltip>
                  </div>
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
import { computed, ref } from 'vue'
import { Delete, Plus, Rank } from '@element-plus/icons-vue'
import { commandTypeLabel, stepSummary } from './taskFlowModel'

const CMD_MIME = 'application/x-flashdock-cmd'
const STEP_MIME = 'application/x-flashdock-step'

export default {
  name: 'TaskFlowCanvas',
  components: { Delete, Plus, Rank },
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
    'reorder-command',
    'reorder-step',
  ],
  setup(props, { emit }) {
    const commands = computed(() => props.subProject?.commands || [])
    const dragKind = ref('')
    const dragCmdIndex = ref(null)
    const dropCmdIndex = ref(null)
    const dragStep = ref(null)
    const dropStep = ref(null)

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

    const clearDragState = () => {
      dragKind.value = ''
      dragCmdIndex.value = null
      dropCmdIndex.value = null
      dragStep.value = null
      dropStep.value = null
    }

    const onDragEnd = () => {
      clearDragState()
    }

    const isNoDragZone = (event) => {
      const el = event?.target
      if (!el?.closest) return false
      return !!el.closest(
        'button, .el-button, .stage-card-actions, .step-del-wrap, a, input, textarea, select',
      )
    }

    const onCommandDragStart = (cIndex, event) => {
      if (isNoDragZone(event)) {
        event.preventDefault()
        return
      }
      dragKind.value = 'command'
      dragCmdIndex.value = cIndex
      dropCmdIndex.value = null
      dragStep.value = null
      dropStep.value = null
      try {
        event.dataTransfer.effectAllowed = 'move'
        event.dataTransfer.setData(CMD_MIME, String(cIndex))
        event.dataTransfer.setData('text/plain', `cmd:${cIndex}`)
      } catch {
        // ignore
      }
    }

    const onCommandDragOver = (cIndex, event) => {
      if (dragKind.value !== 'command') return
      event.dataTransfer.dropEffect = 'move'
      dropCmdIndex.value = cIndex
    }

    const onCommandDragLeave = (cIndex, event) => {
      if (dragKind.value !== 'command') return
      const related = event.relatedTarget
      if (related && event.currentTarget?.contains?.(related)) return
      if (dropCmdIndex.value === cIndex) dropCmdIndex.value = null
    }

    const onCommandDrop = (cIndex, event) => {
      if (dragKind.value !== 'command') return
      const raw =
        event.dataTransfer.getData(CMD_MIME) ||
        event.dataTransfer.getData('text/plain').replace(/^cmd:/, '')
      const from = dragCmdIndex.value ?? Number(raw)
      clearDragState()
      if (!Number.isInteger(from) || from === cIndex) return
      emit('reorder-command', { from, to: cIndex })
    }

    const onStepDragStart = (cIndex, stIndex, event) => {
      if (isNoDragZone(event)) {
        event.preventDefault()
        return
      }
      dragKind.value = 'step'
      dragStep.value = { c: cIndex, st: stIndex }
      dropStep.value = null
      dragCmdIndex.value = null
      dropCmdIndex.value = null
      try {
        event.dataTransfer.effectAllowed = 'move'
        event.dataTransfer.setData(STEP_MIME, JSON.stringify({ c: cIndex, st: stIndex }))
        event.dataTransfer.setData('text/plain', `step:${cIndex}:${stIndex}`)
      } catch {
        // ignore
      }
    }

    const onStepDragOver = (cIndex, stIndex, event) => {
      if (dragKind.value !== 'step') return
      event.stopPropagation()
      event.dataTransfer.dropEffect = 'move'
      dropStep.value = { c: cIndex, st: stIndex }
    }

    const onStepDragLeave = (cIndex, stIndex, event) => {
      if (dragKind.value !== 'step') return
      const related = event.relatedTarget
      if (related && event.currentTarget?.contains?.(related)) return
      if (dropStep.value?.c === cIndex && dropStep.value?.st === stIndex) {
        dropStep.value = null
      }
    }

    const onStepStackDragOver = (cIndex, event) => {
      if (dragKind.value !== 'step') return
      // pill 自己维护落点；栈空白区落到末尾
      if (event.target?.closest?.('.step-pill')) return
      event.dataTransfer.dropEffect = 'move'
      const len = commands.value[cIndex]?.steps?.length || 0
      dropStep.value = { c: cIndex, st: len }
    }

    const resolveStepFrom = (event) => {
      if (dragStep.value) return dragStep.value
      try {
        const raw = event.dataTransfer.getData(STEP_MIME)
        if (raw) return JSON.parse(raw)
      } catch {
        // ignore
      }
      const plain = event.dataTransfer.getData('text/plain') || ''
      const m = plain.match(/^step:(\d+):(\d+)$/)
      if (m) return { c: Number(m[1]), st: Number(m[2]) }
      return null
    }

    const emitStepReorder = (from, toC, toSt) => {
      if (!from) return
      if (from.c === toC && from.st === toSt) return
      emit('reorder-step', {
        fromC: from.c,
        fromSt: from.st,
        toC,
        toSt,
      })
    }

    const onStepDrop = (cIndex, stIndex, event) => {
      if (dragKind.value !== 'step') return
      event.stopPropagation()
      const from = resolveStepFrom(event)
      clearDragState()
      emitStepReorder(from, cIndex, stIndex)
    }

    const onStepStackDrop = (cIndex, event) => {
      if (dragKind.value !== 'step') return
      if (event.target?.closest?.('.step-pill')) return
      const from = resolveStepFrom(event)
      const len = commands.value[cIndex]?.steps?.length || 0
      clearDragState()
      emitStepReorder(from, cIndex, len)
    }

    return {
      commands,
      dragKind,
      dragCmdIndex,
      dropCmdIndex,
      dragStep,
      dropStep,
      isCommandSelected,
      isStepSelected,
      onAddCommand,
      commandTypeLabel,
      stepSummary,
      onCommandDragStart,
      onCommandDragOver,
      onCommandDragLeave,
      onCommandDrop,
      onStepDragStart,
      onStepDragOver,
      onStepDragLeave,
      onStepDrop,
      onStepStackDragOver,
      onStepStackDrop,
      onDragEnd,
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
  border-radius: var(--app-radius-lg, 10px);
  transition: background 0.12s ease, box-shadow 0.12s ease;
}

.stage-col.is-cmd-drop {
  background: color-mix(in srgb, var(--app-accent-color) 8%, transparent);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--app-accent-color) 45%, transparent);
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
  cursor: grab;
  transition: border-color 0.15s, box-shadow 0.15s, opacity 0.12s ease;
  user-select: none;
}

.stage-card:active {
  cursor: grabbing;
}

.stage-card:hover {
  border-color: var(--app-accent-color);
}

.stage-card.selected {
  border-color: var(--app-accent-color);
  background: var(--app-card-active-bg);
  box-shadow: 0 0 0 1px var(--app-accent-color);
}

.stage-card.is-dragging {
  opacity: 0.45;
}

.stage-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 4px;
}

.drag-handle {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  margin: 0;
  border-radius: 4px;
  color: var(--app-text-muted);
  cursor: grab;
  user-select: none;
  box-sizing: border-box;
  line-height: 1;
  pointer-events: none;
}

.drag-handle .el-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin: 0;
}

.stage-card:hover .drag-handle,
.step-pill:hover .drag-handle {
  color: var(--app-accent-color);
  background: color-mix(in srgb, var(--app-accent-color) 12%, transparent);
}

.stage-name {
  flex: 1;
  min-width: 0;
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
  cursor: default;
  user-select: auto;
}

.steps-stack {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 4px;
  min-height: 36px;
  border-radius: var(--app-radius-md, 8px);
  transition: background 0.12s ease, box-shadow 0.12s ease;
}

.steps-stack.is-step-drop-host {
  background: color-mix(in srgb, var(--app-accent-color) 8%, transparent);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--app-accent-color) 40%, transparent);
}

.step-pill {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  border-radius: var(--app-radius-md, 8px);
  border: 1px solid var(--step-border, var(--app-border));
  background: var(--step-bg, var(--app-card-bg));
  cursor: grab;
  font-size: 12px;
  color: var(--app-text-secondary);
  transition: border-color 0.15s, background 0.15s, opacity 0.12s ease;
  user-select: none;
}

.step-pill:active {
  cursor: grabbing;
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

.step-pill.is-dragging {
  opacity: 0.4;
}

.step-pill.is-drop-target {
  border-color: var(--app-accent-color);
  box-shadow: 0 -2px 0 0 var(--app-accent-color);
}

.step-handle {
  opacity: 0.55;
}

.step-pill:hover .step-handle,
.step-pill.selected .step-handle {
  opacity: 1;
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
  line-height: 1;
  box-sizing: border-box;
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

.step-del-wrap {
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.12s ease;
  cursor: default;
  user-select: auto;
}

.step-pill:hover .step-del-wrap,
.step-pill.selected .step-del-wrap {
  opacity: 1;
}

.step-del-wrap.icon-actions .el-button.is-circle {
  width: 20px;
  height: 20px;
  min-height: 20px;
  padding: 0;
}

.add-step-btn {
  width: 100%;
  height: 28px;
  padding: 0;
  margin: 0;
  border: 1px dashed var(--app-border);
  border-radius: var(--app-radius-md, 8px);
  background: transparent;
  color: var(--app-text-muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-sizing: border-box;
  line-height: 1;
}

.add-step-btn .el-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin: 0;
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
  padding: 0;
  margin: 0;
  border-radius: 50%;
  border: 1px solid var(--app-border);
  background: var(--app-card-bg);
  color: var(--app-text-muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  flex-shrink: 0;
  box-sizing: border-box;
  line-height: 1;
}

.insert-btn .el-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin: 0;
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
