<template>
  <div class="home-page">
    <header class="home-hero">
      <div class="hero-copy">
        <h2 class="hero-title">FlashDock</h2>
      </div>
      <div class="hero-actions">
        <template v-if="!hasProjects">
          <el-input
            v-model="machineKeyword"
            clearable
            size="small"
            placeholder="搜索名称 / IP"
            class="app-toolbar-search compact machine-search hero-machine-search"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <div class="icon-actions icon-actions--sm">
            <el-dropdown
              trigger="hover"
              :show-timeout="120"
              :hide-timeout="160"
              @command="onConfigCommand"
            >
              <el-button circle title="配置文件">
                <el-icon><FolderOpened /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <template v-if="configFiles.length">
                    <el-dropdown-item
                      v-for="file in configFiles"
                      :key="file"
                      :command="`switch:${file}`"
                    >
                      <span class="config-item">
                        <el-icon v-if="file === currentConfig" class="config-check"><Check /></el-icon>
                        <span>{{ basename(file) }}</span>
                      </span>
                    </el-dropdown-item>
                  </template>
                  <el-dropdown-item v-else disabled>无法加载配置文件</el-dropdown-item>
                  <el-dropdown-item divided command="edit-pipeline">编辑任务流水线</el-dropdown-item>
                  <el-dropdown-item command="refresh">
                    <span>刷新配置列表</span>
                    <span class="menu-shortcut">{{ labelOf('refreshConfig') }}</span>
                  </el-dropdown-item>
                  <el-dropdown-item command="open-global">打开全局配置</el-dropdown-item>
                  <el-dropdown-item command="open-current">打开当前配置</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <el-tooltip content="编辑任务流水线" placement="top">
              <el-button circle title="编辑任务流水线" @click="openConfigEditor">
                <el-icon><Edit /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="添加机器" placement="top">
              <el-button circle @click="$emit('add-machine')">
                <el-icon><Plus /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="进入终端" placement="top">
              <el-button circle @click="$emit('open-shell')">
                <el-icon><Monitor /></el-icon>
              </el-button>
            </el-tooltip>
            <el-button :icon="Refresh" circle title="刷新" @click="handleRefresh" />
          </div>
        </template>
        <el-button v-else :icon="Refresh" circle title="刷新" @click="handleRefresh" />
      </div>
    </header>

    <div
      class="home-zones"
      :class="{
        'shell-only': !hasProjects,
        'task-minimized': hasProjects && minimizedZone === 'task',
        'shell-minimized': hasProjects && minimizedZone === 'shell',
        'is-focus': hasProjects && !!minimizedZone,
      }"
    >
      <section
        v-if="hasProjects"
        class="zone zone-task"
        :class="{ 'is-minimized': minimizedZone === 'task' }"
        aria-labelledby="zone-task-title"
      >
        <div
          class="zone-head"
          :class="{ 'is-clickable': minimizedZone === 'task' }"
          @click="minimizedZone === 'task' && toggleMinimize('task')"
        >
          <div class="zone-label">
            <span class="zone-dot task-dot" aria-hidden="true"></span>
            <div>
              <h3 id="zone-task-title">任务模式</h3>
            </div>
            <span v-if="minimizedZone === 'task'" class="zone-mini-hint">点击展开</span>
          </div>
          <div class="zone-actions" @click.stop>
            <template v-if="minimizedZone !== 'task'">
              <div class="zone-action-btns icon-actions icon-actions--sm">
                <el-dropdown
                  trigger="hover"
                  :show-timeout="120"
                  :hide-timeout="160"
                  @command="onConfigCommand"
                >
                  <el-button size="small" circle title="配置文件">
                    <el-icon><FolderOpened /></el-icon>
                  </el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <template v-if="configFiles.length">
                        <el-dropdown-item
                          v-for="file in configFiles"
                          :key="file"
                          :command="`switch:${file}`"
                        >
                          <span class="config-item">
                            <el-icon v-if="file === currentConfig" class="config-check"><Check /></el-icon>
                            <span>{{ basename(file) }}</span>
                          </span>
                        </el-dropdown-item>
                      </template>
                      <el-dropdown-item v-else disabled>无法加载配置文件</el-dropdown-item>
                      <el-dropdown-item divided command="edit-pipeline">编辑任务流水线</el-dropdown-item>
                      <el-dropdown-item command="refresh">
                        <span>刷新配置列表</span>
                        <span class="menu-shortcut">{{ labelOf('refreshConfig') }}</span>
                      </el-dropdown-item>
                      <el-dropdown-item command="open-global">打开全局配置</el-dropdown-item>
                      <el-dropdown-item command="open-current">打开当前配置</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
                <el-tooltip content="编辑任务流水线" placement="top">
                  <el-button size="small" circle title="编辑任务流水线" @click="openConfigEditor">
                    <el-icon><Edit /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip
                  :content="minimizedZone === 'shell' ? '恢复双栏' : '收起任务，展开机器列表'"
                  placement="top"
                >
                  <el-button
                    size="small"
                    circle
                    :title="minimizedZone === 'shell' ? '恢复双栏' : '收起任务'"
                    @click="toggleMinimize('task')"
                  >
                    <el-icon>
                      <ArrowDown v-if="minimizedZone === 'shell'" />
                      <ArrowUp v-else />
                    </el-icon>
                  </el-button>
                </el-tooltip>
              </div>
            </template>
            <el-tooltip v-else content="展开任务列表" placement="top">
              <el-button size="small" circle title="展开任务列表" @click="toggleMinimize('task')">
                <el-icon><ArrowDown /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </div>

        <div v-show="minimizedZone !== 'task'" class="zone-body">
          <div class="item-grid">
            <button
              v-for="project in projects"
              :key="project.name"
              type="button"
              class="item-card"
              @click="$emit('select-project', project)"
            >
              <div class="item-icon task-icon">
                <el-icon :size="18"><Folder /></el-icon>
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

      <section
        class="zone zone-shell"
        :class="{ 'is-minimized': hasProjects && minimizedZone === 'shell' }"
        :aria-label="hasProjects ? undefined : '机器列表'"
        :aria-labelledby="hasProjects ? 'zone-shell-title' : undefined"
      >
        <div
          v-if="hasProjects"
          class="zone-head"
          :class="{ 'is-clickable': minimizedZone === 'shell' }"
          @click="minimizedZone === 'shell' && toggleMinimize('shell')"
        >
          <div class="zone-label">
            <span class="zone-dot shell-dot" aria-hidden="true"></span>
            <div>
              <h3 id="zone-shell-title">
                Shell 模式
                <el-tag
                  v-if="connectedCount > 0"
                  size="small"
                  type="primary"
                  effect="plain"
                  class="session-tag"
                >
                  {{ connectedCount }} 会话进行中
                </el-tag>
              </h3>
            </div>
            <span v-if="minimizedZone === 'shell'" class="zone-mini-hint">点击展开</span>
          </div>
          <div class="zone-actions" @click.stop>
            <template v-if="minimizedZone !== 'shell'">
              <el-input
                v-model="machineKeyword"
                clearable
                size="small"
                placeholder="搜索名称 / IP"
                class="app-toolbar-search compact machine-search"
              >
                <template #prefix>
                  <el-icon><Search /></el-icon>
                </template>
              </el-input>
              <div class="zone-action-btns icon-actions icon-actions--sm">
                <el-tooltip content="添加机器" placement="top">
                  <el-button size="small" circle @click="$emit('add-machine')">
                    <el-icon><Plus /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="进入终端" placement="top">
                  <el-button size="small" circle @click="$emit('open-shell')">
                    <el-icon><Monitor /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip
                  :content="minimizedZone === 'task' ? '恢复双栏' : '收起 Shell，展开任务列表'"
                  placement="top"
                >
                  <el-button
                    size="small"
                    circle
                    :title="minimizedZone === 'task' ? '恢复双栏' : '收起 Shell'"
                    @click="toggleMinimize('shell')"
                  >
                    <el-icon>
                      <ArrowUp v-if="minimizedZone === 'task'" />
                      <ArrowDown v-else />
                    </el-icon>
                  </el-button>
                </el-tooltip>
              </div>
            </template>
            <el-tooltip v-else content="展开机器列表" placement="top">
              <el-button size="small" circle title="展开机器列表" @click="toggleMinimize('shell')">
                <el-icon><ArrowUp /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </div>

        <div
          v-show="!hasProjects || minimizedZone !== 'shell'"
          class="zone-body"
        >
          <div v-if="!(machines || []).length" class="app-empty">
            <p class="app-empty-title">暂无机器</p>
            <p class="app-empty-desc">点击右上角 + 添加机器</p>
          </div>
          <div v-else class="zone-list-wrap">
            <MachineConnectList
              :machines="filteredMachines"
              :sessions="sessions"
              :workspace-sessions="workspaceSessions"
              :connecting-name="connectingName"
              :filter-keyword="machineKeyword"
              layout="grid"
              variant="cards"
              empty-text="无匹配机器"
              @connect="onConnectMachine"
            />
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Refresh, ArrowUp, ArrowDown } from '@element-plus/icons-vue'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { machineMatchesKeyword, isMachineConnecting } from '../utils/machineGroups'
import { mergeShortcuts, formatShortcut } from '../utils/shortcuts'
import MachineConnectList from './shell/MachineConnectList.vue'

