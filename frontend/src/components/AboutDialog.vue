<template>
    <el-dialog v-model="visibleProxy" title="关于 Quick Cmd" width="700px" :before-close="handleClose">
        <div class="about-container">
            <div class="brand">
                <div>
                    <h2>Quick Cmd</h2>
                    <p class="subtitle">快速执行与管理命令的桌面工具</p>
                </div>
            </div>

            <el-divider></el-divider>

            <div class="intro" v-html="projectIntro"></div>

            <el-divider></el-divider>

            <div class="meta">
                <p><strong>版本</strong>：v1.2.0</p>
                <p><strong>框架</strong>：Wails v2 + Vue 3 + Element Plus</p>
                <p><strong>开源协议</strong>：MIT</p>
            </div>
        </div>
        <template #footer>
            <span class="dialog-footer">
                <el-button @click="handleClose">关闭</el-button>
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
      <p>这是一个名为 <strong>Quick Cmd</strong> 的 Go 桌面应用程序，主要用于<strong>快速执行和管理各种命令行任务</strong>。基于 Wails 跨平台 GUI 框架，支持本地与远程（SSH）命令执行、SFTP 传输、多项目与环境变量管理，并提供实时终端输出与 ANSI 颜色渲染。</p>
      <ul>
        <li><strong>图形化管理</strong>：项目/子项目/命令分组，所见即所得</li>
        <li><strong>配置驱动</strong>：YAML 配置，支持多文件切换与变量替换</li>
        <li><strong>多执行模式</strong>：本地批处理与远程 SSH 执行</li>
        <li><strong>远程管理</strong>：SSH/SFTP、连接测试、敏感信息加密</li>
        <li><strong>实时交互</strong>：内置终端，进度与状态可视化</li>
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
    gap: 12px;
}

.brand .icon {
    width: 40px;
    height: 40px;
    border-radius: 8px;
}

.subtitle {
    margin: 4px 0 0;
    color: #909399;
    font-size: 12px;
}

.intro {
    line-height: 1.7;
    color: #606266;
}

.intro ul {
    margin: 8px 0 0 18px;
}

.meta {
    color: #909399;
    font-size: 12px;
}
</style>
