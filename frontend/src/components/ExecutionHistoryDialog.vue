<template>
    <el-dialog v-model="visibleProxy" title="执行历史" width="80%" :before-close="handleClose">
        <el-table :data="logs" v-loading="loading" height="360">
            <el-table-column prop="startedAt" label="时间" width="180" />
            <el-table-column prop="project" label="项目" width="120" />
            <el-table-column prop="subProject" label="子项目" width="140" />
            <el-table-column prop="fileName" label="文件名" min-width="220" />
            <el-table-column prop="size" label="大小" width="100">
                <template #default="{ row }">{{ formatSize(row.size) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="180" fixed="right">
                <template #default="{ row }">
                    <el-button size="small" @click="viewLog(row)">查看</el-button>
                    <el-button size="small" @click="openLog(row)">打开</el-button>
                </template>
            </el-table-column>
        </el-table>

        <el-dialog v-model="viewerVisible" title="日志内容" width="70%" append-to-body>
            <pre class="log-viewer">{{ logContent }}</pre>
        </el-dialog>
    </el-dialog>
</template>

<script>
import { ref, watch, computed } from 'vue'
import { ElMessage } from 'element-plus'
import * as App from '../../wailsjs/go/app/App'

export default {
    name: 'ExecutionHistoryDialog',
    props: { modelValue: { type: Boolean, default: false } },
    emits: ['update:modelValue'],
    setup(props, { emit }) {
        const logs = ref([])
        const loading = ref(false)
        const viewerVisible = ref(false)
        const logContent = ref('')

        const visibleProxy = computed({
            get: () => props.modelValue,
            set: (v) => emit('update:modelValue', v)
        })

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

        watch(() => props.modelValue, (open) => { if (open) load() })

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

        const handleClose = () => { visibleProxy.value = false }

        return { visibleProxy, logs, loading, viewerVisible, logContent, formatSize, viewLog, openLog, handleClose }
    }
}
</script>

<style scoped>
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
