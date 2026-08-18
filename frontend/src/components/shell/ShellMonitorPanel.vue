<template>
  <div class="shell-monitor">
    <div class="monitor-header" @dblclick="onChromeTitleDblActivate" @mousedown="onChromeTitlePointerDown">
      <h3>机器监控</h3>
      <div class="monitor-header-spacer" aria-hidden="true" />
      <el-button v-if="activeMachine" class="conn-toggle" size="small" :type="activeConnected ? 'danger' : 'primary'"
        plain :title="activeConnected ? '断开连接（保留终端）' : '重新连接'" :loading="connecting"
        @click="$emit('toggle-connection')">
        <el-icon :size="14">
          <SwitchButton v-if="activeConnected" />
          <Connection v-else />
        </el-icon>
      </el-button>
    </div>

    <div v-if="!activeMachine" class="empty">连接机器后显示监控信息</div>
    <template v-else>
      <el-tabs v-model="monitorTab" class="monitor-tabs" @tab-change="onTabChange">
        <el-tab-pane label="概览" name="overview">
      <div class="machine-title">{{ activeMachine }}</div>

      <div class="field">
        <div class="label">IP</div>
        <div class="value-row">
          <span class="mono">{{ snapshot?.host || '-' }}</span>
          <el-tooltip content="复制" placement="top">
            <el-button size="small" text type="primary" :disabled="!snapshot?.host" @click="copyHost">
              <el-icon>
                <CopyDocument />
              </el-icon>
            </el-button>
          </el-tooltip>
        </div>
      </div>

      <div v-if="activeConnected" class="sysinfo-block">
        <button type="button" class="sysinfo-toggle" @click="sysinfoOpen = !sysinfoOpen">
          <span>系统信息</span>
          <el-icon :class="{ rotated: sysinfoOpen }">
            <ArrowDown />
          </el-icon>
        </button>
        <div v-show="sysinfoOpen" class="sysinfo-body">
          <div v-if="sysinfoLoading && !sysinfo" class="empty-sm">加载中…</div>
          <div v-else-if="sysinfoError" class="error-sm">{{ sysinfoError }}</div>
          <template v-else-if="sysinfo">
            <div v-for="row in sysinfoRows" :key="row.label" class="sysinfo-row">
              <span class="sysinfo-label">{{ row.label }}</span>
              <span class="sysinfo-value">{{ row.value || '-' }}</span>
            </div>
          </template>
        </div>
      </div>

      <div class="field">
        <div class="label">运行时长</div>
        <div class="value">{{ snapshot?.uptimeText || '0' }}</div>
      </div>

      <div class="metric" :class="{ 'is-high': isHighUsage(snapshot?.cpuPercent) }">
        <div class="metric-head">
          <span>CPU</span>
          <span class="metric-value" :class="{ 'is-danger': isHighUsage(snapshot?.cpuPercent) }">
            {{ formatPct(snapshot?.cpuPercent) }}
          </span>
        </div>
        <el-progress :percentage="clampPct(snapshot?.cpuPercent)" :stroke-width="12"
          :status="progressStatus(snapshot?.cpuPercent)" :show-text="false" />
      </div>

      <div class="metric" :class="{ 'is-high': isHighUsage(snapshot?.memPercent) }">
        <div class="metric-head">
          <span>内存</span>
          <span class="metric-value" :class="{ 'is-danger': isHighUsage(snapshot?.memPercent) }">
            {{ formatPct(snapshot?.memPercent) }} · {{ snapshot?.memUsed || '0' }}/{{ snapshot?.memTotal || '0' }}
          </span>
        </div>
        <el-progress :percentage="clampPct(snapshot?.memPercent)" :stroke-width="12"
          :status="progressStatus(snapshot?.memPercent)" :show-text="false" />
      </div>

      <div
        v-if="hasSwap"
        class="metric"
        :class="{ 'is-high': isHighUsage(snapshot?.swapPercent) }"
      >
        <div class="metric-head">
          <span>交换</span>
          <span class="metric-value" :class="{ 'is-danger': isHighUsage(snapshot?.swapPercent) }">
            {{ formatPct(snapshot?.swapPercent) }} · {{ snapshot?.swapUsed || '0' }}/{{ snapshot?.swapTotal || '0' }}
          </span>
        </div>
        <el-progress :percentage="clampPct(snapshot?.swapPercent)" :stroke-width="12"
          :status="progressStatus(snapshot?.swapPercent)" :show-text="false" />
      </div>

      <div class="top-block">
        <div class="label">CPU 占用 TOP5</div>
        <div class="top-head">
          <span class="top-pid">PID</span>
          <span class="top-mem">内存</span>
          <span class="top-cpu">CPU</span>
          <span class="top-cmd">命令</span>
        </div>
        <div v-if="!(snapshot?.topMem || []).length" class="empty-sm">暂无数据</div>
        <div v-for="(p, idx) in (snapshot?.topMem || [])" :key="p.pid + idx" class="top-row">
          <div class="top-pid">{{ p.pid }}</div>
          <div class="top-mem" :class="{ 'is-danger': isHighUsage(p.mem) }">{{ formatPct1(p.mem) }}</div>
          <div class="top-cpu" :class="{ 'is-danger': isHighUsage(p.cpu) }">{{ formatPct1(p.cpu) }}</div>
          <div class="top-cmd">
            <el-tooltip
              :disabled="!p.command"
              placement="top"
              :show-after="250"
              :hide-after="0"
              popper-class="shell-monitor-cmd-tip"
            >
              <template #content>
                <div class="top-cmd-tip">{{ p.command }}</div>
              </template>
              <span class="top-cmd-text">{{ p.command }}</span>
            </el-tooltip>
          </div>
        </div>
      </div>

      <div v-if="netIfaces.length" class="net-block">
        <div class="net-head">
          <span class="net-up">↑ {{ snapshot?.netTxText || '0B/s' }}</span>
          <span class="net-down">↓ {{ snapshot?.netRxText || '0B/s' }}</span>
          <el-select v-model="selectedNetIface" class="net-iface-select" size="small" :disabled="!activeConnected"
            @change="onNetIfaceChange">
            <el-option v-for="iface in netIfaces" :key="iface" :label="iface" :value="iface" />
          </el-select>
        </div>
        <div class="net-chart">
          <div class="net-chart-y">
            <span>{{ netChartMaxText }}</span>
            <span>{{ netChartMidText }}</span>
            <span>0</span>
          </div>
          <div class="net-chart-bars">
            <div v-for="(pt, idx) in netHistory" :key="idx" class="net-bar-group">
              <div class="net-bar net-bar-tx" :style="{ height: barHeight(pt.tx) }"
                :title="`上行 ${formatRate(pt.tx)}`" />
              <div class="net-bar net-bar-rx" :style="{ height: barHeight(pt.rx) }"
                :title="`下行 ${formatRate(pt.rx)}`" />
            </div>
          </div>
        </div>
      </div>

      <div v-if="displayError" class="error">{{ displayError }}</div>
        </el-tab-pane>

        <el-tab-pane label="磁盘" name="disks">
          <div class="tab-toolbar">
            <span class="tab-toolbar-label">磁盘挂载</span>
            <el-button size="small" text type="primary" :loading="diskLoading" @click="loadDisks">刷新</el-button>
          </div>
          <div v-if="diskError" class="error-sm">{{ diskError }}</div>
          <div v-else-if="diskLoading && !diskList.length" class="empty-sm">加载中…</div>
          <div v-else-if="!diskList.length" class="empty-sm">暂无数据</div>
          <div v-else class="data-table-wrap">
            <div class="data-head disk-head">
              <span>路径</span><span>可用</span><span>大小</span><span>已用</span>
            </div>
            <div v-for="(d, idx) in diskList" :key="d.path + idx" class="data-row disk-row">
              <span class="data-cmd" :title="d.path">{{ d.path }}</span>
              <span>{{ d.avail || '-' }}</span>
              <span>{{ d.size || '-' }}</span>
              <span :class="{ 'is-danger': isHighUsage(d.usePercent) }">{{ d.usePct || '-' }}</span>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="进程" name="processes">
          <div class="tab-toolbar">
            <span class="tab-toolbar-label">进程列表</span>
            <el-button size="small" text type="primary" :loading="processLoading" @click="loadProcesses">刷新</el-button>
          </div>
          <div v-if="processError" class="error-sm">{{ processError }}</div>
          <div v-else-if="processLoading && !processList.length" class="empty-sm">加载中…</div>
          <div v-else-if="!processList.length" class="empty-sm">暂无数据</div>
          <div v-else class="data-table-wrap">
            <div class="data-head proc-head">
              <span>PID</span><span>用户</span><span>CPU</span><span>内存</span><span>命令</span>
            </div>
            <div v-for="(p, idx) in processList" :key="p.pid + idx" class="data-row proc-row">
              <span>{{ p.pid }}</span>
              <span>{{ p.user || '-' }}</span>
              <span :class="{ 'is-danger': isHighUsage(p.cpu) }">{{ formatPct1(p.cpu) }}</span>
              <span :class="{ 'is-danger': isHighUsage(p.mem) }">{{ formatPct1(p.mem) }}</span>
              <span class="data-cmd" :title="p.command">{{ p.command }}</span>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="端口" name="ports">
          <div class="tab-toolbar">
            <span class="tab-toolbar-label">监听端口</span>
            <el-button size="small" text type="primary" :loading="portsLoading" @click="loadPorts">刷新</el-button>
          </div>
          <div v-if="portsError" class="error-sm">{{ portsError }}</div>
          <div v-else-if="portsLoading && !portList.length" class="empty-sm">加载中…</div>
          <div v-else-if="!portList.length" class="empty-sm">暂无数据</div>
          <div v-else class="data-table-wrap">
            <div class="data-head port-head">
              <span>协议</span><span>地址</span><span>端口</span><span>PID</span><span>进程</span>
            </div>
            <div v-for="(p, idx) in portList" :key="p.proto + p.port + idx" class="data-row port-row">
              <span>{{ p.proto }}</span>
              <span>{{ p.address || '*' }}</span>
              <span>{{ p.port }}</span>
              <span>{{ p.pid || '-' }}</span>
              <span class="data-cmd" :title="p.process">{{ p.process || '-' }}</span>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </template>
  </div>
