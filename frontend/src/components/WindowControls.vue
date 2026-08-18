<template>
  <div v-if="isWindows" class="window-controls" aria-label="窗口控制">
    <button type="button" class="win-btn" title="最小化" @click="minimise">
      <span class="win-glyph win-min" aria-hidden="true" />
    </button>
    <button type="button" class="win-btn" :title="maximised ? '还原' : '最大化'" @click="toggleMax">
      <span class="win-glyph" :class="maximised ? 'win-restore' : 'win-max'" aria-hidden="true" />
    </button>
    <button type="button" class="win-btn win-close" title="关闭" @click="quit">
      <span class="win-glyph win-x" aria-hidden="true" />
    </button>
  </div>
</template>

<script>
import { ref, onMounted, onUnmounted } from 'vue'
import { WindowMinimise, Quit } from '../../wailsjs/runtime/runtime'
import { isWindowsPlatform } from '../utils/platform'
import { toggleChromeWindowMaximise } from '../utils/windowChrome'
import { WindowIsChromeMaximised } from '../../wailsjs/go/app/App'

export default {
  name: 'WindowControls',
  setup() {
    const isWindows = isWindowsPlatform()
    const maximised = ref(false)
    let timer = null

    const refresh = async () => {
      try {
        maximised.value = !!(await WindowIsChromeMaximised())
      } catch {
        maximised.value = false
      }
    }

    const minimise = () => { WindowMinimise() }
    const toggleMax = async () => {
      await toggleChromeWindowMaximise()
      setTimeout(refresh, 80)
    }
    const quit = () => { Quit() }

    onMounted(() => {
      if (!isWindows) return
      refresh()
      timer = setInterval(refresh, 800)
      window.addEventListener('resize', refresh)
    })
    onUnmounted(() => {
      if (timer) clearInterval(timer)
      window.removeEventListener('resize', refresh)
    })

    return { isWindows, maximised, minimise, toggleMax, quit }
  },
}
</script>

<style scoped>
.window-controls {
  display: inline-flex;
  align-items: stretch;
  height: 100%;
  margin-left: 4px;
  flex-shrink: 0;
  --wails-draggable: no-drag;
}

.win-btn {
  width: 42px;
  height: 100%;
  min-height: 36px;
  border: none;
  padding: 0;
  margin: 0;
  background: transparent;
  color: var(--app-text-secondary, var(--app-text));
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: background 0.12s ease, color 0.12s ease;
}

.win-btn:hover {
  background: color-mix(in srgb, var(--app-text) 8%, transparent);
}

.win-btn.win-close:hover {
  background: #e81123;
  color: #fff;
}

.win-glyph {
  display: block;
  width: 10px;
  height: 10px;
  position: relative;
}

.win-min {
  height: 1px;
  width: 10px;
  background: currentColor;
  margin-top: 5px;
}

.win-max {
  border: 1px solid currentColor;
  box-sizing: border-box;
}

.win-restore {
  border: 1px solid currentColor;
  box-sizing: border-box;
  width: 8px;
  height: 8px;
  margin: 1px 0 0 1px;
  box-shadow: 1px -1px 0 0 currentColor;
}

.win-x::before,
.win-x::after {
  content: '';
  position: absolute;
  left: 4px;
  top: 0;
  width: 1px;
  height: 10px;
  background: currentColor;
}

.win-x::before { transform: rotate(45deg); }
.win-x::after { transform: rotate(-45deg); }
</style>
