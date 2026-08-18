<template>
  <div class="home-page" :class="{ 'has-rail': hasProjects }">
    <!-- 有任务配置时：左侧分区导航（对标 Netcatty Vault sidebar） -->
    <aside v-if="hasProjects" class="home-rail" aria-label="首页分区">
      <div class="rail-brand">
        <img class="rail-brand-mark" :src="brandIconUrl" alt="" aria-hidden="true" />
        <span class="rail-brand-text">FlashDock</span>
      </div>
      <nav class="rail-nav">
        <button type="button" class="rail-item" :class="{ active: homeSection === 'task' }"
          @click="setHomeSection('task')">
          <el-icon :size="16">
            <Folder />
          </el-icon>
          <span>任务</span>
          <span class="rail-count">{{ projects.length }}</span>
        </button>
        <button type="button" class="rail-item" :class="{ active: homeSection === 'shell' }"
          @click="setHomeSection('shell')">
          <el-icon :size="16">
            <Monitor />
          </el-icon>
          <span>主机</span>
          <span v-if="connectedCount > 0" class="rail-live">{{ connectedCount }}</span>
          <span v-else class="rail-count">{{ (machines || []).length }}</span>
        </button>
      </nav>
      <div class="rail-footer">
        <button type="button" class="rail-item rail-settings" @click="$emit('open-system-settings')">
          <el-icon :size="16">
            <Setting />
          </el-icon>
          <span>设置</span>
        </button>
      </div>
    </aside>

    <div class="home-stage">
      <div class="home-surface">
        <!-- 顶栏：搜索 + 主操作（对标 VaultPageHeader） -->
        <header class="home-header">
          <el-input ref="searchInputRef" v-model="machineKeyword" clearable size="large" class="home-search"
            :placeholder="searchPlaceholder" @keydown.enter.exact.prevent="onSearchEnter">
            <template #prefix>
              <el-icon>
                <Search />
              </el-icon>
            </template>
          </el-input>

          <div class="home-header-actions">
            <el-tooltip v-if="showingShell" content="新建主机" placement="bottom">
              <el-button type="primary" class="home-btn-icon home-btn-icon--primary" @click="$emit('add-machine')">
                <el-icon>
                  <Plus />
                </el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip v-else content="编辑流水线" placement="bottom">
              <el-button type="primary" class="home-btn-icon home-btn-icon--primary" @click="openConfigEditor">
                <el-icon>
                  <Edit />
                </el-icon>
              </el-button>
            </el-tooltip>

            <el-tooltip content="打开终端" placement="bottom">
              <el-button class="home-btn-icon" @click="$emit('open-shell')">
                <el-icon>
                  <Monitor />
                </el-icon>
              </el-button>
            </el-tooltip>

            <el-dropdown trigger="click" :show-timeout="120" :hide-timeout="160" @command="onMoreCommand">
              <el-button class="home-btn-icon" title="更多">
                <el-icon>
                  <MoreFilled />
                </el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <template v-if="configFiles.length">
                    <el-dropdown-item v-for="file in configFiles" :key="file" :command="`switch:${file}`">
                      <span class="config-item">
                        <el-icon v-if="file === currentConfig" class="config-check">
                          <Check />
                        </el-icon>
                        <span>{{ basename(file) }}</span>
                      </span>
                    </el-dropdown-item>
                  </template>
                  <el-dropdown-item v-else disabled>无法加载配置文件</el-dropdown-item>
                  <el-dropdown-item divided command="edit-pipeline">编辑任务流水线</el-dropdown-item>
                  <el-dropdown-item command="reload">刷新</el-dropdown-item>
                  <el-dropdown-item command="refresh">刷新配置列表</el-dropdown-item>
                  <el-dropdown-item command="open-global">打开全局配置</el-dropdown-item>
                  <el-dropdown-item command="open-current">打开当前配置</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </header>

        <div class="home-scroll">
          <!-- 任务分区 -->
          <template v-if="showingTask">
            <div class="home-crumb">
              <span class="crumb-active">全部任务</span>
              <span v-if="filteredProjects.length" class="crumb-meta">{{ filteredProjects.length }} 个项目</span>
            </div>

            <div v-if="!filteredProjects.length" class="home-empty">
              <div class="home-empty-icon" aria-hidden="true">
                <el-icon :size="28">
                  <Folder />
                </el-icon>
              </div>
              <p class="home-empty-title">{{ machineKeyword.trim() ? '无匹配任务' : '暂无任务项目' }}</p>
              <p class="home-empty-desc">
                {{ machineKeyword.trim() ? '试试其他关键词' : '在配置中添加项目后，可在此快速进入任务模式' }}
              </p>
            </div>

            <div v-else class="home-section">
              <div class="item-grid">
                <button v-for="project in filteredProjects" :key="project.name" type="button" class="item-card"
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
          </template>

          <!-- 主机分区 -->
          <template v-else>
            <div class="home-crumb">
              <span class="crumb-active">全部主机</span>
              <el-tag v-if="connectedCount > 0" size="small" type="primary" effect="plain" class="session-tag">
                {{ connectedCount }} 会话进行中
              </el-tag>
            </div>

            <div v-if="!(machines || []).length" class="home-empty">
              <div class="home-empty-icon" aria-hidden="true">
                <el-icon :size="28">
                  <Monitor />
                </el-icon>
              </div>
              <p class="home-empty-title">配置你的主机</p>
              <p class="home-empty-desc">保存主机后可快速连接到服务器、虚拟机与容器</p>
              <el-button type="primary" @click="$emit('add-machine')">
                <el-icon>
                  <Plus />
                </el-icon>
                新建主机
              </el-button>
            </div>

            <div v-else class="zone-list-wrap">
              <div v-if="quickConnectHint" class="quick-connect-bar">
                <span class="quick-connect-text">{{ quickConnectHint.text }}</span>
                <el-tooltip :content="quickConnectHint.action" placement="top">
                  <el-button size="small" type="primary" class="quick-connect-btn" @click="onQuickConnect">
                    <el-icon>
                      <Connection />
                    </el-icon>
                  </el-button>
                </el-tooltip>
              </div>

              <div v-if="availableTags.length" class="home-tag-filter">
                <button type="button" class="home-tag-chip" :class="{ active: !selectedTags.length }"
                  @click="selectedTags = []">全部</button>
                <button v-for="t in availableTags" :key="t" type="button" class="home-tag-chip"
                  :class="{ active: selectedTags.includes(t) }" @click="toggleTagFilter(t)">{{ t }}</button>
              </div>

              <div v-if="pinnedMachines.length" class="home-section">
                <div class="home-section-title">
                  <el-icon :size="13">
                    <StarFilled />
                  </el-icon>
                  置顶
                </div>
                <MachineConnectList :machines="pinnedMachines" :sessions="sessions"
                  :workspace-sessions="workspaceSessions" :connecting-name="connectingName"
                  :filter-keyword="machineKeyword" layout="grid" variant="cards" show-context-menu empty-text="无匹配机器"
                  @connect="onConnectMachine" @focus-session="(id) => $emit('focus-session', id)"
                  @edit-machine="(m) => $emit('edit-machine', m)"
                  @copy-machine="(m) => $emit('copy-machine', m)" @delete-machine="(m) => $emit('delete-machine', m)"
                  @toggle-pin="onTogglePin" />
              </div>

              <div v-if="recentMachines.length" class="home-section">
                <div class="home-section-title">
                  <el-icon :size="13">
                    <Clock />
                  </el-icon>
                  最近连接
                </div>
                <MachineConnectList :machines="recentMachines" :sessions="sessions"
                  :workspace-sessions="workspaceSessions" :connecting-name="connectingName"
                  :filter-keyword="machineKeyword" layout="grid" variant="cards" flat show-context-menu
                  empty-text="无匹配机器" @connect="onConnectMachine"
                  @focus-session="(id) => $emit('focus-session', id)"
                  @edit-machine="(m) => $emit('edit-machine', m)"
                  @copy-machine="(m) => $emit('copy-machine', m)" @delete-machine="(m) => $emit('delete-machine', m)"
                  @toggle-pin="onTogglePin" />
              </div>

              <div class="home-section">
                <div v-if="pinnedMachines.length || recentMachines.length" class="home-section-title">
                  全部主机
                </div>
                <MachineConnectList :machines="filteredMachines" :sessions="sessions"
                  :workspace-sessions="workspaceSessions" :connecting-name="connectingName"
                  :filter-keyword="machineKeyword" layout="grid" variant="cards" show-context-menu empty-text="无匹配机器"
                  @connect="onConnectMachine" @focus-session="(id) => $emit('focus-session', id)"
                  @edit-machine="(m) => $emit('edit-machine', m)"
                  @copy-machine="(m) => $emit('copy-machine', m)" @delete-machine="(m) => $emit('delete-machine', m)"
                  @toggle-pin="onTogglePin" />
              </div>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'
