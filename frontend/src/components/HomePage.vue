<template>
  <header
    class="home-header"
    @dblclick="onChromeTitleDblActivate"
    @mousedown="onChromeTitlePointerDown"
  >
    <AppSearch
      ref="searchInputRef"
      v-model="machineKeyword"
      class="home-search-wrap"
      placeholder="查找主机或输入 SSH 地址快速连接…"
      @enter="onSearchEnter"
    />
    <div class="home-header-spacer" />
    <div class="home-header-actions">
      <AppButton class="home-toolbar-btn home-toolbar-btn--icon" title="新建主机" aria-label="新建主机" @click="$emit('add-machine')">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
          <path d="M12 5v14M5 12h14" />
        </svg>
      </AppButton>
      <AppButton
        class="home-toolbar-btn home-toolbar-btn--icon"
        :title="liveSessionCount > 0 ? '返回终端' : '打开终端'"
        :aria-label="liveSessionCount > 0 ? '返回终端' : '打开终端'"
        @click="$emit('open-shell')"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M4 17l6-6-6-6M12 19h8" />
        </svg>
      </AppButton>
      <AppButton class="home-toolbar-btn home-toolbar-btn--icon" title="配置文件" aria-label="配置文件" @click.stop="toggleConfigMenu">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" aria-hidden="true">
          <path d="M6 3h9l4 4v14H6z" /><path d="M14 3v5h5" />
        </svg>
      </AppButton>
    </div>

    <div v-show="configMenuOpen" class="dropdown home-config-dropdown" @click.stop>
      <div class="dd-label">业务配置文件</div>
      <template v-if="configFiles.length">
        <button
          v-for="file in configFiles"
          :key="file"
          type="button"
          class="dd-item"
          :class="{ active: file === currentConfig }"
          @click="onConfigCommand(`switch:${file}`)"
        >
          <svg
            v-if="file === currentConfig"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.4"
            aria-hidden="true"
          >
            <path d="M5 13l4 4L19 7" />
          </svg>
          {{ basename(file) }}
        </button>
      </template>
      <button v-else type="button" class="dd-item" disabled>无法加载配置文件</button>
      <div class="dd-sep" />
      <button type="button" class="dd-item" @click="onConfigCommand('edit-pipeline')">编辑任务流水线</button>
      <button type="button" class="dd-item" @click="onConfigCommand('reload')">刷新</button>
      <button type="button" class="dd-item" @click="onConfigCommand('refresh')">刷新配置列表</button>
      <div class="dd-sep" />
      <button type="button" class="dd-item" @click="onConfigCommand('open-global')">打开全局配置</button>
      <button type="button" class="dd-item" @click="onConfigCommand('open-current')">打开当前配置</button>
    </div>
  </header>

  <div class="home-scroll">
    <template v-if="hasProjects">
      <SectionTitle title="任务项目" :count="projects.length">
        <template #icon>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <rect x="4" y="3" width="16" height="18" rx="2" /><path d="M8 8h8M8 12h8M8 16h5" />
          </svg>
        </template>
      </SectionTitle>

      <div v-if="filteredProjects.length" class="proj-grid">
        <ProjectCard
          v-for="project in filteredProjects"
          :key="project.name"
          :name="project.name"
          :description="project.description || '暂无描述'"
          :meta="`${(project.subprojects || []).length} 子项目`"
          @click="$emit('select-project', project)"
        >
          <template #icon>
            <el-icon :size="16"><Folder /></el-icon>
          </template>
        </ProjectCard>
      </div>
    </template>

    <template v-if="(machines || []).length">
      <div v-if="quickConnectHint" class="quick-connect-bar">
        <span class="quick-connect-text">{{ quickConnectHint.text }}</span>
        <el-tooltip :content="quickConnectHint.action" placement="top">
          <el-button size="small" type="primary" class="quick-connect-btn" @click="onQuickConnect">
            <el-icon><Connection /></el-icon>
          </el-button>
        </el-tooltip>
      </div>

      <SectionTitle v-if="pinnedMachines.length" title="置顶主机" :count="pinnedMachines.length">
        <template #icon>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <rect x="2" y="4" width="20" height="7" rx="2" /><rect x="2" y="13" width="20" height="7" rx="2" /><path d="M6 7.5h.01M6 16.5h.01" />
          </svg>
        </template>
      </SectionTitle>
      <MachineConnectList
        v-if="pinnedMachines.length"
        class="machine-grid"
        :machines="pinnedMachines"
        :sessions="sessions"
        :workspace-sessions="workspaceSessions"
        :connecting-name="connectingName"
        :filter-keyword="machineKeyword"
        layout="grid"
        variant="cards"
        show-context-menu
        empty-text="无匹配机器"
        @connect="onConnectMachine"
        @focus-session="(id) => $emit('focus-session', id)"
        @edit-machine="(m) => $emit('edit-machine', m)"
        @copy-machine="(m) => $emit('copy-machine', m)"
        @delete-machine="(m) => $emit('delete-machine', m)"
        @toggle-pin="onTogglePin"
      />

      <SectionTitle v-if="recentMachines.length" title="最近连接" :count="recentMachines.length">
        <template #icon>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" />
          </svg>
        </template>
      </SectionTitle>
      <MachineConnectList
        v-if="recentMachines.length"
        class="machine-grid"
        :machines="recentMachines"
        :sessions="sessions"
        :workspace-sessions="workspaceSessions"
        :connecting-name="connectingName"
        :filter-keyword="machineKeyword"
        layout="grid"
        variant="cards"
        flat
        show-context-menu
        empty-text="无匹配机器"
        @connect="onConnectMachine"
        @focus-session="(id) => $emit('focus-session', id)"
        @edit-machine="(m) => $emit('edit-machine', m)"
        @copy-machine="(m) => $emit('copy-machine', m)"
        @delete-machine="(m) => $emit('delete-machine', m)"
        @toggle-pin="onTogglePin"
      />

      <SectionTitle title="全部主机" :count="(machines || []).length">
        <template #icon>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" />
          </svg>
        </template>
        <template v-if="availableTags.length" #actions>
          <AppChip :active="!selectedTags.length" @click="selectedTags = []">全部</AppChip>
          <AppChip
            v-for="t in availableTags"
            :key="t"
            :active="selectedTags.includes(t)"
            @click="toggleTagFilter(t)"
          >{{ t }}</AppChip>
        </template>
      </SectionTitle>
      <MachineConnectList
        class="machine-grid"
        :machines="filteredMachines"
        :sessions="sessions"
        :workspace-sessions="workspaceSessions"
        :connecting-name="connectingName"
        :filter-keyword="machineKeyword"
        layout="grid"
        variant="cards"
        show-context-menu
        empty-text="无匹配机器"
        @connect="onConnectMachine"
        @focus-session="(id) => $emit('focus-session', id)"
        @edit-machine="(m) => $emit('edit-machine', m)"
        @copy-machine="(m) => $emit('copy-machine', m)"
        @delete-machine="(m) => $emit('delete-machine', m)"
        @toggle-pin="onTogglePin"
      />
    </template>

    <div v-else-if="!hasProjects" class="home-empty">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <rect x="2" y="4" width="20" height="7" rx="2" /><rect x="2" y="13" width="20" height="7" rx="2" /><path d="M6 7.5h.01M6 16.5h.01" />
      </svg>
      <p class="home-empty-title">配置你的主机</p>
      <p class="home-empty-desc">保存主机后可快速连接到服务器、虚拟机与容器</p>
      <AppButton variant="primary" @click="$emit('add-machine')">新建主机</AppButton>
    </div>

    <div v-if="showHomeEmpty" class="home-empty show">
      <p class="home-empty-title">无匹配结果</p>
      <p class="home-empty-desc">试试其他关键词或标签</p>
    </div>
  </div>
