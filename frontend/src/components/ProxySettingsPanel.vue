<template>
  <div class="proxy-panel">
    <div class="panel-scroll">
      <p class="proxy-hint">
        启用手动代理后，应用内 HTTP 请求（检查更新 / 下载）与 SSH / SFTP 连接均走该代理。
        已建立的远程会话需重新连接后才会切换。
      </p>

      <div class="proxy-block">
        <div class="block-label">代理方式</div>
        <el-radio-group v-model="form.mode" class="proxy-mode-group">
          <el-radio label="none">无代理</el-radio>
          <el-radio label="manual">手动设置代理</el-radio>
        </el-radio-group>
      </div>

      <div v-show="form.mode === 'manual'" class="proxy-block proxy-manual">
        <div class="system-setting-row">
          <div class="system-setting-text">
            <span class="system-setting-label">代理类型</span>
            <span class="system-setting-hint">HTTP（CONNECT）或 SOCKS5</span>
          </div>
          <div class="system-setting-control">
            <el-radio-group v-model="form.type" size="small">
              <el-radio-button label="http">HTTP</el-radio-button>
              <el-radio-button label="socks">SOCKS</el-radio-button>
            </el-radio-group>
          </div>
        </div>

        <div class="system-setting-row">
          <div class="system-setting-text">
            <span class="system-setting-label">主机</span>
            <span class="system-setting-hint">代理服务器 IP 或域名</span>
          </div>
          <div class="system-setting-control">
            <el-input
              v-model="form.host"
              size="small"
              clearable
              placeholder="代理主机"
              class="proxy-host"
            />
          </div>
        </div>

        <div class="system-setting-row">
          <div class="system-setting-text">
            <span class="system-setting-label">端口</span>
          </div>
          <div class="system-setting-control">
            <el-input-number
              v-model="form.port"
              class="proxy-port"
              size="small"
              :min="1"
              :max="65535"
              :step="1"
              controls-position="right"
            />
          </div>
        </div>

        <div class="system-setting-row">
          <div class="system-setting-text">
            <span class="system-setting-label">用户名</span>
            <span class="system-setting-hint">可选，代理需要认证时填写</span>
          </div>
          <div class="system-setting-control">
            <el-input
              v-model="form.user"
              size="small"
              clearable
              placeholder="可选"
              class="proxy-host"
              autocomplete="off"
            />
          </div>
        </div>

        <div class="system-setting-row">
          <div class="system-setting-text">
            <span class="system-setting-label">密码</span>
            <span class="system-setting-hint">可选，与用户名一起用于代理认证</span>
          </div>
          <div class="system-setting-control">
            <el-input
              v-model="form.password"
              size="small"
              clearable
              show-password
              placeholder="可选"
              class="proxy-host"
              autocomplete="new-password"
            />
          </div>
        </div>
      </div>

      <div class="proxy-block">
        <el-button size="small" @click="openTestDialog">检查连接</el-button>
      </div>
    </div>

    <div class="panel-actions icon-actions">
      <el-tooltip content="保存代理设置" placement="top">
        <el-button type="primary" circle :loading="saving" @click="save">
          <el-icon v-if="!saving"><Check /></el-icon>
        </el-button>
      </el-tooltip>
    </div>

    <el-dialog
      v-model="testVisible"
      title="检查代理连接"
      width="420px"
      append-to-body
      destroy-on-close
    >
      <p class="test-hint">输入要访问的地址，使用当前表单中的代理配置进行测试</p>
      <el-input
        v-model="testURL"
        clearable
        placeholder="测试地址"
        @keydown.enter.exact.prevent="runTest"
      />
      <template #footer>
        <div class="dialog-footer icon-actions">
          <el-tooltip content="取消" placement="top">
            <el-button circle @click="testVisible = false">
              <el-icon><Close /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip content="测试连接" placement="top">
            <el-button type="primary" circle :loading="testing" @click="runTest">
              <el-icon v-if="!testing"><Connection /></el-icon>
            </el-button>
          </el-tooltip>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { reactive, ref, watch } from 'vue'
import { Check, Close, Connection } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as App from '../../wailsjs/go/app/App'

const defaultForm = () => ({
  mode: 'none',
  type: 'http',
  host: '',
  port: 7890,
  user: '',
  password: '',
})

