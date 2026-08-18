<template>
  <el-dialog
    :model-value="visible"
    title="选择打开方式"
    width="420px"
    append-to-body
    destroy-on-close
    class="sftp-file-opener-dialog"
    @close="emitClose"
  >
    <div class="opener-filename" :title="fileName">{{ fileName }}</div>
    <div class="opener-options">
      <button
        v-if="canEdit"
        type="button"
        class="opener-option"
        @click="pickBuiltin"
      >
        <div class="opener-option-title">内置编辑器</div>
        <div class="opener-option-desc">在 FlashShell 中编辑文本文件</div>
      </button>
      <button
        type="button"
        class="opener-option"
        :disabled="selectingApp"
        @click="pickSystemApp"
      >
        <div class="opener-option-title">选择应用程序...</div>
        <div class="opener-option-desc">下载到临时目录并用指定应用打开</div>
      </button>
    </div>
    <label class="opener-remember">
      <el-checkbox v-model="rememberChoice" />
      <span>始终使用此方式打开 {{ displayExtension }} 文件</span>
    </label>
    <template #footer>
      <el-button @click="emitClose">取消</el-button>
    </template>
  </el-dialog>
</template>

<script>
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import * as App from '../../../wailsjs/go/app/App'
import { getFileExtension, hasFileExtension, isKnownBinaryFile } from '../../utils/sftpFileOpen'

export default {
  name: 'SftpFileOpenerDialog',
  props: {
    visible: { type: Boolean, default: false },
    fileName: { type: String, default: '' },
  },
  emits: ['update:visible', 'select'],
  setup(props, { emit }) {
    const rememberChoice = ref(false)
    const selectingApp = ref(false)

    const canEdit = computed(() => !isKnownBinaryFile(props.fileName))
    const displayExtension = computed(() => {
      const ext = getFileExtension(props.fileName)
      return ext === 'file' ? '无扩展名' : `.${ext}`
    })

    watch(
      () => props.visible,
      (v) => {
        if (v) {
          rememberChoice.value = hasFileExtension(props.fileName)
          selectingApp.value = false
        }
      },
    )

    const emitClose = () => {
      if (selectingApp.value) return
      emit('update:visible', false)
    }

    const pickBuiltin = () => {
      emit('select', {
        openerType: 'builtin-editor',
        remember: rememberChoice.value,
      })
      emit('update:visible', false)
    }

    const pickSystemApp = async () => {
      selectingApp.value = true
      try {
        const app = await App.SelectSystemApplication()
        if (!app?.path) return
        emit('select', {
          openerType: 'system-app',
          remember: rememberChoice.value,
          systemApp: app,
        })
        emit('update:visible', false)
      } catch (e) {
        ElMessage.error(String(e))
      } finally {
        selectingApp.value = false
      }
    }

    return {
      rememberChoice,
      selectingApp,
      canEdit,
      displayExtension,
      emitClose,
      pickBuiltin,
      pickSystemApp,
    }
  },
}
</script>

<style scoped>
.opener-filename {
  margin-bottom: 12px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.opener-options {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.opener-option {
  display: block;
  width: 100%;
  text-align: left;
  padding: 12px 14px;
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
  color: var(--el-text-color-primary);
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease;
}

.opener-option:hover:not(:disabled) {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.opener-option:disabled {
  opacity: 0.6;
  cursor: wait;
}

.opener-option-title {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.3;
}

.opener-option-desc {
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.opener-remember {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 14px;
  font-size: 13px;
  color: var(--el-text-color-regular);
  cursor: pointer;
  user-select: none;
}
</style>
