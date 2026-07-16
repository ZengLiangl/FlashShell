<template>
    <div class="general-settings-panel" :class="{ embedded }">
        <div class="settings-subnav">
            <button
                v-for="tab in settingsTabs"
                :key="tab.id"
                type="button"
                class="subnav-item"
                :class="{ active: settingsTab === tab.id }"
                @click="settingsTab = tab.id"
            >
                {{ tab.label }}
            </button>
        </div>

        <div class="panel-scroll">
            <!-- 帐号 -->
            <section v-show="settingsTab === 'accounts'" class="settings-section">
                <div class="section-head">
                    <div>
                        <h4>全局 SSH 帐号</h4>
                        <p>添加机器时可一键填充用户名与密码</p>
                    </div>
                    <el-tooltip content="添加帐号" placement="top">
                        <el-button size="small" type="primary" circle @click="addAccount">
                            <el-icon><Plus /></el-icon>
                        </el-button>
                    </el-tooltip>
                </div>
                <el-table :data="accounts" size="small" style="width: 100%" empty-text="暂无帐号">
                    <el-table-column prop="name" label="帐号名称" width="160" />
                    <el-table-column prop="user" label="用户名" width="140" />
                    <el-table-column label="密码">
                        <template #default="scope">
                            {{ scope.row.password ? '******' : '未设置' }}
                        </template>
                    </el-table-column>
                    <el-table-column label="操作" width="100" align="center">
                        <template #default="scope">
                            <div class="icon-actions">
                                <el-tooltip content="编辑" placement="top">
                                    <el-button size="small" text type="primary" @click="editAccount(scope.$index)">
                                        <el-icon><Edit /></el-icon>
                                    </el-button>
                                </el-tooltip>
                                <el-tooltip content="删除" placement="top">
                                    <el-button size="small" text type="danger" @click="removeAccount(scope.$index)">
                                        <el-icon><Delete /></el-icon>
                                    </el-button>
                                </el-tooltip>
                            </div>
                        </template>
                    </el-table-column>
                </el-table>
            </section>

            <!-- 外观 -->
            <section v-show="settingsTab === 'appearance'" class="settings-section appearance-section">
                <div class="appear-layout">
                    <div class="appear-controls">
                        <div class="appear-block">
                            <div class="block-label">界面</div>
                            <el-radio-group v-model="form.themeSettings.mode" size="small">
                                <el-radio-button label="light">浅色</el-radio-button>
                                <el-radio-button label="dark">深色</el-radio-button>
                                <el-radio-button label="system">跟随系统</el-radio-button>
                            </el-radio-group>
                            <div class="preset-grid accent-grid">
                                <button
                                    v-for="accent in uiAccents"
                                    :key="accent.id"
                                    type="button"
                                    class="accent-swatch"
                                    :class="{ active: form.themeSettings.uiAccent === accent.id }"
                                    :title="accent.label"
                                    :style="{ background: accent.light.accent }"
                                    @click="form.themeSettings.uiAccent = accent.id"
                                ></button>
                            </div>
                            <el-select v-model="form.themeSettings.uiFontFamily" size="small" placeholder="界面字体" style="width: 100%">
                                <el-option
                                    v-for="font in uiFonts"
                                    :key="font.id"
                                    :label="font.label"
                                    :value="font.id"
                                />
                            </el-select>
                        </div>

                        <div class="appear-block">
                            <div class="block-label">终端</div>
                            <div class="preset-grid terminal-grid">
                                <button
                                    v-for="preset in terminalPresets"
                                    :key="preset.id"
                                    type="button"
                                    class="term-card"
                                    :class="{ active: form.themeSettings.terminalPreset === preset.id }"
                                    :title="preset.label"
                                    @click="form.themeSettings.terminalPreset = preset.id"
                                >
                                    <span
                                        class="term-card-preview"
                                        :style="{ background: preset.theme.background, color: preset.theme.foreground }"
                                    >
                                        <span class="term-card-dots">
                                            <i :style="{ background: preset.theme.red }"></i>
                                            <i :style="{ background: preset.theme.green }"></i>
                                            <i :style="{ background: preset.theme.blue || preset.theme.cursor }"></i>
                                        </span>
                                        <code>~/</code>
                                    </span>
                                    <span class="term-card-name">{{ preset.label }}</span>
                                </button>
                            </div>
                            <div class="term-font-row">
                                <el-select v-model="form.themeSettings.shellFontFamily" size="small" placeholder="终端字体" class="term-font-select">
                                    <el-option
                                        v-for="font in terminalFonts"
                                        :key="font.id"
                                        :label="font.label"
                                        :value="font.id"
                                    />
                                </el-select>
                                <el-input-number
                                    v-model="form.themeSettings.shellFontSize"
                                    class="term-num"
                                    size="small"
                                    :min="10"
                                    :max="28"
                                    :step="1"
                                    controls-position="right"
                                />
                                <el-input-number
                                    v-model="form.themeSettings.shellLineHeight"
                                    class="term-num"
                                    size="small"
                                    :min="1"
                                    :max="2.5"
                                    :step="0.1"
                                    :precision="1"
                                    controls-position="right"
                                />
                            </div>
                            <div class="term-font-hints">
                                <span>字体</span>
                                <span>字号</span>
                                <span>行高</span>
                            </div>
                        </div>
                    </div>

                    <div class="appear-preview">
                        <div class="block-label">预览</div>
                        <div class="theme-preview" :class="{ dark: previewIsDark }">
                            <div class="preview-ui" :style="previewUiStyle">
                                <div class="preview-bar">
                                    <span class="preview-dot"></span>
                                    <span class="preview-title">FlashDock</span>
                                    <span class="preview-pill">{{ previewUiFontLabel }}</span>
                                </div>
                                <div class="preview-body">
                                    <div class="preview-card">界面卡片预览</div>
                                    <button type="button" class="preview-btn">主按钮</button>
                                </div>
                            </div>
                            <div class="preview-term" :style="previewTermStyle">
                                <div class="preview-term-title">{{ previewTermLabel }}</div>
                                <pre>{{ previewTermSample }}</pre>
                            </div>
                        </div>
                    </div>
                </div>
            </section>

            <!-- 关于 -->
            <section v-show="settingsTab === 'about'" class="settings-section">
                <div class="about-card">
                    <div class="about-row">
                        <span class="about-label">当前版本</span>
                        <span class="version-text">{{ appVersion || '—' }}</span>
                    </div>
                    <div class="about-row about-row--update">
                        <span class="about-label">检查更新</span>
                        <div class="about-update">
                            <div class="icon-actions">
                                <el-tooltip content="检查更新" placement="top">
                                    <el-button size="small" :loading="checkingUpdate" circle @click="checkUpdate">
                                        <el-icon v-if="!checkingUpdate"><Refresh /></el-icon>
                                    </el-button>
                                </el-tooltip>
                                <el-tooltip v-if="updateResult?.hasUpdate && updateResult?.releaseURL" content="打开下载页" placement="top">
                                    <el-button size="small" type="primary" circle @click="openRelease">
                                        <el-icon><Link /></el-icon>
                                    </el-button>
                                </el-tooltip>
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
                        </div>
                    </div>
                    <div class="about-row">
                        <span class="about-label">会话 ID</span>
                        <el-input :model-value="sessionId" readonly size="small" />
                    </div>
                </div>
            </section>
        </div>

        <div class="panel-actions icon-actions">
            <el-tooltip content="保存设置" placement="top">
                <el-button type="primary" circle :loading="saving" @click="save">
                    <el-icon v-if="!saving"><Check /></el-icon>
                </el-button>
            </el-tooltip>
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
                <div class="dialog-footer icon-actions">
                    <el-tooltip content="取消" placement="top">
                        <el-button circle @click="accountEditVisible = false">
                            <el-icon><Close /></el-icon>
                        </el-button>
                    </el-tooltip>
                    <el-tooltip content="确定" placement="top">
                        <el-button type="primary" circle :loading="savingAccount" @click="confirmAccount">
                            <el-icon v-if="!savingAccount"><Check /></el-icon>
                        </el-button>
                    </el-tooltip>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script>