import { Clock, StarFilled } from '@element-plus/icons-vue'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { machineMatchesKeyword, isMachineConnecting, collectMachineTags, machineMatchesTags } from '../utils/machineGroups'
import { parseQuickConnectTarget, findMachineForQuickConnect } from '../utils/quickConnect'
import MachineConnectList from './shell/MachineConnectList.vue'
import defaultAppIcon from '../assets/appicon.png'

function basename(filePath) {
  if (!filePath) return ''
  const normalized = filePath.replace(/\\/g, '/')
  const idx = normalized.lastIndexOf('/')
  return idx >= 0 ? normalized.slice(idx + 1) : filePath
}

function normalizeMinimizedZone(zone) {
  return zone === 'task' || zone === 'shell' ? zone : ''
}

/** 旧双栏 minimizedZone → 新侧栏分区 */
function zoneToSection(zone) {
  // shell 被收起 → 看任务；其余默认主机（贴近 Netcatty 主机首页）
  return zone === 'shell' ? 'task' : 'shell'
}

function sectionToZone(section) {
  return section === 'task' ? 'shell' : 'task'
}

export default {
  name: 'HomePage',
  components: { MachineConnectList, Clock, StarFilled },
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
    'focus-session',
    'add-machine',
    'edit-machine',
    'copy-machine',
    'delete-machine',
    'open-system-settings',
    'open-config-editor',
  ],
  setup(props, { emit }) {
    const machineKeyword = ref('')
    const selectedTags = ref([])
    const configFiles = ref([])
    const currentConfig = ref('')
    const minimizedZone = ref('')
    const historyRecords = ref([])
    const searchInputRef = ref(null)
    const brandIconUrl = ref(defaultAppIcon)

    const focusSearchInput = async () => {
      await nextTick()
      searchInputRef.value?.focus?.()
    }

    const loadBrandIcon = async () => {
      try {
        const [cfg, presets] = await Promise.all([
          App.GetGlobalConfig(),
          App.ListAppIconPresets(),
        ])
        const presetId = cfg?.appIconPreset || 'default'
        const found = (presets || []).find((p) => p?.id === presetId)
        brandIconUrl.value = found?.preview || defaultAppIcon
      } catch {
        brandIconUrl.value = defaultAppIcon
      }
    }

    const hasProjects = computed(() => (props.projects || []).length > 0)
    const homeSection = computed(() => {
      if (!hasProjects.value) return 'shell'
      return zoneToSection(minimizedZone.value)
    })
    const showingTask = computed(() => hasProjects.value && homeSection.value === 'task')
    const showingShell = computed(() => !showingTask.value)

    const availableTags = computed(() => collectMachineTags(props.machines || []))

    const searchPlaceholder = computed(() => {
      if (showingTask.value) return '搜索任务项目名称 / 描述…'
      return '查找主机或输入 SSH 地址…'
    })

    const filteredProjects = computed(() => {
      const kw = machineKeyword.value.trim().toLowerCase()
      const list = props.projects || []
      if (!kw) return list
      return list.filter((p) => {
        const name = String(p?.name || '').toLowerCase()
        const desc = String(p?.description || '').toLowerCase()
        return name.includes(kw) || desc.includes(kw)
      })
    })

    const toggleTagFilter = (tag) => {
      const t = String(tag || '').trim()
      if (!t) return
      if (selectedTags.value.includes(t)) {
        selectedTags.value = selectedTags.value.filter((x) => x !== t)
      } else {
        selectedTags.value = [...selectedTags.value, t]
      }
    }

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

    const loadHistory = async () => {
      try {
        historyRecords.value = (await App.GetShellHistory()) || []
      } catch {
        historyRecords.value = []
      }
    }

    const setHomeSection = async (section) => {
      if (!hasProjects.value) return
      if (section !== 'task' && section !== 'shell') return
      if (homeSection.value === section) {
        await focusSearchInput()
        return
      }
      const nextZone = sectionToZone(section)
      const prev = minimizedZone.value
      minimizedZone.value = nextZone
      try {
        await App.SetHomeMinimizedZone(nextZone)
      } catch (e) {
        minimizedZone.value = prev
        console.error('保存首页分区失败:', e)
        return
      }
      await focusSearchInput()
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
      loadBrandIcon()
    }

    const onMoreCommand = (cmd) => {
      if (cmd === 'reload') {
        handleRefresh()
        return
      }
      onConfigCommand(cmd)
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
      return list.filter(
        (m) => machineMatchesKeyword(m, kw) && machineMatchesTags(m, selectedTags.value),
      )
    })

    const pinnedMachines = computed(() =>
      filteredMachines.value.filter((m) => !!m.pinned),
    )

    const recentMachines = computed(() => {
      const byName = new Map((props.machines || []).map((m) => [m.name, m]))
      const out = []
      const seen = new Set()
      for (const rec of historyRecords.value || []) {
        const name = rec?.machineName || rec?.configName || rec?.name
        if (!name || seen.has(name)) continue
        const m = byName.get(name)
        if (!m) continue
        if (!machineMatchesKeyword(m, machineKeyword.value) || !machineMatchesTags(m, selectedTags.value)) {
          continue
        }
        seen.add(name)
        out.push(m)
        if (out.length >= 8) break
      }
      return out
    })

    const quickTarget = computed(() => parseQuickConnectTarget(machineKeyword.value))
    const quickMatched = computed(() =>
      findMachineForQuickConnect(props.machines || [], quickTarget.value),
    )
    const quickConnectHint = computed(() => {
      if (!showingShell.value) return null
      const t = quickTarget.value
      if (!t) return null
      if (quickMatched.value) {
        return {
          text: `快速连接 ${quickMatched.value.name}（${t.user ? `${t.user}@` : ''}${t.host}${t.port ? `:${t.port}` : ''}）`,
          action: '连接',
          mode: 'connect',
        }
      }
      return {
        text: `未找到匹配机器：${t.user ? `${t.user}@` : ''}${t.host}${t.port ? `:${t.port}` : ''}，可先添加`,
        action: '添加机器',
        mode: 'add',
      }
    })

    const onConnectMachine = (name) => {
      if (isMachineConnecting(name, props.workspaceSessions.length ? props.workspaceSessions : props.sessions)) {
        return
      }
      emit('connect-machine', name)
    }

    const onQuickConnect = () => {
      if (quickConnectHint.value?.mode === 'connect' && quickMatched.value) {
        onConnectMachine(quickMatched.value.name)
        return
      }
      emit('add-machine')
    }

    const onSearchEnter = () => {
      if (showingShell.value && quickConnectHint.value) {
        onQuickConnect()
      }
    }

    const onTogglePin = async (machine) => {
      if (!machine) return
      const key = machine.id || machine.name
      try {
        await App.SetMachinePinned(key, !machine.pinned)
        emit('refresh')
      } catch (e) {
        console.error('置顶失败:', e)
      }
    }

    const handleRefresh = () => {
      loadHistory()
      emit('refresh')
    }

    const onMinimizedZoneChanged = (zone) => {
      minimizedZone.value = normalizeMinimizedZone(zone)
    }

    let offConfigChanged = null
    let offMinimizedZone = null

    onMounted(() => {
      loadConfigMenu()
      loadMinimizedZone()
      loadHistory()
      loadBrandIcon()
      // 用 EventsOn 返回的取消函数解绑，避免 EventsOff(事件名) 清掉 App 等同名监听
      offConfigChanged = EventsOn('config:changed', loadConfigMenu)
      offMinimizedZone = EventsOn('home:minimized-zone', onMinimizedZoneChanged)
    })

    onUnmounted(() => {
      offConfigChanged?.()
      offMinimizedZone?.()
      offConfigChanged = null
      offMinimizedZone = null
    })

    return {
      machineKeyword,
      selectedTags,
      availableTags,
      toggleTagFilter,
      configFiles,
      currentConfig,
      hasProjects,
      homeSection,
      showingTask,
      showingShell,
      filteredProjects,
      filteredMachines,
      pinnedMachines,
      recentMachines,
      quickConnectHint,
      searchPlaceholder,
      searchInputRef,
      brandIconUrl,
      basename,
      onConfigCommand,
      onMoreCommand,
      openConfigEditor,
      onConnectMachine,
      onQuickConnect,
      onSearchEnter,
      onTogglePin,
      handleRefresh,
      setHomeSection,
    }
  },
}
</script>

