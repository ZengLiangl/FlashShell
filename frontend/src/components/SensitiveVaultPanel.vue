<template>
  <div class="sv-panel">
    <p class="sv-tip">与服务凭据库分离。「明文」会复制到剪贴板，约 10 秒后自动清空；假阳性会清空明文并记入误报样本。</p>
    <el-table
      ref="tableRef"
      :data="filtered"
      size="small"
      max-height="420"
      empty-text="暂无脱敏捕获"
      row-key="id"
      :row-class-name="rowClass"
    >
      <el-table-column prop="id" label="ID" width="110" />
      <el-table-column prop="rule" label="规则" min-width="130" show-overflow-tooltip />
      <el-table-column prop="kind" label="类型" width="100" show-overflow-tooltip />
      <el-table-column prop="server" label="服务器" width="100" show-overflow-tooltip>
        <template #default="{ row }">{{ row.server || '—' }}</template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="80">
        <template #default="{ row }">{{ statusLabel(row.status) }}</template>
      </el-table-column>
      <el-table-column prop="auditId" label="审计" width="120" show-overflow-tooltip>
        <template #default="{ row }">
          <el-button v-if="row.auditId" text size="small" type="primary" @click="$emit('jump-audit', row.auditId)">
            {{ row.auditId }}
          </el-button>
          <span v-else>—</span>
        </template>
      </el-table-column>
      <el-table-column prop="expiresAt" label="到期" width="160" show-overflow-tooltip />
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <div class="sv-ops">
            <el-button text size="small" type="primary" :disabled="!row.hasValue" @click="onReveal(row)">明文</el-button>
            <el-button text size="small" :disabled="!row.hasValue" @click="onPromote(row)">转凭据</el-button>
            <el-button text size="small" type="danger" :disabled="row.status === 'discarded'" @click="onDiscard(row)">假阳性</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="settingsOpen" title="敏感库设置" width="520px" append-to-body>
      <div class="sv-settings" v-loading="settingsLoading">
        <h4>保留策略</h4>
        <p class="sv-tip">到期后清空加密明文，保留 hash 与条目状态为「已过期」。</p>
        <div class="row">
          <span>敏感库 TTL（天）</span>
          <el-input-number v-model="ttlDays" :min="1" :max="365" size="small" />
        </div>
        <div class="row actions">
          <el-button size="small" :loading="busy" @click="purgeExpired">立即清理过期明文</el-button>
        </div>

        <h4>误报样本</h4>
        <p class="sv-tip">标假阳性后写入本地样本（无明文），供后续优化规则参考。</p>
        <el-table :data="falsePositives" size="small" max-height="180" empty-text="暂无误报样本">
          <el-table-column prop="id" label="ID" width="110" />
          <el-table-column prop="rule" label="规则" min-width="120" show-overflow-tooltip />
          <el-table-column prop="createdAt" label="时间" width="160" show-overflow-tooltip />
        </el-table>
      </div>
      <template #footer>
        <el-button size="small" @click="settingsOpen = false">取消</el-button>
        <el-button size="small" type="primary" :loading="busy" @click="saveSettings">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="rulesOpen" title="脱敏规则清单" width="720px" append-to-body>
      <div class="sv-settings" v-loading="rulesLoading">
        <p class="sv-tip">内置规则不可修改；自定义规则排在表末，可增删改。保存后写入 redaction.yaml 并立即生效。</p>
        <el-table :data="ruleRows" size="small" max-height="420" empty-text="加载中…">
          <el-table-column prop="name" label="规则" :min-width="120" show-overflow-tooltip />
          <el-table-column prop="kind" label="类型" width="110" show-overflow-tooltip />
          <el-table-column label="来源" width="72">
            <template #default="{ row }">{{ row.builtin ? '内置' : '自定义' }}</template>
          </el-table-column>
          <el-table-column prop="pattern" label="匹配" :min-width="160" show-overflow-tooltip>
            <template #default="{ row }"><code class="mono">{{ row.pattern }}</code></template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <template v-if="!row.builtin">
                <el-button text size="small" @click="editCustomRule(row)">改</el-button>
                <el-button text size="small" type="danger" @click="removeCustomRule(row)">删</el-button>
              </template>
              <span v-else class="muted">—</span>
            </template>
          </el-table-column>
        </el-table>
        <el-button size="small" style="margin-top: 8px" @click="addCustomRule">添加自定义规则</el-button>
      </div>
      <template #footer>
        <el-button size="small" @click="rulesOpen = false">取消</el-button>
        <el-button size="small" type="primary" :loading="busy" @click="saveRules">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="testerOpen" title="脱敏规则测试器" width="640px" append-to-body>
      <p class="sv-tip">粘贴一段文本，实时查看命中规则（不入库）。</p>
      <el-input v-model="testerText" type="textarea" :rows="6" placeholder="password=demo&#10;Bearer eyJhbGci..." />
      <el-table :data="testerHits" size="small" max-height="240" empty-text="无命中" style="margin-top: 10px">
        <el-table-column prop="rule" label="规则" width="160" />
        <el-table-column prop="kind" label="类型" width="110" />
        <el-table-column prop="snippet" label="片段" min-width="180" show-overflow-tooltip />
        <el-table-column prop="secret" label="捕获" width="120" />
      </el-table>
      <template #footer>
        <el-button size="small" @click="runTester">测试</el-button>
        <el-button size="small" type="primary" @click="testerOpen = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="ruleEditOpen" :title="ruleEdit.index >= 0 ? '编辑自定义规则' : '添加自定义规则'" width="480px" append-to-body>
      <div class="promote-form">
        <div class="row"><span>名称</span><el-input v-model="ruleEdit.name" size="small" placeholder="company_token" /></div>
        <div class="row"><span>类型</span>
          <el-select v-model="ruleEdit.kind" size="small" style="width: 100%">
            <el-option label="credential" value="credential" />
            <el-option label="token" value="token" />
            <el-option label="private_key" value="private_key" />
            <el-option label="url_with_auth" value="url_with_auth" />
            <el-option label="generic" value="generic" />
          </el-select>
        </div>
        <div class="row"><span>正则</span><el-input v-model="ruleEdit.pattern" type="textarea" :rows="3" size="small" placeholder="CMP-(?P&lt;secret&gt;[A-Z0-9]{32})" /></div>
      </div>
      <template #footer>
        <el-button size="small" @click="ruleEditOpen = false">取消</el-button>
        <el-button size="small" type="primary" @click="confirmRuleEdit">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="promoteOpen" title="转为服务凭据" width="420px" append-to-body>
      <div class="promote-form">
        <div class="row"><span>敏感 ID</span><el-input v-model="promote.id" disabled size="small" /></div>
        <div class="row"><span>服务器别名</span><el-input v-model="promote.server" size="small" placeholder="list_servers 的 alias" /></div>
        <div class="row"><span>kind</span><el-input v-model="promote.kind" size="small" placeholder="credential / mysql_conn …" /></div>
        <div class="row"><span>label</span><el-input v-model="promote.label" size="small" /></div>
        <div class="row"><span>字段名</span><el-input v-model="promote.field" size="small" placeholder="password / token" /></div>
      </div>
      <template #footer>
        <el-button size="small" @click="promoteOpen = false">取消</el-button>
        <el-button size="small" type="primary" :loading="busy" @click="doPromote">确认转入</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ListMCPSensitive,
  RevealMCPSensitive,
  DiscardMCPSensitive,
  PromoteMCPSensitive,
  ListMCPRedactRules,
  SaveMCPRedactRules,
  TestMCPRedactRules,
  ReloadMCPRedactRules,
  PurgeMCPSensitive,
  ListMCPFalsePositives,
  GetMCPSettings,
  SaveMCPSettings,
} from '../../wailsjs/go/app/App'