import { ref, reactive, watch, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Edit, Delete, Refresh, Link, Plus, Close, Check } from '@element-plus/icons-vue'
import * as App from '../../wailsjs/go/app/App'
import { useTheme } from '../composables/useTheme'
import {
    UI_ACCENTS,
    TERMINAL_PRESETS,
    UI_FONTS,
    TERMINAL_FONTS,
    getUiAccent,
    getUiFont,
    getTerminalFont,
    getTerminalPreset,
} from '../utils/themePresets'

export default {
    name: 'SystemSettingsDialog',
    components: { Edit, Delete, Refresh, Link, Plus, Close, Check },
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
        const settingsTab = ref('appearance')
        const settingsTabs = [
            { id: 'accounts', label: '帐号' },
            { id: 'appearance', label: '外观' },
            { id: 'about', label: '关于' },
        ]
        const form = reactive({
            themeSettings: {
                mode: 'light',
                uiAccent: 'blue',
                terminalPreset: 'classic',
                uiFontFamily: 'system',
                shellFontFamily: 'consolas',
                shellFontSize: 13,
                shellLineHeight: 1.2,
            }
        })

        const uiAccents = UI_ACCENTS
        const terminalPresets = TERMINAL_PRESETS
        const uiFonts = UI_FONTS
        const terminalFonts = TERMINAL_FONTS

        const previewIsDark = computed(() => {
            const mode = form.themeSettings.mode
            if (mode === 'dark') return true
            if (mode === 'light') return false
            return window.matchMedia('(prefers-color-scheme: dark)').matches
        })

        const previewUiStyle = computed(() => {
            const accent = getUiAccent(form.themeSettings.uiAccent)
            const palette = previewIsDark.value ? accent.dark : accent.light
            const font = getUiFont(form.themeSettings.uiFontFamily)
            return {
                '--preview-accent': palette.accent,
                '--preview-accent-bg': palette.accentBg,
                fontFamily: font.value,
                background: previewIsDark.value ? '#1d1e1f' : '#f5f7fa',
                color: previewIsDark.value ? '#e5eaf3' : '#303133',
                borderColor: previewIsDark.value ? '#414243' : '#e4e7ed',
            }
        })

        const previewTermStyle = computed(() => {
            const theme = getTerminalPreset(form.themeSettings.terminalPreset).theme
            const font = getTerminalFont(form.themeSettings.shellFontFamily)
            return {
                background: theme.background,
                color: theme.foreground,
                fontFamily: font.value,
                fontSize: `${form.themeSettings.shellFontSize || 13}px`,
                lineHeight: form.themeSettings.shellLineHeight || 1.2,
            }
        })

        const previewUiFontLabel = computed(() => getUiFont(form.themeSettings.uiFontFamily).label)
        const previewTermLabel = computed(() => getTerminalPreset(form.themeSettings.terminalPreset).label)
        const previewTermSample = computed(() => {
            const theme = getTerminalPreset(form.themeSettings.terminalPreset).theme
            return `user@host:~$ ls -la
drwxr-xr-x  12 user  staff  384 Jul 16 09:00 .
-rw-r--r--   1 user  staff  128 Jul 16 09:00 README.md
user@host:~$ echo "theme: ${getTerminalPreset(form.themeSettings.terminalPreset).label}"
theme preview · ${theme.foreground}`
        })

        const visibleProxy = computed({
            get: () => props.modelValue,
            set: (v) => emit('update:modelValue', v)
        })

        const load = async () => {
            const config = await App.GetSystemSettings()
            form.themeSettings = {
                mode: config.themeSettings?.mode || 'light',
                uiAccent: config.themeSettings?.uiAccent || 'blue',
                terminalPreset: config.themeSettings?.terminalPreset || 'classic',
                uiFontFamily: config.themeSettings?.uiFontFamily || 'system',
                shellFontFamily: config.themeSettings?.shellFontFamily || 'consolas',
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
            settingsTab,
            settingsTabs,
            form,
            uiAccents,
            terminalPresets,
            uiFonts,
            terminalFonts,
            previewIsDark,
            previewUiStyle,
            previewTermStyle,
            previewUiFontLabel,
            previewTermLabel,
            previewTermSample,
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
.general-settings-panel {
    display: flex;
    flex-direction: column;
    min-height: 0;
    height: 100%;
}

.general-settings-panel.embedded {
    padding-bottom: 0;
}

.settings-subnav {
    flex-shrink: 0;
    display: flex;
    gap: 4px;
    padding: 0 0 10px;
    border-bottom: 1px solid var(--app-border);
    margin-bottom: 12px;
}

.subnav-item {
    border: none;
    background: transparent;
    color: var(--app-text-muted);
    font-size: 13px;
    padding: 6px 12px;
    border-radius: 8px;
    cursor: pointer;
}

.subnav-item:hover {
    color: var(--app-accent-color);
    background: var(--app-accent-bg);
}

.subnav-item.active {
    color: var(--app-accent-color);
    background: var(--app-accent-bg);
    font-weight: 650;
}

.panel-scroll {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 0;
}

.panel-actions {
    flex-shrink: 0;
    display: flex;
    justify-content: flex-end;
    align-items: center;
    gap: 10px;
    margin: 0 -18px 0;
    padding: 12px 18px;
    border-top: 1px solid var(--app-border);
    background: var(--app-panel-bg);
}

.panel-actions.icon-actions {
    display: flex;
    width: auto;
    align-self: stretch;
}

.settings-section {
    min-height: 100%;
}

.section-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 12px;
}

.section-head h4 {
    margin: 0;
    font-size: 14px;
    color: var(--app-text);
}

.section-head p {
    margin: 4px 0 0;
    font-size: 12px;
    color: var(--app-text-muted);
}

.appear-layout {
    display: grid;
    grid-template-columns: minmax(0, 1.1fr) minmax(0, 0.9fr);
    gap: 16px;
    align-items: start;
}

.appear-block {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 12px;
    border: 1px solid var(--app-border);
    border-radius: 10px;
    background: var(--app-bg);
    min-width: 0;
    overflow: hidden;
}

.appear-controls {
    display: flex;
    flex-direction: column;
    gap: 16px;
    min-width: 0;
    overflow: hidden;
}

.block-label {
    font-size: 12px;
    font-weight: 650;
    color: var(--app-text-secondary);
}

.accent-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
}

.accent-swatch {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    border: 2px solid transparent;
    cursor: pointer;
    padding: 0;
}

.accent-swatch.active {
    box-shadow: 0 0 0 2px var(--app-panel-bg), 0 0 0 4px currentColor;
    outline: none;
    border-color: #fff;
}

.terminal-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(104px, 1fr));
    gap: 8px;
}

