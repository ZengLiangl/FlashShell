<template>
    <div class="terminal-wrapper">
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

        <div class="terminal-output" ref="terminalOutputRef" @scroll="onScroll">
            <div v-if="outputLines.length === 0" class="empty-output">
                等待命令输出...
            </div>
            <template v-else>
                <div class="virtual-spacer" :style="{ height: topSpacerHeight + 'px' }"></div>
                <div v-for="item in visibleLines" :key="item.index" class="output-line" :class="{
                    'error-line': item.line.isError,
                    'success-line': item.line.isSuccess,
                    'progress-line': item.line.isProgress,
                }" v-html="item.line.html">
                </div>
                <div class="virtual-spacer" :style="{ height: bottomSpacerHeight + 'px' }"></div>
            </template>
            <div ref="bottomMarker"></div>
        </div>
    </div>
</template>

<script>
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'

const LINE_HEIGHT = 20
const OVERSCAN = 20

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
        const scrollTop = ref(0)
        const containerHeight = ref(600)
        const stickToBottom = ref(true)
        let resizeObserver = null

        const visibleRange = computed(() => {
            const total = props.outputLines.length
            if (total === 0) {
                return { start: 0, end: 0 }
            }
            const start = Math.max(0, Math.floor(scrollTop.value / LINE_HEIGHT) - OVERSCAN)
            const visibleCount = Math.ceil(containerHeight.value / LINE_HEIGHT) + OVERSCAN * 2
            const end = Math.min(total, start + visibleCount)
            return { start, end }
        })

        const visibleLines = computed(() => {
            const { start, end } = visibleRange.value
            return props.outputLines.slice(start, end).map((line, offset) => ({
                index: start + offset,
                line,
            }))
        })

        const topSpacerHeight = computed(() => visibleRange.value.start * LINE_HEIGHT)
        const bottomSpacerHeight = computed(() => {
            const remaining = props.outputLines.length - visibleRange.value.end
            return Math.max(0, remaining * LINE_HEIGHT)
        })

        const isNearBottom = () => {
            const el = terminalOutputRef.value
            if (!el) {
                return true
            }
            return el.scrollHeight - el.scrollTop - el.clientHeight <= LINE_HEIGHT * 2
        }

        const scrollToBottom = (force = false) => {
            if (!force && !stickToBottom.value) {
                return
            }
            nextTick(() => {
                nextTick(() => {
                    const el = terminalOutputRef.value
                    if (!el) {
                        return
                    }
                    el.scrollTop = el.scrollHeight
                    scrollTop.value = el.scrollTop
                    stickToBottom.value = true
                })
            })
        }

        const onScroll = () => {
            const el = terminalOutputRef.value
            if (!el) {
                return
            }
            scrollTop.value = el.scrollTop
            stickToBottom.value = isNearBottom()
        }

        const updateContainerHeight = () => {
            if (terminalOutputRef.value) {
                containerHeight.value = terminalOutputRef.value.clientHeight
            }
        }

        const lastOutputSignature = computed(() => {
            const lines = props.outputLines
            if (lines.length === 0) {
                return '0'
            }
            const last = lines[lines.length - 1]
            return `${lines.length}:${last.raw}:${last.html}`
        })

        watch(lastOutputSignature, (signature, prevSignature) => {
            if (signature === '0') {
                stickToBottom.value = true
                scrollToBottom(true)
                return
            }
            if (signature === prevSignature) {
                return
            }
            if (stickToBottom.value) {
                scrollToBottom()
            }
        })

        watch(() => props.status.isRunning, (isRunning) => {
            if (isRunning && stickToBottom.value) {
                scrollToBottom()
            }
        })

        onMounted(() => {
            updateContainerHeight()
            if (typeof ResizeObserver !== 'undefined' && terminalOutputRef.value) {
                resizeObserver = new ResizeObserver(updateContainerHeight)
                resizeObserver.observe(terminalOutputRef.value)
            }
        })

        onUnmounted(() => {
            if (resizeObserver) {
                resizeObserver.disconnect()
                resizeObserver = null
            }
        })

        expose({
            terminalOutputRef,
            scrollToBottom: () => scrollToBottom(true),
            resetScroll: () => {
                stickToBottom.value = true
                scrollToBottom(true)
            },
        })

        return {
            terminalOutputRef,
            bottomMarker,
            visibleLines,
            topSpacerHeight,
            bottomSpacerHeight,
            onScroll,
        }
    }
}
</script>

<style scoped>
.terminal-wrapper {
    display: flex;
    flex-direction: column;
    height: 100%;
}

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

.virtual-spacer {
    width: 100%;
    flex-shrink: 0;
}

.output-line {
    height: 20px;
    margin-bottom: 0;
    overflow: hidden;
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
