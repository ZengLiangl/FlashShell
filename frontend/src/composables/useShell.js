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

export function useShell() {
  const sessions = ref([])
  /** 工作区打开的 tab（含软断开），按连接时间从左到右排序 */
  const openTabs = ref([])
  const activeMachine = ref('')
  const shellMachines = ref([])
  const connectingName = ref('')
  const testingName = ref('')

  const workspaceSessions = computed(() =>
    [...openTabs.value].sort((a, b) => (a.connectedAt || 0) - (b.connectedAt || 0))
  )
  const connectedSessions = computed(() => workspaceSessions.value.filter((s) => s.connected))
  const connectedCount = computed(() => connectedSessions.value.length)
  const openSessionCount = computed(() => openTabs.value.length)

  const upsertOpenTab = (machineName, liveStatus) => {
    const existing = openTabs.value.find((t) => t.machineName === machineName)
    const kind = liveStatus?.kind || (String(machineName).startsWith('local') ? 'local' : 'remote')
    if (existing) {
      existing.connected = true
      existing.kind = kind
      if (liveStatus) {
        existing.host = liveStatus.host || existing.host
        existing.user = liveStatus.user || existing.user
      }
      return
    }
    openTabs.value.push({
      machineName,
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
      backend.filter((s) => s?.connected && s?.machineName).map((s) => [s.machineName, s])
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
        }
      }
      return { ...tab, connected: false }
    })

    for (const [, live] of liveMap) {
      openTabs.value.push({
        ...live,
        connected: true,
        connectedAt: Date.now(),
      })
    }

    if (activeMachine.value && !openTabs.value.some((t) => t.machineName === activeMachine.value)) {
      activeMachine.value = workspaceSessions.value[0]?.machineName || ''
    }
  }

  const syncSessions = async () => {
    try {
      sessions.value = await App.GetShellSessions() || []
      mergeOpenTabsFromBackend(sessions.value)
    } catch {
      sessions.value = []
    }
  }

  const loadMachines = async () => {
    try {
      shellMachines.value = sortMachinesByName(await App.GetMachines() || [])
    } catch {
      shellMachines.value = []
    }
  }

  const handleShellStatus = (list) => {
    sessions.value = Array.isArray(list) ? list : []
    mergeOpenTabsFromBackend(sessions.value)
  }

  const connect = async (machineName) => {
    if (!machineName) return false

    // 防止连点/并发重复 ConnectShell
    if (connectingName.value === machineName) {
      return false
    }
    if (connectingName.value) {
      ElMessage.warning(`正在连接 ${connectingName.value}，请稍候`)
      return false
    }

    connectingName.value = machineName
    try {
      // 先同步一次，避免本地 sessions 过期误判
      await syncSessions()
      if (isMachineConnected(machineName, sessions.value)) {
        upsertOpenTab(machineName, sessions.value.find((s) => s.machineName === machineName))
        activeMachine.value = machineName
        return true
      }

      await App.ConnectShell(machineName)
      activeMachine.value = machineName
      await syncSessions()
      upsertOpenTab(machineName, sessions.value.find((s) => s.machineName === machineName))
      ElMessage.success(`已连接 ${machineName}`)
      return true
    } catch (error) {
      const msg = String(error || '')
      // 后端幂等/竞态：已连接视为成功，切到该会话
      if (msg.includes('已连接')) {
        activeMachine.value = machineName
        await syncSessions()
        upsertOpenTab(machineName, sessions.value.find((s) => s.machineName === machineName))
        return true
      }
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

  const connectOrReconnect = async (machineName) => {
    if (!machineName) return false
    if (machineName === 'local' || String(machineName).startsWith('local-')) {
      return connectLocal(machineName)
    }
    return connect(machineName)
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

  const setupShellEvents = () => {
    // 热重载或重复 mount 时先解绑，避免 shell:data 重复监听导致按键 echo 双份
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
    syncSessions,
    loadMachines,
    connect,
    connectLocal,
    connectOrReconnect,
    disconnect,
    closeSession,
    testMachine,
    setupShellEvents,
    teardownShellEvents,
    isMachineConnected: (name) => isMachineConnected(name, sessions.value),
  }
}
