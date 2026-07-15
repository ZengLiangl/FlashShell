<template>
    <div class="panel-section project-section" :class="{ 'fullscreen': fullScreen }">
        <div class="section-header">
            <h3>项目列表</h3>
            <div class="header-actions icon-actions">
                <el-tooltip content="系统设置" placement="top">
                    <el-button size="small" circle @click="$emit('open-system-settings')">
                        <el-icon><Setting /></el-icon>
                    </el-button>
                </el-tooltip>
                <el-tooltip content="刷新" placement="top">
                    <el-button size="small" circle @click="$emit('refresh')">
                        <el-icon><Refresh /></el-icon>
                    </el-button>
                </el-tooltip>
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
    emits: ['refresh', 'select', 'open-system-settings']
}
</script>

<style scoped>
.panel-section {
    padding: 16px;
    border-bottom: 1px solid var(--app-border);
    display: flex;
    flex-direction: column;
    background: var(--app-panel-bg);
    color: var(--app-text);
    height: 100%;
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
    color: var(--app-text);
}

.header-actions {
    flex-shrink: 0;
}

.no-projects {
    text-align: center;
    color: var(--app-text-muted);
    padding: 20px;
}

.project-list {
    display: grid;
    grid-template-columns: repeat(4, 220px);
    justify-content: center;
    gap: 16px 40px;
    overflow-y: auto;
    overflow-x: hidden;
    flex: 1;
    min-height: 0;
    padding: 12px 32px 20px;
}

.project-item {
    padding: 12px;
    background: var(--app-card-bg);
    border-radius: 10px;
    border: 1px solid var(--app-card-border);
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
    border-color: var(--app-accent-color);
    box-shadow: 0 6px 18px var(--app-card-hover-shadow);
    transform: translateY(-2px);
}

.project-item.active {
    border-color: var(--app-accent-color);
    background: var(--app-card-active-bg);
    box-shadow: 0 6px 18px var(--app-card-hover-shadow);
}

.project-name {
    font-weight: 600;
    color: var(--app-text);
    margin-bottom: 4px;
}

.project-desc {
    font-size: 12px;
    color: var(--app-text-muted);
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
    background: var(--app-accent-bg);
    color: var(--app-accent-color);
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