<style scoped>
.home-page {
  flex: 1;
  min-height: 0;
  display: flex;
  overflow: hidden;
  background: var(--app-inset-bg, var(--app-bg));
}

.home-rail {
  width: 188px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  padding: 10px;
  box-sizing: border-box;
  border-right: 1px solid color-mix(in srgb, var(--app-border) 70%, transparent);
  background: var(--app-inset-bg, var(--app-bg));
}

.rail-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 8px 14px;
  margin-bottom: 4px;
}

.rail-brand-mark {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  object-fit: cover;
  flex-shrink: 0;
  display: block;
  background: transparent;
  border: none;
}

.rail-brand-text {
  font-size: 13px;
  font-weight: 650;
  color: var(--app-text);
}

.rail-nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
  min-height: 0;
}

.rail-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: 40px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: transparent;
  color: var(--app-text-secondary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  box-sizing: border-box;
  text-align: left;
  transition: background 0.15s ease, color 0.15s ease, border-color 0.15s ease;
}

.rail-item:hover {
  background: color-mix(in srgb, var(--app-text) 6%, transparent);
  color: var(--app-text);
}

.rail-item.active {
  background: color-mix(in srgb, var(--app-text) 10%, transparent);
  border-color: color-mix(in srgb, var(--app-border) 80%, transparent);
  color: var(--app-text);
}

