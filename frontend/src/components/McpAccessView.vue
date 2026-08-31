<template>
  <div class="mcp-hub" :class="{ embedded }" v-loading="loading">
    <header class="hub-head">
      <div class="hub-title-wrap">
        <h1 v-if="!embedded">MCP 接入中心</h1>
        <h2 v-else class="hub-embedded-title">MCP 接入</h2>
        <p class="hub-sub" :class="{ 'hub-sub--solo': embedded }">
          <template v-if="embedded">
            一键签发 Scoped Token 并写入客户端配置；通道仅绑 <code>127.0.0.1</code>（或私网 CIDR）。
          </template>
          <template v-else>
            通过 MCP 把 FlashShell 工具暴露给 AI 客户端。一键签发 Scoped Token + 写入客户端配置；通道仅绑
            <code>127.0.0.1</code>（或私网 CIDR），永不监听公网。
          </template>
        </p>
      </div>
      <div class="hub-head-right">
        <div class="stat-group">
          <span class="stat-num">{{ status.toolCount || 35 }}</span>
          <span class="stat-lab">工具</span>
          <span class="stat-sep">/</span>
          <span class="stat-num">{{ status.serverCount ?? 0 }}</span>
          <span class="stat-lab">服务器</span>
        </div>
        <el-button size="small" @click="bindDialog = true">绑定状况</el-button>
      </div>
    </header>

    <section class="hub-section mcp-settings-section">
      <div class="section-label tight">接入设置</div>
      <div class="mcp-settings-grid">
        <div class="mcp-setting-item">
          <div class="mcp-setting-main">
            <span class="mcp-setting-name">自动开启</span>
            <el-switch
              v-model="mcpSettings.autoStart"
              size="small"
              :loading="settingsSaving === 'autoStart'"
              @change="onAutoStartChange"
            />
          </div>
          <p class="mcp-setting-desc">应用启动时自动拉起 MCP 服务（默认关闭）</p>
        </div>
        <div class="mcp-setting-item">
          <div class="mcp-setting-main">
            <span class="mcp-setting-name">MCP 服务</span>
            <el-switch
              v-model="mcpSettings.enabled"
              size="small"
              :loading="settingsSaving === 'enabled'"
              @change="onEnabledChange"
            />
          </div>
          <p class="mcp-setting-desc">仅本次运行生效；下次启动是否自动开取决于「自动开启」</p>
        </div>
      </div>
    </section>

    <section class="hub-section">
      <div class="section-label">
        <span class="check" :class="{ on: status.online }">
          <el-icon v-if="status.online"><CircleCheckFilled /></el-icon>
          <el-icon v-else><WarningFilled /></el-icon>
        </span>
        <span>三个通道 · 本地 {{ status.localAddr || '127.0.0.1' }}</span>
        <el-tooltip content="刷新状态" placement="top">
          <button type="button" class="icon-btn ghost" :disabled="loading" @click="reload">
            <el-icon><Refresh /></el-icon>
          </button>
        </el-tooltip>
      </div>
      <div class="svc-grid">
        <div class="svc-card">
          <div class="svc-title-row">
            <span class="svc-name">stdio sidecar</span>
            <span class="tag host">托管</span>
          </div>
          <div class="svc-meta">Claude Code / Codex / OpenCode · <code>--mcp-stdio</code></div>
          <div class="svc-actions">
            <el-button size="small" @click="copy(status.stdioPath || '')">复制路径</el-button>
          </div>
          <div class="svc-foot mono">{{ displayPath(status.stdioPath) || 'FlashShell --mcp-stdio' }}</div>
        </div>
        <div class="svc-card">
          <div class="svc-title-row">
            <span class="svc-name">Streamable HTTP</span>
            <span class="tag" :class="status.online ? 'online' : 'off'">{{ status.online ? 'ONLINE' : 'OFFLINE' }}</span>
          </div>
          <div class="svc-meta mono">{{ status.httpUrl || '—' }}</div>
          <div class="svc-actions">
            <el-button size="small" :disabled="!status.httpUrl" @click="copy(status.httpUrl)">复制 URL</el-button>
          </div>
        </div>
        <div class="svc-card">
          <div class="svc-title-row">
            <span class="svc-name">观察者 / 审计</span>
            <span class="tag" :class="status.observerOnline ? 'run' : 'off'">{{ status.observerOnline ? '运行中' : '已停止' }}</span>
          </div>
          <div class="svc-meta">内置审计 + 审批队列，策略决策可回溯</div>
        </div>
      </div>
    </section>

    <section class="hub-section">
      <div class="section-bar">
        <div class="section-label tight">一键接入 AI 客户端</div>
        <el-button size="small" type="primary" plain @click="openWizard()">接入向导</el-button>
      </div>
      <p class="section-hint-line">检测到已安装的客户端会优先推荐；接入时自动签发 scoped token（可见服务器 + IP 白名单）。</p>
      <div class="client-grid">
        <div v-for="c in clients" :key="c.id" class="client-card" :class="{ dim: !c.installed && !c.linked }">
          <div class="client-top">
            <div class="client-name-row">
              <span class="client-avatar" :data-id="c.id">{{ avatarLetter(c.name) }}</span>
              <div>
                <div class="client-name-line">
                  <strong>{{ c.name }}</strong>
                  <span v-if="c.linked" class="tag linked">已接入</span>
                  <span v-else-if="c.installed" class="tag host">已安装</span>
                  <span v-else class="tag off">未检测到</span>
                </div>
                <p class="client-desc">{{ c.desc }}</p>
              </div>
            </div>
          </div>
          <div class="client-path-row">
            <el-input :model-value="c.configPath || c.config" readonly size="small" class="path-input" />
            <el-tooltip :content="guidanceTip(c)" placement="top">
              <button
                type="button"
                class="icon-btn guidance-btn"
                :class="{
                  stale: needsGuidance(c),
                  ok: c.guidanceOk,
                }"
                @click="onGuidance(c)"
              >
                规则
                <span v-if="needsGuidance(c)" class="status-dot warn" aria-hidden="true" />
                <span v-else-if="c.guidanceOk" class="status-dot ok" aria-hidden="true" />
              </button>
            </el-tooltip>
            <template v-if="c.linked">
              <el-tooltip content="重新签发 Token 并刷新配置" placement="top">
                <button type="button" class="icon-btn" :disabled="!!busy" @click="refreshClient(c.id)">
                  <el-icon><Refresh /></el-icon>
                </button>
              </el-tooltip>
              <el-tooltip content="打开配置文件" placement="top">
                <button type="button" class="icon-btn" @click="openPath(c.configPath)">
                  <el-icon><FolderOpened /></el-icon>
                </button>
              </el-tooltip>
              <el-tooltip content="移除接入并删除专属 Token" placement="top">
                <button type="button" class="icon-btn danger" :disabled="!!busy" @click="uninstallClient(c)">
                  <el-icon><Delete /></el-icon>
                </button>
              </el-tooltip>
            </template>
            <el-button v-else type="primary" size="small" class="import-btn" :loading="busy === c.id" @click="openWizard(c.id)">
              + 接入
            </el-button>
          </div>
        </div>
      </div>
    </section>

    <section class="hub-section">
      <div class="section-bar">
        <div class="section-label tight">
          Scoped Tokens
          <span class="count-chip">{{ tokens.length }} 个</span>
        </div>
        <el-button size="small" type="primary" plain @click="openIssue">+ 手动生成</el-button>
      </div>
      <div v-if="tokens.length" class="token-table-wrap">
        <el-table :data="tokens" size="small" class="token-table" empty-text="暂无">
        <el-table-column prop="name" label="名称" :min-width="embedded ? 100 : 140" show-overflow-tooltip />
        <el-table-column prop="client" label="客户端" :width="embedded ? 88 : 120" />
        <el-table-column label="可见服务器" :width="embedded ? 88 : 110">
          <template #default="{ row }">
            <el-tooltip :content="(row.servers || []).join(', ') || '全部'" placement="top">
              <span>{{ (row.servers || []).length ? row.servers.length + ' 台' : '全部' }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="IP 白名单" :min-width="embedded ? 100 : 140" show-overflow-tooltip>
          <template #default="{ row }">{{ (row.cidrs || []).join(', ') || '127.0.0.1/32' }}</template>
        </el-table-column>
        <el-table-column label="创建" :width="embedded ? 128 : 160">
          <template #default="{ row }">{{ fmtTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="最后使用" :width="embedded ? 128 : 160">
          <template #default="{ row }">{{ fmtTime(row.lastUsedAt) || '—' }}</template>
        </el-table-column>
        <el-table-column label="操作" :width="embedded ? 72 : 90" fixed="right">
          <template #default="{ row }">
            <el-button text size="small" type="danger" @click="revokeToken(row.id)">删除</el-button>
          </template>
        </el-table-column>
        </el-table>
      </div>
      <div v-else class="empty-hint">暂无 scoped token。一键接入或手动生成时会签发；明文只在创建时展示一次。</div>
      <div class="token-foot">
        <button v-if="tokens.length" type="button" class="link-btn" @click="clearTokens">全部清除（{{ tokens.length }}）</button>
        <span class="muted">Token 只存 SHA256；关闭弹窗后无法再看明文。</span>
      </div>
    </section>

    <section class="hub-section">
      <div class="section-label">手动接入（claude.ai 等）</div>
      <p class="manual-tip">复制配置后把 Token 换成接入时复制的值。stdio 推荐本地 CLI；HTTP 适合浏览器 / 隧道。</p>
      <div class="manual-list">
        <div class="manual-row">
          <div class="manual-left">
            <span class="manual-tag">stdio JSON</span>
            <span class="manual-desc">Claude Code / OpenCode</span>
          </div>
          <code class="mono manual-preview">{{ preview(snippets.stdioJson) }}</code>
          <el-button size="small" @click="copy(snippets.stdioJson)">复制</el-button>
        </div>
        <div class="manual-row">
          <div class="manual-left">
            <span class="manual-tag">Streamable HTTP</span>
            <span class="manual-desc">{{ snippets.httpUrl || 'http://127.0.0.1:&lt;port&gt;/mcp' }}</span>
          </div>
          <code class="mono manual-preview">{{ preview(snippets.httpJson) }}</code>
          <el-button size="small" @click="copy(snippets.httpJson)">复制</el-button>
        </div>
        <div class="manual-row">
          <div class="manual-left">
            <span class="manual-tag">TOML（Codex）</span>
            <span class="manual-desc">~/.codex/config.toml</span>
          </div>
          <code class="mono manual-preview">{{ preview(snippets.toml) }}</code>
          <el-button size="small" @click="copy(snippets.toml)">复制</el-button>
        </div>
      </div>
    </section>

    <!-- 接入向导 -->
    <el-dialog v-model="wizardOpen" :title="wizardTitle" width="520px" append-to-body destroy-on-close>
      <el-form label-position="top" size="small">
        <el-form-item v-if="!wizard.clientId" label="客户端">
          <el-checkbox-group v-model="wizard.clientIds">
            <el-checkbox v-for="c in clients" :key="c.id" :label="c.id" :disabled="!c.installed && !c.linked">
              {{ c.name }}{{ c.installed ? '' : '（未检测到）' }}
            </el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="Token 名">
          <el-input v-model="wizard.tokenName" placeholder="例如 Cursor" />
        </el-form-item>
        <el-form-item label="可见服务器（空=全部）">
          <el-select v-model="wizard.servers" multiple filterable collapse-tags collapse-tags-tooltip placeholder="默认全选可见" style="width: 100%">
            <el-option v-for="a in aliases" :key="a" :label="a" :value="a" />
          </el-select>
        </el-form-item>
        <el-form-item label="IP 白名单 (CIDR)">
          <el-select v-model="wizard.cidrs" multiple filterable allow-create default-first-option placeholder="127.0.0.1/32" style="width: 100%">
            <el-option label="127.0.0.1/32" value="127.0.0.1/32" />
            <el-option label="::1/128" value="::1/128" />
            <el-option label="192.168.0.0/16" value="192.168.0.0/16" />
            <el-option label="10.0.0.0/8" value="10.0.0.0/8" />
          </el-select>
          <div class="form-tip">不允许公网 CIDR。HTTP 接入校验 RemoteAddr。</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="wizardOpen = false">取消</el-button>
        <el-button type="primary" :loading="!!busy" @click="runWizard">生成并写入配置</el-button>
      </template>
    </el-dialog>

    <!-- 一次性 Token -->
    <el-dialog v-model="plainOpen" title="Token 已签发（仅此一次）" width="560px" append-to-body :close-on-click-modal="false">
      <el-alert type="warning" :closable="false" show-icon title="关闭后无法再查看明文。请立即复制并妥善保存。" />
      <div class="plain-box">
        <code>{{ plainToken }}</code>
      </div>
      <template #footer>
        <el-button type="primary" @click="copy(plainToken)">复制 Token</el-button>
        <el-button @click="plainOpen = false">我已保存</el-button>
      </template>
    </el-dialog>

    <!-- Cursor 规则复制 -->
    <el-dialog v-model="cursorRuleOpen" title="Cursor 规则（复制粘贴）" width="640px" append-to-body>
      <p class="form-tip">复制后粘贴到 Cursor 的 User Rules / 项目 Rules。正文不含文件标记，可直接用。</p>
      <el-input v-model="cursorRuleText" type="textarea" :rows="14" readonly class="rule-preview" />
      <template #footer>
        <el-button type="primary" @click="copy(cursorRuleText)">复制</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="bindDialog" title="绑定状况" width="640px" append-to-body>
      <el-table :data="clients" size="small">
        <el-table-column prop="name" label="客户端" width="120" />
        <el-table-column label="检测" width="90">
          <template #default="{ row }">{{ row.installed ? '已安装' : '未检测到' }}</template>
        </el-table-column>
        <el-table-column label="接入" width="90">
          <template #default="{ row }">
            <span class="tag" :class="row.linked ? 'linked' : 'off'">{{ row.linked ? '已接入' : '未接入' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="规则" width="90">
          <template #default="{ row }">{{ row.id === 'cursor' ? '复制' : row.guidanceOk ? '已写' : '待写' }}</template>
        </el-table-column>
        <el-table-column prop="configPath" label="配置路径" show-overflow-tooltip />
      </el-table>
    </el-dialog>
  </div>
</template>

<script>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Refresh,
  Delete,
  CircleCheckFilled,
  WarningFilled,
  FolderOpened,
} from '@element-plus/icons-vue'
import {
  GetMCPStatus,
  GetMCPSettings,
  SaveMCPSettings,
  GetMCPSnippets,
  ListMCPTokens,
  ListMCPClients,
  ListMCPServerAliases,
  InstallMCPClientWith,
  UninstallMCPClient,
  RefreshMCPClient,
  IssueMCPToken,
  RevokeMCPToken,
  ClearMCPTokens,
  OpenMCPPath,
  WriteMCPGuidance,
  GetMCPGuidancePreview,
} from '../../wailsjs/go/app/App'

export default {
  name: 'McpAccessView',
  components: { Refresh, Delete, CircleCheckFilled, WarningFilled, FolderOpened },
  props: {
    embedded: { type: Boolean, default: false },
    active: { type: Boolean, default: true },
  },
  setup(props) {
    const loading = ref(false)
    const busy = ref('')
    const status = ref({})
    const snippets = ref({ stdioJson: '', httpJson: '', toml: '', httpUrl: '' })
    const tokens = ref([])
    const clients = ref([])
    const aliases = ref([])
    const bindDialog = ref(false)
    const wizardOpen = ref(false)
    const issueOnly = ref(false)
    const plainOpen = ref(false)
    const plainToken = ref('')
    const cursorRuleOpen = ref(false)
    const cursorRuleText = ref('')
    const mcpSettings = ref({ enabled: false, autoStart: false })
    const mcpSettingsFull = ref({})
    const settingsSaving = ref('')
    const wizard = reactive({
      clientId: '',
      clientIds: [],
      tokenName: '',
      servers: [],
      cidrs: ['127.0.0.1/32'],
    })

    const wizardTitle = computed(() => {
      if (wizard.clientId) {
        const c = clients.value.find((x) => x.id === wizard.clientId)
        return `接入 ${c?.name || wizard.clientId}`
      }
      return '接入向导'
    })

    const reload = async () => {
      loading.value = true
      try {
        status.value = (await GetMCPStatus()) || {}
        const settings = (await GetMCPSettings()) || {}
        mcpSettingsFull.value = settings
        mcpSettings.value = {
          enabled: !!settings.enabled,
          autoStart: !!settings.autoStart,
        }
        snippets.value = (await GetMCPSnippets()) || {}
        tokens.value = (await ListMCPTokens()) || []
        clients.value = (await ListMCPClients()) || []
        aliases.value = (await ListMCPServerAliases()) || []
      } catch (e) {
        ElMessage.error(`加载 MCP 状态失败: ${e}`)
      } finally {
        loading.value = false
      }
    }

    const persistMcpSettings = async (patch, savingKey) => {
      settingsSaving.value = savingKey
      try {
        const merged = { ...mcpSettingsFull.value, ...patch }
        await SaveMCPSettings(merged)
        mcpSettingsFull.value = merged
        status.value = (await GetMCPStatus()) || {}
        snippets.value = (await GetMCPSnippets()) || {}
        ElMessage.success('已保存')
      } catch (e) {
        ElMessage.error(`保存失败: ${e}`)
        await reload()
      } finally {
        settingsSaving.value = ''
      }
    }

    const onAutoStartChange = async (val) => {
      await persistMcpSettings({ autoStart: !!val }, 'autoStart')
    }

    const onEnabledChange = async (val) => {
      await persistMcpSettings({ enabled: !!val }, 'enabled')
    }

    const showPlain = (tok) => {
      if (tok?.token) {
        plainToken.value = tok.token
        plainOpen.value = true
      }
    }

    const openWizard = (clientId = '') => {
      issueOnly.value = false
      wizard.clientId = clientId || ''
      wizard.clientIds = clientId
        ? [clientId]
        : clients.value.filter((c) => c.installed).map((c) => c.id)
      if (!wizard.clientIds.length && !clientId) {
        wizard.clientIds = clients.value.map((c) => c.id)
      }
      const c = clients.value.find((x) => x.id === (clientId || wizard.clientIds[0]))
      wizard.tokenName = c ? c.name : 'FlashShell Token'
      wizard.servers = [...aliases.value]
      wizard.cidrs = ['127.0.0.1/32']
      wizardOpen.value = true
    }

    const openIssue = () => {
      issueOnly.value = true
      wizard.clientId = ''
      wizard.clientIds = []
      wizard.tokenName = '手动 Token'
      wizard.servers = [...aliases.value]
      wizard.cidrs = ['127.0.0.1/32']
      wizardOpen.value = true
    }

    const runWizard = async () => {
      const opts = {
        tokenName: wizard.tokenName,
        servers: wizard.servers,
        cidrs: wizard.cidrs?.length ? wizard.cidrs : ['127.0.0.1/32'],
      }
      if (issueOnly.value) {
        busy.value = 'issue'
        try {
          const tok = await IssueMCPToken({
            name: opts.tokenName,
            client: 'manual',
            servers: opts.servers,
            cidrs: opts.cidrs,
          })
          wizardOpen.value = false
          showPlain(tok)
          await reload()
          ElMessage.success('已签发 Token')
        } catch (e) {
          ElMessage.error(`签发失败: ${e}`)
        } finally {
          busy.value = ''
        }
        return
      }
      const ids = wizard.clientId ? [wizard.clientId] : wizard.clientIds
      if (!ids.length) {
        ElMessage.warning('请选择至少一个客户端')
        return
      }
      busy.value = 'wizard'
      try {
        let lastTok = null
        for (const id of ids) {
          lastTok = await InstallMCPClientWith(id, {
            tokenName: opts.tokenName || undefined,
            servers: opts.servers,
            cidrs: opts.cidrs,
          })
        }
        wizardOpen.value = false
        showPlain(lastTok)
        await reload()
        ElMessage.success('已写入客户端配置，请在对应客户端 Reload MCP')
      } catch (e) {
        ElMessage.error(`接入失败: ${e}`)
      } finally {
        busy.value = ''
      }
    }

    const refreshClient = async (id) => {
      busy.value = `${id}:refresh`
      try {
        const tok = await RefreshMCPClient(id)
        showPlain(tok)
        ElMessage.success('已重新签发并刷新配置')
        await reload()
      } catch (e) {
        ElMessage.error(`刷新失败: ${e}`)
      } finally {
        busy.value = ''
      }
    }

    const uninstallClient = async (c) => {
      try {
        await ElMessageBox.confirm(
          `将从 ${c.name} 移除 FlashShell，并删除相关 scoped token。`,
          '移除接入',
          { type: 'warning', confirmButtonText: '移除', cancelButtonText: '取消' },
        )
      } catch {
        return
      }
      busy.value = `${c.id}:off`
      try {
        await UninstallMCPClient(c.id)
        ElMessage.success('已移除')
        await reload()
      } catch (e) {
        ElMessage.error(`移除失败: ${e}`)
      } finally {
        busy.value = ''
      }
    }

    const needsGuidance = (c) => c && c.id !== 'cursor' && !c.guidanceOk
    const guidanceTip = (c) => {
      if (!c) return ''
      if (c.id === 'cursor') return '复制规则到 Cursor（User Rules / 项目 Rules）'
      if (c.guidanceOk) return `规则已写入${c.guidancePath ? '：' + c.guidancePath : ''}`
      return '写规则（红点=未写或版本过时）'
    }

    const onGuidance = async (c) => {
      if (c.id === 'cursor') {
        cursorRuleText.value = (await GetMCPGuidancePreview('cursor')) || ''
        cursorRuleOpen.value = true
        return
      }
      try {
        await WriteMCPGuidance(c.id)
        ElMessage.success('已写入规则（仅替换标记区间，写前已备份 .flashshell.bak）')
        await reload()
      } catch (e) {
        ElMessage.error(`写规则失败: ${e}`)
      }
    }

    const revokeToken = async (id) => {
      try {
        await ElMessageBox.confirm('删除后对应客户端将收到 401，需重新接入。', '删除 Token', {
          type: 'warning',
        })
      } catch {
        return
      }
      try {
        await RevokeMCPToken(id)
        await reload()
      } catch (e) {
        ElMessage.error(`删除失败: ${e}`)
      }
    }

    const clearTokens = async () => {
      try {
        await ElMessageBox.confirm(`清除全部 ${tokens.value.length} 个 Token？`, '全部清除', {
          type: 'warning',
        })
      } catch {
        return
      }
      try {
        await ClearMCPTokens()
        ElMessage.success('已清除')
        await reload()
      } catch (e) {
        ElMessage.error(`清除失败: ${e}`)
      }
    }

    const openPath = async (path) => {
      if (!path) return ElMessage.warning('路径为空')
      try {
        await OpenMCPPath(path)
      } catch (e) {
        ElMessage.error(`打开失败: ${e}`)
      }
    }

    const avatarLetter = (name) => String(name || '?').trim().charAt(0).toUpperCase() || '?'
    const displayPath = (p) => {
      const s = String(p || '')
      const m = s.match(/^\/Users\/[^/]+/)
      return m ? `~${s.slice(m[0].length)}` : s
    }
    const preview = (text) => {
      const s = String(text || '').replace(/\s+/g, ' ').trim()
      return s.length <= 72 ? s || '—' : `${s.slice(0, 72)}…`
    }
    const fmtTime = (t) => {
      if (!t) return ''
      const d = new Date(t)
      if (Number.isNaN(d.getTime())) return String(t).replace('T', ' ').slice(0, 19)
      return d.toLocaleString()
    }
    const copy = async (text) => {
      try {
        await navigator.clipboard.writeText(String(text || ''))
        ElMessage.success('已复制')
      } catch {
        ElMessage.warning('复制失败')
      }
    }

    onMounted(() => {
      if (props.active) reload()
    })
    watch(() => props.active, (v) => {
      if (v) reload()
    })
    return {
      embedded: computed(() => props.embedded),
      loading,
      busy,
      status,
      snippets,
      tokens,
      clients,
      aliases,
      bindDialog,
      wizardOpen,
      wizard,
      wizardTitle,
      plainOpen,
      plainToken,
      cursorRuleOpen,
      cursorRuleText,
      mcpSettings,
      settingsSaving,
      onAutoStartChange,
      onEnabledChange,
      reload,
      openWizard,
      openIssue,
      runWizard,
      refreshClient,
      uninstallClient,
      onGuidance,
      needsGuidance,
      guidanceTip,
      revokeToken,
      clearTokens,
      openPath,
      avatarLetter,
      displayPath,
      preview,
      fmtTime,
      copy,
    }
  },
}
</script>

<style scoped>
.mcp-hub {
  flex: 1;
  width: 100%;
  min-width: 0;
  min-height: 0;
  height: 100%;
  box-sizing: border-box;
  overflow: auto;
  padding: var(--app-space-page, 28px 32px 24px);
  background: var(--app-bg);
  color: var(--app-text);
}
.mcp-hub.embedded {
  padding: 0 4px 8px 0;
  background: transparent;
  height: 100%;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
}
.mcp-hub.embedded .hub-head {
  margin-bottom: 10px;
  gap: 10px;
  align-items: center;
}
.mcp-hub.embedded .hub-embedded-title {
  margin: 0;
  font-size: 15px;
  font-weight: 650;
  line-height: 1.3;
}
.hub-sub--solo {
  margin-top: 4px;
}
.mcp-hub.embedded .hub-section {
  margin-bottom: 14px;
}
.mcp-hub.embedded .section-label {
  margin-bottom: 8px;
  font-size: 12px;
}
.mcp-hub.embedded .section-hint-line {
  margin-bottom: 8px;
  font-size: 10px;
  line-height: 1.4;
}
.mcp-hub.embedded .svc-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}
.mcp-hub.embedded .svc-card {
  padding: 10px 11px;
}
.mcp-hub.embedded .svc-meta {
  margin-top: 4px;
  font-size: 10px;
  line-height: 1.35;
}
.mcp-hub.embedded .svc-actions {
  margin-top: 8px;
}
.mcp-hub.embedded .svc-foot {
  margin-top: 6px;
  font-size: 10px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.mcp-hub.embedded .client-grid {
  grid-template-columns: 1fr;
  gap: 8px;
}
.mcp-hub.embedded .client-card {
  padding: 10px 12px;
}
.mcp-hub.embedded .client-top {
  margin-bottom: 8px;
}
.mcp-hub.embedded .client-desc {
  font-size: 11px;
}
.mcp-hub.embedded .token-table-wrap {
  overflow-x: auto;
}
.mcp-hub.embedded .token-table {
  max-height: 168px;
  min-width: 640px;
}
.mcp-hub.embedded .token-table :deep(.el-table__body-wrapper) {
  overflow-y: auto;
}
.mcp-hub.embedded .manual-tip {
  margin: -2px 0 8px;
  font-size: 11px;
}
.mcp-hub.embedded .manual-row {
  grid-template-columns: minmax(0, 140px) minmax(0, 1fr) auto;
  gap: 8px;
  padding: 10px 12px;
}
.mcp-hub.embedded .manual-preview {
  font-size: 10px;
}
.mcp-hub.embedded .hub-sub {
  font-size: 11px;
  line-height: 1.45;
  max-width: none;
}
.mcp-hub.embedded .stat-group {
  font-size: 11px;
}
.mcp-hub.embedded .hub-head-right {
  gap: 8px;
}
.mcp-settings-section {
  margin-bottom: 16px;
}
.mcp-settings-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}
.mcp-setting-item {
  padding: 10px 12px;
  border: 1px solid var(--app-border);
  border-radius: 10px;
  background: var(--app-card-bg);
}
.mcp-setting-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.mcp-setting-name {
  font-size: 13px;
  font-weight: 600;
}
.mcp-setting-desc {
  margin: 6px 0 0;
  font-size: 11px;
  line-height: 1.45;
  color: var(--app-text-muted);
}
.mcp-hub.embedded .mcp-settings-grid {
  grid-template-columns: 1fr;
}
.mcp-hub.embedded .mcp-settings-section {
  margin-bottom: 12px;
}
.hub-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  margin-bottom: 22px;
}
.hub-title-wrap { min-width: 0; flex: 1; }
h1 { margin: 0; font-size: 20px; font-weight: 650; }
.hub-sub {
  margin: 8px 0 0;
  font-size: 12px;
  line-height: 1.55;
  color: var(--app-text-muted);
  max-width: 720px;
}
.hub-sub code,
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
}
.hub-head-right { display: flex; align-items: center; gap: 12px; flex-shrink: 0; }
.stat-group {
  display: inline-flex;
  align-items: baseline;
  gap: 4px;
  font-size: 12px;
  color: var(--app-text-muted);
}
.stat-num { font-weight: 700; color: var(--app-text); }
.stat-sep { opacity: 0.45; margin: 0 2px; }
.hub-section { margin-bottom: 26px; }
.section-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 12px;
}
.section-label.tight { margin-bottom: 0; }
.section-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  gap: 12px;
}
.section-hint-line {
  margin: 0 0 12px;
  font-size: 11px;
  color: var(--app-text-muted);
}
.count-chip {
  margin-left: 8px;
  font-size: 11px;
  font-weight: 500;
  color: var(--app-text-muted);
  border: 1px solid var(--app-border);
  border-radius: 999px;
  padding: 1px 8px;
}
.check { display: inline-flex; color: #9ca3af; font-size: 16px; }
.check.on { color: #22c55e; }
.icon-btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 28px;
  height: 28px;
  padding: 0 8px;
  border-radius: 6px;
  border: 1px solid var(--app-border);
  background: var(--app-card-bg);
  color: var(--app-text-secondary, var(--app-text-muted));
  cursor: pointer;
  flex-shrink: 0;
  font-size: 11px;
}
.icon-btn:hover { color: var(--app-text); }
.icon-btn.ghost { border: none; background: transparent; width: 24px; height: 24px; padding: 0; }
.icon-btn.danger:hover { color: #ef4444; }
.icon-btn.stale { border-color: color-mix(in srgb, #ef4444 40%, var(--app-border)); }
.icon-btn.ok { border-color: color-mix(in srgb, #22c55e 40%, var(--app-border)); }
.guidance-btn { min-width: 52px; }
.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}
.status-dot.warn { background: #ef4444; }
.status-dot.ok { background: #22c55e; }
.icon-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.rule-preview :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.5;
}
.svc-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}
.svc-card,
.client-card {
  background: var(--app-card-bg);
  border: 1px solid var(--app-card-border, var(--app-border));
  border-radius: 10px;
  padding: 14px 16px;
}
.client-card.dim { opacity: 0.72; }
.svc-title-row { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.svc-name { font-size: 13px; font-weight: 650; }
.svc-meta { margin-top: 6px; font-size: 11px; color: var(--app-text-muted); line-height: 1.45; word-break: break-all; }
.svc-actions { margin-top: 12px; display: flex; gap: 6px; flex-wrap: wrap; }
.svc-foot { margin-top: 10px; font-size: 11px; color: var(--app-text-muted); word-break: break-all; }
.tag {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 8px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 700;
  background: var(--app-bg);
  color: var(--app-text-muted);
  border: 1px solid var(--app-border);
}
.tag.host,
.tag.linked { color: #16a34a; border-color: color-mix(in srgb, #22c55e 35%, var(--app-border)); background: color-mix(in srgb, #22c55e 12%, transparent); }
.tag.online,
.tag.run { color: #ca8a04; border-color: color-mix(in srgb, #eab308 40%, var(--app-border)); background: color-mix(in srgb, #eab308 14%, transparent); }
.tag.off { color: var(--app-text-muted); }
.client-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.client-top { margin-bottom: 12px; }
.client-name-row { display: flex; align-items: flex-start; gap: 10px; }
.client-avatar {
  width: 32px; height: 32px; border-radius: 8px; display: inline-flex; align-items: center; justify-content: center;
  font-size: 13px; font-weight: 700; color: #fff; background: #64748b; flex-shrink: 0;
}
.client-avatar[data-id='claude-code'] { background: #d97706; }
.client-avatar[data-id='codex'] { background: #111827; }
.client-avatar[data-id='cursor'] { background: #0f766e; }
.client-avatar[data-id='opencode'] { background: #2563eb; }
.client-name-line { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.client-desc { margin: 4px 0 0; font-size: 12px; color: var(--app-text-muted); }
.client-path-row { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.path-input { flex: 1; min-width: 120px; }
.import-btn { flex-shrink: 0; }
.token-table { width: 100%; border: 1px solid var(--app-border); border-radius: 10px; overflow: hidden; }
.token-foot { margin-top: 8px; font-size: 11px; color: var(--app-text-muted); display: flex; gap: 16px; align-items: center; flex-wrap: wrap; }
.link-btn { border: none; background: transparent; color: var(--app-accent-color, #16a34a); cursor: pointer; padding: 0; font-size: 11px; }
.link-btn:hover { text-decoration: underline; }
.muted { color: var(--app-text-muted); }
.empty-hint {
  font-size: 12px; color: var(--app-text-muted); padding: 14px;
  border: 1px dashed var(--app-border); border-radius: 10px; background: var(--app-card-bg);
}
.manual-tip { margin: -4px 0 12px; font-size: 12px; line-height: 1.5; color: #ca8a04; }
.manual-list { display: flex; flex-direction: column; gap: 8px; }
.manual-row {
  display: grid; grid-template-columns: 200px 1fr auto; gap: 12px; align-items: center;
  padding: 12px 14px; border: 1px solid var(--app-border); border-radius: 10px; background: var(--app-card-bg);
}
.manual-left { display: flex; flex-direction: column; gap: 4px; }
.manual-tag { font-size: 12px; font-weight: 650; }
.manual-desc { font-size: 11px; color: var(--app-text-muted); }
.manual-preview { color: var(--app-text-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.form-tip { margin-top: 4px; font-size: 11px; color: var(--app-text-muted); }
.plain-box {
  margin-top: 12px; padding: 12px; border-radius: 8px; background: var(--app-bg);
  border: 1px solid var(--app-border); word-break: break-all; font-family: ui-monospace, monospace; font-size: 12px;
}
@media (max-width: 1100px) {
  .svc-grid, .client-grid { grid-template-columns: 1fr; }
  .manual-row { grid-template-columns: 1fr; }
  .hub-head { flex-direction: column; }
}
.mcp-hub.embedded .svc-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
.mcp-hub.embedded .client-grid {
  grid-template-columns: 1fr;
}
.mcp-hub.embedded .manual-row {
  grid-template-columns: minmax(0, 140px) minmax(0, 1fr) auto;
}
.mcp-hub.embedded .hub-head {
  flex-direction: row;
}
</style>
