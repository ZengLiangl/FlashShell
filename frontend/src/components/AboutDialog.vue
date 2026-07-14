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
                <p><strong>版本</strong>：v1.2.0</p>
                <p><strong>框架</strong>：Wails v2 + Vue 3 + Element Plus + xterm.js</p>
                <p><strong>开源协议</strong>：MIT</p>
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

export default {
    name: 'AboutDialog',
    props: {
        modelValue: { type: Boolean, required: true },
        introHtml: { type: String, default: '' }
    },
    emits: ['update:modelValue'],
    setup(props, { emit }) {
        const visibleProxy = ref(props.modelValue)
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

        const handleClose = () => { visibleProxy.value = false }

        return { visibleProxy, handleClose, projectIntro }
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
</style>
