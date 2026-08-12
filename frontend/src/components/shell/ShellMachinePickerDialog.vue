<template>
  <el-dialog
    v-model="visibleProxy"
    title="连接 / 快速切换"
    width="600px"
    class="machine-picker-dialog"
    append-to-body
  >
    <div class="picker-shell">
      <div class="app-toolbar picker-toolbar">
        <el-input
          ref="searchInputRef"
          v-model="keyword"
          clearable
          :placeholder="searchPlaceholder"
          size="small"
          class="app-toolbar-search"
          @keydown="onSearchKeydown"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
          <template #suffix>
            <span class="picker-shortcut-hint">{{ shortcutHint }}</span>
          </template>
        </el-input>
        <div class="icon-actions">
          <el-dropdown trigger="click" @command="onLocalShellCommand">
            <el-button class="picker-tool-btn" size="small" circle :icon="Monitor" title="本机终端" />
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="">默认本机终端</el-dropdown-item>
                <el-dropdown-item
                  v-for="opt in localShellOptions"
                  :key="opt.id || opt.command"
                  :command="opt.command"
                >
                  {{ opt.name }}{{ opt.isDefault ? '（默认）' : '' }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-tooltip content="添加机器" placement="top">
            <el-button class="picker-tool-btn" size="small" circle :icon="Plus" @click="$emit('add-machine')" />
          </el-tooltip>
        </div>
      </div>

      <div v-if="!isSearching" class="picker-segment" role="tablist">
        <button
          type="button"
          class="picker-segment-btn"
          :class="{ active: activeTab === 'sessions' }"
          role="tab"
          :aria-selected="activeTab === 'sessions'"
          @click="activeTab = 'sessions'"
        >
          <el-icon :size="14"><Monitor /></el-icon>
          <span class="picker-segment-label">已打开</span>
          <span class="picker-segment-count">{{ openSessionCount }}</span>
        </button>
        <button
          type="button"
          class="picker-segment-btn"
          :class="{ active: activeTab === 'history' }"
          role="tab"
          :aria-selected="activeTab === 'history'"
          @click="activeTab = 'history'"
        >
          <el-icon :size="14"><Clock /></el-icon>
          <span class="picker-segment-label">最近连接</span>
          <span class="picker-segment-count">{{ historyRecords.length }}</span>
        </button>
        <button
          type="button"
          class="picker-segment-btn"
          :class="{ active: activeTab === 'machines' }"
          role="tab"
          :aria-selected="activeTab === 'machines'"
          @click="activeTab = 'machines'"
        >
          <el-icon :size="14"><List /></el-icon>
          <span class="picker-segment-label">全部机器</span>
          <span class="picker-segment-count">{{ machines.length }}</span>
        </button>
      </div>

      <div class="picker-panel">
        <!-- 搜索：会话 / 本机 / 批量 / 最近 / 机器 统一列表 -->
        <div v-if="isSearching" class="picker-pane picker-pane--search">
          <div class="picker-pane-head">
            <span class="picker-pane-hint">{{ searchPaneHint }}</span>
          </div>

          <div v-if="!flatItems.length" class="picker-empty compact">
            <p class="picker-empty-desc">没有匹配「{{ keyword.trim() }}」的标签、本机或机器</p>
          </div>

          <div v-else class="picker-search-scroll" ref="listScrollRef">
            <template v-for="(item, i) in flatItems" :key="item.id">
              <div
                v-if="item.kind === 'header'"
                class="picker-search-section-head"
              >
                <span class="picker-search-section-title">{{ item.title }}</span>
                <span v-if="item.count != null" class="picker-search-section-count">{{ item.count }}</span>
              </div>
              <button
                v-else
                type="button"
                class="picker-nav-item"
                :class="{ selected: i === selectedIdx }"
                @mouseenter="selectedIdx = i"
                @click="runItem(item)"
              >
                <span class="picker-nav-kind">{{ item.kindLabel }}</span>
                <span class="picker-nav-main">
                  <span class="picker-nav-title">{{ item.title }}</span>
                  <span v-if="item.subtitle" class="picker-nav-sub">{{ item.subtitle }}</span>
                </span>
              </button>
            </template>
          </div>
        </div>

        <!-- 已打开：切标签 + 本机 -->
        <div v-show="!isSearching && activeTab === 'sessions'" class="picker-pane">
          <div class="picker-pane-head">
            <span class="picker-pane-hint">聚焦已开标签，或新开本机终端</span>
          </div>
          <div v-if="!sessionNavItems.length" class="picker-empty">
            <el-icon :size="36" class="picker-empty-icon"><Monitor /></el-icon>
            <p class="picker-empty-title">暂无已打开标签</p>
            <p class="picker-empty-desc">连接机器后会出现在这里，也可直接打开本机</p>
            <el-button size="small" type="primary" @click="onAddLocal">打开本机终端</el-button>
          </div>
          <div v-else class="picker-nav-scroll" ref="listScrollRef">
            <button
              v-for="(item, i) in sessionNavItems"
              :key="item.id"
              type="button"
              class="picker-nav-item"
              :class="{ selected: i === selectedIdx }"
              @mouseenter="selectedIdx = i"
              @click="runItem(item)"
            >
              <span class="picker-nav-kind">{{ item.kindLabel }}</span>
              <span class="picker-nav-main">
                <span class="picker-nav-title">{{ item.title }}</span>
                <span v-if="item.subtitle" class="picker-nav-sub">{{ item.subtitle }}</span>
              </span>
            </button>
          </div>
        </div>

        <div v-show="!isSearching && activeTab === 'history'" class="picker-pane">
          <div v-if="historyRecords.length" class="picker-pane-head">
            <span class="picker-pane-hint">{{ historyPaneHint }}</span>
            <button type="button" class="picker-clear-btn" @click="$emit('clear-history')">清空历史</button>
          </div>

          <div v-if="!historyRecords.length" class="picker-empty">
            <el-icon :size="36" class="picker-empty-icon"><Clock /></el-icon>
            <p class="picker-empty-title">暂无连接历史</p>
            <p class="picker-empty-desc">连接过的机器会出现在这里，方便快速重连</p>
            <el-button size="small" type="primary" plain @click="activeTab = 'machines'">浏览全部机器</el-button>
          </div>

          <ShellHistoryList
            v-else-if="historyFiltered.length"
            :records="historyRecords"
            :sessions="sessions"
            :workspace-sessions="workspaceSessions"
            :keyword="keyword"
            :show-head="false"
            embedded
            @connect="onConnect"
            @remove="(row) => $emit('remove-history', row)"
          />

          <div v-else class="picker-empty compact">
            <p class="picker-empty-desc">没有匹配「{{ keyword }}」的记录</p>
          </div>
        </div>

        <div v-show="!isSearching && activeTab === 'machines'" class="picker-pane">
          <div v-if="machines.length" class="picker-pane-head">
            <span class="picker-pane-hint">{{ machinesPaneHint }}</span>
          </div>

          <div v-if="!machines.length" class="picker-empty">
            <el-icon :size="36" class="picker-empty-icon"><Monitor /></el-icon>
            <p class="picker-empty-title">暂无机器配置</p>
            <p class="picker-empty-desc">添加机器后即可在此发起 SSH 连接</p>
            <el-button size="small" type="primary" @click="$emit('add-machine')">添加机器</el-button>
          </div>

          <div v-else class="picker-pane-body">
            <MachineConnectList
              :machines="filteredMachines"
              :sessions="sessions"
              :workspace-sessions="workspaceSessions"
              :connecting-name="connectingName"
              :filter-keyword="keyword"
              show-edit
              show-context-menu
              :empty-text="keyword ? `没有匹配「${keyword}」的机器` : '暂无机器配置'"
              @connect="onConnect"
              @open-window="(m) => $emit('open-window', m)"
              @edit-machine="(m) => $emit('edit-machine', m)"
              @copy-machine="(m) => $emit('copy-machine', m)"
              @delete-machine="(m) => $emit('delete-machine', m)"
            />
          </div>
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<script>
import { computed, ref, watch, nextTick, onMounted } from 'vue'
import { Monitor, Plus, Search, Clock, List } from '@element-plus/icons-vue'
import * as App from '../../../wailsjs/go/app/App'
import {
  machineMatchesKeyword,
  normalizeMachineTags,
  collectMachineTags,
  formatMachineAddr,
} from '../../utils/machineGroups'
import { DEFAULT_SHORTCUTS, formatShortcut } from '../../utils/shortcuts'
import MachineConnectList from './MachineConnectList.vue'
import ShellHistoryList from './ShellHistoryList.vue'

const isLocalSession = (s) =>
  s?.kind === 'local' || String(s?.machineName || '').startsWith('local')

const PICKER_TAB_KEY = 'flashdock.shell.pickerActiveTab'
const PICKER_TABS = ['sessions', 'history', 'machines']

const readLastPickerTab = () => {
  try {
    const v = localStorage.getItem(PICKER_TAB_KEY)
    return PICKER_TABS.includes(v) ? v : ''
  } catch {
    return ''
  }
}

const writeLastPickerTab = (tab) => {
  if (!PICKER_TABS.includes(tab)) return
  try {
    localStorage.setItem(PICKER_TAB_KEY, tab)
  } catch {
    /* ignore */
  }
}

export default {
  name: 'ShellMachinePickerDialog',
  components: { MachineConnectList, ShellHistoryList, Search, Clock, List, Monitor },
  props: {
    modelValue: { type: Boolean, default: false },
    machines: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
    workspaceSessions: { type: Array, default: () => [] },
    connectingName: { type: String, default: '' },
    historyRecords: { type: Array, default: () => [] },
    initialTab: { type: String, default: '' },
  },
  emits: [
    'update:modelValue',
    'connect',
    'focus-session',
    'connect-machines',
    'edit-machine',
    'copy-machine',
    'delete-machine',
    'add-machine',
    'add-local',
    'add-local-command',
    'open-window',
    'clear-history',
    'remove-history',
    'open',
  ],
  setup(props, { emit }) {
    const keyword = ref('')
    const activeTab = ref(readLastPickerTab() || 'sessions')
    const selectedIdx = ref(0)
    const searchInputRef = ref(null)
    const listScrollRef = ref(null)
    const localShellOptions = ref([])

    watch(activeTab, (tab) => {
      writeLastPickerTab(tab)
    })

    const loadLocalShells = async () => {
      try {
        const list = await App.ListLocalShells()
        localShellOptions.value = Array.isArray(list) ? list : []
      } catch {
        localShellOptions.value = []
      }
    }

    onMounted(() => { loadLocalShells() })
    watch(
      () => props.modelValue,
      (open) => { if (open) loadLocalShells() },
    )

    const visibleProxy = computed({
      get: () => props.modelValue,
      set: (v) => emit('update:modelValue', v),
    })

    const isSearching = computed(() => !!String(keyword.value || '').trim())
    const shortcutHint = computed(() => formatShortcut(DEFAULT_SHORTCUTS.connectionManager))
    const searchPlaceholder = computed(
      () => `搜索主机、已开标签或本机…（${shortcutHint.value}）`,
    )

    const openSessions = computed(() =>
      (props.workspaceSessions || []).filter((s) => s?.machineName),
    )
    const openSessionCount = computed(() => openSessions.value.length)

    const filteredMachines = computed(() => {
      const kw = keyword.value
      let list = props.machines || []
      if (String(kw || '').trim()) {
        list = list.filter((m) => machineMatchesKeyword(m, kw))
      }
      return list
    })

    const historyFiltered = computed(() => {
      const kw = String(keyword.value || '').trim().toLowerCase()
      const list = props.historyRecords || []
      if (!kw) return list
      return list.filter((row) => {
        const hay = `${row.machineName || ''} ${row.user || ''} ${row.host || ''} ${row.port || ''}`.toLowerCase()
        return hay.includes(kw)
      })
    })

    const filteredOpenSessions = computed(() => {
      const kw = String(keyword.value || '').trim().toLowerCase()
      const list = openSessions.value
      if (!kw) return list
      return list.filter((s) => {
        const hay = `${s.tabLabel || ''} ${s.machineName || ''} ${s.configName || ''}`.toLowerCase()
        return hay.includes(kw)
      })
    })

    const localMatchesSearch = computed(() => {
      const kw = String(keyword.value || '').trim().toLowerCase()
      if (!kw) return true
      return ['本机', 'local', 'terminal', 'shell', 'powershell', 'cmd'].some((t) => t.includes(kw) || kw.includes(t))
    })

    const batchItems = computed(() => {
      const kw = String(keyword.value || '').trim().toLowerCase()
      if (!kw) return []
      const out = []
      const allTags = collectMachineTags(props.machines || [])
      const matchedTags = allTags.filter((t) => t.toLowerCase().includes(kw) || kw.includes(t.toLowerCase()))
      for (const tag of matchedTags.slice(0, 5)) {
        const names = (props.machines || [])
          .filter((m) => normalizeMachineTags(m.tags).includes(tag))
          .map((m) => m.name)
          .filter(Boolean)
        if (names.length < 2) continue
        out.push({
          id: `batch-tag-${tag}`,
          kind: 'batch',
          kindLabel: '批量',
          title: `连接标签「${tag}」`,
          subtitle: `${names.length} 台 · 错峰连接`,
          run: () => emit('connect-machines', names),
        })
      }
      return out
    })

    const makeSessionItem = (s) => ({
      id: `sess-${s.machineName}`,
      kind: 'session',
      kindLabel: isLocalSession(s) ? '本机' : (s.connected ? '标签' : '标签·断'),
      title: s.tabLabel || s.machineName,
      subtitle: s.configName && s.configName !== s.machineName
        ? s.configName
        : (s.connected ? '点击聚焦' : '点击聚焦并重连'),
      run: () => {
        visibleProxy.value = false
        emit('focus-session', s.machineName)
      },
    })

    const localItem = {
      id: 'local-new',
      kind: 'local',
      kindLabel: '本机',
      title: '打开本机终端',
      subtitle: '新建本地 Shell 标签',
      run: () => {
        emit('add-local')
        visibleProxy.value = false
      },
    }

    const sessionNavItems = computed(() => {
      const items = filteredOpenSessions.value.map(makeSessionItem)
      items.push({ ...localItem })
      return items
    })

    const flatItems = computed(() => {
      if (!isSearching.value) return []
      const rows = []
      const pushHeader = (id, title, count) => {
        rows.push({ id: `hdr-${id}`, kind: 'header', title, count })
      }
      const sessions = filteredOpenSessions.value
      if (sessions.length) {
        pushHeader('sessions', '已打开', sessions.length)
        sessions.forEach((s) => rows.push(makeSessionItem(s)))
      }
      if (localMatchesSearch.value) {
        pushHeader('local', '本机', 1)
        rows.push({ ...localItem })
      }
      if (batchItems.value.length) {
        pushHeader('batch', '批量', batchItems.value.length)
        batchItems.value.forEach((b) => rows.push(b))
      }
      if (historyFiltered.value.length) {
        pushHeader('history', '最近连接', historyFiltered.value.length)
        historyFiltered.value.forEach((row) => {
          rows.push({
            id: `hist-${row.machineName}-${row.connectedAt || ''}`,
            kind: 'history',
            kindLabel: '最近',
            title: row.machineName,
            subtitle: [
              row.user && row.host ? `${row.user}@${row.host}` : (row.host || row.user || ''),
              row.port ? `:${row.port}` : '',
            ].join('') || '点击连接',
            run: () => onConnect(row.machineName),
          })
        })
      }
      if (filteredMachines.value.length) {
        pushHeader('machines', '全部机器', filteredMachines.value.length)
        filteredMachines.value.forEach((m) => {
          const tags = normalizeMachineTags(m.tags)
          rows.push({
            id: `m-${m.id || m.name}`,
            kind: 'machine',
            kindLabel: '机器',
            title: m.name,
            subtitle: [formatMachineAddr(m), tags.join(' · ')].filter(Boolean).join(' | '),
            run: () => onConnect(m.name),
          })
        })
      }
      return rows
    })

    const selectableItems = computed(() => {
      if (isSearching.value) {
        return flatItems.value.map((item, index) => ({ item, index })).filter(({ item }) => item.kind !== 'header')
      }
      if (activeTab.value === 'sessions') {
        return sessionNavItems.value.map((item, index) => ({ item, index }))
      }
      return []
    })

    const historyPaneHint = computed(() => {
      const total = props.historyRecords.length
      return `共 ${total} 条，点击或 Enter 快速连接`
    })

    const machinesPaneHint = computed(() => {
      const total = (props.machines || []).length
      return `共 ${total} 台，按分组浏览或搜索`
    })

    const searchPaneHint = computed(() => {
      const n = selectableItems.value.length
      if (!n) return '无匹配结果'
      return `${n} 项可切换 · ↑↓ 选择 · Enter 确认`
    })

    const resolveDefaultTab = () => {
      // 调用方显式指定（如空历史点「打开连接」）优先
      if (PICKER_TABS.includes(props.initialTab)) return props.initialTab
      const last = readLastPickerTab()
      if (last) return last
      if ((props.workspaceSessions || []).length) return 'sessions'
      return (props.historyRecords || []).length ? 'history' : 'machines'
    }

    const focusSearch = async () => {
      await nextTick()
      requestAnimationFrame(() => {
        searchInputRef.value?.focus?.()
      })
    }

    const resetSelection = () => {
      const first = selectableItems.value[0]
      selectedIdx.value = first ? first.index : 0
    }

    const onDialogOpen = () => {
      activeTab.value = resolveDefaultTab()
      keyword.value = ''
      emit('open')
      nextTick(() => {
        resetSelection()
        focusSearch()
      })
    }

    watch(
      () => props.modelValue,
      (visible) => {
        if (!visible) {
          keyword.value = ''
          return
        }
        onDialogOpen()
      },
    )

    watch([isSearching, activeTab, () => flatItems.value.length, () => sessionNavItems.value.length], () => {
      resetSelection()
    })

    const onConnect = (name) => {
      emit('connect', name)
      visibleProxy.value = false
    }

    const runItem = (item) => {
      if (!item?.run || item.kind === 'header') return
      item.run()
    }

    const applySelected = () => {
      if (isSearching.value || activeTab.value === 'sessions') {
        const item = isSearching.value
          ? flatItems.value[selectedIdx.value]
          : sessionNavItems.value[selectedIdx.value]
        if (item && item.kind !== 'header') {
          runItem(item)
          return
        }
        const next = selectableItems.value.find((x) => x.index >= selectedIdx.value) || selectableItems.value[0]
        if (next) runItem(next.item)
        return
      }
      connectFirstLegacy()
    }

    const connectFirstLegacy = () => {
      if (activeTab.value === 'history') {
        const first = historyFiltered.value[0]
        if (first) onConnect(first.machineName)
        return
      }
      const first = filteredMachines.value[0]
      if (first) onConnect(first.name)
    }

    const moveSel = (delta) => {
      const opts = selectableItems.value
      if (!opts.length) return
      const curPos = opts.findIndex((x) => x.index === selectedIdx.value)
      const nextPos = curPos < 0 ? 0 : (curPos + delta + opts.length) % opts.length
      selectedIdx.value = opts[nextPos].index
      nextTick(() => {
        const root = listScrollRef.value
        if (!root) return
        const el = root.querySelector('.picker-nav-item.selected')
        el?.scrollIntoView?.({ block: 'nearest' })
      })
    }

    const onSearchKeydown = (e) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        moveSel(1)
        return
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        moveSel(-1)
        return
      }
      if (e.key === 'Enter') {
        e.preventDefault()
        applySelected()
      }
    }

    const onAddLocal = () => {
      emit('add-local')
      visibleProxy.value = false
    }

    const onLocalShellCommand = (command) => {
      const cmd = String(command || '').trim()
      if (!cmd) {
        onAddLocal()
        return
      }
      emit('add-local-command', cmd)
      visibleProxy.value = false
    }

    return {
      Monitor,
      Plus,
      searchInputRef,
      listScrollRef,
      visibleProxy,
      keyword,
      activeTab,
      selectedIdx,
      isSearching,
      shortcutHint,
      searchPlaceholder,
      openSessionCount,
      filteredMachines,
      historyFiltered,
      sessionNavItems,
      flatItems,
      historyPaneHint,
      machinesPaneHint,
      searchPaneHint,
      onConnect,
      onAddLocal,
      onLocalShellCommand,
      localShellOptions,
      onSearchKeydown,
      runItem,
    }
  },
}
</script>