.term-card {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 0;
    border: 1px solid var(--app-border);
    border-radius: 8px;
    background: transparent;
    cursor: pointer;
    overflow: hidden;
    color: var(--app-text);
    text-align: left;
}

.term-card.active {
    border-color: var(--app-accent-color);
    box-shadow: 0 0 0 1px var(--app-accent-color);
}

.term-card-preview {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 8px;
    min-height: 52px;
    font-size: 11px;
}

.term-card-dots {
    display: flex;
    gap: 3px;
}

.term-card-dots i {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    display: inline-block;
}

.term-card-preview code {
    font-family: Consolas, monospace;
    opacity: 0.85;
}

.term-card-name {
    padding: 0 8px 8px;
    font-size: 11px;
    color: var(--app-text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.term-font-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 76px 76px;
    gap: 8px;
    align-items: center;
    width: 100%;
    min-width: 0;
}

.term-font-select {
    width: 100%;
    min-width: 0;
}

.term-num {
    width: 76px !important;
}

.term-num :deep(.el-input-number),
.term-num.el-input-number {
    width: 76px;
}

.term-num :deep(.el-input__wrapper) {
    padding-left: 4px;
    padding-right: 28px;
}

.term-font-hints {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 76px 76px;
    gap: 8px;
    margin-top: -4px;
    font-size: 11px;
    color: var(--app-text-muted);
}

.appear-preview {
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-width: 0;
    position: sticky;
    top: 0;
}

.theme-preview {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.preview-ui,
.preview-term {
    border: 1px solid var(--app-border);
    border-radius: 10px;
    overflow: hidden;
    min-height: 120px;
}

.preview-ui {
    display: flex;
    flex-direction: column;
}

.preview-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 10px;
    border-bottom: 1px solid color-mix(in srgb, currentColor 18%, transparent);
}

.preview-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--preview-accent, #409eff);
}

