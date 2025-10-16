<template>
    <div class="terminal-header">
        <h3>终端输出</h3>
        <div class="terminal-actions">
            <div class="actions-left"></div>
            <div class="actions-right">
                <el-button v-if="showBack" size="small" type="primary" text @click="$emit('back')">
                    <el-icon>
                        <ArrowLeft />
                    </el-icon>
                    返回
                </el-button>
                <el-button size="small" @click="$emit('clear')">
                    <el-icon>
                        <Delete />
                    </el-icon>
                    清空
                </el-button>
                <el-button size="small" @click="$emit('refresh')">
                    <el-icon>
                        <Refresh />
                    </el-icon>
                    刷新
                </el-button>
            </div>
        </div>
    </div>
</template>

<script>
export default {
    name: 'TerminalHeader',
    props: {
        showBack: { type: Boolean, default: false }
    },
    emits: ['clear', 'refresh', 'back']
}
</script>

<style scoped>
.terminal-header {
    overflow-y: hidden;
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px;
    border-bottom: 1px solid #e4e7ed;
    background: #f5f7fa;
}

.terminal-header h3 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: #303133;
}

.terminal-actions {
    display: flex;
    gap: 8px;
}
</style>
