<template>
  <el-dialog
    v-model="visible"
    :title="dialogTitle"
    width="640px"
    append-to-body
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    @closed="onClosed"
  >
    <div v-if="item" class="approval-body">
      <div v-if="remainingSecs >= 0" class="countdown" :class="{ urgent: remainingSecs < 60 }">
        剩余 {{ fmtRemain(remainingSecs) }} · 5 分钟超时自动拒绝
      </div>

      <div class="meta-grid">
        <div><span class="mk">时间</span>{{ item.createdAt }}</div>
        <div><span class="mk">工具</span><code>{{ item.tool }}</code></div>
        <div><span class="mk">服务器</span><code>{{ item.server || '—' }}</code></div>
        <div><span class="mk">来源</span>{{ item.source }}</div>
      </div>

      <div v-if="item.toolDesc" class="block">
        <div class="block-title">工具说明</div>
        <p class="tool-desc">{{ item.toolDesc }}</p>
      </div>

      <div v-if="item.reason" class="block">
        <div class="block-title">触发原因</div>
        <p class="reason">{{ item.reason }}</p>
      </div>

      <div v-if="item.outboundHosts?.length" class="block">
        <div class="block-title">违规出站端点</div>
        <ul class="hosts">
          <li v-for="h in item.outboundHosts" :key="h">{{ h }}</li>
        </ul>
      </div>

      <div class="block">
        <div class="block-title">命令摘要</div>
        <pre class="cmd" v-html="highlightedPreview"></pre>
      </div>

      <el-collapse v-if="item.paramsJson || item.paramsJSON">
        <el-collapse-item title="完整参数 JSON" name="params">
          <pre class="json">{{ prettyJson(item.paramsJson || item.paramsJSON) }}</pre>
        </el-collapse-item>
      </el-collapse>

      <div v-if="contextRows.length" class="block">
        <div class="block-title">该服务器最近审计</div>
        <ul class="ctx-list">
          <li v-for="r in contextRows" :key="r.id">
            <span class="ctx-time">{{ r.time }}</span>
            <span class="ctx-tool">{{ r.tool }}</span>
            <span class="badge" :class="r.decision">{{ r.decision }}</span>
          </li>
        </ul>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button v-if="queueLen >= 2" @click="skip">跳过</el-button>
        <el-tooltip content="需在系统设置中配置 API Profile 后才可用" placement="top">
          <el-button disabled>AI 解释（未配置模型）</el-button>
        </el-tooltip>
        <el-button type="danger" plain @click="decide(false)">拒绝</el-button>
        <el-button
          v-if="item?.outboundHosts?.length && !item?.isDanger"
          type="warning"
          @click="decide(true, true)"
        >批准并加白名单</el-button>
        <el-button type="primary" @click="decide(true, false)">放行</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  EventsOn,
  WindowShow,
  WindowUnminimise,
  WindowSetAlwaysOnTop,
} from '../../wailsjs/runtime/runtime'
import { DecideMCPApproval, GetMCPApprovalContext } from '../../wailsjs/go/app/App'

