<template>
  <div
    ref="panelRef"
    class="shell-file-panel"
    :class="{
      collapsed: !expanded,
      'has-search': searchVisible,
      'is-bare': !expanded && !searchVisible,
    }"
  >
    <div
      class="height-handle"
      :class="{ 'is-collapsed-edge': !expanded }"
      title="拖动调整高度"
      @mousedown="startHeightResize"
    />
    <div v-show="searchVisible" class="search-bar">
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
    <div v-if="expanded" class="file-toolbar">
      <div class="toolbar-left">
        <div class="cwd-wrap">
          <el-input
            v-model="pathDraft"
            class="cwd-input"
            size="small"
            :title="cwd"
            placeholder="/"
            @focus="pathSuggestOpen = true"
            @input="onPathDraftInput"
            @keydown.down.exact.prevent="movePathSuggest(1)"
            @keydown.up.exact.prevent="movePathSuggest(-1)"
            @keydown.enter.exact.prevent="onPathEnter"
            @keydown.esc.exact.prevent="pathSuggestOpen = false"
            @blur="onPathBlur"
          />
          <ul v-if="pathSuggestOpen && pathSuggestions.length" class="path-suggest">
            <li
              v-for="(s, i) in pathSuggestions"
              :key="s.type + s.path"
              :class="{ active: i === pathSuggestIndex }"
              @mousedown.prevent="applyPathSuggestion(s.path)"
            >
              <span class="path-suggest-type">{{ pathSuggestTypeLabel(s.type) }}</span>
              <span class="path-suggest-path">{{ s.path }}</span>
            </li>
          </ul>
        </div>
        <el-checkbox v-model="showHidden" size="small" class="hidden-check" @change="reload">
          显示隐藏文件
        </el-checkbox>
        <el-checkbox
          v-model="followCwd"
          size="small"
          class="follow-check"
          :class="{ 'is-paused': followPaused }"
          @change="onFollowCwdChange"
        >
          跟随终端目录
        </el-checkbox>
        <template v-if="followPaused">
          <el-button size="small" text class="follow-resume-btn" @click="resumeFollowToTerminal">
            回到终端目录
          </el-button>
        </template>
        <el-dropdown
          size="small"
          trigger="click"
          :show-timeout="80"
          :hide-timeout="120"
          @command="onBookmarkCommand"
        >
          <el-button
            size="small"
            text
            class="tool-icon-btn"
            :class="{ 'is-bookmarked': isCurrentBookmarked }"
            :title="bookmarkButtonTitle"
            @click="onBookmarkButtonClick"
          >
            <el-icon><StarFilled v-if="isCurrentBookmarked" /><Star v-else /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item
                v-if="bookmarks.length"
                disabled
                class="bm-head"
              >收藏路径</el-dropdown-item>
              <el-dropdown-item
                v-for="bm in bookmarks"
                :key="bm.id"
                :command="`goto:${bm.id}`"
              >
                <span class="bm-row">
                  <span class="bm-path" :title="bm.path">{{ bm.label || bm.path }}</span>
                  <span v-if="bm.global" class="bm-tag">全局</span>
                </span>
              </el-dropdown-item>
              <el-dropdown-item v-if="!bookmarks.length" disabled>暂无收藏路径</el-dropdown-item>
              <el-dropdown-item divided :command="isCurrentBookmarked ? 'remove-current' : 'add-host'">
                {{ isCurrentBookmarked ? '取消收藏' : '收藏此路径' }}
              </el-dropdown-item>
              <el-dropdown-item command="add-global">+全局</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-button
          size="small"
          text
          class="tool-icon-btn"
          title="返回上级目录"
          :disabled="!canGoUp"
          @click="goParent"
        >
          <el-icon><ArrowUp /></el-icon>
        </el-button>
        <el-tooltip content="刷新目录" placement="top">
          <el-button size="small" text class="tool-icon-btn" :loading="loading" @click="reload">
            <el-icon><RefreshRight /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip content="转到终端当前目录" placement="top">
          <el-button size="small" text class="tool-icon-btn" title="转到终端当前目录" @click="goToTerminalCwd">
            <el-icon><Monitor /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip content="新建文件夹" placement="top">
          <el-button size="small" text class="tool-icon-btn" @click="promptMkdir">
            <el-icon><Folder /></el-icon>
          </el-button>
        </el-tooltip>
        <el-dropdown
          size="small"
          trigger="hover"
          :show-timeout="120"
          :hide-timeout="160"
          @command="onUploadCommand"
        >
          <el-button size="small" text class="tool-icon-btn" title="上传">
            <el-icon><Upload /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="files">上传文件</el-dropdown-item>
              <el-dropdown-item command="folder">上传文件夹</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-tooltip content="文件夹同步" placement="top">
          <el-button size="small" text class="tool-icon-btn" title="文件夹同步" @click="openSyncDialog">
            <el-icon><Refresh /></el-icon>
          </el-button>
        </el-tooltip>
        <el-dropdown trigger="click">
          <el-button size="small" text class="tool-icon-btn" title="显示列">
            <el-icon><Setting /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item @click.stop>
                <el-checkbox v-model="colVisible.size">大小</el-checkbox>
              </el-dropdown-item>
              <el-dropdown-item @click.stop>
                <el-checkbox v-model="colVisible.type">类型</el-checkbox>
              </el-dropdown-item>
              <el-dropdown-item @click.stop>
                <el-checkbox v-model="colVisible.mtime">修改时间</el-checkbox>
              </el-dropdown-item>
              <el-dropdown-item @click.stop>
                <el-checkbox v-model="colVisible.mode">权限</el-checkbox>
              </el-dropdown-item>
              <el-dropdown-item @click.stop>
                <el-checkbox v-model="colVisible.owner">用户:组</el-checkbox>
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
      <div class="toolbar-right">
        <el-input
          v-model="nameFilter"
          size="small"
          clearable
          placeholder="筛选文件名"
          class="name-filter"
        />
        <el-tooltip content="收起 SFTP" placement="top">
          <el-button size="small" text class="tool-icon-btn" @click="toggle">
            <el-icon><ArrowDown /></el-icon>
          </el-button>
        </el-tooltip>
      </div>
    </div>

    <div
      v-if="expanded"
      class="file-body shell-drop-zone"
      :style="{ height: bodyHeight + 'px' }"
      @dragenter.prevent="onDragOver"
      @dragover.prevent="onDragOver"
      @drop.prevent="onHtmlDrop"
    >
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
      <div
        class="list-pane"
        :class="{ 'drag-over': dragOver }"
        @click="closeMenu"
        @contextmenu.prevent="onBlankContextMenu"
        @dragenter.prevent="onDragOver"
        @dragover.prevent="onDragOver"
        @dragleave="onDragLeave"
        @drop.prevent="onHtmlDrop"
      >
        <div v-if="dragOver" class="drop-overlay">释放以上传文件/文件夹</div>
        <div v-if="error" class="error">{{ error }}</div>
        <el-table
          v-else
          :data="displayEntries"
          size="small"
          height="100%"
          v-loading="loading"
          empty-text="空目录（可拖拽文件/文件夹到此处上传）"
          tabindex="0"
          highlight-current-row
          @row-click="onRowClick"
          @current-change="onCurrentChange"
          @row-dblclick="onOpen"
          @row-contextmenu="onContextMenu"
          @keydown="onTableKeydown"
        >
          <el-table-column prop="name" min-width="160" show-overflow-tooltip>
            <template #header>
              <span class="sortable-th" @click.stop="toggleSort('name')">
                文件名
                <span v-if="sortKey === 'name'" class="sort-ind">{{ sortAsc ? '↑' : '↓' }}</span>
              </span>
            </template>
            <template #default="{ row }">
              <span class="name-cell">
                <el-icon class="icon">
                  <Folder v-if="row.isDir" />
                  <Link v-else-if="row.type === '链接'" />
                  <Document v-else />
                </el-icon>
                <span class="name-text">{{ row.name }}</span>
                <span v-if="row.linkTarget" class="link-target">→ {{ row.linkTarget }}</span>
              </span>
            </template>
          </el-table-column>
          <el-table-column v-if="colVisible.size" width="90" align="right">
            <template #header>
              <span class="sortable-th" @click.stop="toggleSort('size')">
                大小
                <span v-if="sortKey === 'size'" class="sort-ind">{{ sortAsc ? '↑' : '↓' }}</span>
              </span>
            </template>
            <template #default="{ row }">
              {{ row.isDir ? '-' : formatSize(row.size) }}
            </template>
          </el-table-column>
          <el-table-column v-if="colVisible.type" prop="type" label="类型" width="70" />
          <el-table-column v-if="colVisible.mtime" width="150">
            <template #header>
              <span class="sortable-th" @click.stop="toggleSort('mtime')">
                修改时间
                <span v-if="sortKey === 'mtime'" class="sort-ind">{{ sortAsc ? '↑' : '↓' }}</span>
              </span>
            </template>
            <template #default="{ row }">
              {{ formatTime(row.modTime) }}
            </template>
          </el-table-column>
          <el-table-column v-if="colVisible.mode" prop="mode" label="权限" width="110" show-overflow-tooltip />
          <el-table-column v-if="colVisible.owner" label="用户:组" width="120" show-overflow-tooltip>
            <template #default="{ row }">
              {{ formatOwnerGroup(row) }}
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <Teleport to="body">
      <ul
        v-if="ctx.visible"
        ref="ctxMenuRef"
        class="ctx-menu"
        :style="{ left: ctx.x + 'px', top: ctx.y + 'px' }"
        @click.stop
        @mouseleave="closeMenu"
      >
        <template v-if="!ctx.row">
          <li @click="reloadFromMenu">刷新</li>
          <li @click="promptMkdir">新建文件夹</li>
          <li @click="promptNewFile">新建文件</li>
          <li @click="uploadFilesFromMenu">上传文件</li>
          <li @click="uploadFolderFromMenu">上传文件夹</li>
          <li v-if="canPaste" class="ctx-sep" aria-hidden="true"></li>
          <li v-if="canPaste" @click="pasteEntry">粘贴</li>
        </template>
        <template v-else>
          <li @click="promptMkdir">新建文件夹</li>
          <li @click="promptNewFile">新建文件</li>
          <li class="ctx-sep" aria-hidden="true"></li>
          <li @click="openRowFromMenu">打开</li>
          <template v-if="!ctx.row.isDir">
            <li @click="openWithSystemDefaultEntry">系统默认</li>
            <li @click="openWithEntry">打开方式</li>
            <li v-if="!isBinaryRow(ctx.row)" @click="editEntry">编辑</li>
          </template>
          <li class="ctx-sep" aria-hidden="true"></li>
          <li @click="copyEntry">复制</li>
          <li @click="cutEntry">剪切</li>
          <li v-if="canPaste" @click="pasteEntry">粘贴</li>
          <li @click="copyHere">复制到此处</li>
          <li
            v-if="copyToOtherTargets.length === 1"
            @click="copyToOtherSide(copyToOtherTargets[0])"
          >复制到另一侧</li>
          <li
            v-else-if="copyToOtherTargets.length > 1"
            class="ctx-has-sub"
            @mouseenter="copyOtherSubOpen = true"
            @mouseleave="copyOtherSubOpen = false"
          >
            复制到另一侧
            <ul v-show="copyOtherSubOpen" class="ctx-submenu">
              <li
                v-for="t in copyToOtherTargets"
                :key="t.sessionId"
                :title="t.label"
                @click="copyToOtherSide(t)"
              >{{ t.label }}</li>
            </ul>
          </li>
          <li @click="moveToParent">移动到上级目录</li>
          <li @click="moveToPrompt">移动到…</li>
          <li class="ctx-sep" aria-hidden="true"></li>
          <li @click="promptRename">重命名</li>
          <li @click="promptChmod">修改权限</li>
          <li class="ctx-sep" aria-hidden="true"></li>
          <li @click="downloadEntry">下载</li>
          <li @click="copyPath">复制路径</li>
          <li @click="openInTerminal">在终端打开目录</li>
          <li class="ctx-sep" aria-hidden="true"></li>
          <li class="danger" @click="deleteEntry">删除</li>
        </template>
      </ul>
    </Teleport>

    <el-dialog
      v-model="chmodVisible"
      width="400px"
      append-to-body
      class="sftp-perm-dialog"
      :show-close="true"
      destroy-on-close
    >
      <template #header>
        <div class="perm-header">
          <div class="perm-title">编辑权限</div>
          <div class="perm-filename" :title="chmodName">{{ chmodName }}</div>
        </div>
      </template>
      <div class="perm-body">
        <div
          v-for="role in permRoles"
          :key="role.key"
          class="perm-row"
        >
          <div class="perm-role">{{ role.label }}</div>
          <div class="perm-checks">
            <label
              v-for="bit in permBits"
              :key="bit.key"
              class="perm-check"
            >
              <el-checkbox v-model="chmodPerms[role.key][bit.key]" />
              <span>{{ bit.label }}</span>
            </label>
          </div>
        </div>
        <div class="perm-summary">
          <span>八进制: <code>{{ chmodOctal }}</code></span>
          <span>符号: <code>{{ chmodSymbolic }}</code></span>
        </div>
      </div>
      <template #footer>
        <el-button @click="chmodVisible = false">取消</el-button>
        <el-button type="primary" :loading="chmodSaving" @click="submitChmod">应用</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="editorVisible"
      :title="editorTitle"
      width="80vw"
      top="6vh"
      append-to-body
      destroy-on-close
      class="sftp-editor-dialog"
    >
      <SftpMonacoEditor v-model="editorContent" :path="editorPath" />
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button type="primary" :loading="editorSaving" @click="saveEditor">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="syncVisible" title="文件夹同步" width="480px" append-to-body @open="onSyncDialogOpen">
      <el-form label-width="88px">
        <el-form-item label="本地目录">
          <div class="sync-path-row">
            <el-input v-model="syncForm.localDir" placeholder="选择本地文件夹" clearable />
            <el-button @click="pickSyncLocalDir">浏览</el-button>
          </div>
        </el-form-item>
        <el-form-item label="远端目录">
          <el-input v-model="syncForm.remoteDir" placeholder="远端路径" clearable />
        </el-form-item>
        <el-form-item label="方向">
          <el-radio-group v-model="syncForm.direction">
            <el-radio-button label="upload">本地 → 远端</el-radio-button>
            <el-radio-button label="download">远端 → 本地</el-radio-button>
            <el-radio-button label="both">双向</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <p class="sync-hint">递归同步子目录；按文件名与大小比较差异后传输；进度在传输队列中查看。</p>
      </el-form>
      <template #footer>
        <el-button @click="syncVisible = false">取消</el-button>
        <el-button type="primary" :loading="syncStarting" @click="startFolderSync">开始同步</el-button>
      </template>
    </el-dialog>

    <SftpConflictDialog
      :visible="conflictVisible"
      :item="conflictItem"
      :apply-to-all-count="conflictApplyCount"
      :format-size="formatSize"
      @resolve="onConflictResolve"
    />

    <SftpFileOpenerDialog
      v-model:visible="openerVisible"
      :file-name="openerFileName"
      @select="onOpenerSelect"
    />
  </div>
