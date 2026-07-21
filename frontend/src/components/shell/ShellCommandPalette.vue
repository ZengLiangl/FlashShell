<template>
  <el-dialog
    v-model="visible"
    title="命令面板"
    width="520px"
    append-to-body
    class="cmd-palette-dialog"
    @open="onOpen"
  >
    <el-input
      ref="inputRef"
      v-model="query"
      placeholder="搜索历史命令或片段…"
      clearable
      @keydown.down.prevent="moveSel(1)"
      @keydown.up.prevent="moveSel(-1)"
      @keydown.enter.prevent="applySelected"
      @keydown.esc.prevent="visible = false"
    />
    <el-tabs v-model="tab" class="palette-tabs">
      <el-tab-pane label="历史" name="history">
        <ul class="palette-list">
          <li
            v-for="(item, i) in historyItems"
            :key="'h-' + i"
            :class="{ selected: tab === 'history' && i === selectedIdx }"
            @click="insert(item)"
            @mouseenter="selectedIdx = i"
          >{{ item }}</li>
          <li v-if="!historyItems.length" class="empty">暂无历史</li>
        </ul>
      </el-tab-pane>
      <el-tab-pane label="片段" name="snippets">
        <ul class="palette-list">
          <li
            v-for="(s, i) in filteredSnippets"
            :key="s.id || i"
            :class="{ selected: tab === 'snippets' && i === selectedIdx }"
            @click="insert(s.command)"
            @mouseenter="selectedIdx = i"
          >
            <span class="sn-name">{{ s.name }}</span>
            <span class="sn-cmd">{{ s.command }}</span>
          </li>
          <li v-if="!filteredSnippets.length" class="empty">暂无片段（可在设置 → 快捷键 → 命令片段中配置）</li>
        </ul>
      </el-tab-pane>
    </el-tabs>
  </el-dialog>
</template>

<script>
import { ref, computed, watch, nextTick } from 'vue'
import * as App from '../../../wailsjs/go/app/App'

export default {
  name: 'ShellCommandPalette',
  props: {
    modelValue: { type: Boolean, default: false },
    sessionId: { type: String, default: '' },
    configName: { type: String, default: '' },
  },
  emits: ['update:modelValue', 'insert'],
  setup(props, { emit }) {
    const visible = ref(false)
    const query = ref('')
    const tab = ref('history')
    const selectedIdx = ref(0)
    const inputRef = ref(null)
    const historyItems = ref([])
    const snippets = ref([])

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

    const loadHistory = async () => {
      const scope = props.configName || 'all'
      try {
        historyItems.value = (await App.SearchShellCommandHistory(scope, query.value, 50)) || []
      } catch {
        historyItems.value = []
      }
    }

    const loadSnippets = async () => {
      try {
        const s = await App.GetShortcutSettings()
        snippets.value = s?.snippets || []
      } catch {
        snippets.value = []
      }
    }

    const onOpen = async () => {
      query.value = ''
      tab.value = 'history'
      selectedIdx.value = 0
      await Promise.all([loadHistory(), loadSnippets()])
      await nextTick()
      inputRef.value?.focus?.()
    }

    const currentList = () => (tab.value === 'history' ? historyItems.value : filteredSnippets.value.map((s) => s.command))

    const moveSel = (delta) => {
      const list = tab.value === 'history' ? historyItems.value : filteredSnippets.value
      if (!list.length) return
      selectedIdx.value = (selectedIdx.value + delta + list.length) % list.length
    }

    const insert = (cmd) => {
      if (!cmd) return
      visible.value = false
      emit('insert', cmd.endsWith('\n') ? cmd : cmd + '\n')
    }

    const applySelected = () => {
      if (tab.value === 'history') {
        insert(historyItems.value[selectedIdx.value])
      } else {
        const s = filteredSnippets.value[selectedIdx.value]
        if (s) insert(s.command)
      }
    }

    return {
      visible, query, tab, selectedIdx, inputRef,
      historyItems, filteredSnippets,
      onOpen, moveSel, applySelected, insert,
    }
  },
}
</script>

<style scoped>
.palette-tabs {
  margin-top: 12px;
}
.palette-list {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 280px;
  overflow: auto;
}
.palette-list li {
  padding: 8px 10px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  font-family: var(--app-mono-font, monospace);
}
.palette-list li.selected,
.palette-list li:hover {
  background: var(--app-hover-bg);
}
.palette-list li.empty {
  color: var(--app-text-secondary);
  cursor: default;
  font-family: inherit;
}
.sn-name {
  display: block;
  font-family: inherit;
  font-weight: 500;
  margin-bottom: 2px;
}
.sn-cmd {
  display: block;
  color: var(--app-text-secondary);
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
