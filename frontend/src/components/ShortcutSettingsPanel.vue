<template>
  <div class="shortcut-panel">
    <div class="settings-subnav">
      <button
        type="button"
        class="subnav-item"
        :class="{ active: activeTab === 'shortcuts' }"
        @click="activeTab = 'shortcuts'"
      >
        快捷键
      </button>
      <button
        type="button"
        class="subnav-item"
        :class="{ active: activeTab === 'snippets' }"
        @click="activeTab = 'snippets'"
      >
        命令片段
      </button>
    </div>

    <div v-if="activeTab === 'shortcuts'" class="panel-scroll">
      <div class="tip-bar">
        <span>修饰键为 <strong>{{ modLabel }}</strong>。点击按键区域后按下新组合即可录制，Esc 取消</span>
      </div>

      <section
        v-for="group in shortcutGroups"
        :key="group.title"
        class="bind-group"
      >
        <header class="bind-group-head">
          <span class="bind-group-title">{{ group.title }}</span>
          <span class="bind-group-count">{{ group.items.length }}</span>
        </header>
        <ul class="bind-list">
          <li
            v-for="item in group.items"
            :key="item.id"
            class="bind-row"
            :class="{ recording: recordingId === item.id }"
          >
            <div class="bind-label">
              <span class="bind-name">{{ item.label }}</span>
            </div>
            <button
              type="button"
              class="bind-capture"
              :class="{ active: recordingId === item.id }"
              :title="recordingId === item.id ? '按下组合键…' : '点击录制'"
              @click="(e) => startRecording(item.id, e)"
              @keydown="(e) => onShortcutCapture(item.id, e)"
            >
              <template v-if="recordingId === item.id">
                <span class="bind-recording">按下组合键…</span>
              </template>
              <template v-else-if="shortcutParts(shortcuts[item.id]).length">
                <kbd
                  v-for="(part, i) in shortcutParts(shortcuts[item.id])"
                  :key="`${item.id}-${i}`"
                  class="kbd"
                >{{ part }}</kbd>
              </template>
              <template v-else>
                <span class="bind-empty">未设置</span>
              </template>
            </button>
            <el-tooltip content="重置为默认" placement="top">
              <button type="button" class="bind-reset" @click="resetShortcut(item.id)">
                <el-icon :size="14"><RefreshLeft /></el-icon>
              </button>
            </el-tooltip>
          </li>
        </ul>
      </section>
    </div>

    <div v-else class="panel-scroll">
      <div class="tip-bar">
        <span>在 Shell 命令面板中快速插入。scope 填 <code>global</code> 或机器配置名</span>
      </div>

      <div class="snippet-toolbar">
        <span class="snippet-toolbar-label">共 {{ snippets.length }} 条</span>
        <el-button size="small" type="primary" plain @click="addSnippet">添加片段</el-button>
      </div>

      <div v-if="!snippets.length" class="snippet-empty">
        <p>暂无命令片段</p>
        <el-button size="small" type="primary" @click="addSnippet">添加第一条</el-button>
      </div>

      <ul v-else class="snippet-list">
        <li v-for="(s, i) in snippets" :key="s.id || i" class="snippet-card">
          <div class="snippet-card-top">
            <el-input v-model="s.name" size="small" placeholder="名称" class="sn-name" />
            <el-input v-model="s.scope" size="small" placeholder="global" class="sn-scope" />
            <el-tooltip content="删除" placement="top">
              <el-button size="small" text type="danger" circle @click="snippets.splice(i, 1)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
          <el-input
            v-model="s.command"
            type="textarea"
            :rows="2"
            resize="vertical"
            placeholder="要插入的命令内容"
            class="sn-cmd"
          />
        </li>
      </ul>
    </div>

    <div class="panel-actions icon-actions">
      <el-tooltip v-if="activeTab === 'shortcuts'" content="全部重置" placement="top">
        <el-button circle @click="resetAll">
          <el-icon><RefreshLeft /></el-icon>
        </el-button>
      </el-tooltip>
      <el-tooltip :content="activeTab === 'snippets' ? '保存片段' : '保存快捷键'" placement="top">
        <el-button type="primary" circle :loading="saving" @click="save">
          <el-icon v-if="!saving"><Check /></el-icon>
        </el-button>
      </el-tooltip>
    </div>
  </div>
</template>