</template>

<script>
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as App from '../../../wailsjs/go/app/App'
import { EventsOn } from '../../../wailsjs/runtime/runtime'
import { OnFileDrop, OnFileDropOff } from '../../../wailsjs/runtime/runtime'
import SftpConflictDialog from './SftpConflictDialog.vue'
import SftpFileOpenerDialog from './SftpFileOpenerDialog.vue'
import SftpMonacoEditor from './SftpMonacoEditor.vue'
import {
  mergedBookmarks,
  isPathBookmarked,
  toggleHostBookmark,
  addGlobalBookmark,
  removeBookmark,
  pushPathHistory,
  loadFollowCwd,
  saveFollowCwd,
  suggestPaths,
} from '../../utils/sftpBookmarks'
import {
  getFileExtension,
  getOpenerForFile,
  isKnownBinaryFile,
} from '../../utils/sftpFileOpen'
import {
  setSftpClipboard,
  getSftpClipboard,
  hasSftpClipboardFor,
  clearSftpClipboard,
  extractClipboardLocalPaths,
} from '../../utils/sftpClipboard'
import {
  canRestoreSftpBrowseView,
  shouldSilentSftpRefresh,
} from '../../utils/shellTabViewCache'

export default {
  name: 'ShellFilePanel',
  components: { SftpConflictDialog, SftpFileOpenerDialog, SftpMonacoEditor },
  props: {
    machineName: { type: String, default: '' },
    cwdHint: { type: String, default: '' },
    searchVisible: { type: Boolean, default: false },
    searchQuery: { type: String, default: '' },
    matchSummary: { type: String, default: '' },
    /** @type {{ sessionId: string, destDir: string, label: string }[]} */
    copyToOtherTargets: { type: Array, default: () => [] },
  },
  emits: [
    'layout-change',
    'layout-resize-start',
    'layout-resize-end',
    'cwd-change',
    'search-next',
    'search-prev',
    'close-search',
    'update:searchQuery',
    'update:expanded',
    'transfer-started',
  ],
  setup(props, { emit, expose }) {
    const HEIGHT_KEY = 'shell.sftpBodyHeight'
    const isAuxMissingError = (msg) => /辅助连接(未建立|不存在)/.test(String(msg || ''))
    const setPanelError = (e) => {
      error.value = isAuxMissingError(e) ? '' : String(e || '')
    }
    const DEFAULT_BODY_HEIGHT = 220
    const MIN_BODY_HEIGHT = 120
    const MAX_BODY_RATIO = 0.65

    const expanded = ref(false)
    /** 按机器记住 SFTP 展开状态，切换会话互不影响 */
    const expandedByMachine = reactive({})
    const lastFollowedByMachine = Object.create(null)
    const browseCwdByMachine = Object.create(null)
    const terminalCwdByMachine = Object.create(null)
    const panelRef = ref(null)
    const showHidden = ref(false)
    const followCwd = ref(true)
    const lastFollowedCwd = ref('')
    const terminalCwd = ref('')
    const bookmarksVersion = ref(0)
    const pathSuggestOpen = ref(false)
    const pathSuggestIndex = ref(-1)
    const cwd = ref('')
    const pathDraft = ref('')
    const entries = ref([])
    const nameFilter = ref('')
    const sortKey = ref('name')
    const sortAsc = ref(true)
    const selectedRow = ref(null)
    const COL_KEY = 'shell.sftpColumns'
    const defaultCols = () => ({ size: true, type: true, mtime: true, mode: true, owner: true })
    const loadCols = () => {
      try {
        const raw = localStorage.getItem(COL_KEY)
        if (!raw) return defaultCols()
        return { ...defaultCols(), ...JSON.parse(raw) }
      } catch {
        return defaultCols()
      }
    }
    const colVisible = reactive(loadCols())
    watch(colVisible, (v) => {
      try { localStorage.setItem(COL_KEY, JSON.stringify({ ...v })) } catch { /* ignore */ }
    }, { deep: true })

    const loading = ref(false)
    const error = ref('')
    const treeRoot = ref([])
    const treeWidth = ref(220)
    const bodyHeight = ref(readBodyHeight())
    const treeRef = ref(null)
    const treeRenderKey = ref(0)
    const expandedKeys = ref(['/'])
    const ctx = reactive({ visible: false, x: 0, y: 0, row: null })
    const copyOtherSubOpen = ref(false)
    const ctxMenuRef = ref(null)
    const editorVisible = ref(false)
    const syncVisible = ref(false)
    const syncStarting = ref(false)
    const syncForm = reactive({
      localDir: '',
      remoteDir: '',
      direction: 'upload',
    })
    const editorContent = ref('')
    const editorPath = ref('')
    const editorTitle = ref('编辑文件')
    const editorSaving = ref(false)
    const openerVisible = ref(false)
    const openerFileName = ref('')
    const openerTarget = ref(null)
    const sftpOpenConfig = reactive({
      sftpDefaultOpener: 'ask',
      sftpDefaultSystemApp: null,
      sftpAutoSync: true,
      sftpFileAssociations: {},
    })
    const chmodVisible = ref(false)
    const chmodSaving = ref(false)
    const chmodTarget = ref('')
    const chmodName = ref('')
    const emptyPermBits = () => ({ read: false, write: false, execute: false })
    const chmodPerms = reactive({
      owner: emptyPermBits(),
      group: emptyPermBits(),
      others: emptyPermBits(),
    })
    const permRoles = [
      { key: 'owner', label: '所有者' },
      { key: 'group', label: '群组' },
      { key: 'others', label: '其他' },
    ]
    const permBits = [
      { key: 'read', label: 'R' },
      { key: 'write', label: 'W' },
      { key: 'execute', label: 'X' },
    ]

    const roleToNum = (p) => (p.read ? 4 : 0) + (p.write ? 2 : 0) + (p.execute ? 1 : 0)
    const roleToSym = (p) => `${p.read ? 'r' : '-'}${p.write ? 'w' : '-'}${p.execute ? 'x' : '-'}`
    const chmodOctal = computed(() =>
      `${roleToNum(chmodPerms.owner)}${roleToNum(chmodPerms.group)}${roleToNum(chmodPerms.others)}`,
    )
    const chmodSymbolic = computed(() =>
      roleToSym(chmodPerms.owner) + roleToSym(chmodPerms.group) + roleToSym(chmodPerms.others),
    )

    const applyModeToPerms = (modeStr) => {
      const raw = String(modeStr || '').trim()
      let owner = emptyPermBits()
      let group = emptyPermBits()
      let others = emptyPermBits()
      if (/^[0-7]{3,4}$/.test(raw)) {
        const octal = raw.length === 4 ? raw.slice(1) : raw
        const parse = (n) => ({
          read: (n & 4) !== 0,
          write: (n & 2) !== 0,
          execute: (n & 1) !== 0,
        })
        owner = parse(parseInt(octal[0], 10))
        group = parse(parseInt(octal[1], 10))
        others = parse(parseInt(octal[2], 10))
      } else {
        const pStr = raw.length >= 10 ? raw.slice(1) : raw
        if (pStr.length >= 9) {
          owner = {
            read: pStr[0] === 'r',
            write: pStr[1] === 'w',
            execute: pStr[2] === 'x' || pStr[2] === 's' || pStr[2] === 'S',
          }
          group = {
            read: pStr[3] === 'r',
            write: pStr[4] === 'w',
            execute: pStr[5] === 'x' || pStr[5] === 's' || pStr[5] === 'S',
          }
          others = {
            read: pStr[6] === 'r',
            write: pStr[7] === 'w',
            execute: pStr[8] === 'x' || pStr[8] === 't' || pStr[8] === 'T',
          }
        } else {
          owner = { read: true, write: true, execute: false }
          group = { read: true, write: false, execute: false }
          others = { read: true, write: false, execute: false }
        }
      }
      Object.assign(chmodPerms.owner, owner)
      Object.assign(chmodPerms.group, group)
      Object.assign(chmodPerms.others, others)
    }
    const localSearchQuery = ref(props.searchQuery)
    const searchInputRef = ref(null)
    const dragOver = ref(false)
    let pwdTimer = null
    let resizing = false
    let navSeq = 0
    let dropBound = false

    watch(() => props.searchQuery, (v) => { localSearchQuery.value = v })
    watch(localSearchQuery, (v) => emit('update:searchQuery', v))
    watch(() => props.searchVisible, async (visible) => {
      // 与展开 SFTP 相同：占文档流、先抑制 RO fit，布局稳定后再统一 fit
      emit('layout-resize-start')
      await nextTick()
      if (visible) {
        searchInputRef.value?.focus?.()
      }
      notifyLayout()
    })

    watch(expanded, (v) => {
      emit('update:expanded', v)
      const name = props.machineName
      if (name) expandedByMachine[name] = !!v
    })

    const treeData = computed(() => treeRoot.value)
    const canGoUp = computed(() => {
      const p = cwd.value
      return !!p && p !== '/' && p !== '.'
    })

    const followPaused = computed(() => {
      if (!followCwd.value) return false
      const term = normalizeAbs(terminalCwd.value || props.cwdHint)
      const here = normalizeAbs(cwd.value)
      return !!(term && here && term !== here)
    })

    const markFollowedCwd = (abs) => {
      const p = normalizeAbs(abs)
      if (!p) return
      terminalCwd.value = p
      lastFollowedCwd.value = p
    }

    const snapshotFollowState = (name) => {
      const key = String(name || '').trim()
      if (!key) return
      lastFollowedByMachine[key] = lastFollowedCwd.value
      browseCwdByMachine[key] = cwd.value
      terminalCwdByMachine[key] = terminalCwd.value
    }

    /** 按会话记住目录树/文件列表，切 tab 先还原画面再静默刷新 */
    const viewByMachine = Object.create(null)

    const liveExpandedKeys = () => {
      const keys = treeRef.value?.getExpandedKeys?.()
      if (Array.isArray(keys) && keys.length) return keys.slice()
      return Array.isArray(expandedKeys.value) ? expandedKeys.value.slice() : ['/']
    }

    const snapshotBrowseView = (name) => {
      const key = String(name || '').trim()
      if (!key) return
      viewByMachine[key] = {
        cwd: cwd.value,
        pathDraft: pathDraft.value,
        entries: entries.value,
        selectedPath: selectedRow.value?.path || '',
        nameFilter: nameFilter.value,
        treeRoot: treeRoot.value,
        expandedKeys: liveExpandedKeys(),
      }
    }

    const restoreBrowseView = (name) => {
      const key = String(name || '').trim()
      const cached = key ? viewByMachine[key] : null
      if (!canRestoreSftpBrowseView(cached)) return false
      cwd.value = cached.cwd
      pathDraft.value = cached.pathDraft || cached.cwd
      entries.value = cached.entries || []
      nameFilter.value = cached.nameFilter || ''
      treeRoot.value = cached.treeRoot || []
      expandedKeys.value = cached.expandedKeys?.length ? cached.expandedKeys : ['/']
      selectedRow.value = cached.selectedPath
        ? (entries.value || []).find((e) => e.path === cached.selectedPath) || null
        : null
      treeRenderKey.value += 1
      return true
    }

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
      const marker = '777;cwd;'
      const idx = s.indexOf(marker)
      if (idx >= 0) s = s.slice(idx + marker.length)
      s = s.replace(/\x07/g, '')
      const esc = s.indexOf('\x1b')
      if (esc >= 0) s = s.slice(0, esc)
      s = s.trim()
      if (!s || !s.startsWith('/') || s.includes('777;cwd') || s.includes(']')) return ''
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

    const compareEntries = (a, b, key, asc) => {
      if (!!a.isDir !== !!b.isDir) return a.isDir ? -1 : 1
      let cmp = 0
      if (key === 'size') {
        cmp = (Number(a.size) || 0) - (Number(b.size) || 0)
      } else if (key === 'mtime') {
        cmp = (Number(a.modTime) || 0) - (Number(b.modTime) || 0)
      } else {
        cmp = String(a.name || '').localeCompare(String(b.name || ''), undefined, {
          sensitivity: 'base',
          numeric: true,
        })
      }
      return asc ? cmp : -cmp
    }

    const displayEntries = computed(() => {
      let list = entries.value || []
      const q = String(nameFilter.value || '').trim().toLowerCase()
      if (q) {
        list = list.filter((e) => String(e.name || '').toLowerCase().includes(q))
      }
      const key = sortKey.value || 'name'
      const asc = !!sortAsc.value
      return list.slice().sort((a, b) => compareEntries(a, b, key, asc))
    })

    const clipboardTick = ref(0)
    const canPaste = computed(() => {
      clipboardTick.value
      return hasSftpClipboardFor(props.machineName)
    })
    const bumpClipboard = () => { clipboardTick.value += 1 }

    const toggleSort = (key) => {
      if (sortKey.value === key) {
        sortAsc.value = !sortAsc.value
      } else {
        sortKey.value = key
        sortAsc.value = true
      }
    }

    const onRowClick = (row) => {
      selectedRow.value = row || null
    }

    const onCurrentChange = (row) => {
      selectedRow.value = row || null
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
        if (selectedRow.value?.path) {
          const keep = entries.value.find((e) => e.path === selectedRow.value.path)
          selectedRow.value = keep || null
        }
      } catch (e) {
        setPanelError(e)
        entries.value = []
        selectedRow.value = null
      } finally {
        loading.value = false
      }
    }

    const setCwd = async (next, { rebuildTree = true } = {}) => {
      const abs = normalizeAbs(next)
      cwd.value = abs
      pathDraft.value = abs
      selectedRow.value = null
      emit('cwd-change', abs)
      if (props.machineName) pushPathHistory(props.machineName, abs)
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
        if (seq === navSeq) setPanelError(e)
      }
    }

    /** 供父组件在终端 cwd 变化后直接调用 */
    const applyCwdHint = async (hint) => {
      const abs = normalizeAbs(hint)
      if (!abs) return
      terminalCwd.value = abs
      if (!followCwd.value) return
      // 终端上报的 cwd 没变（含 prompt / OSC / pwd）：尊重手动逛树，不要把 SFTP 拽回去
      // 目录真的变了（不限于 cd）则继续跟随
      if (abs === lastFollowedCwd.value) return
      lastFollowedCwd.value = abs
      const seq = ++navSeq
      // 未展开时也记下路径，展开后使用
      if (!expanded.value) {
        if (abs !== normalizeAbs(cwd.value)) {
          cwd.value = abs
          pathDraft.value = abs
          emit('cwd-change', abs)
        }
        return
      }
      if (abs === normalizeAbs(cwd.value)) {
        return
      }
      cwd.value = abs
      pathDraft.value = abs
      emit('cwd-change', abs)
      if (seq !== navSeq) return
      await reloadList()
      if (seq !== navSeq) return
      try {
        await syncTreeToPath(abs)
      } catch (e) {
        console.warn('同步目录树失败:', e)
      }
    }

    const hostKey = () => props.machineName || ''
    const bookmarks = computed(() => {
      bookmarksVersion.value
      return mergedBookmarks(hostKey())
    })
    const isCurrentBookmarked = computed(() => {
      bookmarksVersion.value
      return isPathBookmarked(hostKey(), cwd.value)
    })
    const bookmarkButtonTitle = computed(() => {
      if (!bookmarks.value.length && !isCurrentBookmarked.value) return '收藏此路径'
      return '收藏路径'
    })
    const pathSuggestions = computed(() => {
      if (!pathSuggestOpen.value) return []
      return suggestPaths({
        hostKey: hostKey(),
        draft: pathDraft.value,
        folderEntries: entries.value,
      })
    })

    const refreshBookmarks = () => { bookmarksVersion.value += 1 }

    const onFollowCwdChange = () => {
      saveFollowCwd(hostKey(), followCwd.value)
      if (followCwd.value) void resumeFollowToTerminal()
    }

    const onBookmarkButtonClick = (e) => {
      if (!bookmarks.value.length && !isCurrentBookmarked.value && cwd.value) {
        e?.stopPropagation?.()
        e?.preventDefault?.()
        toggleHostBookmark(hostKey(), cwd.value)
        refreshBookmarks()
        ElMessage.success('已收藏路径')
      }
    }

    const onBookmarkCommand = async (cmd) => {
      const c = String(cmd || '')
      if (c === 'add-host') {
        if (!cwd.value) return
        toggleHostBookmark(hostKey(), cwd.value)
        refreshBookmarks()
        return
      }
      if (c === 'remove-current') {
        if (!cwd.value) return
        toggleHostBookmark(hostKey(), cwd.value)
        refreshBookmarks()
        return
      }
      if (c === 'add-global') {
        if (!cwd.value) return
        addGlobalBookmark(cwd.value)
        refreshBookmarks()
        ElMessage.success('已添加全局收藏')
        return
      }
      if (c.startsWith('goto:')) {
        const id = c.slice(5)
        const bm = bookmarks.value.find((b) => b.id === id)
        if (bm?.path) await navigateTo(bm.path, { alreadyAbsolute: true })
        return
      }
      if (c.startsWith('remove:')) {
        removeBookmark(hostKey(), c.slice(7))
        refreshBookmarks()
      }
    }

    const pathSuggestTypeLabel = (type) => {
      if (type === 'bookmark') return '收藏'
      if (type === 'history') return '历史'
      return '目录'
    }

    const onPathDraftInput = () => {
      pathSuggestOpen.value = true
      pathSuggestIndex.value = -1
    }

    const movePathSuggest = (delta) => {
      const list = pathSuggestions.value
      if (!list.length) return
      pathSuggestOpen.value = true
      const n = list.length
      pathSuggestIndex.value = (pathSuggestIndex.value + delta + n) % n
    }

    const applyPathSuggestion = async (path) => {
      pathDraft.value = path
      pathSuggestOpen.value = false
      pathSuggestIndex.value = -1
      await submitPathDraft()
    }

    const onPathEnter = async () => {
      if (pathSuggestOpen.value && pathSuggestIndex.value >= 0) {
        const s = pathSuggestions.value[pathSuggestIndex.value]
        if (s) {
          await applyPathSuggestion(s.path)
          return
        }
      }
      pathSuggestOpen.value = false
      await submitPathDraft()
    }

    const onPathBlur = () => {
      setTimeout(() => {
        pathSuggestOpen.value = false
        syncPathDraftFromCwd()
      }, 120)
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
      if (!row) return
      if (row.isDir) {
        await navigateTo(row.path, { alreadyAbsolute: true })
        return
      }
      await openFileByPolicy(row)
    }

    const isBinaryRow = (row) => isKnownBinaryFile(row?.name)

    const applySftpOpenConfig = (cfg) => {
      if (!cfg || typeof cfg !== 'object') return
      if (cfg.sftpDefaultOpener !== undefined) {
        sftpOpenConfig.sftpDefaultOpener = cfg.sftpDefaultOpener || 'ask'
      }
      if (cfg.sftpDefaultSystemApp !== undefined) {
        sftpOpenConfig.sftpDefaultSystemApp = cfg.sftpDefaultSystemApp || null
      }
      if (cfg.sftpAutoSync !== undefined) {
        sftpOpenConfig.sftpAutoSync = cfg.sftpAutoSync !== false
      }
      if (cfg.sftpFileAssociations !== undefined) {
        sftpOpenConfig.sftpFileAssociations = cfg.sftpFileAssociations || {}
      }
    }

    const loadSftpOpenConfig = async () => {
      try {
        const cfg = await App.GetGlobalConfig()
        applySftpOpenConfig(cfg)
      } catch {
        // 使用默认打开策略
      }
    }

    const openBuiltinEditor = async (row) => {
      if (!row?.path || row.isDir || !props.machineName) return
      try {
        editorContent.value = await App.ReadShellRemoteFile(props.machineName, row.path)
        editorPath.value = row.path
        editorTitle.value = `编辑 — ${row.name}`
        editorVisible.value = true
      } catch (e) {
        ElMessage.error(String(e))
      }
    }

    const openWithSystemApp = async (row, appPath) => {
      if (!row?.path || !props.machineName) return
      try {
        await App.OpenShellRemoteFileWithApp(
          props.machineName,
          row.path,
          appPath,
          !!sftpOpenConfig.sftpAutoSync,
        )
      } catch (e) {
        ElMessage.error(String(e))
      }
    }

    const openWithSystemDefault = async (row) => {
      if (!row?.path || !props.machineName) return
      try {
        await App.OpenShellRemoteFileSystemDefault(
          props.machineName,
          row.path,
          !!sftpOpenConfig.sftpAutoSync,
        )
      } catch (e) {
        ElMessage.error(String(e))
      }
    }

    const showOpenerDialog = (row) => {
      openerTarget.value = row
      openerFileName.value = row?.name || ''
      openerVisible.value = true
    }

    const openFileByPolicy = async (row) => {
      if (!row?.path || row.isDir || !props.machineName) return
      const opener = getOpenerForFile(sftpOpenConfig, row.name)
      if (!opener) {
        showOpenerDialog(row)
        return
      }
      if (opener.openerType === 'builtin-editor') {
        await openBuiltinEditor(row)
        return
      }
      if (opener.openerType === 'system-app' && opener.systemApp?.path) {
        await openWithSystemApp(row, opener.systemApp.path)
        return
      }
      showOpenerDialog(row)
    }

    const rememberAssociation = async (fileName, openerType, systemApp) => {
      const ext = getFileExtension(fileName)
      try {
        await App.UpsertSftpFileAssociation(ext, {
          openerType,
          systemApp: systemApp || undefined,
        })
        if (!sftpOpenConfig.sftpFileAssociations) {
          sftpOpenConfig.sftpFileAssociations = {}
        }
        sftpOpenConfig.sftpFileAssociations[ext] = {
          openerType,
          systemApp: systemApp || undefined,
        }
      } catch (e) {
        ElMessage.error('保存文件关联失败: ' + e)
      }
    }

    const onOpenerSelect = async (payload) => {
      const row = openerTarget.value
      openerTarget.value = null
      if (!row || !payload?.openerType) return
      if (payload.remember) {
        await rememberAssociation(row.name, payload.openerType, payload.systemApp)
      }
      if (payload.openerType === 'builtin-editor') {
        await openBuiltinEditor(row)
        return
      }
      if (payload.openerType === 'system-app' && payload.systemApp?.path) {
        await openWithSystemApp(row, payload.systemApp.path)
      }
    }

    const openEntry = async () => {
      const row = ctx.row
      closeMenu()
      await openFileByPolicy(row)
    }

    const openRowFromMenu = async () => {
      const row = ctx.row
      closeMenu()
      await onOpen(row)
    }

    const openWithEntry = () => {
      const row = ctx.row
      closeMenu()
      if (!row?.path || row.isDir) return
      showOpenerDialog(row)
    }

    const openWithSystemDefaultEntry = async () => {
      const row = ctx.row
      closeMenu()
      await openWithSystemDefault(row)
    }

    const goParent = async () => {
      if (!canGoUp.value) return
      await navigateTo('..')
    }

    const goToTerminalCwd = async () => {
      if (!props.machineName || !expanded.value) return
      try {
        const remote = await App.GetShellPtyCwd(props.machineName)
        const abs = normalizeAbs(remote)
        if (!abs) {
          ElMessage.warning('无法获取终端当前目录')
          return
        }
        markFollowedCwd(abs)
        await navigateTo(abs, { alreadyAbsolute: true, trustAbsolute: true })
      } catch (e) {
        ElMessage.error('获取终端目录失败: ' + e)
      }
    }

    const resumeFollowToTerminal = async () => {
      const known = normalizeAbs(terminalCwd.value || props.cwdHint)
      if (!expanded.value) {
        if (known) {
          markFollowedCwd(known)
          cwd.value = known
          pathDraft.value = known
          emit('cwd-change', known)
        }
        return
      }
      if (props.machineName) {
        await goToTerminalCwd()
        return
      }
      if (!known) {
        ElMessage.warning('无法获取终端当前目录')
        return
      }
      markFollowedCwd(known)
      await navigateTo(known, { alreadyAbsolute: true, trustAbsolute: true })
    }

    const reload = async () => {
      if (!props.machineName || !expanded.value) return
      if (!cwd.value) {
        await setCwd(await ensureHome())
        return
      }
      await setCwd(cwd.value)
    }

    const calibrateList = async () => {
      if (!props.machineName || !expanded.value || !cwd.value) return
      try {
        entries.value = await listDir(cwd.value)
        error.value = ''
      } catch (e) {
        // 当前目录突然不存在时保留展示；辅助连接断开则不提示
        setPanelError(e)
      }
    }

    const startPwdTimer = () => {
      stopPwdTimer()
      pwdTimer = setInterval(async () => {
        await calibrateList()
      }, 4000)
    }
    const stopPwdTimer = () => {
      if (pwdTimer) {
        clearInterval(pwdTimer)
        pwdTimer = null
      }
    }

    const bootstrapExpand = async () => {
      let start = cwd.value
      try {
        const remote = await App.GetShellPtyCwd(props.machineName)
        if (remote && String(remote).startsWith('/')) start = remote
      } catch {
        // 使用 cwdHint / home
      }
      await setCwd(start || await ensureHome())
      markFollowedCwd(cwd.value)
      startPwdTimer()
    }

    const toggle = async () => {
      expanded.value = !expanded.value
      if (expanded.value) {
        await bootstrapExpand()
      } else {
        stopPwdTimer()
        closeMenu()
      }
      await nextTick()
      notifyLayout()
    }

    const startHeightResize = (e) => {
      e.preventDefault()
      emit('layout-resize-start')
      const startY = e.clientY
      const opening = !expanded.value
      const startH = bodyHeight.value
      const OPEN_THRESHOLD = 6
      const COLLAPSE_THRESHOLD = 48
      let didOpen = false

      const onMove = (ev) => {
        const delta = startY - ev.clientY
        const maxH = Math.floor(window.innerHeight * MAX_BODY_RATIO)
        if (opening) {
          if (!didOpen) {
            if (delta < OPEN_THRESHOLD) return
            didOpen = true
            expanded.value = true
            bodyHeight.value = delta
            void bootstrapExpand()
          } else {
            bodyHeight.value = Math.min(maxH, Math.max(0, delta))
          }
          // 拖拽中不 fit，松手后再通知，避免连续抖动
          return
        }
        bodyHeight.value = Math.min(maxH, Math.max(MIN_BODY_HEIGHT, startH + delta))
      }

      const finishHeightResize = (shouldNotifyLayout) => {
        window.removeEventListener('mousemove', onMove)
        window.removeEventListener('mouseup', onUp)
        emit('layout-resize-end')
        if (shouldNotifyLayout) notifyLayout()
      }

      const onUp = () => {
        if (opening) {
          if (!didOpen) {
            finishHeightResize(false)
            return
          }
          if (bodyHeight.value < COLLAPSE_THRESHOLD) {
            expanded.value = false
            bodyHeight.value = startH >= MIN_BODY_HEIGHT ? startH : MIN_BODY_HEIGHT
            stopPwdTimer()
            closeMenu()
          } else {
            bodyHeight.value = Math.max(MIN_BODY_HEIGHT, bodyHeight.value)
            localStorage.setItem(HEIGHT_KEY, String(bodyHeight.value))
          }
          finishHeightResize(true)
          return
        }
        localStorage.setItem(HEIGHT_KEY, String(bodyHeight.value))
        finishHeightResize(true)
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

    const adjustContextMenuPosition = async () => {
      await nextTick()
      const el = ctxMenuRef.value
      if (!el) return
      const rect = el.getBoundingClientRect()
      const pad = 8
      const vw = window.innerWidth
      const vh = window.innerHeight
      let { x, y } = ctx
      if (x + rect.width > vw - pad) {
        x = Math.max(pad, vw - rect.width - pad)
      }
      if (y + rect.height > vh - pad) {
        y = Math.max(pad, y - rect.height)
      }
      ctx.x = x
      ctx.y = y
    }

    const onContextMenu = (row, _col, event) => {
      event.preventDefault()
      event.stopPropagation()
      ctx.row = row
      selectedRow.value = row || null
      ctx.x = event.clientX
      ctx.y = event.clientY
      ctx.visible = true
      adjustContextMenuPosition()
    }

    const onBlankContextMenu = (event) => {
      // 点在表格行上时由 row-contextmenu 处理并 stopPropagation
      ctx.row = null
      ctx.x = event.clientX
      ctx.y = event.clientY
      ctx.visible = true
      adjustContextMenuPosition()
    }

    const closeMenu = () => {
      ctx.visible = false
      ctx.row = null
      copyOtherSubOpen.value = false
    }

    const reloadFromMenu = async () => {
      closeMenu()
      await reload()
    }

    const uploadFilesFromMenu = async () => {
      closeMenu()
      await onUploadCommand('files')
    }

    const uploadFolderFromMenu = async () => {
      closeMenu()
      await onUploadCommand('folder')
    }

    const copyPath = async () => {
      const path = ctx.row?.path
      closeMenu()
      if (!path) return
      try {
        await navigator.clipboard.writeText(path)
      } catch {
        // 静默失败：复制路径不弹轻提示
      }
    }

    const downloadEntry = async () => {
      const row = ctx.row
      closeMenu()
      if (!row?.path || !props.machineName) return
      try {
        await App.StartShellDownload(props.machineName, row.path)
        ElMessage.success(`已开始下载：${row.name}`)
        emit('transfer-started', { direction: 'download', name: row.name })
      } catch (e) {
        ElMessage.error('下载失败: ' + e)
      }
    }

    const editEntry = async () => {
      const row = ctx.row
      closeMenu()
      await openBuiltinEditor(row)
    }

    const saveEditor = async () => {
      if (!editorPath.value || !props.machineName) return
      editorSaving.value = true
      try {
        await App.SaveShellRemoteFile(props.machineName, editorPath.value, editorContent.value)
        ElMessage.success('已保存')
        editorVisible.value = false
        reload()
      } catch (e) {
        ElMessage.error('保存失败: ' + e)
      } finally {
        editorSaving.value = false
      }
    }

    const joinRemote = (base, name) => {
      const b = String(base || '/').replace(/\/+$/, '') || ''
      const n = String(name || '').replace(/^\/+/, '')
      if (!b) return '/' + n
      return `${b}/${n}`
    }

    const promptMkdir = async () => {
      closeMenu()
      if (!props.machineName || !cwd.value) return
      try {
        const { value } = await ElMessageBox.prompt('输入新文件夹名称', '新建文件夹', {
          confirmButtonText: '创建',
          cancelButtonText: '取消',
          inputPattern: /.+/,
          inputErrorMessage: '名称不能为空',
        })
        const remotePath = joinRemote(cwd.value, value)
        await App.MkdirShellRemotePath(props.machineName, remotePath)
        ElMessage.success('文件夹已创建')
        reload()
      } catch (e) {
        if (e === 'cancel') return
        ElMessage.error(String(e))
      }
    }

    const promptNewFile = async () => {
      closeMenu()
      if (!props.machineName || !cwd.value) return
      try {
        const { value } = await ElMessageBox.prompt('输入新文件名称', '新建文件', {
          confirmButtonText: '创建',
          cancelButtonText: '取消',
          inputPattern: /.+/,
          inputErrorMessage: '名称不能为空',
        })
        const name = String(value || '').trim()
        if (!name || name.includes('/')) {
          ElMessage.warning('文件名无效')
          return
        }
        const remotePath = joinRemote(cwd.value, name)
        await App.SaveShellRemoteFile(props.machineName, remotePath, '')
        ElMessage.success('文件已创建')
        reload()
      } catch (e) {
        if (e === 'cancel') return
        ElMessage.error(String(e))
      }
    }

    const promptRename = async () => {
      const row = ctx.row
      closeMenu()
      if (!row?.path || !props.machineName) return
      try {
        const { value } = await ElMessageBox.prompt('输入新名称', '重命名', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          inputValue: row.name,
          inputPattern: /.+/,
          inputErrorMessage: '名称不能为空',
        })
        const parent = row.path.replace(/\/[^/]+$/, '') || '/'
        const newPath = joinRemote(parent, value)
        await App.RenameShellRemotePath(props.machineName, row.path, newPath)
        ElMessage.success('已重命名')
        reload()
      } catch (e) {
        if (e === 'cancel') return
        ElMessage.error(String(e))
      }
    }

    const promptChmod = () => {
      const row = ctx.row
      closeMenu()
      if (!row?.path || !props.machineName) return
      chmodTarget.value = row.path
      chmodName.value = row.name || ''
      applyModeToPerms(row.mode)
      chmodVisible.value = true
    }

    const submitChmod = async () => {
      if (!chmodTarget.value || !props.machineName) return
      chmodSaving.value = true
      try {
        const mode = parseInt(chmodOctal.value, 8) & 0xfff
        await App.ChmodShellRemotePath(props.machineName, chmodTarget.value, mode)
        ElMessage.success('权限已更新')
        chmodVisible.value = false
        reload()
      } catch (e) {
        ElMessage.error(String(e))
      } finally {
        chmodSaving.value = false
      }
    }

    const uniqueCopyName = (baseName) => {
      const names = new Set((entries.value || []).map((e) => e.name))
      if (!names.has(baseName)) return baseName
      let candidate = `${baseName}_copy`
      let i = 2
      while (names.has(candidate)) {
        candidate = `${baseName}_copy${i}`
        i += 1
      }
      return candidate
    }

    const parentDirOf = (abs) => {
      const s = normalizeAbs(abs)
      if (!s || s === '/') return '/'
      const i = s.lastIndexOf('/')
      if (i <= 0) return '/'
      return s.slice(0, i) || '/'
    }

    const copyEntry = () => {
      const row = ctx.row || selectedRow.value
      closeMenu()
      if (!row?.path || !props.machineName) return
      setSftpClipboard('copy', props.machineName, row)
      bumpClipboard()
      ElMessage.success(`已复制：${row.name}`)
    }

    const cutEntry = () => {
      const row = ctx.row || selectedRow.value
      closeMenu()
      if (!row?.path || !props.machineName) return
      setSftpClipboard('cut', props.machineName, row)
      bumpClipboard()
      ElMessage.success(`已剪切：${row.name}`)
    }

    const pasteEntry = async () => {
      closeMenu()
      if (!props.machineName || !cwd.value) return
      const clip = getSftpClipboard()
      if (!clip || clip.machineName !== props.machineName || !clip.entry?.path) {
        ElMessage.warning('剪贴板为空')
        return
      }
      const src = clip.entry
      const srcPath = normalizeAbs(src.path)
      let destName = src.name
      const destExists = (entries.value || []).some((e) => e.name === destName)
      if (destExists) {
        if (clip.mode === 'cut' && joinRemote(cwd.value, destName) === srcPath) {
          clearSftpClipboard()
          bumpClipboard()
          return
        }
        destName = uniqueCopyName(src.name)
      }
      const dst = joinRemote(cwd.value, destName)
      try {
        if (clip.mode === 'cut') {
          await App.MoveShellRemotePath(props.machineName, srcPath, dst)
          clearSftpClipboard()
          bumpClipboard()
          ElMessage.success('已移动到当前目录')
        } else {
          await App.CopyShellRemotePath(props.machineName, srcPath, dst)
          ElMessage.success('已粘贴到当前目录')
        }
        await reload()
      } catch (e) {
        ElMessage.error(String(e))
      }
    }

    const copyHere = async () => {
      const row = ctx.row
      closeMenu()
      if (!row?.path || !props.machineName || !cwd.value) return
      const dst = joinRemote(cwd.value, `${row.name}_copy`)
      try {
        await App.CopyShellRemotePath(props.machineName, row.path, dst)
        ElMessage.success('已复制到此处')
        await reload()
      } catch (e) {
        ElMessage.error(String(e))
      }
    }

    const copyToOtherSide = async (target) => {
      const row = ctx.row || selectedRow.value
      closeMenu()
      copyOtherSubOpen.value = false
      if (!row?.path || !props.machineName) return
      if (!target?.sessionId || !target?.destDir) {
        ElMessage.warning('请打开另一侧终端并进入目标目录后再复制')
        return
      }
      try {
        const result = await App.StartShellCopyToOther(
          props.machineName,
          row.path,
          target.sessionId,
          target.destDir,
        )
        const dest = result?.destPath || target.destDir
        if (result?.mode === 'transfer') {
          emit('transfer-started', { direction: 'copy', name: row.name })
          ElMessage.success(`已开始复制到另一侧：${dest}`)
        } else {
          ElMessage.success(`已复制到另一侧：${dest}`)
        }
      } catch (e) {
        ElMessage.error(String(e))
      }
    }

    const moveToParent = async () => {
      const row = ctx.row
      closeMenu()
      if (!row?.path || !props.machineName || !cwd.value) return
      if (!canGoUp.value) {
        ElMessage.warning('已在根目录，无法移动到上级')
        return
      }
      const parent = parentDirOf(cwd.value)
      const dst = joinRemote(parent, row.name)
      if (normalizeAbs(dst) === normalizeAbs(row.path)) {
        ElMessage.warning('目标与当前位置相同')
        return
      }
      try {
        await App.MoveShellRemotePath(props.machineName, row.path, dst)
        ElMessage.success('已移动到上级目录')
        await reload()
      } catch (e) {
        ElMessage.error(String(e))
      }
    }

    const moveToPrompt = async () => {
      const row = ctx.row
      closeMenu()
      if (!row?.path || !props.machineName) return
      try {
        const { value } = await ElMessageBox.prompt('输入目标绝对路径（目录）', '移动到…', {
          confirmButtonText: '移动',
          cancelButtonText: '取消',
          inputValue: parentDirOf(row.path),
          inputPattern: /^\/.*/,
          inputErrorMessage: '请输入绝对路径',
        })
        const destDir = normalizeAbs(value)
        if (!destDir) {
          ElMessage.warning('路径无效')
          return
        }
        const dst = joinRemote(destDir, row.name)
        if (normalizeAbs(dst) === normalizeAbs(row.path)) {
          ElMessage.warning('目标与当前位置相同')
          return
        }
        await App.MoveShellRemotePath(props.machineName, row.path, dst)
        ElMessage.success('已移动')
        await reload()
      } catch (e) {
        if (e === 'cancel') return
        ElMessage.error(String(e))
      }
    }

    const openInTerminal = async () => {
      const row = ctx.row
      closeMenu()
      if (!row?.path || !props.machineName) return
      const dir = row.isDir ? normalizeAbs(row.path) : parentDirOf(row.path)
      try {
        await App.SendShellCd(props.machineName, dir)
        ElMessage.success(`已在终端打开：${dir}`)
      } catch (e) {
        ElMessage.error(String(e))
      }
    }

    const conflictVisible = ref(false)
    const conflictItem = ref(null)
    const conflictApplyCount = ref(1)
    let conflictResolver = null

    const askUploadConflict = (item) => new Promise((resolve) => {
      conflictItem.value = item
      conflictApplyCount.value = Math.max(1, Number(item.applyToAllCount) || 1)
      conflictVisible.value = true
      conflictResolver = resolve
    })

    const onConflictResolve = ({ action, applyToAll }) => {
      conflictVisible.value = false
      const item = conflictItem.value
      conflictItem.value = null
      const fn = conflictResolver
      conflictResolver = null
      fn?.({ action: action || 'skip', applyToAll: !!applyToAll, typeKey: item?.typeKey })
    }

    const startUploads = async (paths) => {
      if (!props.machineName || !expanded.value || !cwd.value) {
        ElMessage.warning('请先打开文件面板并进入目标目录')
        return
      }
      const list = (paths || []).filter(Boolean)
      if (!list.length) return
      let started = 0
      let remembered = null
      const pendingConflicts = []
      for (const localPath of list) {
        const baseName = localPath.replace(/\\/g, '/').split('/').pop()
        const remotePath = joinRemote(cwd.value, baseName)
        try {
          const conflict = await App.CheckShellUploadConflict(props.machineName, localPath, remotePath)
          if (conflict) {
            pendingConflicts.push({
              localPath,
              typeKey: `${conflict.localIsDir ? 'dir' : 'file'}:${conflict.existingType || (conflict.isDir ? 'directory' : 'file')}`,
            })
          }
        } catch { /* ignore */ }
      }
      for (const localPath of list) {
        try {
          const baseName = localPath.replace(/\\/g, '/').split('/').pop()
          const remotePath = joinRemote(cwd.value, baseName)
          let conflict = null
          try {
            conflict = await App.CheckShellUploadConflict(props.machineName, localPath, remotePath)
          } catch { /* ignore */ }
          if (conflict) {
            const typeKey = `${conflict.localIsDir ? 'dir' : 'file'}:${conflict.existingType || (conflict.isDir ? 'directory' : 'file')}`
            const sameTypeLeft = pendingConflicts.filter((c) => c.typeKey === typeKey).length
            const idx = pendingConflicts.findIndex((c) => c.localPath === localPath)
            if (idx >= 0) pendingConflicts.splice(idx, 1)
            let action = 'replace'
            if (remembered?.typeKey === typeKey && remembered?.action) {
              action = remembered.action
            } else {
              const decision = await askUploadConflict({
                fileName: baseName,
                localSize: conflict.localSize,
                remoteSize: conflict.remoteSize,
                remoteMtime: conflict.remoteMtime,
                isDir: !!conflict.isDir,
                localIsDir: !!conflict.localIsDir,
                existingType: conflict.existingType || (conflict.isDir ? 'directory' : 'file'),
                typeKey,
                applyToAllCount: sameTypeLeft,
              })
              action = decision.action
              if (decision.applyToAll) remembered = { action, typeKey }
            }
            if (action === 'skip') continue
            await App.StartShellUpload(props.machineName, localPath, cwd.value, action)
            started++
          } else {
            await App.StartShellUpload(props.machineName, localPath, cwd.value, 'replace')
            started++
          }
        } catch (e) {
          ElMessage.error(`上传失败 (${localPath}): ${e}`)
        }
      }
      if (started > 0) {
        ElMessage.success(`已开始上传 ${started} 项`)
        emit('transfer-started', { direction: 'upload', count: started })
        setTimeout(() => reload(), 1200)
      }
    }

    const onClipboardPaste = async (e) => {
      if (!expanded.value || !props.machineName || !cwd.value) return
      const items = e?.clipboardData?.items
      let imageFile = null
      if (items?.length) {
        for (const it of items) {
          if (it.type?.startsWith('image/')) {
            imageFile = it.getAsFile()
            break
          }
        }
      }
      if (imageFile) {
        e.preventDefault()
        try {
          const buf = await imageFile.arrayBuffer()
          const bytes = new Uint8Array(buf)
          let binary = ''
          for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i])
          const dataUrl = `data:${imageFile.type || 'image/png'};base64,${btoa(binary)}`
          const localPath = await App.SaveClipboardImageForUpload(dataUrl)
          await startUploads([localPath])
        } catch (err) {
          ElMessage.error(`粘贴图片上传失败: ${err}`)
        }
        return
      }

      const localPaths = extractClipboardLocalPaths(e?.clipboardData)
      if (localPaths.length) {
        e.preventDefault()
        try {
          await ElMessageBox.confirm(
            `将上传 ${localPaths.length} 个本地路径到当前目录？`,
            '粘贴上传',
            {
              confirmButtonText: '上传',
              cancelButtonText: '取消',
              type: 'info',
            },
          )
        } catch {
          return
        }
        await startUploads(localPaths)
        return
      }

      if (hasSftpClipboardFor(props.machineName)) {
        e.preventDefault()
        await pasteEntry()
      }
    }

    const onDragOver = (e) => {
      if (!expanded.value || !props.machineName) return
      dragOver.value = true
      if (e?.dataTransfer) e.dataTransfer.dropEffect = 'copy'
    }
    const onDragLeave = (e) => {
      // 避免子元素触发 leave 误关
      const related = e?.relatedTarget
      if (related && e?.currentTarget?.contains?.(related)) return
      dragOver.value = false
    }
    const onHtmlDrop = () => {
      dragOver.value = false
    }

    const isOverDropZone = (x, y) => {
      const el = document.elementFromPoint(x, y)
      return !!el?.closest?.('.shell-drop-zone')
    }

    const bindFileDrop = () => {
      if (dropBound) return
      try {
        // useDropTarget=false：Windows 上仅依赖 CSS 目标时常收不到回调；改为坐标判断落点
        OnFileDrop((x, y, paths) => {
          dragOver.value = false
          if (!expanded.value || !props.machineName) return
          if (!isOverDropZone(x, y)) return
          startUploads(paths)
        }, false)
        dropBound = true
      } catch (e) {
        console.warn('OnFileDrop 绑定失败:', e)
      }
    }

    const unbindFileDrop = () => {
      if (!dropBound) return
      try {
        OnFileDropOff()
      } catch { /* ignore */ }
      dropBound = false
    }

    const onUploadCommand = async (command) => {
      if (!props.machineName || !expanded.value || !cwd.value) {
        ElMessage.warning('请先打开文件面板并进入目标目录')
        return
      }
      try {
        if (command === 'folder') {
          const dir = await App.PickShellUploadFolder()
          if (!dir) return
          await startUploads([dir])
        } else {
          const files = await App.PickShellUploadPaths()
          if (!files?.length) return
          await startUploads(files)
        }
      } catch (e) {
        ElMessage.error('选择文件失败: ' + e)
      }
    }

    const openSyncDialog = () => {
      if (!props.machineName) {
        ElMessage.warning('请先连接远程会话')
        return
      }
      syncVisible.value = true
    }

    const onSyncDialogOpen = () => {
      if (!syncForm.remoteDir) syncForm.remoteDir = cwd.value || '/'
    }

    const pickSyncLocalDir = async () => {
      try {
        const dir = await App.PickShellUploadFolder()
        if (dir) syncForm.localDir = dir
      } catch (e) {
        ElMessage.error('选择目录失败: ' + e)
      }
    }

    const startFolderSync = async () => {
      const localDir = String(syncForm.localDir || '').trim()
      const remoteDir = String(syncForm.remoteDir || '').trim()
      if (!localDir || !remoteDir) {
        ElMessage.warning('请填写本地与远端目录')
        return
      }
      if (!props.machineName) {
        ElMessage.warning('请先连接远程会话')
        return
      }
      syncStarting.value = true
      try {
        await App.StartShellFolderSync(
          props.machineName,
          localDir,
          remoteDir,
          syncForm.direction || 'upload',
        )
        ElMessage.success('已开始文件夹同步')
        syncVisible.value = false
        emit('transfer-started', { direction: 'sync', count: 1 })
      } catch (e) {
        ElMessage.error('同步失败: ' + e)
      } finally {
        syncStarting.value = false
      }
    }

    const deleteEntry = async () => {
      const row = ctx.row || selectedRow.value
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
        if (selectedRow.value?.path === row.path) selectedRow.value = null
        await reload()
      } catch (e) {
        ElMessage.error('删除失败: ' + e)
      }
    }

    const renamePrimary = async () => {
      const row = ctx.row || selectedRow.value
      if (!row) return
      ctx.row = row
      await promptRename()
    }
    const onTableKeydown = (e) => {
      const tag = String(e.target?.tagName || '').toLowerCase()
      if (tag === 'input' || tag === 'textarea') return
      const mod = e.ctrlKey || e.metaKey
      if (mod && !e.shiftKey && !e.altKey && (e.key === 'c' || e.key === 'C')) {
        e.preventDefault()
        void copyEntry()
        return
      }
      if (mod && !e.shiftKey && !e.altKey && (e.key === 'x' || e.key === 'X')) {
        e.preventDefault()
        void cutEntry()
        return
      }
      if (mod && !e.shiftKey && !e.altKey && (e.key === 'v' || e.key === 'V')) {
        e.preventDefault()
        void pasteEntry()
        return
      }
      if (e.key === 'F5' || (mod && !e.shiftKey && !e.altKey && (e.key === 'r' || e.key === 'R'))) {
        e.preventDefault()
        void reload()
        return
      }
      if (mod && e.shiftKey && !e.altKey && (e.key === 'n' || e.key === 'N')) {
        e.preventDefault()
        void promptMkdir()
        return
      }
      if (e.key === 'Enter' && !mod && !e.altKey) {
        e.preventDefault()
        if (selectedRow.value) void onOpen(selectedRow.value)
        return
      }
      if (e.key === 'F2') {
        e.preventDefault()
        void renamePrimary()
        return
      }
      if (e.key === 'Delete' || (e.key === 'Backspace' && mod)) {
        e.preventDefault()
        void deleteEntry()
        return
      }
      if (e.key === 'Backspace' && !mod && !e.altKey) {
        e.preventDefault()
        void goParent()
      }
    }

    watch(() => props.machineName, async (name, prev) => {
      snapshotFollowState(prev)
      snapshotBrowseView(prev)
      closeMenu()
      stopPwdTimer()
      const key = String(name || '').trim()
      lastFollowedCwd.value = lastFollowedByMachine[key] || ''
      terminalCwd.value = terminalCwdByMachine[key] || ''
      followCwd.value = loadFollowCwd(key)
      refreshBookmarks()
      const shouldExpand = !!(name && expandedByMachine[name])
      if (expanded.value !== shouldExpand) {
        expanded.value = shouldExpand
      } else if (name) {
        // 展开态未变时也同步状态栏指示
        emit('update:expanded', shouldExpand)
      }
      const restored = restoreBrowseView(key)
      if (!restored) {
        cwd.value = ''
        pathDraft.value = ''
        entries.value = []
        selectedRow.value = null
        nameFilter.value = ''
        treeRoot.value = []
        expandedKeys.value = ['/']
      }
      await nextTick()
      notifyLayout()
      if (!name || !shouldExpand) return
      try {
        let liveTerm = ''
        try {
          const remote = await App.GetShellPtyCwd(name)
          if (remote && String(remote).startsWith('/')) liveTerm = normalizeAbs(remote)
        } catch {
          // 使用 cwdHint / home
        }
        if (!liveTerm) {
          liveTerm = (props.cwdHint && String(props.cwdHint).startsWith('/'))
            ? normalizeAbs(props.cwdHint)
            : ''
        }
        if (liveTerm) terminalCwd.value = liveTerm
        if (props.machineName !== name) return
        const savedBrowse = normalizeAbs(browseCwdByMachine[key] || cwd.value || '')
        const savedFollowed = normalizeAbs(lastFollowedCwd.value)
        const stillPaused = !!(
          followCwd.value
          && savedFollowed
          && savedBrowse
          && savedBrowse !== savedFollowed
          && (!liveTerm || liveTerm === savedFollowed)
        )
        const applyTarget = async (target, { markFollowed = false } = {}) => {
          if (props.machineName !== name) return
          const abs = normalizeAbs(target)
          if (!abs) return
          if (shouldSilentSftpRefresh({
            restored,
            cwd: normalizeAbs(cwd.value),
            target: abs,
            treeHasNodes: (treeRoot.value || []).length > 0,
          })) {
            await calibrateList()
          } else {
            await setCwd(abs)
          }
          if (props.machineName !== name) return
          if (markFollowed) markFollowedCwd(abs)
        }
        if (stillPaused) {
          await applyTarget(savedBrowse)
        } else if (followCwd.value && liveTerm) {
          await applyTarget(liveTerm, { markFollowed: true })
        } else {
          await applyTarget(savedBrowse || liveTerm || await ensureHome())
        }
        if (props.machineName !== name) return
        startPwdTimer()
      } catch (e) {
        setPanelError(e)
      }
    })

    let offExternalEdit = null
    let offSystemSettings = null

    onMounted(() => {
      bindFileDrop()
      window.addEventListener('paste', onClipboardPaste)
      void loadSftpOpenConfig()
      offExternalEdit = EventsOn('shell:external-edit', (payload) => {
        if (!payload || payload.machineName !== props.machineName) return
        if (payload.status === 'uploaded') {
          reload()
        } else if (payload.status === 'error' && payload.message) {
          ElMessage.error(payload.message)
        }
      })
      offSystemSettings = EventsOn('system-settings:changed', (payload) => {
        applySftpOpenConfig(payload)
      })
    })

    onUnmounted(() => {
      stopPwdTimer()
      closeMenu()
      unbindFileDrop()
      window.removeEventListener('paste', onClipboardPaste)
      offExternalEdit?.()
      offExternalEdit = null
      offSystemSettings?.()
      offSystemSettings = null
    })

    expose({
      toggle,
      applyCwdHint,
      startUploads,
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
      panelRef,
      expanded,
      showHidden,
      followCwd,
      followPaused,
      lastFollowedCwd,
      terminalCwd,
      bookmarks,
      isCurrentBookmarked,
      bookmarkButtonTitle,
      pathSuggestOpen,
      pathSuggestIndex,
      pathSuggestions,
      cwd,
      pathDraft,
      entries,
      displayEntries,
      nameFilter,
      sortKey,
      sortAsc,
      toggleSort,
      selectedRow,
      onRowClick,
      onCurrentChange,
      canPaste,
      colVisible,
      onTableKeydown,
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
      ctxMenuRef,
      dragOver,
      localSearchQuery,
      searchInputRef,
      conflictVisible,
      conflictItem,
      conflictApplyCount,
      onConflictResolve,
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
      goToTerminalCwd,
      resumeFollowToTerminal,
      submitPathDraft,
      syncPathDraftFromCwd,
      onPathDraftInput,
      movePathSuggest,
      applyPathSuggestion,
      onPathEnter,
      onPathBlur,
      pathSuggestTypeLabel,
      onFollowCwdChange,
      onBookmarkButtonClick,
      onBookmarkCommand,
      startHeightResize,
      startResize,
      onContextMenu,
      onBlankContextMenu,
      closeMenu,
      copyPath,
      downloadEntry,
      editEntry,
      saveEditor,
      openEntry,
      openRowFromMenu,
      openWithEntry,
      openWithSystemDefaultEntry,
      isBinaryRow,
      openerVisible,
      openerFileName,
      onOpenerSelect,
      editorVisible,
      editorContent,
      editorTitle,
      editorSaving,
      promptMkdir,
      promptNewFile,
      reloadFromMenu,
      uploadFilesFromMenu,
      uploadFolderFromMenu,
      promptRename,
      promptChmod,
      submitChmod,
      copyEntry,
      cutEntry,
      pasteEntry,
      copyHere,
      copyToOtherSide,
      copyToOtherTargets: computed(() => props.copyToOtherTargets || []),
      copyOtherSubOpen,
      moveToParent,
      moveToPrompt,
      openInTerminal,
      chmodVisible,
      chmodName,
      chmodPerms,
      chmodOctal,
      chmodSymbolic,
      permRoles,
      permBits,
      chmodSaving,
      deleteEntry,
      onDragOver,
      onDragLeave,
      onHtmlDrop,
      onUploadCommand,
      openSyncDialog,
      onSyncDialogOpen,
      pickSyncLocalDir,
      startFolderSync,
      syncVisible,
      syncStarting,
      syncForm,
    }
  },
}
</script>

