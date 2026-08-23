<template>
    <div class="terminal-header term-toolbar" @dblclick="onChromeTitleDblActivate" @mousedown="onChromeTitlePointerDown">
        <div class="t-title">
            <StatusDot :state="statusRunning ? 'on' : 'off'" />
            <span>{{ title }}</span>
        </div>
        <div class="terminal-actions">
            <div class="actions-left">
                <div v-if="searchVisible" class="search-bar">
                    <el-input
                        ref="searchInputRef"
                        v-model="localQuery"
                        size="small"
                        placeholder="搜索 (Enter 下一个, Shift+Enter 上一个)"
                        clearable
                        @keydown.enter.exact.prevent="$emit('search-next')"
                        @keydown.enter.shift.exact.prevent="$emit('search-prev')"
                        @keydown.esc.prevent="$emit('close-search')"
                    />
                    <span v-if="localQuery" class="search-count">{{ matchSummary }}</span>
                    <div class="search-actions">
                        <button type="button" class="search-icon-btn" title="上一个" @click="$emit('search-prev')">
                            <el-icon :size="14"><CaretTop /></el-icon>
                        </button>
                        <button type="button" class="search-icon-btn" title="下一个" @click="$emit('search-next')">
                            <el-icon :size="14"><CaretBottom /></el-icon>
                        </button>
                        <span class="search-sep" aria-hidden="true"></span>
                        <button type="button" class="search-icon-btn search-close" title="关闭" @click="$emit('close-search')">
                            <el-icon :size="14"><Close /></el-icon>
                        </button>
                    </div>
                </div>
            </div>
            <div class="actions-right icon-actions">
                <AppIconBtn v-if="showSearchToggle" title="搜索" @click="$emit('toggle-search')">
                    <el-icon :size="14"><Search /></el-icon>
                </AppIconBtn>
                <AppIconBtn v-if="showBack" title="返回" @click="$emit('back')">
                    <el-icon :size="14"><ArrowLeft /></el-icon>
                </AppIconBtn>
                <template v-if="showInlineActions">
                    <AppIconBtn title="清空" @click="$emit('clear')">
                        <el-icon :size="14"><Delete /></el-icon>
                    </AppIconBtn>
                    <AppIconBtn title="刷新" @click="$emit('refresh')">
                        <el-icon :size="14"><Refresh /></el-icon>
                    </AppIconBtn>
                </template>
                <template v-if="showChrome">
                    <span class="chrome-sep" aria-hidden="true" />
                    <ModeSwitcher
                        v-if="hasProjects || hasTask"
                        compact
                        float-align="end"
                        :model-value="activeView"
                        :has-projects="hasProjects"
                        :has-machines="hasMachines"
                        :has-task="hasTask"
                        :task-running="taskRunning"
                        :connected-count="connectedCount"
                        :projects="projects"
                        :selected-project-name="selectedProjectName"
                        :sessions="sessions"
                        :active-session-id="activeSessionId"
                        @change="(v) => $emit('change-view', v)"
                        @select-project="(p) => $emit('select-project', p)"
                        @focus-session="(id) => $emit('focus-session', id)"
                    />
                    <AppChromeIcons />
                </template>
            </div>
        </div>
    </div>
</template>

<script>
import { ref, watch, nextTick } from 'vue'
import ModeSwitcher from './ModeSwitcher.vue'
import AppChromeIcons from './AppChromeIcons.vue'
import { AppIconBtn, StatusDot } from './ui'
import { onChromeTitleDblActivate, onChromeTitlePointerDown } from '../utils/windowChrome'

export default {
    name: 'TerminalHeader',
    components: { ModeSwitcher, AppChromeIcons, AppIconBtn, StatusDot },
    props: {
        title: { type: String, default: '终端输出' },
        statusRunning: { type: Boolean, default: false },
        showBack: { type: Boolean, default: false },
        searchVisible: { type: Boolean, default: false },
        searchQuery: { type: String, default: '' },
        matchSummary: { type: String, default: '' },
        showChrome: { type: Boolean, default: false },
        showSearchToggle: { type: Boolean, default: true },
        showInlineActions: { type: Boolean, default: true },
        hasTask: { type: Boolean, default: false },
        hasProjects: { type: Boolean, default: false },
        hasMachines: { type: Boolean, default: false },
        taskRunning: { type: Boolean, default: false },
        connectedCount: { type: Number, default: 0 },
        projects: { type: Array, default: () => [] },
        selectedProjectName: { type: String, default: '' },
        activeView: { type: String, default: 'task' },
        sessions: { type: Array, default: () => [] },
        activeSessionId: { type: String, default: '' },
    },
    emits: [
        'clear', 'refresh', 'back', 'toggle-search', 'search-next', 'search-prev', 'close-search', 'update:searchQuery',
        'change-view', 'select-project', 'focus-session',
    ],
    setup(props, { emit }) {
        const localQuery = ref(props.searchQuery)
        const searchInputRef = ref(null)

        watch(() => props.searchQuery, (v) => { localQuery.value = v })
        watch(localQuery, (v) => emit('update:searchQuery', v))
        watch(() => props.searchVisible, async (visible) => {
            if (visible) {
                await nextTick()
                searchInputRef.value?.focus?.()
            }
        })

        return { localQuery, searchInputRef, onChromeTitleDblActivate, onChromeTitlePointerDown }
    }
}
</script>

<style scoped>
.terminal-header.term-toolbar {
    flex-shrink: 0;
    overflow: visible;
    position: relative;
    z-index: 20;
    display: flex;
    align-items: center;
    gap: 8px;
    height: 38px;
    min-height: 38px;
    padding: 0 10px;
    box-sizing: border-box;
    background: color-mix(in oklch, var(--term-bg) 96%, white);
    border-bottom: 1px solid var(--term-border);
    color: var(--term-dim);
}

.t-title {
    display: flex;
    align-items: center;
    gap: 7px;
    font-family: var(--font-mono);
    font-size: 12.5px;
    color: var(--term-fg);
    white-space: nowrap;
    flex-shrink: 0;
}

.terminal-actions {
    display: flex;
    gap: 4px;
    flex: 1;
    justify-content: flex-end;
    align-items: center;
    min-width: 0;
}

.actions-left { flex: 1; min-width: 0; }

.actions-right {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    flex-shrink: 0;
}

.actions-right :deep(.icon-btn) {
    color: var(--term-dim);
    width: 26px;
    height: 26px;
}

.actions-right :deep(.icon-btn:hover) {
    background: var(--term-hover);
    color: var(--term-fg);
}

.chrome-sep {
    width: 1px;
    height: 14px;
    margin: 0 4px;
    background: var(--term-border);
}

.search-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 0 8px;
    height: 30px;
}

.search-count {
    font-family: var(--font-mono);
    font-size: 12px;
    color: var(--muted);
}

.search-icon-btn {
    width: 24px;
    height: 24px;
    color: var(--fg-2);
}

.search-icon-btn:hover {
    background: var(--surface-2);
}
</style>
