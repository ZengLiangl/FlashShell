<template>
  <div class="keymap-panel">
    <div class="panel-scroll">
      <div class="tip-bar">
        <span>
          为组合键指定要发送到当前终端的字符串（优先于应用快捷键）。
          支持转义 <code>\n</code> <code>\r</code> <code>\t</code> <code>\e</code> <code>\\</code> <code>\xHH</code>。
          普通键需带修饰键（{{ modLabel }} / Alt / Shift）；F1–F12 可单独映射。
        </span>
      </div>

      <div class="keymap-toolbar">
        <span class="keymap-toolbar-label">共 {{ entries.length }} 条</span>
        <el-button size="small" type="primary" plain @click="addEntry">添加映射</el-button>
      </div>

      <div v-if="!entries.length" class="keymap-empty">
        <p>暂无按键映射</p>
        <el-button size="small" type="primary" @click="addEntry">添加第一条</el-button>
      </div>

      <ul v-else class="keymap-list">
        <li
          v-for="(item, i) in entries"
          :key="item.id"
          class="keymap-card"
          :class="{ recording: recordingId === item.id }"
        >
          <div class="keymap-card-top">
            <el-switch v-model="item.enabled" size="small" />
            <el-input v-model="item.name" size="small" placeholder="备注（可选）" class="km-name" />
            <button
              type="button"
              class="bind-capture"
              :class="{ active: recordingId === item.id }"
              :title="recordingId === item.id ? '按下组合键…' : '点击录制'"
              @click="(e) => startRecording(item.id, e)"
              @keydown="(e) => onCapture(item, e)"
            >
              <template v-if="recordingId === item.id">
                <span class="bind-recording">按下组合键…</span>
              </template>
              <template v-else-if="bindingParts(item.binding).length">
                <kbd
                  v-for="(part, pi) in bindingParts(item.binding)"
                  :key="`${item.id}-${pi}`"
                  class="kbd"
                >{{ part }}</kbd>
              </template>
              <template v-else>
                <span class="bind-empty">点击录制</span>
              </template>
            </button>
            <el-tooltip content="删除" placement="top">
              <el-button size="small" text type="danger" circle @click="removeEntry(i)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
          <el-input
            v-model="item.sendString"
            type="textarea"
            :rows="2"
            resize="vertical"
            placeholder="发送字符串，例如 ls -la\n 或 \e[A"
            class="km-send"
          />
          <div class="keymap-card-meta">
            <span class="meta-action">动作：发送字符串</span>
            <span v-if="item.sendString" class="meta-preview" :title="previewOf(item.sendString)">
              预览：{{ previewOf(item.sendString) }}
            </span>
          </div>
        </li>
      </ul>
    </div>

    <div class="panel-actions icon-actions">
      <el-tooltip content="保存按键映射" placement="top">
        <el-button type="primary" circle :loading="saving" @click="save">
          <el-icon v-if="!saving"><Check /></el-icon>
        </el-button>
      </el-tooltip>
    </div>
  </div>
</template>

<script>
import { Check, Delete } from '@element-plus/icons-vue'
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import * as App from '../../wailsjs/go/app/App'
import {
  normalizeKeyMapSettings,
  emptyKeyMapSettings,
  formatKeyMapParts,
  keymapBindingFromEvent,
  expandSendString,
} from '../utils/keymaps'
import { modKeyLabel } from '../utils/platform'