<style scoped>
.shell-file-panel {
  position: relative;
  flex-shrink: 0;
  border-top: 1px solid var(--shell-chrome-border, var(--app-border));
  background: var(--shell-chrome-bg, var(--app-panel-bg));
  display: flex;
  flex-direction: column;
  max-height: none;
  box-shadow: inset 0 1px 0 var(--shell-chrome-highlight, transparent);
}

.shell-file-panel.collapsed {
  overflow: visible;
}

.shell-file-panel.is-bare {
  border-top: none;
  background: transparent;
  box-shadow: none;
}

/* 仅搜索：与 SFTP 工具条一样占文档流，不挡住终端 */
.shell-file-panel.has-search.collapsed .search-bar {
  border-bottom: none;
}

.height-handle {
  height: 10px;
  cursor: row-resize;
  flex-shrink: 0;
  background: transparent;
  border-bottom: 1px solid transparent;
}

.height-handle.is-collapsed-edge {
  position: absolute;
  top: -3px;
  left: 0;
  right: 0;
  height: 8px;
  z-index: 5;
}

.height-handle:hover {
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 8%, transparent);
}

.file-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 8px 6px 10px;
  min-height: 32px;
  flex-shrink: 0;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  flex: 1;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
  margin-left: 4px;
}

.name-filter {
  width: 140px;
  margin-right: 4px;
}

