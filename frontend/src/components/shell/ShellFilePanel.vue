<template>
  <div class="shell-file-panel" :class="{ collapsed: !expanded }">
    <div
      v-if="expanded"
      class="height-handle"
      title="拖动调整高度"
      @mousedown="startHeightResize"
    />
    <div v-if="searchVisible" class="search-bar">
      <el-input
        ref="searchInputRef"
        v-model="localSearchQuery"
        size="small"
        placeholder="搜索 (Enter 下一个, Shift+Enter 上一个)"
        clearable
        @keydown.enter.exact.prevent="emit('search-next')"
        @keydown.enter.shift.exact.prevent="emit('search-prev')"
        @keydown.esc.prevent="emit('close-search')"
      />
      <span v-if="localSearchQuery" class="search-count">{{ matchSummary }}</span>
      <div class="search-actions">
        <button type="button" class="search-icon-btn" title="上一个" @click="emit('search-prev')">
          <el-icon :size="14"><CaretTop /></el-icon>
        </button>
        <button type="button" class="search-icon-btn" title="下一个" @click="emit('search-next')">
          <el-icon :size="14"><CaretBottom /></el-icon>
        </button>
        <span class="search-sep" aria-hidden="true"></span>
        <button type="button" class="search-icon-btn search-close" title="关闭" @click="emit('close-search')">
          <el-icon :size="14"><Close /></el-icon>
        </button>
      </div>
    </div>
    <div class="file-toolbar">
      <div class="toolbar-left">
        <el-button size="small" :type="expanded ? 'primary' : 'default'" @click="toggle">
          <el-icon><FolderOpened /></el-icon>
          文件
        </el-button>
        <template v-if="expanded">
          <el-button
            size="small"
            text
            class="parent-btn"
            title="返回上级目录"
            :disabled="!canGoUp"
            @click="goParent"
          >
            <el-icon :size="16"><ArrowUp /></el-icon>
          </el-button>
          <el-input
            v-model="pathDraft"
            class="cwd-input"
            size="small"
            :title="cwd"
            placeholder="/"
            @keydown.enter.exact.prevent="submitPathDraft"
            @blur="syncPathDraftFromCwd"
          />
          <el-checkbox v-model="showHidden" size="small" @change="reload">显示隐藏文件</el-checkbox>
          <el-button size="small" text :loading="loading" @click="reload">刷新目录</el-button>
        </template>
      </div>
      <div class="toolbar-right">
        <el-button size="small" @click="emit('toggle-search')">
          <el-icon><Search /></el-icon>
          搜索
        </el-button>
        <el-button size="small" @click="emit('clear')">
          <el-icon><Delete /></el-icon>
          清空
        </el-button>
        <el-button size="small" @click="emit('refresh')">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <div v-if="expanded" class="file-body" :style="{ height: bodyHeight + 'px' }">
      <div class="tree-pane" :style="{ width: treeWidth + 'px' }">
        <el-tree
          :key="treeRenderKey"
          ref="treeRef"
          :data="treeData"
          :props="{ label: 'name', children: 'children' }"
          node-key="path"
          highlight-current
          :current-node-key="cwd"
          :default-expanded-keys="expandedKeys"
          :expand-on-click-node="false"
          @node-click="onTreeClick"
          @node-expand="onNodeExpand"
        />
      </div>
      <div
        class="split-handle"
        title="拖动调整宽度"
        @mousedown="startResize"
      ></div>
      <div class="list-pane" @click="closeMenu">
        <div v-if="error" class="error">{{ error }}</div>
        <el-table
          v-else
          :data="entries"
          size="small"
          height="100%"
          v-loading="loading"
          empty-text="空目录"
          @row-dblclick="onOpen"
          @row-contextmenu="onContextMenu"
        >
          <el-table-column prop="name" label="文件名" min-width="160" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="name-cell">
                <el-icon class="icon">
                  <Folder v-if="row.isDir" />
                  <Document v-else />
                </el-icon>
                {{ row.name }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="大小" width="90" align="right">
            <template #default="{ row }">
              {{ row.isDir ? '-' : formatSize(row.size) }}
            </template>
          </el-table-column>
          <el-table-column prop="type" label="类型" width="70" />
          <el-table-column label="修改时间" width="150">
            <template #default="{ row }">
              {{ formatTime(row.modTime) }}
            </template>
          </el-table-column>
          <el-table-column prop="mode" label="权限" width="110" show-overflow-tooltip />
          <el-table-column label="用户:组" width="120" show-overflow-tooltip>
            <template #default="{ row }">
              {{ formatOwnerGroup(row) }}
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <ul
      v-if="ctx.visible"
      class="ctx-menu"
      :style="{ left: ctx.x + 'px', top: ctx.y + 'px' }"
      @click.stop
    >
      <li @click="copyPath">复制路径</li>
      <li class="danger" @click="deleteEntry">删除</li>
    </ul>
  </div>
</template>

<script>
import { computed, nextTick, onUnmounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as App from '../../../wailsjs/go/app/App'

export default {
  name: 'ShellFilePanel',
  props: {
    machineName: { type: String, default: '' },
    cwdHint: { type: String, default: '' },
    searchVisible: { type: Boolean, default: false },
    searchQuery: { type: String, default: '' },
    matchSummary: { type: String, default: '' },
  },
  emits: [
    'layout-change',
    'cwd-change',
    'clear',
    'refresh',
    'toggle-search',
    'search-next',
    'search-prev',
    'close-search',
    'update:searchQuery',
  ],
  setup(props, { emit, expose }) {
    const HEIGHT_KEY = 'shell.sftpBodyHeight'
    const DEFAULT_BODY_HEIGHT = 220
    const MIN_BODY_HEIGHT = 120
    const MAX_BODY_RATIO = 0.65

    const expanded = ref(false)
    const showHidden = ref(false)
    const cwd = ref('')
    const pathDraft = ref('')
    const entries = ref([])
    const loading = ref(false)
    const error = ref('')
    const treeRoot = ref([])
    const treeWidth = ref(220)
    const bodyHeight = ref(readBodyHeight())
    const treeRef = ref(null)
    const treeRenderKey = ref(0)
    const expandedKeys = ref(['/'])
    const ctx = reactive({ visible: false, x: 0, y: 0, row: null })
    const localSearchQuery = ref(props.searchQuery)
    const searchInputRef = ref(null)
    let pwdTimer = null
    let resizing = false
    let navSeq = 0

    watch(() => props.searchQuery, (v) => { localSearchQuery.value = v })
    watch(localSearchQuery, (v) => emit('update:searchQuery', v))
    watch(() => props.searchVisible, async (visible) => {
      if (visible) {
        await nextTick()
        searchInputRef.value?.focus?.()
      }
      notifyLayout()
    })

    const treeData = computed(() => treeRoot.value)
    const canGoUp = computed(() => {
      const p = cwd.value
      return !!p && p !== '/' && p !== '.'
    })

    function readBodyHeight() {
      const n = Number(localStorage.getItem(HEIGHT_KEY))
      if (Number.isFinite(n) && n >= MIN_BODY_HEIGHT) return n
      return DEFAULT_BODY_HEIGHT
    }

    const notifyLayout = () => {
      emit('layout-change')
    }

    const normalizeAbs = (p) => {
      let s = String(p || '').trim()
      if (!s) return '/'
      if (!s.startsWith('/')) s = `/${s}`
      if (s.length > 1) s = s.replace(/\/+$/, '')
      return s || '/'
    }

    const basename = (p) => {
      const s = normalizeAbs(p)
      if (s === '/') return '/'
      const i = s.lastIndexOf('/')
      return i >= 0 ? s.slice(i + 1) : s
    }

    const ancestorPaths = (abs) => {
      const s = normalizeAbs(abs)
      if (s === '/') return ['/']
      const parts = s.split('/').filter(Boolean)
      const out = ['/']
      let cur = ''
      for (const part of parts) {
        cur += `/${part}`
        out.push(cur)
      }
      return out
    }

    const formatSize = (n) => {
      if (n == null) return '-'
      const units = ['B', 'K', 'M', 'G', 'T']
      let v = n
      let i = 0
      while (v >= 1024 && i < units.length - 1) {
        v /= 1024
        i++
      }
      return i === 0 ? `${v}${units[i]}` : `${v.toFixed(1)}${units[i]}`
    }

    const formatTime = (ts) => {
      if (!ts) return '-'
      const d = new Date(ts * 1000)
      const pad = (n) => String(n).padStart(2, '0')
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
    }

    const formatOwnerGroup = (row) => {
      const o = row.owner || '-'
      const g = row.group || '-'
      return `${o}:${g}`
    }

    const sortEntries = (list) => {
      return (list || []).slice().sort((a, b) => {
        if (!!a.isDir !== !!b.isDir) return a.isDir ? -1 : 1
        return String(a.name || '').localeCompare(String(b.name || ''), undefined, {
          sensitivity: 'base',
          numeric: true,
        })
      })
    }

    const listDir = async (dir) => {
      if (!props.machineName) return []
      const list = await App.ListShellFiles(props.machineName, dir, showHidden.value) || []
      return sortEntries(list)
    }

    const listDirNodes = async (dir) => {
      const list = await listDir(dir)
      return list.filter((e) => e.isDir).map((e) => ({
        name: e.name,
        path: normalizeAbs(e.path),
        isDir: true,
        children: [],
        _loaded: false,
      }))
    }

    const ensureHome = async () => {
      if (props.cwdHint && props.cwdHint.startsWith('/')) {
        return normalizeAbs(props.cwdHint)
      }
      try {
        return normalizeAbs(await App.GetShellRemoteHome(props.machineName))
      } catch {
        return '/'
      }
    }

    /** 沿当前路径构建并展开目录树 */
    const syncTreeToPath = async (abs) => {
      const target = normalizeAbs(abs)
      const chain = ancestorPaths(target)
      const rootChildren = await listDirNodes('/')
      const root = {
        name: '/',
        path: '/',
        isDir: true,
        children: rootChildren,
        _loaded: true,
      }

      let parent = root
      for (let i = 1; i < chain.length; i++) {
        const want = chain[i]
        let child = (parent.children || []).find((c) => c.path === want)
        if (!child) {
          child = {
            name: basename(want),
            path: want,
            isDir: true,
            children: [],
            _loaded: false,
          }
          parent.children = [...(parent.children || []), child]
        }
        if (!child._loaded) {
          try {
            child.children = await listDirNodes(want)
            child._loaded = true
          } catch {
            child.children = []
            child._loaded = true
          }
        }
        parent = child
      }

      treeRoot.value = [root]
      expandedKeys.value = chain
      treeRenderKey.value += 1
      await nextTick()
      treeRef.value?.setCurrentKey?.(target)
    }

    const reloadList = async () => {
      if (!props.machineName || !expanded.value || !cwd.value) return
      loading.value = true
      error.value = ''
      try {
        entries.value = await listDir(cwd.value)
      } catch (e) {
        error.value = String(e)
        entries.value = []
      } finally {
        loading.value = false
      }
    }

    const setCwd = async (next, { rebuildTree = true } = {}) => {
      const abs = normalizeAbs(next)
      cwd.value = abs
      pathDraft.value = abs
      emit('cwd-change', abs)
      await reloadList()
      if (rebuildTree) {
        try {
          await syncTreeToPath(abs)
        } catch (e) {
          console.warn('同步目录树失败:', e)
        }
      }
    }

    const syncPathDraftFromCwd = () => {
      pathDraft.value = cwd.value || '/'
    }

    const submitPathDraft = async () => {
      if (!props.machineName || !expanded.value) return
      const raw = String(pathDraft.value || '').trim()
      if (!raw) {
        pathDraft.value = cwd.value || '/'
        return
      }
      try {
        const base = cwd.value || await ensureHome()
        const resolved = normalizeAbs(await App.ResolveShellPath(props.machineName, base, raw))
        const exists = await App.ShellDirExists(props.machineName, resolved)
        if (!exists) {
          ElMessage.warning(`目录不存在：${resolved}`)
          pathDraft.value = cwd.value || '/'
          return
        }
        if (resolved === normalizeAbs(cwd.value)) {
          pathDraft.value = resolved
          return
        }
        await setCwd(resolved)
      } catch (e) {
        ElMessage.error('路径无效: ' + e)
        pathDraft.value = cwd.value || '/'
      }
    }

    /**
     * 跳转到目标目录；若不存在则保留原目录。
     * target 可为相对/绝对/~；若已是校验过的绝对路径可直接传入。
     */
    const navigateTo = async (target, { alreadyAbsolute = false, trustAbsolute = false } = {}) => {
      if (!props.machineName || !expanded.value) return
      const seq = ++navSeq
      const prev = cwd.value || await ensureHome()
      try {
        let next
        if (alreadyAbsolute && typeof target === 'string' && target.startsWith('/')) {
          const abs = normalizeAbs(target)
          if (trustAbsolute) {
            next = abs
          } else {
            const exists = await App.ShellDirExists(props.machineName, abs)
            next = exists ? abs : normalizeAbs(prev)
            if (!exists) {
              ElMessage.warning(`目录不存在，已保留：${normalizeAbs(prev)}`)
            }
          }
        } else {
          next = normalizeAbs(await App.ApplyShellCd(props.machineName, prev, target))
          if (next === normalizeAbs(prev)) {
            const resolved = normalizeAbs(await App.ResolveShellPath(props.machineName, prev, target))
            if (resolved !== next) {
              ElMessage.warning(`目录不存在，已保留：${next}`)
            }
          }
        }
        if (seq !== navSeq) return
        if (next === normalizeAbs(cwd.value)) {
          await reloadList()
          return
        }
        await setCwd(next)
      } catch (e) {
        if (seq === navSeq) error.value = String(e)
      }
    }

    /** 供父组件在终端 cwd 变化后直接调用 */
    const applyCwdHint = async (hint) => {
      if (!hint || !String(hint).startsWith('/')) return
      const abs = normalizeAbs(hint)
      // 未展开时也记下路径，展开后使用
      if (!expanded.value) {
        if (abs !== normalizeAbs(cwd.value)) {
          cwd.value = abs
          pathDraft.value = abs
          emit('cwd-change', abs)
        }
        return
      }
      // 同目录：不重复刷树/列表（真实 cwd 每个 prompt 都会上报）
      if (abs === normalizeAbs(cwd.value)) {
        return
      }
      await setCwd(abs)
    }

    const onTreeClick = async (data) => {
      if (!data?.path) return
      await navigateTo(data.path, { alreadyAbsolute: true })
    }

    const onNodeExpand = async (data) => {
      if (!data || data._loaded) return
      try {
        data.children = await listDirNodes(data.path)
        data._loaded = true
      } catch {
        data.children = []
        data._loaded = true
      }
    }

    const onOpen = async (row) => {
      if (!row?.isDir) return
      await navigateTo(row.path, { alreadyAbsolute: true })
    }

    const goParent = async () => {
      if (!canGoUp.value) return
      await navigateTo('..')
    }

    const reload = async () => {
      if (!props.machineName || !expanded.value) return
      if (!cwd.value) {
        await setCwd(await ensureHome())
        return
      }
      await setCwd(cwd.value)
    }

    watch(() => props.cwdHint, async (hint) => {
      await applyCwdHint(hint)
    })

    const calibrateList = async () => {
      if (!props.machineName || !expanded.value || !cwd.value) return
      try {
        entries.value = await listDir(cwd.value)
        error.value = ''
      } catch (e) {
        // 当前目录突然不存在时保留展示，仅提示
        error.value = String(e)
      }
    }

    const startPwdTimer = () => {
      stopPwdTimer()
      pwdTimer = setInterval(calibrateList, 2000)
    }
    const stopPwdTimer = () => {
      if (pwdTimer) {
        clearInterval(pwdTimer)
        pwdTimer = null
      }
    }

    const toggle = async () => {
      expanded.value = !expanded.value
      if (expanded.value) {
        await setCwd(cwd.value || await ensureHome())
        startPwdTimer()
      } else {
        stopPwdTimer()
        closeMenu()
      }
      await nextTick()
      notifyLayout()
    }

    const startHeightResize = (e) => {
      e.preventDefault()
      const startY = e.clientY
      const startH = bodyHeight.value
      const onMove = (ev) => {
        const delta = startY - ev.clientY
        const maxH = Math.floor(window.innerHeight * MAX_BODY_RATIO)
        bodyHeight.value = Math.min(maxH, Math.max(MIN_BODY_HEIGHT, startH + delta))
        notifyLayout()
      }
      const onUp = () => {
        window.removeEventListener('mousemove', onMove)
        window.removeEventListener('mouseup', onUp)
        localStorage.setItem(HEIGHT_KEY, String(bodyHeight.value))
        notifyLayout()
      }
      window.addEventListener('mousemove', onMove)
      window.addEventListener('mouseup', onUp)
    }

    const startResize = (e) => {
      resizing = true
      const startX = e.clientX
      const startW = treeWidth.value
      const onMove = (ev) => {
        if (!resizing) return
        const next = startW + (ev.clientX - startX)
        treeWidth.value = Math.max(140, Math.min(480, next))
      }
      const onUp = () => {
        resizing = false
        window.removeEventListener('mousemove', onMove)
        window.removeEventListener('mouseup', onUp)
      }
      window.addEventListener('mousemove', onMove)
      window.addEventListener('mouseup', onUp)
    }

    const onContextMenu = (row, _col, event) => {
      event.preventDefault()
      ctx.row = row
      ctx.x = event.clientX
      ctx.y = event.clientY
      ctx.visible = true
    }

    const closeMenu = () => {
      ctx.visible = false
      ctx.row = null
    }

    const copyPath = async () => {
      const path = ctx.row?.path
      closeMenu()
      if (!path) return
      try {
        await navigator.clipboard.writeText(path)
        ElMessage.success('已复制路径')
      } catch {
        ElMessage.error('复制失败')
      }
    }

    const deleteEntry = async () => {
      const row = ctx.row
      if (!row || !props.machineName) return
      // 先关菜单再弹确认，避免被右键菜单挡住
      closeMenu()
      try {
        await ElMessageBox.confirm(
          `确定删除「${row.name}」？${row.isDir ? '（将递归删除目录）' : ''}\n此操作不可恢复。`,
          '删除确认',
          {
            type: 'warning',
            confirmButtonText: '删除',
            cancelButtonText: '取消',
            distinguishCancelAndClose: true,
            closeOnClickModal: false,
          },
        )
      } catch (e) {
        // 用户取消 / 关闭
        if (e === 'cancel' || e === 'close' || e?.action === 'cancel' || e?.action === 'close') return
        return
      }
      try {
        await App.DeleteShellFile(props.machineName, row.path)
        ElMessage.success('已删除')
        await reload()
      } catch (e) {
        ElMessage.error('删除失败: ' + e)
      }
    }

    watch(() => props.machineName, async () => {
      cwd.value = ''
      pathDraft.value = ''
      entries.value = []
      closeMenu()
      if (expanded.value) await reload()
    })

    onUnmounted(() => {
      stopPwdTimer()
      closeMenu()
    })

    expose({
      applyCwdHint,
      focusSearch: async () => {
        await nextTick()
        const input = searchInputRef.value
        if (!input) return
        if (typeof input.focus === 'function') input.focus()
        // Element Plus el-input 内部原生 input
        input.$el?.querySelector?.('input')?.focus?.()
      },
    })

    return {
      expanded,
      showHidden,
      cwd,
      pathDraft,
      entries,
      loading,
      error,
      treeData,
      treeWidth,
      bodyHeight,
      treeRef,
      treeRenderKey,
      expandedKeys,
      canGoUp,
      ctx,
      localSearchQuery,
      searchInputRef,
      searchVisible: computed(() => props.searchVisible),
      matchSummary: computed(() => props.matchSummary),
      emit,
      formatSize,
      formatTime,
      formatOwnerGroup,
      toggle,
      reload,
      applyCwdHint,
      onTreeClick,
      onNodeExpand,
      onOpen,
      goParent,
      submitPathDraft,
      syncPathDraftFromCwd,
      startHeightResize,
      startResize,
      onContextMenu,
      closeMenu,
      copyPath,
      deleteEntry,
    }
  },
}
</script>

<style scoped>
.shell-file-panel {
  position: relative;
  flex-shrink: 0;
  border-top: 1px solid var(--app-border);
  background: var(--app-panel-bg);
  display: flex;
  flex-direction: column;
  max-height: none;
}

.height-handle {
  height: 5px;
  cursor: row-resize;
  flex-shrink: 0;
  background: transparent;
  border-bottom: 1px solid transparent;
}

.height-handle:hover {
  background: rgba(64, 158, 255, 0.25);
}

.file-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 10px;
  min-height: 36px;
  flex-shrink: 0;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.parent-btn {
  padding: 4px 6px;
  min-width: auto;
}

.search-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
}

.search-bar :deep(.el-input) {
  flex: 1;
  max-width: 420px;
}

.search-count {
  font-size: 12px;
  font-weight: 600;
  color: var(--app-accent-color, #409eff);
  white-space: nowrap;
  min-width: 2.5em;
}

.search-actions {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 0;
  border-radius: 0;
  background: transparent;
  flex-shrink: 0;
}

.search-sep {
  width: 1px;
  height: 12px;
  margin: 0 2px;
  background: color-mix(in srgb, var(--app-text-muted, #909399) 35%, transparent);
}

.search-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--app-text-secondary, #909399);
  cursor: pointer;
  transition: color 0.15s ease, background 0.15s ease;
}

.search-icon-btn:hover {
  color: var(--app-accent-color, #409eff);
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 14%, transparent);
}

.search-icon-btn:active {
  transform: translateY(0.5px);
}

.search-close:hover {
  color: #f56c6c;
  background: rgba(245, 108, 108, 0.12);
}

.cwd-input {
  flex: 1;
  min-width: 120px;
  max-width: 480px;
}

.cwd-input :deep(.el-input__wrapper) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
}

.file-body {
  display: flex;
  flex-shrink: 0;
  border-top: 1px solid var(--app-border);
  overflow: hidden;
}

.tree-pane {
  flex-shrink: 0;
  overflow: auto;
  padding: 6px;
  box-sizing: border-box;
}

.split-handle {
  flex-shrink: 0;
  width: 5px;
  cursor: col-resize;
  background: transparent;
  border-left: 1px solid var(--app-border);
  border-right: 1px solid transparent;
}

.split-handle:hover {
  background: rgba(64, 158, 255, 0.25);
  border-right-color: rgba(64, 158, 255, 0.35);
}

.list-pane {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  padding: 0;
}

.name-cell {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.icon {
  color: var(--app-text-secondary);
}

.empty, .error {
  padding: 16px;
  font-size: 12px;
  color: var(--app-text-muted);
}

.error {
  color: var(--terminal-error);
}

.ctx-menu {
  position: fixed;
  z-index: 4000;
  margin: 0;
  padding: 4px 0;
  list-style: none;
  min-width: 140px;
  background: var(--app-card-bg);
  border: 1px solid var(--app-border);
  border-radius: 6px;
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.18);
}

.ctx-menu li {
  padding: 8px 14px;
  font-size: 13px;
  color: var(--app-text);
  cursor: pointer;
}

.ctx-menu li:hover {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}

.ctx-menu li.danger {
  color: var(--terminal-error);
}

.ctx-menu li.danger:hover {
  background: rgba(245, 108, 108, 0.12);
  color: var(--terminal-error);
}
</style>
