<template>
  <div class="shell-terminal-tabs">
    <div class="tabs-bar" :class="{ 'is-drop-unsplit': draggingSplitPane }" @dragover.prevent="onTabsBarDragOver"
      @drop.prevent="onTabsBarDrop">
      <el-button class="home-btn" size="small" text title="返回首页" @click="$emit('back')">
        <el-icon :size="14">
          <ArrowLeft />
        </el-icon>
      </el-button>
      <el-button class="folder-btn" size="small" text title="连接（最近 / 全部机器）" @click="$emit('open-picker')">
        <el-icon :size="16">
          <Folder />
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
            @mousedown="onTabMouseDown($event, session.machineName)">
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
          <el-button class="add-session-btn" size="small" text title="新建本机" @click="$emit('add-local')">
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
                <el-dropdown-item command="local">本机</el-dropdown-item>
                <el-dropdown-item command="remote">远程连接…</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
      <div class="tabs-bar-spacer" aria-hidden="true">
        <span v-if="draggingSplitPane" class="unsplit-hint">拖到此处移出分屏</span>
      </div>
      <div class="tabs-bar-right">
        <el-tooltip v-if="sessions.length && connectedCount >= 1"
          :content="broadcastEnabled ? '关闭命令广播 (Esc)' : '开启命令广播'" placement="bottom">
          <el-button class="broadcast-toggle" size="small" text :class="{ active: broadcastEnabled }"
            @click="toggleBroadcast">
            <el-icon :size="15">
              <Promotion />
            </el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip content="命令面板 (历史/片段)" placement="bottom">
          <el-button v-if="sessions.length" size="small" text title="命令面板"
            @click="$emit('open-command-palette')">
            <el-icon :size="15"><Memo /></el-icon>
          </el-button>
        </el-tooltip>
        <el-button v-if="sessions.length && !isLocalSession(activeTab)" class="transfer-btn" size="small" text
          title="文件传输" @click="$emit('open-transfer')">
          <el-badge :value="transferActiveCount" :hidden="!transferActiveCount" :max="99">
            <el-icon :size="15">
              <Upload />
            </el-icon>
          </el-badge>
        </el-button>
      </div>
    </div>

    <ShellBroadcastBar v-if="sessions.length && broadcastEnabled" :enabled="broadcastEnabled"
      :targets="broadcastTargets" :sessions="sessions" @update:enabled="(v) => $emit('update:broadcast-enabled', v)"
      @update:targets="(v) => $emit('update:broadcast-targets', v)" />

    <div v-if="sessions.length === 0" class="empty-slot">
      <slot name="empty" />
    </div>
    <template v-else>
      <div class="terminal-stack"
        :class="{ 'is-split': splitViewVisible, 'is-drag-over': !!draggingTab && !draggingSplitPane }"
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
        }">
          <div v-if="splitViewVisible && splitSessionIds.includes(session.machineName)" class="split-pane-header"
            draggable="true" @dragstart="onPaneDragStart($event, session.machineName)" @dragend="onPaneDragEnd"
            @mousedown="activeTab = session.machineName"
            @contextmenu.prevent="onPaneContextMenu($event, session.machineName)">
            <span class="split-pane-name" :title="tabLabel(session)">{{ tabLabel(session) }}</span>
            <button type="button" class="split-pane-unsplit" title="移出分屏"
              @click.stop="removeFromSplit(session.machineName)">
              ×
            </button>
          </div>
          <ShellTerminal :ref="(el) => setTerminalRef(session.machineName, el)" :machine-name="session.machineName"
            :connected="!!session.connected" :connecting="!!session.connecting"
            :active="isTerminalActive(session.machineName)" :view-visible="viewVisible" :search-query="searchQuery"
            :broadcast-enabled="broadcastEnabled"
            :in-split="splitViewVisible && splitSessionIds.includes(session.machineName)"
            @open-search="(text) => $emit('open-search', text)" @reconnect="(name) => $emit('reconnect', name)"
            @clear-cache="(name) => $emit('clear', name)" @search-result="(payload) => $emit('search-result', payload)"
            @cwd-sync="(payload) => $emit('cwd-sync', payload)" @remove-from-split="removeFromSplit"
            @exit-split="exitSplit" @focus-session="onFocusSession" />
        </div>
      </div>

      <ul v-if="paneMenu.visible" class="pane-ctx-menu" :style="{ left: paneMenu.x + 'px', top: paneMenu.y + 'px' }"
        @click.stop @mouseleave="hidePaneMenu">
        <li @click="onPaneMenuRemove">移出分屏</li>
        <li @click="onPaneMenuExit">取消全部分屏</li>
      </ul>

      <slot name="footer" :active-machine="activeTab" />
    </template>
  </div>
