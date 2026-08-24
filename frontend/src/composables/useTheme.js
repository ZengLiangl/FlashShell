import { ref, watch } from 'vue'
import * as App from '../../wailsjs/go/app/App'
import { WindowSetDarkTheme, WindowSetLightTheme, WindowSetSystemDefaultTheme, WindowSetBackgroundColour } from '../../wailsjs/runtime/runtime'
import { getUiAccent, getUiFont, isCustomUiAccent, applyElementPrimaryCssVars, buildAccentDesignTokens, UI_ACCENTS, TERMINAL_PRESETS } from '../utils/themePresets'
import { clampShellFontSize, SHELL_FONT_SIZE_DEFAULT } from '../utils/shellTerminalUx'

const isDark = ref(false)
const terminalPreset = ref('classic')
const themeMode = ref('light')
const uiAccent = ref('blue')
const uiFontFamily = ref('system')
const uiFontSize = ref(14)
const shellFontFamily = ref('consolas')
const shellFontSize = ref(SHELL_FONT_SIZE_DEFAULT)
const shellLineHeight = ref(1.2)

let systemMediaQuery = null
let systemListener = null

const ACCENT_CLASSES = [...UI_ACCENTS.map((a) => `ui-accent-${a.id}`), 'ui-accent-custom']
const TERMINAL_CLASSES = TERMINAL_PRESETS.map((p) => `terminal-preset-${p.id}`)
const UI_FONT_SIZE_BASE = 14

function clampUiFontSize(size) {
  const n = Number(size)
  if (!Number.isFinite(n) || n <= 0) return UI_FONT_SIZE_BASE
  return Math.min(20, Math.max(12, Math.round(n)))
}