</template>

<script>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { Connection, SwitchButton, ArrowDown } from '@element-plus/icons-vue'
import { EventsOn } from '../../../wailsjs/runtime/runtime'
import * as App from '../../../wailsjs/go/app/App'
import { onChromeTitleDblActivate, onChromeTitlePointerDown } from '../../utils/windowChrome'
import {
  hasUsefulMonitorSnapshot,
  shouldDiscardMonitorResult,
  shouldKeepCurrentMonitorSnapshot,
  shouldReplaceMonitorCache,
} from '../../utils/shellTabViewCache'

const isAuxMissingError = (msg) => /辅助连接(未建立|不存在)/.test(String(msg || ''))
const DEFAULT_INTERVAL_MS = 1000
const NET_CHART_DEFAULT_MAX = 200 * 1024

const clampIntervalMs = (ms) => {
  const n = Number(ms)
  if (!Number.isFinite(n) || n < 200) return DEFAULT_INTERVAL_MS
  return Math.min(60000, Math.round(n))
}

export default {
  name: 'ShellMonitorPanel',
  components: { Connection, SwitchButton, ArrowDown },
  props: {
    activeMachine: { type: String, default: '' },
    activeConnected: { type: Boolean, default: false },
    connecting: { type: Boolean, default: false },
  },
  emits: ['toggle-connection'],
  setup(props) {
    const snapshot = ref(null)
    const loading = ref(false)
    const intervalMs = ref(DEFAULT_INTERVAL_MS)
    const netHistory = ref([])
    const selectedNetIface = ref('')
    const netIfaces = ref([])
    const sysinfoOpen = ref(false)
    const sysinfoLoading = ref(false)
    const sysinfo = ref(null)
    const sysinfoError = ref('')
    const monitorTab = ref('overview')
    const processList = ref([])
    const processLoading = ref(false)
    const processError = ref('')
    const portList = ref([])
    const portsLoading = ref(false)
    const portsError = ref('')
    const diskList = ref([])
    const diskLoading = ref(false)
    const diskError = ref('')
    let timer = null
    const NET_HISTORY_LEN = 24
    /** 按会话记住上次监控画面，切 tab 先展示缓存再后台刷新 */
    const cacheByMachine = Object.create(null)

    /** 占用 ≥80% 视为过高，标红提示 */
    const HIGH_USAGE = 80

    const loadInterval = async () => {
      try {
        const config = await App.GetSystemSettings()
        intervalMs.value = clampIntervalMs(config?.shellMonitorIntervalMs)
      } catch {
        intervalMs.value = DEFAULT_INTERVAL_MS
      }
    }

    const zeroSnapshot = (name, host = '') => ({
      machineName: name || '',
      host: host || '',
      uptimeText: '0',
      cpuPercent: 0,
      memPercent: 0,
      memUsed: '0',
      memTotal: '0',
      swapPercent: 0,
      swapUsed: '0',
      swapTotal: '0',
      topMem: [],
      netIface: '',
      netIfaces: [],
      netRxRate: 0,
      netTxRate: 0,
      netRxText: '0B/s',
      netTxText: '0B/s',
      error: '',
    })

    /** 连接中 / 未连接：不拉监控，全部归零 */
    const isIdle = () => !props.activeConnected || props.connecting

    const hasUsefulSnapshot = hasUsefulMonitorSnapshot

    const captureMonitorCache = (name) => {
      const key = String(name || '').trim()
      if (!key) return
      if (!shouldReplaceMonitorCache({
        current: snapshot.value,
        hasExisting: !!cacheByMachine[key],
      })) return
      cacheByMachine[key] = {
        snapshot: snapshot.value,
        netHistory: netHistory.value.slice(),
        selectedNetIface: selectedNetIface.value,
        netIfaces: netIfaces.value.slice(),
        sysinfo: sysinfo.value,
        sysinfoError: sysinfoError.value,
        processList: processList.value.slice(),
        processError: processError.value,
        portList: portList.value.slice(),
        portsError: portsError.value,
        diskList: diskList.value.slice(),
        diskError: diskError.value,
      }
    }

    const restoreMonitorCache = (name) => {
      const key = String(name || '').trim()
      const cached = key ? cacheByMachine[key] : null
      if (!cached || !hasUsefulSnapshot(cached.snapshot)) return false
      snapshot.value = cached.snapshot
      netHistory.value = cached.netHistory.slice()
      selectedNetIface.value = cached.selectedNetIface
      netIfaces.value = cached.netIfaces.slice()
      sysinfo.value = cached.sysinfo
      sysinfoError.value = cached.sysinfoError
      processList.value = cached.processList.slice()
      processError.value = cached.processError
      portList.value = cached.portList.slice()
      portsError.value = cached.portsError
      diskList.value = cached.diskList.slice()
      diskError.value = cached.diskError
      return true
    }

    const resetToZero = (host = '') => {
      stopTimer()
      netHistory.value = []
      selectedNetIface.value = ''
      netIfaces.value = []
      sysinfo.value = null
      sysinfoError.value = ''
      if (!props.activeMachine) {
        snapshot.value = null
        return
      }
      snapshot.value = zeroSnapshot(props.activeMachine, host || snapshot.value?.host || '')
    }

    const displayError = computed(() => {
      const err = snapshot.value?.error
      if (!err || isAuxMissingError(err)) return ''
      return err
    })

    /** 有交换分区（total>0）才展示；无 Swap 的机器不占位 */
    const hasSwap = computed(() => {
      const total = String(snapshot.value?.swapTotal || '').trim()
      if (!total || total === '0' || total === '0 B' || total === '0B') return false
      const pct = Number(snapshot.value?.swapPercent)
      // 后端仅在 swap total>0 时写入；再兜底排除纯零串
      return Number.isFinite(pct) || /[1-9]/.test(total)
    })

    const clampPct = (v) => {
      const n = Number(v) || 0
      return Math.max(0, Math.min(100, Math.round(n)))
    }
    const formatPct = (v) => `${clampPct(v)}%`
    const formatPct1 = (v) => {
      const n = Number(v)
      if (!Number.isFinite(n)) return '0.0%'
      return `${n.toFixed(1)}%`
    }
    const isHighUsage = (v) => {
      const n = Number(v)
      return Number.isFinite(n) && n >= HIGH_USAGE
    }
    const progressStatus = (v) => (isHighUsage(v) ? 'exception' : undefined)

    const formatRate = (bps) => {
      const n = Number(bps) || 0
      if (n < 1024) return `${Math.round(n)}B/s`
      if (n < 1024 * 1024) return `${Math.round(n / 1024)}K/s`
      return `${(n / 1024 / 1024).toFixed(1)}M/s`
    }

    const pushNetHistory = (rx, tx) => {
      const next = [...netHistory.value, { rx: Number(rx) || 0, tx: Number(tx) || 0 }]
      netHistory.value = next.slice(-NET_HISTORY_LEN)
    }

    const netChartMax = computed(() => {
      let max = NET_CHART_DEFAULT_MAX
      for (const pt of netHistory.value) {
        max = Math.max(max, pt.rx, pt.tx)
      }
      return max
    })

    const netChartMaxText = computed(() => formatRate(netChartMax.value).replace('/s', ''))
    const netChartMidText = computed(() => formatRate(netChartMax.value / 2).replace('/s', ''))

    const barHeight = (v) => {
      const max = netChartMax.value || 1
      const n = Number(v) || 0
      if (n <= 0) return '0'
      const pct = (n / max) * 100
      return `${Math.max(2, Math.min(100, pct))}%`
    }

    const sysinfoRows = computed(() => {
      const i = sysinfo.value
      if (!i) return []
      return [
        { label: '主机名', value: i.hostname },
        { label: '操作系统', value: i.os },
        { label: '内核', value: i.kernel },
        { label: '架构', value: i.arch },
        { label: 'CPU', value: i.cpuModel },
        { label: '磁盘', value: i.diskSummary },
      ]
    })

    const loadSystemInfo = async () => {
      if (!props.activeMachine || isIdle()) {
        if (!sysinfo.value) sysinfoError.value = ''
        return
      }
      const machineAtStart = props.activeMachine
      const had = !!(sysinfo.value && sysinfo.value.machineName === props.activeMachine)
      sysinfoLoading.value = !had
      if (!had) sysinfoError.value = ''
      try {
        const data = await App.GetShellSystemInfo(props.activeMachine)
        if (shouldDiscardMonitorResult({
          idle: isIdle(),
          activeMachine: props.activeMachine,
          machineAtStart,
        })) return
        if (data?.error) {
          // 辅助通道短暂缺失：不直接刷红，稍后由连接态变化 / 展开时重试
          if (isAuxMissingError(data.error)) {
            sysinfoError.value = ''
            if (!had) sysinfo.value = null
          } else {
            sysinfoError.value = data.error
            sysinfo.value = data
          }
        } else {
          sysinfo.value = data
        }
      } catch (e) {
        if (shouldDiscardMonitorResult({
          idle: isIdle(),
          activeMachine: props.activeMachine,
          machineAtStart,
        })) return
        if (isAuxMissingError(e)) {
          sysinfoError.value = ''
          if (!had) sysinfo.value = null
        } else {
          sysinfoError.value = String(e)
          if (!had) sysinfo.value = null
        }
      } finally {
        sysinfoLoading.value = false
      }
    }

    const syncNetIfaces = (snap) => {
      const list = Array.isArray(snap?.netIfaces) ? snap.netIfaces.filter(Boolean) : []
      if (list.length) netIfaces.value = list
      const iface = snap?.netIface || list[0] || ''
      if (iface && (!selectedNetIface.value || !netIfaces.value.includes(selectedNetIface.value))) {
        selectedNetIface.value = iface
      }
    }

    const onNetIfaceChange = () => {
      netHistory.value = []
      refresh()
    }

    const loadProcesses = async () => {
      if (!props.activeMachine || isIdle()) {
        processList.value = []
        processError.value = ''
        return
      }
      processLoading.value = true
      processError.value = ''
      const machineAtStart = props.activeMachine
      try {
        const data = await App.GetShellProcessList(props.activeMachine)
        if (shouldDiscardMonitorResult({
          idle: isIdle(),
          activeMachine: props.activeMachine,
          machineAtStart,
        })) return
        if (isAuxMissingError(data?.error)) return
        if (data?.error) {
          processError.value = data.error
          processList.value = []
        } else {
          processList.value = data?.processes || []
        }
      } catch (e) {
        if (shouldDiscardMonitorResult({
          idle: isIdle(),
          activeMachine: props.activeMachine,
          machineAtStart,
        })) return
        if (!isAuxMissingError(e)) processError.value = String(e)
        processList.value = []
      } finally {
        processLoading.value = false
      }
    }

    const loadPorts = async () => {
      if (!props.activeMachine || isIdle()) {
        portList.value = []
        portsError.value = ''
        return
      }
      portsLoading.value = true
      portsError.value = ''
      const machineAtStart = props.activeMachine
      try {
        const data = await App.GetShellListenPorts(props.activeMachine)
        if (shouldDiscardMonitorResult({
          idle: isIdle(),
          activeMachine: props.activeMachine,
          machineAtStart,
        })) return
        if (isAuxMissingError(data?.error)) return
        if (data?.error) {
          portsError.value = data.error
          portList.value = []
        } else {
          portList.value = data?.ports || []
        }
      } catch (e) {
        if (shouldDiscardMonitorResult({
          idle: isIdle(),
          activeMachine: props.activeMachine,
          machineAtStart,
        })) return
        if (!isAuxMissingError(e)) portsError.value = String(e)
        portList.value = []
      } finally {
        portsLoading.value = false
      }
    }

    const loadDisks = async () => {
      if (!props.activeMachine || isIdle()) {
        diskList.value = []
        diskError.value = ''
        return
      }
      diskLoading.value = true
      diskError.value = ''
      const machineAtStart = props.activeMachine
      try {
        const data = await App.GetShellDiskList(props.activeMachine)
        if (shouldDiscardMonitorResult({
          idle: isIdle(),
          activeMachine: props.activeMachine,
          machineAtStart,
        })) return
        if (isAuxMissingError(data?.error)) return
        if (data?.error) {
          diskError.value = data.error
          diskList.value = []
        } else {
          diskList.value = data?.disks || []
        }
      } catch (e) {
        if (shouldDiscardMonitorResult({
          idle: isIdle(),
          activeMachine: props.activeMachine,
          machineAtStart,
        })) return
        if (!isAuxMissingError(e)) diskError.value = String(e)
        diskList.value = []
      } finally {
        diskLoading.value = false
      }
    }

    const onTabChange = (name) => {
      if (name === 'processes') loadProcesses()
      if (name === 'ports') loadPorts()
      if (name === 'disks') loadDisks()
    }

    const refresh = async () => {
      if (!props.activeMachine) {
        snapshot.value = null
        return
      }
      if (isIdle()) {
        resetToZero(snapshot.value?.host || '')
        return
      }
      loading.value = true
      const machineAtStart = props.activeMachine
      try {
        const snap = await App.GetShellMonitor(props.activeMachine, selectedNetIface.value || '')
        // 轮询返回时若已断开/改切机器，丢弃结果，切勿把当前画面清零
        if (shouldDiscardMonitorResult({
          idle: isIdle(),
          activeMachine: props.activeMachine,
          machineAtStart,
        })) {
          return
        }
        const nextSnap = {
          ...zeroSnapshot(props.activeMachine),
          ...snap,
          uptimeText: snap?.uptimeText || '0',
          memUsed: snap?.memUsed || '0',
          memTotal: snap?.memTotal || '0',
          swapPercent: snap?.swapPercent || 0,
          swapUsed: snap?.swapUsed || '0',
          swapTotal: snap?.swapTotal || '0',
          topMem: snap?.topMem || [],
          netIface: snap?.netIface || '',
          netIfaces: snap?.netIfaces || [],
          netRxText: snap?.netRxText || '0B/s',
          netTxText: snap?.netTxText || '0B/s',
          error: isAuxMissingError(snap?.error) ? '' : (snap?.error || ''),
        }
        // 辅助通道短暂缺失或空快照：只保住当前这台机器的上次有效画面
        if (shouldKeepCurrentMonitorSnapshot({
          current: snapshot.value,
          activeMachine: props.activeMachine,
          incoming: nextSnap,
          auxMissing: isAuxMissingError(snap?.error),
        })) {
          return
        }
        if (isAuxMissingError(snap?.error) || !hasUsefulSnapshot(nextSnap)) {
          snapshot.value = {
            ...zeroSnapshot(props.activeMachine, snap?.host || ''),
            host: snap?.host || '',
          }
          return
        }
        snapshot.value = nextSnap
        syncNetIfaces(snap)
        if (snap?.netIface) {
          pushNetHistory(snap.netRxRate, snap.netTxRate)
        }
        captureMonitorCache(props.activeMachine)
      } catch (e) {
        if (shouldDiscardMonitorResult({
          idle: isIdle(),
          activeMachine: props.activeMachine,
          machineAtStart,
        })) {
          return
        }
        if (isAuxMissingError(e)) {
          if (shouldKeepCurrentMonitorSnapshot({
            current: snapshot.value,
            activeMachine: props.activeMachine,
            incoming: null,
            auxMissing: true,
          })) return
          snapshot.value = zeroSnapshot(props.activeMachine, snapshot.value?.host || '')
        } else {
          snapshot.value = {
            ...zeroSnapshot(props.activeMachine, snapshot.value?.host || ''),
            error: String(e),
          }
        }
      } finally {
        loading.value = false
      }
    }

    const copyHost = async () => {
      const host = snapshot.value?.host
      if (!host) return
      try {
        await navigator.clipboard.writeText(host)
      } catch {
        // 静默失败：复制 IP 不弹轻提示
      }
    }

    const stopTimer = () => {
      if (timer) {
        clearInterval(timer)
        timer = null
      }
    }

    const startTimer = () => {
      stopTimer()
      if (!props.activeMachine) {
        snapshot.value = null
        return
      }
      if (isIdle()) {
        resetToZero()
        return
      }
      refresh()
      timer = setInterval(refresh, intervalMs.value)
    }

    const onSettingsChanged = (payload) => {
      const next = clampIntervalMs(payload?.shellMonitorIntervalMs)
      if (next === intervalMs.value) return
      intervalMs.value = next
      if (props.activeMachine && !isIdle()) {
        startTimer()
      }
    }

    watch(
      () => [props.activeMachine, props.activeConnected, props.connecting],
      ([name], prevTuple) => {
        const prevName = prevTuple?.[0]
        if (prevName && prevName !== name) captureMonitorCache(prevName)
        if (isIdle()) {
          resetToZero()
          return
        }
        if (name && name !== prevName) {
          restoreMonitorCache(name)
        }
        loadSystemInfo()
        startTimer()
        if (monitorTab.value === 'processes') loadProcesses()
        if (monitorTab.value === 'ports') loadPorts()
        if (monitorTab.value === 'disks') loadDisks()
      },
      { immediate: true },
    )

    watch(sysinfoOpen, (open) => {
      if (open && !isIdle() && !sysinfo.value && !sysinfoLoading.value) {
        loadSystemInfo()
      }
    })
    let offSystemSettingsChanged = null
    onMounted(async () => {
      await loadInterval()
      // 用返回的取消函数解绑，避免 EventsOff 清掉其它组件（如 ShellTerminal）的同名监听
      offSystemSettingsChanged = EventsOn('system-settings:changed', onSettingsChanged)
      startTimer()
    })
    onUnmounted(() => {
      stopTimer()
      offSystemSettingsChanged?.()
      offSystemSettingsChanged = null
    })

    return {
      snapshot,
      onChromeTitleDblActivate,
      onChromeTitlePointerDown,
      loading,
      displayError,
      hasSwap,
      netHistory,
      netIfaces,
      selectedNetIface,
      sysinfoOpen,
      sysinfoLoading,
      sysinfo,
      sysinfoError,
      sysinfoRows,
      monitorTab,
      processList,
      processLoading,
      processError,
      portList,
      portsLoading,
      portsError,
      diskList,
      diskLoading,
      diskError,
      loadProcesses,
      loadPorts,
      loadDisks,
      onTabChange,
      onNetIfaceChange,
      netChartMaxText,
      netChartMidText,
      clampPct,
      formatPct,
      formatPct1,
      formatRate,
      barHeight,
      isHighUsage,
      progressStatus,
      copyHost,
    }
  },
}
</script>

