<template>
  <div class="shell-terminal-tabs">
    <div
      class="tabs-bar"
      :class="{ 'is-drop-unsplit': draggingSplitPane }"
      @dragover.prevent="onTabsBarDragOver"
      @drop.prevent="onTabsBarDrop"
      @dblclick="onChromeTitleDblActivate"
      @mousedown="onChromeTitlePointerDown"
    >
      <el-button class="home-btn" size="small" text title="返回首页" @click="$emit('back')">
        <el-icon :size="14">
          <ArrowLeft />
        </el-icon>
      </el-button>

      <div class="tabs-bar-left">
        <div v-if="sessions.length" class="custom-session-tabs">
          <div v-for="session in orderedSessions" :key="session.machineName" class="session-tab" :class="{
            active: activeTab === session.machineName,
            'in-split': hasSplitGroup && splitSessionIds.includes(session.machineName),
            'drop-before': dropReorderTarget === session.machineName && !dropReorderAfter,
            'drop-after': dropReorderTarget === session.machineName && dropReorderAfter,
            'is-reorder-dragging': tabReorderFrom === session.machineName,
          }" :data-session-id="session.machineName" @click="onTabClick(session.machineName)"
            @mousedown="onTabMouseDown($event, session.machineName)"
            @contextmenu.prevent="onTabContextMenu($event, session.machineName)">
            <div class="session-tab-main">
              <span class="session-tab-status" :class="tabStatusClass(session)" aria-hidden="true" />
              <span class="session-tab-label">{{ tabDisplayLabel(session) }}</span>
            </div>
            <button type="button" class="session-tab-close" title="关闭" @mousedown.stop
              @click.stop="onTabRemove(session.machineName)">
              ×
            </button>
          </div>
        </div>

        <div class="add-session-wrap">
          <el-button class="add-session-btn" size="small" text title="连接 / 快速切换（Ctrl+E）" @click="$emit('open-picker')">
            <el-icon :size="15">
              <Plus />
            </el-icon>
          </el-button>
          <el-dropdown trigger="hover" :show-timeout="120" :hide-timeout="160" @command="onAddCommand">
            <el-button class="add-session-more" size="small" text title="更多连接方式">
              <el-icon :size="12">
                <ArrowDown />
              </el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="remote">连接 / 快速切换…</el-dropdown-item>
                <el-dropdown-item command="local">本机终端</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
      <div class="tabs-bar-spacer" aria-hidden="true">
        <span v-if="draggingSplitPane" class="unsplit-hint">拖到此处移出分屏</span>
      </div>
      <div class="tabs-bar-right">
        <ModeSwitcher
          v-if="hasProjects || hasTask"
          compact
          float-align="end"
          model-value="shell"
          :has-projects="hasProjects"
          :has-machines="hasMachines"
          :has-task="hasTask"
          :task-running="taskRunning"
          :connected-count="connectedCount"
          :projects="projects"
          :selected-project-name="selectedProjectName"
          :sessions="sessions"
          :active-session-id="activeTab"
          @change="(v) => $emit('change-view', v)"
          @select-project="(p) => $emit('select-project', p)"
          @focus-session="(id) => $emit('focus-session', id)"
        />
        <AppChromeIcons />
      </div>
    </div>

    <div v-if="sessions.length === 0" class="empty-slot">
      <slot name="empty" />
    </div>
    <template v-else>
      <div class="terminal-body" :class="{ 'has-aux-bars': broadcastEnabled || composeEnabled }">
        <!-- 浮层：不占文档流，避免挤压终端高度触发 fit 抖动 -->
        <div v-if="broadcastEnabled || composeEnabled" class="shell-aux-bars">
          <ShellBroadcastBar
            v-if="broadcastEnabled"
            :enabled="broadcastEnabled"
            :targets="broadcastTargets"
            :sessions="sessions"
            @update:enabled="(v) => $emit('update:broadcast-enabled', v)"
            @update:targets="(v) => $emit('update:broadcast-targets', v)"
          />
          <ShellComposeBar
            v-if="composeEnabled"
            :enabled="composeEnabled"
            :session-id="activeTab"
            :broadcast-enabled="broadcastEnabled"
            :broadcast-targets="broadcastTargets"
            @update:enabled="(v) => (composeEnabled = v)"
          />
        </div>

        <div v-if="sessions.length" class="terminal-inline-actions">
          <el-tooltip v-if="connectedCount >= 1"
            :content="broadcastEnabled ? '关闭命令广播 (Esc)' : '开启命令广播'" placement="bottom">
            <el-button class="broadcast-toggle" size="small" text :class="{ active: broadcastEnabled }"
              @click="toggleBroadcast">
              <el-icon :size="15">
                <Promotion />
              </el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip :content="composeEnabled ? '关闭撰写栏' : '开启撰写栏（多行命令）'" placement="bottom">
            <el-button class="compose-toggle" size="small" text :class="{ active: composeEnabled }"
              @click="toggleCompose">
              <el-icon :size="15">
                <EditPen />
              </el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip content="命令面板 (历史/片段，默认 Ctrl/⌘+Shift+P)" placement="bottom">
            <el-button size="small" text title="命令面板"
              @click="$emit('open-command-palette')">
              <el-icon :size="15"><Memo /></el-icon>
            </el-button>
          </el-tooltip>
          <el-button v-if="!isLocalSession(activeTab)" class="transfer-btn" size="small" text
            title="文件传输" @click="$emit('open-transfer')">
            <el-badge :value="transferActiveCount" :hidden="!transferActiveCount" :max="99">
              <el-icon :size="15">
                <Upload />
              </el-icon>
            </el-badge>
          </el-button>
        </div>

        <div class="terminal-stack"
        :class="{
          'is-split': splitViewVisible,
          'is-drag-over': !!draggingTab && !draggingSplitPane,
          'is-pane-zoomed': splitViewVisible && !!zoomedSessionId,
        }"
        :style="splitGridStyle" @dragover.prevent="onStackDragOver" @dragleave="onStackDragLeave"
        @drop.prevent="onStackDrop">
        <div v-if="draggingTab && !draggingSplitPane" class="split-drop-overlay">
          <p class="split-drop-hint">拖到区域加入分屏（最多 4 个）</p>
          <div class="split-drop-zones">
            <div v-for="zone in dropZones" :key="zone.id" class="drop-zone"
              :class="[zone.id, { disabled: zone.disabled, active: dropTargetZone === zone.id && !!draggingTab }]"
              :data-zone-id="zone.id" @dragover.prevent="onZoneDragOver(zone)" @drop.prevent="onZoneDrop(zone)">
              {{ zone.label }}
            </div>
          </div>
        </div>

        <div v-for="session in orderedSessions" :key="session.machineName" class="terminal-pane" :class="{
          'is-active': isTerminalActive(session.machineName),
          'is-split-pane': splitViewVisible && splitSessionIds.includes(session.machineName),
          'is-split-hidden': splitViewVisible && !splitSessionIds.includes(session.machineName),
          'is-focused': activeTab === session.machineName,
          'is-zoomed': zoomedSessionId === session.machineName,
        }">
          <div v-if="splitViewVisible && splitSessionIds.includes(session.machineName)" class="split-pane-header"
            draggable="true" @dragstart="onPaneDragStart($event, session.machineName)" @dragend="onPaneDragEnd"
            @mousedown="activeTab = session.machineName"
            @dblclick.stop="togglePaneZoom(session.machineName)"
            @contextmenu.prevent="onPaneContextMenu($event, session.machineName)">
            <span class="split-pane-name" :title="paneTitle(session)">{{ tabLabel(session) }}</span>
            <button
              type="button"
              class="split-pane-zoom"
              :title="zoomedSessionId === session.machineName ? '还原分屏' : '最大化窗格'"
              @click.stop="togglePaneZoom(session.machineName)"
            >
              {{ zoomedSessionId === session.machineName ? '⊡' : '▣' }}
            </button>
            <button type="button" class="split-pane-unsplit" title="移出分屏"
              @click.stop="removeFromSplit(session.machineName)">
              ×
            </button>
          </div>
          <ShellTerminal :ref="(el) => setTerminalRef(session.machineName, el)" :machine-name="session.machineName"
            :config-name="session.configName || ''"
            :connected="!!session.connected" :connecting="!!session.connecting"
            :ever-connected="!!session.everConnected"
            :reconnecting="!!session.reconnecting"
            :reconnect-attempt="session.reconnectAttempt || 0"
            :reconnect-max="session.reconnectMax || 0"
            :reconnect-delay-sec="session.reconnectDelaySec || 0"
            :tab-label="session.tabLabel || ''"
            :host="sessionMeta(session).host"
            :user="sessionMeta(session).user"
            :jump-chain="sessionMeta(session).jumpChain"
            :proxy-jump="sessionMeta(session).proxyJump"
            :terminal-preset-override="sessionMeta(session).terminalPreset"
            :local-echo="!!sessionMeta(session).localEcho"
            :active="isTerminalActive(session.machineName)" :view-visible="viewVisible" :search-query="searchQuery"
            :in-split="splitViewVisible && splitSessionIds.includes(session.machineName)"
            :suppress-resize-observer-fit="filePanelLayoutDragging"
            @open-search="(text) => $emit('open-search', text)" @reconnect="(name) => $emit('reconnect', name)"
            @clear-cache="(name) => $emit('clear', name)" @search-result="(payload) => $emit('search-result', payload)"
            @cwd-sync="(payload) => $emit('cwd-sync', payload)" @remove-from-split="removeFromSplit"
            @exit-split="exitSplit" @focus-session="onFocusSession" />
        </div>
      </div>
      </div>

      <ul v-if="tabMenu.visible" class="pane-ctx-menu" :style="{ left: tabMenu.x + 'px', top: tabMenu.y + 'px' }"
        @mousedown.stop @click.stop>
        <li @mousedown.prevent="onTabMenuDuplicate">复制标签页</li>
        <li @mousedown.prevent="onTabMenuClose">关闭</li>
        <li :class="{ disabled: !tabMenuHasRight }" @mousedown.prevent="onTabMenuCloseRight">关闭右侧</li>
        <li @mousedown.prevent="onTabMenuCloseAll">全部关闭</li>
      </ul>

      <ul v-if="paneMenu.visible" class="pane-ctx-menu" :style="{ left: paneMenu.x + 'px', top: paneMenu.y + 'px' }"
        @mousedown.stop @click.stop>
        <li @mousedown.prevent="onPaneMenuZoom">{{ zoomedSessionId === paneMenu.sessionId ? '还原分屏' : '最大化窗格' }}</li>
        <li @mousedown.prevent="onPaneMenuRemove">移出分屏</li>
        <li @mousedown.prevent="onPaneMenuExit">取消全部分屏</li>
      </ul>

      <slot name="footer" :active-machine="activeTab" />
    </template>
  </div>
