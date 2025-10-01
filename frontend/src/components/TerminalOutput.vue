<template>
    <div class="terminal-wrapper">
        <!-- 进度条区域 -->
        <transition name="progress-slide" appear>
            <div v-if="status.isRunning" class="progress-section">
                <div class="progress-info">
                    <div class="progress-text">
                        <span class="project-name">{{ status.currentCommand }}</span>
                        <transition name="command-fade" mode="out-in">
                            <span v-if="status.currentStep" key="current" class="current-command">
                                正在执行: {{ status.currentStep }}
                            </span>
                            <span v-else key="waiting" class="current-command">
                                准备执行...
                            </span>
                        </transition>
                    </div>
                    <div class="progress-stats">
                        {{ Math.max(1, status.completedSteps + 1) }}/{{ status.totalSteps }} 命令
                    </div>
                </div>
                <el-progress :percentage="progressPercentage" :status="progressStatus" :stroke-width="8"
                    :show-text="true" class="execution-progress" />
            </div>
        </transition>

        <div class="terminal-output" ref="terminalOutputRef">
            <div v-for="(line, index) in outputLines" :key="index" class="output-line" :class="{
                'error-line': line.isError,
                'success-line': line.isSuccess,
                'progress-line': line.isProgress,
            }" v-html="line.html">
            </div>
            <div v-if="outputLines.length === 0" class="empty-output">
                等待命令输出...
            </div>
            <div ref="bottomMarker"></div>
        </div>
    </div>
</template>

<script>
import { ref, watch, nextTick } from 'vue'

export default {
    name: 'TerminalOutput',
    props: {
        status: { type: Object, required: true },
        outputLines: { type: Array, required: true },
        progressPercentage: { type: Number, required: true },
        progressStatus: { type: String, required: true }
    },
    setup(props, { expose }) {
        const terminalOutputRef = ref(null)
        const bottomMarker = ref(null)

        const scrollToBottom = () => {
            nextTick(() => {
                if (bottomMarker.value && typeof bottomMarker.value.scrollIntoView === 'function') {
                    bottomMarker.value.scrollIntoView({ behavior: 'auto', block: 'end' })
                } else if (terminalOutputRef.value) {
                    terminalOutputRef.value.scrollTop = terminalOutputRef.value.scrollHeight
                }
            })
        }

        watch(() => props.outputLines.length, () => {
            scrollToBottom()
        })

        expose({ terminalOutputRef })

        return { terminalOutputRef, bottomMarker }
    }
}
</script>

<style scoped>
.terminal-wrapper {
    display: flex;
    flex-direction: column;
    height: 100%;
}

/* 进度与输出样式从 App.vue 迁移 */
.progress-section {
    padding: 12px 16px;
    background: #f8f9fa;
    border-bottom: 1px solid #e4e7ed;
}

.progress-info {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
}

.progress-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
}

.project-name {
    font-weight: 600;
    color: #303133;
    font-size: 14px;
}

.current-command {
    font-size: 12px;
    color: #606266;
}

.progress-stats {
    font-size: 12px;
    color: #909399;
    font-weight: 500;
}

.execution-progress {
    margin: 0;
}

.terminal-output {
    flex: 1;
    padding: 16px;
    background: #1e1e1e;
    color: #d4d4d4;
    font-family: "Consolas", "Monaco", "Courier New", monospace;
    font-size: 13px;
    line-height: 1.4;
    overflow-y: auto;
    white-space: pre-wrap;
}

.output-line {
    margin-bottom: 2px;
    word-break: break-all;
}

.error-line {
    color: #f56c6c;
}

.success-line {
    color: #67c23a;
}

.progress-line {
    color: #409eff;
    font-weight: 500;
}

.empty-output {
    color: #909399;
    text-align: center;
    margin-top: 50px;
}
</style>
