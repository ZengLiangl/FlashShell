<template>
  <div class="shortcut-panel">
    <div class="shortcut-section">
      <p class="shortcut-hint">
        修饰键按当前系统显示为 <strong>{{ modLabel }}</strong>。点击右侧按键区域后按下新组合即可修改
      </p>
      <div class="shortcut-grid">
        <div
          v-for="item in shortcutItems"
          :key="item.id"
          class="shortcut-row"
          :class="{ recording: recordingId === item.id }"
        >
          <div class="shortcut-meta">
            <span class="shortcut-name">{{ item.label }}</span>
            <span class="shortcut-id">{{ item.id }}</span>
          </div>
          <button
            type="button"
            class="shortcut-key"
            :class="{ active: recordingId === item.id }"
            @click="(e) => startRecording(item.id, e)"
            @keydown="(e) => onShortcutCapture(item.id, e)"
          >
            <template v-if="recordingId === item.id">按下组合键…</template>
            <template v-else>{{ formatShortcut(shortcuts[item.id]) || '未设置' }}</template>
          </button>
          <el-button size="small" text @click="resetShortcut(item.id)">重置</el-button>
        </div>
      </div>
    </div>

    <div class="panel-actions">
      <el-button @click="resetAll">全部重置</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存快捷键</el-button>
    </div>
  </div>
</template>

<script>
import { ref, reactive, watch } from 'vue'
import { ElMessage } from 'element-plus'
import * as App from '../../wailsjs/go/app/App'
import {
  DEFAULT_SHORTCUTS,
  SHORTCUT_LABELS,
  mergeShortcuts,
  formatShortcut,
  bindingFromEvent,
} from '../utils/shortcuts'
import { modKeyLabel } from '../utils/platform'

export default {
  name: 'ShortcutSettingsPanel',
  props: {
    active: { type: Boolean, default: false },
  },
  setup(props) {
    const saving = ref(false)
    const recordingId = ref('')
    const shortcuts = reactive(mergeShortcuts())
    const shortcutItems = Object.keys(DEFAULT_SHORTCUTS).map((id) => ({
      id,
      label: SHORTCUT_LABELS[id] || id,
    }))
    const modLabel = modKeyLabel()

    const applyShortcutData = (data) => {
      const merged = mergeShortcuts(data)
      Object.keys(merged).forEach((id) => {
        shortcuts[id] = { ...merged[id] }
      })
    }

    const load = async () => {
      try {
        applyShortcutData(await App.GetShortcutSettings())
      } catch {
        applyShortcutData()
      }
    }

    watch(
      () => props.active,
      (open) => {
        if (open) load()
      },
      { immediate: true },
    )

    const startRecording = (id, e) => {
      recordingId.value = id
      e?.currentTarget?.focus?.()
    }

    const onShortcutCapture = (id, e) => {
      e.preventDefault()
      e.stopPropagation()
      if (e.key === 'Escape') {
        recordingId.value = ''
        return
      }
      const binding = bindingFromEvent(e)
      if (!binding) return
      shortcuts[id] = binding
      recordingId.value = ''
    }

    const resetShortcut = (id) => {
      shortcuts[id] = { ...DEFAULT_SHORTCUTS[id] }
      recordingId.value = ''
    }

    const resetAll = () => {
      Object.keys(DEFAULT_SHORTCUTS).forEach((id) => {
        shortcuts[id] = { ...DEFAULT_SHORTCUTS[id] }
      })
      recordingId.value = ''
    }

    const save = async () => {
      saving.value = true
      try {
        await App.SaveShortcutSettings({ ...shortcuts })
        ElMessage.success('快捷键已保存')
      } catch (e) {
        ElMessage.error(`保存失败: ${e}`)
      } finally {
        saving.value = false
      }
    }

    return {
      saving,
      recordingId,
      shortcuts,
      shortcutItems,
      modLabel,
      formatShortcut,
      startRecording,
      onShortcutCapture,
      resetShortcut,
      resetAll,
      save,
    }
  },
}
</script>

<style scoped>
.shortcut-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 360px;
}

.shortcut-section {
  padding: 12px 14px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-bg);
}

.shortcut-hint {
  margin: 0 0 12px;
  font-size: 12px;
  color: var(--app-text-muted);
  line-height: 1.5;
}

.shortcut-hint code {
  padding: 0 4px;
  border-radius: 3px;
  background: var(--app-accent-bg);
  font-size: 11px;
}

.shortcut-grid {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.shortcut-row {
  display: grid;
  grid-template-columns: 1fr minmax(140px, 180px) auto;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 6px;
  transition: background 0.15s ease;
}

.shortcut-row:hover,
.shortcut-row.recording {
  background: var(--app-panel-bg);
}

.shortcut-meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.shortcut-name {
  font-size: 13px;
  color: var(--app-text);
}

.shortcut-id {
  font-size: 11px;
  color: var(--app-text-muted);
}

.shortcut-key {
  justify-self: stretch;
  min-height: 32px;
  padding: 0 12px;
  border: 1px solid var(--app-border);
  border-radius: 6px;
  background: var(--app-card-bg, var(--app-panel-bg));
  color: var(--app-text);
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  cursor: pointer;
  text-align: center;
}

.shortcut-key:hover {
  border-color: var(--app-accent-color);
}

.shortcut-key.active {
  border-color: var(--app-accent-color);
  color: var(--app-accent-color);
  box-shadow: 0 0 0 2px var(--app-accent-bg);
}

.panel-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: auto;
  padding-top: 12px;
  border-top: 1px solid var(--app-border);
}
</style>
