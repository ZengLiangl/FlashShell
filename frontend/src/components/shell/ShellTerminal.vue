<template>
  <div class="shell-terminal" ref="containerRef">
    <div ref="terminalRef" class="terminal-host"></div>
  </div>
</template>

<script>
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import * as App from '../../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'

export default {
  name: 'ShellTerminal',
  props: {
    machineName: { type: String, required: true },
    active: { type: Boolean, default: false },
    connected: { type: Boolean, default: false },
  },
  setup(props, { expose }) {
    const containerRef = ref(null)
    const terminalRef = ref(null)
    const term = ref(null)
    const fitAddon = ref(null)
    let resizeObserver = null
    let fitTimers = []
    let initialized = false

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

    const initTerminal = async () => {
      if (initialized || !terminalRef.value || !props.connected || !props.active) return
      await nextTick()
      const terminal = new Terminal({
        cursorBlink: true,
        fontSize: 13,
        fontFamily: 'Consolas, "Courier New", monospace',
        theme: {
          background: '#0d1117',
          foreground: '#c9d1d9',
          cursor: '#58a6ff',
        },
      })
      const fit = new FitAddon()
      terminal.loadAddon(fit)
      terminal.open(terminalRef.value)

      terminal.onData((data) => {
        App.SendShellInput(props.machineName, data).catch(() => {})
      })

      term.value = terminal
      fitAddon.value = fit
      initialized = true
      setupObservers()
      scheduleFit()
      terminal.focus()
    }

    const destroyTerminal = () => {
      clearFitTimers()
      if (resizeObserver) {
        resizeObserver.disconnect()
        resizeObserver = null
      }
      if (term.value) {
        term.value.dispose()
        term.value = null
        fitAddon.value = null
      }
      initialized = false
    }

    const clear = () => term.value?.clear()

    const handleShellData = (payload) => {
      if (!term.value || !payload || payload.machineName !== props.machineName) return
      if (!payload.data) return
      term.value.write(decodeBase64(payload.data))
    }

    const handleShellLine = (payload) => {
      if (!term.value || !payload || payload.machineName !== props.machineName) return
      if (!payload.line) return
      term.value.writeln(`\x1b[33m${payload.line}\x1b[0m`)
    }

    const handleShellClear = (payload) => {
      if (!payload || payload.machineName !== props.machineName) return
      clear()
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

    onMounted(() => {
      EventsOn('shell:data', handleShellData)
      EventsOn('shell:line', handleShellLine)
      EventsOn('shell:clear', handleShellClear)
      if (props.connected && props.active) initTerminal()
    })

    onUnmounted(() => {
      window.removeEventListener('resize', scheduleFit)
      destroyTerminal()
      EventsOff('shell:data')
      EventsOff('shell:line')
      EventsOff('shell:clear')
    })

    expose({ clear, fitAndResize })

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
