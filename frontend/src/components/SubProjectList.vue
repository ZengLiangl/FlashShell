<template>
    <div class="panel-section subproject-section task-left-panel">
        <div class="section-header tl-head" @dblclick="onChromeTitleDblActivate" @mousedown="onChromeTitlePointerDown">
            <div ref="tabsWrapRef" class="project-tabs-wrap">
                <button
                    v-if="tabsScrollable"
                    type="button"
                    class="tabs-nav tabs-nav-left"
                    :disabled="!tabsCanScrollLeft"
                    title="向左滚动项目"
                    aria-label="向左滚动项目"
                    @click="scrollTabsBy(-1)"
                >
                    <el-icon><ArrowLeft /></el-icon>
                </button>
                <div
                    ref="tabsRef"
                    class="project-tabs"
                    :class="{
                        'is-scrollable': tabsScrollable,
                        'is-scrolling': tabsScrolling,
                        'fade-left': tabsCanScrollLeft,
                        'fade-right': tabsCanScrollRight,
                    }"
                    role="tablist"
                    aria-label="切换项目"
                    @wheel="onTabsWheel"
                    @pointerdown="onTabsPointerDown"
                    @scroll.passive="updateTabsScrollState"
                    @click.capture="onTabsClickCapture"
                    @mousedown.stop
                    @dblclick.stop
                >
                    <template v-for="(project, index) in projects" :key="project.name">
                        <span v-if="index > 0" class="project-tab-sep" aria-hidden="true">｜</span>
                        <button
                            type="button"
                            role="tab"
                            class="project-tab"
                            :class="{ active: project.name === selectedProject?.name, 'is-locked': status.isRunning }"
                            :aria-selected="project.name === selectedProject?.name"
                            :aria-disabled="status.isRunning ? 'true' : 'false'"
                            :title="project.name"
                            @mousedown.prevent
                            @click="onSelectProject(project.name)"
                        >
                            {{ project.name }}
                        </button>
                    </template>
                </div>
                <button
                    v-if="tabsScrollable"
                    type="button"
                    class="tabs-nav tabs-nav-right"
                    :disabled="!tabsCanScrollRight"
                    title="向右滚动项目"
                    aria-label="向右滚动项目"
                    @click="scrollTabsBy(1)"
                >
                    <el-icon><ArrowRight /></el-icon>
                </button>
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
    data() {
        return {
            // 项目标签横向滚动的状态：只在溢出时才显示边缘渐隐和抓手光标
            tabsScrollable: false,
            tabsCanScrollLeft: false,
            tabsCanScrollRight: false,
            tabsScrolling: false
        }
    },
    watch: {
        selectedProject: {
            handler() {
                this.$nextTick(() => this.revealActiveTab())
            }
        },
        projects: {
            handler() {
                this.$nextTick(() => {
                    this.updateTabsScrollState()
                    this.revealActiveTab('auto')
                })
            }
        }
    },
    created() {
        // 不要放进 data：function ref 会反复赋值，放响应式里会把展开交互打挂
        this.blockRefs = Object.create(null)
        this.cmdRefs = Object.create(null)
        // 拖拽滚动的临时状态，同样不放进响应式
        this.tabDrag = null
        this.tabsSuppressClick = false
        this.tabsResizeObserver = null
        this.tabsHasLayout = false
    },
    mounted() {
        this.updateTabsScrollState()
        this.$nextTick(() => {
            this.updateTabsScrollState()
            this.revealActiveTab('auto')
        })
        if (this.$refs.tabsRef && typeof ResizeObserver !== 'undefined') {
            this.tabsResizeObserver = new ResizeObserver(() => this.updateTabsScrollState())
            this.tabsResizeObserver.observe(this.$refs.tabsRef)
        }
        window.addEventListener('resize', this.updateTabsScrollState)
    },
    beforeUnmount() {
        this.tabsResizeObserver?.disconnect()
        this.tabsResizeObserver = null
        window.removeEventListener('resize', this.updateTabsScrollState)
        this.detachTabsPointerTracking()
    },
    methods: {
        /* ---- 项目标签横向滚动：隐藏原生滚动条，用滚轮/拖拽/自动定位代替 ---- */
        updateTabsScrollState() {
            const el = this.$refs.tabsRef
            const wrap = this.$refs.tabsWrapRef
            if (!el) return
            const wasHidden = !this.tabsHasLayout
            this.tabsHasLayout = el.clientWidth > 0
            if (wasHidden && this.tabsHasLayout) this.revealActiveTab('auto')
            // 用容器宽度判断是否需要箭头：箭头本身会占宽度，不能拿标签条自身宽度判断，
            // 否则面板变宽后箭头会“赖着不走”，明明放得下还要滚。
            const available = wrap ? wrap.clientWidth : el.clientWidth
            const scrollable = el.scrollWidth > available + 1
            const max = Math.max(0, el.scrollWidth - el.clientWidth)
            this.tabsScrollable = scrollable
            this.tabsCanScrollLeft = scrollable && el.scrollLeft > 1
            this.tabsCanScrollRight = scrollable && el.scrollLeft < max - 1
        },
        onTabsWheel(event) {
            const el = this.$refs.tabsRef
            if (!el || event.ctrlKey) return
            const max = el.scrollWidth - el.clientWidth
            if (max <= 1) return
            // 触控板横向手势交给浏览器原生滚动，保留惯性
            if (Math.abs(event.deltaX) > Math.abs(event.deltaY)) return
            const scale = event.deltaMode === 1 ? 16 : event.deltaMode === 2 ? el.clientWidth : 1
            const next = Math.max(0, Math.min(max, el.scrollLeft + event.deltaY * scale))
            if (next === el.scrollLeft) return
            event.preventDefault()
            el.scrollLeft = next
        },
        scrollTabsBy(direction) {
            const el = this.$refs.tabsRef
            if (!el) return
            const max = el.scrollWidth - el.clientWidth
            if (max <= 1) return
            const step = Math.max(72, Math.round(el.clientWidth * 0.7))
            const target = Math.max(0, Math.min(max, el.scrollLeft + direction * step))
            if (Math.abs(target - el.scrollLeft) < 1) return
            if (typeof el.scrollTo === 'function') {
                el.scrollTo({ left: target, behavior: 'smooth' })
            } else {
                el.scrollLeft = target
            }
        },
        onTabsPointerDown(event) {
            if (event.button !== 0 || event.pointerType === 'touch') return
            const el = this.$refs.tabsRef
            if (!el || el.scrollWidth - el.clientWidth <= 1) return
            // 新的一次按下：清掉上一次可能残留的点击抑制
            this.tabsSuppressClick = false
            this.tabDrag = {
                pointerId: event.pointerId,
                startX: event.clientX,
                startScroll: el.scrollLeft,
                moved: false
            }
            window.addEventListener('pointermove', this.onTabsPointerMove, true)
            window.addEventListener('pointerup', this.onTabsPointerUp, true)
            window.addEventListener('pointercancel', this.onTabsPointerUp, true)
            window.addEventListener('blur', this.onTabsPointerUp, true)
        },
        onTabsPointerMove(event) {
            const drag = this.tabDrag
            if (!drag || drag.pointerId !== event.pointerId) return
            const el = this.$refs.tabsRef
            if (!el) return
            const dx = event.clientX - drag.startX
            if (!drag.moved) {
                if (Math.abs(dx) < 4) return
                drag.moved = true
                this.tabsScrolling = true
                // 拖动滚动后不要顺手把手指停下的项目选中
                this.tabsSuppressClick = true
                event.preventDefault()
                // 指针移出标签条也继续跟随
                try {
                    el.setPointerCapture(event.pointerId)
                } catch (e) {
                    /* ignore */
                }
            }
            el.scrollLeft = drag.startScroll - dx
        },
        onTabsPointerUp(event) {
            const drag = this.tabDrag
            if (!drag) return
            if (event?.pointerId !== undefined && event.pointerId !== drag.pointerId) return
            const el = this.$refs.tabsRef
            if (el && typeof el.hasPointerCapture === 'function' && el.hasPointerCapture(drag.pointerId)) {
                try {
                    el.releasePointerCapture(drag.pointerId)
                } catch (e) {
                    /* ignore */
                }
            }
            this.tabDrag = null
            this.detachTabsPointerTracking()
            this.tabsScrolling = false
            if (drag.moved) {
                // click 在 pointerup 之后同轮触发，延后一帧再放开抑制
                window.setTimeout(() => {
                    this.tabsSuppressClick = false
                }, 0)
            }
        },
        detachTabsPointerTracking() {
            window.removeEventListener('pointermove', this.onTabsPointerMove, true)
            window.removeEventListener('pointerup', this.onTabsPointerUp, true)
            window.removeEventListener('pointercancel', this.onTabsPointerUp, true)
            window.removeEventListener('blur', this.onTabsPointerUp, true)
        },
        onTabsClickCapture(event) {
            if (!this.tabsSuppressClick) return
            event.preventDefault()
            event.stopPropagation()
        },
        revealActiveTab(behavior = 'smooth') {
            const el = this.$refs.tabsRef
            if (!el || el.clientWidth <= 0) return
            const active = el.querySelector('.project-tab.active')
            if (!active) return
            const box = el.getBoundingClientRect()
            const tab = active.getBoundingClientRect()
            const pad = 8
            let target = null
            if (tab.left < box.left + pad) target = el.scrollLeft + (tab.left - box.left) - pad
            else if (tab.right > box.right - pad) target = el.scrollLeft + (tab.right - box.right) + pad
            if (target === null) return
            target = Math.max(0, Math.min(target, el.scrollWidth - el.clientWidth))
            if (Math.abs(target - el.scrollLeft) < 1) return
            if (typeof el.scrollTo === 'function') {
                el.scrollTo({ left: target, behavior })
            } else {
                el.scrollLeft = target
            }
        },
        /* ---- 子项目/命令列表 ---- */
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
            if (!name || name === this.selectedProject?.name) return
            if (this.status?.isRunning) {
                ElMessage.warning('任务执行中，请先停止后再切换项目')
                return
            }
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

.project-tabs-wrap {
    position: relative;
    display: flex;
    align-items: center;
    gap: 2px;
    flex: 1;
    min-width: 0;
    /* 这条热区要能拖动/滚动标签，全局窗口拖拽会让位给标签条 */
    --wails-draggable: no-drag;
}

/* 左右滚动按钮：只有标签溢出时才出现，明确告诉用户还能左右翻 */
.tabs-nav {
    flex: 0 0 auto;
    width: 20px;
    height: 24px;
    padding: 0;
    display: grid;
    place-items: center;
    border: none;
    border-radius: 6px;
    background: none;
    color: var(--muted);
    cursor: pointer;
    transition: color 0.12s ease, background-color 0.12s ease, opacity 0.12s ease;
}

.tabs-nav:hover:not(:disabled) {
    color: var(--fg);
    background: color-mix(in oklch, var(--fg) 8%, transparent);
}

.tabs-nav:active:not(:disabled) {
    background: color-mix(in oklch, var(--fg) 12%, transparent);
}

.tabs-nav:disabled {
    opacity: 0.35;
    cursor: default;
}

.tabs-nav:focus-visible {
    outline: 1px solid var(--accent);
    outline-offset: -1px;
}

.tabs-nav .el-icon {
    font-size: 13px;
}

.project-tabs {
    display: flex;
    align-items: center;
    gap: 0;
    min-width: 0;
    flex: 1;
    overflow-x: auto;
    overflow-y: hidden;
    overscroll-behavior-x: contain;
    /* 隐藏原生滚动条：它会压在项目名上并抢走点击 */
    scrollbar-width: none;
    -ms-overflow-style: none;
    -webkit-user-select: none;
    user-select: none;
}

/* 左右还有未显示的项目时，用渐隐提示边缘被裁切 */
.project-tabs.fade-left {
    -webkit-mask-image: linear-gradient(to right, transparent 0, #000 16px);
    mask-image: linear-gradient(to right, transparent 0, #000 16px);
}

.project-tabs.fade-right {
    -webkit-mask-image: linear-gradient(to left, transparent 0, #000 16px);
    mask-image: linear-gradient(to left, transparent 0, #000 16px);
}

.project-tabs.fade-left.fade-right {
    -webkit-mask-image: linear-gradient(
        to right,
        transparent 0,
        #000 16px,
        #000 calc(100% - 16px),
        transparent 100%
    );
    mask-image: linear-gradient(
        to right,
        transparent 0,
        #000 16px,
        #000 calc(100% - 16px),
        transparent 100%
    );
}

.project-tabs::-webkit-scrollbar {
    display: none;
    width: 0;
    height: 0;
}

.project-tabs.is-scrollable {
    cursor: grab;
}

.project-tabs.is-scrollable.is-scrolling,
.project-tabs.is-scrolling .project-tab {
    cursor: grabbing;
}

.project-tab {
    flex: 0 0 auto;
    max-width: 120px;
    min-height: 24px;
    padding: 4px 6px;
    border: none;
    border-radius: 6px;
    background: none;
    color: var(--muted);
    font-size: 13px;
    font-weight: 500;
    line-height: 1.3;
    cursor: pointer;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    transition: color 0.12s ease, background-color 0.12s ease;
    -webkit-user-select: none;
    user-select: none;
}

.project-tab:hover:not(.is-locked) {
    color: var(--fg);
    background: color-mix(in oklch, var(--fg) 7%, transparent);
}

.project-tab.active {
    color: var(--accent);
    font-weight: 600;
}

.project-tab.active:hover:not(.is-locked) {
    background: color-mix(in oklch, var(--accent) 12%, transparent);
}

.project-tab.is-locked {
    cursor: not-allowed;
    opacity: 0.65;
}

.project-tab.is-locked:hover {
    background: none;
    color: var(--muted);
}

.project-tab.is-locked.active {
    color: var(--accent);
}

.project-tab:focus-visible {
    outline: 1px solid var(--accent);
    outline-offset: -1px;
}

.project-tab-sep {
    flex: 0 0 auto;
    color: var(--border);
    font-size: 12px;
    user-select: none;
    padding: 0 1px;
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
