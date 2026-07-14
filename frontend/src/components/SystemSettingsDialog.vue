<template>
    <div class="general-settings-panel" :class="{ embedded }">
        <el-form label-width="120px">
            <el-divider content-position="left">全局 SSH 帐号</el-divider>
            <div class="account-toolbar">
                <el-button size="small" type="primary" @click="addAccount">添加帐号</el-button>
            </div>
            <el-table :data="accounts" size="small" style="width: 100%; margin-bottom: 16px;">
                <el-table-column prop="name" label="帐号名称" width="160" />
                <el-table-column prop="user" label="用户名" width="140" />
                <el-table-column label="密码">
                    <template #default="scope">
                        {{ scope.row.password ? '******' : '未设置' }}
                    </template>
                </el-table-column>
                <el-table-column label="操作" width="140">
                    <template #default="scope">
                        <el-button size="small" link @click="editAccount(scope.$index)">编辑</el-button>
                        <el-button size="small" link type="danger" @click="removeAccount(scope.$index)">删除</el-button>
                    </template>
                </el-table-column>
            </el-table>

            <el-divider content-position="left">执行日志</el-divider>
            <el-form-item label="日志落盘">
                <el-switch v-model="form.logSettings.enabled" />
            </el-form-item>
            <el-form-item label="落盘路径">
                <el-input v-model="form.logSettings.path" placeholder="~/.flashdock/logs" />
            </el-form-item>

            <el-divider content-position="left">外观</el-divider>
            <el-form-item label="界面主题">
                <el-radio-group v-model="form.themeSettings.mode">
                    <el-radio value="light">浅色</el-radio>
                    <el-radio value="dark">深色</el-radio>
                    <el-radio value="system">跟随系统</el-radio>
                </el-radio-group>
            </el-form-item>
            <el-form-item label="终端主题">
                <el-select v-model="form.themeSettings.terminalPreset" style="width: 100%">
                    <el-option label="Classic" value="classic" />
                    <el-option label="Monokai" value="monokai" />
                    <el-option label="Solarized" value="solarized" />
                </el-select>
            </el-form-item>
            <el-form-item label="Shell 字号">
                <el-input-number
                    v-model="form.themeSettings.shellFontSize"
                    :min="10"
                    :max="28"
                    :step="1"
                />
                <span class="field-hint">px</span>
            </el-form-item>
            <el-form-item label="Shell 行高">
                <el-input-number
                    v-model="form.themeSettings.shellLineHeight"
                    :min="1"
                    :max="2.5"
                    :step="0.1"
                    :precision="1"
                />
                <span class="field-hint">倍数</span>
            </el-form-item>

            <el-divider content-position="left">窗口会话</el-divider>
            <el-form-item label="会话 ID">
                <el-input :model-value="sessionId" readonly />
            </el-form-item>

            <el-divider content-position="left">版本更新</el-divider>
            <el-form-item label="当前版本">
                <span class="version-text">{{ appVersion }}</span>
            </el-form-item>
            <el-form-item label="检查更新">
                <div class="update-row">
                    <el-button size="small" :loading="checkingUpdate" @click="checkUpdate">检查更新</el-button>
                    <el-button
                        v-if="updateResult?.hasUpdate && updateResult?.releaseURL"
                        size="small"
                        type="primary"
                        @click="openRelease"
                    >
                        打开下载页
                    </el-button>
                </div>
                <div v-if="updateResult?.hasUpdate" class="update-tip warn">
                    发现新版本 {{ updateResult.latestVersion }}
                </div>
                <div v-else-if="updateResult && !updateResult.error" class="update-tip ok">
                    已是最新版本
                </div>
                <div v-else-if="updateResult?.error" class="update-tip err">
                    {{ updateResult.error }}
                </div>
            </el-form-item>
        </el-form>

        <div class="panel-actions">
            <el-button type="primary" :loading="saving" @click="save">保存设置</el-button>
        </div>

        <el-dialog v-model="accountEditVisible" :title="editingAccountIndex >= 0 ? '编辑帐号' : '添加帐号'" width="480px" append-to-body>
            <el-form :model="accountForm" label-width="90px">
                <el-form-item label="帐号名称">
                    <el-input v-model="accountForm.name" placeholder="例如：生产环境" />
                </el-form-item>
                <el-form-item label="用户名">
                    <el-input v-model="accountForm.user" placeholder="SSH 用户名" />
                </el-form-item>
                <el-form-item label="密码">
                    <el-input v-model="accountForm.password" type="password" show-password placeholder="SSH 密码" />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="accountEditVisible = false">取消</el-button>
                <el-button type="primary" :loading="savingAccount" @click="confirmAccount">确定</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script>
import { ref, reactive, watch, computed } from 'vue'
import { ElMessage } from 'element-plus'
import * as App from '../../wailsjs/go/app/App'
import { useTheme } from '../composables/useTheme'

