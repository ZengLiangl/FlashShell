<template>
  <el-dialog
    v-model="visible"
    title="快速切换"
    width="520px"
    append-to-body
    class="quick-switcher-dialog"
    @open="onOpen"
  >
    <el-input
      ref="inputRef"
      v-model="query"
      placeholder="搜索机器、标签或动作…"
      clearable
      @keydown.down.prevent="moveSel(1)"
      @keydown.up.prevent="moveSel(-1)"
      @keydown.enter.prevent="applySelected"
      @keydown.esc.prevent="visible = false"
    />

    <div v-if="!items.length" class="qs-empty">无匹配项</div>
    <ul v-else class="qs-list" role="listbox">
      <li
        v-for="(item, i) in items"
        :key="item.id"
        class="qs-item"
        :class="{ selected: i === selectedIdx }"
        role="option"
        @mouseenter="selectedIdx = i"
        @click="runItem(item)"
      >
        <span class="qs-kind">{{ item.kindLabel }}</span>
        <div class="qs-main">
          <div class="qs-title">{{ item.title }}</div>
          <div v-if="item.subtitle" class="qs-sub">{{ item.subtitle }}</div>
        </div>
      </li>
    </ul>
  </el-dialog>
</template>

<script>
import { computed, nextTick, ref, watch } from 'vue'
import {
  formatMachineAddr,
  machineMatchesKeyword,
  normalizeMachineTags,
} from '../utils/machineGroups'

export default {
  name: 'QuickSwitcher',
  props: {
    modelValue: { type: Boolean, default: false },
    machines: { type: Array, default: () => [] },
    sessions: { type: Array, default: () => [] },
  },
  emits: [
    'update:modelValue',
    'connect',
    'open-settings',
    'open-machine-config',
    'open-shell',
    'open-command-palette',
  ],
  setup(props, { emit }) {
    const visible = ref(props.modelValue)
    const query = ref('')
    const selectedIdx = ref(0)
    const inputRef = ref(null)

    watch(
      () => props.modelValue,
      (v) => {
        visible.value = v
      },
    )
    watch(visible, (v) => emit('update:modelValue', v))

    const actionItems = [
      {
        id: 'act-shell',
        kind: 'action',
        kindLabel: '动作',
        title: '进入终端',
        subtitle: '切换到 Shell 工作区',
        keywords: 'shell terminal 终端',
        run: () => emit('open-shell'),
      },
      {
        id: 'act-machines',
        kind: 'action',
        kindLabel: '动作',
        title: '机器配置',
        subtitle: '打开连接管理',
        keywords: 'machine config 机器 配置',
        run: () => emit('open-machine-config'),
      },
      {
        id: 'act-settings',
        kind: 'action',
        kindLabel: '动作',
        title: '系统设置',
        subtitle: '主题、代理、快捷键等',
        keywords: 'settings 设置 系统',
        run: () => emit('open-settings'),
      },
      {
        id: 'act-palette',
        kind: 'action',
        kindLabel: '动作',
        title: '命令面板',
        subtitle: '历史命令与片段',
        keywords: 'command palette 命令 面板 片段',
        run: () => emit('open-command-palette'),
      },
    ]

    const sessionItems = computed(() =>
      (props.sessions || [])
        .filter((s) => s?.machineName)
        .map((s) => ({
          id: `sess-${s.machineName}`,
          kind: 'session',
          kindLabel: s.connected ? '会话' : '会话·断开',
          title: s.tabLabel || s.machineName,
          subtitle: s.configName && s.configName !== s.machineName ? s.configName : s.machineName,
          keywords: `${s.tabLabel || ''} ${s.machineName} ${s.configName || ''}`,
          run: () => emit('connect', s.configName || s.machineName),
        })),
    )

    const machineItems = computed(() =>
      (props.machines || []).map((m) => {
        const tags = normalizeMachineTags(m.tags)
        return {
          id: `m-${m.id || m.name}`,
          kind: 'machine',
          kindLabel: '机器',
          title: m.name,
          subtitle: [formatMachineAddr(m), tags.join(' · '), m.notes].filter(Boolean).join(' | '),
          keywords: `${m.name} ${formatMachineAddr(m)} ${tags.join(' ')} ${m.notes || ''} ${m.group || ''}`,
          run: () => emit('connect', m.name),
          machine: m,
        }
      }),
    )

    const items = computed(() => {
      const kw = String(query.value || '').trim().toLowerCase()
      const pool = [...sessionItems.value, ...machineItems.value, ...actionItems]
      const filtered = !kw
        ? pool
        : pool.filter((item) => {
            if (item.machine) return machineMatchesKeyword(item.machine, kw)
            return String(item.keywords || item.title || '')
              .toLowerCase()
              .includes(kw)
          })
      return filtered.slice(0, 40)
    })

    watch(items, () => {
      selectedIdx.value = 0
    })

    const onOpen = () => {
      query.value = ''
      selectedIdx.value = 0
      nextTick(() => inputRef.value?.focus?.())
    }

    const moveSel = (delta) => {
      const n = items.value.length
      if (!n) return
      selectedIdx.value = (selectedIdx.value + delta + n) % n
    }

    const runItem = (item) => {
      if (!item?.run) return
      visible.value = false
      item.run()
    }

    const applySelected = () => {
      const item = items.value[selectedIdx.value]
      if (item) runItem(item)
    }

    return {
      visible,
      query,
      selectedIdx,
      inputRef,
      items,
      onOpen,
      moveSel,
      applySelected,
      runItem,
    }
  },
}
</script>

<style scoped>
.qs-empty {
  margin-top: 16px;
  text-align: center;
  color: var(--app-text-muted);
  font-size: 13px;
  padding: 24px 0;
}

.qs-list {
  list-style: none;
  margin: 12px 0 0;
  padding: 0;
  max-height: 360px;
  overflow: auto;
}

.qs-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 12px;
  border-radius: var(--app-radius-md, 8px);
  cursor: pointer;
}

.qs-item.selected,
.qs-item:hover {
  background: var(--app-accent-bg);
}

.qs-kind {
  flex-shrink: 0;
  margin-top: 2px;
  font-size: 10px;
  line-height: 1;
  padding: 4px 6px;
  border-radius: 999px;
  color: var(--app-text-muted);
  border: 1px solid var(--app-border);
  background: var(--app-panel-bg);
}

.qs-main {
  min-width: 0;
  flex: 1;
}

.qs-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
}

.qs-sub {
  margin-top: 2px;
  font-size: 12px;
  color: var(--app-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