</template>

<script>
import { ref, reactive, watch, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { ArrowLeft, ArrowDown, Upload, Plus, Promotion, Memo, EditPen } from '@element-plus/icons-vue'
import ShellTerminal from './ShellTerminal.vue'
import ShellBroadcastBar from './ShellBroadcastBar.vue'
import ShellComposeBar from './ShellComposeBar.vue'
import ModeSwitcher from '../ModeSwitcher.vue'
import AppChromeIcons from '../AppChromeIcons.vue'
import { cwdBasename } from '../../utils/shellTerminalUx'
import { onChromeTitleDblActivate, onChromeTitlePointerDown } from '../../utils/windowChrome'

const MAX_SPLIT = 4
const DRAG_REORDER_PX = 4
const DRAG_SPLIT_PX = 10

const findTabTargetAt = (x, y, excludeId = '') => {
  const tabs = document.querySelectorAll('.shell-terminal-tabs .session-tab[data-session-id]')
  for (const tab of tabs) {
    const id = tab.dataset.sessionId
    if (!id || id === excludeId) continue
    const rect = tab.getBoundingClientRect()
    if (x >= rect.left && x <= rect.right && y >= rect.top && y <= rect.bottom) {
      return { id, after: x > rect.left + rect.width / 2 }
    }
  }
  return null
}

const moveTabInOrder = (order, fromId, toId, insertAfter = false) => {
  const next = [...order]
  const fromIdx = next.indexOf(fromId)
  let toIdx = next.indexOf(toId)
  if (fromIdx < 0 || toIdx < 0 || fromIdx === toIdx) return order
  next.splice(fromIdx, 1)
  if (fromIdx < toIdx) toIdx -= 1
  next.splice(insertAfter ? toIdx + 1 : toIdx, 0, fromId)
  return next
}

const findSplitZoneAt = (x, y, zones) => {
  for (const zone of zones) {
    if (zone.disabled) continue
    const el = document.querySelector(`.shell-terminal-tabs .drop-zone.${zone.id}`)
    if (!el) continue
    const rect = el.getBoundingClientRect()
    if (x >= rect.left && x <= rect.right && y >= rect.top && y <= rect.bottom) return zone.id
  }
  const stack = document.querySelector('.shell-terminal-tabs .terminal-stack')
  if (!stack) return ''
  const rect = stack.getBoundingClientRect()
  if (x >= rect.left && x <= rect.right && y >= rect.top && y <= rect.bottom) return 'stack'
  return ''
}

const isLocalSession = (name) => {
  const n = String(name || '')
  return n === 'local' || n.startsWith('local-')
}

const localTabLabel = (name) => {
  if (name === 'local') return '本机'
  const n = String(name || '').replace(/^local-/, '')
  return n ? `本机-${n}` : '本机'
}

export default {
  name: 'ShellTerminalTabs',
  components: {
    ShellTerminal,
    ShellBroadcastBar,
    ShellComposeBar,
    ModeSwitcher,
    AppChromeIcons,
    ArrowLeft,
    ArrowDown,
    Upload,
    Plus,
    Promotion,
    Memo,
    EditPen,
  },
  props: {
    sessions: { type: Array, default: () => [] },
    machines: { type: Array, default: () => [] },
    activeMachine: { type: String, default: '' },
    searchQuery: { type: String, default: '' },
    viewVisible: { type: Boolean, default: true },
    transferActiveCount: { type: Number, default: 0 },
    broadcastEnabled: { type: Boolean, default: false },
    broadcastTargets: { type: Array, default: () => [] },
    splitSessionIds: { type: Array, default: () => [] },
    /** SFTP 面板高度拖拽中：终端 ResizeObserver 不 fit，松手后 workspace 统一 fit */
    filePanelLayoutDragging: { type: Boolean, default: false },
    hasTask: { type: Boolean, default: false },
    hasProjects: { type: Boolean, default: false },
    hasMachines: { type: Boolean, default: false },
    taskRunning: { type: Boolean, default: false },
    projects: { type: Array, default: () => [] },
    selectedProjectName: { type: String, default: '' },
  },
  emits: [
    'update:activeMachine', 'close-session', 'close-sessions', 'clear', 'open-picker', 'add-local',
    'back', 'open-search', 'reconnect', 'search-result', 'open-transfer', 'open-command-palette', 'cwd-sync',
    'duplicate-session',
    'update:broadcast-enabled', 'update:broadcast-targets', 'update:split-session-ids',
    'reorder-tabs',
    'change-view', 'select-project', 'focus-session',
  ],
  setup(props, { emit, expose }) {
    const activeTab = ref(props.activeMachine)
    const terminalRefs = ref({})
    const draggingTab = ref('')
    const draggingSplitPane = ref('')
    const dropTargetZone = ref('')
    const dropReorderTarget = ref('')
    const dropReorderAfter = ref(false)
    const tabReorderFrom = ref('')
    const tabDragMoved = ref(false)
    const localTabOrder = ref([])
    const paneMenu = reactive({ visible: false, x: 0, y: 0, sessionId: '' })
    const tabMenu = reactive({ visible: false, x: 0, y: 0, sessionId: '' })
    /** 分屏最大化：仅展示该窗格 */
    const zoomedSessionId = ref('')

    const sessionMeta = (session) => {
      const key = session?.configName || session?.machineName || ''
      const m = (props.machines || []).find((x) => x?.name === key || x?.id === key)
      return {
        host: session?.host || m?.host || m?.list_host || '',
        user: session?.user || m?.user || m?.list_user || '',
        jumpChain: Array.isArray(m?.jumpChain) ? m.jumpChain : [],
        proxyJump: m?.proxyJump || '',
        terminalPreset: m?.terminalPreset || '',
        localEcho: !!m?.localEcho,
      }
    }

    const orderedSessions = computed(() => {
      const map = new Map((props.sessions || []).map((s) => [s.machineName, s]))
      const order = localTabOrder.value.length
        ? localTabOrder.value
        : (props.sessions || []).map((s) => s.machineName)
      return order.map((id) => map.get(id)).filter(Boolean)
    })

    const syncLocalTabOrderFromProps = () => {
      const ids = (props.sessions || []).map((s) => s.machineName)
      if (!ids.length) {
        localTabOrder.value = []
        return
      }
      if (!localTabOrder.value.length) {
        localTabOrder.value = [...ids]
        return
      }
      const next = localTabOrder.value.filter((id) => ids.includes(id))
      for (const id of ids) {
        if (!next.includes(id)) next.push(id)
      }
      const propsSig = ids.join('\0')
      const localSig = next.join('\0')
      if (!tabReorderFrom.value && propsSig !== localSig) {
        localTabOrder.value = [...ids]
        return
      }
      localTabOrder.value = next
    }

    watch(() => (props.sessions || []).map((s) => s.machineName).join('\0'), syncLocalTabOrderFromProps, {
      immediate: true,
    })

    const hasSplitGroup = computed(() => props.splitSessionIds.length >= 2)
    /** 当前激活 Tab 属于分屏组时才展示分屏；切到组外 Tab 则临时全屏，分屏配置保留 */
    const splitViewVisible = computed(
      () => hasSplitGroup.value && props.splitSessionIds.includes(activeTab.value),
    )

    watch(() => props.activeMachine, (val) => {
      activeTab.value = val || ''
      if (!val) return
      nextTick(() => {
        terminalRefs.value[val]?.wakeTerminal?.()
        setTimeout(() => terminalRefs.value[val]?.wakeTerminal?.(), 120)
      })
    })

    watch(activeTab, (val) => {
      emit('update:activeMachine', val)
      nextTickFit(val)
      nextTick(() => {
        terminalRefs.value[val]?.wakeTerminal?.()
        setTimeout(() => terminalRefs.value[val]?.wakeTerminal?.(), 120)
      })
    })

    watch(splitViewVisible, (visible) => {
      if (!visible) {
        zoomedSessionId.value = ''
      }
      if (visible) {
        setTimeout(() => {
          props.splitSessionIds.forEach((id) => terminalRefs.value[id]?.fitAndResize?.())
        }, 80)
      } else if (activeTab.value) {
        setTimeout(() => terminalRefs.value[activeTab.value]?.fitAndResize?.(), 80)
      }
    })

    const nextTickFit = (name) => {
      // 切 tab 后立刻 fit，勿再拖 40ms，否则重建终端会先以默认行列露一帧
      nextTick(() => {
        requestAnimationFrame(() => {
          if (splitViewVisible.value) {
            props.splitSessionIds.forEach((id) => terminalRefs.value[id]?.fitAndResize?.())
          } else {
            terminalRefs.value[name]?.fitAndResize?.()
          }
        })
      })
    }

    watch(
      () => props.splitSessionIds,
      (ids) => {
        if (ids.length === 1) {
          emit('update:split-session-ids', [])
        }
        if (zoomedSessionId.value && !ids.includes(zoomedSessionId.value)) {
          zoomedSessionId.value = ''
        }
      },
    )

    const hidePaneMenu = () => {
      paneMenu.visible = false
      paneMenu.sessionId = ''
    }

    const hideTabMenu = () => {
      tabMenu.visible = false
      tabMenu.sessionId = ''
    }

    const hideAllCtxMenus = () => {
      hidePaneMenu()
      hideTabMenu()
    }

    const onDocPointerDown = (e) => {
      if (e.target?.closest?.('.pane-ctx-menu')) return
      hideAllCtxMenus()
    }
    onMounted(() => document.addEventListener('mousedown', onDocPointerDown, true))
    onUnmounted(() => {
      document.removeEventListener('mousedown', onDocPointerDown, true)
      document.body.classList.remove('tab-drag-active', 'tab-reorder-active')
    })

    const setTerminalRef = (name, el) => {
      if (el) terminalRefs.value[name] = el
      else delete terminalRefs.value[name]
    }

    const tabDisplayLabel = (session) => {
      const base =
        session?.tabLabel
        || (session?.kind === 'local' || isLocalSession(session?.machineName)
          ? localTabLabel(session.machineName)
          : (session?.configName || session?.machineName))
      const host = String(sessionMeta(session).host || '').trim()
      const cwdBase = cwdBasename(session?.lastCwd)
      // 动态标题：有 cwd 时追加目录名；否则在标签不含主机时附上 hostname
      if (cwdBase) {
        return `${base}:${cwdBase}`
      }
      if (host && base && !String(base).includes(host)) {
        return `${base}@${host}`
      }
      return base
    }

    const tabLabel = (session) => {
      const base = tabDisplayLabel(session)
      if (session?.connecting) return `${base} (连接中)`
      if (!session?.connected) return `${base} (未连接)`
      return base
    }

    const paneTitle = (session) => {
      const cwd = String(session?.lastCwd || '').trim()
      const label = tabLabel(session)
      return cwd ? `${label}\n${cwd}` : label
    }

    const tabStatusClass = (session) => {
      if (session?.connecting) return 'is-connecting'
      if (session?.connected) return 'is-connected'
      return 'is-disconnected'
    }

    const togglePaneZoom = (sessionId) => {
      if (!sessionId || !splitViewVisible.value) return
      if (!props.splitSessionIds.includes(sessionId)) return
      zoomedSessionId.value = zoomedSessionId.value === sessionId ? '' : sessionId
      activeTab.value = sessionId
      setTimeout(() => {
        if (zoomedSessionId.value) {
          terminalRefs.value[zoomedSessionId.value]?.fitAndResize?.()
        } else {
          props.splitSessionIds.forEach((id) => terminalRefs.value[id]?.fitAndResize?.())
        }
      }, 80)
    }

    const onPaneMenuZoom = () => {
      const id = paneMenu.sessionId
      hidePaneMenu()
      if (id) togglePaneZoom(id)
    }

    const exitSplit = () => {
      zoomedSessionId.value = ''
      emit('update:split-session-ids', [])
      hidePaneMenu()
      setTimeout(() => terminalRefs.value[activeTab.value]?.fitAndResize?.(), 80)
    }

    const removeFromSplit = (sessionId) => {
      hidePaneMenu()
      const id = sessionId || paneMenu.sessionId
      if (!id) return
      if (zoomedSessionId.value === id) zoomedSessionId.value = ''
      const next = props.splitSessionIds.filter((x) => x !== id)
      if (next.length < 2) {
        zoomedSessionId.value = ''
        emit('update:split-session-ids', [])
        if (id) activeTab.value = id
      } else {
        emit('update:split-session-ids', next)
        if (activeTab.value === id) activeTab.value = next[0]
      }
      setTimeout(() => {
        const ids = next.length >= 2 ? next : [activeTab.value]
        ids.forEach((sid) => terminalRefs.value[sid]?.fitAndResize?.())
      }, 80)
    }

    /** 点击 Tab / 聚焦分屏窗格：同步左侧监控与底部栏 */
    const selectTab = (name) => {
      activeTab.value = name
    }

    const onTabClick = (name) => {
      if (tabDragMoved.value) return
      selectTab(name)
    }

    const onFocusSession = (name) => {
      if (!name || activeTab.value === name) return
      activeTab.value = name
    }

    const onTabRemove = (name) => {
      emit('close-session', name)
    }

    const tabMenuHasRight = computed(() => {
      if (!tabMenu.sessionId) return false
      const ids = orderedSessions.value.map((s) => s.machineName)
      const idx = ids.indexOf(tabMenu.sessionId)
      return idx >= 0 && idx < ids.length - 1
    })

    const onTabContextMenu = (e, sessionId) => {
      hidePaneMenu()
      tabMenu.sessionId = sessionId
      tabMenu.x = e.clientX
      tabMenu.y = e.clientY
      tabMenu.visible = true
      activeTab.value = sessionId
    }

    const onTabMenuDuplicate = () => {
      const id = tabMenu.sessionId
      hideTabMenu()
      if (id) emit('duplicate-session', id)
    }

    const onTabMenuClose = () => {
      const id = tabMenu.sessionId
      hideTabMenu()
      if (id) emit('close-session', id)
    }

    const onTabMenuCloseRight = () => {
      if (!tabMenuHasRight.value) return
      const id = tabMenu.sessionId
      const ids = orderedSessions.value.map((s) => s.machineName)
      hideTabMenu()
      if (!id) return
      const idx = ids.indexOf(id)
      if (idx < 0 || idx >= ids.length - 1) return
      emit('close-sessions', ids.slice(idx + 1))
    }

    const onTabMenuCloseAll = () => {
      const ids = orderedSessions.value.map((s) => s.machineName)
      hideTabMenu()
      if (ids.length) emit('close-sessions', ids)
    }

    const onAddCommand = (cmd) => {
      if (cmd === 'remote') emit('open-picker')
      else emit('add-local')
    }

    const isTerminalActive = (name) => {
      if (splitViewVisible.value) {
        return props.splitSessionIds.includes(name)
      }
      return activeTab.value === name
    }

    const connectedCount = computed(() =>
      (props.sessions || []).filter((s) => s.connected).length,
    )

    const toggleBroadcast = () => {
      const next = !props.broadcastEnabled
      emit('update:broadcast-enabled', next)
      if (next && !props.broadcastTargets.length) {
        const ids = (props.sessions || []).filter((s) => s.connected).map((s) => s.machineName)
        emit('update:broadcast-targets', ids)
      }
    }

    const composeEnabled = ref(false)
    const toggleCompose = () => {
      composeEnabled.value = !composeEnabled.value
    }

    const splitGridStyle = computed(() => {
      if (!splitViewVisible.value) return {}
      if (zoomedSessionId.value) {
        return { gridTemplateColumns: '1fr', gridTemplateRows: '1fr' }
      }
      const n = props.splitSessionIds.length
      if (n === 2) return { gridTemplateColumns: '1fr 1fr', gridTemplateRows: '1fr' }
      return { gridTemplateColumns: '1fr 1fr', gridTemplateRows: '1fr 1fr' }
    })

    const dropZones = computed(() => {
      const full = props.splitSessionIds.length >= MAX_SPLIT
      return [
        { id: 'left', label: '左侧', disabled: full },
        { id: 'right', label: '右侧', disabled: full },
        { id: 'top', label: '上方', disabled: full },
        { id: 'bottom', label: '下方', disabled: full },
      ]
    })

    const onTabMouseDown = (e, sessionId) => {
      if (e.button !== 0 || e.target.closest('.session-tab-close')) return
      hideAllCtxMenus()

      const startX = e.clientX
      const startY = e.clientY
      let mode = ''
      let lastX = startX
      let lastY = startY

      const clearDragUi = () => {
        tabReorderFrom.value = ''
        draggingTab.value = ''
        dropReorderTarget.value = ''
        dropReorderAfter.value = false
        dropTargetZone.value = ''
        document.body.classList.remove('tab-drag-active', 'tab-reorder-active')
      }

      const applyLocalReorder = (from, to, after) => {
        localTabOrder.value = moveTabInOrder(localTabOrder.value, from, to, after)
      }

      const onMove = (ev) => {
        lastX = ev.clientX
        lastY = ev.clientY
        const dx = lastX - startX
        const dy = lastY - startY

        if (!mode) {
          if (Math.abs(dx) >= DRAG_REORDER_PX && Math.abs(dx) >= Math.abs(dy)) mode = 'reorder'
          else if (dy >= DRAG_SPLIT_PX && dy > Math.abs(dx)) mode = 'split'
          if (mode) {
            tabDragMoved.value = true
            ev.preventDefault()
            document.body.classList.add('tab-drag-active')
          }
        }

        if (mode === 'reorder') {
          document.body.classList.add('tab-reorder-active')
          tabReorderFrom.value = sessionId
          draggingTab.value = ''
          dropTargetZone.value = ''
          const target = findTabTargetAt(lastX, lastY, sessionId)
          dropReorderTarget.value = target?.id || ''
          dropReorderAfter.value = !!target?.after
        } else if (mode === 'split') {
          document.body.classList.remove('tab-reorder-active')
          tabReorderFrom.value = ''
          dropReorderTarget.value = ''
          draggingTab.value = sessionId
          const zoneId = findSplitZoneAt(lastX, lastY, dropZones.value)
          dropTargetZone.value = zoneId === 'stack' ? 'right' : zoneId
        }
      }

      const onUp = (ev) => {
        const x = ev?.clientX ?? lastX
        const y = ev?.clientY ?? lastY

        if (mode === 'reorder') {
          const target = findTabTargetAt(x, y, sessionId)
          const to = target?.id || dropReorderTarget.value
          const after = target ? target.after : dropReorderAfter.value
          if (to && to !== sessionId) {
            applyLocalReorder(sessionId, to, after)
            emit('reorder-tabs', { from: sessionId, to, after })
          }
        } else if (mode === 'split') {
          let zoneId = findSplitZoneAt(x, y, dropZones.value)
          if (zoneId === 'stack') zoneId = 'right'
          if (!zoneId) zoneId = dropTargetZone.value
          const zone = dropZones.value.find((z) => z.id === zoneId)
          if (zone && !zone.disabled) {
            applySplit(buildSplitIds(sessionId, zone.id))
          }
        }

        clearDragUi()
        window.removeEventListener('mousemove', onMove, true)
        window.removeEventListener('mouseup', onUp, true)
        window.setTimeout(() => {
          tabDragMoved.value = false
        }, 0)
      }

      window.addEventListener('mousemove', onMove, true)
      window.addEventListener('mouseup', onUp, true)
    }

    const onPaneDragStart = (e, sessionId) => {
      draggingSplitPane.value = sessionId
      draggingTab.value = ''
      e.dataTransfer.effectAllowed = 'move'
      e.dataTransfer.setData('text/plain', sessionId)
      e.dataTransfer.setData('application/x-flashdock-unsplit', sessionId)
    }

    const onPaneDragEnd = () => {
      draggingSplitPane.value = ''
    }

    const onTabsBarDragOver = (e) => {
      if (draggingSplitPane.value || e.dataTransfer?.types?.includes?.('application/x-flashdock-unsplit')) {
        e.dataTransfer.dropEffect = 'move'
      }
    }

    const onTabsBarDrop = (e) => {
      const id = draggingSplitPane.value
        || e.dataTransfer.getData('application/x-flashdock-unsplit')
        || e.dataTransfer.getData('text/plain')
      if (id && props.splitSessionIds.includes(id)) {
        removeFromSplit(id)
      }
      draggingSplitPane.value = ''
    }

    const onStackDragOver = () => {
      if (draggingTab.value) dropTargetZone.value = 'stack'
    }

    const onStackDragLeave = (e) => {
      if (!e.currentTarget.contains(e.relatedTarget)) {
        dropTargetZone.value = ''
      }
    }

    const buildSplitIds = (draggedId, zoneId) => {
      const dragged = draggedId || draggingTab.value
      if (!dragged) return null

      let ids = [...props.splitSessionIds]
      const inSplit = ids.includes(dragged)

      if (ids.length >= MAX_SPLIT && !inSplit) return null

      if (ids.length === 0) {
        const anchor = activeTab.value && activeTab.value !== dragged ? activeTab.value : ''
        if (zoneId === 'left' || zoneId === 'top') {
          ids = anchor ? [dragged, anchor] : [dragged]
        } else {
          ids = anchor ? [anchor, dragged] : [dragged]
        }
      } else if (!inSplit) {
        if (zoneId === 'left' || zoneId === 'top') ids.unshift(dragged)
        else ids.push(dragged)
      }

      ids = [...new Set(ids)].slice(0, MAX_SPLIT)
      if (ids.length < 2) {
        const other = props.sessions.find((s) => s.machineName !== dragged)?.machineName
        if (other && !ids.includes(other)) ids.push(other)
      }
      return ids.length >= 2 ? ids : null
    }

    const applySplit = (ids) => {
      if (!ids || ids.length < 2) return
      emit('update:split-session-ids', ids)
      if (!ids.includes(activeTab.value)) activeTab.value = ids[0]
      draggingTab.value = ''
      dropTargetZone.value = ''
      setTimeout(() => {
        ids.forEach((id) => terminalRefs.value[id]?.fitAndResize?.())
      }, 80)
    }

    const onZoneDragOver = (zone) => {
      if (!zone.disabled) dropTargetZone.value = zone.id
    }

    const onZoneDrop = (zone) => {
      if (zone.disabled) return
      const ids = buildSplitIds(draggingTab.value, zone.id)
      applySplit(ids)
    }

    const onStackDrop = () => {
      if (draggingSplitPane.value) return
      const ids = buildSplitIds(draggingTab.value, 'right')
      applySplit(ids)
    }

    const onPaneContextMenu = (e, sessionId) => {
      if (!splitViewVisible.value || !props.splitSessionIds.includes(sessionId)) return
      hideTabMenu()
      paneMenu.sessionId = sessionId
      paneMenu.x = e.clientX
      paneMenu.y = e.clientY
      paneMenu.visible = true
      activeTab.value = sessionId
    }

    const onPaneMenuRemove = () => removeFromSplit(paneMenu.sessionId)
    const onPaneMenuExit = () => exitSplit()

    const clearActive = () => {
      terminalRefs.value[activeTab.value]?.clear?.()
      emit('clear', activeTab.value)
    }

    const getActiveTerminal = () => terminalRefs.value[activeTab.value]

    const emptyResult = () => ({ found: false, resultIndex: -1, resultCount: 0 })
    const findNext = () => getActiveTerminal()?.findNext?.() ?? emptyResult()
    const findPrevious = () => getActiveTerminal()?.findPrevious?.() ?? emptyResult()
    const clearSearch = () => getActiveTerminal()?.clearSearch?.()
    const pasteClipboard = () => getActiveTerminal()?.pasteClipboard?.()
    const fitActive = () => {
      if (splitViewVisible.value) {
        if (zoomedSessionId.value) {
          terminalRefs.value[zoomedSessionId.value]?.fitAndResize?.()
          return
        }
        props.splitSessionIds.forEach((id) => terminalRefs.value[id]?.fitAndResize?.())
        return
      }
      getActiveTerminal()?.fitAndResize?.()
    }
    const getSelection = () => getActiveTerminal()?.getSelection?.() || ''

    const selectTabByIndex = (index) => {
      const list = orderedSessions.value
      if (!list.length) return false
      const i = Math.max(0, Math.min(list.length - 1, index))
      const id = list[i]?.machineName
      if (!id) return false
      activeTab.value = id
      return true
    }

    const selectNextTab = (delta = 1) => {
      const list = orderedSessions.value
      if (!list.length) return false
      const cur = list.findIndex((s) => s.machineName === activeTab.value)
      const next = cur < 0 ? 0 : (cur + delta + list.length) % list.length
      activeTab.value = list[next].machineName
      return true
    }

    const closeActiveTab = () => {
      const id = activeTab.value
      if (!id) return false
      emit('close-session', id)
      return true
    }

    const focusSplitNeighbor = (dir) => {
      const ids = props.splitSessionIds || []
      if (ids.length < 2) return false
      const cur = ids.indexOf(activeTab.value)
      if (cur < 0) {
        activeTab.value = ids[0]
        return true
      }
      let next = cur
      if (dir === 'left' || dir === 'up') next = (cur - 1 + ids.length) % ids.length
      else next = (cur + 1) % ids.length
      activeTab.value = ids[next]
      return true
    }

    expose({
      clearActive,
      findNext,
      findPrevious,
      clearSearch,
      fitActive,
      getSelection,
      pasteClipboard,
      togglePaneZoom,
      selectTabByIndex,
      selectNextTab,
      closeActiveTab,
      focusSplitNeighbor,
    })

    return {
      activeTab,
      dropReorderTarget,
      dropReorderAfter,
      tabReorderFrom,
      orderedSessions,
      sessionMeta,
      draggingTab,
      draggingSplitPane,
      dropZones,
      paneMenu,
      hidePaneMenu,
      tabMenu,
      tabMenuHasRight,
      hideTabMenu,
      hasSplitGroup,
      onChromeTitleDblActivate,
      onChromeTitlePointerDown,
      splitViewVisible,
      zoomedSessionId,
      setTerminalRef,
      onTabRemove,
      onTabContextMenu,
      onTabMenuDuplicate,
      onTabMenuClose,
      onTabMenuCloseRight,
      onTabMenuCloseAll,
      clearActive,
      tabLabel,
      tabDisplayLabel,
      paneTitle,
      tabStatusClass,
      onAddCommand,
      isLocalSession,
      isTerminalActive,
      splitGridStyle,
      selectTab,
      onTabClick,
      onFocusSession,
      onTabMouseDown,
      onPaneDragStart,
      onPaneDragEnd,
      onTabsBarDragOver,
      onTabsBarDrop,
      onStackDragOver,
      onStackDragLeave,
      onStackDrop,
      onZoneDragOver,
      onZoneDrop,
      onPaneContextMenu,
      onPaneMenuZoom,
      onPaneMenuRemove,
      onPaneMenuExit,
      removeFromSplit,
      exitSplit,
      togglePaneZoom,
      toggleBroadcast,
      composeEnabled,
      toggleCompose,
      connectedCount,
    }
  },
}
</script>

<style scoped>
.shell-terminal-tabs {
  position: relative;
  flex: 1;
  min-height: 0;
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.tabs-bar {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
  background: var(--app-panel-bg);
  border-bottom: 1px solid var(--app-border);
  padding: 0 6px 0 4px;
  min-height: 36px;
  height: 36px;
  box-sizing: border-box;
}

.tabs-bar.is-drop-unsplit {
  outline: 1px dashed var(--app-accent-color);
  outline-offset: -2px;
  background: color-mix(in srgb, var(--app-accent-color) 8%, var(--app-panel-bg));
}

.unsplit-hint {
  font-size: 11px;
  color: var(--app-accent-color);
  white-space: nowrap;
}

.custom-session-tabs {
  display: flex;
  align-items: center;
  gap: 3px;
  flex: 0 1 auto;
  min-width: 0;
  max-width: 100%;
  overflow-x: auto;
  overflow-y: hidden;
}

.tabs-bar-left {
  display: flex;
  align-items: center;
  flex: 0 1 auto;
  min-width: 0;
  max-width: calc(100% - 96px);
}

.tabs-bar-right {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  gap: 2px;
}

.tabs-chrome-sep {
  width: 1px;
  height: 14px;
  margin: 0 4px 0 6px;
  background: color-mix(in srgb, var(--app-text-muted, #909399) 35%, transparent);
  flex-shrink: 0;
}

.tabs-bar-right :deep(.el-tooltip__trigger),
.tabs-bar-right :deep(.el-badge) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.tabs-bar-right :deep(.el-button) {
  box-sizing: border-box;
  width: 26px;
  height: 26px;
  min-width: 26px;
  min-height: 26px;
  padding: 0;
  margin: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.session-tab.drop-before {
  box-shadow: inset 2px 0 0 var(--app-accent-color);
}

.session-tab.drop-after {
  box-shadow: inset -2px 0 0 var(--app-accent-color);
}

.session-tab.is-reorder-dragging {
  opacity: 0.72;
}

:global(body.tab-drag-active) {
  user-select: none;
  cursor: grabbing;
}

:global(body.tab-reorder-active) .shell-terminal-tabs .session-tab.is-reorder-dragging {
  pointer-events: none;
}

.session-tab {
  display: inline-flex;
  align-items: center;
  box-sizing: border-box;
  gap: 0;
  height: 28px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 500;
  line-height: 1;
  color: var(--app-text-muted);
  background: transparent;
  border: none;
  border-radius: 7px;
  cursor: grab;
  user-select: none;
  flex-shrink: 0;
  max-width: 200px;
  position: relative;
  transition: color 0.15s ease;
}

.session-tab:hover::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 7px;
  background: color-mix(in srgb, var(--app-text) 7%, transparent);
  pointer-events: none;
  z-index: 0;
}

.session-tab:hover {
  color: var(--app-text-secondary);
}

.session-tab:active {
  cursor: grabbing;
}

.session-tab.active {
  color: var(--app-text);
  font-weight: 600;
}

.session-tab.active::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 7px;
  background: color-mix(in srgb, var(--app-text) 9%, transparent);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--app-text) 8%, transparent);
  pointer-events: none;
  z-index: 0;
}

/* 选中指示：底边短线，和连接状态点解耦 */
.session-tab.active::after {
  content: '';
  position: absolute;
  left: 10px;
  right: 10px;
  bottom: 2px;
  height: 2px;
  border-radius: 1px;
  background: var(--app-accent-color);
  opacity: 0.9;
  pointer-events: none;
  z-index: 1;
}

.session-tab-main,
.session-tab-close {
  position: relative;
  z-index: 1;
}

.session-tab.in-split:not(.active) {
  box-shadow: inset 0 -2px 0 color-mix(in srgb, var(--app-accent-color) 70%, transparent);
}

.session-tab-main {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  min-width: 0;
  flex: 1 1 auto;
}

.session-tab-status {
  flex-shrink: 0;
  width: 6px;
  height: 6px;
  margin-right: 7px;
  border-radius: 50%;
  background: var(--app-text-muted);
}

.session-tab-status.is-connecting {
  background: var(--app-text-muted);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--app-text-muted) 22%, transparent);
  animation: shell-tab-status-pulse 1.2s ease-in-out infinite;
}

