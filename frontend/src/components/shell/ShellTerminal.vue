<template>
  <div
    class="shell-terminal"
    ref="containerRef"
    @contextmenu.prevent="onContextMenu"
    @mousedown="onPaneFocus"
  >
    <div ref="terminalRef" class="terminal-host"></div>
    <!-- 与底部状态栏 / SFTP 面板留出空隙，避免提示符贴边 -->
    <div class="terminal-bottom-gap" aria-hidden="true" />
    <ShellConnectionOverlay
      :status="overlayStatus"
      :machine-name="displayName"
      :host="host"
      :user="user"
      :jump-chain="jumpChain"
      :proxy-jump="proxyJump"
      :reconnect-attempt="reconnectAttempt"
      :reconnect-max="reconnectMax"
      :reconnect-delay-sec="reconnectDelaySec"
      @reconnect="$emit('reconnect', machineName)"
    />
    <div v-if="passwordAssistVisible" class="shell-password-assist" @mousedown.stop @click.stop>
      <span class="shell-password-assist-label">密码</span>
      <input
        ref="passwordAssistInputRef"
        v-model="passwordAssistValue"
        class="shell-password-assist-input"
        type="password"
        autocomplete="off"
        autocorrect="off"
        autocapitalize="off"
        spellcheck="false"
        placeholder="输入后回车发送（不记日志）"
        @keydown.enter.prevent="sendPasswordAssist"
        @keydown.esc.prevent="dismissPasswordAssist"
      />
      <button type="button" class="shell-password-assist-send" @click="sendPasswordAssist">发送</button>
      <button type="button" class="shell-password-assist-dismiss" title="关闭" @click="dismissPasswordAssist">×</button>
    </div>
    <Teleport to="body">
      <ul
        v-if="ctx.visible"
        ref="ctxMenuRef"
        class="ctx-menu"
        :style="{ left: ctx.x + 'px', top: ctx.y + 'px' }"
        @click.stop
        @mouseleave="hideContextMenu"
      >
        <li @click="onCopy">复制</li>
        <li @click="onPaste">粘贴</li>
        <li @click="onFind">查找</li>
        <li class="danger" @click="onClearCache">清空缓存</li>
        <template v-if="inSplit">
          <li class="sep" role="separator"></li>
          <li @click="onRemoveFromSplit">移出分屏</li>
          <li @click="onExitSplit">取消全部分屏</li>
        </template>
      </ul>
    </Teleport>
  </div>
</template>

