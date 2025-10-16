<template>
    <div class="panel-section project-section" :class="{ 'fullscreen': fullScreen }">
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
                <div class="card-header">
                    <div class="avatar-icon">
                        <el-icon>
                            <Folder />
                        </el-icon>
                    </div>
                    <div class="header-meta">
                        <div class="project-name">{{ project.name }}</div>
                        <div class="project-desc">{{ project.description }}</div>
                    </div>
                </div>
                <div class="card-footer">
                    <el-tag size="small" type="info" effect="plain">{{ (project.subprojects || []).length }}
                        子项目</el-tag>
                    <el-tag v-if="project.workdir" size="small" effect="light">工作目录</el-tag>
                </div>
            </div>
        </div>
    </div>
</template>

<script>
export default {
    name: 'ProjectList',
    props: {
        projects: { type: Array, required: true },
        selectedProjectName: { type: String, default: '' },
        fullScreen: { type: Boolean, default: false }
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

.project-section.fullscreen {
    max-height: none;
    border-bottom: none;
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
    display: grid;
    grid-template-columns: repeat(4, 220px);
    justify-content: center;
    gap: 16px 40px;
    /* 行间距 24, 列间距 28 */
    overflow-y: auto;
    overflow-x: hidden;
    flex: 1;
    min-height: 0;
    padding: 12px 32px 20px;
    /* 增加左右内边距以提升可视留白 */
}

.project-item {
    padding: 12px;
    background: white;
    border-radius: 10px;
    border: 1px solid #ebeef5;
    cursor: pointer;
    transition: all 0.2s ease-in-out;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
    width: 100%;
    aspect-ratio: 1 / 1;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    overflow: hidden;
}

.project-item:hover {
    border-color: #409eff;
    box-shadow: 0 6px 18px rgba(64, 158, 255, 0.15);
    transform: translateY(-2px);
}

.project-item.active {
    border-color: #409eff;
    background: #f3f9ff;
    box-shadow: 0 6px 18px rgba(64, 158, 255, 0.2);
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

.card-header {
    display: flex;
    align-items: center;
    gap: 10px;
}

.avatar-icon {
    width: 36px;
    height: 36px;
    border-radius: 8px;
    background: #ecf5ff;
    color: #409eff;
    display: flex;
    align-items: center;
    justify-content: center;
}

.header-meta {
    display: flex;
    flex-direction: column;
    gap: 2px;
}

.card-footer {
    display: flex;
    gap: 6px;
    margin-top: 10px;
}
</style>
