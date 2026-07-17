<template>
  <div class="shell-monitor">
    <div class="monitor-header">
      <h3>机器监控</h3>
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
          <div v-if="sysinfoLoading" class="empty-sm">加载中…</div>
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
          <div class="top-cmd" :title="p.command">{{ p.command }}</div>
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
    </template>
  </div>
</template>

<script>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Connection, SwitchButton, ArrowDown } from '@element-plus/icons-vue'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'
import * as App from '../../../wailsjs/go/app/App'

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
    let timer = null
    const NET_HISTORY_LEN = 24

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
        sysinfo.value = null
        sysinfoError.value = ''
        return
      }
      sysinfoLoading.value = true
      sysinfoError.value = ''
      try {
        const data = await App.GetShellSystemInfo(props.activeMachine)
        if (data?.error) {
          sysinfoError.value = data.error
          sysinfo.value = data
        } else {
          sysinfo.value = data
        }
      } catch (e) {
        sysinfoError.value = String(e)
        sysinfo.value = null
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
        // 轮询返回时若已断开/改切机器，丢弃结果
        if (isIdle() || props.activeMachine !== machineAtStart) {
          resetToZero(snap?.host || '')
          return
        }
        // 辅助通道缺失：保留标题布局，数值归零
        if (isAuxMissingError(snap?.error)) {
          snapshot.value = {
            ...zeroSnapshot(props.activeMachine, snap?.host || ''),
            host: snap?.host || '',
          }
        } else {
          snapshot.value = {
            ...zeroSnapshot(props.activeMachine),
            ...snap,
            uptimeText: snap?.uptimeText || '0',
            memUsed: snap?.memUsed || '0',
            memTotal: snap?.memTotal || '0',
            topMem: snap?.topMem || [],
            netIface: snap?.netIface || '',
            netIfaces: snap?.netIfaces || [],
            netRxText: snap?.netRxText || '0B/s',
            netTxText: snap?.netTxText || '0B/s',
            error: snap?.error || '',
          }
          syncNetIfaces(snap)
          if (snap?.netIface) {
            pushNetHistory(snap.netRxRate, snap.netTxRate)
          }
        }
      } catch (e) {
        if (isIdle() || props.activeMachine !== machineAtStart) {
          resetToZero()
          return
        }
        if (isAuxMissingError(e)) {
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
        ElMessage.success('已复制 IP')
      } catch {
        ElMessage.error('复制失败')
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
      () => {
        if (isIdle()) {
          resetToZero()
          return
        }
        netHistory.value = []
        selectedNetIface.value = ''
        netIfaces.value = []
        sysinfo.value = null
        sysinfoError.value = ''
        loadSystemInfo()
        startTimer()
      },
      { immediate: true },
    )

    watch(sysinfoOpen, (open) => {
      if (open && !isIdle() && !sysinfo.value && !sysinfoLoading.value) {
        loadSystemInfo()
      }
    })
    onMounted(async () => {
      await loadInterval()
      EventsOn('system-settings:changed', onSettingsChanged)
      startTimer()
    })
    onUnmounted(() => {
      stopTimer()
      EventsOff('system-settings:changed')
    })

    return {
      snapshot,
      loading,
      displayError,
      netHistory,
      netIfaces,
      selectedNetIface,
      sysinfoOpen,
      sysinfoLoading,
      sysinfo,
      sysinfoError,
      sysinfoRows,
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
  grid-template-columns: 56px 52px 52px minmax(0, 1fr);
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
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
</style>
