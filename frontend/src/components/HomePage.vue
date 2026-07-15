<template>
  <div class="home-page">
    <header class="home-hero">
      <div class="hero-copy">
        <h2 class="hero-title">FlashDock</h2>
        <!-- <p class="hero-subtitle">
          {{ hasProjects ? '选择一种方式开始：跑任务，或连机器进 Shell' : '连接机器，进入 Shell 开始工作' }}
        </p> -->
      </div>
      <div class="hero-actions">
        <!-- <el-button
          v-if="hasTask"
          type="primary"
          @click="$emit('resume-task')"
        >
          {{ taskRunning ? '返回执行中任务' : '继续当前任务' }}
        </el-button> -->
        <!-- <el-button @click="$emit('open-execution-history')">执行历史</el-button> -->
        <el-button :icon="Refresh" circle title="刷新" @click="handleRefresh" />
      </div>
    </header>

    <div class="home-zones" :class="{ 'shell-only': !hasProjects }">
      <!-- 任务区：有项目时才展示 -->
      <section v-if="hasProjects" class="zone zone-task" aria-labelledby="zone-task-title">
        <div class="zone-head">
          <div class="zone-label">
            <span class="zone-dot task-dot" aria-hidden="true"></span>
            <div>
              <h3 id="zone-task-title">任务模式</h3>
              <!-- <p>选择项目，按预设流程执行命令</p> -->
            </div>
          </div>
        </div>

        <div class="zone-body">
          <div class="item-grid">
            <button v-for="project in projects" :key="project.name" type="button" class="item-card"
              @click="$emit('select-project', project)">
              <div class="item-icon task-icon">
                <el-icon :size="18">
                  <Folder />
                </el-icon>
              </div>
              <div class="item-meta">
                <span class="item-name">{{ project.name }}</span>
                <span class="item-desc">{{ project.description || '暂无描述' }}</span>
              </div>
              <span class="item-badge">{{ (project.subprojects || []).length }} 子项目</span>
            </button>
          </div>
        </div>
      </section>

      <!-- Shell 区 -->
      <section class="zone zone-shell" aria-labelledby="zone-shell-title">
        <div class="zone-head">
          <div class="zone-label">
            <span class="zone-dot shell-dot" aria-hidden="true"></span>
            <div>
              <h3 id="zone-shell-title">
                Shell 模式
                <el-tag v-if="connectedCount > 0" size="small" type="success" effect="plain" class="session-tag">
                  {{ connectedCount }} 会话进行中
                </el-tag>
              </h3>
              <!-- <p>点机器直接连接；或先进终端再选机器</p> -->
            </div>
          </div>
          <div class="zone-actions">
            <el-input
              v-model="machineKeyword"
              clearable
              size="small"
              placeholder="搜索机器"
              class="machine-search"
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
            <div class="zone-action-btns icon-actions icon-actions--sm">
              <el-tooltip content="添加机器" placement="top">
                <el-button size="small" type="primary" plain circle @click="$emit('add-machine')">
                  <el-icon><Plus /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="进入终端" placement="top">
                <el-button size="small" type="success" plain circle @click="$emit('open-shell')">
                  <el-icon><Monitor /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </div>
        </div>

        <div class="zone-body">
          <div v-if="machines.length === 0" class="empty-hint">
            <p>暂无机器</p>
            <span>点击右上角 + 添加机器</span>
          </div>
          <div v-else-if="machineGroups.length === 0" class="empty-hint">
            <p>无匹配机器</p>
            <span>试试其他关键词</span>
          </div>
          <template v-else>
            <div v-for="group in machineGroups" :key="group.name" class="machine-group">
              <div class="group-title">{{ group.name }}</div>
              <div class="item-grid machines">
                <button
                  v-for="machine in group.machines"
                  :key="machine.id || machine.name"
                  type="button"
                  class="item-card machine-card"
                  :class="{ connecting: connectingName === machine.name }"
                  :disabled="!!connectingName"
                  @click="onConnectMachine(machine.name)"
                >
                  <div class="item-icon machine-icon">
                    <el-icon :size="18" :class="{ 'is-loading': connectingName === machine.name }">
                      <Loading v-if="connectingName === machine.name" />
                      <Monitor v-else />
                    </el-icon>
                  </div>
                  <div class="item-meta">
                    <span class="item-name">{{ machine.name }}</span>
                    <span class="item-desc">
                      {{ connectingName === machine.name ? '连接中…' : machineHost(machine) }}
                    </span>
                  </div>
                  <span v-if="connectingName === machine.name" class="item-badge connecting-badge">连接中</span>
                  <el-icon v-else class="item-chevron">
                    <ArrowRight />
                  </el-icon>
                </button>
              </div>
            </div>
          </template>
        </div>
      </section>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import * as App from '../../wailsjs/go/app/App'
