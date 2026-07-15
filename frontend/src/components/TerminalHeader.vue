<template>
    <div class="terminal-header">
        <h3>{{ title }}</h3>
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
                <el-tooltip content="搜索" placement="top">
                    <el-button size="small" @click="$emit('toggle-search')">
                        <el-icon><Search /></el-icon>
                    </el-button>
                </el-tooltip>
                <el-tooltip v-if="showBack" content="返回" placement="top">
                    <el-button size="small" type="primary" text @click="$emit('back')">
                        <el-icon><ArrowLeft /></el-icon>
                    </el-button>
                </el-tooltip>
                <el-tooltip content="清空" placement="top">
                    <el-button size="small" @click="$emit('clear')">
                        <el-icon><Delete /></el-icon>
                    </el-button>
                </el-tooltip>
                <el-tooltip content="刷新" placement="top">
                    <el-button size="small" @click="$emit('refresh')">
                        <el-icon><Refresh /></el-icon>
                    </el-button>
                </el-tooltip>
            </div>
        </div>
    </div>
</template>

<script>
import { ref, watch, nextTick } from 'vue'

export default {
    name: 'TerminalHeader',
    props: {
        title: { type: String, default: '终端输出' },
        showBack: { type: Boolean, default: false },
        searchVisible: { type: Boolean, default: false },
        searchQuery: { type: String, default: '' },
        matchSummary: { type: String, default: '' }
    },
    emits: ['clear', 'refresh', 'back', 'toggle-search', 'search-next', 'search-prev', 'close-search', 'update:searchQuery'],
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

        return { localQuery, searchInputRef }
    }
}
</script>

<style scoped>
.terminal-header {
    flex-shrink: 0;
    overflow-y: hidden;
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px;
    border-bottom: 1px solid var(--app-border, #e4e7ed);
    background: var(--app-panel-bg, #f5f7fa);
    gap: 12px;
}

.terminal-header h3 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--app-text, #303133);
    white-space: nowrap;
}

.terminal-actions {
    display: flex;
    gap: 8px;
    flex: 1;
    justify-content: flex-end;
    align-items: center;
    min-width: 0;
}

.actions-left {
    flex: 1;
    min-width: 0;
}

.search-bar {
    display: flex;
    align-items: center;
    gap: 8px;
}

.search-count {
    font-size: 12px;
    color: var(--app-text-secondary, #909399);
    white-space: nowrap;
    min-width: 2.5em;
}

.search-actions {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    padding: 0;
    border-radius: 0;
    background: transparent;
}

.search-sep {
    width: 1px;
    height: 12px;
    margin: 0 2px;
    background: color-mix(in srgb, var(--app-text-muted, #909399) 35%, transparent);
}

.search-icon-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    height: 26px;
    padding: 0;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--app-text-secondary, #909399);
    cursor: pointer;
    transition: color 0.15s ease, background 0.15s ease;
}

.search-icon-btn:hover {
    color: var(--app-accent-color, #409eff);
    background: color-mix(in srgb, var(--app-accent-color, #409eff) 14%, transparent);
}

.search-icon-btn:active {
    transform: translateY(0.5px);
}

.search-close:hover {
    color: #f56c6c;
    background: rgba(245, 108, 108, 0.12);
}
</style>
