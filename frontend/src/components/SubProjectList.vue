<template>
    <div class="panel-section subproject-section task-left-panel">
        <div class="section-header tl-head" @dblclick="onChromeTitleDblActivate" @mousedown="onChromeTitlePointerDown">
            <div class="project-tabs" role="tablist" aria-label="切换项目">
                <template v-for="(project, index) in projects" :key="project.name">
                    <span v-if="index > 0" class="project-tab-sep" aria-hidden="true">｜</span>
                    <button
                        type="button"
                        role="tab"
                        class="project-tab"
                        :class="{ active: project.name === selectedProject?.name }"
                        :aria-selected="project.name === selectedProject?.name"
                        :disabled="status.isRunning"
                        :title="project.name"
                        @click="onSelectProject(project.name)"
                    >
                        {{ project.name }}
                    </button>
                </template>
            </div>
            <div class="header-actions icon-actions" @click.stop>
                <el-tooltip content="刷新配置文件" placement="top">
                    <el-button
                        size="small"
                        circle
                        :disabled="status.isRunning"
                        @click="onRefresh"
                    >
                        <el-icon><Refresh /></el-icon>
                    </el-button>
                </el-tooltip>
            </div>
        </div>

        <div v-if="subProjects.length > 0" class="subproject-list tree">
            <div
                v-for="subProject in subProjects"
                :key="subProject.name"
                :ref="(el) => setBlockRef(subProject.name, el)"
                class="subproject-block"
                :class="{
                    expanded: expandedSubProjects[subProject.name],
                    running: isSubProjectRunning(subProject),
                }"
            >
                <div class="subproject-row sp-row tree-item">
                    <button
                        type="button"
                        class="row-main"
                        @click="onToggleSub(subProject.name)"
                    >
                        <el-icon class="chevron chev" :class="{ open: expandedSubProjects[subProject.name] }">
                            <ArrowRight />
                        </el-icon>
                        <div class="row-text sp-main">
                            <div class="subproject-name sp-name" :title="subProject.name">{{ subProject.name }}</div>
                            <div v-if="subProject.description" class="subproject-desc sp-desc">{{ subProject.description }}</div>
                            <div class="subproject-meta sp-meta">
                                <span class="meta-pill tag tag-info">{{ subProject.stepCount }} 个命令</span>
                            </div>
                        </div>
                    </button>
                    <div class="row-actions icon-actions" @click.stop>
                        <el-tooltip content="干跑" placement="top">
                            <el-button
                                size="small"
                                circle
                                :disabled="status.isRunning"
                                @click="$emit('dry-run-sub', subProject)"
                            >
                                <el-icon><Document /></el-icon>
                            </el-button>
                        </el-tooltip>
                        <el-tooltip
                            content="执行"
                            placement="top"
                            :disabled="isSubProjectRunning(subProject)"
                        >
                            <el-button
                                size="small"
                                type="primary"
                                circle
                                class="run-btn"
                                :loading="isSubProjectRunning(subProject)"
                                :disabled="status.isRunning && !isSubProjectRunning(subProject)"
                                @click="$emit('execute-sub', subProject)"
                            >
                                <el-icon v-if="!isSubProjectRunning(subProject)"><VideoPlay /></el-icon>
                            </el-button>
                        </el-tooltip>
                        <el-tooltip v-if="isSubProjectRunning(subProject)" content="停止" placement="top">
                            <el-button size="small" type="danger" circle @click="$emit('stop-sub', subProject)">
                                <el-icon><VideoPause /></el-icon>
                            </el-button>
                        </el-tooltip>
                    </div>
                </div>

                <div v-if="expandedSubProjects[subProject.name]" class="commands-tree cmd-list">
                    <div
                        v-for="command in subProject.commands"
                        :key="command.name"
                        :ref="(el) => setCmdRef(`${subProject.name}-${command.name}`, el)"
                        class="command-block"
                        :class="{ open: expandedCommands[`${subProject.name}-${command.name}`] }"
                    >
                        <div class="command-row cmd-row tree-item">
                            <button
                                type="button"
                                class="row-main"
                                @click="onToggleCmd(subProject.name, command.name)"
                            >
                                <el-icon class="chevron sm" :class="{ open: expandedCommands[`${subProject.name}-${command.name}`] }">
                                    <ArrowRight />
                                </el-icon>
                                <div class="row-text">
                                    <div class="command-name" :title="command.name">{{ command.name }}</div>
                                    <div class="command-meta">
                                        <el-tag size="small" :type="getCommandTagType(command.type)" effect="plain">
                                            {{ getCommandTypeText(command.type) }}
                                        </el-tag>
                                        <el-tag v-if="command.parallel" size="small" type="warning" effect="plain">
                                            并行
                                        </el-tag>
                                        <span class="meta-pill tag tag-info">{{ command.steps?.length || 0 }} 步骤</span>
                                    </div>
                                </div>
                            </button>
                            <div class="row-actions icon-actions" @click.stop>
                                <el-tooltip content="仅执行此命令" placement="top">
                                    <el-button
                                        size="small"
                                        type="primary"
                                        circle
                                        class="run-btn"
                                        :disabled="status.isRunning"
                                        @click="$emit('execute-cmd', { subProject, command })"
                                    >
                                        <el-icon><VideoPlay /></el-icon>
                                    </el-button>
                                </el-tooltip>
                            </div>
                        </div>

                        <div v-if="expandedCommands[`${subProject.name}-${command.name}`]" class="steps-list step-list">
                            <div v-for="(step, index) in command.steps" :key="index" class="step-item">
                                <div class="step-number s-num">{{ index + 1 }}</div>
                                <div class="step-content s-cmd">
                                    <div class="step-command" :title="stepDisplay(step)">{{ stepDisplay(step) }}</div>
                                    <div v-if="stepMeta(step)" class="step-meta">{{ stepMeta(step) }}</div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        <el-empty v-else description="请选择项目查看可执行项目" />
    </div>