export default {
    name: 'SystemSettingsDialog',
    props: {
        modelValue: { type: Boolean, default: false },
        embedded: { type: Boolean, default: false },
        active: { type: Boolean, default: false },
    },
    emits: ['update:modelValue', 'saved'],
    setup(props, { emit }) {
        const saving = ref(false)
        const savingAccount = ref(false)
        const sessionId = ref('')
        const appVersion = ref('')
        const checkingUpdate = ref(false)
        const updateResult = ref(null)
        const accounts = ref([])
        const accountEditVisible = ref(false)
        const editingAccountIndex = ref(-1)
        const accountForm = reactive({ id: '', name: '', user: '', password: '' })
        const { applyThemeSettings } = useTheme()
        const form = reactive({
            logSettings: { enabled: false, path: '~/.flashdock/logs' },
            themeSettings: { mode: 'light', terminalPreset: 'classic', shellFontSize: 13, shellLineHeight: 1.2 }
        })

        const visibleProxy = computed({
            get: () => props.modelValue,
            set: (v) => emit('update:modelValue', v)
        })

        const load = async () => {
            const config = await App.GetSystemSettings()
            form.logSettings = { ...config.logSettings }
            form.themeSettings = {
                mode: config.themeSettings?.mode || 'light',
                terminalPreset: config.themeSettings?.terminalPreset || 'classic',
                shellFontSize: config.themeSettings?.shellFontSize > 0 ? config.themeSettings.shellFontSize : 13,
                shellLineHeight: config.themeSettings?.shellLineHeight > 0 ? config.themeSettings.shellLineHeight : 1.2,
            }
            accounts.value = await App.GetGlobalAccounts() || []
            const session = await App.GetSessionInfo()
            sessionId.value = session.sessionId || ''
            try {
                appVersion.value = await App.GetAppVersion() || ''
            } catch {
                appVersion.value = ''
            }
            updateResult.value = null
        }

        const checkUpdate = async () => {
            checkingUpdate.value = true
            try {
                updateResult.value = await App.CheckForUpdates()
            } catch (e) {
                updateResult.value = { error: String(e), hasUpdate: false }
            } finally {
                checkingUpdate.value = false
            }
        }

        const openRelease = () => {
            const url = updateResult.value?.releaseURL
            if (url) App.OpenReleaseURL(url)
        }

        watch(() => props.modelValue, (open) => {
            if (!props.embedded && open) load()
        })
        watch(() => props.active, (open) => {
            if (props.embedded && open) load()
        }, { immediate: true })

        const resetAccountForm = () => {
            accountForm.id = crypto.randomUUID()
            accountForm.name = ''
            accountForm.user = ''
            accountForm.password = ''
        }

        const addAccount = () => {
            editingAccountIndex.value = -1
            resetAccountForm()
            accountEditVisible.value = true
        }

        const editAccount = (index) => {
            editingAccountIndex.value = index
            const account = accounts.value[index]
            accountForm.id = account.id || crypto.randomUUID()
            accountForm.name = account.name || ''
            accountForm.user = account.user || ''
            accountForm.password = account.password || ''
            accountEditVisible.value = true
        }

        const saveGlobalAccounts = async (message = '全局 SSH 帐号已保存') => {
            savingAccount.value = true
            try {
                await App.SaveGlobalAccountsFromDTO(accounts.value)
                if (message) ElMessage.success(message)
            } finally {
                savingAccount.value = false
            }
        }

        const removeAccount = async (index) => {
            accounts.value.splice(index, 1)
            try {
                await saveGlobalAccounts('帐号已删除')
            } catch (e) {
                ElMessage.error(`删除失败: ${e}`)
                await load()
            }
        }

        const confirmAccount = async () => {
            if (!accountForm.name.trim() || !accountForm.user.trim()) {
                ElMessage.warning('请填写帐号名称和用户名')
                return
            }
            const payload = {
                id: accountForm.id || crypto.randomUUID(),
                name: accountForm.name.trim(),
                user: accountForm.user.trim(),
                password: accountForm.password
            }
            if (editingAccountIndex.value >= 0) {
                accounts.value[editingAccountIndex.value] = payload
            } else {
                accounts.value.push(payload)
            }
            try {
                await saveGlobalAccounts()
                accountEditVisible.value = false
            } catch (e) {
                ElMessage.error(`保存失败: ${e}`)
                await load()
            }
        }

        const save = async () => {
            saving.value = true
            try {
                const config = await App.GetSystemSettings()
                config.logSettings = { ...form.logSettings }
                config.themeSettings = { ...form.themeSettings }
                await App.SaveSystemSettings(config)
                applyThemeSettings(form.themeSettings)
                ElMessage.success('系统设置已保存')
                emit('saved')
                if (!props.embedded) visibleProxy.value = false
            } catch (e) {
                ElMessage.error(`保存失败: ${e}`)
            } finally {
                saving.value = false
            }
        }

        return {
            embedded: computed(() => props.embedded),
            form,
            accounts,
            saving,
            savingAccount,
            sessionId,
            appVersion,
            checkingUpdate,
            updateResult,
            checkUpdate,
            openRelease,
            accountEditVisible,
            editingAccountIndex,
            accountForm,
            addAccount,
            editAccount,
            removeAccount,
            confirmAccount,
            save,
        }
    }
}
</script>

<style scoped>
.general-settings-panel.embedded {
    padding-bottom: 8px;
}

.panel-actions {
    display: flex;
    justify-content: flex-end;
    margin-top: 16px;
    padding-top: 12px;
    border-top: 1px solid var(--app-border);
}

.account-toolbar {
    margin-bottom: 8px;
}

.version-text {
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: 13px;
}

.update-row {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
}

.update-tip {
    margin-top: 8px;
    font-size: 12px;
    line-height: 1.45;
}

.update-tip.warn {
    color: #e6a23c;
}

.update-tip.ok {
    color: #67c23a;
}

.update-tip.err {
    color: #f56c6c;
}

.field-hint {
    margin-left: 8px;
    color: var(--app-text-muted);
    font-size: 12px;
}
</style>
