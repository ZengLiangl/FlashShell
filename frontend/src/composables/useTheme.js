import { ref, watch } from 'vue'
import * as App from '../../wailsjs/go/app/App'
import { WindowSetDarkTheme, WindowSetLightTheme, WindowSetSystemDefaultTheme, WindowSetBackgroundColour } from '../../wailsjs/runtime/runtime'

const isDark = ref(false)
const terminalPreset = ref('classic')
const themeMode = ref('light')

function applyWindowChrome(dark) {
  if (dark) {
    WindowSetDarkTheme()
    WindowSetBackgroundColour(20, 20, 20, 255)
  } else {
    WindowSetLightTheme()
    WindowSetBackgroundColour(255, 255, 255, 255)
  }
}

function applyDomTheme(mode, preset) {
  const root = document.documentElement
  const dark = mode === 'dark'
  root.classList.toggle('dark', dark)
  root.classList.remove('terminal-preset-classic', 'terminal-preset-monokai', 'terminal-preset-solarized')
  root.classList.add(`terminal-preset-${preset || 'classic'}`)

  if (mode === 'system') {
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    root.classList.toggle('dark', prefersDark)
    isDark.value = prefersDark
    applyWindowChrome(prefersDark)
    return
  }

  applyWindowChrome(dark)
  isDark.value = dark
}

export function useTheme() {
  const loadTheme = async () => {
    try {
      const settings = await App.GetThemeSettings()
      themeMode.value = settings.mode || 'light'
      terminalPreset.value = settings.terminalPreset || 'classic'
      applyDomTheme(themeMode.value, terminalPreset.value)
    } catch {
      applyDomTheme('light', 'classic')
    }
  }

  const saveTheme = async (mode, preset) => {
    themeMode.value = mode
    terminalPreset.value = preset
    applyDomTheme(mode, preset)
    await App.SaveThemeSettings({ mode, terminalPreset: preset })
  }

  watch(themeMode, (mode) => applyDomTheme(mode, terminalPreset.value))

  return { isDark, themeMode, terminalPreset, loadTheme, saveTheme, applyDomTheme }
}