export default {
  name: 'ProxySettingsPanel',
  components: { Check, Close, Connection },
  props: {
    active: { type: Boolean, default: false },
  },
  setup(props) {
    const saving = ref(false)
    const testing = ref(false)
    const testVisible = ref(false)
    const testURL = ref('')
    const form = reactive(defaultForm())

    const applyConfig = (cfg) => {
      const p = cfg?.proxySettings || {}
      form.mode = p.mode === 'manual' ? 'manual' : 'none'
      form.type = p.type === 'socks' ? 'socks' : 'http'
      form.host = p.host || ''
      form.port = p.port > 0 ? p.port : 7890
      form.user = p.user || ''
      form.password = p.password || ''
    }

    const load = async () => {
      try {
        const cfg = await App.GetSystemSettings()
        applyConfig(cfg)
      } catch (e) {
        ElMessage.error(`加载代理设置失败: ${e}`)
      }
    }

    const proxyPayload = () => ({
      mode: form.mode,
      type: form.type,
      host: String(form.host || '').trim(),
      port: form.port || 7890,
      user: String(form.user || '').trim(),
      password: form.password || '',
    })

    const save = async () => {
      if (form.mode === 'manual') {
        if (!String(form.host || '').trim()) {
          ElMessage.warning('请填写代理主机')
          return
        }
        if (!form.port || form.port < 1 || form.port > 65535) {
          ElMessage.warning('请填写有效端口')
          return
        }
      }
      saving.value = true
      try {
        const cfg = await App.GetSystemSettings()
        cfg.proxySettings = proxyPayload()
        await App.SaveSystemSettings(cfg)
        ElMessage.success('代理设置已保存')
        await maybeReconnectSessions()
      } catch (e) {
        ElMessage.error(`保存失败: ${e}`)
      } finally {
        saving.value = false
      }
    }

    const maybeReconnectSessions = async () => {
      let sessions = []
      try {
        sessions = ((await App.GetShellSessions()) || []).filter(
          (s) => s?.machineName && s.kind !== 'local' && !String(s.machineName).startsWith('local'),
        )
      } catch {
        return
      }
      if (!sessions.length) return
      try {
        await ElMessageBox.confirm(
          `代理已更新。是否立即重连当前 ${sessions.length} 个远程会话？`,
          '重连会话',
          {
            confirmButtonText: '全部重连',
            cancelButtonText: '稍后手动',
            type: 'info',
          },
        )
      } catch {
        return
      }
      let ok = 0
      let fail = 0
      for (const s of sessions) {
        try {
          await App.ReconnectShell(s.machineName)
          ok += 1
        } catch {
          fail += 1
        }
      }
      if (fail) ElMessage.warning(`已重连 ${ok} 个，失败 ${fail} 个`)
      else ElMessage.success(`已重连 ${ok} 个远程会话`)
    }

    const openTestDialog = () => {
      testVisible.value = true
    }

    const runTest = async () => {
      testing.value = true
      try {
        const msg = await App.TestProxyConnection(proxyPayload(), testURL.value)
        ElMessage.success(msg || '连接成功')
      } catch (e) {
        ElMessage.error(String(e?.message || e || '连接失败'))
      } finally {
        testing.value = false
      }
    }

    watch(
      () => props.active,
      (open) => {
        if (open) load()
      },
      { immediate: true },
    )

    return {
      form,
      saving,
      testing,
      testVisible,
      testURL,
      save,
      openTestDialog,
      runTest,
    }
  },
}
</script>

<style scoped>
.proxy-panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
  overflow: hidden;
}

.panel-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding-right: 4px;
}

.proxy-hint {
  margin: 0 0 16px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--app-text-muted);
}

.proxy-block {
  margin-bottom: 18px;
}

.block-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
  margin-bottom: 10px;
}

.proxy-mode-group {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
}

.proxy-manual {
  padding: 12px 14px;
  border: 1px solid var(--app-border);
  border-radius: var(--app-radius-md, 8px);
  background: var(--app-panel-bg, transparent);
}

.system-setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 0;
}

.system-setting-row + .system-setting-row {
  border-top: 1px solid var(--app-border);
}

.system-setting-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.system-setting-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--app-text);
}

.system-setting-hint {
  font-size: 12px;
  color: var(--app-text-muted);
}

.system-setting-control {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.proxy-host {
  width: 200px;
}

.proxy-port {
  width: 120px;
}

.panel-actions {
  flex-shrink: 0;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 12px;
  border-top: 1px solid var(--app-border);
  margin-top: 8px;
}

.test-hint {
  margin: 0 0 12px;
  font-size: 13px;
  color: var(--app-text-muted);
  line-height: 1.45;
}
</style>