.rail-count,
.rail-live {
  margin-left: auto;
  font-size: 11px;
  font-weight: 600;
  min-width: 1.2em;
  text-align: center;
  line-height: 1.4;
  border-radius: 999px;
  padding: 1px 7px;
  color: var(--app-text-muted);
  background: color-mix(in srgb, var(--app-text) 8%, transparent);
}

.rail-live {
  color: var(--app-accent-color);
  background: var(--app-accent-bg);
}

.rail-footer {
  padding-top: 8px;
  border-top: 1px solid color-mix(in srgb, var(--app-border) 70%, transparent);
}

.rail-settings {
  color: var(--app-text-muted);
}

.home-stage {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  padding: 10px;
  box-sizing: border-box;
}

.home-surface {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border: 1px solid color-mix(in srgb, var(--app-border) 70%, transparent);
  border-radius: var(--app-radius-panel, 14px);
  background: var(--app-panel-bg);
  box-shadow: var(--app-surface-shadow, 0 4px 12px rgba(0, 0, 0, 0.06));
  overflow: hidden;
}

.home-header {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 56px;
  padding: 10px 16px;
  box-sizing: border-box;
  border-bottom: 1px solid color-mix(in srgb, var(--app-border) 70%, transparent);
  background: color-mix(in srgb, var(--app-panel-bg) 92%, transparent);
}