</template>

<script>
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'
import * as App from '../../wailsjs/go/app/App'
import { machineMatchesKeyword, isMachineConnecting, collectMachineTags, machineMatchesTags } from '../utils/machineGroups'
import { parseQuickConnectTarget, findMachineForQuickConnect } from '../utils/quickConnect'
import { onChromeTitleDblActivate, onChromeTitlePointerDown } from '../utils/windowChrome'
import { useConfigFileMenu } from '../composables/useConfigFileMenu'
import MachineConnectList from './shell/MachineConnectList.vue'
import { AppSearch, AppChip, AppButton, ProjectCard, SectionTitle } from './ui'

export default {
  name: 'HomePage',
  components: { MachineConnectList, AppSearch, AppChip, AppButton, ProjectCard, SectionTitle },
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
    'open-config-editor',
  ],
  setup(props, { emit }) {
    const machineKeyword = ref('')
    const selectedTags = ref([])
    const historyRecords = ref([])
    const searchInputRef = ref(null)

    const {
      configFiles,
      currentConfig,
      configMenuOpen,
      basename,
      toggleConfigMenu,
      closeConfigMenu,
      onConfigCommand: runConfigCommand,
    } = useConfigFileMenu({
      onEditPipeline: () => emit('open-config-editor'),
      onReload: () => emit('refresh'),
    })

    const onConfigCommand = (cmd) => runConfigCommand(cmd)

    const focusSearchInput = async () => {
      await nextTick()
      searchInputRef.value?.focus?.()
    }

    const hasProjects = computed(() => (props.projects || []).length > 0)

    const availableTags = computed(() => collectMachineTags(props.machines || []))

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

    const liveSessions = computed(() =>
      (props.workspaceSessions || []).filter((s) => s?.machineName && !String(s.machineName).startsWith('__pending__')),
    )

    const liveSessionCount = computed(() => {
      const live = liveSessions.value.filter((s) => s.connected || s.connecting).length
      return live || liveSessions.value.length
    })

    const showHomeEmpty = computed(() => {
      const kw = machineKeyword.value.trim()
      if (!kw && !selectedTags.value.length) return false
      const hasProjHit = hasProjects.value && filteredProjects.value.length > 0
      const hasMachHit = filteredMachines.value.length > 0
      return !hasProjHit && !hasMachHit
    })

    const quickTarget = computed(() => parseQuickConnectTarget(machineKeyword.value))
    const quickMatched = computed(() =>
      findMachineForQuickConnect(props.machines || [], quickTarget.value),
    )
    const quickConnectHint = computed(() => {
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

    const toggleTagFilter = (tag) => {
      const t = String(tag || '').trim()
      if (!t) return
      if (selectedTags.value.includes(t)) {
        selectedTags.value = selectedTags.value.filter((x) => x !== t)
      } else {
        selectedTags.value = [...selectedTags.value, t]
      }
    }

    const loadHistory = async () => {
      try {
        historyRecords.value = (await App.GetShellHistory()) || []
      } catch {
        historyRecords.value = []
      }
    }

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
      if (quickConnectHint.value) {
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

    onMounted(() => {
      loadHistory()
      document.addEventListener('click', closeConfigMenu)
    })

    onUnmounted(() => {
      document.removeEventListener('click', closeConfigMenu)
    })

    return {
      machineKeyword,
      selectedTags,
      hasProjects,
      availableTags,
      filteredProjects,
      filteredMachines,
      pinnedMachines,
      recentMachines,
      liveSessionCount,
      quickConnectHint,
      showHomeEmpty,
      searchInputRef,
      onChromeTitleDblActivate,
      onChromeTitlePointerDown,
      toggleTagFilter,
      onConnectMachine,
      onQuickConnect,
      onSearchEnter,
      onTogglePin,
      handleRefresh,
      focusSearchInput,
      configFiles,
      currentConfig,
      configMenuOpen,
      basename,
      toggleConfigMenu,
      onConfigCommand,
    }
  },
}
</script>

<style scoped>
.home-header-spacer {
  flex: 1;
}

.home-header {
  position: relative;
}

.home-header-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.home-header-actions :deep(.home-toolbar-btn) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 36px;
  min-height: 36px;
  padding: 0 12px;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--fg-2);
  font-size: 13px;
  font-weight: 500;
  box-shadow: none;
  transition: background 0.13s ease, border-color 0.13s ease, color 0.13s ease;
}