function newEntry() {
  return {
    id: `km-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
    enabled: true,
    name: '',
    action: 'sendString',
    sendString: '',
    binding: { key: '', useMod: false, useAlt: false, useShift: false },
  }
}

export default {
  name: 'KeyMapSettingsPanel',
  components: { Check, Delete },
  props: {
    active: { type: Boolean, default: false },
  },
  setup(props) {
    const saving = ref(false)
    const recordingId = ref('')
    const entries = ref([])
    const modLabel = modKeyLabel()

    const bindingParts = (binding) => formatKeyMapParts(binding)

    const previewOf = (raw) => {
      const expanded = expandSendString(raw)
      return expanded
        .replace(/\n/g, '⏎')
        .replace(/\r/g, '␍')
        .replace(/\t/g, '⇥')
        .replace(/\x1b/g, 'ESC')
        .slice(0, 80)
    }

    const load = async () => {
      try {
        const data = await App.GetKeyMapSettings()
        entries.value = normalizeKeyMapSettings(data).entries
      } catch {
        entries.value = emptyKeyMapSettings().entries
      }
    }

    watch(
      () => props.active,
      (open) => {
        if (open) {
          recordingId.value = ''
          load()
        }
      },
      { immediate: true },
    )

    const addEntry = () => {
      entries.value.push(newEntry())
    }

    const removeEntry = (index) => {
      entries.value.splice(index, 1)
      recordingId.value = ''
    }

    const startRecording = (id, e) => {
      recordingId.value = id
      e?.currentTarget?.focus?.()
    }

    const onCapture = (item, e) => {
      e.preventDefault()
      e.stopPropagation()
      if (e.key === 'Escape') {
        recordingId.value = ''
        return
      }
      const binding = keymapBindingFromEvent(e)
      if (!binding) return
      item.binding = binding
      recordingId.value = ''
    }

    const save = async () => {
      const invalid = entries.value.find((e) => e.enabled && !e.binding?.key)
      if (invalid) {
        ElMessage.warning('请先为启用的映射录制组合键')
        return
      }
      saving.value = true
      try {
        await App.SaveKeyMapSettings({ entries: entries.value })
        ElMessage.success('按键映射已保存')
      } catch (e) {
        ElMessage.error(`保存失败: ${e}`)
      } finally {
        saving.value = false
      }
    }

    return {
      saving,
      recordingId,
      entries,
      modLabel,
      bindingParts,
      previewOf,
      addEntry,
      removeEntry,
      startRecording,
      onCapture,
      save,
    }
  },
}
</script>

<style scoped>
.keymap-panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
  overflow: hidden;
}

.panel-scroll {
  flex: 1 1 0;
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 0 4px 12px 0;
}

.tip-bar {
  flex-shrink: 0;
  margin-bottom: 14px;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--app-text-muted);
  background: color-mix(in srgb, var(--app-accent-bg) 55%, transparent);
  border: 1px solid color-mix(in srgb, var(--app-accent-color) 18%, transparent);
}

.tip-bar code {
  padding: 0 4px;
  border-radius: 3px;
  font-size: 11px;
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}

.keymap-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.keymap-toolbar-label {
  font-size: 12px;
  color: var(--app-text-muted);
}

.keymap-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 36px 16px;
  border: 1px dashed var(--app-border);
  border-radius: 10px;
  color: var(--app-text-muted);
  font-size: 13px;
}

.keymap-empty p {
  margin: 0;
}

.keymap-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.keymap-card {
  padding: 12px;
  border: 1px solid var(--app-border);
  border-radius: 10px;
  background: var(--app-card-bg, var(--app-bg));
}

.keymap-card.recording {
  background: var(--app-accent-bg);
}

.keymap-card-top {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.km-name {
  width: 140px;
  flex-shrink: 0;
}

.bind-capture {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  flex: 1;
  min-width: 148px;
  min-height: 32px;
  padding: 4px 10px;
  border: 1px dashed var(--app-border);
  border-radius: 8px;
  background: var(--app-panel-bg);
  cursor: pointer;
  transition: border-color 0.12s ease, background 0.12s ease, box-shadow 0.12s ease;
}

.bind-capture:hover {
  border-color: var(--app-accent-color);
  border-style: solid;
}

.bind-capture.active {
  border-style: solid;
  border-color: var(--app-accent-color);
  background: color-mix(in srgb, var(--app-accent-bg) 80%, transparent);
  box-shadow: 0 0 0 2px var(--app-accent-bg);
}

.kbd {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 22px;
  height: 22px;
  padding: 0 6px;
  border-radius: 4px;
  border: 1px solid var(--app-border);
  background: var(--app-card-bg, var(--app-bg));
  color: var(--app-text);
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-weight: 600;
  line-height: 1;
  box-shadow: 0 1px 0 color-mix(in srgb, var(--app-border) 80%, transparent);
}

.bind-recording {
  font-size: 12px;
  color: var(--app-accent-color);
}

.bind-empty {
  font-size: 12px;
  color: var(--app-text-muted);
}

.km-send {
  width: 100%;
}

.km-send :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.45;
}

.keymap-card-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 6px;
  font-size: 11px;
  color: var(--app-text-muted);
}

.meta-preview {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
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
</style>