<script>
import { ref, reactive, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { WebglAddon } from 'xterm-addon-webgl'
import { getTerminalSelectionText } from '../../utils/shellSelection'
import { findInTerminalBuffer, selectTerminalMatch, highlightViewportMatches, clearSearchDecorations, lineToSearchModel, charRangeToCells, indexOfSearchMatch } from '../../utils/shellTerminalSearch'
import 'xterm/css/xterm.css'
import * as App from '../../../wailsjs/go/app/App'
import { EventsOn } from '../../../wailsjs/runtime/runtime'
import {
  registerShellWriter,
  resetShellWriterReplay,
  setShellOutputSessionActive,
} from '../../utils/shellOutputBuffer'
import { SHELL_TERMINAL_SCROLLBACK, clampShellTerminalScrollback } from '../../constants/shellMemory'
import { useTheme } from '../../composables/useTheme'
import { terminalThemeForPreset, getTerminalFont } from '../../utils/themePresets'
import {
  highlightShellChunk,
  isProbablyBinary,
  mergeLogHighlightConfig,
  updateTuiModeDepth,
} from '../../utils/shellLogHighlight'
import {
  ensureShellAsciiInputListeners,
  notifyShellTerminalBlur,
  notifyShellTerminalFocus,
  setShellAsciiInputEnabled,
} from '../../utils/shellAsciiInput'
import {
  decodeOsc52ClipboardPayload,
  prefixLineTimestamps,
  looksLikePasswordPrompt,
} from '../../utils/shellTerminalUx'
import { ClipboardSetText } from '../../../wailsjs/runtime/runtime'
import ShellConnectionOverlay from './ShellConnectionOverlay.vue'

export default {
  name: 'ShellTerminal',
  components: { ShellConnectionOverlay },
    props: {
    machineName: { type: String, required: true },
    configName: { type: String, default: '' },
    active: { type: Boolean, default: false },
    /** Shell 工作区是否可见；隐藏时不 fit/不转发输入，但保留 xterm 以免回放协议应答 */
    viewVisible: { type: Boolean, default: true },
    connected: { type: Boolean, default: false },
    connecting: { type: Boolean, default: false },
    tabLabel: { type: String, default: '' },
    host: { type: String, default: '' },
    user: { type: String, default: '' },
    jumpChain: { type: Array, default: () => [] },
    proxyJump: { type: String, default: '' },
    searchQuery: { type: String, default: '' },
    /** 当前是否处于分屏窗格中 */
    inSplit: { type: Boolean, default: false },
    /** 是否曾成功连接过（用于区分「尚未连上」与「已断开」） */
    everConnected: { type: Boolean, default: false },
    /** 自动重连中 */
    reconnecting: { type: Boolean, default: false },
    reconnectAttempt: { type: Number, default: 0 },
    reconnectMax: { type: Number, default: 0 },
    reconnectDelaySec: { type: Number, default: 0 },
    /** 本机终端配色覆盖；空则跟随全局 */
    terminalPresetOverride: { type: String, default: '' },
  },
  emits: [
    'open-search', 'clear-cache', 'reconnect', 'search-result', 'cwd-sync',
    'remove-from-split', 'exit-split', 'focus-session',
  ],
  setup(props, { expose, emit }) {
    const containerRef = ref(null)
    const terminalRef = ref(null)
    const ctxMenuRef = ref(null)
    const term = ref(null)
    const fitAddon = ref(null)
    const ctx = reactive({ visible: false, x: 0, y: 0, selection: '' })
    const { shellFontSize, shellLineHeight, terminalPreset, shellFontFamily } = useTheme()
    const displayName = computed(() => props.tabLabel || props.machineName || '')
    const effectiveTerminalPreset = computed(() => {
      const override = String(props.terminalPresetOverride || '').trim()
      return override || terminalPreset.value
    })
    const overlayStatus = computed(() => {
      if (props.reconnecting) return 'reconnecting'
      if (props.connecting) return 'connecting'
      if (!props.connected && props.everConnected) return 'disconnected'
      return ''
    })
    const passwordAssistVisible = ref(false)
    const passwordAssistValue = ref('')
    const passwordAssistInputRef = ref(null)
    let passwordAssistEnabled = true
    let passwordDetectTail = ''
    let passwordAssistDismissedUntil = 0
    const PASSWORD_ASSIST_DISMISS_MS = 8000
    const PASSWORD_DETECT_TAIL_MAX = 400

    const dismissPasswordAssist = () => {
      passwordAssistVisible.value = false
      passwordAssistValue.value = ''
      passwordAssistDismissedUntil = Date.now() + PASSWORD_ASSIST_DISMISS_MS
    }

    const showPasswordAssist = () => {
      if (!passwordAssistEnabled) return
      if (!props.connected) return
      if (suppressInputForwardDepth > 0) return
      if (passwordAssistVisible.value) return
      if (Date.now() < passwordAssistDismissedUntil) return
      passwordAssistValue.value = ''
      passwordAssistVisible.value = true
      nextTick(() => {
        passwordAssistInputRef.value?.focus?.()
      })
    }

    const sendPasswordAssist = async () => {
      const value = passwordAssistValue.value
      passwordAssistVisible.value = false
      passwordAssistValue.value = ''
      passwordDetectTail = ''
      // 短冷却，避免同一提示立刻再次弹出
      passwordAssistDismissedUntil = Date.now() + 1500
      if (!props.connected || !value) return
      try {
        await App.SendShellInput(props.machineName, `${value}\r`)
      } catch {
        // ignore
      }
    }

    const notePasswordPromptText = (text) => {
      if (!passwordAssistEnabled || !text) return
      if (suppressInputForwardDepth > 0) return
      if (!props.connected) return
      if (passwordAssistVisible.value) return
      if (Date.now() < passwordAssistDismissedUntil) return
      passwordDetectTail = (passwordDetectTail + text).slice(-PASSWORD_DETECT_TAIL_MAX)
      if (looksLikePasswordPrompt(passwordDetectTail)) {
        showPasswordAssist()
      }
    }

    let resizeObserver = null
    let fitTimers = []
    let fitDebounceTimer = null
    let fitRaf1 = 0
    let fitRaf2 = 0
    let lastFitCols = 0
    let lastFitRows = 0
    let fitQuietUntil = 0
    let suppressRoUntil = 0
    let initialized = false
    let unregisterWriter = null
    let inputListener = null
    /** 缓冲回放期间抑制 onData→PTY，避免 DA/OSC 颜色应答被当成键盘输入 */
    let suppressInputForwardDepth = 0
    let lastSearchResults = { resultIndex: -1, resultCount: 0, capped: false }
    let cwdSyncTimer = null
    let inputLine = ''
    let logHighlightEnabled = true
    let logHighlightConfig = mergeLogHighlightConfig(null)
    let tuiModeDepth = 0
    let scrollbackLines = SHELL_TERMINAL_SCROLLBACK
    let cursorLineHighlightEnabled = false
    let lineTimestampsEnabled = false
    /** WebGL 渲染（默认关闭；失败回退 canvas） */
    let shellUseWebgl = false
    /** 非活动且非分屏时休眠（默认开启） */
    let shellTabHibernate = true
    let webglAddon = null
    let hibernateTimer = null
    /** 休眠后延迟销毁 xterm DOM（毫秒）；0 表示立即销毁 */
    const HIBERNATE_DISPOSE_DELAY_MS = 0
    /** 行时间戳：是否处于行首（跳过回放时由 aroundReplay 重置） */
    const lineTsState = { atLineStart: true }
    let cursorLineDecoration = null
    let cursorLineMarker = null
    let cursorMoveListener = null
    let osc52Disposable = null
    const textDecoder = new TextDecoder('utf-8')
    /**
     * 不用 xterm-addon-search：它对超长折行（grep 出的整行 JSON）会拼成巨串同步扫描，必卡死。
     * 改为按物理行查找 + 异步分片计数。
     */
    const SEARCH_COUNT_CAP = 999
    let searchCountToken = 0
    let searchCountTimer = null
    let lastCountedQuery = ''
    /** 异步计数是否完成（完成前不报序号，避免大缓冲同步扫） */
    let searchIndexReady = false
    /** @type {Array<{ row: number, col: number }>} */
    let searchMatchList = []
    let activeSearchQuery = ''
    let lastActiveMatch = null
    /** @type {Array<{ dispose?: Function }>} */
    let searchDecorationBucket = []
    let searchHighlightTimer = null
    let scrollListener = null

    const clearViewportHighlights = () => {
      if (searchHighlightTimer != null) {
        clearTimeout(searchHighlightTimer)
        searchHighlightTimer = null
      }
      clearSearchDecorations(searchDecorationBucket)
    }

    const refreshViewportHighlights = (query = activeSearchQuery, activeMatch = lastActiveMatch) => {
      const terminal = term.value
      if (!terminal || !query) {
        clearViewportHighlights()
        return
      }
      highlightViewportMatches(terminal, query, activeMatch, searchDecorationBucket)
    }

    const scheduleViewportHighlights = () => {
      if (!activeSearchQuery || !term.value) return
      if (searchHighlightTimer != null) clearTimeout(searchHighlightTimer)
      searchHighlightTimer = setTimeout(() => {
        searchHighlightTimer = null
        refreshViewportHighlights()
      }, 50)
    }

    const cancelAsyncMatchCount = () => {
      searchCountToken += 1
      if (searchCountTimer != null) {
        clearTimeout(searchCountTimer)
        searchCountTimer = null
      }
    }

    const resetSearchMatchIndex = () => {
      searchIndexReady = false
      searchMatchList = []
    }

    const resolveSearchIndex = (match) => {
      if (!searchIndexReady || !match) return -1
      return indexOfSearchMatch(searchMatchList, match)
    }

    const emitSearchResult = (payload) => {
      lastSearchResults = {
        resultIndex: payload.resultIndex ?? -1,
        resultCount: payload.resultCount ?? 0,
        capped: !!payload.capped,
      }
      if (!props.active) return
      emit('search-result', {
        found: !!payload.found,
        ...lastSearchResults,
      })
    }

    /**
     * 分片扫描缓冲：统计总数并缓存匹配坐标（≤999）。
     * 不阻塞导航；完成后用二分得到「第几处」，大日志只吃空闲时间片。
     */
    const scheduleAsyncMatchCount = (query) => {
      if (!query || query === lastCountedQuery) return
      cancelAsyncMatchCount()
      lastCountedQuery = query
      resetSearchMatchIndex()
      const token = searchCountToken
      const terminal = term.value
      if (!terminal) return

      const needle = query.toLowerCase()
      const needleLen = needle.length
      const buf = terminal.buffer.active
      const totalRows = buf.length
      let row = 0
      /** @type {Array<{ row: number, col: number }>} */
      const matches = []

      const finish = (capped) => {
        if (token !== searchCountToken) return
        searchMatchList = matches
        searchIndexReady = true
        const active = lastActiveMatch
        const idx = active && query === activeSearchQuery
          ? indexOfSearchMatch(matches, active)
          : -1
        emitSearchResult({
          found: matches.length > 0 || !!active,
          resultIndex: idx,
          resultCount: matches.length,
          capped,
        })
      }

      const step = () => {
        if (token !== searchCountToken || !term.value) return
        const deadline = performance.now() + 6
        while (row < totalRows) {
          const line = buf.getLine(row)
          const rowIndex = row
          row += 1
          if (!line) continue
          // 与查找高亮同一套单元格映射，避免 trim/宽字符导致计数与选区不一致
          const model = lineToSearchModel(line)
          const hay = model.text.toLowerCase()
          if (!hay) continue
          let from = 0
          while (matches.length < SEARCH_COUNT_CAP) {
            const at = hay.indexOf(needle, from)
            if (at < 0) break
            const { col } = charRangeToCells(model, at, needleLen)
            matches.push({ row: rowIndex, col })
            from = at + Math.max(1, needleLen)
          }
          if (matches.length >= SEARCH_COUNT_CAP) {
            finish(true)
            return
          }
          if (performance.now() >= deadline) break
        }
        if (row >= totalRows) {
          finish(false)
          return
        }
        searchCountTimer = setTimeout(step, 0)
      }
      searchCountTimer = setTimeout(step, 0)
    }

    const hideContextMenu = () => {
      ctx.visible = false
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
      if (y + rect.height > vh - pad) {
        y = Math.max(pad, vh - rect.height - pad)
      }
      ctx.x = x
      ctx.y = y
    }

    const onContextMenu = (e) => {
      // 右键时先保存选区：点击菜单项可能触发 mousedown 清掉 xterm 选区
      ctx.selection = getTerminalSelectionText(term.value)
      ctx.x = e.clientX
      ctx.y = e.clientY
      ctx.visible = true
      adjustContextMenuPosition()
    }

    const onCopy = async () => {
      hideContextMenu()
      const text = ctx.selection || getTerminalSelectionText(term.value)
      if (!text) return
      try {
        await navigator.clipboard.writeText(text)
      } catch {
        // 静默失败：终端复制不弹轻提示
      }
    }

    const onPaste = async () => {
      hideContextMenu()
      await pasteClipboard()
    }

    const pasteClipboard = async () => {
      if (!props.connected) return false
      try {
        const text = await navigator.clipboard.readText()
        if (!text) return false
        await App.SendShellInput(props.machineName, text)
        term.value?.focus?.()
        return true
      } catch {
        return false
      }
    }

    const getSelection = () => getTerminalSelectionText(term.value)

    const onFind = () => {
      const selected = ctx.selection || getSelection()
      hideContextMenu()
      nextTick(() => emit('open-search', selected))
    }

    const onClearCache = () => {
      hideContextMenu()
      term.value?.clear?.()
      emit('clear-cache', props.machineName)
      App.ClearShellOutput(props.machineName).catch(() => {})
    }

    const onRemoveFromSplit = () => {
      hideContextMenu()
      emit('remove-from-split', props.machineName)
    }

    const onExitSplit = () => {
      hideContextMenu()
      emit('exit-split')
    }

    const onPaneFocus = () => {
      emit('focus-session', props.machineName)
    }

    const onTermAsciiFocus = () => {
      if (!props.active || !props.viewVisible) return
      notifyShellTerminalFocus()
    }

    const onTermAsciiBlur = () => {
      notifyShellTerminalBlur()
    }

    const bindAsciiInputListeners = (terminal) => {
      unbindAsciiInputListeners(terminal)
      const el = terminal?.textarea
      if (!el) return
      el.addEventListener('focus', onTermAsciiFocus)
      el.addEventListener('blur', onTermAsciiBlur)
      terminal._flashdockAsciiFocus = onTermAsciiFocus
      terminal._flashdockAsciiBlur = onTermAsciiBlur
      terminal._flashdockAsciiEl = el
      if (document.activeElement === el && props.active && props.viewVisible) {
        notifyShellTerminalFocus()
      }
    }

    const unbindAsciiInputListeners = (terminal) => {
      const el = terminal?._flashdockAsciiEl
      const onFocus = terminal?._flashdockAsciiFocus
      const onBlur = terminal?._flashdockAsciiBlur
      if (el && onFocus) el.removeEventListener('focus', onFocus)
      if (el && onBlur) el.removeEventListener('blur', onBlur)
      if (terminal) {
        terminal._flashdockAsciiEl = null
        terminal._flashdockAsciiFocus = null
        terminal._flashdockAsciiBlur = null
      }
    }

    const decodeBase64 = (b64) => {
      if (b64 instanceof Uint8Array) return b64
      const binary = atob(b64)
      const bytes = new Uint8Array(binary.length)
      for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i)
      }
      return bytes
    }

    const writeTerminal = (terminal, data) => new Promise((resolve) => {
      try {
        terminal.write(data, () => resolve())
      } catch {
        resolve()
      }
    })

    const writeDataToTerminal = (terminal, data) => {
      const bytes = data instanceof Uint8Array ? data : decodeBase64(data)
      return writeHighlighted(terminal, bytes)
    }

    const applyLineTimestampsIfNeeded = (text) => {
      if (!lineTimestampsEnabled || tuiModeDepth > 0 || !text) return text
      return prefixLineTimestamps(text, lineTsState)
    }

    const writeHighlighted = (terminal, bytes) => {
      const raw = bytes instanceof Uint8Array ? bytes : decodeBase64(bytes)
      if (isProbablyBinary(raw)) {
        return writeTerminal(terminal, raw)
      }
      let text = ''
      try {
        text = textDecoder.decode(raw, { stream: true })
      } catch {
        return writeTerminal(terminal, raw)
      }
      tuiModeDepth = updateTuiModeDepth(tuiModeDepth, text)
      notePasswordPromptText(text)
      if (!logHighlightEnabled) {
        return writeTerminal(terminal, applyLineTimestampsIfNeeded(text))
      }
      try {
        // 不因 less/TUI 整块跳过：按行高亮，含交互序列的行仍原样透传
        const highlighted = highlightShellChunk(text, logHighlightConfig)
        return writeTerminal(terminal, applyLineTimestampsIfNeeded(highlighted))
      } catch {
        return writeTerminal(terminal, applyLineTimestampsIfNeeded(text))
      }
    }

    const clearCursorLineHighlight = () => {
      try {
        cursorLineDecoration?.dispose?.()
      } catch {
        // ignore
      }
      cursorLineDecoration = null
      try {
        cursorLineMarker?.dispose?.()
      } catch {
        // ignore
      }
      cursorLineMarker = null
    }

    const refreshCursorLineHighlight = () => {
      const terminal = term.value
      clearCursorLineHighlight()
      if (!terminal || !cursorLineHighlightEnabled || !initialized) return
      try {
        cursorLineMarker = terminal.registerMarker?.(0)
        if (!cursorLineMarker) return
        cursorLineDecoration = terminal.registerDecoration?.({
          marker: cursorLineMarker,
          layer: 'bottom',
        })
        if (!cursorLineDecoration) return
        cursorLineDecoration.onRender?.((element) => {
          if (!element || !term.value) return
          const cols = term.value.cols || 80
          const cell = term.value._core?._renderService?.dimensions?.css?.cell
          const width = cell?.width ? Math.ceil(cell.width * cols) : '100%'
          element.style.left = '0'
          element.style.width = typeof width === 'number' ? `${width}px` : width
          element.style.backgroundColor = 'rgba(128, 128, 128, 0.18)'
          element.style.pointerEvents = 'none'
        })
      } catch {
        clearCursorLineHighlight()
      }
    }

    const bindCursorLineHighlight = (terminal) => {
      cursorMoveListener?.dispose?.()
      cursorMoveListener = null
      if (!terminal) return
      cursorMoveListener = terminal.onCursorMove?.(() => {
        if (cursorLineHighlightEnabled) refreshCursorLineHighlight()
      }) || null
      if (cursorLineHighlightEnabled) refreshCursorLineHighlight()
    }

    const writeOsc52ToClipboard = async (text) => {
      if (!text) return
      try {
        await navigator.clipboard.writeText(text)
        return
      } catch {
        // fallback Wails
      }
      try {
        ClipboardSetText(text)
      } catch {
        // ignore
      }
    }

    const bindOsc52Clipboard = (terminal) => {
      osc52Disposable?.dispose?.()
      osc52Disposable = null
      const register = terminal?.parser?.registerOscHandler
      if (typeof register !== 'function') return
      try {
        osc52Disposable = register.call(terminal.parser, 52, (data) => {
          const text = decodeOsc52ClipboardPayload(data)
          if (text != null) {
            writeOsc52ToClipboard(text)
          }
          return true
        })
      } catch {
        osc52Disposable = null
      }
    }

    const loadLogHighlightSetting = async () => {
      try {
        const cfg = await App.GetSystemSettings()
        logHighlightEnabled = cfg?.shellLogHighlight !== false
        logHighlightConfig = mergeLogHighlightConfig(cfg)
        scrollbackLines = clampShellTerminalScrollback(cfg?.shellTerminalScrollback)
        setShellAsciiInputEnabled(cfg?.shellAsciiInput !== false)
        cursorLineHighlightEnabled = !!cfg?.shellCursorLineHighlight
        lineTimestampsEnabled = !!cfg?.shellLineTimestamps
        passwordAssistEnabled = cfg?.shellPasswordAssist !== false
        shellUseWebgl = !!cfg?.themeSettings?.shellUseWebgl
        shellTabHibernate = cfg?.themeSettings?.shellTabHibernate !== false
        if (term.value) {
          term.value.options.scrollback = scrollbackLines
          refreshCursorLineHighlight()
        }
      } catch {
        logHighlightEnabled = true
        setShellAsciiInputEnabled(true)
        cursorLineHighlightEnabled = false
        lineTimestampsEnabled = false
        passwordAssistEnabled = true
        shellUseWebgl = false
        shellTabHibernate = true
      }
    }

    /** 高亮在写入时注入 ANSI；配置变更后清屏并回放缓冲，无需重连 */
    const reapplyLogHighlightToTerminal = () => {
      if (!term.value || !initialized) return
      tuiModeDepth = 0
      lineTsState.atLineStart = true
      term.value.clear()
      resetShellWriterReplay(props.machineName)
      detachWriter()
      if (props.connected || props.connecting) {
        attachWriter(term.value, { replay: true })
      }
    }

    const onSystemSettingsChanged = (payload) => {
      if (!payload || typeof payload !== 'object') return
      let highlightChanged = false
      if (Object.prototype.hasOwnProperty.call(payload, 'shellLogHighlight')) {
        const next = payload.shellLogHighlight !== false
        if (next !== logHighlightEnabled) {
          logHighlightEnabled = next
          highlightChanged = true
        }
      }
      // disabled 可能为 null/[]，不能用真值判断，否则「全部开启」时配色/规则不更新
      if (
        Object.prototype.hasOwnProperty.call(payload, 'shellLogHighlightColors') ||
        Object.prototype.hasOwnProperty.call(payload, 'shellLogHighlightDisabled') ||
        Object.prototype.hasOwnProperty.call(payload, 'shellLogHighlightKeywords')
      ) {
        const nextConfig = mergeLogHighlightConfig(payload)
        if (JSON.stringify(nextConfig) !== JSON.stringify(logHighlightConfig)) {
          logHighlightConfig = nextConfig
          highlightChanged = true
        }
      }
      if (highlightChanged) {
        reapplyLogHighlightToTerminal()
      }
      if (Object.prototype.hasOwnProperty.call(payload, 'shellTerminalScrollback')) {
        scrollbackLines = clampShellTerminalScrollback(payload.shellTerminalScrollback)
        if (term.value) {
          term.value.options.scrollback = scrollbackLines
        }
      }
      if (Object.prototype.hasOwnProperty.call(payload, 'shellAsciiInput')) {
        setShellAsciiInputEnabled(payload.shellAsciiInput !== false)
      }
      if (Object.prototype.hasOwnProperty.call(payload, 'shellCursorLineHighlight')) {
        cursorLineHighlightEnabled = !!payload.shellCursorLineHighlight
        refreshCursorLineHighlight()
      }
      if (Object.prototype.hasOwnProperty.call(payload, 'shellLineTimestamps')) {
        const next = !!payload.shellLineTimestamps
        if (next !== lineTimestampsEnabled) {
          lineTimestampsEnabled = next
          lineTsState.atLineStart = true
        }
      }
      if (Object.prototype.hasOwnProperty.call(payload, 'shellPasswordAssist')) {
        passwordAssistEnabled = payload.shellPasswordAssist !== false
        if (!passwordAssistEnabled) dismissPasswordAssist()
      }
      if (Object.prototype.hasOwnProperty.call(payload, 'shellUseWebgl')) {
        const next = !!payload.shellUseWebgl
        if (next !== shellUseWebgl) {
          shellUseWebgl = next
          if (initialized && props.active && props.viewVisible) {
            destroyTerminal()
            wakeTerminal()
          }
        }
      }
      if (Object.prototype.hasOwnProperty.call(payload, 'shellTabHibernate')) {
        shellTabHibernate = payload.shellTabHibernate !== false
        if (!shellTabHibernate) {
          clearHibernateTimer()
        } else if (!props.active && !props.inSplit) {
          scheduleHibernate()
        }
      }
    }

    const clearFitTimers = () => {
      fitTimers.forEach((id) => clearTimeout(id))
      fitTimers = []
      if (fitDebounceTimer) {
        clearTimeout(fitDebounceTimer)
        fitDebounceTimer = null
      }
      if (fitRaf1) {
        cancelAnimationFrame(fitRaf1)
        fitRaf1 = 0
      }
      if (fitRaf2) {
        cancelAnimationFrame(fitRaf2)
        fitRaf2 = 0
      }
    }

    const canFit = () => {
      if (!term.value || !fitAddon.value) return false
      if (!props.active || !props.viewVisible) return false
      const el = containerRef.value || terminalRef.value
      if (!el) return false
      // display:none / 折叠后宽高为 0，此时 fit 会得到极小 cols，远端提示符会折行残留主机名碎片
      if (el.clientWidth < 80 || el.clientHeight < 60) return false
      return true
    }

    const runFit = ({ syncPty = true } = {}) => {
      if (!canFit()) return
      try {
        fitAddon.value.fit()
        const { cols, rows } = term.value
        // 拒绝异常尺寸，避免把远端 PTY 缩成几列
        if (!cols || !rows || cols < 20 || rows < 5) return
        // 行列未变：只压制后续 RO，避免滚动条显隐导致 fit 振荡抖动
        if (cols === lastFitCols && rows === lastFitRows) {
          fitQuietUntil = Date.now() + 160
          suppressRoUntil = Date.now() + 160
          return
        }
        lastFitCols = cols
        lastFitRows = rows
        fitQuietUntil = Date.now() + 220
        suppressRoUntil = Date.now() + 220
        // 必须与本地 fit 同步通知远端，否则行列短暂不一致会导致光标跑飞、提示符碎片
        if (!syncPty || !props.connected) return
        App.ResizeShell(props.machineName, cols, rows).catch(() => {})
      } catch {
        // ignore
      }
    }

    /** 合并短时间多次布局变化，避免连打 fit 造成页面闪烁 */
    const scheduleFit = (opts = {}) => {
      if (fitDebounceTimer) clearTimeout(fitDebounceTimer)
      const delay = opts.immediate ? 0 : 64
      fitDebounceTimer = setTimeout(() => {
        fitDebounceTimer = null
        if (!opts.force && Date.now() < fitQuietUntil) return
        if (fitRaf1) cancelAnimationFrame(fitRaf1)
        fitRaf1 = requestAnimationFrame(() => {
          fitRaf1 = 0
          fitRaf2 = requestAnimationFrame(() => {
            fitRaf2 = 0
            runFit(opts)
          })
        })
      }, delay)
    }

    const fitAndResize = (opts = {}) => {
      fitQuietUntil = 0
      scheduleFit({
        immediate: true,
        force: true,
        syncPty: opts.syncPty !== false,
      })
    }

    const applyTerminalAppearance = () => {
      if (!term.value) return
      term.value.options.fontSize = shellFontSize.value || 13
      term.value.options.lineHeight = shellLineHeight.value || 1.2
      term.value.options.fontFamily = getTerminalFont(shellFontFamily.value).value
      term.value.options.theme = terminalThemeForPreset(effectiveTerminalPreset.value)
      scheduleFit()
    }

    const disposeWebglAddon = () => {
      try {
        webglAddon?.dispose?.()
      } catch {
        // ignore
      }
      webglAddon = null
    }

    /** 尝试启用 WebGL；失败则保持默认 canvas 渲染 */
    const tryEnableWebgl = (terminal) => {
      disposeWebglAddon()
      if (!shellUseWebgl || !terminal) return
      try {
        const addon = new WebglAddon()
        addon.onContextLoss?.(() => {
          try {
            addon.dispose()
          } catch {
            // ignore
          }
          if (webglAddon === addon) webglAddon = null
        })
        terminal.loadAddon(addon)
        webglAddon = addon
      } catch (err) {
        console.warn('[shell] WebGL 渲染不可用，已回退 canvas:', err)
        disposeWebglAddon()
      }
    }

    const teardownObservers = () => {
      if (resizeObserver) {
        resizeObserver.disconnect()
        resizeObserver = null
      }
      window.removeEventListener('resize', scheduleFit)
    }

    const clearHibernateTimer = () => {
      if (hibernateTimer) {
        clearTimeout(hibernateTimer)
        hibernateTimer = null
      }
    }

    const shouldHibernate = () =>
      shellTabHibernate && !props.active && !props.inSplit

    /** 轻量休眠：停 fit / 卸 writer；可选销毁 xterm，缓冲保留供 wake 回放 */
    const enterHibernateLight = () => {
      clearFitTimers()
      clearCwdSyncTimer()
      teardownObservers()
      detachWriter()
      notifyShellTerminalBlur()
    }

    const scheduleHibernate = () => {
      clearHibernateTimer()
      if (!shouldHibernate()) return
      const disposeNow = () => {
        if (!shouldHibernate()) return
        if (initialized) destroyTerminal()
        else enterHibernateLight()
      }
      if (HIBERNATE_DISPOSE_DELAY_MS <= 0) {
        disposeNow()
        return
      }
      // 延迟销毁前先停 fit / writer，降低后台开销
      enterHibernateLight()
      hibernateTimer = setTimeout(disposeNow, HIBERNATE_DISPOSE_DELAY_MS)
    }

    const initTerminal = async () => {
      if (initialized || !props.active || !props.viewVisible) return
      clearHibernateTimer()
      for (let i = 0; i < 4; i++) {
        await nextTick()
        if (initialized || !props.active || !props.viewVisible) return
        if (terminalRef.value) break
      }
      if (initialized || !terminalRef.value || !props.active || !props.viewVisible) return
      const terminal = new Terminal({
        cursorBlink: true,
        scrollback: scrollbackLines,
        fontSize: shellFontSize.value || 13,
        lineHeight: shellLineHeight.value || 1.2,
        fontFamily: getTerminalFont(shellFontFamily.value).value,
        theme: terminalThemeForPreset(effectiveTerminalPreset.value),
        // 视口内搜索高亮依赖 proposed decoration API
        allowProposedApi: true,
        overviewRulerWidth: 10,
      })
      const fit = new FitAddon()
      terminal.loadAddon(fit)
      terminal.open(terminalRef.value)
      tryEnableWebgl(terminal)

      // 拦截原生复制：Ctrl/Cmd+C、系统菜单复制也会走 xterm 默认选区（含 less 视觉换行）
      const host = terminal.element || terminalRef.value
      const onNativeCopy = (e) => {
        const text = getTerminalSelectionText(terminal)
        if (!text) return
        e.preventDefault()
        e.stopImmediatePropagation()
        if (e.clipboardData) {
          e.clipboardData.setData('text/plain', text)
        } else {
          navigator.clipboard.writeText(text).catch(() => {})
        }
      }
      host?.addEventListener?.('copy', onNativeCopy, true)
      terminal._flashdockCopyHandler = onNativeCopy
      terminal._flashdockCopyHost = host

      bindAsciiInputListeners(terminal)
      bindInputHandler(terminal)
      bindOsc52Clipboard(terminal)
      bindCursorLineHighlight(terminal)

      scrollListener?.dispose?.()
      scrollListener = terminal.onScroll?.(() => {
        if (activeSearchQuery) scheduleViewportHighlights()
        if (cursorLineHighlightEnabled) refreshCursorLineHighlight()
      }) || null

      term.value = terminal
      fitAddon.value = fit
      initialized = true
      if (props.connected || props.connecting) attachWriter(terminal, { replay: true })
      setupObservers()
      scheduleFit()
      terminal.focus()
    }

    const destroyTerminal = () => {
      clearHibernateTimer()
      clearFitTimers()
      clearCwdSyncTimer()
      cancelAsyncMatchCount()
      clearViewportHighlights()
      clearCursorLineHighlight()
      activeSearchQuery = ''
      lastActiveMatch = null
      lastCountedQuery = ''
      resetSearchMatchIndex()
      scrollListener?.dispose?.()
      scrollListener = null
      cursorMoveListener?.dispose?.()
      cursorMoveListener = null
      osc52Disposable?.dispose?.()
      osc52Disposable = null
      detachWriter()
      resetShellWriterReplay(props.machineName)
      tuiModeDepth = 0
      lineTsState.atLineStart = true
      teardownObservers()
      inputListener?.dispose?.()
      inputListener = null
      disposeWebglAddon()
      if (term.value) {
        unbindAsciiInputListeners(term.value)
        notifyShellTerminalBlur()
        const host = term.value._flashdockCopyHost
        const handler = term.value._flashdockCopyHandler
        if (host && handler) host.removeEventListener('copy', handler, true)
        term.value.dispose()
        term.value = null
        fitAddon.value = null
      }
      initialized = false
      lastFitCols = 0
      lastFitRows = 0
      fitQuietUntil = 0
      suppressRoUntil = 0
      lastSearchResults = { resultIndex: -1, resultCount: 0, capped: false }
    }

    const clear = () => term.value?.clear()

    const toSearchResult = (found) => ({
      found: !!found,
      resultIndex: lastSearchResults.resultIndex,
      resultCount: lastSearchResults.resultCount,
      capped: lastSearchResults.capped,
    })

    const findNext = () => {
      const query = props.searchQuery.trim()
      const terminal = term.value
      if (!terminal || !query) {
        cancelAsyncMatchCount()
        clearViewportHighlights()
        activeSearchQuery = ''
        lastActiveMatch = null
        lastCountedQuery = ''
        resetSearchMatchIndex()
        emitSearchResult({ found: false, resultIndex: -1, resultCount: 0 })
        return toSearchResult(false)
      }
      try {
        const sameQuery = query === lastCountedQuery
        const match = findInTerminalBuffer(terminal, query, { reverse: false })
        if (match) selectTerminalMatch(terminal, match)
        else terminal.clearSelection()
        activeSearchQuery = query
        lastActiveMatch = match
        refreshViewportHighlights(query, match)
        emitSearchResult({
          found: !!match,
          resultIndex: resolveSearchIndex(match),
          resultCount: sameQuery ? lastSearchResults.resultCount : 0,
          capped: sameQuery ? lastSearchResults.capped : false,
        })
        if (!sameQuery) scheduleAsyncMatchCount(query)
        return toSearchResult(!!match)
      } catch (err) {
        console.warn('[shell-search] findNext failed:', err)
        return toSearchResult(false)
      }
    }

    const findPrevious = () => {
      const query = props.searchQuery.trim()
      const terminal = term.value
      if (!terminal || !query) {
        cancelAsyncMatchCount()
        clearViewportHighlights()
        activeSearchQuery = ''
        lastActiveMatch = null
        lastCountedQuery = ''
        resetSearchMatchIndex()
        emitSearchResult({ found: false, resultIndex: -1, resultCount: 0 })
        return toSearchResult(false)
      }
      try {
        const sameQuery = query === lastCountedQuery
        const match = findInTerminalBuffer(terminal, query, { reverse: true })
        if (match) selectTerminalMatch(terminal, match)
        else terminal.clearSelection()
        activeSearchQuery = query
        lastActiveMatch = match
        refreshViewportHighlights(query, match)
        emitSearchResult({
          found: !!match,
          resultIndex: resolveSearchIndex(match),
          resultCount: sameQuery ? lastSearchResults.resultCount : 0,
          capped: sameQuery ? lastSearchResults.capped : false,
        })
        if (!sameQuery) scheduleAsyncMatchCount(query)
        return toSearchResult(!!match)
      } catch (err) {
        console.warn('[shell-search] findPrevious failed:', err)
        return toSearchResult(false)
      }
    }

    const clearSearch = () => {
      cancelAsyncMatchCount()
      clearViewportHighlights()
      activeSearchQuery = ''
      lastActiveMatch = null
      lastCountedQuery = ''
      resetSearchMatchIndex()
      term.value?.clearSelection?.()
      emitSearchResult({ found: false, resultIndex: -1, resultCount: 0 })
    }

    const stripAnsi = (s) => String(s || '')
      .replace(/\x1b\[[0-9?]*[ -/]*[@-~]/g, '')
      .replace(/\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)/g, '')

    const readTerminalLine = (terminal, rowOffset = 0) => {
      const buf = terminal.buffer.active
      const row = buf.baseY + buf.cursorY + rowOffset
      if (row < 0) return ''
      const line = buf.getLine(row)
      if (!line) return ''
      let text = ''
      for (let i = 0; i < line.length; i++) {
        text += line.getCell(i)?.getChars() || ''
      }
      return stripAnsi(text)
    }

    /** 从终端行提取 cd 命令（支持历史命令、Tab 补全后的完整行） */
    const extractCdCommand = (raw) => {
      const plain = stripAnsi(raw).trim()
      if (!plain) return ''
      const tail = plain.includes('\n')
        ? plain.split('\n').pop()?.trim() || plain
        : plain.replace(/^.*[$#>]\s*/, '').trim() || plain
      if (!/^cd(\s|$)/i.test(tail)) return ''
      const m = tail.match(/^cd(?:\s+([^;|&]+))?/i)
      if (!m) return ''
      return m[1] !== undefined ? `cd ${m[1].trim()}` : 'cd'
    }

    const mightBeCdEnter = () => {
      if (/^cd(\s|$)/i.test(inputLine.trim())) return true
      if (!term.value) return false
      return /^cd(\s|$)/i.test(extractCdCommand(readTerminalLine(term.value)))
    }

    const clearCwdSyncTimer = () => {
      if (cwdSyncTimer) {
        clearTimeout(cwdSyncTimer)
        cwdSyncTimer = null
      }
    }

    const onEnterForCwd = () => {
      const shouldSync = mightBeCdEnter()
      inputLine = ''
      if (shouldSync) {
        scheduleCwdSyncAfterEnter()
      }
    }

    const trackInputLine = (data) => {
      for (let i = 0; i < data.length; i++) {
        const ch = data[i]
        if (ch === '\r' || ch === '\n') {
          onEnterForCwd()
        } else if (ch === '\x7f' || ch === '\b') {
          inputLine = inputLine.slice(0, -1)
        } else if (ch === '\x03') {
          inputLine = ''
        } else if (ch === '\x1b') {
          // 方向键 / Tab 补全序列：inputLine 不可靠
          inputLine = ''
        } else if (ch >= ' ' || ch === '\t') {
          inputLine += ch
        }
      }
    }

    /**
     * 回车后延迟读取终端上一行（已执行的 cd 命令，含 Tab 补全结果）。
     * 避免 inputLine 停在 cd ap 而实际执行的是 cd app/。
     */
    const scheduleCwdSyncAfterEnter = () => {
      clearCwdSyncTimer()
      cwdSyncTimer = setTimeout(async () => {
        if (!term.value) return
        let cdLine = extractCdCommand(readTerminalLine(term.value, -1))
        if (!/^cd(\s|$)/i.test(cdLine)) {
          cdLine = extractCdCommand(readTerminalLine(term.value, 0))
        }
        if (!/^cd(\s|$)/i.test(cdLine)) return
        try {
          const cwd = await App.SyncShellCwd(props.machineName, cdLine)
          if (cwd) {
            emit('cwd-sync', { machineName: props.machineName, cwd })
          }
        } catch (e) {
          console.warn('cwd sync failed:', e)
        }
      }, 200)
    }

    const scheduleCwdSync = () => {
      scheduleCwdSyncAfterEnter()
    }

    const beginSuppressInputForward = () => {
      suppressInputForwardDepth += 1
    }

    const endSuppressInputForward = () => {
      suppressInputForwardDepth = Math.max(0, suppressInputForwardDepth - 1)
    }

    const bindInputHandler = (terminal) => {
      inputListener?.dispose?.()
      inputListener = terminal.onData((data) => {
        // 回放历史输出时 xterm 会应答 DA/OSC 查询；不能当作用户输入发回 PTY
        if (suppressInputForwardDepth > 0) return
        if (!props.active || !props.viewVisible) return
        if (!props.connected) {
          if (props.connecting) return
          // 断开后：Enter 触发重连，其余输入忽略
          if (data === '\r' || data === '\n') {
            emit('reconnect', props.machineName)
          }
          return
        }
        trackInputLine(data)
        App.SendShellInput(props.machineName, data).catch(() => {})
      })
    }

    const attachWriter = (terminal, { replay = false } = {}) => {
      unregisterWriter?.()
      let holdingReplaySuppress = false
      const releaseReplaySuppress = () => {
        if (!holdingReplaySuppress) return
        holdingReplaySuppress = false
        endSuppressInputForward()
      }
      const unreg = registerShellWriter(
        props.machineName,
        {
          writeData: (data) => writeDataToTerminal(terminal, data),
          writeLine: (line) => writeTerminal(terminal, `\x1b[33m${line}\x1b[0m\r\n`),
          clear: () => terminal.clear(),
        },
        {
          replay,
          aroundReplay: (phase) => {
            if (phase === 'start') {
              holdingReplaySuppress = true
              beginSuppressInputForward()
              lineTsState.atLineStart = true
              return
            }
            releaseReplaySuppress()
            // 回放结束后直接落底，避免逐块写入时的滚动过程
            try {
              terminal.scrollToBottom?.()
            } catch {
              // ignore
            }
            if (cursorLineHighlightEnabled) refreshCursorLineHighlight()
          },
        },
      )
      unregisterWriter = () => {
        // 中途卸载时也要释放，避免 suppress 计数泄漏导致键盘永久失效
        releaseReplaySuppress()
        unreg()
      }
    }

    const detachWriter = () => {
      unregisterWriter?.()
      unregisterWriter = null
    }

    const setupObservers = () => {
      if (resizeObserver) return
      const target = containerRef.value || terminalRef.value
      if (target && window.ResizeObserver) {
        resizeObserver = new ResizeObserver(() => {
          if (Date.now() < fitQuietUntil || Date.now() < suppressRoUntil) return
          scheduleFit()
        })
        resizeObserver.observe(target)
      }
      window.addEventListener('resize', scheduleFit)
    }

    const ensureWriter = () => {
      if (!term.value || unregisterWriter) return
      attachWriter(term.value, { replay: true })
    }

    const wakeTerminal = async () => {
      clearHibernateTimer()
      if (!props.active || !props.viewVisible) return
      if (!initialized) await initTerminal()
      if (!term.value) {
        await nextTick()
        await initTerminal()
      }
      if (props.connected || props.connecting) ensureWriter()
      setupObservers()
      scheduleFit()
      await nextTick()
      term.value?.focus()
    }

    watch(() => [props.connected, props.connecting], async ([val, connecting]) => {
      if (!term.value) {
        if ((val || connecting) && props.active) await initTerminal()
        return
      }
      if (val || connecting) {
        ensureWriter()
        scheduleFit()
        if (props.active && val) {
          scheduleCwdSync()
          await nextTick()
          term.value.focus()
        }
      } else {
        detachWriter()
        clearCwdSyncTimer()
      }
    })

    watch(() => props.active, async (val) => {
      setShellOutputSessionActive(props.machineName, val)
      if (val && props.viewVisible) {
        await wakeTerminal()
        return
      }
      clearFitTimers()
      clearCwdSyncTimer()
      notifyShellTerminalBlur()
      if (shellTabHibernate && !props.inSplit) {
        scheduleHibernate()
      }
    })

    watch(() => props.inSplit, (inSplit) => {
      if (inSplit) {
        clearHibernateTimer()
        return
      }
      if (!props.active && shellTabHibernate) {
        scheduleHibernate()
      }
    })

    watch(() => props.viewVisible, async (visible) => {
      if (!visible) {
        // 仅隐藏：保留 xterm，避免回首页/任务模式时销毁→回放把协议应答灌进输入
        // （省内存模式会卸载整个工作区；此处不做 hibernate dispose）
        clearHibernateTimer()
        clearFitTimers()
        teardownObservers()
        notifyShellTerminalBlur()
        return
      }
      if (!props.active) {
        if (shellTabHibernate && !props.inSplit) scheduleHibernate()
        return
      }
      await wakeTerminal()
    })

    watch(() => props.searchQuery, (query) => {
      if (!props.active || !query.trim()) {
        clearSearch()
      }
      // 查找与匹配计数由工作区触发（findNext/findPrevious），避免重复前进
    })

    watch([shellFontSize, shellLineHeight, terminalPreset, shellFontFamily, effectiveTerminalPreset], () => {
      applyTerminalAppearance()
    })

    watch(() => props.connected, (ok) => {
      if (!ok) dismissPasswordAssist()
    })

    watch(() => props.terminalPresetOverride, () => {
      applyTerminalAppearance()
    })

    const onThemeChanged = (settings) => {
      if (settings?.shellFontSize > 0) shellFontSize.value = settings.shellFontSize
      if (settings?.shellLineHeight > 0) shellLineHeight.value = settings.shellLineHeight
      if (settings?.terminalPreset) terminalPreset.value = settings.terminalPreset
      if (settings?.shellFontFamily) shellFontFamily.value = settings.shellFontFamily
      applyTerminalAppearance()
    }

    let offThemeChanged = null
    let offSystemSettingsChanged = null

    onMounted(() => {
      ensureShellAsciiInputListeners()
      setShellOutputSessionActive(props.machineName, props.active)
      loadLogHighlightSetting().finally(() => {
        if (props.active && props.viewVisible) wakeTerminal()
      })
      offThemeChanged = EventsOn('theme:changed', onThemeChanged)
      offSystemSettingsChanged = EventsOn('system-settings:changed', onSystemSettingsChanged)
      window.addEventListener('click', hideContextMenu)
      window.addEventListener('blur', hideContextMenu)
    })

    onUnmounted(() => {
      setShellOutputSessionActive(props.machineName, false)
      notifyShellTerminalBlur()
      offThemeChanged?.()
      offSystemSettingsChanged?.()
      window.removeEventListener('click', hideContextMenu)
      window.removeEventListener('blur', hideContextMenu)
      window.removeEventListener('resize', scheduleFit)
      destroyTerminal()
    })

    expose({ clear, fitAndResize, findNext, findPrevious, clearSearch, getSelection, wakeTerminal, pasteClipboard })

    return {
      containerRef,
      terminalRef,
      ctxMenuRef,
      ctx,
      displayName,
      overlayStatus,
      passwordAssistVisible,
      passwordAssistValue,
      passwordAssistInputRef,
      sendPasswordAssist,
      dismissPasswordAssist,
      hideContextMenu,
      onContextMenu,
      onCopy,
      onPaste,
      onFind,
      onClearCache,
      onRemoveFromSplit,
      onExitSplit,
      onPaneFocus,
    }
  },
}
</script>

<style scoped>
.shell-terminal {
  flex: 1;
  min-height: 0;
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #0d1117;
  overflow: hidden;
  position: relative;
}

.ctx-menu {
  position: fixed;
  z-index: 5000;
  margin: 0;
  padding: 4px 0;
  min-width: 120px;
  list-style: none;
  background: var(--app-panel-bg, #1e1e1e);
  border: 1px solid var(--app-border, #333);
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.25);
}

.ctx-menu li {
  padding: 6px 14px;
  font-size: 13px;
  color: var(--app-text, #e6e6e6);
  cursor: pointer;
}

.ctx-menu li:hover {
  background: var(--app-accent-bg, rgba(64, 158, 255, 0.15));
  color: var(--app-accent-color, #409eff);
}

.ctx-menu li.danger {
  color: #f56c6c;
}

.ctx-menu li.sep {
  height: 1px;
  margin: 4px 8px;
  padding: 0;
  background: var(--app-border, #333);
  pointer-events: none;
  cursor: default;
}

.ctx-menu li.sep:hover {
  background: var(--app-border, #333);
  color: inherit;
}

.terminal-host {
  flex: 1;
  min-height: 0;
  width: 100%;
  padding: 4px 8px;
  box-sizing: border-box;
  overflow: hidden;
}

.terminal-bottom-gap {
  flex-shrink: 0;
  height: 16px;
  background: inherit;
}

.terminal-host :deep(.xterm),
.terminal-host :deep(.xterm-viewport),
.terminal-host :deep(.xterm-screen) {
  width: 100% !important;
  height: 100% !important;
}

.terminal-host :deep(.xterm-viewport) {
  overflow-y: auto;
  /* 预留滚动条槽，避免显隐导致可用宽度来回变、触发 fit 振荡 */
  scrollbar-gutter: stable;
}

.shell-password-assist {
  position: absolute;
  left: 12px;
  right: 12px;
  bottom: 12px;
  z-index: 25;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 10px;
  background: color-mix(in srgb, var(--app-card-bg, #1e1e1e) 92%, transparent);
  border: 1px solid var(--app-border, #333);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.28);
}

.shell-password-assist-label {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--app-text-muted, #909399);
}

.shell-password-assist-input {
  flex: 1;
  min-width: 0;
  height: 28px;
  padding: 0 10px;
  border-radius: 6px;
  border: 1px solid var(--app-border, #444);
  background: var(--app-panel-bg, #121212);
  color: var(--app-text, #e6e6e6);
  font-size: 13px;
  outline: none;
}

.shell-password-assist-input:focus {
  border-color: var(--app-accent-color, #409eff);
}

.shell-password-assist-send {
  flex-shrink: 0;
  height: 28px;
  padding: 0 12px;
  border: none;
  border-radius: 6px;
  background: var(--app-accent-color, #409eff);
  color: #fff;
  font-size: 12px;
  cursor: pointer;
}

.shell-password-assist-dismiss {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--app-text-muted, #909399);
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
}
</style>