.preview-title {
    font-size: 13px;
    font-weight: 650;
}

.preview-pill {
    margin-left: auto;
    font-size: 11px;
    padding: 2px 8px;
    border-radius: 999px;
    background: var(--preview-accent-bg, #ecf5ff);
    color: var(--preview-accent, #409eff);
}

.preview-body {
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.preview-card {
    padding: 10px 12px;
    border-radius: 8px;
    border: 1px solid color-mix(in srgb, currentColor 14%, transparent);
    font-size: 12px;
}

.preview-btn {
    align-self: flex-start;
    border: none;
    border-radius: 6px;
    padding: 6px 12px;
    background: var(--preview-accent, #409eff);
    color: #fff;
    font-size: 12px;
    cursor: default;
}

.preview-term {
    padding: 10px 12px;
}

.preview-term-title {
    font-size: 11px;
    opacity: 0.7;
    margin-bottom: 8px;
}

.preview-term pre {
    margin: 0;
    white-space: pre-wrap;
    word-break: break-word;
    font-family: inherit;
    font-size: inherit;
    line-height: inherit;
}

.about-card {
    display: flex;
    flex-direction: column;
    gap: 14px;
    padding: 14px;
    border: 1px solid var(--app-border);
    border-radius: 10px;
    background: var(--app-bg);
}

.about-row {
    display: grid;
    grid-template-columns: 88px minmax(0, 1fr);
    gap: 12px;
    align-items: center;
}

.about-row--update {
    align-items: start;
}

.about-label {
    font-size: 13px;
    color: var(--app-text-secondary);
    line-height: 28px;
}

.about-update {
    display: flex;
    flex-direction: row;
    flex-wrap: wrap;
    align-items: center;
    justify-content: flex-start;
    gap: 10px;
    min-width: 0;
}

.about-update .icon-actions {
    flex-shrink: 0;
    justify-content: flex-start;
}

.version-text {
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: 13px;
}

.update-tip {
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

@media (max-width: 860px) {
    .appear-layout {
        grid-template-columns: 1fr;
    }

    .appear-preview {
        position: static;
    }
}
</style>