</template>

<script>
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { onChromeTitleDblActivate, onChromeTitlePointerDown } from '../utils/windowChrome'

export default {
    name: 'SubProjectList',
    components: { Refresh },
    props: {
        selectedProject: { type: Object, default: null },
        projects: { type: Array, default: () => [] },
        subProjects: { type: Array, required: true },
        expandedSubProjects: { type: Object, required: true },
        expandedCommands: { type: Object, required: true },
        status: { type: Object, required: true },
        getCommandTagType: { type: Function, required: true },
        getCommandTypeText: { type: Function, required: true },
        isSubProjectRunning: { type: Function, required: true }
    },
    emits: ['toggle-sub', 'toggle-cmd', 'execute-sub', 'execute-cmd', 'stop-sub', 'dry-run-sub', 'select-project', 'refresh'],
    created() {
        // 不要放进 data：function ref 会反复赋值，放响应式里会把展开交互打挂
        this.blockRefs = Object.create(null)
        this.cmdRefs = Object.create(null)
    },
    methods: {
        onChromeTitleDblActivate,
        onChromeTitlePointerDown,
        setBlockRef(name, el) {
            if (el) this.blockRefs[name] = el
            else delete this.blockRefs[name]
        },
        setCmdRef(key, el) {
            if (el) this.cmdRefs[key] = el
            else delete this.cmdRefs[key]
        },
        scrollElIntoList(el) {
            if (!el || typeof el.scrollIntoView !== 'function') return
            requestAnimationFrame(() => {
                el.scrollIntoView({ block: 'nearest', inline: 'nearest', behavior: 'smooth' })
            })
        },
        onToggleSub(name) {
            const willExpand = !this.expandedSubProjects[name]
            this.$emit('toggle-sub', name)
            if (!willExpand) return
            this.$nextTick(() => {
                this.scrollElIntoList(this.blockRefs[name])
            })
        },
        onToggleCmd(subName, cmdName) {
            const key = `${subName}-${cmdName}`
            const willExpand = !this.expandedCommands[key]
            this.$emit('toggle-cmd', subName, cmdName)
            if (!willExpand) return
            this.$nextTick(() => {
                this.scrollElIntoList(this.cmdRefs[key])
            })
        },
        onSelectProject(name) {
            if (this.status?.isRunning) {
                ElMessage.warning('任务执行中，请先停止后再切换项目')
                return
            }
            if (!name || name === this.selectedProject?.name) return
            const project = (this.projects || []).find((p) => p.name === name)
            if (project) this.$emit('select-project', project)
        },
        onRefresh() {
            if (this.status?.isRunning) {
                ElMessage.warning('任务执行中，请先停止后再刷新配置')
                return
            }
            this.$emit('refresh')
        },
        stepDisplay(step) {
            if (typeof step === 'string') return step
            return step?.command || step?.cmd || ''
        },
        stepMeta(step) {
            if (typeof step === 'string') return ''
            const parts = []
            if (step?.when) parts.push(`when: ${step.when}`)
            if (step?.onFail && step.onFail !== 'abort') parts.push(`on_fail: ${step.onFail}`)
            if (step?.retry > 0) parts.push(`retry: ${step.retry}`)
            return parts.join(' · ')
        }
    }
}
</script>

<style scoped>
.task-left-panel {
    padding: 0;
    border-bottom: none;
    display: flex;
    flex-direction: column;
    background: var(--surface-2);
    color: var(--fg);
    height: 100%;
}