.home-header-actions :deep(.home-toolbar-btn:hover) {
  border-color: var(--border-strong);
  background: var(--surface-2);
  color: var(--fg);
}

.home-header-actions :deep(.home-toolbar-btn--icon) {
  width: 36px;
  min-width: 36px;
  padding: 0;
}

.home-header-actions :deep(.home-toolbar-btn svg) {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
}

.home-config-dropdown {
  position: absolute;
  top: calc(100% - 4px);
  right: 18px;
  min-width: 200px;
  z-index: 200;
}

.home-config-dropdown .dd-item svg {
  width: 12px;
  height: 12px;
  flex-shrink: 0;
}

.home-config-dropdown .dd-item:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.home-search-wrap {
  flex: 1;
  max-width: 560px;
}

.home-empty-title {
  margin: 0 0 6px;
  font-size: 15px;
  font-weight: 600;
  color: var(--fg-2);
}

.home-empty-desc {
  margin: 0 0 14px;
  font-size: 13px;
  color: var(--muted);
}

.quick-connect-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 8px 2px 12px;
  padding: 8px 12px;
  border-radius: 9px;
  border: 1px solid color-mix(in oklch, var(--accent) 28%, var(--border));
  background: var(--accent-soft);
}

.quick-connect-text {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  color: var(--fg-2);
}

.quick-connect-btn {
  flex-shrink: 0;
}

.home-empty.show {
  display: block;
}

@media (max-width: 720px) {
  .home-header {
    flex-wrap: wrap;
  }

  .home-search-wrap {
    flex: 1 1 100%;
    max-width: none;
  }
}
</style>