export default {
  name: 'SensitiveVaultPanel',
  props: {
    highlightId: { type: String, default: '' },
  },
  emits: ['jump-audit', 'meta'],
  setup(props, { emit }) {
    const loading = ref(false)
    const busy = ref(false)
    const settingsLoading = ref(false)
    const rows = ref([])
    const rules = ref([])
    const customRules = ref([])
    const falsePositives = ref([])
    const tableRef = ref(null)
    const testerOpen = ref(false)
    const settingsOpen = ref(false)
    const rulesOpen = ref(false)
    const rulesLoading = ref(false)
    const promoteOpen = ref(false)
    const ruleEditOpen = ref(false)
    const testerText = ref('')
    const testerHits = ref([])
    const ttlDays = ref(30)
    const settingsSnapshot = ref(null)
    const promote = reactive({ id: '', server: '', kind: '', label: '', field: 'password' })
    const ruleEdit = reactive({ index: -1, name: '', kind: 'generic', pattern: '' })
    let clearTimer = null

    const filtered = computed(() => rows.value)
    const ruleRows = computed(() => {
      const builtin = (rules.value || []).filter((r) => r.builtin).map((r) => ({
        ...r,
        builtin: true,
      }))
      const custom = (customRules.value || []).map((r, i) => ({
        name: r.name,
        kind: r.kind || 'generic',
        pattern: r.pattern,
        builtin: false,
        index: i,
      }))
      return [...builtin, ...custom]
    })

    const statusLabel = (st) => {
      if (st === 'expired') return '已过期'
      if (st === 'discarded') return '已丢弃'
      return '有效'
    }

    const rowClass = ({ row }) => (props.highlightId && row.id === props.highlightId ? 'sv-hl' : '')

    const emitMeta = () => {
      emit('meta', { count: rows.value.length, loading: loading.value })
    }

    const syncCustomFromRules = (list) => {
      customRules.value = (list || [])
        .filter((r) => !r.builtin)
        .map((r) => ({
          name: r.name,
          kind: r.kind || 'generic',
          pattern: r.pattern,
        }))
    }

    const reload = async () => {
      loading.value = true
      emitMeta()
      try {
        rows.value = (await ListMCPSensitive().catch(() => [])) || []
        rules.value = (await ListMCPRedactRules().catch(() => [])) || []
        syncCustomFromRules(rules.value)
      } finally {
        loading.value = false
        emitMeta()
      }
    }

    const openSettings = async () => {
      settingsOpen.value = true
      settingsLoading.value = true
      try {
        const s = (await GetMCPSettings().catch(() => ({}))) || {}
        settingsSnapshot.value = s
        ttlDays.value = s.redactionTTLDays || 30
        falsePositives.value = (await ListMCPFalsePositives().catch(() => [])) || []
      } finally {
        settingsLoading.value = false
      }
    }

    const openRules = async () => {
      rulesOpen.value = true
      rulesLoading.value = true
      try {
        rules.value = (await ListMCPRedactRules().catch(() => [])) || []
        syncCustomFromRules(rules.value)
      } finally {
        rulesLoading.value = false
      }
    }

    const saveSettings = async () => {
      busy.value = true
      try {
        const cur = settingsSnapshot.value || (await GetMCPSettings().catch(() => ({}))) || {}
        await SaveMCPSettings({
          enabled: !!cur.enabled,
          autoStart: !!cur.autoStart,
          httpPort: cur.httpPort || 18765,
          bindLan: !!cur.bindLan,
          defaultPolicy: cur.defaultPolicy || 'trusted',
          aiMode: cur.aiMode || 'normal',
          armedUntil: cur.armedUntil || '',
          emergencyStop: !!cur.emergencyStop,
          auditRetentionDays: cur.auditRetentionDays ?? 90,
          outboundAllowlistDisabled: !!cur.outboundAllowlistDisabled,
          outboundHosts: cur.outboundHosts || [],
          redactionTTLDays: ttlDays.value || 30,
          customDangerPatterns: cur.customDangerPatterns || [],
        })
        ElMessage.success('已保存敏感库设置')
        settingsOpen.value = false
      } catch (e) {
        ElMessage.error(`保存失败: ${e}`)
      } finally {
        busy.value = false
      }
    }

    const saveRules = async () => {
      busy.value = true
      try {
        await SaveMCPRedactRules(customRules.value.map((r) => ({
          name: r.name,
          kind: r.kind || 'generic',
          pattern: r.pattern,
        })))
        rules.value = (await ListMCPRedactRules().catch(() => [])) || []
        syncCustomFromRules(rules.value)
        ElMessage.success('已保存脱敏规则')
        rulesOpen.value = false
      } catch (e) {
        ElMessage.error(`保存失败: ${e}`)
      } finally {
        busy.value = false
      }
    }

    const addCustomRule = () => {
      ruleEdit.index = -1
      ruleEdit.name = ''
      ruleEdit.kind = 'generic'
      ruleEdit.pattern = ''
      ruleEditOpen.value = true
    }

    const editCustomRule = (row) => {
      ruleEdit.index = row.index
      ruleEdit.name = row.name
      ruleEdit.kind = row.kind || 'generic'
      ruleEdit.pattern = row.pattern
      ruleEditOpen.value = true
    }

    const removeCustomRule = (row) => {
      const i = row.index
      if (i == null || i < 0) return
      customRules.value = customRules.value.filter((_, idx) => idx !== i)
    }

    const confirmRuleEdit = () => {
      const name = String(ruleEdit.name || '').trim()
      const pattern = String(ruleEdit.pattern || '').trim()
      const kind = String(ruleEdit.kind || 'generic').trim() || 'generic'
      if (!name || !pattern) {
        ElMessage.warning('名称与正则不能为空')
        return
      }
      try {
        new RegExp(pattern)
      } catch {
        ElMessage.error('无效正则')
        return
      }
      const builtinNames = new Set((rules.value || []).filter((r) => r.builtin).map((r) => String(r.name).toLowerCase()))
      if (builtinNames.has(name.toLowerCase())) {
        ElMessage.error('不能使用内置规则名')
        return
      }
      const dup = customRules.value.some((r, i) => i !== ruleEdit.index && String(r.name).toLowerCase() === name.toLowerCase())
      if (dup) {
        ElMessage.error('自定义规则名已存在')
        return
      }
      const item = { name, kind, pattern }
      if (ruleEdit.index >= 0) {
        const next = [...customRules.value]
        next[ruleEdit.index] = item
        customRules.value = next
      } else {
        customRules.value = [...customRules.value, item]
      }
      ruleEditOpen.value = false
    }

    const purgeExpired = async () => {
      busy.value = true
      try {
        const n = await PurgeMCPSensitive()
        ElMessage.success(`已清理 ${n || 0} 条过期敏感明文`)
        await reload()
        falsePositives.value = (await ListMCPFalsePositives().catch(() => [])) || []
      } catch (e) {
        ElMessage.error(`清理失败: ${e}`)
      } finally {
        busy.value = false
      }
    }

    const reloadRules = async () => {
      try {
        await ReloadMCPRedactRules()
        rules.value = (await ListMCPRedactRules().catch(() => [])) || []
        syncCustomFromRules(rules.value)
        ElMessage.success(`已重载规则（${rules.value.length} 条）`)
      } catch (e) {
        ElMessage.error(`重载失败: ${e}`)
      }
    }

    const openTester = () => {
      testerOpen.value = true
      testerHits.value = []
    }

    const runTester = async () => {
      try {
        testerHits.value = (await TestMCPRedactRules(testerText.value || '')) || []
        ElMessage.success(`命中 ${testerHits.value.length} 处`)
      } catch (e) {
        ElMessage.error(`测试失败: ${e}`)
      }
    }

    const clearClipboardSoon = () => {
      if (clearTimer) clearTimeout(clearTimer)
      clearTimer = setTimeout(async () => {
        try {
          await navigator.clipboard.writeText('')
          ElMessage.info('剪贴板已自动清空')
        } catch {
          /* ignore */
        }
      }, 10000)
    }

    const onReveal = async (row) => {
      try {
        const plain = await RevealMCPSensitive(row.id)
        await navigator.clipboard.writeText(plain || '')
        ElMessage.success('已复制到剪贴板，约 10 秒后自动清空')
        clearClipboardSoon()
      } catch (e) {
        ElMessage.error(`查看明文失败: ${e}`)
      }
    }

    const onDiscard = async (row) => {
      try {
        await ElMessageBox.confirm(`将「${row.id}」标为假阳性并清空明文？`, '确认', { type: 'warning' })
        await DiscardMCPSensitive(row.id)
        ElMessage.success('已标为假阳性')
        await reload()
      } catch (e) {
        if (e !== 'cancel') ElMessage.error(`操作失败: ${e}`)
      }
    }

    const onPromote = (row) => {
      promote.id = row.id
      promote.server = row.server || ''
      promote.kind = row.kind || 'credential'
      promote.label = row.label || row.rule || row.id
      promote.field = String(row.kind || '').includes('token') ? 'token' : 'password'
      promoteOpen.value = true
    }

    const doPromote = async () => {
      busy.value = true
      try {
        const saved = await PromoteMCPSensitive({ ...promote })
        ElMessage.success(`已转入服务凭据 ${saved?.id || ''}`)
        promoteOpen.value = false
      } catch (e) {
        ElMessage.error(`转凭据失败: ${e}`)
      } finally {
        busy.value = false
      }
    }

    watch(() => props.highlightId, (id) => {
      if (!id) return
      reload()
    })

    onBeforeUnmount(() => {
      if (clearTimer) clearTimeout(clearTimer)
    })

    reload()

    return {
      loading, busy, settingsLoading, rulesLoading, rows, rules, customRules, ruleRows, falsePositives, filtered, tableRef,
      testerOpen, settingsOpen, rulesOpen, promoteOpen, ruleEditOpen, testerText, testerHits, ttlDays, promote, ruleEdit,
      statusLabel, rowClass, reload, reloadRules, openTester, openSettings, openRules, runTester,
      saveSettings, saveRules, purgeExpired, addCustomRule, editCustomRule, removeCustomRule, confirmRuleEdit,
      onReveal, onDiscard, onPromote, doPromote,
    }
  },
}
</script>

<style scoped>
.sv-panel { display: flex; flex-direction: column; gap: 8px; min-height: 0; flex: 1 1 auto; }
.sv-tip {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.45;
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
}
.sv-settings h4 { margin: 12px 0 8px; font-size: 13px; }
.sv-settings .row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 8px 0;
  font-size: 12px;
}
.sv-settings .row.actions { margin-top: 4px; }
.muted { color: var(--el-text-color-secondary); font-size: 12px; }
.sv-ops {
  display: inline-flex;
  align-items: center;
  flex-wrap: nowrap;
  white-space: nowrap;
  gap: 0;
}
.sv-ops :deep(.el-button) {
  margin: 0;
  padding-left: 6px;
  padding-right: 6px;
}
.promote-form { display: flex; flex-direction: column; gap: 10px; }
.promote-form .row {
  display: grid;
  grid-template-columns: 88px 1fr;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}
:deep(.sv-hl) > td {
  background: color-mix(in srgb, var(--el-color-primary) 12%, transparent) !important;
}
</style>