</template>

<script>
import { ref, reactive, watch, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { ArrowLeft, ArrowDown, Folder, Upload, Plus, Promotion, Memo } from '@element-plus/icons-vue'
import ShellTerminal from './ShellTerminal.vue'
import ShellBroadcastBar from './ShellBroadcastBar.vue'

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
    ArrowLeft,
    ArrowDown,
    Folder,
    Upload,
    Plus,
    Promotion,
    Memo,
  },
  props: {
    sessions: { type: Array, default: () => [] },
    activeMachine: { type: String, default: '' },
    searchQuery: { type: String, default: '' },
    viewVisible: { type: Boolean, default: true },
    transferActiveCount: { type: Number, default: 0 },
    broadcastEnabled: { type: Boolean, default: false },
    broadcastTargets: { type: Array, default: () => [] },
    splitSessionIds: { type: Array, default: () => [] },
  },
  emits: [
    'update:activeMachine', 'close-session', 'clear', 'open-picker', 'add-local',
    'back', 'open-search', 'reconnect', 'search-result', 'open-transfer', 'open-command-palette', 'cwd-sync',
    'update:broadcast-enabled', 'update:broadcast-targets', 'update:split-session-ids',
    'reorder-tabs',
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
      if (visible) {
        setTimeout(() => {
          props.splitSessionIds.forEach((id) => terminalRefs.value[id]?.fitAndResize?.())
        }, 80)
      } else if (activeTab.value) {
        setTimeout(() => terminalRefs.value[activeTab.value]?.fitAndResize?.(), 80)
      }
    })

    const nextTickFit = (name) => {
      setTimeout(() => {
        if (splitViewVisible.value) {
          props.splitSessionIds.forEach((id) => terminalRefs.value[id]?.fitAndResize?.())
        } else {
          terminalRefs.value[name]?.fitAndResize?.()
        }
      }, 40)
    }

    watch(
      () => props.splitSessionIds,
      (ids) => {
        if (ids.length === 1) {
          emit('update:split-session-ids', [])
        }
      },
    )

    const hidePaneMenu = () => {
      paneMenu.visible = false
      paneMenu.sessionId = ''
    }

    const onDocClick = () => hidePaneMenu()
    onMounted(() => document.addEventListener('click', onDocClick))
    onUnmounted(() => {
      document.removeEventListener('click', onDocClick)
      document.body.classList.remove('tab-drag-active', 'tab-reorder-active')
    })

    const setTerminalRef = (name, el) => {
      if (el) terminalRefs.value[name] = el
      else delete terminalRefs.value[name]
    }

    const tabDisplayLabel = (session) => {
      return (
        session?.tabLabel
        || (session?.kind === 'local' || isLocalSession(session?.machineName)
          ? localTabLabel(session.machineName)
          : (session?.configName || session?.machineName))
      )
    }

    const tabLabel = (session) => {
      const base = tabDisplayLabel(session)
      if (session?.connecting) return `${base} (连接中)`
      if (!session?.connected) return `${base} (未连接)`
      return base
    }

    const tabStatusClass = (session) => {
      if (session?.connecting) return 'is-connecting'
      if (session?.connected) return 'is-connected'
      return 'is-disconnected'
    }

    const exitSplit = () => {
      emit('update:split-session-ids', [])
      hidePaneMenu()
      setTimeout(() => terminalRefs.value[activeTab.value]?.fitAndResize?.(), 80)
    }

    const removeFromSplit = (sessionId) => {
      hidePaneMenu()
      const id = sessionId || paneMenu.sessionId
      if (!id) return
      const next = props.splitSessionIds.filter((x) => x !== id)
      if (next.length < 2) {
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

    const splitGridStyle = computed(() => {
      if (!splitViewVisible.value) return {}
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
    const fitActive = () => {
      if (splitViewVisible.value) {
        props.splitSessionIds.forEach((id) => terminalRefs.value[id]?.fitAndResize?.())
        return
      }
      getActiveTerminal()?.fitAndResize?.()
    }
    const getSelection = () => getActiveTerminal()?.getSelection?.() || ''

    expose({ clearActive, findNext, findPrevious, clearSearch, fitActive, getSelection })

    return {
      activeTab,
      dropReorderTarget,
      dropReorderAfter,
      tabReorderFrom,
      orderedSessions,
      draggingTab,
      draggingSplitPane,
      dropZones,
      paneMenu,
      hidePaneMenu,
      hasSplitGroup,
      splitViewVisible,
      setTerminalRef,
      onTabRemove,
      clearActive,
      tabLabel,
      tabDisplayLabel,
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
      onPaneMenuRemove,
      onPaneMenuExit,
      removeFromSplit,
      exitSplit,
      toggleBroadcast,
      connectedCount,
    }
  },
}
</script>

<style scoped>
.shell-terminal-tabs {
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
  gap: 4px;
  flex-shrink: 0;
  background: var(--app-panel-bg);
  border-bottom: 1px solid var(--app-border);
  padding: 0 4px 0 4px;
  min-height: 40px;
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
  align-items: flex-end;
  gap: 2px;
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
  align-items: stretch;
  box-sizing: border-box;
  gap: 0;
  height: 34px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 600;
  line-height: 1;
  color: var(--app-text-secondary);
  background: color-mix(in srgb, var(--app-panel-bg) 88%, var(--app-card-bg));
  border: 1px solid var(--app-border);
  border-bottom: none;
  border-radius: 6px 6px 0 0;
  cursor: grab;
  user-select: none;
  flex-shrink: 0;
  max-width: 200px;
  transition: background 0.15s ease, border-color 0.15s ease, color 0.15s ease;
}

.session-tab:hover {
  background: var(--app-card-bg);
  border-color: color-mix(in srgb, var(--app-border) 55%, var(--app-text-muted));
}

.session-tab:active {
  cursor: grabbing;
}

.session-tab.active {
  color: var(--app-accent-color);
  background: var(--app-card-bg);
  border-color: color-mix(in srgb, var(--app-accent-color) 35%, var(--app-border));
}

.session-tab.in-split {
  box-shadow: inset 0 -2px 0 var(--app-accent-color);
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
  margin-right: 6px;
  border-radius: 50%;
  background: var(--app-text-muted);
}

.session-tab-status.is-connecting {
  background: var(--app-text-muted);
}

.session-tab-status.is-disconnected {
  background: var(--app-danger-color, #f56c6c);
}

.session-tab-status.is-connected {
  background: #6fbf73;
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

.folder-btn {
  flex-shrink: 0;
  margin: 0 2px;
}

.add-session-wrap {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  margin-left: 2px;
}

.add-session-btn {
  color: var(--app-text-secondary);
  padding: 4px 6px;
}

.add-session-more {
  color: var(--app-text-secondary);
  padding: 4px 4px;
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
  padding: 4px 10px;
}

.broadcast-toggle {
  flex-shrink: 0;
  color: var(--app-text-secondary);
  padding: 4px 8px;
}

.broadcast-toggle:hover,
.broadcast-toggle.active {
  color: var(--app-accent-color);
}

.broadcast-toggle.active {
  background: var(--app-accent-bg);
  border-radius: var(--app-radius-sm, 6px);
}

.transfer-btn {
  flex-shrink: 0;
  margin-left: 0;
  color: var(--app-text-secondary);
  padding: 4px 10px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
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

.split-pane-unsplit {
  flex-shrink: 0;
  border: none;
  background: transparent;
  color: var(--app-text-muted);
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  padding: 0 4px;
  border-radius: 4px;
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