.name-filter :deep(.el-input__wrapper) {
  min-height: 26px;
  height: 26px;
  font-size: 12px;
}

.sortable-th {
  cursor: pointer;
  user-select: none;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.sortable-th:hover {
  color: var(--app-accent-color, #409eff);
}

.sort-ind {
  font-size: 11px;
  opacity: 0.85;
}

.tool-icon-btn {
  width: 26px;
  height: 26px;
  min-height: 26px;
  min-width: 26px;
  padding: 0;
  margin: 0;
  border-radius: var(--app-radius-sm, 6px);
  color: var(--app-text-secondary);
}

.tool-icon-btn :deep(.el-icon) {
  font-size: 14px;
}

.tool-icon-btn:hover {
  color: var(--app-accent-color, #409eff);
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 12%, transparent);
}

.hidden-check {
  margin: 0 4px 0 6px;
  height: 26px;
  display: inline-flex;
  align-items: center;
}

.hidden-check :deep(.el-checkbox__label) {
  font-size: 12px;
  padding-left: 6px;
  line-height: 26px;
}

.search-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px 6px 10px;
  min-height: 32px;
  box-sizing: border-box;
  border-bottom: 1px solid var(--shell-chrome-divider, var(--app-border));
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

.cwd-wrap {
  position: relative;
  flex: 1;
  min-width: 120px;
  max-width: 480px;
}

.cwd-wrap .cwd-input {
  max-width: none;
  width: 100%;
}

.path-suggest {
  position: absolute;
  left: 0;
  right: 0;
  top: calc(100% + 2px);
  z-index: 30;
  margin: 0;
  padding: 4px 0;
  list-style: none;
  max-height: 220px;
  overflow: auto;
  border-radius: 6px;
  border: 1px solid var(--app-border, #e4e7ed);
  background: var(--app-card-bg, #fff);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

.path-suggest li {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  cursor: pointer;
  font-size: 12px;
}

.path-suggest li.active,
.path-suggest li:hover {
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 12%, transparent);
}

.path-suggest-type {
  flex-shrink: 0;
  color: var(--app-text-muted, #909399);
  min-width: 2.5em;
}

.path-suggest-path {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--app-text, #303133);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tool-icon-btn.is-bookmarked {
  color: var(--el-color-warning, #e6a23c);
}

.bm-row {
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: 320px;
}

.bm-path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.bm-tag {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--app-text-muted, #909399);
}

.follow-check {
  margin-left: 4px;
  white-space: nowrap;
}

.follow-check.is-paused {
  opacity: 0.55;
}

.follow-resume-btn {
  margin-left: 0;
  height: 26px;
  padding: 0 6px;
  font-size: 12px;
}

.cwd-input :deep(.el-input__wrapper) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  min-height: 26px;
  height: 26px;
  padding: 0 8px;
}

.file-body {
  display: flex;
  flex-shrink: 0;
  border-top: 1px solid var(--shell-chrome-divider, var(--app-border));
  background: var(--app-panel-bg);
  overflow: hidden;
  --wails-drop-target: drop;
}

.tree-pane {
  flex-shrink: 0;
  overflow: auto;
  padding: 6px;
  box-sizing: border-box;
}

.tree-pane :deep(.el-tree) {
  background: transparent;
  color: var(--app-text);
  --el-tree-node-hover-bg-color: var(--app-accent-bg);
  --el-tree-text-color: var(--app-text);
}

.tree-pane :deep(.el-tree-node__content) {
  border-radius: 4px;
  height: 28px;
}

.tree-pane :deep(.el-tree-node__content:focus),
.tree-pane :deep(.el-tree-node__content:focus-visible) {
  outline: none;
}

.tree-pane :deep(.el-tree--highlight-current .el-tree-node.is-current > .el-tree-node__content) {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
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
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 25%, transparent);
  border-right-color: color-mix(in srgb, var(--app-accent-color, #409eff) 35%, transparent);
}

.list-pane {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  padding: 0;
  position: relative;
}

.list-pane.drag-over {
  outline: 2px dashed var(--app-accent-color, #409eff);
  outline-offset: -4px;
}

.drop-overlay {
  position: absolute;
  inset: 0;
  z-index: 5;
  display: flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 12%, transparent);
  color: var(--app-accent-color, #409eff);
  font-size: 14px;
  font-weight: 600;
  pointer-events: none;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.link-target {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
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
  z-index: 5000;
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

.ctx-menu li.ctx-sep {
  height: 1px;
  margin: 4px 8px;
  padding: 0;
  background: var(--app-border, #e4e7ed);
  pointer-events: none;
  cursor: default;
}

.ctx-menu li.ctx-sep:hover {
  background: var(--app-border, #e4e7ed);
  color: inherit;
}

.ctx-menu li.ctx-has-sub {
  position: relative;
  padding-right: 28px;
}

.ctx-menu li.ctx-has-sub::after {
  content: '›';
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--app-text-secondary);
}

.ctx-submenu {
  position: absolute;
  left: 100%;
  top: -4px;
  margin: 0 0 0 2px;
  padding: 4px 0;
  list-style: none;
  min-width: 200px;
  max-width: 360px;
  background: var(--app-card-bg);
  border: 1px solid var(--app-border);
  border-radius: 6px;
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.18);
  z-index: 5001;
}

.ctx-submenu li {
  padding: 8px 14px;
  font-size: 13px;
  color: var(--app-text);
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.ctx-submenu li:hover {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}
</style>

<style>
/* append-to-body 弹框，需非 scoped */
.sftp-perm-dialog .el-dialog__header {
  padding: 16px 20px 8px;
  margin-right: 0;
}

.sftp-perm-dialog .el-dialog__body {
  padding: 8px 20px 12px;
}

.sftp-perm-dialog .el-dialog__footer {
  padding: 8px 20px 16px;
}

.sftp-perm-dialog .perm-header {
  padding-right: 24px;
}

.sftp-perm-dialog .perm-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary, #303133);
  line-height: 1.4;
}

.sftp-perm-dialog .perm-filename {
  margin-top: 2px;
  font-size: 13px;
  color: var(--el-text-color-secondary, #909399);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sftp-perm-dialog .perm-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 8px 0 4px;
}

.sftp-perm-dialog .perm-row {
  display: flex;
  align-items: center;
  gap: 16px;
}

.sftp-perm-dialog .perm-role {
  width: 56px;
  flex-shrink: 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary, #303133);
}

.sftp-perm-dialog .perm-checks {
  display: flex;
  align-items: center;
  gap: 18px;
}

.sftp-perm-dialog .perm-check {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  font-size: 13px;
  color: var(--el-text-color-regular, #606266);
  user-select: none;
}

.sftp-perm-dialog .perm-check .el-checkbox {
  height: auto;
  margin-right: 0;
}

.sftp-perm-dialog .perm-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 4px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter, #ebeef5);
  font-size: 12px;
  color: var(--el-text-color-secondary, #909399);
}

.sftp-perm-dialog .perm-summary code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--el-text-color-primary, #303133);
  font-size: 12px;
}

.sync-path-row {
  display: flex;
  gap: 8px;
  width: 100%;
}

.sync-path-row .el-input {
  flex: 1;
}

.sync-hint {
  margin: 0 0 0 88px;
  font-size: 12px;
  color: var(--app-text-muted);
  line-height: 1.4;
}
</style>
