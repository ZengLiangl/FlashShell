<template>
  <div class="shell-terminal" ref="containerRef">
    <div ref="terminalRef" class="terminal-host"></div>
  </div>
</template>

<script>
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { SearchAddon } from 'xterm-addon-search'
import 'xterm/css/xterm.css'
import * as App from '../../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'
import { registerShellWriter } from '../../utils/shellOutputBuffer'
import { useTheme } from '../../composables/useTheme'

export default {
  name: 'ShellTerminal',
  props: {
    machineName: { type: String, required: true },
    active: { type: Boolean, default: false },
    connected: { type: Boolean, default: false },
    searchQuery: { type: String, default: '' },
  },
  emits: ['cd-hint'],
  setup(props, { expose, emit }) {
    const containerRef = ref(null)
    const terminalRef = ref(null)
    const term = ref(null)
    const fitAddon = ref(null)
    const searchAddon = ref(null)
    const { shellFontSize, shellLineHeight, terminalPreset } = useTheme()
    let resizeObserver = null
    let fitTimers = []
    let initialized = false
    let unregisterWriter = null
    let inputLineBuf = ''

    const trackInputForCd = (data) => {
      for (let i = 0; i < data.length; i++) {
        const ch = data[i]
        // 跳过 CSI/OSC 等终端控制序列，避免污染输入缓冲
        if (ch === '\u001b') {
          i += skipEscSequence(data, i)
          continue
        }
        if (ch === '\r' || ch === '\n') {
          const line = inputLineBuf.trim()
          inputLineBuf = ''
          const target = parseCdTarget(line)
          if (target != null) {
            emit('cd-hint', { machineName: props.machineName, target })
          }
          continue
        }
        if (ch === '\u007f' || ch === '\b') {
          inputLineBuf = inputLineBuf.slice(0, -1)
          continue
        }
        // Ctrl+C / Ctrl+U 清空当前输入缓冲
        if (ch === '\u0003' || ch === '\u0015') {
          inputLineBuf = ''
          continue
        }
        if (ch >= ' ' || ch === '\t') {
          inputLineBuf += ch
        }
      }
    }

    const skipEscSequence = (data, start) => {
      // 返回需要额外跳过的字符数（不含起始 ESC）
      if (start + 1 >= data.length) return 0
      const next = data[start + 1]
      if (next === '[') {
        // CSI: ESC [ ... finalbyte @-~ 
        let j = start + 2
        while (j < data.length) {
          const c = data[j]
          if (c >= '@' && c <= '~') return j - start
          j++
        }
        return data.length - start - 1
      }
      if (next === ']') {
        // OSC: ESC ] ... BEL or ST
        let j = start + 2
        while (j < data.length) {
          if (data[j] === '\u0007') return j - start
          if (data[j] === '\u001b' && data[j + 1] === '\\') return j + 1 - start
          j++
        }
        return data.length - start - 1
      }
      // 简单 ESC + 单字符
      return 1
    }

    /** @returns {string|null} cd 目标；非 cd 命令返回 null */
    const parseCdTarget = (line) => {
      const m = line.match(/^(?:builtin\s+)?cd(?:\s+--)?(?:\s+(.*))?$/)
      if (!m) return null
      let target = (m[1] || '~').trim()
      if ((target.startsWith('"') && target.endsWith('"')) ||
          (target.startsWith("'") && target.endsWith("'"))) {
        target = target.slice(1, -1)
      }
      if (target.length > 1) {
        target = target.replace(/\/+$/, '')
      }
      return target
    }

    const terminalThemeForPreset = (preset) => {
      const themes = {
        classic: { background: '#0d1117', foreground: '#c9d1d9', cursor: '#58a6ff' },
        monokai: { background: '#272822', foreground: '#f8f8f2', cursor: '#f8f8f0' },
        solarized: { background: '#002b36', foreground: '#839496', cursor: '#93a1a1' },
      }
      return themes[preset] || themes.classic
    }

    const decodeBase64 = (b64) => {
      const binary = atob(b64)
      const bytes = new Uint8Array(binary.length)
      for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i)
      }
      return bytes
    }

    const clearFitTimers = () => {
      fitTimers.forEach((id) => clearTimeout(id))
      fitTimers = []
    }

    const scheduleFit = () => {
      clearFitTimers()
      const run = () => {
        if (!term.value || !fitAddon.value || !props.active || !props.connected) return
        try {
          fitAddon.value.fit()
          const { cols, rows } = term.value
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
      term.value.options.theme = terminalThemeForPreset(terminalPreset.value)
      scheduleFit()
    }

    const initTerminal = async () => {
      if (initialized || !terminalRef.value || !props.connected || !props.active) return
      await nextTick()
      const terminal = new Terminal({
        cursorBlink: true,
        fontSize: shellFontSize.value || 13,
        lineHeight: shellLineHeight.value || 1.2,
        fontFamily: 'Consolas, "Courier New", monospace',
        theme: terminalThemeForPreset(terminalPreset.value),
      })
      const fit = new FitAddon()
      const search = new SearchAddon()
      terminal.loadAddon(fit)
      terminal.loadAddon(search)
      terminal.open(terminalRef.value)

      terminal.onData((data) => {
        App.SendShellInput(props.machineName, data).catch(() => {})
        trackInputForCd(data)
      })

      term.value = terminal
      fitAddon.value = fit
      searchAddon.value = search
      initialized = true
      attachWriter(terminal)
      setupObservers()
      scheduleFit()
      terminal.focus()
    }

    const destroyTerminal = () => {
      clearFitTimers()
      detachWriter()
      if (resizeObserver) {
        resizeObserver.disconnect()
        resizeObserver = null
      }
      if (term.value) {
        term.value.dispose()
        term.value = null
        fitAddon.value = null
        searchAddon.value = null
      }
      initialized = false
    }

    const clear = () => term.value?.clear()

    const searchOptions = () => ({
      caseSensitive: false,
      regex: false,
      incremental: false,
    })

    const findNext = () => {
      const query = props.searchQuery.trim()
      if (!searchAddon.value || !query) return false
      return searchAddon.value.findNext(query, searchOptions())
    }

    const findPrevious = () => {
      const query = props.searchQuery.trim()
      if (!searchAddon.value || !query) return false
      return searchAddon.value.findPrevious(query, searchOptions())
    }

    const clearSearch = () => {
      searchAddon.value?.clearDecorations()
    }

    const attachWriter = (terminal) => {
      unregisterWriter?.()
      unregisterWriter = registerShellWriter(props.machineName, {
        writeData: (b64) => terminal.write(decodeBase64(b64)),
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
      if (val) {
        if (props.active) await initTerminal()
      } else {
        window.removeEventListener('resize', scheduleFit)
        destroyTerminal()
      }
    })

    watch(() => props.active, async (val) => {
      if (val && props.connected) {
        if (!initialized) await initTerminal()
        else {
          setupObservers()
          scheduleFit()
        }
        await nextTick()
        term.value?.focus()
      }
    })

    watch(() => props.searchQuery, (query) => {
      if (!props.active || !query.trim()) {
        clearSearch()
        return
      }
      findNext()
    })

    watch([shellFontSize, shellLineHeight, terminalPreset], () => {
      applyTerminalAppearance()
    })

    const onThemeChanged = (settings) => {
      if (settings?.shellFontSize > 0) shellFontSize.value = settings.shellFontSize
      if (settings?.shellLineHeight > 0) shellLineHeight.value = settings.shellLineHeight
      if (settings?.terminalPreset) terminalPreset.value = settings.terminalPreset
      applyTerminalAppearance()
    }

    onMounted(() => {
      if (props.connected && props.active) initTerminal()
      EventsOn('theme:changed', onThemeChanged)
    })

    onUnmounted(() => {
      EventsOff('theme:changed')
      window.removeEventListener('resize', scheduleFit)
      destroyTerminal()
    })

    expose({ clear, fitAndResize, findNext, findPrevious, clearSearch })

    return { containerRef, terminalRef }
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
