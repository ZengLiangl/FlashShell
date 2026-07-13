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
                    <el-button size="small" text @click="$emit('search-prev')">上一个</el-button>
                    <el-button size="small" text @click="$emit('search-next')">下一个</el-button>
                    <el-button size="small" text @click="$emit('close-search')">关闭</el-button>
                </div>
            </div>
            <div class="actions-right">
                <el-button size="small" @click="$emit('toggle-search')">
                    <el-icon><Search /></el-icon>
                    搜索
                </el-button>
                <el-button v-if="showBack" size="small" type="primary" text @click="$emit('back')">
                    <el-icon><ArrowLeft /></el-icon>
                    返回
                </el-button>
                <el-button size="small" @click="$emit('clear')">
                    <el-icon><Delete /></el-icon>
                    清空
                </el-button>
                <el-button size="small" @click="$emit('refresh')">
                    <el-icon><Refresh /></el-icon>
                    刷新
                </el-button>
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
}
</style>
