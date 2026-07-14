<template>
    <el-dialog v-model="visibleProxy" title="关于 FlashDock" width="720px" :before-close="handleClose">
        <div class="about-container">
            <div class="brand">
                <img class="brand-mark" src="../assets/appicon.png" alt="" aria-hidden="true" />
                <div>
                    <h2>FlashDock <span class="cn">闪舵</span></h2>
                    <p class="subtitle">一次停靠 · 本地任务与远程 Shell 同港出海</p>
                </div>
            </div>

            <el-divider />

            <div class="intro" v-html="projectIntro"></div>

            <el-divider />

            <div class="meta">
                <p><strong>版本</strong>：{{ versionLabel }}</p>
                <p><strong>框架</strong>：Wails v2 + Vue 3 + Element Plus + xterm.js</p>
                <p><strong>开源协议</strong>：MIT</p>
            </div>

            <div class="update-block" v-loading="checking">
                <div v-if="updateResult?.hasUpdate" class="update-banner">
                    <div class="update-banner-title">发现新版本 {{ updateResult.latestVersion }}</div>
                    <div class="update-banner-sub">当前 {{ updateResult.currentVersion }}</div>
                    <pre v-if="updateResult.releaseNotes" class="update-notes">{{ updateResult.releaseNotes }}</pre>
                    <el-button type="primary" size="small" @click="openRelease">查看 Release / 下载</el-button>
                </div>
                <div v-else-if="updateResult && !updateResult.error" class="update-ok">
                    已是最新版本
                </div>
                <div v-else-if="updateResult?.error" class="update-err">
                    {{ updateResult.error }}
                </div>
                <el-button size="small" text :loading="checking" @click="checkUpdate">检查更新</el-button>
            </div>
        </div>
        <template #footer>
            <span class="dialog-footer">
                <el-button type="primary" @click="handleClose">关闭</el-button>
            </span>
        </template>
    </el-dialog>
</template>

<script>
import { ref, watch, computed } from 'vue'
import * as App from '../../wailsjs/go/app/App'

export default {
    name: 'AboutDialog',
    props: {
        modelValue: { type: Boolean, required: true },
        introHtml: { type: String, default: '' }
    },
    emits: ['update:modelValue'],
    setup(props, { emit }) {
        const visibleProxy = ref(props.modelValue)
        const versionLabel = ref('…')
        const checking = ref(false)
        const updateResult = ref(null)

        watch(() => props.modelValue, v => (visibleProxy.value = v))
        watch(visibleProxy, v => emit('update:modelValue', v))

        const defaultIntro = `
      <p><strong>FlashDock（闪舵）</strong> 是一款跨平台桌面运维工作台：用 YAML 驱动本地与远程命令流水线，同时提供多会话 SSH 终端与 SFTP，让构建发布和交互排障在同一应用内完成。</p>
      <ul>
        <li><strong>任务模式</strong>：项目 / 子项目 / 步骤一键执行，实时终端回传</li>
        <li><strong>Shell 模式</strong>：多 Tab SSH、右键操作、SFTP 文件管理</li>
        <li><strong>并行调度</strong>：任务与 Shell 互不打断，首页随时切换</li>
        <li><strong>配置驱动</strong>：业务 YAML + 全局机器 / 环境变量 / 主题 / 快捷键</li>
        <li><strong>安全连接</strong>：敏感信息加密；支持导入 Xshell / FinalShell</li>
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

        watch(visibleProxy, async (open) => {
            if (!open) return
            updateResult.value = null
            await loadVersion()
            await checkUpdate()
        })

        const handleClose = () => { visibleProxy.value = false }

        return {
            visibleProxy,
            handleClose,
            projectIntro,
            versionLabel,
            checking,
            updateResult,
            checkUpdate,
            openRelease,
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

.update-notes {
    margin: 10px 0;
    max-height: 160px;
    overflow: auto;
    padding: 8px 10px;
    border-radius: 8px;
    background: var(--app-card-bg, #fff);
    border: 1px solid var(--app-border, #e4e7ed);
    font-size: 12px;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-word;
    color: var(--app-text-secondary, #606266);
    font-family: inherit;
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
</style>