import { groupMachines } from '../utils/machineGroups'

export default {
  name: 'HomePage',
  props: {
    projects: { type: Array, required: true },
    connectedCount: { type: Number, default: 0 },
    hasTask: { type: Boolean, default: false },
    taskRunning: { type: Boolean, default: false },
    connectingName: { type: String, default: '' },
  },
  emits: [
    'refresh',
    'select-project',
    'resume-task',
    'open-shell',
    'connect-machine',
    'add-machine',
    'open-system-settings',
  ],
  setup(props, { emit }) {
    const machines = ref([])
    const machineKeyword = ref('')

    const loadMachines = async () => {
      try {
        machines.value = await App.GetMachines() || []
      } catch {
        machines.value = []
      }
    }

    const filteredMachines = computed(() => {
      const kw = machineKeyword.value.trim().toLowerCase()
      const list = machines.value || []
      if (!kw) return list
      return list.filter((m) =>
        (m.name || '').toLowerCase().includes(kw) ||
        (m.group || '').toLowerCase().includes(kw) ||
        (m.key_file || '').toLowerCase().includes(kw) ||
        (m.host || '').toLowerCase().includes(kw)
      )
    })

    const machineGroups = computed(() => groupMachines(filteredMachines.value))
    const hasProjects = computed(() => (props.projects || []).length > 0)

    const machineHost = (machine) => {
      const host = machine.host || machine.ip || ''
      const port = machine.port ? `:${machine.port}` : ''
      if (host) return `${host}${port}`
      return machine.key_file || '点击连接'
    }

    const onConnectMachine = (name) => {
      if (props.connectingName) return
      emit('connect-machine', name)
    }

    const handleRefresh = async () => {
      await loadMachines()
      emit('refresh')
    }

    onMounted(loadMachines)

    return {
      Refresh,
      machines,
      machineKeyword,
      machineGroups,
      hasProjects,
      machineHost,
      onConnectMachine,
      loadMachines,
      handleRefresh,
    }
  },
}
</script>

<style scoped>
.home-page {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 28px 32px 24px;
  background:
    radial-gradient(ellipse 80% 50% at 0% 0%, color-mix(in srgb, var(--app-accent-color) 8%, transparent), transparent 55%),
    radial-gradient(ellipse 70% 45% at 100% 0%, color-mix(in srgb, #67c23a 7%, transparent), transparent 50%),
    var(--app-bg);
}

.home-hero {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 24px;
  flex-shrink: 0;
}

.hero-title {
  margin: 0 0 6px;
  font-size: 26px;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--app-text);
  line-height: 1.2;
}

.hero-subtitle {
  margin: 0;
  font-size: 14px;
  color: var(--app-text-muted);
  line-height: 1.45;
}

.hero-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.home-zones {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  align-items: stretch;
}

.home-zones.shell-only {
  grid-template-columns: 1fr;
}

.home-zones.shell-only .item-grid.machines {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 10px;
}

.zone {
  display: flex;
  flex-direction: column;
  min-height: 0;
  border: 1px solid var(--app-border);
  border-radius: 14px;
  background: var(--app-panel-bg);
  overflow: hidden;
}

.zone-task {
  border-top: 3px solid var(--app-accent-color);
}

.zone-shell {
  border-top: 3px solid #67c23a;
}

.zone-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 18px 12px;
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
}