.session-tab-status.is-disconnected {
  background: var(--app-danger-color, var(--el-color-danger, #f56c6c));
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--app-danger-color, #f56c6c) 18%, transparent);
}

.session-tab-status.is-connected {
  background: var(--app-success-color, #67c23a);
  box-shadow:
    0 0 0 2px color-mix(in srgb, var(--app-success-color, #67c23a) 22%, transparent),
    0 0 6px color-mix(in srgb, var(--app-success-color, #67c23a) 35%, transparent);
}

@keyframes shell-tab-status-pulse {
  0%, 100% { opacity: 0.45; }
  50% { opacity: 1; }
}

.session-tab-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
  line-height: 1.2;
}

.session-tab-close {
  border: none;
  background: transparent;
  color: var(--app-text-muted);
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  padding: 0;
  margin-left: 0;
  width: 0;
  opacity: 0;
  overflow: hidden;
  flex-shrink: 0;
  align-self: center;
  pointer-events: none;
  transition: opacity 0.15s ease, width 0.15s ease, margin 0.15s ease, color 0.15s ease;
}

.session-tab:hover .session-tab-close,
.session-tab.active .session-tab-close {
  width: 14px;
  margin-left: 5px;
  opacity: 1;
  pointer-events: auto;
}

.session-tab-close:hover {
  color: var(--app-danger-color, #f56c6c);
}

.add-session-wrap {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  margin-left: 2px;
}

.add-session-btn {
  color: var(--app-text-secondary);
  padding: 2px 4px !important;
  width: 26px !important;
  height: 26px !important;
  min-width: 26px !important;
}

.add-session-more {
  color: var(--app-text-secondary);
  padding: 2px 2px !important;
  width: 18px !important;
  height: 26px !important;
  min-width: 18px !important;
  margin-left: -2px;
}

.add-session-btn:hover,
.add-session-more:hover {
  color: var(--app-accent-color);
}

.home-btn {
  flex-shrink: 0;
  margin-left: 0;
  color: var(--app-text-secondary);
  padding: 2px 6px !important;
  width: 26px !important;
  height: 26px !important;
  min-width: 26px !important;
}

.broadcast-toggle {
  flex-shrink: 0;
  color: var(--app-text-secondary);
}

.broadcast-toggle:hover,
.broadcast-toggle.active {
  color: var(--app-accent-color);
}

.broadcast-toggle.active {
  background: var(--app-accent-bg);
  border-radius: var(--app-radius-sm, 6px);
}

.compose-toggle {
  flex-shrink: 0;
  color: var(--app-text-secondary);
}

.compose-toggle:hover,
.compose-toggle.active {
  color: var(--app-accent-color);
}

.compose-toggle.active {
  background: var(--app-accent-bg);
  border-radius: var(--app-radius-sm, 6px);
}

.transfer-btn {
  flex-shrink: 0;
  color: var(--app-text-secondary);
}

.transfer-btn:hover {
  color: var(--app-accent-color);
}

.transfer-btn :deep(.el-badge__content) {
  transform: translateY(-2px) translateX(4px);
}

.home-btn:hover {
  color: var(--app-accent-color);
}

.tabs-bar-spacer {
  flex: 1;
  min-width: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.empty-slot {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.terminal-body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  position: relative;
  overflow: hidden;
}

/* 广播 / 撰写：浮在终端上方，不改变 xterm 容器尺寸 */
.shell-aux-bars {
  position: absolute;
  top: 8px;
  left: 8px;
  right: 118px;
  z-index: 5;
  display: flex;
  flex-direction: column;
  pointer-events: none;
  box-sizing: border-box;
  border-radius: 10px;
  overflow: hidden;
  background: var(--app-card-bg);
  border: 1px solid var(--app-border);
  box-shadow: 0 8px 24px color-mix(in srgb, #000 28%, transparent);
}

.shell-aux-bars > * {
  pointer-events: auto;
}

.shell-aux-bars :deep(.shell-broadcast-bar),
.shell-aux-bars :deep(.shell-compose-bar) {
  background: transparent;
  backdrop-filter: none;
  border-bottom-color: color-mix(in srgb, var(--app-border) 80%, transparent);
}

.shell-aux-bars :deep(.shell-compose-bar:last-child),
.shell-aux-bars :deep(.shell-broadcast-bar:last-child) {
  border-bottom: none;
}

.terminal-body.has-aux-bars .terminal-inline-actions {
  /* 避开浮层右侧，避免与关闭/发送按钮叠在一起 */
  top: 8px;
  z-index: 7;
}

.terminal-inline-actions {
  position: absolute;
  top: 8px;
  right: 12px;
  z-index: 6;
  display: inline-flex;
  align-items: center;
  gap: 1px;
  padding: 2px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--app-panel-bg, #1e1e1e) 72%, transparent);
  backdrop-filter: blur(8px);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.2);
}

.terminal-inline-actions :deep(.el-tooltip__trigger),
.terminal-inline-actions :deep(.el-badge) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.terminal-inline-actions :deep(.el-button) {
  box-sizing: border-box;
  width: 26px;
  height: 26px;
  min-width: 26px;
  min-height: 26px;
  padding: 0;
  margin: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--app-text-secondary);
}

.terminal-inline-actions :deep(.el-button:hover),
.terminal-inline-actions :deep(.el-button.active) {
  color: var(--app-accent-color);
  background: color-mix(in srgb, var(--app-accent-color) 14%, transparent);
}

.terminal-stack {
  flex: 1;
  min-height: 0;
  width: 100%;
  position: relative;
}

.terminal-stack.is-split {
  display: grid;
  gap: 2px;
  background: var(--app-border);
  position: relative;
}

.terminal-pane {
  position: absolute;
  inset: 0;
  visibility: hidden;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  background: var(--terminal-bg, #1e1e1e);
}

.terminal-pane.is-active {
  visibility: visible;
  z-index: 1;
}

.terminal-stack.is-split .terminal-pane.is-split-pane {
  position: relative;
  inset: auto;
  visibility: visible;
  z-index: 1;
}

.terminal-stack.is-split .terminal-pane.is-split-hidden {
  display: none;
}

.terminal-stack.is-split.is-pane-zoomed .terminal-pane.is-split-pane:not(.is-zoomed) {
  display: none;
}

.terminal-stack.is-split.is-pane-zoomed .terminal-pane.is-zoomed {
  grid-column: 1 / -1;
  grid-row: 1 / -1;
}

.terminal-pane.is-focused.is-split-pane {
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--app-accent-color) 55%, transparent);
}

.split-pane-header {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  height: 26px;
  padding: 0 6px 0 10px;
  background: var(--app-panel-bg);
  border-bottom: 1px solid var(--app-border);
  cursor: grab;
  user-select: none;
}

.split-pane-header:active {
  cursor: grabbing;
}

.split-pane-name {
  flex: 1;
  min-width: 0;
  font-size: 11px;
  font-weight: 600;
  color: var(--app-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.terminal-pane.is-focused .split-pane-name {
  color: var(--app-accent-color);
}

.split-pane-zoom,
.split-pane-unsplit {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--app-text-secondary);
  cursor: pointer;
  font-size: 12px;
  line-height: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
}

.split-pane-zoom:hover {
  background: var(--app-hover-bg, rgba(128, 128, 128, 0.15));
  color: var(--app-text);
}

.split-pane-unsplit:hover {
  color: var(--app-danger-color, #f56c6c);
  background: color-mix(in srgb, var(--app-danger-color, #f56c6c) 12%, transparent);
}

.terminal-pane :deep(.shell-terminal) {
  flex: 1;
  min-height: 0;
  min-width: 0;
  position: relative;
}

.pane-ctx-menu {
  position: fixed;
  z-index: 3000;
  margin: 0;
  padding: 4px 0;
  list-style: none;
  min-width: 140px;
  background: var(--app-card-bg);
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius-md, 8px);
  box-shadow: 0 8px 24px color-mix(in srgb, #000 18%, transparent);
}

.pane-ctx-menu li {
  padding: 8px 14px;
  font-size: 12px;
  color: var(--app-text);
  cursor: pointer;
}

.pane-ctx-menu li:hover {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}

.pane-ctx-menu li.disabled {
  opacity: 0.4;
  cursor: default;
  pointer-events: none;
}

.split-drop-overlay {
  position: absolute;
  inset: 0;
  z-index: 20;
  background: color-mix(in srgb, var(--app-accent-color) 12%, transparent);
  backdrop-filter: blur(1px);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  pointer-events: none;
}

.split-drop-hint {
  margin: 0;
  font-size: 13px;
  color: var(--app-text);
  font-weight: 500;
}

.split-drop-zones {
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-template-rows: 1fr 1fr;
  gap: 8px;
  width: min(420px, 80%);
  height: min(220px, 60%);
  pointer-events: auto;
}

.drop-zone {
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px dashed var(--app-accent-color);
  border-radius: 8px;
  background: color-mix(in srgb, var(--app-card-bg) 85%, transparent);
  color: var(--app-accent-color);
  font-size: 13px;
  font-weight: 500;
}

.drop-zone:hover:not(.disabled),
.drop-zone.active:not(.disabled) {
  background: color-mix(in srgb, var(--app-accent-color) 18%, var(--app-card-bg));
  border-style: solid;
}

.drop-zone.disabled {
  opacity: 0.35;
  pointer-events: none;
}
</style>