<style scoped>
.shell-monitor {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-width: 0;
  padding: 10px 12px;
  box-sizing: border-box;
  background: var(--app-panel-bg);
  color: var(--app-text);
  overflow: auto;
}

.monitor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
}

.monitor-header-spacer {
  flex: 1;
  min-width: 8px;
  align-self: stretch;
}

.monitor-header h3 {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
}

.conn-toggle {
  flex-shrink: 0;
  gap: 4px;
}

.machine-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 12px;
}

.field,
.metric,
.top-block {
  margin-bottom: 14px;
}

.label {
  font-size: 12px;
  color: var(--app-text-muted);
  margin-bottom: 4px;
}

.top-block .label {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
  margin-bottom: 6px;
}

.value-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.mono,
.value {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 13px;
}

.metric {
  margin-bottom: 16px;
}

.metric-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 6px;
}

.metric-head span:first-child {
  color: var(--app-text);
  letter-spacing: 0.02em;
}

.metric-value {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
}

.metric :deep(.el-progress-bar__outer) {
  height: 12px;
  background: color-mix(in srgb, var(--app-border) 65%, transparent);
}

.metric :deep(.el-progress-bar__inner) {
  border-radius: 6px;
}

.metric.is-high .metric-head span:first-child {
  color: var(--el-color-danger);
  font-weight: 600;
}

