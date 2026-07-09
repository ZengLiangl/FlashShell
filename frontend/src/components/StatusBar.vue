<template>
    <div class="status-bar">
        <div class="status-info">
            <div class="status-container">
                <transition name="status-fade" mode="out-in">
                    <el-tag v-if="status.isRunning" key="running" type="warning" size="small">
                        <el-icon>
                            <Loading />
                        </el-icon>
                        <span class="status-text">
                            执行中:
                            <span v-if="status.currentCommand"> - {{ status.currentCommand }}</span>
                        </span>
                    </el-tag>
                    <el-tag v-else key="ready" type="success" size="small">
                        <el-icon>
                            <Check />
                        </el-icon>
                        <span class="status-text">就绪</span>
                    </el-tag>
                </transition>
            </div>

            <transition name="project-fade">
                <el-tag v-if="selectedProject" size="small" type="info">
                    项目: {{ selectedProject.name }}
                </el-tag>
            </transition>
        </div>

        <div class="status-actions">
            <transition name="button-slide">
                <el-button v-if="status.isRunning" size="small" type="danger" @click="$emit('stop-all')">
                    停止执行
                </el-button>
            </transition>

            <span class="app-info">{{ appInfo }}</span>
        </div>
    </div>
</template>

<script>
export default {
    name: 'StatusBar',
    props: {
        status: { type: Object, required: true },
        selectedProject: { type: Object, default: null },
        appInfo: { type: String, default: 'Quick Cmd' }
    },
    emits: ['stop-all']
}
</script>

<style scoped>
.status-bar {
    flex-shrink: 0;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 16px;
    border-top: 1px solid var(--app-border);
    background: var(--app-panel-bg);
    color: var(--app-text);
    box-sizing: border-box;
}

.status-info {
    display: flex;
    align-items: center;
    gap: 12px;
}

.status-container {
    min-width: 180px;
}

.status-text {
    margin-left: 6px;
}

.status-actions {
    display: flex;
    align-items: center;
    gap: 12px;
}

.app-info {
    color: var(--app-text-muted);
    font-size: 12px;
}
</style>