function applyWindowChrome(dark) {
  if (dark) {
    WindowSetDarkTheme()
    WindowSetBackgroundColour(20, 20, 20, 255)
  } else {
    WindowSetLightTheme()
    // 与顶栏 --app-panel-bg 一致，避免标题栏与菜单栏之间露灰缝
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

function applyAccentAndFont(accentId, fontId, fontSize, dark) {
  const root = document.documentElement
  const body = document.body
  root.classList.remove(...ACCENT_CLASSES)
  const accentClass = isCustomUiAccent(accentId) ? 'ui-accent-custom' : `ui-accent-${accentId || 'blue'}`
  root.classList.add(accentClass)

  const accent = getUiAccent(accentId)
  const palette = dark ? accent.dark : accent.light
  const tokens = buildAccentDesignTokens(palette.accent, dark, palette.accentBg)
  root.style.setProperty('--accent', tokens.accent)
  root.style.setProperty('--accent-strong', tokens.accentStrong)
  root.style.setProperty('--accent-soft', tokens.accentSoft)
  root.style.setProperty('--on-accent', tokens.onAccent)
  root.style.setProperty('--app-accent-color', tokens.accent)
  root.style.setProperty('--app-accent-bg', tokens.accentSoft)
  root.style.setProperty(
    '--app-card-hover-shadow',
    dark
      ? `color-mix(in srgb, ${tokens.accent} 25%, transparent)`
      : `color-mix(in srgb, ${tokens.accent} 15%, transparent)`,
  )
  applyElementPrimaryCssVars(root, tokens.accent, dark)

  const font = getUiFont(fontId)
  root.style.setProperty('--app-font-family', font.value)
  if (body) body.style.fontFamily = font.value

  const size = clampUiFontSize(fontSize)
  root.style.setProperty('--app-font-size', `${size}px`)
  root.style.setProperty('--app-font-scale', String(size / UI_FONT_SIZE_BASE))
  root.style.setProperty('--el-font-size-base', `${size}px`)
  root.style.setProperty('--el-font-size-extra-large', `${Math.round((size * 20) / 14)}px`)
  root.style.setProperty('--el-font-size-large', `${Math.round((size * 18) / 14)}px`)
  root.style.setProperty('--el-font-size-medium', `${Math.round((size * 16) / 14)}px`)
  root.style.setProperty('--el-font-size-small', `${Math.round((size * 13) / 14)}px`)
  root.style.setProperty('--el-font-size-extra-small', `${Math.round((size * 12) / 14)}px`)
  root.style.fontSize = `${size}px`
  if (body) body.style.fontSize = `${size}px`
}

function applyDomTheme(
  mode,
  preset,
  accentId = uiAccent.value,
  fontId = uiFontFamily.value,
  fontSize = uiFontSize.value,
) {
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
    applyAccentAndFont(accentId, fontId, fontSize, dark)
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
    uiFontSize: clampUiFontSize(settings.uiFontSize),
    shellFontFamily: settings.shellFontFamily || 'consolas',
    shellFontSize: clampShellFontSize(settings.shellFontSize),
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
      uiFontSize.value = settings.uiFontSize
      shellFontFamily.value = settings.shellFontFamily
      shellFontSize.value = settings.shellFontSize
      shellLineHeight.value = settings.shellLineHeight
      applyDomTheme(themeMode.value, terminalPreset.value, uiAccent.value, uiFontFamily.value, uiFontSize.value)
    } catch {
      applyDomTheme('light', 'classic', 'blue', 'system', 14)
    }
  }

  const saveTheme = async (partial) => {
    const next = normalizeSettings({
      mode: themeMode.value,
      uiAccent: uiAccent.value,
      terminalPreset: terminalPreset.value,
      uiFontFamily: uiFontFamily.value,
      uiFontSize: uiFontSize.value,
      shellFontFamily: shellFontFamily.value,
      shellFontSize: shellFontSize.value,
      shellLineHeight: shellLineHeight.value,
      ...partial,
    })
    themeMode.value = next.mode
    uiAccent.value = next.uiAccent
    terminalPreset.value = next.terminalPreset
    uiFontFamily.value = next.uiFontFamily
    uiFontSize.value = next.uiFontSize
    shellFontFamily.value = next.shellFontFamily
    shellFontSize.value = next.shellFontSize
    shellLineHeight.value = next.shellLineHeight
    applyDomTheme(next.mode, next.terminalPreset, next.uiAccent, next.uiFontFamily, next.uiFontSize)
    await App.SaveThemeSettings(next)
  }

  const applyThemeSettings = (settings) => {
    if (!settings) return
    const next = normalizeSettings({
      mode: settings.mode || themeMode.value,
      uiAccent: settings.uiAccent || uiAccent.value,
      terminalPreset: settings.terminalPreset || terminalPreset.value,
      uiFontFamily: settings.uiFontFamily || uiFontFamily.value,
      uiFontSize: settings.uiFontSize || uiFontSize.value,
      shellFontFamily: settings.shellFontFamily || shellFontFamily.value,
      shellFontSize: settings.shellFontSize || shellFontSize.value,
      shellLineHeight: settings.shellLineHeight || shellLineHeight.value,
    })
    themeMode.value = next.mode
    uiAccent.value = next.uiAccent
    terminalPreset.value = next.terminalPreset
    uiFontFamily.value = next.uiFontFamily
    uiFontSize.value = next.uiFontSize
    shellFontFamily.value = next.shellFontFamily
    shellFontSize.value = next.shellFontSize
    shellLineHeight.value = next.shellLineHeight
    applyDomTheme(next.mode, next.terminalPreset, next.uiAccent, next.uiFontFamily, next.uiFontSize)
  }

  watch(themeMode, (mode) => applyDomTheme(mode, terminalPreset.value, uiAccent.value, uiFontFamily.value, uiFontSize.value))

  return {
    isDark,
    themeMode,
    uiAccent,
    terminalPreset,
    uiFontFamily,
    uiFontSize,
    shellFontFamily,
    shellFontSize,
    shellLineHeight,
    loadTheme,
    saveTheme,
    applyDomTheme,
    applyThemeSettings,
  }
}
