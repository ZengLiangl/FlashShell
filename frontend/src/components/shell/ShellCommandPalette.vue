<template>
  <el-dialog
    v-model="visible"
    title="命令面板"
    width="560px"
    append-to-body
    class="cmd-palette-dialog"
    @open="onOpen"
  >
    <el-input
      ref="inputRef"
      v-model="query"
      placeholder="搜索历史命令、片段或机器…"
      clearable
      @keydown.down.prevent="moveSel(1)"
      @keydown.up.prevent="moveSel(-1)"
      @keydown.enter.prevent="applySelected"
      @keydown.esc.prevent="visible = false"
    />

    <el-tabs v-model="tab" class="palette-tabs">
      <el-tab-pane label="历史" name="history">
        <div class="tip-bar">
          <span>点击插入并执行；也可「仅插入」。片段不直接执行的内容同样会记入历史。</span>
        </div>

        <div class="snippet-toolbar">
          <span class="snippet-toolbar-label">共 {{ historyItems.length }} 条</span>
          <el-button
            size="small"
            type="danger"
            plain
            :disabled="!historyItems.length"
            @click="clearHistory"
          >清空历史</el-button>
        </div>

        <div v-if="!historyItems.length" class="snippet-empty">
          <p>暂无历史命令</p>
        </div>

        <ul v-else class="snippet-list">
          <li
            v-for="(item, i) in historyItems"
            :key="'h-' + i"
            class="snippet-card history-card"
            :class="{ selected: tab === 'history' && i === selectedIdx }"
            @mouseenter="selectedIdx = i"
          >
            <button type="button" class="snippet-main" @click="insert(item, true)">
              <div class="sn-cmd history-cmd" :title="item">{{ item }}</div>
            </button>
            <div class="snippet-ops icon-actions icon-actions--sm" @click.stop>
              <el-tooltip content="仅插入" placement="top">
                <el-button size="small" text circle @click="insert(item, false)">
                  <el-icon><DocumentCopy /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="插入并执行" placement="top">
                <el-button size="small" text type="primary" circle @click="insert(item, true)">
                  <el-icon><VideoPlay /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </li>
        </ul>
      </el-tab-pane>

      <el-tab-pane label="片段" name="snippets">
        <div class="tip-bar">
          <span>点击插入到终端；可新增、编辑（含快捷键）、删除。</span>
        </div>

        <div class="snippet-toolbar">
          <span class="snippet-toolbar-label">共 {{ filteredSnippets.length }} 条</span>
          <el-button size="small" type="primary" plain @click="openEditor()">添加片段</el-button>
        </div>

        <div v-if="!filteredSnippets.length" class="snippet-empty">
          <p>暂无匹配片段</p>
          <el-button size="small" type="primary" @click="openEditor()">添加第一条</el-button>
        </div>

        <ul v-else class="snippet-list">
          <li
            v-for="(s, i) in filteredSnippets"
            :key="s.id || i"
            class="snippet-card"
            :class="{ selected: tab === 'snippets' && i === selectedIdx }"
            @mouseenter="selectedIdx = i"
          >
            <button type="button" class="snippet-main" @click="insertSnippet(s)">
              <div class="snippet-card-top">
                <span class="sn-name">{{ s.name || '(未命名)' }}</span>
                <span v-if="s.execute" class="sn-badge">执行</span>
                <span v-if="bindingLabel(s.binding)" class="sn-keys">{{ bindingLabel(s.binding) }}</span>
              </div>
              <div class="sn-cmd" :title="s.command">{{ s.command || '(空命令)' }}</div>
            </button>
            <div class="snippet-ops icon-actions icon-actions--sm" @click.stop>
              <el-tooltip content="编辑" placement="top">
                <el-button size="small" text circle @click="openEditor(s)">
                  <el-icon><Edit /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button size="small" text type="danger" circle @click="removeSnippet(s)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </li>
        </ul>
      </el-tab-pane>

      <el-tab-pane label="机器" name="machines">
        <div class="tip-bar">
          <span>按名称 / IP / 用户搜索并连接机器。</span>
        </div>
        <div v-if="!filteredMachines.length" class="snippet-empty">
          <p>暂无匹配机器</p>
        </div>
        <ul v-else class="snippet-list">
          <li
            v-for="(m, i) in filteredMachines"
            :key="m.id || m.name"
            class="snippet-card"
            :class="{ selected: tab === 'machines' && i === selectedIdx }"
            @mouseenter="selectedIdx = i"
          >
            <button type="button" class="snippet-main" @click="connectMachine(m)">
              <div class="snippet-card-top">
                <span class="sn-name">{{ m.name }}</span>
                <span v-if="m.pinned" class="sn-badge">置顶</span>
              </div>
              <div class="sn-cmd" :title="machineAddr(m)">{{ machineAddr(m) }}</div>
            </button>
            <div class="snippet-ops icon-actions icon-actions--sm" @click.stop>
              <el-tooltip content="连接" placement="top">
                <el-button size="small" text type="primary" circle @click="connectMachine(m)">
                  <el-icon><VideoPlay /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </li>
        </ul>
      </el-tab-pane>
    </el-tabs>

    <el-dialog
      v-model="editorVisible"
      :title="editorForm.id ? '编辑片段' : '添加片段'"
      width="460px"
      append-to-body
      destroy-on-close
      class="snippet-editor-dialog"
      @closed="onEditorClosed"
    >
      <el-form label-position="top" size="small" @submit.prevent>
        <el-form-item label="名称">
          <el-input v-model="editorForm.name" placeholder="片段名称" maxlength="64" />
        </el-form-item>
        <el-form-item label="命令">
          <el-input
            v-model="editorForm.command"
            type="textarea"
            :rows="3"
            resize="vertical"
            placeholder="要插入或执行的命令"
            class="editor-cmd"
          />
        </el-form-item>
        <el-form-item label="作用域">
          <el-input v-model="editorForm.scope" placeholder="global 或机器配置名" />
        </el-form-item>
        <el-form-item label="快捷键">
          <div class="editor-bind-row">
            <button
              type="button"
              class="bind-capture"
              :class="{ active: recordingBind }"
              :title="recordingBind ? '按下组合键…' : '点击录制快捷键'"
              @click="(e) => startBindRecord(e)"
              @keydown="onBindCapture"
            >
              <template v-if="recordingBind">
                <span class="bind-recording">按下组合键…</span>
              </template>
              <template v-else-if="bindingParts(editorForm.binding).length">
                <kbd
                  v-for="(part, pi) in bindingParts(editorForm.binding)"
                  :key="`ed-b-${pi}`"
                  class="kbd"
                >{{ part }}</kbd>
              </template>
              <template v-else>
                <span class="bind-empty">录制快捷键</span>
              </template>
            </button>
            <el-tooltip v-if="editorForm.binding?.key" content="清除快捷键" placement="top">
              <el-button size="small" text circle @click="clearBind">
                <el-icon><RefreshLeft /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </el-form-item>
        <el-form-item label="直接执行">
          <div class="editor-execute">
            <el-switch v-model="editorForm.execute" />
            <span class="editor-execute-hint">开启后发送时自动换行执行</span>
          </div>
        </el-form-item>
        <el-form-item label="连接后执行">
          <div class="editor-execute">
            <el-switch v-model="editorForm.onConnect" />
            <span class="editor-execute-hint">会话连接成功后自动插入/执行；支持 <code v-pre>{{变量}}</code></span>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer icon-actions">
          <el-tooltip content="取消" placement="top">
            <el-button circle @click="editorVisible = false">
              <el-icon><Close /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip content="保存" placement="top">
            <el-button type="primary" circle :loading="editorSaving" @click="saveEditor">
              <el-icon v-if="!editorSaving"><Check /></el-icon>
            </el-button>
          </el-tooltip>
        </div>
      </template>
    </el-dialog>
  </el-dialog>
