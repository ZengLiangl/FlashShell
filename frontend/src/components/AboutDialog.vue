<template>
    <el-dialog
        v-model="visibleProxy"
        :title="promptMode ? '发现新版本' : '关于 FlashDock'"
        width="720px"
        :before-close="handleClose"
    >
        <div class="about-container">
            <div class="brand">
                <img class="brand-mark" src="../assets/appicon.png" alt="" aria-hidden="true" />
                <div>
                    <h2>FlashDock <span class="cn">闪舵</span></h2>
                    <p class="subtitle">一次停靠 · 本地任务与远程 Shell 同港出海</p>
                </div>
            </div>

            <el-divider />

            <div v-if="!promptMode" class="intro" v-html="projectIntro"></div>
            <el-divider v-if="!promptMode" />

            <div class="meta">
                <p><strong>版本</strong>：{{ versionLabel }}</p>
                <p v-if="!promptMode"><strong>框架</strong>：Wails v2 + Vue 3 + Element Plus + xterm.js</p>
                <p v-if="!promptMode"><strong>开源协议</strong>：MIT</p>
            </div>

            <div class="update-block" v-loading="checking">
                <div v-if="updateResult?.hasUpdate" class="update-banner">
                    <div class="update-banner-title">发现新版本 {{ updateResult.latestVersion }}</div>
                    <div class="update-banner-sub">当前 {{ updateResult.currentVersion }}</div>
                    <div
                        v-if="updateResult.releaseNotes"
                        class="update-notes markdown-body"
                        v-html="renderReleaseNotes(updateResult.releaseNotes)"
                        @click="onNotesClick"
                    ></div>
                    <div v-if="updateResult.assetName" class="asset-line">
                        适配安装包：{{ updateResult.assetName }}
                    </div>
                    <div v-else-if="updateResult.error" class="update-err" style="margin-top: 8px">
                        {{ updateResult.error }}
                    </div>
                    <div class="update-actions">
                        <el-button
                            type="primary"
                            size="small"
                            :loading="downloading"
                            :disabled="!canDownload"
                            @click="downloadUpdate"
                        >
                            {{ downloading ? `下载中 ${downloadPercent}%` : '下载安装包' }}
                        </el-button>
                        <el-button size="small" @click="openRelease">查看 Release</el-button>
                    </div>
                    <el-progress
                        v-if="downloading || downloadPercent > 0"
                        :percentage="downloadPercent"
                        :stroke-width="10"
                        style="margin-top: 10px"
                    />
                    <div v-if="downloadMessage" class="download-msg" :class="{ err: downloadFailed }">
                        {{ downloadMessage }}
                    </div>
                </div>
                <div v-else-if="updateResult && !updateResult.error" class="update-ok">
                    已是最新版本
                </div>
                <div v-else-if="updateResult?.error" class="update-err">
                    {{ updateResult.error }}
                </div>
                <el-button v-if="!promptMode" size="small" text :loading="checking" @click="checkUpdate">
                    检查更新
                </el-button>
            </div>
        </div>
        <template #footer>
            <span class="dialog-footer">
                <el-button
                    v-if="promptMode && updateResult?.hasUpdate"
                    @click="skipThisVersion"
                >
                    跳过此版本
                </el-button>
                <el-button type="primary" @click="handleClose">关闭</el-button>
            </span>
        </template>
    </el-dialog>
</template>

<script>
import { ref, watch, computed, onMounted, onUnmounted } from 'vue'
import { marked } from 'marked'
import { ElMessage } from 'element-plus'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

marked.setOptions({
    breaks: true,
    gfm: true,
})