.metric-value.is-danger,
.top-mem.is-danger,
.top-cpu.is-danger {
  color: var(--el-color-danger);
  font-weight: 700;
}

.top-head,
.top-row {
  display: grid;
  grid-template-columns: 48px 48px 52px minmax(0, 1fr);
  gap: 6px;
  align-items: center;
  padding: 6px 0;
  font-size: 12px;
}

.top-head {
  color: var(--app-text-muted);
  border-bottom: 1px solid var(--app-border);
  padding-bottom: 4px;
  margin-bottom: 2px;
}

.top-row {
  border-bottom: 1px solid var(--app-border);
}

.top-pid,
.top-mem,
.top-cpu {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--app-text-secondary);
}

.top-cmd {
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
}

.top-cmd-text,
.top-cmd :deep(.el-tooltip__trigger) {
  display: block;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: default;
}

.top-cmd-tip {
  max-width: min(420px, 70vw);
  word-break: break-all;
  white-space: pre-wrap;
  line-height: 1.45;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
}

.empty,
.empty-sm {
  color: var(--app-text-muted);
  font-size: 12px;
  padding: 16px 0;
  text-align: center;
}

.empty-sm {
  padding: 8px 0;
  text-align: left;
}

.error {
  color: var(--terminal-error);
  font-size: 12px;
  margin-top: 8px;
}

