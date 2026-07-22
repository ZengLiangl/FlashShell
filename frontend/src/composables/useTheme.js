import { ref, watch } from 'vue'
import * as App from '../../wailsjs/go/app/App'
import { WindowSetDarkTheme, WindowSetLightTheme, WindowSetSystemDefaultTheme, WindowSetBackgroundColour } from '../../wailsjs/runtime/runtime'
import { getUiAccent, getUiFont, isCustomUiAccent, UI_ACCENTS, TERMINAL_PRESETS } from '../utils/themePresets'

const isDark = ref(false)
const terminalPreset = ref('classic')
const themeMode = ref('light')
const uiAccent = ref('blue')
const uiFontFamily = ref('system')
const shellFontFamily = ref('consolas')
const shellFontSize = ref(13)
const shellLineHeight = ref(1.2)

let systemMediaQuery = null
let systemListener = null

const ACCENT_CLASSES = [...UI_ACCENTS.map((a) => `ui-accent-${a.id}`), 'ui-accent-custom']
const TERMINAL_CLASSES = TERMINAL_PRESETS.map((p) => `terminal-preset-${p.id}`)

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

function applyAccentAndFont(accentId, fontId, dark) {
  const root = document.documentElement
  const body = document.body
  root.classList.remove(...ACCENT_CLASSES)
  const accentClass = isCustomUiAccent(accentId) ? 'ui-accent-custom' : `ui-accent-${accentId || 'blue'}`
  root.classList.add(accentClass)

  const accent = getUiAccent(accentId)
  const palette = dark ? accent.dark : accent.light
  root.style.setProperty('--app-accent-color', palette.accent)
  root.style.setProperty('--app-accent-bg', palette.accentBg)

  const font = getUiFont(fontId)
  root.style.setProperty('--app-font-family', font.value)
  if (body) body.style.fontFamily = font.value
}

function applyDomTheme(mode, preset, accentId = uiAccent.value, fontId = uiFontFamily.value) {
  const root = document.documentElement
  const body = document.body
  root.classList.remove(...TERMINAL_CLASSES)
  root.classList.add(`terminal-preset-${preset || 'classic'}`)

  clearSystemListener()

  const setDark = (dark) => {
    root.classList.toggle('dark', dark)
    body?.classList.toggle('dark', dark)
    isDark.value = dark
    applyWindowChrome(dark)
    applyAccentAndFont(accentId, fontId, dark)
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

function normalizeSettings(settings = {}) {
  return {
    mode: settings.mode || 'light',
    uiAccent: settings.uiAccent || 'blue',
    terminalPreset: settings.terminalPreset || 'classic',
    uiFontFamily: settings.uiFontFamily || 'system',
    shellFontFamily: settings.shellFontFamily || 'consolas',
    shellFontSize: settings.shellFontSize > 0 ? settings.shellFontSize : 13,
    shellLineHeight: settings.shellLineHeight > 0 ? settings.shellLineHeight : 1.2,
  }
}

export function useTheme() {
  const loadTheme = async () => {
    try {
      const settings = normalizeSettings(await App.GetThemeSettings())
      themeMode.value = settings.mode
      uiAccent.value = settings.uiAccent
      terminalPreset.value = settings.terminalPreset
      uiFontFamily.value = settings.uiFontFamily
      shellFontFamily.value = settings.shellFontFamily
      shellFontSize.value = settings.shellFontSize
      shellLineHeight.value = settings.shellLineHeight
      applyDomTheme(themeMode.value, terminalPreset.value, uiAccent.value, uiFontFamily.value)
    } catch {
      applyDomTheme('light', 'classic', 'blue', 'system')
    }
  }

  const saveTheme = async (partial) => {
    const next = normalizeSettings({
      mode: themeMode.value,
      uiAccent: uiAccent.value,
      terminalPreset: terminalPreset.value,
      uiFontFamily: uiFontFamily.value,
      shellFontFamily: shellFontFamily.value,
      shellFontSize: shellFontSize.value,
      shellLineHeight: shellLineHeight.value,
      ...partial,
    })
    themeMode.value = next.mode
    uiAccent.value = next.uiAccent
    terminalPreset.value = next.terminalPreset
    uiFontFamily.value = next.uiFontFamily
    shellFontFamily.value = next.shellFontFamily
    shellFontSize.value = next.shellFontSize
    shellLineHeight.value = next.shellLineHeight
    applyDomTheme(next.mode, next.terminalPreset, next.uiAccent, next.uiFontFamily)
    await App.SaveThemeSettings(next)
  }

  const applyThemeSettings = (settings) => {
    if (!settings) return
    const next = normalizeSettings({
      mode: settings.mode || themeMode.value,
      uiAccent: settings.uiAccent || uiAccent.value,
      terminalPreset: settings.terminalPreset || terminalPreset.value,
      uiFontFamily: settings.uiFontFamily || uiFontFamily.value,
      shellFontFamily: settings.shellFontFamily || shellFontFamily.value,
      shellFontSize: settings.shellFontSize || shellFontSize.value,
      shellLineHeight: settings.shellLineHeight || shellLineHeight.value,
    })
    themeMode.value = next.mode
    uiAccent.value = next.uiAccent
    terminalPreset.value = next.terminalPreset
    uiFontFamily.value = next.uiFontFamily
    shellFontFamily.value = next.shellFontFamily
    shellFontSize.value = next.shellFontSize
    shellLineHeight.value = next.shellLineHeight
    applyDomTheme(next.mode, next.terminalPreset, next.uiAccent, next.uiFontFamily)
  }

  watch(themeMode, (mode) => applyDomTheme(mode, terminalPreset.value, uiAccent.value, uiFontFamily.value))

  return {
    isDark,
    themeMode,
    uiAccent,
    terminalPreset,
    uiFontFamily,
    shellFontFamily,
    shellFontSize,
    shellLineHeight,
    loadTheme,
    saveTheme,
    applyDomTheme,
    applyThemeSettings,
  }
}
