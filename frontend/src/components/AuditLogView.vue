<template>
  <div class="audit-page" v-loading="loading">
    <header class="audit-head">
      <div class="head-left">
        <h1>审计日志</h1>
        <p class="sub">
          每一次 AI 工具调用、审批裁决、危险命令拦截都写入审计。决策类型对齐 Reeve：auto / approved / denied / blocked / cancelled。params 已脱敏。
        </p>
      </div>
      <div class="head-actions">
        <el-button size="small" :loading="loading" @click="reload">
          <el-icon><Refresh /></el-icon>
          全刷
        </el-button>
        <span class="count-pill">{{ total }} 条</span>
        <el-button size="small" @click="settingsOpen = true">
          <el-icon><Setting /></el-icon>
          安全设置
        </el-button>
        <el-button size="small" @click="onExport('csv')">导出 CSV</el-button>
        <el-button size="small" type="danger" plain @click="clearAll">清空</el-button>
      </div>
    </header>

    <section v-if="pending.length" class="approval-panel">
      <div class="approval-head">
        <h2>待审批 <span class="count-chip">{{ pending.length }}</span></h2>
        <div class="approval-actions">
          <el-button size="small" type="primary" plain :disabled="!selectedIds.length" @click="batchDecide(true)">
            选中放行 ({{ selectedIds.length }})
          </el-button>
          <el-button size="small" type="danger" plain :disabled="!selectedIds.length" @click="batchDecide(false)">
            选中拒绝
          </el-button>
          <el-button size="small" type="primary" @click="batchDecideAll(true)">全部放行</el-button>
          <el-button size="small" type="danger" plain @click="batchDecideAll(false)">全部拒绝</el-button>
        </div>
      </div>
      <el-table
        :data="pending"
        size="small"
        row-key="id"
        @selection-change="onSelChange"
      >
        <el-table-column type="selection" width="40" />
        <el-table-column prop="createdAt" label="触发时间" width="160" />
        <el-table-column prop="tool" label="工具" width="130" />
        <el-table-column label="说明" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">{{ row.toolDesc || '—' }}</template>
        </el-table-column>
        <el-table-column prop="server" label="服务器" width="110" show-overflow-tooltip />
        <el-table-column prop="source" label="来源" width="120" show-overflow-tooltip />
        <el-table-column label="摘要" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">{{ row.summary || row.preview }}</template>
        </el-table-column>
        <el-table-column label="剩余" width="80">
          <template #default="{ row }">
            <span :class="{ urgent: row.remainingSecs < 60 }">{{ fmtRemain(row.remainingSecs) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button text size="small" type="primary" @click="singleDecide(row.id, true)">放行</el-button>
            <el-button
              v-if="row.outboundHosts?.length && !row.isDanger"
              text
              size="small"
              type="warning"
              @click="singleDecide(row.id, true, true)"
            >加白放行</el-button>
            <el-button text size="small" type="danger" @click="singleDecide(row.id, false)">拒绝</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <section class="status-row">
      <div
        v-for="c in cards"
        :key="c.key"
        class="stat-card"
        :class="{ active: activeCard === c.key }"
        @click="onCardClick(c)"
      >
        <span class="k">{{ c.label }}</span>
        <span class="v" :class="c.tone">{{ stats[c.key] ?? 0 }}</span>
      </div>
    </section>

    <section class="filters">
      <el-select v-model="filter.tool" placeholder="全部工具" clearable filterable size="small" style="width: 140px" @change="reload">
        <el-option v-for="t in toolOptions" :key="t" :label="t" :value="t" />
      </el-select>
      <el-select v-model="filter.module" placeholder="工具类别" clearable size="small" style="width: 120px" @change="reload">
        <el-option v-for="m in moduleOptions" :key="m.value" :label="m.label" :value="m.value" />
      </el-select>
      <el-select v-model="filter.decision" placeholder="全部决策" clearable size="small" style="width: 120px" @change="onDecisionSelect">
        <el-option label="全部决策" value="" />
        <el-option label="auto 自动放行" value="auto" />
        <el-option label="approved 已批准" value="approved" />
        <el-option label="denied 已拒绝" value="denied" />
        <el-option label="blocked 已拦截" value="blocked" />
        <el-option label="cancelled 已取消" value="cancelled" />
      </el-select>
      <el-select v-model="filter.server" placeholder="服务器" clearable filterable size="small" style="width: 130px" @change="reload">
        <el-option v-for="s in serverOptions" :key="s" :label="s" :value="s" />
      </el-select>
      <el-select v-model="rangePreset" size="small" style="width: 110px" @change="onRangePreset">
        <el-option label="全部时间" value="all" />
        <el-option label="今天" value="today" />
        <el-option label="最近 7 天" value="7d" />
        <el-option label="最近 30 天" value="30d" />
      </el-select>
      <el-date-picker v-model="dateStart" type="date" size="small" placeholder="开始" value-format="YYYY-MM-DD" style="width: 124px" @change="onDateParts" />
      <span class="date-sep">—</span>
      <el-date-picker v-model="dateEnd" type="date" size="small" placeholder="结束" value-format="YYYY-MM-DD" style="width: 124px" @change="onDateParts" />
      <el-input v-model="filter.keyword" clearable size="small" style="width: 220px" placeholder="搜索 tool / params / result / reason..." @keyup.enter="reload" @clear="reload" />
      <el-button size="small" @click="reload">查询</el-button>
      <div class="filters-right">
        <el-checkbox v-model="onlyBlocked" @change="onOnlyBlocked">仅看 blocked/denied</el-checkbox>
      </div>
    </section>

    <p class="hint">过滤与导出不影响策略本身。决策原因见「原因」列；敏感明文在敏感库，审计行仅存 [REDACTED:…] 占位。</p>

    <div class="table-wrap">
      <el-table :data="pageRows" size="small" class="audit-table" empty-text="暂无审计记录" row-key="id">
        <el-table-column type="expand">
          <template #default="{ row }">
            <div class="expand">
              <div><b>参数</b><pre>{{ pretty(row.params) }}</pre></div>
              <div><b>结果</b><pre>{{ pretty(row.result) }}</pre></div>
              <div class="meta-grid">
                <div><span class="mk">原因</span>{{ row.reason || '—' }}</div>
                <div><span class="mk">审批者</span>{{ row.approver || '—' }}</div>
                <div><span class="mk">模块</span>{{ row.module || '—' }}</div>
                <div><span class="mk">ID</span>{{ row.id }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="time" label="时间" width="170" />
        <el-table-column prop="source" label="来源" min-width="140" show-overflow-tooltip />
        <el-table-column label="服务器" width="120" show-overflow-tooltip>
          <template #default="{ row }">{{ row.server || '—' }}</template>
        </el-table-column>
        <el-table-column prop="tool" label="动作" min-width="140" show-overflow-tooltip />
        <el-table-column label="结果" min-width="120" show-overflow-tooltip>
          <template #default="{ row }"><span class="result-text">{{ resultPreview(row) }}</span></template>
        </el-table-column>
        <el-table-column label="决策" width="110" align="center">
          <template #default="{ row }">
            <span class="badge" :class="norm(row.decision)">{{ decisionLabel(row.decision) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="reason" label="原因" min-width="160" show-overflow-tooltip />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button text size="small" @click="copyCmd(row)">复制</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <div class="pager">
      <span class="pager-info">{{ total }} 条 {{ pageFrom }}-{{ pageTo }} / 共 {{ total }}</span>
      <el-pagination v-model:current-page="page" :page-size="pageSize" :total="total" layout="prev, pager, next" background small />
    </div>

    <el-dialog v-model="settingsOpen" title="AI 安全设置" width="560px" append-to-body>
      <div class="settings-body" v-loading="settingsLoading">
        <h4>全局 AI 总开关</h4>
        <el-radio-group v-model="sec.aiMode" size="small">
          <el-radio-button value="normal">按服务器档位</el-radio-button>
          <el-radio-button value="armed">限时放行</el-radio-button>
          <el-radio-button value="emergency">紧急停止</el-radio-button>
        </el-radio-group>
        <div class="row" v-if="sec.aiMode === 'armed'">
          <span>到期时间（RFC3339）</span>
          <el-input v-model="sec.armedUntil" size="small" placeholder="2026-08-28T18:00:00+08:00" />
        </div>
        <el-checkbox v-model="sec.emergencyStop">紧急停止（与 emergency 模式叠加）</el-checkbox>

        <h4>出站白名单</h4>
        <p class="tip">默认启用。命令中的公网地址不在白名单 → 升级审批（不直接拒绝）。裸 IP 永不命中。</p>
        <el-checkbox v-model="sec.outboundAllowlistDisabled">关闭出站白名单检查</el-checkbox>
        <el-input v-model="sec.outboundHostsText" type="textarea" :rows="3" placeholder="每行一个 host，如 mirrors.aliyun.com" />

        <h4>审计保留 / 敏感库 TTL</h4>
        <div class="row">
          <span>审计保留（天，0=永久）</span>
          <el-input-number v-model="sec.auditRetentionDays" :min="0" :max="999" size="small" />
        </div>
        <div class="row">
          <span>敏感库 TTL（天）</span>
          <el-input-number v-model="sec.redactionTTLDays" :min="1" :max="365" size="small" />
        </div>
        <div class="row">
          <span>默认服务器策略</span>
          <el-select v-model="sec.defaultPolicy" size="small" style="width: 160px">
            <el-option label="disabled" value="disabled" />
            <el-option label="readonly" value="readonly" />
            <el-option label="approval" value="approval" />
            <el-option label="allowlist" value="allowlist" />
            <el-option label="trusted" value="trusted" />
          </el-select>
        </div>

        <h4>危险黑名单（内置，只读）</h4>
        <p class="tip">内置规则任何档位（含 trusted）都拦截，不可关闭或修改。命令正则与敏感路径一并列出。</p>
        <el-table :data="builtinDanger" size="small" max-height="220" empty-text="加载中…" class="builtin-danger-table">
          <el-table-column prop="label" label="规则" :min-width="140" show-overflow-tooltip />
          <el-table-column prop="kind" label="类型" width="72">
            <template #default="{ row }">{{ row.kind === 'path' ? '路径' : '命令' }}</template>
          </el-table-column>
          <el-table-column prop="pattern" label="匹配" :min-width="180" show-overflow-tooltip>
            <template #default="{ row }"><code class="mono-cell">{{ row.pattern }}</code></template>
          </el-table-column>
        </el-table>

        <h4>危险黑名单（自定义）</h4>
        <p class="tip">每行一条正则；命中后与内置同级 → blocked。仅可增删自定义项，不影响上方内置规则。</p>
        <el-input v-model="sec.customDangerText" type="textarea" :rows="4" placeholder="^rm\s+-rf\s+/data/" />

        <h4>敏感库（元数据）</h4>
        <el-table :data="sensitive" size="small" max-height="180" empty-text="暂无脱敏捕获">
          <el-table-column prop="id" label="ID" width="120" />
          <el-table-column prop="kind" label="规则/类型" />
          <el-table-column prop="label" label="标签" />
        </el-table>
      </div>
      <template #footer>
        <el-button size="small" @click="purgeNow">立即清理过期审计</el-button>
        <el-button size="small" @click="settingsOpen = false">取消</el-button>
        <el-button size="small" type="primary" @click="saveSettings">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Setting } from '@element-plus/icons-vue'
import {
  GetMCPAuditStats,
  GetMCPAuditMeta,
  QueryMCPAudit,
  ExportMCPAudit,
  ClearMCPAudit,
  GetMCPSettings,
  SaveMCPSettings,
  PurgeMCPAudit,
  ListMCPSensitive,
  ListMCPCustomDangerPatterns,
  ListMCPBuiltinDangerPatterns,
  SaveMCPCustomDangerPatterns,
  ListMCPApprovals,
  DecideMCPApproval,
  DecideMCPApprovalBatch,
} from '../../wailsjs/go/app/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

const FALLBACK_TOOLS = [
  'list_servers', 'ssh_exec', 'sftp_read', 'sftp_write', 'system_info', 'deploy_run',
]

const MODULES = [
  { value: 'ssh', label: 'ssh_' },
  { value: 'sftp', label: 'sftp_' },
  { value: 'web', label: 'web_' },
  { value: 'deploy', label: 'deploy_' },
  { value: 'apps', label: 'install/vault' },
  { value: 'inspect', label: '巡检' },
  { value: 'skills', label: '技能' },
  { value: 'other', label: 'other' },
]

export default {
  name: 'AuditLogView',
  components: { Refresh, Setting },
  setup() {
    const loading = ref(false)
    const settingsLoading = ref(false)
    const stats = ref({})
    const rows = ref([])
    const meta = ref({ tools: [], modules: [], servers: [], sources: [] })
    const sensitive = ref([])
    const builtinDanger = ref([])
    const pending = ref([])
    const selectedIds = ref([])
    let pendingTimer = null
    const filter = reactive({ tool: '', module: '', server: '', decision: '', keyword: '', startTime: '', endTime: '' })
    const dateStart = ref('')
    const dateEnd = ref('')
    const rangePreset = ref('all')
    const onlyBlocked = ref(false)
    const activeCard = ref('total')
    const settingsOpen = ref(false)
    const page = ref(1)
    const pageSize = 50
    const sec = reactive({
      aiMode: 'normal',
      armedUntil: '',
      emergencyStop: false,
      outboundAllowlistDisabled: false,
      outboundHostsText: '',
      auditRetentionDays: 90,
      redactionTTLDays: 30,
      defaultPolicy: 'trusted',
      enabled: false,
      autoStart: false,
      httpPort: 18765,
      bindLan: false,
      customDangerText: '',
    })

    const cards = [
      { key: 'total', label: '总计', tone: '', decision: '' },
      { key: 'auto', label: '自动放行', tone: 'ok', decision: 'auto' },
      { key: 'approved', label: '已批准', tone: 'info', decision: 'approved' },
      { key: 'denied', label: '已拒绝', tone: 'warn', decision: 'denied' },
      { key: 'blocked', label: '已拦截', tone: 'bad', decision: 'blocked' },
      { key: 'cancelled', label: '已取消', tone: 'pink', decision: 'cancelled' },
    ]

    const toolOptions = computed(() => (meta.value.tools?.length ? meta.value.tools : FALLBACK_TOOLS))
    const serverOptions = computed(() => meta.value.servers || [])
    const moduleOptions = computed(() => MODULES)
    const total = computed(() => rows.value.length)
    const pageRows = computed(() => {
      const start = (page.value - 1) * pageSize
      return rows.value.slice(start, start + pageSize)
    })
    const pageFrom = computed(() => (total.value === 0 ? 0 : (page.value - 1) * pageSize + 1))
    const pageTo = computed(() => Math.min(page.value * pageSize, total.value))

    const buildFilter = () => ({ ...filter, limit: 5000 })

    const fmtLocal = (d) => {
      const p = (n) => String(n).padStart(2, '0')
      return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
    }

    const loadPending = async () => {
      try {
        pending.value = (await ListMCPApprovals()) || []
      } catch {
        pending.value = []
      }
    }

    const fmtRemain = (s) => {
      const n = Number(s) || 0
      const m = Math.floor(n / 60)
      const sec = n % 60
      return `${m}:${String(sec).padStart(2, '0')}`
    }

    const onSelChange = (rows) => {
      selectedIds.value = (rows || []).map((r) => r.id)
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

    const singleDecide = async (id, allow, addOutbound = false) => {
      try {
        if (addOutbound) {
          const row = pending.value.find((p) => p.id === id)
          if (row?.outboundHosts?.length) {
            await ElMessageBox.confirm(
              `批准的同时将永久加入出站白名单：\n${row.outboundHosts.join('\n')}\n\n可随时在审计 → 安全设置中移除。`,
              '批准并加白名单',
              { type: 'warning', confirmButtonText: '确认加入', cancelButtonText: '取消' },
            )
          }
        }
        let rejectReason = ''
        if (!allow) {
          rejectReason = await promptRejectReason()
        }
        await DecideMCPApproval(id, allow, !!addOutbound, rejectReason)
        ElMessage.success(allow ? (addOutbound ? '已放行并加白名单' : '已放行') : '已拒绝')
        await loadPending()
        await reload()
      } catch (e) {
        if (e === 'cancel' || e === 'close') return
        ElMessage.error(String(e))
      }
    }

    const batchDecide = async (allow) => {
      if (!selectedIds.value.length) return
      try {
        let rejectReason = ''
        if (!allow) {
          rejectReason = await promptRejectReason()
        }
        await DecideMCPApprovalBatch(selectedIds.value, allow, rejectReason)
        ElMessage.success(allow ? '批量放行完成' : '批量拒绝完成')
        selectedIds.value = []
        await loadPending()
        await reload()
      } catch (e) {
        ElMessage.error(String(e))
      }
    }

    const batchDecideAll = async (allow) => {
      const ids = pending.value.map((p) => p.id)
      if (!ids.length) return
      try {
        let rejectReason = ''
        if (!allow) {
          rejectReason = await promptRejectReason()
        }
        await DecideMCPApprovalBatch(ids, allow, rejectReason)
        ElMessage.success(allow ? '已全部放行' : '已全部拒绝')
        await loadPending()
        await reload()
      } catch (e) {
        ElMessage.error(String(e))
      }
    }

    const reload = async () => {
      loading.value = true
      try {
        const [st, mt, list] = await Promise.all([
          GetMCPAuditStats().catch(() => ({})),
          GetMCPAuditMeta().catch(() => ({})),
          QueryMCPAudit(buildFilter()).catch(() => []),
        ])
        stats.value = st || {}
        meta.value = mt || {}
        rows.value = list || []
        page.value = 1
      } catch (e) {
        ElMessage.error(`加载审计失败: ${e}`)
      } finally {
        loading.value = false
      }
    }

    const loadSettings = async () => {
      settingsLoading.value = true
      try {
        const s = (await GetMCPSettings()) || {}
        sec.aiMode = s.aiMode || 'normal'
        sec.armedUntil = s.armedUntil || ''
        sec.emergencyStop = !!s.emergencyStop
        sec.outboundAllowlistDisabled = !!s.outboundAllowlistDisabled
        sec.outboundHostsText = (s.outboundHosts || []).join('\n')
        sec.auditRetentionDays = s.auditRetentionDays ?? 90
        sec.redactionTTLDays = s.redactionTTLDays || 30
        sec.defaultPolicy = s.defaultPolicy || 'trusted'
        sec.enabled = !!s.enabled
        sec.autoStart = !!s.autoStart
        sec.httpPort = s.httpPort || 18765
        sec.bindLan = !!s.bindLan
        sensitive.value = (await ListMCPSensitive().catch(() => [])) || []
        builtinDanger.value = (await ListMCPBuiltinDangerPatterns().catch(() => [])) || []
        const danger = (await ListMCPCustomDangerPatterns().catch(() => [])) || []
        sec.customDangerText = danger.join('\n')
      } finally {
        settingsLoading.value = false
      }
    }

    watch(settingsOpen, (v) => { if (v) loadSettings() })

    const saveSettings = async () => {
      try {
        const hosts = String(sec.outboundHostsText || '').split(/\n+/).map((x) => x.trim()).filter(Boolean)
        const danger = String(sec.customDangerText || '').split(/\n+/).map((x) => x.trim()).filter(Boolean)
        await SaveMCPCustomDangerPatterns(danger)
        await SaveMCPSettings({
          enabled: sec.enabled,
          autoStart: sec.autoStart,
          httpPort: sec.httpPort,
          bindLan: sec.bindLan,
          defaultPolicy: sec.defaultPolicy,
          aiMode: sec.aiMode,
          armedUntil: sec.armedUntil,
          emergencyStop: sec.emergencyStop,
          auditRetentionDays: sec.auditRetentionDays,
          outboundAllowlistDisabled: sec.outboundAllowlistDisabled,
          outboundHosts: hosts,
          redactionTTLDays: sec.redactionTTLDays,
        })
        ElMessage.success('已保存安全设置')
        settingsOpen.value = false
      } catch (e) {
        ElMessage.error(`保存失败: ${e}`)
      }
    }

    const purgeNow = async () => {
      try {
        const n = await PurgeMCPAudit()
        ElMessage.success(`已清理 ${n || 0} 条过期审计`)
        await reload()
      } catch (e) {
        ElMessage.error(`清理失败: ${e}`)
      }
    }

    const onCardClick = (c) => {
      activeCard.value = c.key
      filter.decision = c.decision || ''
      onlyBlocked.value = false
      reload()
    }
    const onDecisionSelect = () => {
      activeCard.value = filter.decision || 'total'
      reload()
    }
    const onOnlyBlocked = (v) => {
      if (v) {
        filter.decision = 'blocked'
        activeCard.value = 'blocked'
      } else {
        filter.decision = ''
        activeCard.value = 'total'
      }
      reload()
    }
    const onRangePreset = (v) => {
      const now = new Date()
      if (v === 'all') {
        filter.startTime = ''
        filter.endTime = ''
        dateStart.value = ''
        dateEnd.value = ''
      } else if (v === 'today') {
        const day = fmtLocal(now).slice(0, 10)
        dateStart.value = day
        dateEnd.value = day
        filter.startTime = `${day} 00:00:00`
        filter.endTime = `${day} 23:59:59`
      } else {
        const days = v === '7d' ? 7 : 30
        const start = new Date(now.getTime() - days * 86400000)
        dateStart.value = fmtLocal(start).slice(0, 10)
        dateEnd.value = fmtLocal(now).slice(0, 10)
        filter.startTime = fmtLocal(start)
        filter.endTime = fmtLocal(now)
      }
      reload()
    }
    const onDateParts = () => {
      rangePreset.value = 'all'
      filter.startTime = dateStart.value ? `${dateStart.value} 00:00:00` : ''
      filter.endTime = dateEnd.value ? `${dateEnd.value} 23:59:59` : ''
      reload()
    }

    const clearAll = async () => {
      try {
        await ElMessageBox.confirm('清空全部审计？不可恢复。', '清空', { type: 'warning' })
        await ClearMCPAudit()
        ElMessage.success('已清空')
        await reload()
      } catch (e) {
        if (e !== 'cancel') ElMessage.error(`失败: ${e}`)
      }
    }
    const onExport = async (fmt) => {
      try {
        const path = await ExportMCPAudit(fmt, buildFilter())
        if (path) ElMessage.success(`已导出 ${path}`)
      } catch (e) {
        if (e) ElMessage.error(`导出失败: ${e}`)
      }
    }
    const copyCmd = async (row) => {
      const text = row.params || row.result || ''
      try {
        await navigator.clipboard.writeText(text)
        ElMessage.success('已复制（请尽快粘贴）')
      } catch {
        ElMessage.warning('复制失败')
      }
    }

    const norm = (d) => ({
      success: 'auto', auto: 'auto', approved: 'approved', manual: 'approved',
      denied: 'denied', rejected: 'denied', error: 'denied', timeout: 'denied',
      blocked: 'blocked', cancelled: 'cancelled', pending: 'cancelled', approval: 'cancelled',
    }[d] || d)

    const decisionLabel = (d) => ({
      auto: 'auto', success: 'auto', approved: 'approved', manual: 'approved',
      denied: 'denied', blocked: 'blocked', cancelled: 'cancelled',
    }[norm(d)] || d || '—')

    const resultPreview = (row) => {
      const r = String(row.result || '').replace(/\s+/g, ' ').trim()
      if (!r) return '—'
      return r.length <= 40 ? r : `${r.slice(0, 40)}…`
    }
    const pretty = (s) => {
      try { return JSON.stringify(JSON.parse(s), null, 2) } catch { return s || '' }
    }

    let offQueued = null
    let offResolved = null
    onMounted(() => {
      reload()
      loadPending()
      pendingTimer = setInterval(loadPending, 1000)
      // 必须用 disposer，禁止 EventsOff(事件名) 误删 App / 审批弹窗的监听
      offQueued = EventsOn('approval:queued', loadPending)
      offResolved = EventsOn('approval:resolved', () => { loadPending(); reload() })
    })
    onBeforeUnmount(() => {
      if (pendingTimer) clearInterval(pendingTimer)
      offQueued?.()
      offResolved?.()
    })

    return {
      loading, settingsLoading, stats, filter, dateStart, dateEnd, rangePreset, onlyBlocked,
      activeCard, settingsOpen, page, pageSize, cards, sec, sensitive, builtinDanger, pending, selectedIds,
      toolOptions, serverOptions, moduleOptions, total, pageRows, pageFrom, pageTo,
      reload, loadPending, fmtRemain, onSelChange, singleDecide, batchDecide, batchDecideAll,
      onCardClick, onDecisionSelect, onOnlyBlocked, onRangePreset, onDateParts,
      clearAll, onExport, copyCmd, saveSettings, purgeNow, norm, decisionLabel, resultPreview, pretty,
    }
  },
}
</script>

<style scoped>
.audit-page {
  flex: 1 1 auto;
  align-self: stretch;
  width: 100%;
  min-width: 0;
  min-height: 0;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  overflow: auto;
  padding: var(--app-space-page, 28px 32px 24px);
  background: var(--app-bg);
  color: var(--app-text);
}
.audit-head { display: flex; justify-content: space-between; gap: 16px; margin-bottom: 16px; flex-shrink: 0; }
.head-left { min-width: 0; flex: 1; }
h1 { margin: 0; font-size: 18px; font-weight: 650; }
.sub { margin: 6px 0 0; font-size: 12px; line-height: 1.5; color: var(--app-text-muted); max-width: 760px; }
.head-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; flex-wrap: wrap; }
.count-pill {
  display: inline-flex; align-items: center; height: 28px; padding: 0 10px; border-radius: 999px;
  border: 1px solid var(--app-border); background: var(--app-card-bg); font-size: 12px; color: var(--app-text-muted);
}
.approval-panel {
  margin-bottom: 14px;
  padding: 14px 16px;
  border: 1px solid color-mix(in srgb, #eab308 35%, var(--app-border));
  border-radius: 10px;
  background: color-mix(in srgb, #eab308 6%, var(--app-card-bg));
  flex-shrink: 0;
}
.approval-head { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-bottom: 10px; flex-wrap: wrap; }
.approval-head h2 { margin: 0; font-size: 14px; font-weight: 650; display: flex; align-items: center; gap: 8px; }
.count-chip {
  display: inline-flex; align-items: center; justify-content: center; min-width: 22px; height: 22px;
  padding: 0 6px; border-radius: 999px; font-size: 11px; font-weight: 650;
  background: color-mix(in srgb, #eab308 20%, transparent); color: #ca8a04;
}
.approval-actions { display: flex; flex-wrap: wrap; gap: 6px; }
.urgent { color: #ef4444; font-weight: 600; }
.status-row { display: grid; grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 10px; margin-bottom: 14px; flex-shrink: 0; }
.stat-card {
  background: var(--app-card-bg); border: 1px solid var(--app-card-border, var(--app-border));
  border-radius: 10px; padding: 12px 14px; cursor: pointer;
}
.stat-card.active, .stat-card:hover { border-color: color-mix(in srgb, var(--app-accent-color, #22c55e) 45%, var(--app-border)); }
.k { font-size: 12px; color: var(--app-text-muted); }
.v { font-size: 22px; font-weight: 650; display: block; margin-top: 4px; font-variant-numeric: tabular-nums; }
.v.ok { color: #22c55e; } .v.warn { color: #eab308; } .v.info { color: #22d3ee; } .v.pink { color: #f472b6; } .v.bad { color: #ef4444; }
.filters { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; margin-bottom: 8px; flex-shrink: 0; }
.date-sep { color: var(--app-text-muted); font-size: 12px; }
.filters-right { margin-left: auto; }
.hint { margin: 0 0 10px; font-size: 11px; color: var(--app-text-muted); flex-shrink: 0; }
.table-wrap {
  flex: 1 1 auto; min-height: 220px; overflow: auto; border: 1px solid var(--app-border);
  border-radius: 10px; background: var(--app-card-bg);
}
.result-text { font-size: 12px; color: var(--app-text-muted); }
.badge {
  display: inline-flex; align-items: center; height: 20px; padding: 0 8px; border-radius: 4px;
  font-size: 11px; font-weight: 650;
}
.badge.auto { color: #16a34a; background: color-mix(in srgb, #22c55e 16%, transparent); }
.badge.approved { color: #0891b2; background: color-mix(in srgb, #22d3ee 16%, transparent); }
.badge.denied { color: #ca8a04; background: color-mix(in srgb, #eab308 16%, transparent); }
.badge.blocked { color: #ef4444; background: color-mix(in srgb, #ef4444 14%, transparent); }
.badge.cancelled { color: #db2777; background: color-mix(in srgb, #f472b6 16%, transparent); }
.expand { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; padding: 8px 12px 16px; }
.expand pre {
  margin: 6px 0 0; padding: 10px; background: var(--app-inset-bg, var(--app-bg)); border-radius: 8px;
  font-size: 11px; max-height: 240px; overflow: auto; white-space: pre-wrap; word-break: break-all;
}
.meta-grid { grid-column: 1 / -1; display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 8px; font-size: 12px; color: var(--app-text-muted); }
.mk { margin-right: 6px; }
.pager { display: flex; justify-content: flex-end; align-items: center; gap: 12px; margin-top: 12px; flex-shrink: 0; }
.pager-info { font-size: 12px; color: var(--app-text-muted); }
.settings-body h4 { margin: 14px 0 8px; font-size: 13px; }
.settings-body .tip { font-size: 12px; color: var(--app-text-muted); margin: 0 0 8px; }
.settings-body .row { display: flex; align-items: center; gap: 10px; margin: 8px 0; font-size: 12px; }
.builtin-danger-table { margin-bottom: 4px; }
.mono-cell { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; font-size: 11px; }
@media (max-width: 1100px) {
  .status-row { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .expand { grid-template-columns: 1fr; }
  .meta-grid { grid-template-columns: 1fr 1fr; }
}
</style>
