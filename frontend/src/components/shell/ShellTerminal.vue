<template>
  <div
    class="shell-terminal"
    ref="containerRef"
    @contextmenu.prevent="onContextMenu"
    @mousedown="onPaneFocus"
  >
    <div ref="terminalRef" class="terminal-host"></div>
    <ShellConnectionOverlay
      :status="overlayStatus"
      :machine-name="displayName"
      :host="host"
      :user="user"
      :jump-chain="jumpChain"
      :proxy-jump="proxyJump"
      @reconnect="$emit('reconnect', machineName)"
    />
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
    broadcastEnabled: { type: Boolean, default: false },
    broadcastTargets: { type: Array, default: () => [] },
    /** 当前是否处于分屏窗格中 */
    inSplit: { type: Boolean, default: false },
    /** 是否曾成功连接过（用于区分「尚未连上」与「已断开」） */
    everConnected: { type: Boolean, default: false },
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
    const overlayStatus = computed(() => {
      if (props.connecting) return 'connecting'
      if (!props.connected && props.everConnected) return 'disconnected'
      return ''
    })
    let resizeObserver = null
    let fitTimers = []
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
      if (!logHighlightEnabled) {
        return writeTerminal(terminal, raw)
      }
      try {
        // 不因 less/TUI 整块跳过：按行高亮，含交互序列的行仍原样透传
        return writeTerminal(terminal, highlightShellChunk(text, logHighlightConfig))
      } catch {
        return writeTerminal(terminal, raw)
      }
    }

    const loadLogHighlightSetting = async () => {
      try {
        const cfg = await App.GetSystemSettings()
        logHighlightEnabled = cfg?.shellLogHighlight !== false
        logHighlightConfig = mergeLogHighlightConfig(cfg)
        scrollbackLines = clampShellTerminalScrollback(cfg?.shellTerminalScrollback)
        setShellAsciiInputEnabled(cfg?.shellAsciiInput !== false)
        if (term.value) {
          term.value.options.scrollback = scrollbackLines
        }
      } catch {
        logHighlightEnabled = true
        setShellAsciiInputEnabled(true)
      }
    }

    /** 高亮在写入时注入 ANSI；配置变更后清屏并回放缓冲，无需重连 */
    const reapplyLogHighlightToTerminal = () => {
      if (!term.value || !initialized) return
      tuiModeDepth = 0
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
    }

    const clearFitTimers = () => {
      fitTimers.forEach((id) => clearTimeout(id))
      fitTimers = []
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

    const scheduleFit = () => {
      clearFitTimers()
      const run = () => {
        if (!canFit()) return
        try {
          fitAddon.value.fit()
          if (!props.connected) return
          const { cols, rows } = term.value
          // 拒绝异常尺寸，避免把远端 PTY 缩成几列
          if (!cols || !rows || cols < 20 || rows < 5) return
          App.ResizeShell(props.machineName, cols, rows).catch(() => {})
        } catch {
          // ignore
        }
      }
      requestAnimationFrame(() => {
        requestAnimationFrame(run)
      })
      ;[0, 80, 200, 500].forEach((delay) => {
        fitTimers.push(setTimeout(run, delay))
      })
    }

    const fitAndResize = () => {
      scheduleFit()
    }

    const applyTerminalAppearance = () => {
      if (!term.value) return
      term.value.options.fontSize = shellFontSize.value || 13
      term.value.options.lineHeight = shellLineHeight.value || 1.2
      term.value.options.fontFamily = getTerminalFont(shellFontFamily.value).value
      term.value.options.theme = terminalThemeForPreset(terminalPreset.value)
      scheduleFit()
    }

    const initTerminal = async () => {
      if (initialized || !props.active || !props.viewVisible) return
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
        theme: terminalThemeForPreset(terminalPreset.value),
        // 视口内搜索高亮依赖 proposed decoration API
        allowProposedApi: true,
        overviewRulerWidth: 10,
      })
      const fit = new FitAddon()
      terminal.loadAddon(fit)
      terminal.open(terminalRef.value)

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

      scrollListener?.dispose?.()
      scrollListener = terminal.onScroll?.(() => {
        if (activeSearchQuery) scheduleViewportHighlights()
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
      clearFitTimers()
      clearCwdSyncTimer()
      cancelAsyncMatchCount()
      clearViewportHighlights()
      activeSearchQuery = ''
      lastActiveMatch = null
      lastCountedQuery = ''
      resetSearchMatchIndex()
      scrollListener?.dispose?.()
      scrollListener = null
      detachWriter()
      resetShellWriterReplay(props.machineName)
      tuiModeDepth = 0
      if (resizeObserver) {
        resizeObserver.disconnect()
        resizeObserver = null
      }
      inputListener?.dispose?.()
      inputListener = null
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
        if (props.broadcastEnabled) return
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
              return
            }
            releaseReplaySuppress()
            // 回放结束后直接落底，避免逐块写入时的滚动过程
            try {
              terminal.scrollToBottom?.()
            } catch {
              // ignore
            }
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
        resizeObserver = new ResizeObserver(() => scheduleFit())
        resizeObserver.observe(target)
      }
      window.addEventListener('resize', scheduleFit)
    }

    const ensureWriter = () => {
      if (!term.value || unregisterWriter) return
      attachWriter(term.value, { replay: true })
    }

    const wakeTerminal = async () => {
      if (!props.active || !props.viewVisible) return
      if (!initialized) await initTerminal()
      if (!term.value) {
        await nextTick()
        await initTerminal()
      }
      if (props.connected || props.connecting) ensureWriter()
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
      // 切换 Tab 时保留 xterm，避免销毁后整缓冲回放造成「从第一行滚到末行」
      clearFitTimers()
      clearCwdSyncTimer()
      notifyShellTerminalBlur()
    })

    watch(() => props.viewVisible, async (visible) => {
      if (!visible) {
        // 仅隐藏：保留 xterm，避免回首页/任务模式时销毁→回放把协议应答灌进输入
        clearFitTimers()
        notifyShellTerminalBlur()
        return
      }
      if (!props.active) return
      await wakeTerminal()
    })

    watch(() => props.searchQuery, (query) => {
      if (!props.active || !query.trim()) {
        clearSearch()
      }
      // 查找与匹配计数由工作区触发（findNext/findPrevious），避免重复前进
    })

    watch([shellFontSize, shellLineHeight, terminalPreset, shellFontFamily], () => {
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
  height: 100%;
  padding: 4px 8px;
  box-sizing: border-box;
}

.terminal-host :deep(.xterm),
.terminal-host :deep(.xterm-viewport),
.terminal-host :deep(.xterm-screen) {
  width: 100% !important;
  height: 100% !important;
}

.terminal-host :deep(.xterm-viewport) {
  overflow-y: auto;
}
</style>
