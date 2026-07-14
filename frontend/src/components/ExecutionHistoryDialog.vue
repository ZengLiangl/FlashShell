<template>
    <div class="history-panel" :class="{ embedded }">
        <el-table :data="logs" v-loading="loading" class="history-table" max-height="480">
            <el-table-column prop="startedAt" label="时间" width="180" />
            <el-table-column prop="project" label="项目" width="120" />
            <el-table-column prop="subProject" label="子项目" width="140" />
            <el-table-column prop="fileName" label="文件名" min-width="180" />
            <el-table-column prop="size" label="大小" width="100">
                <template #default="{ row }">{{ formatSize(row.size) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="160" fixed="right">
                <template #default="{ row }">
                    <el-button size="small" @click="viewLog(row)">查看</el-button>
                    <el-button size="small" @click="openLog(row)">打开</el-button>
                </template>
            </el-table-column>
        </el-table>

        <el-dialog v-model="viewerVisible" title="日志内容" width="70%" append-to-body>
            <pre class="log-viewer">{{ logContent }}</pre>
        </el-dialog>
    </div>
</template>

<script>
import { ref, watch, computed } from 'vue'
import { ElMessage } from 'element-plus'
import * as App from '../../wailsjs/go/app/App'

export default {
    name: 'ExecutionHistoryDialog',
    props: {
        modelValue: { type: Boolean, default: false },
        embedded: { type: Boolean, default: false },
        active: { type: Boolean, default: false },
    },
    emits: ['update:modelValue'],
    setup(props) {
        const logs = ref([])
        const loading = ref(false)
        const viewerVisible = ref(false)
        const logContent = ref('')

        const load = async () => {
            loading.value = true
            try {
                logs.value = await App.GetExecutionLogs(100)
            } catch (e) {
                ElMessage.error(`加载历史失败: ${e}`)
            } finally {
                loading.value = false
            }
        }

        watch(() => props.modelValue, (open) => {
            if (!props.embedded && open) load()
        })
        watch(() => props.active, (open) => {
            if (props.embedded && open) load()
        }, { immediate: true })

        const formatSize = (size) => {
            if (size < 1024) return `${size} B`
            if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
            return `${(size / 1024 / 1024).toFixed(1)} MB`
        }

        const viewLog = async (row) => {
            try {
                logContent.value = await App.ReadExecutionLog(row.fileName)
                viewerVisible.value = true
            } catch (e) {
                ElMessage.error(`读取日志失败: ${e}`)
            }
        }

        const openLog = async (row) => {
            try {
                await App.OpenExecutionLog(row.fileName)
            } catch (e) {
                ElMessage.error(`打开日志失败: ${e}`)
            }
        }

        return {
            embedded: computed(() => props.embedded),
            logs,
            loading,
            viewerVisible,
            logContent,
            formatSize,
            viewLog,
            openLog,
        }
    }
}
</script>

<style scoped>
.history-panel {
    height: min(56vh, 480px);
    min-height: 320px;
}

.history-panel.embedded {
    height: 100%;
}

.history-table {
    width: 100%;
}

.log-viewer {
    max-height: 60vh;
    overflow: auto;
    background: var(--terminal-bg, #1e1e1e);
    color: var(--terminal-fg, #d4d4d4);
    padding: 12px;
    border-radius: 6px;
    white-space: pre-wrap;
    font-family: Consolas, Monaco, monospace;
    font-size: 12px;
}
</style>