.home-search {
  flex: 1;
  min-width: 0;
}

.home-search :deep(.el-input__wrapper) {
  min-height: 40px;
  border-radius: 10px;
  background: var(--app-inset-bg, var(--app-bg));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--app-border) 80%, transparent) inset;
}

.home-header-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.home-btn-icon {
  box-sizing: border-box;
  width: 36px;
  height: 36px;
  min-width: 36px;
  min-height: 36px;
  padding: 0;
  margin: 0;
  border-radius: 8px;
  background: color-mix(in srgb, var(--app-text) 5%, transparent);
  border-color: color-mix(in srgb, var(--app-border) 80%, transparent);
  color: var(--app-text-secondary);
}

.home-btn-icon:hover {
  color: var(--app-accent-color);
  background: var(--app-accent-bg);
  border-color: color-mix(in srgb, var(--app-accent-color) 35%, var(--app-border));
}

.home-btn-icon--primary {
  width: 48px;
  min-width: 48px;
  height: 36px;
  min-height: 36px;
  background: var(--app-accent-color);
  border-color: var(--app-accent-color);
  color: #fff;
  box-shadow: 0 2px 8px color-mix(in srgb, var(--app-accent-color) 35%, transparent);
}

.home-btn-icon--primary:hover {
  background: color-mix(in srgb, var(--app-accent-color) 88%, #000);
  border-color: color-mix(in srgb, var(--app-accent-color) 88%, #000);
  color: #fff;
}

.quick-connect-btn {
  width: 28px;
  height: 28px;
  padding: 0;
}

.home-scroll {
  flex: 1;
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 14px 16px 18px;
  box-sizing: border-box;
  scrollbar-gutter: stable;
}

.home-crumb {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  padding: 0 2px;
}

.crumb-active {
  font-size: 14px;
  font-weight: 650;
  color: var(--app-accent-color);
}

.crumb-meta {
  font-size: 12px;
  color: var(--app-text-muted);
}

.session-tag {
  vertical-align: middle;
}

.home-section {
  margin-bottom: 18px;
}

.home-section-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--app-text-muted);
  margin: 0 0 10px 2px;
  letter-spacing: 0.02em;
}