.net-block {
  margin-bottom: 10px;
  padding-top: 4px;
  border-top: 1px solid var(--app-border);
}

.net-head {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 11px;
  margin-bottom: 4px;
  color: var(--app-text-muted);
}

.net-up {
  color: color-mix(in srgb, var(--app-text-muted) 82%, #9a8470 18%);
}

.net-down {
  color: color-mix(in srgb, var(--app-text-muted) 82%, #708870 18%);
}

.net-iface {
  margin-left: auto;
  color: var(--app-text-muted);
  font-size: 11px;
}

.net-chart {
  display: flex;
  gap: 4px;
  height: 52px;
  align-items: stretch;
}

.net-chart-y {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  font-size: 10px;
  color: var(--app-text-muted);
  width: 28px;
  flex-shrink: 0;
}

.net-chart-bars {
  flex: 1;
  display: flex;
  align-items: flex-end;
  gap: 2px;
  border-bottom: 1px solid color-mix(in srgb, var(--app-border) 80%, transparent);
  padding-bottom: 1px;
  min-width: 0;
}

.net-bar-group {
  flex: 1 1 0;
  min-width: 0;
  height: 100%;
  display: flex;
  flex-direction: row;
  align-items: flex-end;
  justify-content: center;
  gap: 1px;
}

.net-bar {
  flex: 1 1 0;
  min-width: 0;
  max-width: 6px;
  min-height: 0;
  border-radius: 1px 1px 0 0;
  opacity: 0.85;
}

.net-bar-tx {
  background: color-mix(in srgb, var(--app-text-muted) 78%, #9a8470 22%);
}

.net-bar-rx {
  background: color-mix(in srgb, var(--app-text-muted) 78%, #708870 22%);
}

.sysinfo-block {
  margin-bottom: 14px;
}

.sysinfo-toggle {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 0;
  border: none;
  background: transparent;
  color: var(--app-text);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}

.sysinfo-toggle .el-icon {
  transition: transform 0.2s ease;
  color: var(--app-text-muted);
}

.sysinfo-toggle .el-icon.rotated {
  transform: rotate(180deg);
}

.sysinfo-body {
  padding: 4px 0 8px;
}

.sysinfo-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-bottom: 8px;
}

.sysinfo-label {
  font-size: 11px;
  color: var(--app-text-muted);
}

.sysinfo-value {
  font-size: 12px;
  line-height: 1.4;
  word-break: break-word;
}

.error-sm {
  color: var(--terminal-error);
  font-size: 12px;
}

.net-iface-select {
  margin-left: auto;
  width: 88px;
  flex-shrink: 0;
}

.net-iface-select :deep(.el-input__wrapper) {
  padding: 0 6px;
}

.net-iface-select :deep(.el-input__inner) {
  font-size: 11px;
}

.monitor-tabs :deep(.el-tabs__header) {
  margin-bottom: 8px;
}

.monitor-tabs :deep(.el-tabs__item) {
  font-size: 12px;
  padding: 0 10px;
  height: 30px;
}

.tab-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.tab-toolbar-label {
  font-size: 12px;
  font-weight: 600;
}

.data-table-wrap {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.data-head,
.data-row {
  display: grid;
  gap: 6px;
  align-items: center;
  font-size: 11px;
  padding: 5px 0;
}

.proc-head,
.proc-row {
  grid-template-columns: 48px 52px 44px 44px minmax(0, 1fr);
}

.port-head,
.port-row {
  grid-template-columns: 44px minmax(0, 1fr) 44px 44px minmax(0, 1fr);
}

.disk-head,
.disk-row {
  grid-template-columns: minmax(0, 1.4fr) 64px 64px 48px;
}

.data-head {
  color: var(--app-text-muted);
  border-bottom: 1px solid var(--app-border);
}

.data-row {
  border-bottom: 1px solid color-mix(in srgb, var(--app-border) 70%, transparent);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

.data-cmd {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>

<style>
.shell-monitor-cmd-tip {
  max-width: min(420px, 70vw) !important;
}
</style>
