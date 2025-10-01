<template>
    <div class="panel-section project-section">
        <div class="section-header">
            <h3>项目列表</h3>
            <div>
                <el-button size="small" @click="$emit('refresh')">
                    <el-icon>
                        <Refresh />
                    </el-icon>
                </el-button>
            </div>
        </div>
        <div v-if="projects.length === 0" class="no-projects">
            <p>暂无项目配置</p>
            <p>项目数量: {{ projects.length }}</p>
        </div>
        <div v-else class="project-list">
            <div v-for="project in projects" :key="project.name" class="project-item"
                :class="{ active: selectedProjectName === project.name }" @click="$emit('select', project)">
                <div class="project-name">{{ project.name }}</div>
                <div class="project-desc">{{ project.description }}</div>
            </div>
        </div>
    </div>
</template>

<script>
export default {
    name: 'ProjectList',
    props: {
        projects: { type: Array, required: true },
        selectedProjectName: { type: String, default: '' }
    },
    emits: ['refresh', 'select']
}
</script>

<style scoped>
.panel-section {
    padding: 16px;
    border-bottom: 1px solid #e4e7ed;
    display: flex;
    flex-direction: column;
}

.project-section {
    flex-shrink: 0;
    max-height: 40vh;
}

.section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
}

.section-header h3 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: #303133;
}

.project-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    overflow-y: auto;
    overflow-x: hidden;
    flex: 1;
    min-height: 0;
}

.project-item {
    padding: 12px;
    background: white;
    border-radius: 6px;
    border: 1px solid #e4e7ed;
    cursor: pointer;
    transition: all 0.2s;
}

.project-item:hover {
    border-color: #409eff;
    box-shadow: 0 2px 8px rgba(64, 158, 255, 0.1);
}

.project-item.active {
    border-color: #409eff;
    background: #ecf5ff;
}

.project-name {
    font-weight: 600;
    color: #303133;
    margin-bottom: 4px;
}

.project-desc {
    font-size: 12px;
    color: #909399;
}
</style>