.tl-head {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 12px 8px;
    flex-shrink: 0;
}

.project-tabs {
    display: flex;
    align-items: center;
    gap: 0;
    min-width: 0;
    flex: 1;
    overflow-x: auto;
    scrollbar-width: thin;
}

.project-tab {
    flex: 0 0 auto;
    max-width: 120px;
    padding: 4px 2px;
    border: none;
    background: none;
    color: var(--muted);
    font-size: 13px;
    font-weight: 500;
    line-height: 1.3;
    cursor: pointer;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.project-tab:hover:not(:disabled) {
    color: var(--fg);
}

.project-tab.active {
    color: var(--accent);
    font-weight: 600;
}

.project-tab:disabled {
    cursor: not-allowed;
    opacity: 0.65;
}

.project-tab-sep {
    flex: 0 0 auto;
    color: var(--border);
    font-size: 12px;
    user-select: none;
    padding: 0 2px;
}

.header-actions {
    flex-shrink: 0;
}

.header-actions :deep(.el-button) {
    width: 28px !important;
    height: 28px !important;
    min-width: 28px !important;
}

.tree {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 0 8px 14px;
}

.subproject-block {
    position: relative;
    z-index: 1;
    flex-shrink: 0;
    border: none;
    border-radius: 0;
    background: transparent;
    overflow: visible;
}

.subproject-block.expanded { z-index: 4; }
.subproject-block.running { z-index: 5; }

.sp-row,
.cmd-row {
    position: relative;
    border-radius: 8px;
}

.sp-row:hover,
.cmd-row:hover {
    background: color-mix(in oklch, var(--fg) 6%, transparent);
}

.row-text,
.sp-main {
    min-width: 0;
    flex: 1;
    overflow: hidden;
}

.row-actions {
    position: absolute;
    right: 6px;
    top: 50%;
    transform: translateY(-50%);
    opacity: 0;
    pointer-events: none;
    z-index: 2;
    padding: 2px 0 2px 10px;
    background: var(--surface-2);
    border-radius: 8px;
    transition: opacity 0.12s ease;
}

.sp-row:hover .row-actions,
.cmd-row:hover .row-actions {
    opacity: 1;
    pointer-events: auto;
    background: color-mix(in oklch, var(--fg) 6%, var(--surface-2));
}

.commands-tree {
    margin: 2px 0 2px 22px;
    border-left: 1px solid var(--border);
    padding-left: 8px;
    border-top: none;
    background: transparent;
    border-radius: 0;
}

.steps-list {
    margin: 2px 0 2px 20px;
    border-left: 1px solid var(--border);
    padding-left: 8px;
    border: none;
    border-left: 1px solid var(--border);
    background: transparent;
    border-radius: 0;
}

.step-item {
    font-family: var(--font-mono);
    font-size: 12px;
    color: var(--muted);
    padding: 4px 8px;
    border-radius: 6px;
    border-top: none;
}

.step-item:hover {
    background: color-mix(in oklch, var(--fg) 5%, transparent);
    color: var(--fg-2);
}

.step-number,
.s-num {
    color: var(--muted);
    font-size: 12px;
    min-width: 14px;
    text-align: right;
    background: none;
    width: auto;
    height: auto;
    border-radius: 0;
    margin-top: 0;
}

.step-command,
.s-cmd {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-family: var(--font-mono);
    font-size: 12px;
}

.subproject-name,
.sp-name,
.command-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.subproject-name,
.sp-name {
    font-size: 13.5px;
    font-weight: 500;
    color: var(--fg);
}

.command-name {
    font-size: 13px;
    color: var(--fg-2);
}

.subproject-desc,
.sp-desc {
    font-size: 12px;
    color: var(--muted);
    margin-top: 2px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.chevron,
.chev {
    width: 14px;
    height: 14px;
    color: var(--muted);
}

.chevron.open,
.chev.open {
    transform: rotate(90deg);
    color: var(--muted);
}

.row-main {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px;
    border: none;
    background: transparent;
    color: inherit;
    text-align: left;
    cursor: pointer;
    border-radius: 8px;
}

.row-actions {
    flex-shrink: 0;
    display: flex;
    gap: 3px;
}

.row-actions :deep(.el-button) {
    width: 24px !important;
    height: 24px !important;
    min-width: 24px !important;
    min-height: 24px !important;
}

.row-actions :deep(.run-btn) {
    color: var(--accent);
    background: var(--accent-soft);
}

.row-actions :deep(.run-btn:hover) {
    background: color-mix(in oklch, var(--accent) 22%, transparent);
}
</style>