.zone-label {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  min-width: 0;
}

.zone-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-top: 7px;
  flex-shrink: 0;
}

.task-dot {
  background: var(--app-accent-color);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--app-accent-color) 22%, transparent);
}

.shell-dot {
  background: #67c23a;
  box-shadow: 0 0 0 3px rgba(103, 194, 58, 0.22);
}

.zone-label h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 650;
  color: var(--app-text);
  line-height: 1.35;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.zone-label p {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--app-text-muted);
  line-height: 1.4;
}

.session-tag {
  vertical-align: middle;
}

.zone-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.zone-action-btns {
  flex-shrink: 0;
}

.machine-search {
  width: 148px;
}

.zone-body {
  flex: 1;
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 14px 16px 18px;
  scrollbar-gutter: stable;
}

.zone-body::-webkit-scrollbar {
  width: 8px;
}

.zone-body::-webkit-scrollbar-track {
  background: transparent;
}

.zone-body::-webkit-scrollbar-thumb {
  background: color-mix(in srgb, var(--app-text-muted) 35%, transparent);
  border-radius: 4px;
}

.zone-body::-webkit-scrollbar-thumb:hover {
  background: color-mix(in srgb, var(--app-text-muted) 55%, transparent);
}

.empty-hint {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 180px;
  color: var(--app-text-muted);
  text-align: center;
}

.empty-hint p {
  margin: 0;
  font-size: 14px;
  font-weight: 560;
  color: var(--app-text-secondary);
}

.empty-hint span {
  font-size: 12px;
}

.item-grid {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.item-grid.machines {
  gap: 8px;
}

.machine-group+.machine-group {
  margin-top: 16px;
  padding-top: 14px;
  border-top: 1px dashed var(--app-border);
}

.group-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--app-text-muted);
  letter-spacing: 0.02em;
  margin-bottom: 8px;
  text-transform: none;
}

.item-card {
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 12px 12px;
  border: 1px solid var(--app-card-border);
  border-radius: 10px;
  background: var(--app-card-bg);
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease, transform 0.15s ease;
}

.item-card:hover {
  border-color: var(--app-accent-color);
  background: var(--app-card-active-bg);
  transform: translateY(-1px);
}

.machine-card:hover {
  border-color: #67c23a;
}

.machine-card:disabled {
  cursor: wait;
  opacity: 0.72;
  transform: none;
}

.machine-card.connecting {
  border-color: #67c23a;
  background: color-mix(in srgb, #67c23a 10%, var(--app-card-bg));
  opacity: 1;
}

.connecting-badge {
  color: #67c23a;
  background: rgba(103, 194, 58, 0.16);
}

.item-icon {
  width: 36px;
  height: 36px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.task-icon {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}

.machine-icon {
  background: rgba(103, 194, 58, 0.14);
  color: #67c23a;
}

.item-meta {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.item-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-desc {
  font-size: 12px;
  color: var(--app-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-badge {
  font-size: 11px;
  color: var(--app-text-muted);
  padding: 2px 8px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--app-text-muted) 12%, transparent);
  white-space: nowrap;
}

.item-chevron {
  color: var(--app-text-muted);
  font-size: 14px;
  opacity: 0.7;
}

.machine-card:hover .item-chevron {
  color: #67c23a;
  opacity: 1;
}

@media (max-width: 960px) {
  .home-page {
    padding: 20px 16px 20px;
  }

  .home-hero {
    flex-direction: column;
  }

  .home-zones {
    grid-template-columns: 1fr;
    grid-auto-rows: minmax(220px, 1fr);
    overflow-y: auto;
  }

  .zone {
    max-height: min(420px, 48vh);
  }
}
</style>