</template>

<script>
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Edit, Delete, Close, Check, RefreshLeft, DocumentCopy, VideoPlay } from '@element-plus/icons-vue'
import * as App from '../../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'
import { normalizeSnippets, normalizeSnippet, emptySnippetBinding, buildSnippetPayload } from '../../utils/shortcuts'
import { formatKeyMapBinding, formatKeyMapParts, keymapBindingFromEvent } from '../../utils/keymaps'
import { machineMatchesKeyword, formatMachineAddr } from '../../utils/machineGroups'

function emptyEditor() {
  return {
    id: '',
    name: '',
    command: '',
    scope: 'global',
    execute: true,
    onConnect: false,
    binding: emptySnippetBinding(),
  }
}

export default {
  name: 'ShellCommandPalette',
  components: { Edit, Delete, Close, Check, RefreshLeft, DocumentCopy, VideoPlay },
  props: {
    modelValue: { type: Boolean, default: false },
    sessionId: { type: String, default: '' },
    configName: { type: String, default: '' },
    machines: { type: Array, default: () => [] },
  },
  emits: ['update:modelValue', 'insert', 'connect'],
  setup(props, { emit }) {
    const visible = ref(false)
    const query = ref('')
    const tab = ref('history')
    const selectedIdx = ref(0)
    const inputRef = ref(null)
    const historyItems = ref([])
    const snippets = ref([])
    const editorVisible = ref(false)
    const editorSaving = ref(false)
    const editorForm = ref(emptyEditor())
    const recordingBind = ref(false)

    watch(() => props.modelValue, (v) => { visible.value = v }, { immediate: true })
    watch(visible, (v) => emit('update:modelValue', v))
    watch(query, async () => {
      selectedIdx.value = 0
      await loadHistory()
    })
    watch(tab, () => { selectedIdx.value = 0 })

    const filteredSnippets = computed(() => {
      const scope = props.configName || 'global'
      const q = query.value.toLowerCase()
      return (snippets.value || []).filter((s) => {
        if (s.scope && s.scope !== 'global' && s.scope !== scope) return false
        if (!q) return true
        return (s.name || '').toLowerCase().includes(q) || (s.command || '').toLowerCase().includes(q)
      })
    })

    const filteredMachines = computed(() => {
      const list = props.machines || []
      const q = query.value
      const filtered = String(q || '').trim()
        ? list.filter((m) => machineMatchesKeyword(m, q))
        : list
      return [...filtered].sort((a, b) => {
        if (!!a.pinned !== !!b.pinned) return a.pinned ? -1 : 1
        return String(a.name || '').localeCompare(String(b.name || ''), 'zh')
      })
    })

    const machineAddr = (m) => formatMachineAddr(m)

    const bindingLabel = (binding) => formatKeyMapBinding(binding)
    const bindingParts = (binding) => formatKeyMapParts(binding)

    const historyScope = () => props.configName || 'all'
    const recordScope = () => props.configName || 'global'

    const loadHistory = async () => {
      try {
        historyItems.value = (await App.SearchShellCommandHistory(historyScope(), query.value, 50)) || []
      } catch {
        historyItems.value = []
      }
    }

    const loadSnippets = async () => {
      try {
        const s = await App.GetShortcutSettings()
        snippets.value = normalizeSnippets(s?.snippets)
      } catch {
        snippets.value = []
      }
    }

    const onShortcutsChanged = () => {
      loadSnippets()
    }

    onMounted(() => {
      EventsOn('shortcuts:changed', onShortcutsChanged)
    })
    onBeforeUnmount(() => {
      EventsOff('shortcuts:changed')
    })

    const onOpen = async () => {
      query.value = ''
      tab.value = 'history'
      selectedIdx.value = 0
      recordingBind.value = false
      await Promise.all([loadHistory(), loadSnippets()])
      await nextTick()
      inputRef.value?.focus?.()
    }

    const moveSel = (delta) => {
      const list =
        tab.value === 'history'
          ? historyItems.value
          : tab.value === 'machines'
            ? filteredMachines.value
            : filteredSnippets.value
      if (!list.length) return
      selectedIdx.value = (selectedIdx.value + delta + list.length) % list.length
    }

    const connectMachine = (m) => {
      if (!m?.name) return
      visible.value = false
      emit('connect', m.name)
    }

    const recordHistory = async (cmd) => {
      const text = String(cmd || '').replace(/\r?\n/g, '').trim()
      if (!text) return
      try {
        await App.RecordShellCommandHistory(recordScope(), text)
      } catch (e) {
        console.warn('记录命令历史失败:', e)
      }
    }

    const insert = async (cmd, execute = true) => {
      if (cmd == null || cmd === '') return
      const raw = String(cmd)
      await recordHistory(raw)
      visible.value = false
      let text = raw
      if (execute && !/[\r\n]$/.test(text)) text += '\n'
      emit('insert', text)
    }

    const insertSnippet = async (snippet) => {
      if (!snippet) return
      const text = await buildSnippetPayload(snippet, { promptVars: true })
      if (!text) return
      await recordHistory(text.replace(/\r?\n$/, ''))
      visible.value = false
      emit('insert', text)
    }

    const clearHistory = async () => {
      try {
        await ElMessageBox.confirm('确定清空全部命令历史？此操作不可恢复。', '清空历史', {
          type: 'warning',
          confirmButtonText: '清空',
          cancelButtonText: '取消',
        })
      } catch {
        return
      }
      try {
        await App.ClearShellCommandHistory('all')
        historyItems.value = []
        selectedIdx.value = 0
        ElMessage.success('历史已清空')
      } catch (e) {
        ElMessage.error(`清空失败: ${e}`)
      }
    }

    const applySelected = () => {
      if (tab.value === 'history') {
        insert(historyItems.value[selectedIdx.value], true)
        return
      }
      if (tab.value === 'machines') {
        connectMachine(filteredMachines.value[selectedIdx.value])
        return
      }
      const s = filteredSnippets.value[selectedIdx.value]
      if (s) insertSnippet(s)
    }

    const openEditor = (snippet = null) => {
      recordingBind.value = false
      if (snippet) {
        editorForm.value = {
          id: snippet.id || '',
          name: snippet.name || '',
          command: snippet.command || '',
          scope: snippet.scope || 'global',
          execute: snippet.execute !== false,
          onConnect: !!snippet.onConnect,
          binding: snippet.binding?.key
            ? { ...snippet.binding }
            : emptySnippetBinding(),
        }
      } else {
        editorForm.value = emptyEditor()
        editorForm.value.scope = props.configName || 'global'
      }
      editorVisible.value = true
    }

    const onEditorClosed = () => {
      recordingBind.value = false
      editorForm.value = emptyEditor()
    }

    const startBindRecord = (e) => {
      recordingBind.value = true
      e?.currentTarget?.focus?.()
    }

    const onBindCapture = (e) => {
      e.preventDefault()
      e.stopPropagation()
      if (e.key === 'Escape') {
        recordingBind.value = false
        return
      }
      const binding = keymapBindingFromEvent(e)
      if (!binding) return
      editorForm.value.binding = binding
      recordingBind.value = false
    }

    const clearBind = () => {
      editorForm.value.binding = emptySnippetBinding()
      recordingBind.value = false
    }

    const persistSnippets = async (nextList) => {
      const settings = await App.GetShortcutSettings()
      const payload = {
        newWindow: settings.newWindow,
        machineConfig: settings.machineConfig,
        connectionManager: settings.connectionManager,
        envVars: settings.envVars,
        systemSettings: settings.systemSettings,
        refreshConfig: settings.refreshConfig,
        find: settings.find,
        copy: settings.copy,
        paste: settings.paste,
        clearOutput: settings.clearOutput,
        commandPalette: settings.commandPalette,
        snippets: nextList.map((s) => {
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
              key: s.binding.key,
              useMod: !!s.binding.useMod,
              useAlt: !!s.binding.useAlt,
              useShift: !!s.binding.useShift,
            }
          }
          return out
        }),
      }
      await App.SaveShortcutSettings(payload)
    }

    const saveEditor = async () => {
      const form = editorForm.value
      const name = (form.name || '').trim()
      const command = (form.command || '').trim()
      if (!name) {
        ElMessage.warning('请填写片段名称')
        return
      }
      if (!command) {
        ElMessage.warning('请填写命令内容')
        return
      }
      if (form.binding?.key && !command) {
        ElMessage.warning('已绑定快捷键的片段请填写命令内容')
        return
      }
      editorSaving.value = true
      try {
        const next = [...snippets.value]
        const binding = form.binding?.key ? { ...form.binding } : emptySnippetBinding()
        if (form.id) {
          const idx = next.findIndex((s) => s.id === form.id)
          if (idx < 0) {
            ElMessage.error('片段不存在或已删除')
            return
          }
          next[idx] = normalizeSnippet({
            ...next[idx],
            name,
            command,
            scope: (form.scope || 'global').trim() || 'global',
            execute: !!form.execute,
            onConnect: !!form.onConnect,
            binding,
          })
        } else {
          next.push(
            normalizeSnippet({
              id: `sn-${Date.now()}`,
              name,
              command,
              scope: (form.scope || 'global').trim() || 'global',
              execute: !!form.execute,
              onConnect: !!form.onConnect,
              binding,
            }),
          )
        }
        await persistSnippets(next)
        snippets.value = normalizeSnippets(next)
        editorVisible.value = false
        ElMessage.success('片段已保存')
        tab.value = 'snippets'
      } catch (e) {
        ElMessage.error(`保存失败: ${e}`)
      } finally {
        editorSaving.value = false
      }
    }

    const removeSnippet = async (snippet) => {
      try {
        await ElMessageBox.confirm(`确定删除片段「${snippet.name || '未命名'}」？`, '删除片段', {
          type: 'warning',
          confirmButtonText: '删除',
          cancelButtonText: '取消',
        })
      } catch {
        return
      }
      try {
        const next = snippets.value.filter((s) => s.id !== snippet.id)
        await persistSnippets(next)
        snippets.value = next
        selectedIdx.value = 0
        ElMessage.success('片段已删除')
      } catch (e) {
        ElMessage.error(`删除失败: ${e}`)
      }
    }

    return {
      visible,
      query,
      tab,
      selectedIdx,
      inputRef,
      historyItems,
      filteredSnippets,
      filteredMachines,
      machineAddr,
      connectMachine,
      editorVisible,
      editorSaving,
      editorForm,
      recordingBind,
      bindingLabel,
      bindingParts,
      onOpen,
      moveSel,
      applySelected,
      insert,
      insertSnippet,
      openEditor,
      onEditorClosed,
      startBindRecord,
      onBindCapture,
      clearBind,
      saveEditor,
      removeSnippet,
      clearHistory,
    }
  },
}
</script>

