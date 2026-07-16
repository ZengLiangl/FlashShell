import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { isMachineConnected, sortMachinesByName } from '../utils/machineGroups'
import {
  pushShellOutput,
  clearShellOutput,
  removeShellOutput,
  discardShellOutputBuffer,
} from '../utils/shellOutputBuffer'
import { remoteConfigName, buildKnownMachineNames } from '../utils/sessionId'

const isLocalSession = (name) => {
  const n = String(name || '')
  return n === 'local' || n.startsWith('local-')
}

export function useShell() {
  const sessions = ref([])
  /** 工作区打开的 tab（含软断开），按连接时间从左到右排序 */
  const openTabs = ref([])
  const activeMachine = ref('')
  const shellMachines = ref([])
  const connectingName = ref('')
  const testingName = ref('')
  /** 命令广播目标会话 ID 列表；空表示关闭广播 */
  const broadcastTargets = ref([])
  const broadcastEnabled = ref(false)
  /** 平行视图：同时显示的会话 ID（最多 4） */
  const splitSessionIds = ref([])

  const resolveRemoteConfigName = (sessionID) =>
    remoteConfigName(sessionID, buildKnownMachineNames(shellMachines.value))

  const workspaceSessions = computed(() =>
    [...openTabs.value].sort((a, b) => (a.connectedAt || 0) - (b.connectedAt || 0)),
  )
  const connectedSessions = computed(() => workspaceSessions.value.filter((s) => s.connected))
  const connectedCount = computed(() => connectedSessions.value.length)
  const openSessionCount = computed(() => openTabs.value.length)

  const upsertOpenTab = (sessionID, liveStatus) => {
    const existing = openTabs.value.find((t) => t.machineName === sessionID)
    const kind = liveStatus?.kind || (isLocalSession(sessionID) ? 'local' : 'remote')
    const configName = liveStatus?.configName || (kind === 'local' ? sessionID : resolveRemoteConfigName(sessionID))
    const tabLabel = liveStatus?.tabLabel || sessionID
    if (existing) {
      existing.connected = true
      existing.kind = kind
      existing.configName = configName
      existing.tabLabel = tabLabel
      if (liveStatus) {
        existing.host = liveStatus.host || existing.host
        existing.user = liveStatus.user || existing.user
      }
      return
    }
    openTabs.value.push({
      machineName: sessionID,
      configName,
      tabLabel,
      connected: true,
      connectedAt: Date.now(),
      host: liveStatus?.host || '',
      user: liveStatus?.user || '',
      isRunning: liveStatus?.isRunning || false,
      currentCommand: liveStatus?.currentCommand || '',
      kind,
    })
  }

  const mergeOpenTabsFromBackend = (list) => {
    const backend = Array.isArray(list) ? list : []
    const liveMap = new Map(
      backend.filter((s) => s?.connected && s?.machineName).map((s) => [s.machineName, s]),
    )

    openTabs.value = openTabs.value.map((tab) => {
      const live = liveMap.get(tab.machineName)
      if (live) {
        liveMap.delete(tab.machineName)
        return {
          ...tab,
          ...live,
          connected: true,
          connectedAt: tab.connectedAt || Date.now(),
          tabLabel: live.tabLabel || tab.tabLabel,
          configName: live.configName || tab.configName,
        }
      }
      return { ...tab, connected: false }
    })

    for (const [, live] of liveMap) {
      openTabs.value.push({
        ...live,
        connected: true,
        connectedAt: Date.now(),
        tabLabel: live.tabLabel || live.machineName,
        configName: live.configName || live.machineName,
      })
    }

    if (activeMachine.value && !openTabs.value.some((t) => t.machineName === activeMachine.value)) {
      activeMachine.value = workspaceSessions.value[0]?.machineName || ''
    }
    // 清理已关闭会话的广播/分屏目标
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

  /** 始终新建远程会话（同机可开多个） */
  const connect = async (configName) => {
    if (!configName) return false
    if (isLocalSession(configName)) {
      return connectLocal('')
    }
    if (connectingName.value) {
      ElMessage.warning(`正在连接 ${connectingName.value}，请稍候`)
      return false
    }
    connectingName.value = configName
    try {
      const sessionID = await App.ConnectShell(configName)
      if (!sessionID) throw new Error('未返回会话 ID')
      activeMachine.value = sessionID
      await syncSessions()
      upsertOpenTab(
        sessionID,
        sessions.value.find((s) => s.machineName === sessionID) || {
          machineName: sessionID,
          configName,
          kind: 'remote',
        },
      )
      ElMessage.success(`已连接 ${configName}`)
      return true
    } catch (error) {
      ElMessage.error('连接失败: ' + error)
      return false
    } finally {
      connectingName.value = ''
    }
  }

  const connectLocal = async (sessionID = '') => {
    if (connectingName.value) {
      ElMessage.warning(`正在连接 ${connectingName.value}，请稍候`)
      return false
    }
    connectingName.value = sessionID || '本机'
    try {
      const id = await App.ConnectLocalShell(sessionID || '')
      if (!id) throw new Error('未返回会话 ID')
      activeMachine.value = id
      await syncSessions()
      upsertOpenTab(id, sessions.value.find((s) => s.machineName === id) || { machineName: id, kind: 'local' })
      ElMessage.success(sessionID ? '已重新连接本机' : '已打开本机')
      return true
    } catch (error) {
      ElMessage.error('打开本机失败: ' + error)
      return false
    } finally {
      connectingName.value = ''
    }
  }

  /** 软断开后按会话 ID 重连 */
  const connectOrReconnect = async (sessionID) => {
    if (!sessionID) return false
    if (isLocalSession(sessionID)) {
      return connectLocal(sessionID)
    }
    if (connectingName.value) {
      ElMessage.warning(`正在连接 ${connectingName.value}，请稍候`)
      return false
    }
    connectingName.value = sessionID
    try {
      const id = await App.ReconnectShell(sessionID)
      activeMachine.value = id || sessionID
      await syncSessions()
      upsertOpenTab(id || sessionID, sessions.value.find((s) => s.machineName === (id || sessionID)))
      ElMessage.success('已重新连接')
      return true
    } catch (error) {
      ElMessage.error('重连失败: ' + error)
      return false
    } finally {
      connectingName.value = ''
    }
  }

  /** 软断开：关闭 SSH，保留 tab 与终端历史 */
  const disconnect = async (machineName) => {
    if (!machineName) return
    try {
      await App.DisconnectShell(machineName)
      discardShellOutputBuffer(machineName)
      const tab = openTabs.value.find((t) => t.machineName === machineName)
      if (tab) tab.connected = false
      sessions.value = (sessions.value || []).filter((s) => s.machineName !== machineName)
    } catch (error) {
      ElMessage.error('断开失败: ' + error)
    }
  }

  /** 关闭 tab：断开（若仍连接）并移除工作区会话 */
  const closeSession = async (machineName) => {
    if (!machineName) return
    try {
      await App.DisconnectShell(machineName)
    } catch {
      // ignore
    }
    removeShellOutput(machineName)
    openTabs.value = openTabs.value.filter((t) => t.machineName !== machineName)
    sessions.value = (sessions.value || []).filter((s) => s.machineName !== machineName)
    broadcastTargets.value = broadcastTargets.value.filter((id) => id !== machineName)
    splitSessionIds.value = splitSessionIds.value.filter((id) => id !== machineName)
    if (activeMachine.value === machineName) {
      activeMachine.value = workspaceSessions.value[0]?.machineName || ''
    }
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

  const setupShellEvents = () => {
    teardownShellEvents()
    EventsOn('shell:status', handleShellStatus)
    EventsOn('shell:data', (payload) => {
      if (payload?.machineName && payload?.data) {
        pushShellOutput(payload.machineName, 'data', payload.data)
      }
    })
    EventsOn('shell:line', (payload) => {
      if (payload?.machineName && payload?.line) {
        pushShellOutput(payload.machineName, 'line', payload.line)
      }
    })
    EventsOn('shell:clear', (payload) => {
      if (payload?.machineName) clearShellOutput(payload.machineName)
    })
    syncSessions()
  }

  const teardownShellEvents = () => {
    EventsOff('shell:status', 'shell:data', 'shell:line', 'shell:clear')
  }

  onMounted(() => {
    setupShellEvents()
    loadMachines()
  })

  onUnmounted(() => {
    teardownShellEvents()
  })

  return {
    sessions,
    openTabs,
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
    closeSession,
    testMachine,
    toggleBroadcastTarget,
    setSplitSessions,
    toggleSplitSession,
    setupShellEvents,
    teardownShellEvents,
    isMachineConnected: (name) => isMachineConnected(name, sessions.value),
  }
}