<script>
import { RefreshLeft, Check, Delete } from '@element-plus/icons-vue'
import { ref, reactive, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import * as App from '../../wailsjs/go/app/App'
import {
  DEFAULT_SHORTCUTS,
  SHORTCUT_LABELS,
  SHORTCUT_GROUPS,
  mergeShortcuts,
  formatShortcut,
  bindingFromEvent,
} from '../utils/shortcuts'
import { modKeyLabel } from '../utils/platform'

export default {
  name: 'ShortcutSettingsPanel',
  components: { RefreshLeft, Check, Delete },
  props: {
    active: { type: Boolean, default: false },
  },
  setup(props) {
    const saving = ref(false)
    const recordingId = ref('')
    const activeTab = ref('shortcuts')
    const shortcuts = reactive(mergeShortcuts())
    const snippets = ref([])
    const modLabel = modKeyLabel()

    const shortcutGroups = computed(() =>
      SHORTCUT_GROUPS.map((g) => ({
        title: g.title,
        items: g.ids.map((id) => ({
          id,
          label: SHORTCUT_LABELS[id] || id,
        })),
      })),
    )

    const shortcutParts = (binding) => {
      const label = formatShortcut(binding)
      return label ? label.split('+').filter(Boolean) : []
    }

    const applyShortcutData = (data) => {
      const merged = mergeShortcuts(data)
      Object.keys(merged).forEach((id) => {
        shortcuts[id] = { ...merged[id] }
      })
      snippets.value = [...(data?.snippets || [])]
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
        if (open) {
          activeTab.value = 'shortcuts'
          recordingId.value = ''
          load()
        }
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

    const addSnippet = () => {
      snippets.value.push({
        id: `sn-${Date.now()}`,
        name: '新片段',
        command: '',
        scope: 'global',
      })
    }

    const save = async () => {
      saving.value = true
      try {
        await App.SaveShortcutSettings({ ...shortcuts, snippets: snippets.value })
        ElMessage.success(activeTab.value === 'snippets' ? '命令片段已保存' : '快捷键已保存')
      } catch (e) {
        ElMessage.error(`保存失败: ${e}`)
      } finally {
        saving.value = false
      }
    }

    return {
      saving,
      recordingId,
      activeTab,
      shortcuts,
      shortcutGroups,
      modLabel,
      shortcutParts,
      startRecording,
      onShortcutCapture,
      resetShortcut,
      resetAll,
      save,
      snippets,
      addSnippet,
    }
  },
}
</script>

<style scoped>
.shortcut-panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
  overflow: hidden;
}

/* —— 与系统设置 subnav 一致 —— */
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

.tip-bar strong {
  color: var(--app-accent-color);
  font-weight: 600;
}

.tip-bar code {
  padding: 0 4px;
  border-radius: 3px;
  font-size: 11px;
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}

/* —— 快捷键分组 —— */
.bind-group {
  margin-bottom: 14px;
  border: 1px solid var(--app-border);
  border-radius: 10px;
  background: var(--app-card-bg, var(--app-bg));
  overflow: hidden;
}

.bind-group:last-child {
  margin-bottom: 0;
}

.bind-group-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px 8px;
  border-bottom: 1px solid var(--app-border);
  background: color-mix(in srgb, var(--app-panel-bg) 70%, transparent);
}

.bind-group-title {
  font-size: 12px;
  font-weight: 650;
  letter-spacing: 0.02em;
  color: var(--app-text-secondary);
  text-transform: none;
}

.bind-group-count {
  margin-left: auto;
  font-size: 11px;
  color: var(--app-text-muted);
}

.bind-list {
  list-style: none;
  margin: 0;
  padding: 4px 0;
}

.bind-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 10px;
  padding: 8px 14px;
  transition: background 0.12s ease;
}

.bind-row:hover,
.bind-row.recording {
  background: var(--app-accent-bg);
}

.bind-label {
  min-width: 0;
}

.bind-name {
  display: block;
  font-size: 13px;
  color: var(--app-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.bind-capture {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
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

.bind-reset {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--app-text-muted);
  cursor: pointer;
  opacity: 0.55;
  transition: opacity 0.12s ease, color 0.12s ease, background 0.12s ease;
}

.bind-row:hover .bind-reset,
.bind-row.recording .bind-reset {
  opacity: 1;
}

.bind-reset:hover {
  color: var(--app-accent-color);
  background: var(--app-accent-bg);
}

/* —— 命令片段 —— */
.snippet-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.snippet-toolbar-label {
  font-size: 12px;
  color: var(--app-text-muted);
}

.snippet-empty {
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

.snippet-empty p {
  margin: 0;
}

.snippet-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.snippet-card {
  padding: 12px;
  border: 1px solid var(--app-border);
  border-radius: 10px;
  background: var(--app-card-bg, var(--app-bg));
}

.snippet-card-top {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.sn-name {
  width: 140px;
  flex-shrink: 0;
}

.sn-scope {
  width: 110px;
  flex-shrink: 0;
}

.sn-cmd {
  width: 100%;
}

.sn-cmd :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.45;
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