function basename(filePath) {
  if (!filePath) return ''
  const normalized = filePath.replace(/\\/g, '/')
  const idx = normalized.lastIndexOf('/')
  return idx >= 0 ? normalized.slice(idx + 1) : filePath
}

function normalizeMinimizedZone(zone) {
  return zone === 'task' || zone === 'shell' ? zone : ''
}

export default {
  name: 'HomePage',
  components: { MachineConnectList, ArrowUp, ArrowDown },
  props: {
    projects: { type: Array, required: true },
    machines: { type: Array, default: () => [] },
    connectedCount: { type: Number, default: 0 },
    hasTask: { type: Boolean, default: false },
    taskRunning: { type: Boolean, default: false },
    connectingName: { type: String, default: '' },
    sessions: { type: Array, default: () => [] },
    workspaceSessions: { type: Array, default: () => [] },
  },
  emits: [
    'refresh',
    'select-project',
    'resume-task',
    'open-shell',
    'connect-machine',
    'add-machine',
    'open-system-settings',
    'open-config-editor',
  ],
  setup(props, { emit }) {
    const machineKeyword = ref('')
    const configFiles = ref([])
    const currentConfig = ref('')
    const shortcuts = ref(mergeShortcuts())
    const minimizedZone = ref('')

    const hasProjects = computed(() => (props.projects || []).length > 0)
    const labelOf = (id) => formatShortcut(shortcuts.value[id])

    const openConfigEditor = () => {
      emit('open-config-editor')
    }

    const loadMinimizedZone = async () => {
      try {
        minimizedZone.value = normalizeMinimizedZone(await App.GetHomeMinimizedZone())
      } catch {
        minimizedZone.value = ''
      }
    }

    const toggleMinimize = async (zone) => {
      if (!hasProjects.value) return
      const next = minimizedZone.value === zone ? '' : zone
      const prev = minimizedZone.value
      minimizedZone.value = next
      try {
        await App.SetHomeMinimizedZone(next)
      } catch (e) {
        minimizedZone.value = prev
        console.error('保存首页布局失败:', e)
      }
    }

    const loadConfigMenu = async () => {
      try {
        const [files, current] = await Promise.all([
          App.GetConfigFiles(),
          App.GetCurrentConfigPath(),
        ])
        configFiles.value = files || []
        currentConfig.value = current || ''
      } catch {
        configFiles.value = []
        currentConfig.value = ''
      }
    }

    const loadShortcuts = async () => {
      try {
        shortcuts.value = mergeShortcuts(await App.GetShortcutSettings())
      } catch {
        shortcuts.value = mergeShortcuts()
      }
    }

    const onConfigCommand = (cmd) => {
      if (cmd === 'edit-pipeline') {
        openConfigEditor()
        return
      }
      if (cmd === 'refresh') {
        App.RefreshConfigMenuWithEvent()
        return
      }
      if (cmd === 'open-global') {
        App.OpenGlobalConfigWithEvent()
        return
      }
      if (cmd === 'open-current') {
        App.OpenCurrentConfigWithEvent()
        return
      }
      if (typeof cmd === 'string' && cmd.startsWith('switch:')) {
        const file = cmd.slice('switch:'.length)
        if (file && file !== currentConfig.value) {
          App.SwitchConfigFileWithEvent(file)
        }
      }
    }

    const filteredMachines = computed(() => {
      const kw = machineKeyword.value
      const list = props.machines || []
      if (!String(kw || '').trim()) return list
      return list.filter((m) => machineMatchesKeyword(m, kw))
    })

    const onConnectMachine = (name) => {
      if (isMachineConnecting(name, props.workspaceSessions.length ? props.workspaceSessions : props.sessions)) {
        return
      }
      emit('connect-machine', name)
    }

    const handleRefresh = () => emit('refresh')

    const onMinimizedZoneChanged = (zone) => {
      minimizedZone.value = normalizeMinimizedZone(zone)
    }

    onMounted(() => {
      loadConfigMenu()
      loadShortcuts()
      loadMinimizedZone()
      EventsOn('config:changed', loadConfigMenu)
      EventsOn('shortcuts:changed', loadShortcuts)
      EventsOn('home:minimized-zone', onMinimizedZoneChanged)
    })

    onUnmounted(() => {
      EventsOff('config:changed')
      EventsOff('shortcuts:changed')
      EventsOff('home:minimized-zone')
    })

    return {
      Refresh,
      machineKeyword,
      configFiles,
      currentConfig,
      hasProjects,
      filteredMachines,
      minimizedZone,
      basename,
      labelOf,
      onConfigCommand,
      openConfigEditor,
      onConnectMachine,
      handleRefresh,
      toggleMinimize,
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
  padding: var(--app-space-page, 28px 32px 24px);
  background: var(--app-bg);
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

.hero-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.hero-machine-search {
  width: min(240px, 42vw);
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
  grid-template-rows: minmax(0, 1fr);
  gap: 0;
}

/* 收起后改为上下布局：窄横条 + 全宽内容区 */
.home-zones.task-minimized,
.home-zones.shell-minimized {
  grid-template-columns: 1fr;
  gap: 10px;
}

.home-zones.task-minimized {
  grid-template-rows: auto minmax(0, 1fr);
}

.home-zones.shell-minimized {
  grid-template-rows: minmax(0, 1fr) auto;
}

.zone {
  display: flex;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius-panel, 14px);
  background: var(--app-panel-bg);
  box-shadow: var(--app-surface-shadow, none);
  overflow: hidden;
}

.zone-task {
  border-top: 2px solid color-mix(in srgb, var(--app-accent-color) 45%, var(--app-border));
}

.zone-shell {
  border-top: 2px solid color-mix(in srgb, var(--app-accent-color) 45%, var(--app-border));
}

.zone.is-minimized {
  flex: 0 0 auto;
  max-height: none;
  border-top: 1px solid var(--app-border);
}

.zone.is-minimized .zone-head {
  padding: 8px 14px;
  border-bottom: none;
  min-height: 44px;
  background: color-mix(in srgb, var(--app-text-muted) 5%, var(--app-panel-bg));
}

.zone.is-minimized .zone-head.is-clickable {
  cursor: pointer;
}

.zone.is-minimized .zone-head.is-clickable:hover {
  background: color-mix(in srgb, var(--app-accent-color) 8%, var(--app-panel-bg));
}

.zone.is-minimized .zone-label {
  align-items: center;
}

.zone.is-minimized .zone-label h3 {
  font-size: 13px;
  font-weight: 600;
}

.zone.is-minimized .zone-dot {
  margin-top: 0;
}

.zone-mini-hint {
  margin-left: 4px;
  font-size: 12px;
  color: var(--app-text-muted);
  white-space: nowrap;
}

.zone-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: var(--app-space-panel-head, 16px 18px 12px);
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
}

.zone-label {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.zone-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.task-dot {
  background: var(--app-accent-color);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--app-accent-color) 16%, transparent);
}

