<template>
  <div class="home-page">
    <!-- 任务模式 -->
    <section class="home-section">
      <div class="section-header">
        <div class="section-title">
          <el-icon class="section-icon task-icon">
            <Folder />
          </el-icon>
          <div class="section-text">
            <h3>任务模式</h3>
            <span class="section-desc">执行预设的子项目与命令流程</span>
          </div>
        </div>
        <div class="header-actions">
          <el-button size="small" @click="$emit('open-execution-history')">执行历史</el-button>
          <el-button size="small" @click="handleRefresh">
            <el-icon>
              <Refresh />
            </el-icon>
          </el-button>
        </div>
      </div>

      <div class="mode-panel task-panel">
        <div v-if="projects.length === 0" class="empty-hint">暂无项目配置</div>
        <div v-else class="card-grid">
          <div
            v-for="project in projects"
            :key="project.name"
            class="entry-card task-card"
            @click="$emit('select-project', project)"
          >
            <div class="card-header">
              <div class="avatar-icon task-avatar">
                <el-icon>
                  <Folder />
                </el-icon>
              </div>
              <div class="header-meta">
                <div class="card-name">{{ project.name }}</div>
                <div class="card-desc">{{ project.description || '暂无描述' }}</div>
              </div>
            </div>
            <div class="card-footer">
              <el-tag size="small" type="info" effect="plain">{{ (project.subprojects || []).length }} 子项目</el-tag>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- Shell 模式 -->
    <section class="home-section shell-section">
      <div class="section-header">
        <div class="section-title">
          <el-icon class="section-icon shell-icon">
            <Monitor />
          </el-icon>
          <div class="section-text">
            <h3>Shell 模式</h3>
            <span class="section-desc">SSH 连接机器，交互式执行命令</span>
          </div>
          <el-tag v-if="connectedCount > 0" size="small" type="success" effect="plain">
            {{ connectedCount }} 个会话进行中
          </el-tag>
        </div>
        <el-button size="small" type="primary" plain @click="$emit('open-shell')">
          进入 Shell 终端
        </el-button>
      </div>

      <div class="mode-panel shell-panel">
        <div class="shell-panel-header">
          <div class="shell-panel-title" @click="$emit('open-shell')">
            <div class="avatar-icon shell-avatar">
              <el-icon>
                <Monitor />
              </el-icon>
            </div>
            <div>
              <div class="card-name">远程 Shell</div>
              <div class="card-desc">管理多个 SSH 会话，点击标题进入终端</div>
            </div>
          </div>
          <div class="shell-panel-actions">
            <el-button size="small" text type="primary" @click="$emit('add-machine')">
              <el-icon>
                <Plus />
              </el-icon>
              添加机器
            </el-button>
            <el-icon class="shell-enter-icon" @click="$emit('open-shell')">
              <ArrowRight />
            </el-icon>
          </div>
        </div>

        <div class="shell-panel-body">
          <div v-if="machineGroups.length === 0" class="empty-hint">暂无机器，请点击右上角添加</div>
          <div v-for="group in machineGroups" :key="group.name" class="machine-group">
            <div class="group-title">{{ group.name }}</div>
            <div class="group-cards">
              <div
                v-for="machine in group.machines"
                :key="machine.name"
                class="entry-card machine-card compact-card"
                @click="$emit('connect-machine', machine.name)"
              >
                <div class="card-header">
                  <div class="avatar-icon machine-avatar">
                    <el-icon>
                      <Connection />
                    </el-icon>
                  </div>
                  <div class="header-meta">
                    <div class="card-name">{{ machine.name }}</div>
                    <div class="card-desc">{{ machine.key_file || '密钥/密码' }}</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script>
import { ref, computed, onMounted } from 'vue'
import * as App from '../../wailsjs/go/app/App'
import { groupMachines } from '../utils/machineGroups'

export default {
  name: 'HomePage',
  props: {
    projects: { type: Array, required: true },
    connectedCount: { type: Number, default: 0 },
  },
  emits: [
    'refresh',
    'select-project',
    'open-shell',
    'connect-machine',
    'add-machine',
    'open-system-settings',
    'open-execution-history',
  ],
  setup(_, { emit }) {
    const machines = ref([])

    const loadMachines = async () => {
      try {
        machines.value = await App.GetMachines() || []
      } catch {
        machines.value = []
      }
    }

    const machineGroups = computed(() => groupMachines(machines.value))

    const handleRefresh = async () => {
      await loadMachines()
      emit('refresh')
    }

    onMounted(loadMachines)

    return { machines, machineGroups, loadMachines, handleRefresh }
  },
}
</script>

<style scoped>
.home-page {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 24px 28px 36px;
  background: var(--app-bg);
}

.home-section {
  margin-bottom: 28px;
}

.shell-section {
  margin-bottom: 0;
}

.mode-panel {
  border: 1px solid var(--app-border);
  border-radius: 12px;
  background: var(--app-panel-bg);
  overflow: hidden;
}

.task-panel {
  padding: 16px;
  border-color: rgba(64, 158, 255, 0.35);
}

.shell-panel {
  border-color: rgba(103, 194, 58, 0.45);
}

.shell-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--app-border);
  background: rgba(103, 194, 58, 0.06);
}

.shell-panel-title {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  flex: 1;
  cursor: pointer;
  transition: opacity 0.2s ease;
}

.shell-panel-title:hover {
  opacity: 0.85;
}

.shell-panel-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.shell-enter-icon {
  color: #67c23a;
  font-size: 18px;
  cursor: pointer;
  padding: 4px;
}

.shell-panel-body {
  padding: 16px;
}

.machine-group + .machine-group {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px dashed var(--app-border);
}

.machine-group .group-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text-secondary);
  margin-bottom: 10px;
}

.group-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  justify-content: start;
  gap: 12px;
}

.compact-card {
  aspect-ratio: auto;
  min-height: 72px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
  gap: 12px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.section-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.section-title h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--app-text);
  line-height: 1.3;
}

.section-desc {
  font-size: 12px;
  color: var(--app-text-muted);
  line-height: 1.3;
}

.section-icon {
  font-size: 20px;
  flex-shrink: 0;
}

.task-icon {
  color: var(--app-accent-color);
}

.shell-icon {
  color: #67c23a;
}

.header-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.empty-hint {
  text-align: center;
  color: var(--app-text-muted);
  padding: 28px 16px;
}

.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  justify-content: start;
  gap: 14px;
}

.entry-card {
  padding: 14px;
  background: var(--app-card-bg);
  border-radius: 10px;
  border: 1px solid var(--app-card-border);
  cursor: pointer;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
  min-height: 120px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.entry-card:hover {
  border-color: var(--app-accent-color);
  box-shadow: 0 6px 18px var(--app-card-hover-shadow);
  transform: translateY(-2px);
}

.machine-card:hover {
  border-color: #e6a23c;
}

.card-header {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.avatar-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.task-avatar {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}

.shell-avatar {
  background: rgba(103, 194, 58, 0.15);
  color: #67c23a;
}

.machine-avatar {
  background: rgba(230, 162, 60, 0.15);
  color: #e6a23c;
}

.header-meta {
  min-width: 0;
}

.card-name {
  font-weight: 600;
  color: var(--app-text);
  margin-bottom: 4px;
  word-break: break-all;
}

.card-desc {
  font-size: 12px;
  color: var(--app-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.card-footer {
  display: flex;
  gap: 6px;
  margin-top: 12px;
}
</style>
