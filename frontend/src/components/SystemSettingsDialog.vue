<template>
    <el-dialog v-model="visibleProxy" title="系统设置" width="640px" :before-close="handleClose">
        <el-form label-width="120px">
            <el-divider content-position="left">执行日志</el-divider>
            <el-form-item label="日志落盘">
                <el-switch v-model="form.logSettings.enabled" />
            </el-form-item>
            <el-form-item label="落盘路径">
                <el-input v-model="form.logSettings.path" placeholder="~/.cmd-config/logs" />
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

            <el-divider content-position="left">窗口会话</el-divider>
            <el-form-item label="会话 ID">
                <el-input :model-value="sessionId" readonly />
            </el-form-item>
        </el-form>

        <template #footer>
            <el-button @click="handleClose">取消</el-button>
            <el-button type="primary" :loading="saving" @click="save">保存</el-button>
        </template>
    </el-dialog>
</template>

<script>
import { ref, reactive, watch, computed } from 'vue'
import { ElMessage } from 'element-plus'
import * as App from '../../wailsjs/go/app/App'
import { useTheme } from '../composables/useTheme'

export default {
    name: 'SystemSettingsDialog',
    props: {
        modelValue: { type: Boolean, default: false }
    },
    emits: ['update:modelValue'],
    setup(props, { emit }) {
        const saving = ref(false)
        const sessionId = ref('')
        const { saveTheme } = useTheme()
        const form = reactive({
            logSettings: { enabled: false, path: '~/.cmd-config/logs' },
            themeSettings: { mode: 'light', terminalPreset: 'classic' }
        })

        const visibleProxy = computed({
            get: () => props.modelValue,
            set: (v) => emit('update:modelValue', v)
        })

        const load = async () => {
            const config = await App.GetSystemSettings()
            form.logSettings = { ...config.logSettings }
            form.themeSettings = { ...config.themeSettings }
            const session = await App.GetSessionInfo()
            sessionId.value = session.sessionId || ''
        }

        watch(() => props.modelValue, (open) => {
            if (open) load()
        })

        const save = async () => {
            saving.value = true
            try {
                const config = await App.GetSystemSettings()
                config.logSettings = { ...form.logSettings }
                config.themeSettings = { ...form.themeSettings }
                await App.SaveSystemSettings(config)
                await saveTheme(form.themeSettings.mode, form.themeSettings.terminalPreset)
                ElMessage.success('系统设置已保存')
                visibleProxy.value = false
            } catch (e) {
                ElMessage.error(`保存失败: ${e}`)
            } finally {
                saving.value = false
            }
        }

        const handleClose = () => { visibleProxy.value = false }

        return { visibleProxy, form, saving, sessionId, save, handleClose }
    }
}
</script>