.shell-dot {
  background: var(--app-accent-color);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--app-accent-color) 16%, transparent);
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
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.machine-search {
  width: 160px;
}

.zone-body {
  flex: 1;
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
  padding: var(--app-space-panel-body, 14px 16px 18px);
  background: var(--app-inset-bg);
  scrollbar-gutter: stable;
}

.zone-list-wrap {
  width: 100%;
}

.item-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 10px;
  align-content: start;
}

.item-card {
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-height: 60px;
  padding: 12px;
  border: 1px solid var(--app-card-border);
  border-radius: 10px;
  background: var(--app-card-bg);
  box-shadow: var(--app-surface-shadow, none);
  color: inherit;
  text-align: left;
  cursor: pointer;
  box-sizing: border-box;
  transition: border-color 0.15s ease, background 0.15s ease, transform 0.15s ease, box-shadow 0.15s ease;
}

.item-card:hover {
  border-color: var(--app-accent-color);
  background: var(--app-card-active-bg);
  transform: translateY(-1px);
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

.config-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.config-check {
  color: var(--app-accent-color);
  font-size: 14px;
}

.menu-shortcut {
  margin-left: 24px;
  color: var(--app-text-muted);
  font-size: 12px;
}

@media (max-width: 960px) {
  .home-page {
    padding: 20px 16px 20px;
  }

  .home-hero {
    flex-direction: column;
  }

  .home-zones,
  .home-zones.shell-only {
    grid-template-columns: 1fr;
    overflow-y: auto;
  }

  .home-zones:not(.shell-only) {
    grid-auto-rows: minmax(220px, 1fr);
  }

  .home-zones.shell-only {
    grid-template-rows: minmax(220px, 1fr);
    grid-auto-rows: unset;
  }

  .home-zones.task-minimized,
  .home-zones.shell-minimized {
    grid-template-columns: 1fr;
    grid-auto-rows: auto;
  }

  .home-zones.task-minimized {
    grid-template-rows: auto minmax(220px, 1fr);
  }

  .home-zones.shell-minimized {
    grid-template-rows: minmax(220px, 1fr) auto;
  }

  .zone:not(.is-minimized) {
    max-height: min(520px, 60vh);
  }
}
</style>