.zone-list-wrap {
  width: 100%;
}

.quick-connect-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid color-mix(in srgb, var(--app-accent-color) 30%, var(--app-border));
  background: color-mix(in srgb, var(--app-accent-color) 10%, var(--app-card-bg));
}

.quick-connect-text {
  font-size: 13px;
  color: var(--app-text);
  line-height: 1.4;
}

.home-tag-filter {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}

.home-tag-chip {
  border: 1px solid var(--app-border);
  background: var(--app-card-bg);
  color: var(--app-text-secondary);
  font-size: 11px;
  line-height: 1;
  padding: 5px 10px;
  border-radius: 999px;
  cursor: pointer;
}

.home-tag-chip.active {
  border-color: color-mix(in srgb, var(--app-accent-color) 55%, var(--app-border));
  color: var(--app-accent-color);
  background: var(--app-accent-bg);
}

.home-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  gap: 8px;
  min-height: 280px;
  padding: 48px 16px;
  color: var(--app-text-muted);
}

.home-empty-icon {
  width: 64px;
  height: 64px;
  border-radius: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 6px;
  background: color-mix(in srgb, var(--app-text) 6%, transparent);
  color: var(--app-text-secondary);
  opacity: 0.85;
}

.home-empty-title {
  margin: 0;
  font-size: 16px;
  font-weight: 650;
  color: var(--app-text);
}

.home-empty-desc {
  margin: 0 0 8px;
  font-size: 13px;
  line-height: 1.45;
  max-width: 320px;
  color: var(--app-text-muted);
}

.item-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 12px;
  align-content: start;
}

.item-card {
  display: grid;
  grid-template-columns: 40px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  width: 100%;
  height: 68px;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--app-card-border, var(--app-border)) 85%, transparent);
  border-radius: 12px;
  background: var(--app-card-bg);
  box-shadow: 0 4px 12px color-mix(in srgb, var(--app-text) 4%, transparent);
  color: inherit;
  text-align: left;
  cursor: pointer;
  box-sizing: border-box;
  transition: border-color 0.15s ease, background 0.15s ease, transform 0.15s ease, box-shadow 0.15s ease;
}

.item-card:hover {
  border-color: var(--app-accent-color);
  background: var(--app-card-active-bg, var(--app-accent-bg));
  transform: translateY(-2px);
  box-shadow: 0 8px 18px color-mix(in srgb, var(--app-text) 8%, transparent);
}

.item-icon {
  width: 40px;
  height: 40px;
  border-radius: 11px;
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
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-desc {
  font-size: 11px;
  color: var(--app-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
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
  .home-page.has-rail {
    flex-direction: column;
  }

  .home-rail {
    width: 100%;
    flex-direction: row;
    align-items: center;
    gap: 8px;
    padding: 8px 10px;
    border-right: none;
    border-bottom: 1px solid color-mix(in srgb, var(--app-border) 70%, transparent);
  }

  .rail-brand {
    padding: 0 8px 0 0;
    margin: 0;
  }

  .rail-brand-text {
    display: none;
  }

  .rail-nav {
    flex-direction: row;
    flex: 1;
  }

  .rail-footer {
    border-top: none;
    padding-top: 0;
  }

  .rail-item {
    width: auto;
    padding: 0 12px;
  }

  .rail-count {
    display: none;
  }

  .home-stage {
    padding: 8px;
  }

  .home-header {
    flex-wrap: wrap;
  }

  .home-search {
    flex: 1 1 100%;
  }

  .home-header-actions {
    width: 100%;
    justify-content: flex-end;
    flex-wrap: wrap;
  }
}
</style>
