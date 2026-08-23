<template>
    <div class="terminal-wrapper">
        <div v-if="showInlineActions" class="terminal-inline-actions icon-actions icon-actions--sm">
            <el-tooltip content="清空" placement="bottom">
                <el-button size="small" circle @click="$emit('clear')">
                    <el-icon><Delete /></el-icon>
                </el-button>
            </el-tooltip>
            <el-tooltip content="刷新" placement="bottom">
                <el-button size="small" circle @click="$emit('refresh')">
                    <el-icon><Refresh /></el-icon>
                </el-button>
            </el-tooltip>
        </div>

        <transition name="progress-slide" appear>
            <div v-if="remoteFailure && !status.isRunning" class="failure-banner">
                <span class="failure-text">远程任务失败：{{ remoteFailure.machineName }}</span>
                <el-button size="small" type="warning" @click="$emit('open-failure-shell')">
                    进入 Shell
                </el-button>
            </div>
        </transition>

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
                }" v-html="renderLineHtml(item)">
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
        progressStatus: { type: String, required: true },
        remoteFailure: { type: Object, default: null },
        searchQuery: { type: String, default: '' },
        activeMatchIndex: { type: Number, default: -1 },
        showInlineActions: { type: Boolean, default: false },
    },
    emits: ['search-matches', 'open-failure-shell', 'clear', 'refresh'],
    setup(props, { expose, emit }) {
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

        const matchLineIndices = computed(() => {
            const q = props.searchQuery.trim().toLowerCase()
            if (!q) return []
            const indices = []
            props.outputLines.forEach((line, index) => {
                const text = (line.raw || line.text || '').toLowerCase()
                if (text.includes(q)) indices.push(index)
            })
            return indices
        })

        const escapeRegExp = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

        const renderLineHtml = (item) => {
            const html = item.line.html || ''
            const q = props.searchQuery.trim()
            if (!q) return html
            const text = item.line.raw || item.line.text || ''
            if (!text.toLowerCase().includes(q.toLowerCase())) return html

            const regex = new RegExp(`(${escapeRegExp(q)})`, 'gi')
            const plain = text.replace(/</g, '&lt;').replace(/>/g, '&gt;')
            const highlighted = plain.replace(regex, '<mark class="search-highlight">$1</mark>')
            const isActive = matchLineIndices.value[props.activeMatchIndex] === item.index
            return isActive ? `<span class="search-active-line">${highlighted}</span>` : highlighted
        }

        const scrollToLine = (lineIndex) => {
            const el = terminalOutputRef.value
            if (!el || lineIndex < 0) return
            el.scrollTop = Math.max(0, lineIndex * LINE_HEIGHT - containerHeight.value / 2)
            scrollTop.value = el.scrollTop
            stickToBottom.value = false
        }

        watch(matchLineIndices, (indices) => {
            emit('search-matches', indices)
        }, { immediate: true })

        watch(() => props.activeMatchIndex, (idx) => {
            const lineIndex = matchLineIndices.value[idx]
            if (lineIndex !== undefined) scrollToLine(lineIndex)
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
            renderLineHtml,
        }
    }
}
</script>

<style scoped>
.terminal-wrapper {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    overflow: hidden;
    background: var(--term-bg);
    position: relative;
}

.terminal-inline-actions {
    position: absolute;
    top: 8px;
    right: 12px;
    z-index: 5;
    display: flex;
    gap: 3px;
    padding: 0;
    border-radius: 0;
    background: transparent;
    backdrop-filter: none;
    box-shadow: none;
}

.terminal-output {
    flex: 1;
    min-height: 0;
    margin: 0;
    padding: 12px 16px;
    background: var(--term-bg);
    color: var(--term-fg);
    font-family: var(--font-mono);
    font-size: 13.5px;
    line-height: 1.6;
    overflow-y: auto;
    white-space: pre-wrap;
    border: none;
    border-radius: 0;
    box-shadow: none;
}

.error-line {
    color: var(--term-red);
}

.success-line {
    color: var(--term-green);
}

.progress-line {
    color: var(--term-cyan);
    font-weight: 500;
}

.empty-output {
    color: var(--term-dim);
}

.failure-banner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 8px 12px;
    margin-bottom: 8px;
    border: 1px solid color-mix(in srgb, var(--el-color-warning) 45%, transparent);
    border-radius: 6px;
    background: color-mix(in srgb, var(--el-color-warning) 12%, transparent);
}

.failure-text {
    font-size: 13px;
    color: var(--app-text);
}

.progress-section {
    flex-shrink: 0;
    padding: 12px 16px;
    background: var(--app-panel-bg);
    border-bottom: 1px solid var(--app-border);
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
    color: var(--app-text);
    font-size: 14px;
}

.current-command {
    font-size: 12px;
    color: var(--app-text-secondary);
}

.progress-stats {
    font-size: 12px;
    color: var(--app-text-muted);
    font-weight: 500;
}

.execution-progress {
    margin: 0;
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
    min-height: 1.6em;
}

:deep(.search-active-line) {
    outline: 1px solid var(--term-amber);
}
</style>