<style scoped>
.picker-shell {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.picker-toolbar {
  margin-bottom: 0;
}

.picker-shortcut-hint {
  font-size: 11px;
  color: var(--app-text-muted, #909399);
  padding-right: 4px;
  white-space: nowrap;
}

.picker-segment {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 6px;
  padding: 4px;
  border-radius: var(--app-radius-lg, 10px);
  background: color-mix(in srgb, var(--app-panel-bg, #f5f7fa) 88%, transparent);
  border: 1px solid var(--app-border, #e4e7ed);
}

.picker-segment-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 36px;
  padding: 0 10px;
  border: none;
  border-radius: var(--app-radius-md, 8px);
  background: transparent;
  color: var(--app-text-secondary, #606266);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease, box-shadow 0.15s ease;
}

.picker-segment-btn:hover {
  color: var(--app-text, #303133);
  background: color-mix(in srgb, var(--app-card-bg, #fff) 70%, transparent);
}

.picker-segment-btn.active {
  color: var(--app-accent-color, #409eff);
  background: var(--app-card-bg, #fff);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}

.picker-segment-label {
  line-height: 1;
}

.picker-segment-count {
  min-width: 18px;
  padding: 2px 6px;
  border-radius: 999px;
  font-size: 11px;
  line-height: 1.2;
  font-weight: 600;
  color: var(--app-text-muted, #909399);
  background: color-mix(in srgb, var(--app-text-muted, #909399) 12%, transparent);
}

.picker-segment-btn.active .picker-segment-count {
  color: var(--app-accent-color, #409eff);
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 14%, transparent);
}

.picker-panel {
  min-height: 360px;
  max-height: min(52vh, 440px);
  display: flex;
  flex-direction: column;
  border: 1px solid var(--app-border, #e4e7ed);
  border-radius: var(--app-radius-xl, 12px);
  background: var(--app-card-bg, #fff);
  overflow: hidden;
}

.picker-pane {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.picker-pane-head {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 14px 8px;
  border-bottom: 1px solid color-mix(in srgb, var(--app-border, #e4e7ed) 70%, transparent);
}

.picker-pane-hint {
  font-size: 12px;
  color: var(--app-text-muted, #909399);
  line-height: 1.4;
}

.picker-clear-btn {
  flex-shrink: 0;
  border: none;
  background: transparent;
  padding: 0;
  font-size: 12px;
  color: var(--app-text-muted, #909399);
  cursor: pointer;
  transition: color 0.12s ease;
}

.picker-clear-btn:hover {
  color: #f56c6c;
}

.picker-pane-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 8px;
}

.picker-pane--search {
  min-height: 0;
}

.picker-search-scroll,
.picker-nav-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 8px 10px;
}

.picker-search-section-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 8px 4px;
}

.picker-search-section-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--app-text-secondary, #606266);
}

.picker-search-section-count {
  min-width: 16px;
  padding: 1px 6px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  line-height: 1.3;
  color: var(--app-accent-color, #409eff);
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 14%, transparent);
}

.picker-nav-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  width: 100%;
  text-align: left;
  border: none;
  border-radius: var(--app-radius-md, 8px);
  background: transparent;
  padding: 10px 12px;
  cursor: pointer;
  color: inherit;
}

.picker-nav-item.selected,
.picker-nav-item:hover {
  background: var(--app-accent-bg, color-mix(in srgb, var(--app-accent-color, #409eff) 12%, transparent));
}

.picker-nav-kind {
  flex-shrink: 0;
  margin-top: 2px;
  font-size: 10px;
  line-height: 1;
  padding: 4px 6px;
  border-radius: 999px;
  color: var(--app-text-muted);
  border: 1px solid var(--app-border);
  background: var(--app-panel-bg, transparent);
}

.picker-nav-main {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.picker-nav-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
}

.picker-nav-sub {
  font-size: 12px;
  color: var(--app-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.picker-pane :deep(.history-list-wrap) {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 8px;
}

.picker-pane :deep(.history-scroll) {
  max-height: none;
  border: none;
  background: transparent;
  padding: 0;
}

.picker-pane :deep(.ml-list) {
  border: none;
  background: transparent;
  padding: 0;
  gap: 4px;
}

.picker-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 36px 24px;
  text-align: center;
}

.picker-empty.compact {
  padding: 28px 16px;
}

.picker-empty-icon {
  color: var(--app-text-muted, #909399);
  opacity: 0.65;
}

.picker-empty-title {
  margin: 4px 0 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text, #303133);
}

.picker-empty-desc {
  margin: 0 0 6px;
  font-size: 12px;
  color: var(--app-text-muted, #909399);
  line-height: 1.5;
  max-width: 280px;
}
</style>

<!-- append-to-body 弹窗：统一圆钮盒模型 -->
<style>
.machine-picker-dialog .picker-tool-btn.el-button.is-circle {
  width: 28px !important;
  height: 28px !important;
  min-width: 28px !important;
  max-width: 28px !important;
  padding: 0 !important;
  margin: 0 !important;
  box-sizing: border-box !important;
  border-style: solid !important;
  border-width: 1px !important;
  font-size: 14px !important;
  line-height: 1 !important;
}

.machine-picker-dialog .picker-tool-btn.el-button--default.is-circle {
  background-color: var(--el-fill-color-light) !important;
  border-color: var(--el-border-color) !important;
}

.machine-picker-dialog .picker-tool-btn.el-button.is-circle .el-icon {
  width: 14px !important;
  height: 14px !important;
  font-size: 14px !important;
  margin: 0 !important;
}

.machine-picker-dialog .el-dialog__body {
  padding-top: 12px;
}
</style>