function escapeHtml(s) {
  return String(s || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

export default {
  name: 'McpApprovalDialog',
  setup() {
    const visible = ref(false)
    const item = ref(null)
    const queue = []
    const seenIds = new Set()
    const contextRows = ref([])
    const remainingSecs = ref(0)
    let tickTimer = null
    let offQueued = null
    let offQueuedCompat = null
    let offResolved = null
    let pinnedTop = false

    const queueLen = computed(() => queue.length + (visible.value ? 1 : 0))

    const dialogTitle = computed(() => {
      const n = queueLen.value
      return n > 1 ? `MCP 审批（队列 ${n} 条）` : 'MCP 审批'
    })

    const highlightedPreview = computed(() => {
      let html = escapeHtml(item.value?.preview || item.value?.summary || '')
      for (const h of item.value?.outboundHosts || []) {
        if (!h) continue
        const re = new RegExp(h.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi')
        html = html.replace(re, (m) => `<mark class="bad-host">${m}</mark>`)
      }
      return html
    })

    const syncRemain = () => {
      if (!item.value?.expiresAt) {
        remainingSecs.value = item.value?.remainingSecs ?? 0
        return
      }
      const t = new Date(item.value.expiresAt).getTime()
      remainingSecs.value = Math.max(0, Math.floor((t - Date.now()) / 1000))
    }

    const startTick = () => {
      stopTick()
      syncRemain()
      tickTimer = setInterval(syncRemain, 1000)
    }
    const stopTick = () => {
      if (tickTimer) {
        clearInterval(tickTimer)
        tickTimer = null
      }
    }

    const bringToFront = (danger) => {
      try {
        WindowUnminimise()
        WindowShow()
        if (danger) {
          WindowSetAlwaysOnTop(true)
          pinnedTop = true
        }
      } catch {
        // 非 Wails 环境忽略
      }
    }

    const clearPin = () => {
      if (!pinnedTop) return
      try {
        WindowSetAlwaysOnTop(false)
      } catch {
        // ignore
      }
      pinnedTop = false
    }

    const loadContext = async (server) => {
      if (!server) {
        contextRows.value = []
        return
      }
      try {
        contextRows.value = (await GetMCPApprovalContext(server)) || []
      } catch {
        contextRows.value = []
      }
    }

    const showNext = async () => {
      if (visible.value || !queue.length) return
      item.value = queue.shift()
      visible.value = true
      bringToFront(!!item.value?.isDanger)
      await loadContext(item.value?.server)
      startTick()
    }

    const purgeQueueId = (id) => {
      if (!id) return
      seenIds.delete(id)
      for (let i = queue.length - 1; i >= 0; i--) {
        if (queue[i]?.id === id) queue.splice(i, 1)
      }
      if (item.value?.id === id) {
        visible.value = false
        item.value = null
        stopTick()
        clearPin()
        showNext()
      }
    }

    const onQueued = (payload) => {
      // Wails 事件参数可能是 (data) 或额外包装
      const p = payload && typeof payload === 'object' ? payload : null
      if (!p?.id) return
      if (seenIds.has(p.id)) return
      seenIds.add(p.id)
      queue.push(p)
      // 需要人工审核时一律弹出，避免静默进队后超时被拒
      showNext()
    }

    const onResolved = (payload) => {
      const id = payload?.id
      if (id) purgeQueueId(id)
    }

    const fmtRemain = (s) => {
      const m = Math.floor((Number(s) || 0) / 60)
      const sec = (Number(s) || 0) % 60
      return `${m}:${String(sec).padStart(2, '0')}`
    }

    const prettyJson = (s) => {
      try {
        return JSON.stringify(JSON.parse(s), null, 2)
      } catch {
        return s || ''
      }
    }

    const skip = () => {
      if (item.value) queue.push(item.value)
      visible.value = false
      item.value = null
      stopTick()
      clearPin()
      showNext()
    }

    const promptRejectReason = async () => {
      const { value } = await ElMessageBox.prompt(
        '可选：填写拒绝原因，将通过 MCP 返回给调用方',
        '拒绝审批',
        {
          confirmButtonText: '确认拒绝',
          cancelButtonText: '取消',
          inputPlaceholder: '例如：该命令可能影响生产服务',
          inputType: 'textarea',
        },
      )
      return String(value || '').trim()
    }

    const decide = async (allow, addOutbound = false) => {
      try {
        if (addOutbound && item.value?.outboundHosts?.length) {
          await ElMessageBox.confirm(
            `批准的同时将永久加入出站白名单：\n${item.value.outboundHosts.join('\n')}\n\n可随时在审计 → 安全设置中移除。`,
            '批准并加白名单',
            { type: 'warning', confirmButtonText: '确认加入', cancelButtonText: '取消' },
          )
        }
        let rejectReason = ''
        if (!allow) {
          rejectReason = await promptRejectReason()
        }
        if (item.value?.id) await DecideMCPApproval(item.value.id, allow, !!addOutbound, rejectReason)
        ElMessage.success(allow ? (addOutbound ? '已放行并加白名单' : '已放行') : '已拒绝')
      } catch (e) {
        if (e === 'cancel' || e === 'close') return
        const msg = String(e)
        if (msg.includes('notfound') || msg.includes('没有这条')) {
          visible.value = false
          item.value = null
          stopTick()
          clearPin()
          showNext()
          return
        }
        ElMessage.error(`审批失败: ${e}`)
        return
      }
      if (item.value?.id) seenIds.delete(item.value.id)
      visible.value = false
      item.value = null
      stopTick()
      clearPin()
      showNext()
    }

    const onClosed = () => {
      stopTick()
      clearPin()
    }

    watch(visible, (v) => {
      if (!v) {
        stopTick()
        clearPin()
      }
    })

    onMounted(() => {
      // 用 EventsOn 返回的 disposer，禁止 EventsOff(事件名) 误删其它组件监听
      offQueued = EventsOn('approval:queued', onQueued)
      offQueuedCompat = EventsOn('mcp:approval', onQueued)
      offResolved = EventsOn('approval:resolved', onResolved)
    })
    onBeforeUnmount(() => {
      offQueued?.()
      offQueuedCompat?.()
      offResolved?.()
      stopTick()
      clearPin()
    })

    return {
      visible,
      item,
      contextRows,
      remainingSecs,
      queueLen,
      dialogTitle,
      highlightedPreview,
      fmtRemain,
      prettyJson,
      skip,
      decide,
      onClosed,
    }
  },
}
</script>

<style scoped>
.approval-body { font-size: 13px; }
.countdown {
  margin-bottom: 12px;
  padding: 8px 12px;
  border-radius: 8px;
  background: color-mix(in srgb, #eab308 12%, var(--app-panel-bg));
  font-size: 12px;
  color: #ca8a04;
}
.countdown.urgent {
  background: color-mix(in srgb, #ef4444 12%, var(--app-panel-bg));
  color: #ef4444;
}
.meta-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px 16px;
  margin-bottom: 12px;
}
.mk { color: var(--app-text-muted); margin-right: 8px; }
.block { margin-bottom: 12px; }
.block-title { font-size: 12px; font-weight: 600; margin-bottom: 6px; color: var(--app-text-muted); }
.reason { margin: 0; color: var(--el-color-warning); line-height: 1.45; }
.tool-desc { margin: 0; color: var(--app-text); line-height: 1.5; font-size: 12px; }
.hosts { margin: 0; padding-left: 18px; color: var(--el-color-danger); font-family: ui-monospace, monospace; font-size: 12px; }
.cmd, .json {
  margin: 0;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid var(--app-border);
  background: var(--app-panel-bg);
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 12px;
  max-height: 200px;
  overflow: auto;
}
.ctx-list { list-style: none; margin: 0; padding: 0; font-size: 12px; }
.ctx-list li { display: flex; gap: 8px; align-items: center; padding: 4px 0; border-bottom: 1px solid var(--app-border); }
.ctx-time { color: var(--app-text-muted); width: 140px; flex-shrink: 0; }
.ctx-tool { flex: 1; font-family: ui-monospace, monospace; }
.badge { font-size: 10px; padding: 1px 6px; border-radius: 999px; background: var(--app-bg); }
.badge.approved { color: #16a34a; }
.badge.denied, .badge.blocked { color: #ef4444; }
.badge.auto { color: var(--app-text-muted); }
.dialog-footer { display: flex; justify-content: flex-end; gap: 8px; flex-wrap: wrap; }
:deep(mark.bad-host) {
  color: var(--el-color-danger);
  background: transparent;
  text-decoration: underline wavy var(--el-color-danger);
  font-weight: 600;
}
</style>
