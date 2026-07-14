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
        <!-- <span>{{ activeConnected ? '断开' : '连接' }}</span> -->
      </el-button>
    </div>

    <div v-if="!activeMachine" class="empty">连接机器后显示监控信息</div>
    <template v-else>
      <div class="machine-title">{{ activeMachine }}</div>

      <div class="field">
        <div class="label">IP</div>
        <div class="value-row">
          <span class="mono">{{ snapshot?.host || '-' }}</span>
          <el-button size="small" text :disabled="!snapshot?.host" @click="copyHost">复制</el-button>
        </div>
      </div>

      <div class="field">
        <div class="label">运行时长</div>
        <div class="value">{{ snapshot?.uptimeText || '-' }}</div>
      </div>

      <div class="metric">
        <div class="metric-head">
          <span>CPU</span>
          <span>{{ formatPct(snapshot?.cpuPercent) }}</span>
        </div>
        <el-progress :percentage="clampPct(snapshot?.cpuPercent)" :stroke-width="10" :show-text="false" />
      </div>

      <div class="metric">
        <div class="metric-head">
          <span>内存</span>
          <span>{{ formatPct(snapshot?.memPercent) }} · {{ snapshot?.memUsed || '-' }}/{{ snapshot?.memTotal || '-'
            }}</span>
        </div>
        <el-progress :percentage="clampPct(snapshot?.memPercent)" :stroke-width="10" status="warning"
          :show-text="false" />
      </div>

      <div class="top-block">
        <div class="label">CPU 占用 TOP4</div>
        <div class="top-head">
          <span class="top-pid">PID</span>
          <span class="top-mem">内存</span>
          <span class="top-cpu">CPU</span>
          <span class="top-cmd">命令</span>
        </div>
        <div v-if="!(snapshot?.topMem || []).length" class="empty-sm">暂无数据</div>
        <div v-for="(p, idx) in (snapshot?.topMem || [])" :key="p.pid + idx" class="top-row">
          <div class="top-pid">{{ p.pid }}</div>
          <div class="top-mem">{{ formatPct1(p.mem) }}</div>
          <div class="top-cpu">{{ formatPct1(p.cpu) }}</div>
          <div class="top-cmd" :title="p.command">{{ p.command }}</div>
        </div>
      </div>

      <div v-if="snapshot?.error" class="error">{{ snapshot.error }}</div>
    </template>
  </div>
</template>

<script>
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Connection, SwitchButton } from '@element-plus/icons-vue'
import * as App from '../../../wailsjs/go/app/App'

export default {
  name: 'ShellMonitorPanel',
  components: { Connection, SwitchButton },
  props: {
    activeMachine: { type: String, default: '' },
    activeConnected: { type: Boolean, default: false },
    connecting: { type: Boolean, default: false },
  },
  emits: ['toggle-connection'],
  setup(props) {
    const snapshot = ref(null)
    const loading = ref(false)
    let timer = null

    const clampPct = (v) => {
      const n = Number(v) || 0
      return Math.max(0, Math.min(100, Math.round(n)))
    }
    const formatPct = (v) => `${clampPct(v)}%`
    const formatPct1 = (v) => {
      const n = Number(v)
      if (!Number.isFinite(n)) return '-'
      return `${n.toFixed(1)}%`
    }

    const refresh = async () => {
      if (!props.activeMachine) {
        snapshot.value = null
        return
      }
      loading.value = true
      try {
        snapshot.value = await App.GetShellMonitor(props.activeMachine)
      } catch (e) {
        snapshot.value = {
          machineName: props.activeMachine,
          error: String(e),
          topMem: [],
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

    const startTimer = () => {
      stopTimer()
      if (!props.activeMachine) return
      refresh()
      timer = setInterval(refresh, 4000)
    }
    const stopTimer = () => {
      if (timer) {
        clearInterval(timer)
        timer = null
      }
    }

    watch(() => props.activeMachine, startTimer, { immediate: true })
    onMounted(startTimer)
    onUnmounted(stopTimer)

    return { snapshot, loading, clampPct, formatPct, formatPct1, copyHost }
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

.metric-head {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  margin-bottom: 4px;
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
</style>
