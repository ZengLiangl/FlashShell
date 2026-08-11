<template>
    <div class="panel-section subproject-section">
        <div class="section-header">
            <div class="header-left">
                <el-tooltip content="返回项目列表" placement="top">
                    <el-button size="small" text circle class="back-btn" @click="$emit('back')">
                        <el-icon><ArrowLeft /></el-icon>
                    </el-button>
                </el-tooltip>
                <div class="header-titles">
                    <h3>可执行项目</h3>
                    <span v-if="selectedProject" class="project-chip">{{ selectedProject.name }}</span>
                </div>
            </div>
        </div>

        <div v-if="subProjects.length > 0" class="subproject-list">
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
                <div class="subproject-row">
                    <button
                        type="button"
                        class="row-main"
                        @click="onToggleSub(subProject.name)"
                    >
                        <el-icon class="chevron" :class="{ open: expandedSubProjects[subProject.name] }">
                            <ArrowRight />
                        </el-icon>
                        <div class="row-text">
                            <div class="subproject-name" :title="subProject.name">{{ subProject.name }}</div>
                            <div v-if="subProject.description" class="subproject-desc">{{ subProject.description }}</div>
                            <div class="subproject-meta">
                                <span class="meta-pill">{{ subProject.stepCount }} 个命令</span>
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

                <div v-if="expandedSubProjects[subProject.name]" class="commands-tree">
                    <div
                        v-for="command in subProject.commands"
                        :key="command.name"
                        :ref="(el) => setCmdRef(`${subProject.name}-${command.name}`, el)"
                        class="command-block"
                        :class="{ open: expandedCommands[`${subProject.name}-${command.name}`] }"
                    >
                        <div class="command-row">
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
                                        <span class="meta-pill quiet">{{ command.steps?.length || 0 }} 步骤</span>
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

                        <div v-if="expandedCommands[`${subProject.name}-${command.name}`]" class="steps-list">
                            <div v-for="(step, index) in command.steps" :key="index" class="step-item">
                                <div class="step-number">{{ index + 1 }}</div>
                                <div class="step-content">
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
export default {
    name: 'SubProjectList',
    props: {
        selectedProject: { type: Object, default: null },
        subProjects: { type: Array, required: true },
        expandedSubProjects: { type: Object, required: true },
        expandedCommands: { type: Object, required: true },
        status: { type: Object, required: true },
        getCommandTagType: { type: Function, required: true },
        getCommandTypeText: { type: Function, required: true },
        isSubProjectRunning: { type: Function, required: true }
    },
    emits: ['toggle-sub', 'toggle-cmd', 'execute-sub', 'execute-cmd', 'stop-sub', 'dry-run-sub', 'back'],
    created() {
        // 不要放进 data：function ref 会反复赋值，放响应式里会把展开交互打挂
        this.blockRefs = Object.create(null)
        this.cmdRefs = Object.create(null)
    },
    methods: {
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
.panel-section {
    padding: 10px 10px 8px;
    border-bottom: 1px solid var(--app-border);
    display: flex;
    flex-direction: column;
    background: transparent;
    color: var(--app-text);
}

.subproject-section {
    flex: 1;
    min-height: 0;
}

.section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
    padding: 0 2px;
    flex-shrink: 0;
}

.header-left {
    display: flex;
    align-items: center;
    gap: 4px;
    min-width: 0;
    min-height: 28px;
}

.back-btn {
    color: var(--app-text-muted) !important;
    width: 28px !important;
    height: 28px !important;
    min-height: 28px !important;
    padding: 0 !important;
    margin: 0 !important;
    display: inline-flex !important;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
}

.back-btn :deep(.el-icon) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 14px;
    margin: 0;
    line-height: 1;
}

.header-titles {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    min-height: 28px;
}

.header-titles h3 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    line-height: 28px;
    color: var(--app-text);
    white-space: nowrap;
}

