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
        代码片段
      </button>
    </div>

    <div v-if="activeTab === 'shortcuts'" class="panel-scroll">
      <div class="tip-bar">
        <span>修饰键为 <strong>{{ modLabel }}</strong>。点击按键区域后按下新组合即可录制，Esc 取消</span>
      </div>
      <div v-if="shortcutConflicts.length" class="conflict-bar">
        快捷键冲突：
        <span v-for="(c, i) in shortcutConflicts" :key="i">{{ c.label }}{{ i < shortcutConflicts.length - 1 ? '；' : '' }}</span>
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

    <div v-else-if="activeTab === 'snippets'" class="panel-scroll">
      <div class="tip-bar">
        <span>
          命令面板可插入；绑定快捷键后可在 Shell 直接触发。
          「直接执行」开启时自动换行。支持 <code>\n</code> <code>\t</code> <code>\e</code> 与 <code v-pre>{{变量}}</code> 占位符。
        </span>
      </div>

      <div class="snippet-toolbar">
        <span class="snippet-toolbar-label">共 {{ snippets.length }} 条</span>
        <div class="snippet-toolbar-actions">
          <el-button size="small" plain @click="importSnippets">导入</el-button>
          <el-button size="small" plain @click="exportSnippets">导出</el-button>
          <el-button size="small" type="primary" plain @click="addSnippet">添加片段</el-button>
        </div>
      </div>

      <div v-if="!snippets.length" class="snippet-empty">
        <p>暂无代码片段</p>
        <el-button size="small" type="primary" @click="addSnippet">添加第一条</el-button>
      </div>

      <ul v-else class="snippet-list">
        <li
          v-for="(s, i) in snippets"
          :key="s.id || i"
          class="snippet-card"
          :class="{ recording: recordingId === snippetRecId(s) }"
        >
          <div class="snippet-row snippet-row-main">
            <div class="snippet-col">
              <span class="snippet-label">名称</span>
              <el-input v-model="s.name" size="small" placeholder="未命名片段" />
            </div>
            <div class="snippet-col snippet-col-scope">
              <span class="snippet-label">作用域</span>
              <el-input v-model="s.scope" size="small" placeholder="global" />
            </div>
            <el-tooltip content="删除" placement="top">
              <el-button
                class="snippet-del"
                size="small"
                text
                type="danger"
                circle
                @click="removeSnippet(i)"
              >
                <el-icon><Delete /></el-icon>
              </el-button>
            </el-tooltip>
          </div>

          <div class="snippet-row">
            <div class="snippet-col snippet-col-full">
              <span class="snippet-label">命令</span>
              <el-input
                v-model="s.command"
                type="textarea"
                :rows="2"
                resize="vertical"
                placeholder="例如 ls -la 或 git status"
                class="sn-cmd"
              />
            </div>
          </div>

          <div class="snippet-footer">
            <div class="snippet-footer-bind">
              <span class="snippet-label">快捷键</span>
              <button
                type="button"
                class="bind-capture sn-bind"
                :class="{ active: recordingId === snippetRecId(s) }"
                :title="recordingId === snippetRecId(s) ? '按下组合键…' : '点击录制'"
                @click="(e) => startRecording(snippetRecId(s), e)"
                @keydown="(e) => onSnippetCapture(s, e)"
              >
                <template v-if="recordingId === snippetRecId(s)">
                  <span class="bind-recording">按下组合键…</span>
                </template>
                <template v-else-if="bindingParts(s.binding).length">
                  <kbd
                    v-for="(part, pi) in bindingParts(s.binding)"
                    :key="`${s.id}-b-${pi}`"
                    class="kbd"
                  >{{ part }}</kbd>
                </template>
                <template v-else>
                  <span class="bind-empty">点击录制</span>
                </template>
              </button>
              <el-tooltip v-if="s.binding?.key" content="清除快捷键" placement="top">
                <button type="button" class="bind-reset" @click="clearSnippetBinding(s)">
                  <el-icon :size="14"><RefreshLeft /></el-icon>
                </button>
              </el-tooltip>
            </div>
            <div class="snippet-footer-exec">
              <el-switch v-model="s.execute" size="small" />
              <span class="sn-execute-label">直接执行</span>
            </div>
            <div class="snippet-footer-exec">
              <el-switch v-model="s.onConnect" size="small" />
              <span class="sn-execute-label">连接后自动执行</span>
            </div>
          </div>
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
  normalizeSnippets,
  emptySnippetBinding,
  normalizeSnippet,
  findShortcutConflicts,
} from '../utils/shortcuts'
import { formatKeyMapParts, keymapBindingFromEvent } from '../utils/keymaps'
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

    const shortcutConflicts = computed(() => findShortcutConflicts(shortcuts))

    const shortcutParts = (binding) => {
      const label = formatShortcut(binding)
      return label ? label.split('+').filter(Boolean) : []
    }

    const bindingParts = (binding) => formatKeyMapParts(binding)

    const snippetRecId = (s) => `sn-bind:${s.id}`

    const applyShortcutData = (data) => {
      const merged = mergeShortcuts(data)
      Object.keys(merged).forEach((id) => {
        shortcuts[id] = { ...merged[id] }
      })
      snippets.value = normalizeSnippets(data?.snippets)
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

    const onSnippetCapture = (snippet, e) => {
      e.preventDefault()
      e.stopPropagation()
      if (e.key === 'Escape') {
        recordingId.value = ''
        return
      }
      const binding = keymapBindingFromEvent(e)
      if (!binding) return
      snippet.binding = binding
      recordingId.value = ''
    }

    const clearSnippetBinding = (snippet) => {
      snippet.binding = emptySnippetBinding()
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
      snippets.value.push(
        normalizeSnippet({
          id: `sn-${Date.now()}`,
          name: '新片段',
          command: '',
          scope: 'global',
          execute: true,
        }),
      )
    }

    const exportSnippets = () => {
      const blob = new Blob([JSON.stringify(snippets.value, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'flashdock-snippets.json'
      a.click()
      URL.revokeObjectURL(url)
      ElMessage.success('已导出片段')
    }

    const importSnippets = () => {
      const input = document.createElement('input')
      input.type = 'file'
      input.accept = '.json,application/json'
      input.onchange = async () => {
        const file = input.files?.[0]
        if (!file) return
        try {
          const text = await file.text()
          const parsed = JSON.parse(text)
          const list = Array.isArray(parsed) ? parsed : parsed?.snippets
          if (!Array.isArray(list)) throw new Error('无效的片段文件')
          const imported = normalizeSnippets(list)
          snippets.value = [...snippets.value, ...imported.map((s) => ({
            ...s,
            id: s.id || `sn-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`,
          }))]
          ElMessage.success(`已导入 ${imported.length} 条片段`)
        } catch (e) {
          ElMessage.error('导入失败: ' + e)
        }
      }
      input.click()
    }

    const removeSnippet = (index) => {
      snippets.value.splice(index, 1)
      recordingId.value = ''
    }

    const save = async () => {
      if (activeTab.value === 'snippets') {
        const invalid = snippets.value.find((s) => s.binding?.key && !(s.command || '').trim())
        if (invalid) {
          ElMessage.warning('已绑定快捷键的片段请填写命令内容')
          return
        }
      }
      saving.value = true
      try {
        const payload = {
          newWindow: { ...shortcuts.newWindow },
          machineConfig: { ...shortcuts.machineConfig },
          connectionManager: { ...shortcuts.connectionManager },
          envVars: { ...shortcuts.envVars },
          systemSettings: { ...shortcuts.systemSettings },
          refreshConfig: { ...shortcuts.refreshConfig },
          find: { ...shortcuts.find },
          copy: { ...shortcuts.copy },
          paste: { ...shortcuts.paste },
          clearOutput: { ...shortcuts.clearOutput },
          commandPalette: { ...shortcuts.commandPalette },
          paneZoom: { ...shortcuts.paneZoom },
          nextTab: { ...shortcuts.nextTab },
          prevTab: { ...shortcuts.prevTab },
          closeTab: { ...shortcuts.closeTab },
          toggleBroadcast: { ...shortcuts.toggleBroadcast },
          openSftp: { ...shortcuts.openSftp },
          openLocalShell: { ...shortcuts.openLocalShell },
          splitFocusLeft: { ...shortcuts.splitFocusLeft },
          splitFocusRight: { ...shortcuts.splitFocusRight },
          splitFocusUp: { ...shortcuts.splitFocusUp },
          splitFocusDown: { ...shortcuts.splitFocusDown },
          snippets: snippets.value.map((s) => {
            const out = {
              id: s.id,
              name: s.name,
              command: s.command,
              scope: s.scope || 'global',
              execute: !!s.execute,
              onConnect: !!s.onConnect,
            }
            if (s.binding?.key) {
              out.binding = {
                key: String(s.binding.key),
                useMod: !!s.binding.useMod,
                useAlt: !!s.binding.useAlt,
                useShift: !!s.binding.useShift,
              }
            }
            return out
          }),
        }
        await App.SaveShortcutSettings(payload)
        // 保存后回读，确保面板与运行时一致
        applyShortcutData(await App.GetShortcutSettings())
        ElMessage.success(activeTab.value === 'snippets' ? '代码片段已保存' : '快捷键已保存')
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
      shortcutConflicts,
      modLabel,
      shortcutParts,
      bindingParts,
      snippetRecId,
      startRecording,
      onShortcutCapture,
      onSnippetCapture,
      clearSnippetBinding,
      resetShortcut,
      resetAll,
      save,
      snippets,
      addSnippet,
      importSnippets,
      exportSnippets,
      removeSnippet,
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

.conflict-bar {
  margin: -6px 0 14px;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-color-warning-dark-2, #b88230);
  background: color-mix(in srgb, var(--el-color-warning, #e6a23c) 14%, transparent);
  border: 1px solid color-mix(in srgb, var(--el-color-warning, #e6a23c) 28%, transparent);
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
.bind-row.recording .bind-reset,
.snippet-card:hover .bind-reset,
.snippet-card.recording .bind-reset {
  opacity: 1;
}

.bind-reset:hover {
  color: var(--app-accent-color);
  background: var(--app-accent-bg);
}

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

.snippet-toolbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
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
  gap: 12px;
}

.snippet-card {
  padding: 12px 14px;
  border: 1px solid var(--app-border);
  border-radius: 10px;
  background: var(--app-card-bg, var(--app-bg));
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: border-color 0.12s ease, background 0.12s ease;
}

.snippet-card:hover {
  border-color: color-mix(in srgb, var(--app-accent-color) 45%, var(--app-border));
}

.snippet-card.recording {
  border-color: var(--app-accent-color);
  background: var(--app-accent-bg);
}

.snippet-row {
  display: flex;
  align-items: flex-end;
  gap: 10px;
}

.snippet-row-main {
  align-items: flex-end;
}

.snippet-col {
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-width: 0;
  flex: 1;
}

.snippet-col-scope {
  flex: 0 0 140px;
}

.snippet-col-full {
  flex: 1;
}

.snippet-label {
  font-size: 11px;
  color: var(--app-text-muted);
  font-weight: 500;
  line-height: 1;
  letter-spacing: 0.02em;
}

.snippet-del {
  flex-shrink: 0;
  margin-bottom: 1px;
  opacity: 0.55;
  transition: opacity 0.12s ease;
}

.snippet-card:hover .snippet-del {
  opacity: 1;
}

.sn-cmd {
  width: 100%;
}

.sn-cmd :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.45;
}

.snippet-footer {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
  margin-top: 2px;
  padding-top: 10px;
  border-top: 1px solid color-mix(in srgb, var(--app-border) 70%, transparent);
}

.snippet-footer-bind {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.snippet-footer-bind .snippet-label {
  flex-shrink: 0;
  padding-bottom: 9px;
}

.sn-bind {
  flex: 1;
  min-width: 120px;
  max-width: 280px;
}

.snippet-footer-exec {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  padding-bottom: 4px;
}

.sn-execute-label {
  font-size: 12px;
  color: var(--app-text-secondary);
  white-space: nowrap;
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
