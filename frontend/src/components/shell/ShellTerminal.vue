<template>
  <div class="shell-terminal" ref="containerRef" @contextmenu.prevent="onContextMenu">
    <div ref="terminalRef" class="terminal-host"></div>
    <ul
      v-if="ctx.visible"
      class="ctx-menu"
      :style="{ left: ctx.x + 'px', top: ctx.y + 'px' }"
      @click.stop
    >
      <li @click="onCopy">复制</li>
      <li @click="onPaste">粘贴</li>
      <li @click="onFind">查找</li>
      <li class="danger" @click="onClearCache">清空缓存</li>
    </ul>
  </div>
</template>

<script>
import { ref, reactive, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { SearchAddon } from 'xterm-addon-search'
import 'xterm/css/xterm.css'
import * as App from '../../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'
import { registerShellWriter } from '../../utils/shellOutputBuffer'
import { useTheme } from '../../composables/useTheme'
import { terminalThemeForPreset, getTerminalFont } from '../../utils/themePresets'
import {
  highlightShellChunk,
  isProbablyBinary,
  mergeLogHighlightConfig,
  mergeLogHighlightColors,
  mergeLogHighlightRules,
  rulesToDisabled,
  DEFAULT_SHELL_LOG_COLORS,
} from '../../utils/shellLogHighlight'

export default {
  name: 'ShellTerminal',
  props: {
    machineName: { type: String, required: true },
    active: { type: Boolean, default: false },
    /** Shell 工作区是否可见（回首页时为 false，避免 display:none 下 fit 成极窄列宽） */
    viewVisible: { type: Boolean, default: true },
    connected: { type: Boolean, default: false },
    searchQuery: { type: String, default: '' },
  },
  emits: ['open-search', 'clear-cache', 'reconnect', 'search-result', 'cwd-sync'],
  setup(props, { expose, emit }) {
    const containerRef = ref(null)
    const terminalRef = ref(null)
    const term = ref(null)
    const fitAddon = ref(null)
    const searchAddon = ref(null)
    const ctx = reactive({ visible: false, x: 0, y: 0, selection: '' })
    const { shellFontSize, shellLineHeight, terminalPreset, shellFontFamily } = useTheme()
    let resizeObserver = null
    let fitTimers = []
    let initialized = false
    let unregisterWriter = null
    let inputListener = null
    let searchResultsListener = null
    let lastSearchResults = { resultIndex: -1, resultCount: 0 }
    let cwdSyncTimer = null
    let inputLine = ''
    let logHighlightEnabled = true
    let logHighlightConfig = mergeLogHighlightConfig(null)
    const textDecoder = new TextDecoder('utf-8')

    const SEARCH_DECORATIONS = {
      // 普通匹配：淡琥珀色；当前定位：蓝色，和选区颜色一致便于辨认
      matchBackground: '#3d3728',
      matchBorder: '#a9945a',
      matchOverviewRuler: '#a9945a',
      activeMatchBackground: '#1f6feb',
      activeMatchBorder: '#79c0ff',
      activeMatchColorOverviewRuler: '#79c0ff',
    }

    const hideContextMenu = () => {
      ctx.visible = false
    }

    const onContextMenu = (e) => {
      // 右键时先保存选区：点击菜单项可能触发 mousedown 清掉 xterm 选区
      ctx.selection = term.value?.getSelection?.() || ''
      ctx.x = e.clientX
      ctx.y = e.clientY
      ctx.visible = true
    }

    const onCopy = async () => {
      hideContextMenu()
      const text = term.value?.getSelection?.() || ''
      if (!text) {
        ElMessage.info('没有选中内容')
        return
      }
      try {
        await navigator.clipboard.writeText(text)
        ElMessage.success('已复制')
      } catch (err) {
        ElMessage.error(`复制失败: ${err}`)
      }
    }

    const onPaste = async () => {
      hideContextMenu()
      if (!props.connected) {
        ElMessage.info('当前未连接，请先连接')
        return
      }
      try {
        const text = await navigator.clipboard.readText()
        if (!text) return
        await App.SendShellInput(props.machineName, text)
        term.value?.focus?.()
      } catch (err) {
        ElMessage.error(`粘贴失败: ${err}`)
      }
    }

    const getSelection = () => term.value?.getSelection?.() || ''

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

    const decodeBase64 = (b64) => {
      const binary = atob(b64)
      const bytes = new Uint8Array(binary.length)
      for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i)
      }
      return bytes
    }

    const writeHighlighted = (terminal, bytes) => {
      if (!logHighlightEnabled || isProbablyBinary(bytes)) {
        terminal.write(bytes)
        return
      }
      try {
        const text = textDecoder.decode(bytes, { stream: true })
        terminal.write(highlightShellChunk(text, logHighlightConfig))
      } catch {
        terminal.write(bytes)
      }
    }

    const loadLogHighlightSetting = async () => {
      try {
        const cfg = await App.GetSystemSettings()
        logHighlightEnabled = cfg?.shellLogHighlight !== false
        logHighlightConfig = mergeLogHighlightConfig(cfg)
      } catch {
        logHighlightEnabled = true
      }
    }

    const onSystemSettingsChanged = (payload) => {
      if (payload && Object.prototype.hasOwnProperty.call(payload, 'shellLogHighlight')) {
        logHighlightEnabled = payload.shellLogHighlight !== false
      }
      if (payload?.shellLogHighlightColors || payload?.shellLogHighlightDisabled) {
        logHighlightConfig = mergeLogHighlightConfig(payload)
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
      if (initialized || !terminalRef.value || !props.active || !props.viewVisible) return
      await nextTick()
      const terminal = new Terminal({
        cursorBlink: true,
        fontSize: shellFontSize.value || 13,
        lineHeight: shellLineHeight.value || 1.2,
        fontFamily: getTerminalFont(shellFontFamily.value).value,
        theme: terminalThemeForPreset(terminalPreset.value),
        // SearchAddon 高亮依赖 experimental decoration API
        allowProposedApi: true,
        overviewRulerWidth: 14,
      })
      const fit = new FitAddon()
      const search = new SearchAddon()
      terminal.loadAddon(fit)
      terminal.loadAddon(search)
      terminal.open(terminalRef.value)

      bindInputHandler(terminal)

      searchResultsListener?.dispose?.()
      searchResultsListener = search.onDidChangeResults?.((e) => {
        lastSearchResults = {
          resultIndex: e?.resultIndex ?? -1,
          resultCount: e?.resultCount ?? 0,
        }
        if (props.active) {
          emit('search-result', {
            found: (e?.resultCount ?? 0) > 0 && (e?.resultIndex ?? -1) >= 0,
            ...lastSearchResults,
          })
        }
      }) || null

      term.value = terminal
      fitAddon.value = fit
      searchAddon.value = search
      initialized = true
      if (props.connected) attachWriter(terminal)
      setupObservers()
      scheduleFit()
      terminal.focus()
    }

    const destroyTerminal = () => {
      clearFitTimers()
      clearCwdSyncTimer()
      detachWriter()
      searchResultsListener?.dispose?.()
      searchResultsListener = null
      if (resizeObserver) {
        resizeObserver.disconnect()
        resizeObserver = null
      }
      inputListener?.dispose?.()
      inputListener = null
      if (term.value) {
        term.value.dispose()
        term.value = null
        fitAddon.value = null
        searchAddon.value = null
      }
      initialized = false
      lastSearchResults = { resultIndex: -1, resultCount: 0 }
    }

    const clear = () => term.value?.clear()

    const searchOptions = () => ({
      caseSensitive: false,
      regex: false,
      incremental: false,
      decorations: SEARCH_DECORATIONS,
    })

    const toSearchResult = (found) => ({
      found: !!found,
      resultIndex: lastSearchResults.resultIndex,
      resultCount: lastSearchResults.resultCount,
    })

    const findNext = () => {
      const query = props.searchQuery.trim()
      if (!searchAddon.value || !query) {
        lastSearchResults = { resultIndex: -1, resultCount: 0 }
        return toSearchResult(false)
      }
      try {
        const found = searchAddon.value.findNext(query, searchOptions())
        return toSearchResult(found)
      } catch (err) {
        console.warn('[shell-search] findNext failed:', err)
        return toSearchResult(false)
      }
    }

    const findPrevious = () => {
      const query = props.searchQuery.trim()
      if (!searchAddon.value || !query) {
        lastSearchResults = { resultIndex: -1, resultCount: 0 }
        return toSearchResult(false)
      }
      try {
        const found = searchAddon.value.findPrevious(query, searchOptions())
        return toSearchResult(found)
      } catch (err) {
        console.warn('[shell-search] findPrevious failed:', err)
        return toSearchResult(false)
      }
    }

    const clearSearch = () => {
      searchAddon.value?.clearDecorations()
      lastSearchResults = { resultIndex: -1, resultCount: 0 }
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

    const bindInputHandler = (terminal) => {
      inputListener?.dispose?.()
      inputListener = terminal.onData((data) => {
        if (!props.active || !props.viewVisible) return
        if (!props.connected) {
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

    const attachWriter = (terminal) => {
      unregisterWriter?.()
      unregisterWriter = registerShellWriter(props.machineName, {
        writeData: (b64) => writeHighlighted(terminal, decodeBase64(b64)),
        writeLine: (line) => terminal.writeln(`\x1b[33m${line}\x1b[0m`),
        clear: () => terminal.clear(),
      })
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

    watch(() => props.connected, async (val) => {
      if (!term.value) {
        if (val && props.active) await initTerminal()
        return
      }
      if (val) {
        if (props.active) attachWriter(term.value)
        scheduleFit()
        if (props.active) {
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
      if (val && props.viewVisible) {
        if (!initialized) await initTerminal()
        else {
          setupObservers()
          scheduleFit()
          if (term.value && props.connected) attachWriter(term.value)
        }
        await nextTick()
        term.value?.focus()
      } else if (term.value) {
        detachWriter()
      }
    })

    watch(() => props.viewVisible, async (visible) => {
      if (!visible) {
        clearFitTimers()
        detachWriter()
        return
      }
      if (!props.active) return
      await nextTick()
      if (!initialized) await initTerminal()
      else {
        scheduleFit()
        if (term.value && props.connected) attachWriter(term.value)
      }
      await nextTick()
      term.value?.focus()
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

    onMounted(() => {
      loadLogHighlightSetting()
      if (props.active && props.viewVisible) initTerminal()
      EventsOn('theme:changed', onThemeChanged)
      EventsOn('system-settings:changed', onSystemSettingsChanged)
      window.addEventListener('click', hideContextMenu)
      window.addEventListener('blur', hideContextMenu)
    })

    onUnmounted(() => {
      EventsOff('theme:changed')
      EventsOff('system-settings:changed')
      window.removeEventListener('click', hideContextMenu)
      window.removeEventListener('blur', hideContextMenu)
      window.removeEventListener('resize', scheduleFit)
      destroyTerminal()
    })

    expose({ clear, fitAndResize, findNext, findPrevious, clearSearch, getSelection })

    return {
      containerRef,
      terminalRef,
      ctx,
      onContextMenu,
      onCopy,
      onPaste,
      onFind,
      onClearCache,
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
  z-index: 3000;
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
