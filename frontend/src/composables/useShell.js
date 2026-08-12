import { ref, computed, watch, onUnmounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { sortMachinesByName, isMachineConnected } from '../utils/machineGroups'
import {
  pushShellOutput,
  clearShellOutput,
  removeShellOutput,
  discardShellOutputBuffer,
  finalizeShellOutputMigration,
} from '../utils/shellOutputBuffer'
import { remoteConfigName, buildKnownMachineNames } from '../utils/sessionId'
import { parseHostKeyError } from '../utils/hostKey'

const SHELL_LAYOUT_KEY = 'flashdock.shell.layout.v1'

const readShellLayout = () => {
  try {
    return JSON.parse(localStorage.getItem(SHELL_LAYOUT_KEY) || '{}') || {}
  } catch {
    return {}
  }
}

const writeShellLayout = (payload) => {
  try {
    localStorage.setItem(SHELL_LAYOUT_KEY, JSON.stringify(payload || {}))
  } catch {
    // ignore
  }
}

const isLocalSession = (name) => {
  const n = String(name || '')
  return n === 'local' || n.startsWith('local-')
}

const isPendingSession = (name) => String(name || '').startsWith('__pending__')

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

const createPendingId = () =>
  `__pending__${Date.now().toString(36)}__${Math.random().toString(36).slice(2, 8)}`

const localTabLabel = (sessionID) => {
  if (!sessionID || sessionID === 'local') return '本机'
  const n = String(sessionID).replace(/^local-/, '')
  return n ? `本机-${n}` : '本机'
}

export function useShell() {
  const sessions = ref([])
  /** 工作区打开的 tab（含软断开、连接中占位） */
  const openTabs = ref([])
  /** 用户拖拽后的 tab 顺序（machineName 列表） */
  const tabOrder = ref([])
  const activeMachine = ref('')
  const shellMachines = ref([])
  const connectingName = computed(() => {
    const tab = openTabs.value.find((t) => t.connecting)
    return tab?.tabLabel || tab?.configName || ''
  })
  const testingName = ref('')
  /** 命令广播目标会话 ID 列表；空表示关闭广播 */
  const savedLayout = readShellLayout()
  const broadcastTargets = ref(Array.isArray(savedLayout.broadcastTargets) ? savedLayout.broadcastTargets : [])
  const broadcastEnabled = ref(!!savedLayout.broadcastEnabled)
  /** 平行视图：同时显示的会话 ID（最多 4） */
  const splitSessionIds = ref(Array.isArray(savedLayout.splitSessionIds) ? savedLayout.splitSessionIds : [])
  /** 待信任的主机密钥（连接失败时弹出对话框） */
  const pendingHostKey = ref(null)
  /** 正在关闭的会话：避免 shell:status 在 Disconnect 完成前又把 tab 加回来 */
  const closingSessionIds = new Set()

  const updateTabLastCwd = (machineName, cwd) => {
    if (!machineName || !cwd) return
    const tab = openTabs.value.find((t) => t.machineName === machineName)
    if (!tab) return
    const next = String(cwd).trim()
    if (!next) return
    tab.lastCwd = next
  }

  const persistShellLayout = () => {
    writeShellLayout({
      broadcastEnabled: broadcastEnabled.value,
      broadcastTargets: broadcastTargets.value,
      splitSessionIds: splitSessionIds.value,
    })
  }

  watch(broadcastEnabled, persistShellLayout)
  watch(broadcastTargets, persistShellLayout, { deep: true })
  watch(splitSessionIds, persistShellLayout, { deep: true })

  const resolveRemoteConfigName = (sessionID) =>
    remoteConfigName(sessionID, buildKnownMachineNames(shellMachines.value))

  const sortTabs = (tabs) => {
    const order = tabOrder.value
    return [...tabs].sort((a, b) => {
      const ia = order.indexOf(a.machineName)
      const ib = order.indexOf(b.machineName)
      if (ia >= 0 && ib >= 0) return ia - ib
      if (ia >= 0) return -1
      if (ib >= 0) return 1
      return (a.connectedAt || 0) - (b.connectedAt || 0)
    })
  }

  const workspaceSessions = computed(() => sortTabs(openTabs.value))

  const connectedSessions = computed(() => workspaceSessions.value.filter((s) => s.connected))
  const connectedCount = computed(() => connectedSessions.value.length)
  const openSessionCount = computed(() => openTabs.value.length)

  const ensureTabOrder = (machineName) => {
    if (!machineName || tabOrder.value.includes(machineName)) return
    tabOrder.value = [...tabOrder.value, machineName]
  }

  const replaceTabOrder = (from, to) => {
    if (!from || !to || from === to) return
    tabOrder.value = tabOrder.value.map((id) => (id === from ? to : id))
    const seen = new Set()
    tabOrder.value = tabOrder.value.filter((id) => {
      if (seen.has(id)) return false
      seen.add(id)
      return true
    })
    if (!tabOrder.value.includes(to)) tabOrder.value.push(to)
  }

  const pruneOrphanLocalPending = () => {
    openTabs.value = openTabs.value.filter(
      (t) => !(isPendingSession(t.machineName) && t.kind === 'local'),
    )
  }

  const removeTabOrder = (machineName) => {
    tabOrder.value = tabOrder.value.filter((id) => id !== machineName)
  }

  const syncTabOrder = () => {
    const base = tabOrder.value.length
      ? [...tabOrder.value]
      : sortTabs(openTabs.value).map((t) => t.machineName)
    for (const tab of openTabs.value) {
      if (!base.includes(tab.machineName)) base.push(tab.machineName)
    }
    return base
  }

  const reorderTabs = (fromId, toId, insertAfter = false) => {
    if (!fromId || !toId || fromId === toId) return
    const order = syncTabOrder()
    const next = moveTabInOrder(order, fromId, toId, insertAfter)
    if (next.join('\0') === order.join('\0')) return
    tabOrder.value = [...next]
    const tabMap = new Map(openTabs.value.map((t) => [t.machineName, t]))
    openTabs.value = next.map((id) => tabMap.get(id)).filter(Boolean)
  }

  const upsertOpenTab = (sessionID, liveStatus, opts = {}) => {
    const existing = openTabs.value.find((t) => t.machineName === sessionID)
    const kind = liveStatus?.kind || (isLocalSession(sessionID) ? 'local' : 'remote')
    const configName = liveStatus?.configName || (kind === 'local' ? sessionID : resolveRemoteConfigName(sessionID))
    const tabLabel = liveStatus?.tabLabel || (kind === 'local' ? localTabLabel(sessionID) : sessionID)
    if (existing) {
      existing.connected = opts.connected !== undefined ? opts.connected : true
      existing.connecting = !!opts.connecting
      if (existing.connected) existing.everConnected = true
      existing.kind = kind
      existing.configName = configName
      existing.tabLabel = tabLabel
      if (liveStatus) {
        existing.host = liveStatus.host || existing.host
        existing.user = liveStatus.user || existing.user
      }
      return existing
    }
    const connected = opts.connected !== undefined ? opts.connected : true
    const tab = {
      machineName: sessionID,
      configName,
      tabLabel,
      connected,
      connecting: !!opts.connecting,
      everConnected: !!connected,
      connectedAt: Date.now(),
      host: liveStatus?.host || '',
      user: liveStatus?.user || '',
      isRunning: liveStatus?.isRunning || false,
      currentCommand: liveStatus?.currentCommand || '',
      kind,
    }
    openTabs.value.push(tab)
    ensureTabOrder(sessionID)
    return tab
  }

  const addPendingTab = ({ configName, kind, tabLabel, sessionID = '', activate = true }) => {
    const pendingId = createPendingId()
    const label =
      tabLabel ||
      (kind === 'local' ? (sessionID ? localTabLabel(sessionID) : '本机') : configName || '远程')
    upsertOpenTab(
      pendingId,
      {
        machineName: pendingId,
        configName: configName || sessionID || pendingId,
        tabLabel: label,
        kind: kind || 'remote',
      },
      { connected: false, connecting: true },
    )
    if (activate) activeMachine.value = pendingId
    if (kind !== 'local') pushShellOutput(pendingId, 'line', '连接中...')
    return pendingId
  }

  const finalizePendingTab = (pendingId, sessionID, liveStatus) => {
    const connected = !!liveStatus?.connected
    const connecting = !!liveStatus?.connecting
    const idx = openTabs.value.findIndex((t) => t.machineName === pendingId)
    const kind = liveStatus?.kind || (isLocalSession(sessionID) ? 'local' : 'remote')
    finalizeShellOutputMigration(pendingId, sessionID)
    if (idx >= 0) {
      const prev = openTabs.value[idx]
      openTabs.value[idx] = {
        ...prev,
        ...liveStatus,
        machineName: sessionID,
        connected,
        connecting,
        everConnected: prev.everConnected || connected,
        tabLabel: liveStatus?.tabLabel || prev.tabLabel,
        configName: liveStatus?.configName || prev.configName,
        kind,
      }
      replaceTabOrder(pendingId, sessionID)
    } else {
      upsertOpenTab(sessionID, liveStatus, { connected, connecting })
    }
    // 移除 merge 竞态产生的重复 tab
    dedupeOpenTabs(sessionID)
    if (activeMachine.value === pendingId) activeMachine.value = sessionID
  }

  const failPendingTab = (pendingId, error) => {
    const tab = openTabs.value.find((t) => t.machineName === pendingId)
    if (tab) {
      tab.connecting = false
      tab.connected = false
    }
    pushShellOutput(pendingId, 'line', `连接失败: ${error}`)
  }

  const markTabConnecting = (sessionID) => {
    const tab = openTabs.value.find((t) => t.machineName === sessionID)
    if (tab) {
      tab.connecting = true
      tab.connected = false
      if (!isLocalSession(sessionID)) pushShellOutput(sessionID, 'line', '连接中...')
      activeMachine.value = sessionID
      return sessionID
    }
    return addPendingTab({
      configName: resolveRemoteConfigName(sessionID),
      kind: isLocalSession(sessionID) ? 'local' : 'remote',
      tabLabel: isLocalSession(sessionID) ? localTabLabel(sessionID) : sessionID,
      sessionID,
    })
  }

  const findDisconnectedTab = (target) => {
    const key = String(target || '').trim()
    if (!key) return null
    const byName = openTabs.value.filter(
      (t) => t.machineName === key && !t.connected && !isPendingSession(t.machineName),
    )
    if (byName.length === 1) return byName[0]
    const byCfg = openTabs.value.filter(
      (t) => t.configName === key && !t.connected && !t.connecting,
    )
    if (byCfg.length === 1) return byCfg[0]
    return null
  }

  const applyLiveToTab = (sessionID, live) => {
    const tab = openTabs.value.find((t) => t.machineName === sessionID)
    if (!tab) return false
    Object.assign(tab, live || {}, {
      machineName: sessionID,
      connected: !!live?.connected,
      connecting: !!live?.connecting,
      tabLabel: live?.tabLabel || tab.tabLabel,
      configName: live?.configName || tab.configName,
    })
    return true
  }

  const markTabFailed = (sessionID, errorText = '') => {
    const tab = openTabs.value.find((t) => t.machineName === sessionID)
    if (!tab) return
    tab.connecting = false
    tab.connected = false
    if (errorText) pushShellOutput(sessionID, 'line', errorText)
  }

  /** 同一 machineName 只保留一个 tab；优先保留已连接且非连接中的状态 */
  const dedupeOpenTabs = (preferId = '') => {
    const byName = new Map()
    for (const tab of openTabs.value) {
      const key = tab.machineName
      const existing = byName.get(key)
      if (!existing) {
        byName.set(key, tab)
        continue
      }
      const score = (t) => {
        let s = 0
        if (t.connected) s += 4
        if (!t.connecting) s += 2
        if (t.machineName === preferId) s += 1
        return s
      }
      if (score(tab) >= score(existing)) byName.set(key, tab)
    }
    const order = tabOrder.value.length
      ? [...tabOrder.value]
      : sortTabs(openTabs.value).map((t) => t.machineName)
    const ordered = []
    const seenOrder = new Set()
    for (const id of order) {
      if (!byName.has(id) || seenOrder.has(id)) continue
      seenOrder.add(id)
      ordered.push(byName.get(id))
    }
    for (const [id, tab] of byName) {
      if (!seenOrder.has(id)) ordered.push(tab)
    }
    openTabs.value = ordered
    tabOrder.value = ordered.map((t) => t.machineName)
  }

  const pendingTabForLive = (live, tabs) => {
    if (!live?.machineName) return null
    if (tabs.some((t) => t.machineName === live.machineName && !isPendingSession(t.machineName))) {
      return null
    }
    const cfg = live.configName || live.machineName
    const pendings = tabs.filter((t) => isPendingSession(t.machineName) && t.connecting)
    if (!pendings.length) return null

    const exact = pendings.find((t) => t.configName === cfg || t.configName === live.machineName)
    if (exact) return exact

    if (isLocalSession(live.machineName)) {
      const localPending = pendings.filter((t) => t.kind === 'local' && t.configName === 'local')
      if (localPending.length) return localPending[0]
    }
    return null
  }

  const mergeOpenTabsFromBackend = (list) => {
    const backend = Array.isArray(list) ? list : []
    const liveMap = new Map(
      backend
        .filter((s) => s?.machineName && (s.connected || s.connecting))
        .filter((s) => !closingSessionIds.has(s.machineName))
        .map((s) => [s.machineName, s]),
    )

    openTabs.value = openTabs.value
      .filter((tab) => !closingSessionIds.has(tab.machineName))
      .filter((tab) => !isPendingSession(tab.machineName) || !tab.connected)
      .map((tab) => {
        if (isPendingSession(tab.machineName)) return tab
        const live = liveMap.get(tab.machineName)
        if (live) {
          liveMap.delete(tab.machineName)
          return {
            ...tab,
            ...live,
            connected: !!live.connected,
            connecting: !!live.connecting,
            connectedAt: tab.connectedAt || Date.now(),
            tabLabel: live.tabLabel || tab.tabLabel,
            configName: live.configName || tab.configName,
            lastCwd: tab.lastCwd,
          }
        }
        if (tab.connecting || isPendingSession(tab.machineName)) return tab
        if (tab.connected && isLocalSession(tab.machineName)) return tab
        return { ...tab, connected: false, connecting: false }
      })

    for (const [sessionID, live] of liveMap) {
      if (closingSessionIds.has(sessionID)) continue
      if (openTabs.value.some((t) => t.machineName === sessionID)) {
        applyLiveToTab(sessionID, live)
        continue
      }
      const pending = pendingTabForLive(live, openTabs.value)
      if (pending) {
        finalizeShellOutputMigration(pending.machineName, sessionID)
        const idx = openTabs.value.indexOf(pending)
        if (idx >= 0) {
          openTabs.value[idx] = {
            ...pending,
            ...live,
            machineName: sessionID,
            connected: !!live.connected,
            connecting: !!live.connecting,
            connectedAt: pending.connectedAt || Date.now(),
            tabLabel: live.tabLabel || pending.tabLabel,
            configName: live.configName || pending.configName,
            kind: live.kind || pending.kind || (isLocalSession(sessionID) ? 'local' : 'remote'),
          }
          replaceTabOrder(pending.machineName, sessionID)
          if (activeMachine.value === pending.machineName) activeMachine.value = sessionID
        }
        continue
      }
      if (openTabs.value.some((t) => t.machineName === sessionID)) continue
      // 本机 tab 仅由 connectLocal 显式打开，避免 status 竞态产生无法输入的占位 tab
      if (isLocalSession(sessionID)) continue
      openTabs.value.push({
        ...live,
        connected: !!live.connected,
        connecting: !!live.connecting,
        connectedAt: Date.now(),
        tabLabel: live.tabLabel || live.machineName,
        configName: live.configName || live.machineName,
        kind: live.kind || 'remote',
      })
      ensureTabOrder(sessionID)
    }

    dedupeOpenTabs()

    if (activeMachine.value && !openTabs.value.some((t) => t.machineName === activeMachine.value)) {
      activeMachine.value = workspaceSessions.value[0]?.machineName || ''
    }
    const openIds = new Set(openTabs.value.map((t) => t.machineName))
    broadcastTargets.value = broadcastTargets.value.filter((id) => openIds.has(id))
    splitSessionIds.value = splitSessionIds.value.filter((id) => openIds.has(id))
  }

  const syncSessions = async () => {
    try {
      sessions.value = (await App.GetShellSessions()) || []
      mergeOpenTabsFromBackend(sessions.value)
    } catch {
      sessions.value = []
    }
  }

  const loadMachines = async () => {
    try {
      shellMachines.value = sortMachinesByName((await App.GetMachines()) || [])
    } catch {
      shellMachines.value = []
    }
  }

  const handleShellStatus = (list) => {
    sessions.value = Array.isArray(list) ? list : []
    mergeOpenTabsFromBackend(sessions.value)
  }

  /** 始终新建远程会话（同机可开多个）；若已有断开 tab 则重连该 tab */
  const connect = async (configName) => {
    if (!configName) return false
    if (isLocalSession(configName)) {
      return connectLocal('')
    }
    const inFlight = openTabs.value.find((t) => t.configName === configName && t.connecting)
    if (inFlight) {
      activeMachine.value = inFlight.machineName
      return false
    }
    const disconnected = findDisconnectedTab(configName)
    if (disconnected) {
      return connectOrReconnect(disconnected.machineName)
    }
    const pendingId = addPendingTab({ configName, kind: 'remote', tabLabel: configName })
    try {
      const sessionID = await App.ConnectShell(configName)
      if (!sessionID) throw new Error('未返回会话 ID')
      if (!openTabs.value.some((t) => t.machineName === pendingId)) return false
      const live =
        sessions.value.find((s) => s.machineName === sessionID) || {
          machineName: sessionID,
          configName,
          kind: 'remote',
          connected: false,
          connecting: true,
        }
      finalizePendingTab(pendingId, sessionID, live)
      await syncSessions()
      activeMachine.value = sessionID
      return true
    } catch (error) {
      failPendingTab(pendingId, error)
      const hk = parseHostKeyError(error)
      if (hk) {
        pendingHostKey.value = { ...hk, configName, pendingId }
      } else if (/私钥需要口令|passphrase/i.test(String(error || ''))) {
        try {
          const { value } = await ElMessageBox.prompt('请输入私钥口令', '密钥口令', {
            inputType: 'password',
            confirmButtonText: '连接',
            cancelButtonText: '取消',
          })
          // 暂存到机器敏感字段需后端 API；先提示用户到机器配置保存口令后重试
          if (value) {
            ElMessage.info('请到机器配置或密钥库填写「密钥口令」后重新连接（口令会加密保存）')
          }
        } catch {
          // cancel
        }
        ElMessage.error('连接失败: ' + error)
      } else {
        ElMessage.error('连接失败: ' + error)
      }
      return false
    }
  }

  const connectLocal = async (sessionID = '', command = '') => {
    if (sessionID) {
      return connectOrReconnect(sessionID)
    }
    try {
      const id = command
        ? await App.ConnectLocalShellCommand('', command)
        : await App.ConnectLocalShell('')
      if (!id) throw new Error('未返回会话 ID')

      pruneOrphanLocalPending()
      const live =
        sessions.value.find((s) => s.machineName === id) ||
        (await App.GetShellSessions())?.find((s) => s.machineName === id) || {
          machineName: id,
          configName: id,
          kind: 'local',
          tabLabel: localTabLabel(id),
          connected: true,
          connecting: false,
        }
      upsertOpenTab(id, live, { connected: true, connecting: false })
      pruneOrphanLocalPending()
      activeMachine.value = id
      await nextTick()
      await syncSessions()
      const tab = openTabs.value.find((t) => t.machineName === id)
      if (tab) {
        tab.connected = true
        tab.everConnected = true
        tab.connecting = false
        tab.kind = 'local'
      }
      ElMessage.success(command ? '已打开本机 Shell' : '已打开本机')
      return true
    } catch (error) {
      pruneOrphanLocalPending()
      ElMessage.error('打开本机失败: ' + error)
      return false
    }
  }

  const connectPendingTab = async (pendingId) => {
    const tab = openTabs.value.find((t) => t.machineName === pendingId)
    if (!tab || tab.connected || tab.connecting) return false

    const configName = tab.configName
    const kind = tab.kind || 'remote'
    try {
      let realId
      if (kind === 'local' || isLocalSession(configName)) {
        realId = await App.ConnectLocalShell('')
      } else {
        realId = await App.ConnectShell(configName)
      }
      if (!realId) throw new Error('未返回会话 ID')
      if (!openTabs.value.some((t) => t.machineName === pendingId)) return false
      const isLocal = kind === 'local' || isLocalSession(configName)
      const live =
        (await App.GetShellSessions())?.find((s) => s.machineName === realId) || {
          machineName: realId,
          configName: isLocal ? realId : configName,
          kind: isLocal ? 'local' : 'remote',
          tabLabel: tab.tabLabel || (isLocal ? localTabLabel(realId) : configName),
          connected: isLocal,
          connecting: !isLocal,
        }
      finalizePendingTab(pendingId, realId, live)
      await syncSessions()
      activeMachine.value = realId
      return true
    } catch (error) {
      if (openTabs.value.some((t) => t.machineName === pendingId)) {
        failPendingTab(pendingId, error)
      }
      ElMessage.error('连接失败: ' + error)
      return false
    }
  }

  /** 软断开后按会话 ID 重连（复用已有 tab，不新建） */
  const connectOrReconnect = async (sessionID) => {
    if (!sessionID) return false
    const existing = openTabs.value.find((t) => t.machineName === sessionID)
    if (existing?.connected && !existing?.connecting) {
      activeMachine.value = sessionID
      return true
    }
    if (existing?.connecting) {
      activeMachine.value = sessionID
      return false
    }
    if (isPendingSession(sessionID)) return connectPendingTab(sessionID)

    if (isLocalSession(sessionID)) {
      try {
        const id = await App.ConnectLocalShell(sessionID)
        const realId = id || sessionID
        const live =
          sessions.value.find((s) => s.machineName === realId) ||
          (await App.GetShellSessions())?.find((s) => s.machineName === realId) || {
            machineName: realId,
            configName: realId,
            kind: 'local',
            tabLabel: localTabLabel(realId),
            connected: true,
            connecting: false,
          }
        upsertOpenTab(realId, live, { connected: true, connecting: false })
        activeMachine.value = realId
        await nextTick()
        await syncSessions()
        const tab = openTabs.value.find((t) => t.machineName === realId)
        if (tab) {
          tab.connected = true
          tab.everConnected = true
          tab.connecting = false
        }
        return true
      } catch (error) {
        const tab = openTabs.value.find((t) => t.machineName === sessionID)
        if (tab) {
          tab.connecting = false
          tab.connected = false
        }
        ElMessage.error('重连失败: ' + error)
        return false
      }
    }

    const pendingId = markTabConnecting(sessionID)
    try {
      const id = await App.ReconnectShell(sessionID)
      const realId = id || sessionID
      if (!openTabs.value.some((t) => t.machineName === sessionID || t.machineName === pendingId)) {
        return false
      }
      await syncSessions()
      activeMachine.value = realId
      return true
    } catch (error) {
      if (isPendingSession(pendingId)) {
        failPendingTab(pendingId, error)
      } else {
        const tab = openTabs.value.find((t) => t.machineName === sessionID)
        if (tab) {
          tab.connecting = false
          pushShellOutput(sessionID, 'line', `连接失败: ${error}`)
        }
      }
      ElMessage.error('重连失败: ' + error)
      return false
    }
  }

  /** 软断开：关闭 SSH，保留 tab 与终端历史 */
  const disconnect = async (machineName) => {
    if (!machineName || isPendingSession(machineName)) return
    try {
      await App.DisconnectShell(machineName)
      discardShellOutputBuffer(machineName)
      const tab = openTabs.value.find((t) => t.machineName === machineName)
      if (tab) {
        tab.connected = false
        tab.connecting = false
      }
      sessions.value = (sessions.value || []).filter((s) => s.machineName !== machineName)
    } catch (error) {
      ElMessage.error('断开失败: ' + error)
    }
  }

  /** 关闭 tab：先从 UI 移除，再后台断开，避免等 Disconnect 才消失 */
  const closeSession = async (machineName) => {
    if (!machineName) return
    closingSessionIds.add(machineName)
    removeShellOutput(machineName)
    openTabs.value = openTabs.value.filter((t) => t.machineName !== machineName)
    removeTabOrder(machineName)
    sessions.value = (sessions.value || []).filter((s) => s.machineName !== machineName)
    broadcastTargets.value = broadcastTargets.value.filter((id) => id !== machineName)
    splitSessionIds.value = splitSessionIds.value.filter((id) => id !== machineName)
    if (activeMachine.value === machineName) {
      activeMachine.value = workspaceSessions.value[0]?.machineName || ''
    }
    try {
      if (!isPendingSession(machineName)) {
        await App.DisconnectShell(machineName)
      }
    } catch {
      // ignore
    } finally {
      closingSessionIds.delete(machineName)
    }
  }

  /** 批量关闭 tab：先清空 UI，再并行断开 */
  const closeSessions = async (machineNames) => {
    const names = [...new Set((machineNames || []).filter(Boolean))]
    if (!names.length) return
    const nameSet = new Set(names)
    for (const name of names) closingSessionIds.add(name)
    for (const name of names) removeShellOutput(name)
    openTabs.value = openTabs.value.filter((t) => !nameSet.has(t.machineName))
    tabOrder.value = tabOrder.value.filter((id) => !nameSet.has(id))
    sessions.value = (sessions.value || []).filter((s) => !nameSet.has(s.machineName))
    broadcastTargets.value = broadcastTargets.value.filter((id) => !nameSet.has(id))
    splitSessionIds.value = splitSessionIds.value.filter((id) => !nameSet.has(id))
    if (nameSet.has(activeMachine.value)) {
      activeMachine.value = workspaceSessions.value[0]?.machineName || ''
    }
    await Promise.all(
      names.map(async (name) => {
        try {
          if (!isPendingSession(name)) await App.DisconnectShell(name)
        } catch (e) {
          console.error('关闭会话失败:', name, e)
        } finally {
          closingSessionIds.delete(name)
        }
      }),
    )
  }

  const testMachine = async (machineName) => {
    testingName.value = machineName
    try {
      await App.TestMachineConnection(machineName)
      ElMessage.success('连接测试成功')
    } catch (error) {
      ElMessage.error('连接测试失败: ' + error)
    } finally {
      testingName.value = ''
    }
  }

  const toggleBroadcastTarget = (sessionID) => {
    const set = new Set(broadcastTargets.value)
    if (set.has(sessionID)) set.delete(sessionID)
    else set.add(sessionID)
    broadcastTargets.value = [...set]
  }

  const setSplitSessions = (ids) => {
    const list = (ids || []).filter(Boolean).slice(0, 4)
    splitSessionIds.value = list
  }

  const toggleSplitSession = (sessionID) => {
    const set = new Set(splitSessionIds.value)
    if (set.has(sessionID)) set.delete(sessionID)
    else if (set.size < 4) set.add(sessionID)
    splitSessionIds.value = [...set]
  }

  let shellEventsActive = false
  let offShellStatus = null
  let offShellData = null
  let offShellLine = null
  let offShellClear = null
  let offShellReconnecting = null
  let offShellReconnected = null
  let offShellReconnectFailed = null

  const setupShellEvents = () => {
    if (shellEventsActive) return
    offShellStatus = EventsOn('shell:status', handleShellStatus)
    offShellData = EventsOn('shell:data', (payload) => {
      if (payload?.machineName && payload?.data) {
        pushShellOutput(payload.machineName, 'data', payload.data)
      }
    })
    offShellLine = EventsOn('shell:line', (payload) => {
      if (!payload?.machineName || !payload?.line) return
      pushShellOutput(payload.machineName, 'line', payload.line)
      const line = String(payload.line)
      if (line.startsWith('连接失败') || line.startsWith('[连接断开]')) {
        markTabFailed(payload.machineName)
        const hk = parseHostKeyError(line)
        if (hk) {
          const tab = openTabs.value.find((t) => t.machineName === payload.machineName)
          pendingHostKey.value = {
            ...hk,
            configName: tab?.configName || resolveRemoteConfigName(payload.machineName),
            sessionId: payload.machineName,
          }
        }
      }
    })
    offShellClear = EventsOn('shell:clear', (payload) => {
      if (payload?.machineName) clearShellOutput(payload.machineName)
    })
    offShellReconnecting = EventsOn('shell:reconnecting', (payload) => {
      const name = payload?.machineName
      if (!name) return
      const tab = openTabs.value.find((t) => t.machineName === name)
      if (tab) {
        tab.connecting = true
        tab.reconnecting = true
        tab.reconnectAttempt = payload.attempt || 1
        tab.reconnectMax = payload.maxAttempts || 3
        tab.reconnectDelaySec = payload.delaySec || 0
      }
      const n = payload.attempt || 1
      const max = payload.maxAttempts || 3
      const delay = payload.delaySec || 0
      const delayHint = delay > 0 ? `，退避 ${delay}s` : ''
      pushShellOutput(name, 'line', `正在重连…（第 ${n}/${max} 次${delayHint}）`)
    })
    offShellReconnected = EventsOn('shell:reconnected', (payload) => {
      const name = payload?.machineName
      if (!name) return
      const tab = openTabs.value.find((t) => t.machineName === name)
      if (tab) {
        tab.connecting = false
        tab.reconnecting = false
        tab.connected = true
        tab.everConnected = true
        tab.reconnectAttempt = 0
        tab.reconnectDelaySec = 0
      }
      pushShellOutput(name, 'line', '重连成功')
      syncSessions()
    })
    offShellReconnectFailed = EventsOn('shell:reconnect-failed', (payload) => {
      const name = payload?.machineName
      if (!name) return
      const tab = openTabs.value.find((t) => t.machineName === name)
      if (tab) {
        tab.connecting = false
        tab.reconnecting = false
        tab.connected = false
        tab.reconnectAttempt = 0
        tab.reconnectDelaySec = 0
      }
      pushShellOutput(name, 'line', '自动重连失败，按 Enter 手动重连')
    })
    shellEventsActive = true
    syncSessions()
  }

  const teardownShellEvents = () => {
    offShellStatus?.()
    offShellData?.()
    offShellLine?.()
    offShellClear?.()
    offShellReconnecting?.()
    offShellReconnected?.()
    offShellReconnectFailed?.()
    offShellStatus = null
    offShellData = null
    offShellLine = null
    offShellClear = null
    offShellReconnecting = null
    offShellReconnected = null
    offShellReconnectFailed = null
    shellEventsActive = false
  }

  /** 首次进入 Shell 或需要后台会话同步时调用 */
  const ensureShellReady = async () => {
    setupShellEvents()
    await loadMachines()
  }

  onUnmounted(() => {
    teardownShellEvents()
  })

  return {
    sessions,
    openTabs,
    tabOrder,
    activeMachine,
    shellMachines,
    connectingName,
    testingName,
    workspaceSessions,
    connectedSessions,
    connectedCount,
    openSessionCount,
    broadcastEnabled,
    broadcastTargets,
    splitSessionIds,
    syncSessions,
    loadMachines,
    connect,
    connectLocal,
    connectOrReconnect,
    disconnect,
    pendingHostKey,
    closeSession,
    closeSessions,
    testMachine,
    toggleBroadcastTarget,
    setSplitSessions,
    toggleSplitSession,
    reorderTabs,
    setupShellEvents,
    teardownShellEvents,
    ensureShellReady,
    updateTabLastCwd,
    isMachineConnected: (name) => isMachineConnected(name, sessions.value),
  }
}
