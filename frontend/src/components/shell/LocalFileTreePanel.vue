<template>
  <div class="local-file-tree">
    <div class="panel-header">
      <h3>本机文件</h3>
    </div>
    <div class="tree-toolbar">
      <el-checkbox v-model="showHidden" size="small" @change="reloadTree">隐藏文件</el-checkbox>
      <el-button size="small" text :loading="loading" title="刷新" @click="reloadTree">
        <el-icon><RefreshRight /></el-icon>
      </el-button>
    </div>
    <div v-if="loading && !treeData.length" class="empty">加载中…</div>
    <div v-else class="tree-wrap">
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
  </div>
</template>

<script>
import { ref, computed, onMounted, nextTick } from 'vue'
import { RefreshRight } from '@element-plus/icons-vue'
import * as App from '../../../wailsjs/go/app/App'

export default {
  name: 'LocalFileTreePanel',
  components: { RefreshRight },
  emits: ['path-change'],
  setup(props, { emit }) {
    const loading = ref(false)
    const showHidden = ref(false)
    const cwd = ref('')
    const treeRoot = ref([])
    const treeRef = ref(null)
    const treeRenderKey = ref(0)
    const expandedKeys = ref([])

    const treeData = computed(() => treeRoot.value)

    const normalizePath = (p) => {
      let s = String(p || '').trim()
      if (!s) return ''
      if (s.length > 1) s = s.replace(/[\\/]+$/, '')
      return s
    }

    const basename = (p) => {
      const s = normalizePath(p)
      if (!s) return ''
      const i = Math.max(s.lastIndexOf('/'), s.lastIndexOf('\\'))
      return i >= 0 ? s.slice(i + 1) : s
    }

    const ancestorPaths = (abs) => {
      const s = normalizePath(abs)
      if (!s) return []

      if (/^[A-Za-z]:[\\/]/.test(s)) {
        const sep = s.includes('\\') ? '\\' : '/'
        const parts = s.split(/[\\/]/).filter(Boolean)
        const drive = parts[0].endsWith(':') ? parts[0] : `${parts[0]}:`
        const out = [drive + sep]
        let cur = drive + sep
        for (let i = 1; i < parts.length; i++) {
          cur = cur.endsWith(sep) ? cur + parts[i] : cur + sep + parts[i]
          out.push(cur)
        }
        return out
      }

      if (s.startsWith('/')) {
        const parts = s.split('/').filter(Boolean)
        const out = []
        let cur = ''
        for (const part of parts) {
          cur += `/${part}`
          out.push(cur)
        }
        return out
      }

      return [s]
    }

    const listDirNodes = async (dir) => {
      const list = (await App.ListLocalFiles(dir, showHidden.value)) || []
      return list
        .filter((e) => e.isDir)
        .map((e) => ({
          name: e.name,
          path: normalizePath(e.path),
          isDir: true,
          children: [],
          _loaded: false,
        }))
        .sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base', numeric: true }))
    }

    const syncTreeToPath = async (abs) => {
      const target = normalizePath(abs)
      if (!target) return
      const chain = ancestorPaths(target)
      const rootPath = chain[0] || target
      const root = {
        name: '主目录',
        path: rootPath,
        isDir: true,
        children: await listDirNodes(rootPath),
        _loaded: true,
      }

      let parent = root
      for (let i = 1; i < chain.length; i++) {
        const want = chain[i]
        let child = (parent.children || []).find((c) => normalizePath(c.path) === want)
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
      expandedKeys.value = chain.length ? chain : [rootPath]
      treeRenderKey.value += 1
      cwd.value = target
      await nextTick()
      treeRef.value?.setCurrentKey?.(target)
    }

    const reloadTree = async () => {
      loading.value = true
      try {
        await syncTreeToPath(cwd.value || (await App.GetLocalHomeDir()) || '')
      } finally {
        loading.value = false
      }
    }

    const onTreeClick = (node) => {
      if (!node?.isDir) return
      cwd.value = node.path
      emit('path-change', node.path)
      if (!node._loaded) {
        listDirNodes(node.path).then((children) => {
          node.children = children
          node._loaded = true
        })
      }
    }

    const onNodeExpand = async (node) => {
      if (!node?.isDir || node._loaded) return
      try {
        node.children = await listDirNodes(node.path)
        node._loaded = true
      } catch {
        node.children = []
        node._loaded = true
      }
    }

    onMounted(async () => {
      loading.value = true
      try {
        let home = ''
        try {
          home = (await App.GetLocalHomeDir()) || ''
        } catch {
          home = ''
        }
        await syncTreeToPath(home)
      } catch (e) {
        console.warn('加载本机文件树失败:', e)
      } finally {
        loading.value = false
      }
    })

    return {
      loading,
      showHidden,
      cwd,
      treeData,
      treeRef,
      treeRenderKey,
      expandedKeys,
      reloadTree,
      onTreeClick,
      onNodeExpand,
    }
  },
}
</script>

<style scoped>
.local-file-tree {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  padding: 10px 10px 12px;
  box-sizing: border-box;
  color: var(--app-text);
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.panel-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}

.tree-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}

.tree-wrap {
  flex: 1;
  min-height: 0;
  overflow: auto;
  border: 1px solid var(--app-border);
  border-radius: 6px;
  padding: 6px;
  background: var(--app-bg);
}

.empty {
  padding: 16px 8px;
  font-size: 12px;
  color: var(--app-text-muted);
}

.tree-wrap :deep(.el-tree) {
  background: transparent;
  color: var(--app-text);
  --el-tree-node-hover-bg-color: var(--app-accent-bg);
  --el-tree-text-color: var(--app-text);
}

.tree-wrap :deep(.el-tree-node__content) {
  border-radius: 4px;
  height: 28px;
}

.tree-wrap :deep(.el-tree-node__content:focus),
.tree-wrap :deep(.el-tree-node__content:focus-visible) {
  outline: none;
}

.tree-wrap :deep(.el-tree--highlight-current .el-tree-node.is-current > .el-tree-node__content) {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}
</style>
