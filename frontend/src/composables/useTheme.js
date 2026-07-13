import { ref, watch } from 'vue'
import * as App from '../../wailsjs/go/app/App'
import { WindowSetDarkTheme, WindowSetLightTheme, WindowSetSystemDefaultTheme, WindowSetBackgroundColour } from '../../wailsjs/runtime/runtime'

const isDark = ref(false)
const terminalPreset = ref('classic')
const themeMode = ref('light')
const shellFontSize = ref(13)
const shellLineHeight = ref(1.2)

let systemMediaQuery = null
let systemListener = null

function applyWindowChrome(dark) {
  if (dark) {
    WindowSetDarkTheme()
    WindowSetBackgroundColour(20, 20, 20, 255)
  } else {
    WindowSetLightTheme()
    WindowSetBackgroundColour(255, 255, 255, 255)
  }
}

function clearSystemListener() {
  if (systemMediaQuery && systemListener) {
    systemMediaQuery.removeEventListener('change', systemListener)
  }
  systemMediaQuery = null
  systemListener = null
}

function applyDomTheme(mode, preset) {
  const root = document.documentElement
  const body = document.body
  root.classList.remove('terminal-preset-classic', 'terminal-preset-monokai', 'terminal-preset-solarized')
  root.classList.add(`terminal-preset-${preset || 'classic'}`)

  clearSystemListener()

  const setDark = (dark) => {
    root.classList.toggle('dark', dark)
    body?.classList.toggle('dark', dark)
    isDark.value = dark
    applyWindowChrome(dark)
  }

  if (mode === 'system') {
    systemMediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    setDark(systemMediaQuery.matches)
    systemListener = (e) => setDark(e.matches)
    systemMediaQuery.addEventListener('change', systemListener)
    WindowSetSystemDefaultTheme()
    return
  }

  setDark(mode === 'dark')
}

export function useTheme() {
  const loadTheme = async () => {
    try {
      const settings = await App.GetThemeSettings()
      themeMode.value = settings.mode || 'light'
      terminalPreset.value = settings.terminalPreset || 'classic'
      shellFontSize.value = settings.shellFontSize > 0 ? settings.shellFontSize : 13
      shellLineHeight.value = settings.shellLineHeight > 0 ? settings.shellLineHeight : 1.2
      applyDomTheme(themeMode.value, terminalPreset.value)
    } catch {
      applyDomTheme('light', 'classic')
    }
  }

  const saveTheme = async (mode, preset, fontSize, lineHeight) => {
    themeMode.value = mode
    terminalPreset.value = preset
    if (fontSize > 0) shellFontSize.value = fontSize
    if (lineHeight > 0) shellLineHeight.value = lineHeight
    applyDomTheme(mode, preset)
    await App.SaveThemeSettings({
      mode,
      terminalPreset: preset,
      shellFontSize: shellFontSize.value,
      shellLineHeight: shellLineHeight.value,
    })
  }

  const applyThemeSettings = (settings) => {
    if (!settings) return
    themeMode.value = settings.mode || themeMode.value
    terminalPreset.value = settings.terminalPreset || terminalPreset.value
    if (settings.shellFontSize > 0) shellFontSize.value = settings.shellFontSize
    if (settings.shellLineHeight > 0) shellLineHeight.value = settings.shellLineHeight
    applyDomTheme(themeMode.value, terminalPreset.value)
  }

  watch(themeMode, (mode) => applyDomTheme(mode, terminalPreset.value))

  return {
    isDark,
    themeMode,
    terminalPreset,
    shellFontSize,
    shellLineHeight,
    loadTheme,
    saveTheme,
    applyDomTheme,
    applyThemeSettings,
  }
}