<style scoped>
.palette-tabs {
  margin-top: 12px;
}

.tip-bar {
  margin-bottom: 12px;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--app-text-muted);
  background: color-mix(in srgb, var(--app-accent-bg) 55%, transparent);
  border: 1px solid color-mix(in srgb, var(--app-accent-color) 18%, transparent);
}

.snippet-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
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
  padding: 28px 16px;
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
  gap: 8px;
  max-height: 280px;
  overflow: auto;
}

.snippet-card {
  display: flex;
  align-items: stretch;
  gap: 4px;
  padding: 4px;
  border: 1px solid var(--app-border);
  border-radius: 10px;
  background: var(--app-card-bg, var(--app-bg));
  transition: border-color 0.12s ease, background 0.12s ease;
}

.snippet-card:hover,
.snippet-card.selected {
  border-color: var(--app-accent-color);
  background: var(--app-accent-bg);
}

.snippet-main {
  flex: 1;
  min-width: 0;
  border: none;
  background: transparent;
  text-align: left;
  padding: 8px 10px;
  border-radius: 8px;
  cursor: pointer;
  color: inherit;
  font: inherit;
}

.snippet-card-top {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
  min-width: 0;
}

.sn-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sn-badge {
  flex-shrink: 0;
  font-size: 10px;
  font-weight: 600;
  padding: 0 5px;
  border-radius: 4px;
  color: var(--app-accent-color);
  background: color-mix(in srgb, var(--app-accent-color) 14%, transparent);
}

.sn-keys {
  flex-shrink: 0;
  margin-left: auto;
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-weight: 600;
  color: var(--app-text-muted);
  padding: 1px 6px;
  border-radius: 4px;
  border: 1px solid var(--app-border);
  background: var(--app-panel-bg);
}

.sn-cmd {
  font-size: 12px;
  color: var(--app-text-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.history-cmd {
  font-size: 13px;
  color: var(--app-text);
  white-space: pre-wrap;
  word-break: break-all;
  line-height: 1.45;
}

.snippet-ops {
  flex-shrink: 0;
  align-self: center;
  opacity: 0.4;
  padding-right: 4px;
  transition: opacity 0.12s ease;
}

.snippet-card:hover .snippet-ops,
.snippet-card.selected .snippet-ops {
  opacity: 1;
}

.editor-cmd :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.45;
}

.editor-bind-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.bind-capture {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  flex: 1;
  min-width: 0;
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

.editor-execute {
  display: flex;
  align-items: center;
  gap: 8px;
}

.editor-execute-hint {
  font-size: 12px;
  color: var(--app-text-muted);
}

.dialog-footer {
  justify-content: flex-end;
}
</style>
