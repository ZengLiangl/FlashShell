<template>
    <div class="panel-section subproject-section">
        <div class="section-header">
            <h3>可执行项目</h3>
            <div class="header-actions">
                <el-tag v-if="selectedProject" size="small">{{ selectedProject.name }}</el-tag>
                <el-tooltip content="返回" placement="top">
                    <el-button size="small" type="primary" text circle @click="$emit('back')">
                        <el-icon><ArrowLeft /></el-icon>
                    </el-button>
                </el-tooltip>
            </div>
        </div>

        <div v-if="subProjects.length > 0" class="subproject-list">
            <div v-for="subProject in subProjects" :key="subProject.name" class="subproject-container">
                <div class="subproject-item">
                    <div class="subproject-info">
                        <div class="subproject-header">
                            <el-button size="small" text @click="$emit('toggle-sub', subProject.name)"
                                class="expand-button">
                                <el-icon>
                                    <ArrowRight v-if="!expandedSubProjects[subProject.name]" />
                                    <ArrowDown v-else />
                                </el-icon>
                            </el-button>
                            <div class="subproject-title">
                                <div class="subproject-name">{{ subProject.name }}</div>
                                <div class="subproject-desc">{{ subProject.description }}</div>
                                <div class="subproject-meta">
                                    <el-tag size="small" type="info">{{ subProject.stepCount }} 个命令</el-tag>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="subproject-actions icon-actions">
                        <el-tooltip :content="isSubProjectRunning(subProject) ? '运行中' : '执行'" placement="top">
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

                <div v-if="expandedSubProjects[subProject.name]" class="commands-container">
                    <div v-for="command in subProject.commands" :key="command.name" class="command-container">
                        <div class="command-item">
                            <div class="command-header">
                                <el-button size="small" text @click="$emit('toggle-cmd', subProject.name, command.name)"
                                    class="expand-button">
                                    <el-icon>
                                        <ArrowRight v-if="!expandedCommands[`${subProject.name}-${command.name}`]" />
                                        <ArrowDown v-else />
                                    </el-icon>
                                </el-button>
                                <div class="command-info">
                                    <div class="command-name">{{ command.name }}</div>
                                </div>
                            </div>
                            <div class="command-meta">
                                <el-tag size="small" :type="getCommandTagType(command.type)" effect="light">
                                    {{ getCommandTypeText(command.type) }}
                                </el-tag>
                                <el-tag size="small" type="info" effect="plain">
                                    {{ command.steps?.length || 0 }} 命令
                                </el-tag>
                            </div>
                        </div>

                        <div v-if="expandedCommands[`${subProject.name}-${command.name}`]" class="steps-container">
                            <div v-for="(step, index) in command.steps" :key="index" class="step-item">
                                <div class="step-number">{{ index + 1 }}</div>
                                <div class="step-content">
                                    <div class="step-command">{{ stepDisplay(step) }}</div>
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
    emits: ['toggle-sub', 'toggle-cmd', 'execute-sub', 'stop-sub', 'back'],
    methods: {
        stepDisplay(step) {
            if (typeof step === 'string') return step
            return step?.command || step?.cmd || ''
        },
        stepMeta(step) {
            if (typeof step === 'string') return ''
            const parts = []
            if (step?.onFail && step.onFail !== 'abort') parts.push(`on_fail: ${step.onFail}`)
            if (step?.retry > 0) parts.push(`retry: ${step.retry}`)
            return parts.join(' · ')
        }
    }
}
</script>

<style scoped>
.panel-section {
    padding: 16px;
    border-bottom: 1px solid var(--app-border);
    display: flex;
    flex-direction: column;
    background: var(--app-panel-bg);
    color: var(--app-text);
}

.subproject-section {
    flex: 1;
    min-height: 0;
}

.section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
}

.header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
}

.subproject-list {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    overflow-x: hidden;
}

.subproject-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px;
    margin-bottom: 8px;
    background: var(--app-card-bg);
    border-radius: 6px;
    border: 1px solid var(--app-border);
}

.subproject-info {
    flex: 1;
}

.subproject-name {
    font-weight: 600;
    color: var(--app-text);
    margin-bottom: 4px;
}

.subproject-desc {
    font-size: 12px;
    color: var(--app-text-muted);
    margin-bottom: 6px;
}

.subproject-meta {
    display: flex;
    gap: 6px;
    align-items: center;
    margin-top: 6px;
}

.subproject-actions {
    flex-shrink: 0;
}

.subproject-actions :deep(.run-btn.is-loading) {
    pointer-events: none;
}

.subproject-actions :deep(.run-btn .el-icon.is-loading),
.subproject-actions :deep(.run-btn .el-loading-spinner),
.subproject-actions :deep(.run-btn > span) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    margin: 0;
}

.subproject-container {
    margin-bottom: 8px;
}

.subproject-header {
    display: flex;
    align-items: center;
    gap: 8px;
}

.expand-button {
    padding: 4px !important;
    min-width: auto !important;
    width: 24px;
    height: 24px;
}

.subproject-title {
    flex: 1;
}

.commands-container {
    margin-left: 32px;
    margin-top: 8px;
    border-left: 2px solid var(--app-border);
    padding-left: 12px;
}

.command-container {
    margin-bottom: 8px;
}

.command-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 12px;
    background: var(--app-card-bg);
    border-radius: 4px;
    border: 1px solid var(--app-border);
}

.command-header {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
}

.command-info {
    flex: 1;
}

.command-name {
    display: flex;
    align-items: center;
    gap: 6px;
    font-weight: 500;
    color: #606266;
    font-size: 13px;
    margin-bottom: 2px;
}

.command-type-icon {
    font-size: 14px;
    color: var(--app-accent-color, #409eff);
}

.command-desc {
    font-size: 11px;
    color: #909399;
}

.command-meta {
    display: flex;
    gap: 6px;
    align-items: center;
}

.steps-container {
    margin-left: 32px;
    margin-top: 6px;
    border-left: 2px solid #f0f0f0;
    padding-left: 12px;
}

.step-item {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    padding: 6px 0;
    border-bottom: 1px solid #f5f5f5;
}

.step-item:last-child {
    border-bottom: none;
}

.step-number {
    background: var(--app-accent-color, #409eff);
    color: white;
    border-radius: 50%;
    width: 18px;
    height: 18px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 10px;
    font-weight: 500;
    flex-shrink: 0;
    margin-top: 2px;
}

.step-content {
    flex: 1;
}

.step-command {
    font-family: "Consolas", "Monaco", "Courier New", monospace;
    font-size: 12px;
    color: var(--app-text);
    background: var(--step-bg);
    padding: 4px 8px;
    border-radius: 3px;
    border: 1px solid var(--step-border);
    margin-bottom: 2px;
}

.step-meta {
    font-size: 11px;
    color: var(--app-text-muted);
    margin-bottom: 4px;
}

.step-desc {
    font-size: 11px;
    color: #909399;
}
</style>