.project-chip {
    font-size: 11px;
    line-height: 18px;
    height: 18px;
    padding: 0 8px;
    box-sizing: border-box;
    display: inline-flex;
    align-items: center;
    border-radius: 999px;
    color: var(--app-accent-color);
    background: var(--app-accent-bg);
    border: 1px solid color-mix(in srgb, var(--app-accent-color) 28%, transparent);
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.subproject-list {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    overflow-x: hidden;
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding-bottom: 8px;
    /* 独立层叠上下文，避免子项互相盖住展开内容 */
    isolation: isolate;
}

.subproject-block {
    position: relative;
    z-index: 1;
    flex-shrink: 0;
    border: 1px solid var(--app-card-border);
    border-radius: var(--app-radius-lg, 10px);
    background: var(--app-card-bg);
    /* 禁止裁切展开内容；圆角由自身 border-radius + 内部底色配合 */
    overflow: visible;
    transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.subproject-block:hover {
    border-color: color-mix(in srgb, var(--app-accent-color) 35%, var(--app-card-border));
}

.subproject-block.expanded {
    z-index: 4;
}

.subproject-block.running {
    z-index: 5;
    border-color: color-mix(in srgb, var(--app-accent-color) 50%, var(--app-card-border));
}

.subproject-row,
.command-row {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 10px 10px 10px 6px;
    border-radius: var(--app-radius-md, 8px);
    transition: background 0.12s ease;
}

.subproject-row:hover,
.command-row:hover {
    background: color-mix(in srgb, var(--app-accent-color) 8%, transparent);
}

.command-row {
    padding: 8px 8px 8px 4px;
    margin: 0 4px;
    border-radius: var(--app-radius-sm, 6px);
}

.row-main {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: flex-start;
    gap: 6px;
    margin: 0;
    padding: 2px 4px;
    border: none;
    background: transparent;
    color: inherit;
    text-align: left;
    cursor: pointer;
    border-radius: var(--app-radius-sm, 6px);
}

.row-main:hover {
    background: transparent;
}

.chevron {
    flex-shrink: 0;
    /* 与标题单行同高，图标在行盒内居中，避免相对整块内容顶对齐偏位 */
    width: 18px;
    height: 18px;
    margin-top: 0;
    color: var(--app-text-muted);
    transition: transform 0.15s ease;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    line-height: 1;
}

.chevron.open {
    transform: rotate(90deg);
    color: var(--app-accent-color);
}

.chevron.sm {
    width: 16px;
    height: 17px;
}

.row-text {
    flex: 1;
    min-width: 0;
}

.subproject-name {
    font-weight: 600;
    font-size: 13px;
    color: var(--app-text);
    line-height: 18px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.command-name {
    font-weight: 500;
    font-size: 12.5px;
    color: var(--app-text);
    line-height: 17px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.subproject-desc {
    margin-top: 2px;
    font-size: 12px;
    color: var(--app-text-muted);
    line-height: 1.35;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
}

.subproject-meta,
.command-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    align-items: center;
    margin-top: 6px;
}

.meta-pill {
    font-size: 11px;
    line-height: 1;
    padding: 3px 7px;
    border-radius: 999px;
    color: var(--app-text-secondary);
    background: color-mix(in srgb, var(--app-text) 6%, transparent);
    border: 1px solid var(--app-border);
}

.meta-pill.quiet {
    background: transparent;
}

.row-actions {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    align-self: center;
    gap: 6px;
    padding: 0 2px 0 0;
    height: 28px;
}

.row-actions :deep(.el-button) {
    width: 28px !important;
    height: 28px !important;
    min-width: 28px !important;
    min-height: 28px !important;
    padding: 0 !important;
    margin: 0 !important;
    display: inline-flex !important;
    align-items: center;
    justify-content: center;
    box-sizing: border-box;
}

.row-actions :deep(.el-button .el-icon) {
    font-size: 14px;
    margin: 0;
}

.row-actions :deep(.run-btn.is-loading) {
    pointer-events: none;
}

.row-actions :deep(.run-btn .el-icon.is-loading),
.row-actions :deep(.run-btn .el-loading-spinner),
.row-actions :deep(.run-btn > span) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    margin: 0;
}

.commands-tree {
    position: relative;
    z-index: 1;
    border-top: 1px solid var(--app-border);
    background: color-mix(in srgb, var(--app-inset-bg) 65%, var(--app-card-bg));
    padding: 4px 0 6px;
    overflow: visible;
    border-radius: 0 0 calc(var(--app-radius-lg, 10px) - 1px) calc(var(--app-radius-lg, 10px) - 1px);
}

.command-block {
    position: relative;
    z-index: 1;
    overflow: visible;
    background: transparent;
}

.command-block.open {
    z-index: 2;
}

.command-block + .command-block {
    border-top: 1px dashed color-mix(in srgb, var(--app-border) 80%, transparent);
}

.steps-list {
    position: relative;
    z-index: 1;
    margin: 0 12px 10px 28px;
    padding: 6px 8px;
    border-radius: var(--app-radius-md, 8px);
    border: 1px solid var(--app-border);
    background: var(--step-bg, color-mix(in srgb, var(--app-inset-bg) 80%, transparent));
}

.step-item {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    padding: 6px 0;
}

.step-item + .step-item {
    border-top: 1px solid color-mix(in srgb, var(--app-border) 70%, transparent);
}

.step-number {
    flex-shrink: 0;
    width: 18px;
    height: 18px;
    margin-top: 1px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 10px;
    font-weight: 600;
    color: #fff;
    background: var(--app-accent-color);
}

.step-content {
    flex: 1;
    min-width: 0;
}

.step-command {
    font-family: Consolas, Monaco, "Courier New", monospace;
    font-size: 11.5px;
    line-height: 1.45;
    color: var(--app-text);
    word-break: break-all;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
}

.step-meta {
    margin-top: 3px;
    font-size: 11px;
    color: var(--app-text-muted);
}
</style>