export default {
    name: 'AboutDialog',
    props: {
        modelValue: { type: Boolean, required: true },
        introHtml: { type: String, default: '' },
        /** 启动/首页更新提示模式：显示「跳过此版本」 */
        promptMode: { type: Boolean, default: false },
        /** 外部预检查结果，避免重复请求 */
        initialUpdateResult: { type: Object, default: null },
    },
    emits: ['update:modelValue', 'skipped', 'dismissed'],
    setup(props, { emit }) {
        const visibleProxy = ref(props.modelValue)
        const versionLabel = ref('…')
        const checking = ref(false)
        const updateResult = ref(null)
        const downloading = ref(false)
        const downloadPercent = ref(0)
        const downloadMessage = ref('')
        const downloadFailed = ref(false)

        watch(() => props.modelValue, v => (visibleProxy.value = v))
        watch(visibleProxy, v => emit('update:modelValue', v))

        const canDownload = computed(() =>
            !!(updateResult.value?.hasUpdate && updateResult.value?.downloadURL && !downloading.value)
        )

        const renderReleaseNotes = (md) => {
            const text = String(md || '').trim()
            if (!text) return ''
            try {
                return marked.parse(text)
            } catch {
                return text
                    .replace(/&/g, '&amp;')
                    .replace(/</g, '&lt;')
                    .replace(/>/g, '&gt;')
                    .replace(/\n/g, '<br>')
            }
        }

        const defaultIntro = `
      <p><strong>FlashDock（闪舵）</strong>：跨平台桌面运维工作台，任务流水线与远程 Shell 集于一体。</p>
      <ul>
        <li><strong>任务模式</strong>：YAML 驱动本地 / 远程命令，一键执行并实时回传</li>
        <li><strong>Shell 模式</strong>：多会话 SSH、SFTP，支持导入 Xshell / FinalShell</li>
        <li><strong>并行使用</strong>：任务与 Shell 互不打断，首页随时切换</li>
        <li><strong>配置与安全</strong>：机器 / 环境变量 / 主题可配，敏感信息加密存储</li>
      </ul>
    `

        const projectIntro = computed(() => props.introHtml || defaultIntro)

        const loadVersion = async () => {
            try {
                versionLabel.value = await App.GetAppVersion() || 'v—'
            } catch {
                versionLabel.value = 'v—'
            }
        }

        const checkUpdate = async () => {
            checking.value = true
            downloadPercent.value = 0
            downloadMessage.value = ''
            downloadFailed.value = false
            try {
                updateResult.value = await App.CheckForUpdates()
            } catch (e) {
                updateResult.value = { error: String(e), hasUpdate: false }
            } finally {
                checking.value = false
            }
        }

        const openRelease = () => {
            const url = updateResult.value?.releaseURL
            if (url) App.OpenReleaseURL(url)
        }

        const onNotesClick = (e) => {
            const a = e.target?.closest?.('a')
            if (!a) return
            const href = a.getAttribute('href') || ''
            if (!/^https?:\/\//i.test(href)) return
            e.preventDefault()
            App.OpenReleaseURL(href)
        }

        const onDownloadProgress = (payload) => {
            if (!payload) return
            downloadPercent.value = Number(payload.percent) || 0
            if (payload.status === 'start' || payload.status === 'downloading') {
                downloading.value = true
                downloadFailed.value = false
                downloadMessage.value = payload.status === 'start' ? '开始下载…' : '正在下载…'
            } else if (payload.status === 'done') {
                downloading.value = false
                downloadPercent.value = 100
                downloadFailed.value = false
                downloadMessage.value = '下载完成，已打开下载目录'
            } else if (payload.status === 'error') {
                downloading.value = false
                downloadFailed.value = true
                downloadMessage.value = payload.message || '下载失败'
            }
        }

        const downloadUpdate = async () => {
            if (!canDownload.value) return
            downloading.value = true
            downloadFailed.value = false
            downloadMessage.value = '准备下载…'
            downloadPercent.value = 0
            try {
                const result = await App.DownloadUpdate()
                if (result?.success) {
                    ElMessage.success(result.message || '下载完成')
                    downloadMessage.value = result.message || '下载完成，已打开下载目录'
                    downloadPercent.value = 100
                } else {
                    downloadFailed.value = true
                    downloadMessage.value = result?.message || '下载失败'
                    ElMessage.error(downloadMessage.value)
                }
            } catch (e) {
                downloadFailed.value = true
                downloadMessage.value = String(e)
                ElMessage.error(downloadMessage.value)
            } finally {
                downloading.value = false
            }
        }

        const skipThisVersion = async () => {
            const ver = updateResult.value?.latestVersion
            if (!ver) {
                handleClose()
                return
            }
            try {
                await App.SkipUpdateVersion(ver)
                ElMessage.success(`已跳过 ${ver}，有更新版本时再提醒`)
            } catch (e) {
                ElMessage.error('跳过失败: ' + e)
                return
            }
            emit('skipped', ver)
            visibleProxy.value = false
        }

        watch(visibleProxy, async (open) => {
            if (!open) return
            downloadPercent.value = 0
            downloadMessage.value = ''
            downloadFailed.value = false
            await loadVersion()
            if (props.initialUpdateResult) {
                updateResult.value = props.initialUpdateResult
            } else {
                updateResult.value = null
                await checkUpdate()
            }
        })

        const handleClose = () => {
            if (props.promptMode) emit('dismissed')
            visibleProxy.value = false
        }

        onMounted(() => {
            EventsOn('update:download-progress', onDownloadProgress)
        })
        onUnmounted(() => {
            EventsOff('update:download-progress')
        })

        return {
            visibleProxy,
            handleClose,
            projectIntro,
            versionLabel,
            checking,
            updateResult,
            checkUpdate,
            openRelease,
            renderReleaseNotes,
            onNotesClick,
            canDownload,
            downloading,
            downloadPercent,
            downloadMessage,
            downloadFailed,
            downloadUpdate,
            skipThisVersion,
            promptMode: computed(() => props.promptMode),
        }
    }
}
</script>

<style scoped>
.about-container {
    padding: 4px 8px;
}

.brand {
    display: flex;
    align-items: center;
    gap: 14px;
}

.brand-mark {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    object-fit: cover;
    flex-shrink: 0;
    display: block;
}

.brand h2 {
    margin: 0;
    font-size: 22px;
    letter-spacing: 0.02em;
}

.brand .cn {
    margin-left: 6px;
    font-size: 16px;
    font-weight: 600;
    color: var(--app-text-muted, #909399);
}

.subtitle {
    margin: 6px 0 0;
    color: var(--app-text-muted, #909399);
    font-size: 13px;
}

.intro {
    line-height: 1.75;
    color: var(--app-text-secondary, #606266);
}

.intro ul {
    margin: 10px 0 0 18px;
    padding: 0;
}

.intro li {
    margin: 4px 0;
}

.meta {
    color: var(--app-text-muted, #909399);
    font-size: 12px;
}

.meta p {
    margin: 4px 0;
}

.update-block {
    margin-top: 16px;
    min-height: 36px;
}

.update-banner {
    padding: 12px 14px;
    margin-bottom: 8px;
    border-radius: 10px;
    border: 1px solid color-mix(in srgb, #e6a23c 45%, var(--app-border, #e4e7ed));
    background: color-mix(in srgb, #e6a23c 10%, transparent);
}

.update-banner-title {
    font-size: 14px;
    font-weight: 650;
    color: var(--app-text, #303133);
}

.update-banner-sub {
    margin-top: 2px;
    font-size: 12px;
    color: var(--app-text-muted, #909399);
}

.asset-line {
    margin: 8px 0 4px;
    font-size: 12px;
    color: var(--app-text-muted, #909399);
}

.update-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: 10px;
}

.download-msg {
    margin-top: 8px;
    font-size: 12px;
    color: #67c23a;
    line-height: 1.4;
    word-break: break-all;
}

.download-msg.err {
    color: #f56c6c;
}

.update-notes {
    margin: 10px 0;
    max-height: 200px;
    overflow: auto;
    padding: 10px 12px;
    border-radius: 8px;
    background: var(--app-card-bg, #fff);
    border: 1px solid var(--app-border, #e4e7ed);
    font-size: 12px;
    line-height: 1.55;
    word-break: break-word;
    color: var(--app-text-secondary, #606266);
}

.update-notes :deep(h1),
.update-notes :deep(h2),
.update-notes :deep(h3) {
    margin: 0.65em 0 0.35em;
    font-size: 13px;
    font-weight: 650;
    color: var(--app-text, #303133);
    line-height: 1.35;
}

.update-notes :deep(h1:first-child),
.update-notes :deep(h2:first-child),
.update-notes :deep(h3:first-child) {
    margin-top: 0;
}

.update-notes :deep(p),
.update-notes :deep(ul),
.update-notes :deep(ol) {
    margin: 0.35em 0;
}

.update-notes :deep(ul),
.update-notes :deep(ol) {
    padding-left: 1.35em;
}

.update-notes :deep(li) {
    margin: 0.2em 0;
}

.update-notes :deep(hr) {
    margin: 0.75em 0;
    border: none;
    border-top: 1px solid var(--app-border, #e4e7ed);
}

.update-notes :deep(a) {
    color: var(--app-accent-color, #409eff);
    text-decoration: none;
}

.update-notes :deep(a:hover) {
    text-decoration: underline;
}

.update-notes :deep(code) {
    padding: 0 4px;
    border-radius: 3px;
    background: color-mix(in srgb, var(--app-text-muted, #909399) 14%, transparent);
    font-size: 0.95em;
}

.update-notes :deep(strong) {
    color: var(--app-text, #303133);
    font-weight: 650;
}

.update-ok {
    margin-bottom: 6px;
    font-size: 13px;
    color: #67c23a;
}

.update-err {
    margin-bottom: 6px;
    font-size: 12px;
    color: #f56c6c;
    line-height: 1.45;
}

.dialog-footer {
    display: inline-flex;
    gap: 8px;
}
</style>
