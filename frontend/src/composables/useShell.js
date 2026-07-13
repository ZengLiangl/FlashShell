import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { isMachineConnected } from '../utils/machineGroups'
import {
  pushShellOutput,
  clearShellOutput,
  removeShellOutput,
} from '../utils/shellOutputBuffer'

export function useShell() {
  const sessions = ref([])
  const activeMachine = ref('')
  const shellMachines = ref([])
  const connectingName = ref('')
  const testingName = ref('')

  const connectedSessions = computed(() => sessions.value.filter((s) => s.connected))
  const connectedCount = computed(() => connectedSessions.value.length)

  const syncSessions = async () => {
    try {
      sessions.value = await App.GetShellSessions() || []
      if (activeMachine.value && !isMachineConnected(activeMachine.value, sessions.value)) {
        activeMachine.value = connectedSessions.value[0]?.machineName || ''
      }
    } catch {
      sessions.value = []
    }
  }

  const loadMachines = async () => {
    try {
      shellMachines.value = await App.GetMachines() || []
    } catch {
      shellMachines.value = []
    }
  }

  const handleShellStatus = (list) => {
    sessions.value = Array.isArray(list) ? list : []
    if (activeMachine.value && !isMachineConnected(activeMachine.value, sessions.value)) {
      activeMachine.value = connectedSessions.value[0]?.machineName || ''
    }
  }

  const connect = async (machineName, taskRunning = false) => {
    if (taskRunning) {
      ElMessage.warning('任务正在执行，请先停止')
      return false
    }
    if (isMachineConnected(machineName, sessions.value)) {
      activeMachine.value = machineName
      return true
    }
    connectingName.value = machineName
    try {
      await App.ConnectShell(machineName)
      activeMachine.value = machineName
      await syncSessions()
      ElMessage.success(`已连接 ${machineName}`)
      return true
    } catch (error) {
      ElMessage.error('连接失败: ' + error)
      return false
    } finally {
      connectingName.value = ''
    }
  }

  const disconnect = async (machineName) => {
    try {
      await App.DisconnectShell(machineName)
      removeShellOutput(machineName)
      if (activeMachine.value === machineName) {
        activeMachine.value = connectedSessions.value[0]?.machineName || ''
      }
      await syncSessions()
    } catch (error) {
      ElMessage.error('断开失败: ' + error)
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
    activeMachine,
    shellMachines,
    connectingName,
    testingName,
    connectedSessions,
    connectedCount,
    syncSessions,
    loadMachines,
    connect,
    disconnect,
    testMachine,
    setupShellEvents,
    teardownShellEvents,
    isMachineConnected: (name) => isMachineConnected(name, sessions.value),
  }
}
