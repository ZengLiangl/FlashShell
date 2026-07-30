/** UI 强调色（界面主题变体） */
export const UI_ACCENTS = [
  {
    id: 'blue',
    label: '默认蓝',
    light: { accent: '#409eff', accentBg: '#ecf5ff' },
    dark: { accent: '#79bbff', accentBg: '#1a3a5c' },
  },
  {
    id: 'indigo',
    label: '靛蓝',
    light: { accent: '#4f46e5', accentBg: '#eef2ff' },
    dark: { accent: '#818cf8', accentBg: '#312e81' },
  },
  {
    id: 'violet',
    label: '紫罗兰',
    light: { accent: '#7c3aed', accentBg: '#f5f3ff' },
    dark: { accent: '#a78bfa', accentBg: '#4c1d95' },
  },
  {
    id: 'purple',
    label: '葡萄紫',
    light: { accent: '#9333ea', accentBg: '#faf5ff' },
    dark: { accent: '#c084fc', accentBg: '#581c87' },
  },
  {
    id: 'fuchsia',
    label: '品红',
    light: { accent: '#c026d3', accentBg: '#fdf4ff' },
    dark: { accent: '#e879f9', accentBg: '#701a75' },
  },
  {
    id: 'hotpink',
    label: '亮粉',
    light: { accent: '#ff4da6', accentBg: '#fff0f7' },
    dark: { accent: '#ff7abc', accentBg: '#6b1a45' },
  },
  {
    id: 'rose',
    label: '玫红',
    light: { accent: '#e11d48', accentBg: '#fff1f2' },
    dark: { accent: '#fb7185', accentBg: '#881337' },
  },
  {
    id: 'red',
    label: '正红',
    light: { accent: '#dc2626', accentBg: '#fef2f2' },
    dark: { accent: '#f87171', accentBg: '#7f1d1d' },
  },
  {
    id: 'orange',
    label: '橘橙',
    light: { accent: '#ea580c', accentBg: '#fff7ed' },
    dark: { accent: '#fb923c', accentBg: '#7c2d12' },
  },
  {
    id: 'amber',
    label: '琥珀',
    light: { accent: '#d97706', accentBg: '#fff7e8' },
    dark: { accent: '#fbbf24', accentBg: '#78350f' },
  },
  {
    id: 'yellow',
    label: '金黄',
    light: { accent: '#ca8a04', accentBg: '#fefce8' },
    dark: { accent: '#facc15', accentBg: '#713f12' },
  },
  {
    id: 'lime',
    label: '青柠',
    light: { accent: '#65a30d', accentBg: '#f7fee7' },
    dark: { accent: '#a3e635', accentBg: '#365314' },
  },
  {
    id: 'green',
    label: '森绿',
    light: { accent: '#16a34a', accentBg: '#e8f8ee' },
    dark: { accent: '#4ade80', accentBg: '#14532d' },
  },
  {
    id: 'emerald',
    label: '翠绿',
    light: { accent: '#059669', accentBg: '#ecfdf5' },
    dark: { accent: '#34d399', accentBg: '#064e3b' },
  },
  {
    id: 'teal',
    label: '青石',
    light: { accent: '#0d9488', accentBg: '#e6fffa' },
    dark: { accent: '#2dd4bf', accentBg: '#134e4a' },
  },
  {
    id: 'cyan',
    label: '青蓝',
    light: { accent: '#0891b2', accentBg: '#ecfeff' },
    dark: { accent: '#22d3ee', accentBg: '#164e63' },
  },
  {
    id: 'sky',
    label: '天空',
    light: { accent: '#0284c7', accentBg: '#f0f9ff' },
    dark: { accent: '#38bdf8', accentBg: '#0c4a6e' },
  },
  {
    id: 'slate',
    label: '石墨',
    light: { accent: '#475569', accentBg: '#f1f5f9' },
    dark: { accent: '#94a3b8', accentBg: '#1e293b' },
  },
  {
    id: 'zinc',
    label: '锌灰',
    light: { accent: '#52525b', accentBg: '#f4f4f5' },
    dark: { accent: '#a1a1aa', accentBg: '#27272a' },
  },
]

/** 终端配色预设（xterm theme） */
export const TERMINAL_PRESETS = [
  {
    id: 'classic',
    label: 'Classic',
    theme: {
      background: '#0d1117',
      foreground: '#c9d1d9',
      cursor: '#58a6ff',
      selectionBackground: '#1f6feb',
      selectionForeground: '#ffffff',
      black: '#484f58',
      red: '#ff7b72',
      green: '#3fb950',
      yellow: '#d29922',
      blue: '#58a6ff',
      magenta: '#bc8cff',
      cyan: '#39c5cf',
      white: '#b1bac4',
    },
  },
  {
    id: 'monokai',
    label: 'Monokai',
    theme: {
      background: '#272822',
      foreground: '#f8f8f2',
      cursor: '#f8f8f0',
      selectionBackground: '#49483e',
      selectionForeground: '#f8f8f2',
      black: '#272822',
      red: '#f92672',
      green: '#a6e22e',
      yellow: '#f4bf75',
      blue: '#66d9ef',
      magenta: '#ae81ff',
      cyan: '#a1efe4',
      white: '#f8f8f2',
    },
  },
  {
    id: 'solarized',
    label: 'Solarized Dark',
    theme: {
      background: '#002b36',
      foreground: '#839496',
      cursor: '#93a1a1',
      selectionBackground: '#073642',
      selectionForeground: '#fdf6e3',
      black: '#073642',
      red: '#dc322f',
      green: '#859900',
      yellow: '#b58900',
      blue: '#268bd2',
      magenta: '#d33682',
      cyan: '#2aa198',
      white: '#eee8d5',
    },
  },
  {
    id: 'solarized-light',
    label: 'Solarized Light',
    theme: {
      background: '#fdf6e3',
      foreground: '#657b83',
      cursor: '#586e75',
      selectionBackground: '#eee8d5',
      selectionForeground: '#073642',
      black: '#073642',
      red: '#dc322f',
      green: '#859900',
      yellow: '#b58900',
      blue: '#268bd2',
      magenta: '#d33682',
      cyan: '#2aa198',
      white: '#eee8d5',
    },
  },
  {
    id: 'dracula',
    label: 'Dracula',
    theme: {
      background: '#282a36',
      foreground: '#f8f8f2',
      cursor: '#f8f8f2',
      selectionBackground: '#44475a',
      selectionForeground: '#f8f8f2',
      black: '#21222c',
      red: '#ff5555',
      green: '#50fa7b',
      yellow: '#f1fa8c',
      blue: '#bd93f9',
      magenta: '#ff79c6',
      cyan: '#8be9fd',
      white: '#f8f8f2',
    },
  },
  {
    id: 'nord',
    label: 'Nord',
    theme: {
      background: '#2e3440',
      foreground: '#d8dee9',
      cursor: '#d8dee9',
      selectionBackground: '#434c5e',
      selectionForeground: '#eceff4',
      black: '#3b4252',
      red: '#bf616a',
      green: '#a3be8c',
      yellow: '#ebcb8b',
      blue: '#81a1c1',
      magenta: '#b48ead',
      cyan: '#88c0d0',
      white: '#e5e9f0',
    },
  },
  {
    id: 'one-dark',
    label: 'One Dark',
    theme: {
      background: '#282c34',
      foreground: '#abb2bf',
      cursor: '#528bff',
      selectionBackground: '#3e4451',
      selectionForeground: '#abb2bf',
      black: '#282c34',
      red: '#e06c75',
      green: '#98c379',
      yellow: '#e5c07b',
      blue: '#61afef',
      magenta: '#c678dd',
      cyan: '#56b6c2',
      white: '#abb2bf',
    },
  },
  {
    id: 'tokyo-night',
    label: 'Tokyo Night',
    theme: {
      background: '#1a1b26',
      foreground: '#a9b1d6',
      cursor: '#c0caf5',
      selectionBackground: '#33467c',
      selectionForeground: '#c0caf5',
      black: '#15161e',
      red: '#f7768e',
      green: '#9ece6a',
      yellow: '#e0af68',
      blue: '#7aa2f7',
      magenta: '#bb9af7',
      cyan: '#7dcfff',
      white: '#a9b1d6',
    },
  },
  {
    id: 'catppuccin-mocha',
    label: 'Catppuccin Mocha',
    theme: {
      background: '#1e1e2e',
      foreground: '#cdd6f4',
      cursor: '#f5e0dc',
      selectionBackground: '#585b70',
      selectionForeground: '#cdd6f4',
      black: '#45475a',
      red: '#f38ba8',
      green: '#a6e3a1',
      yellow: '#f9e2af',
      blue: '#89b4fa',
      magenta: '#cba6f7',
      cyan: '#94e2d5',
      white: '#bac2de',
    },
  },
  {
    id: 'catppuccin-latte',
    label: 'Catppuccin Latte',
    theme: {
      background: '#eff1f5',
      foreground: '#4c4f69',
      cursor: '#dc8a78',
      selectionBackground: '#acb0be',
      selectionForeground: '#4c4f69',
      black: '#5c5f77',
      red: '#d20f39',
      green: '#40a02b',
      yellow: '#df8e1d',
      blue: '#1e66f5',
      magenta: '#8839ef',
      cyan: '#179299',
      white: '#acb0be',
    },
  },
  {
    id: 'gruvbox-dark',
    label: 'Gruvbox Dark',
    theme: {
      background: '#282828',
      foreground: '#ebdbb2',
      cursor: '#ebdbb2',
      selectionBackground: '#504945',
      selectionForeground: '#ebdbb2',
      black: '#282828',
      red: '#cc241d',
      green: '#98971a',
      yellow: '#d79921',
      blue: '#458588',
      magenta: '#b16286',
      cyan: '#689d6a',
      white: '#a89984',
    },
  },
  {
    id: 'gruvbox-light',
    label: 'Gruvbox Light',
    theme: {
      background: '#fbf1c7',
      foreground: '#3c3836',
      cursor: '#3c3836',
      selectionBackground: '#d5c4a1',
      selectionForeground: '#3c3836',
      black: '#fbf1c7',
      red: '#cc241d',
      green: '#98971a',
      yellow: '#d79921',
      blue: '#458588',
      magenta: '#b16286',
      cyan: '#689d6a',
      white: '#7c6f64',
    },
  },
  {
    id: 'ayu-dark',
    label: 'Ayu Dark',
    theme: {
      background: '#0b0e14',
      foreground: '#bfbdb6',
      cursor: '#e6b450',
      selectionBackground: '#1c212b',
      selectionForeground: '#bfbdb6',
      black: '#11151c',
      red: '#f07178',
      green: '#aad94c',
      yellow: '#ffb454',
      blue: '#59c2ff',
      magenta: '#d2a6ff',
      cyan: '#95e6cb',
      white: '#bfbdb6',
    },
  },
  {
    id: 'material',
    label: 'Material Dark',
    theme: {
      background: '#263238',
      foreground: '#eeffff',
      cursor: '#ffcc00',
      selectionBackground: '#546e7a',
      selectionForeground: '#eeffff',
      black: '#000000',
      red: '#ff5370',
      green: '#c3e88d',
      yellow: '#ffcb6b',
      blue: '#82aaff',
      magenta: '#c792ea',
      cyan: '#89ddff',
      white: '#ffffff',
    },
  },
  {
    id: 'campbell',
    label: 'Campbell',
    theme: {
      background: '#0c0c0c',
      foreground: '#cccccc',
      cursor: '#ffffff',
      selectionBackground: '#264f78',
      selectionForeground: '#ffffff',
      black: '#0c0c0c',
      red: '#c50f1f',
      green: '#13a10e',
      yellow: '#c19c00',
      blue: '#0037da',
      magenta: '#881798',
      cyan: '#3a96dd',
      white: '#cccccc',
    },
  },
  {
    id: 'ubuntu',
    label: 'Ubuntu',
    theme: {
      background: '#300a24',
      foreground: '#eeeeee',
      cursor: '#bbbbbb',
      selectionBackground: '#b5d5ff',
      selectionForeground: '#000000',
      black: '#2e3436',
      red: '#cc0000',
      green: '#4e9a06',
      yellow: '#c4a000',
      blue: '#3465a4',
      magenta: '#75507b',
      cyan: '#06989a',
      white: '#d3d7cf',
    },
  },
  {
    id: 'github-light',
    label: 'GitHub Light',
    theme: {
      background: '#ffffff',
      foreground: '#24292f',
      cursor: '#0969da',
      selectionBackground: '#0969da',
      selectionForeground: '#ffffff',
      black: '#24292f',
      red: '#cf222e',
      green: '#116329',
      yellow: '#4d2d00',
      blue: '#0969da',
      magenta: '#8250df',
      cyan: '#1b7c83',
      white: '#6e7781',
    },
  },
  {
    id: 'github-dark',
    label: 'GitHub Dark',
    theme: {
      background: '#0d1117',
      foreground: '#e6edf3',
      cursor: '#2f81f7',
      selectionBackground: '#1f6feb',
      selectionForeground: '#ffffff',
      black: '#484f58',
      red: '#ff7b72',
      green: '#3fb950',
      yellow: '#d29922',
      blue: '#58a6ff',
      magenta: '#bc8cff',
      cyan: '#39c5cf',
      white: '#b1bac4',
    },
  },
  {
    id: 'rose-pine',
    label: 'Rosé Pine',
    theme: {
      background: '#191724',
      foreground: '#e0def4',
      cursor: '#ebbcba',
      selectionBackground: '#26233a',
      selectionForeground: '#e0def4',
      black: '#26233a',
      red: '#eb6f92',
      green: '#31748f',
      yellow: '#f6c177',
      blue: '#9ccfd8',
      magenta: '#c4a7e7',
      cyan: '#ebbcba',
      white: '#e0def4',
    },
  },
  {
    id: 'everforest-dark',
    label: 'Everforest Dark',
    theme: {
      background: '#2d353b',
      foreground: '#d3c6aa',
      cursor: '#d3c6aa',
      selectionBackground: '#475258',
      selectionForeground: '#d3c6aa',
      black: '#475258',
      red: '#e67e80',
      green: '#a7c080',
      yellow: '#dbbc7f',
      blue: '#7fbbb3',
      magenta: '#d699b6',
      cyan: '#83c092',
      white: '#d3c6aa',
    },
  },
]

const UI_FALLBACK = 'system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Microsoft YaHei", sans-serif'
const TERM_FALLBACK = 'Consolas, "Courier New", monospace'

export const UI_FONTS = [
  {
    id: 'system',
    label: '系统默认',
    value: UI_FALLBACK,
  },
  {
    id: 'segoe',
    label: 'Segoe UI',
    value: '"Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif',
  },
  {
    id: 'yahei',
    label: '微软雅黑',
    value: '"Microsoft YaHei", "PingFang SC", "Segoe UI", sans-serif',
  },
  {
    id: 'pingfang',
    label: '苹方 / 冬青黑',
    value: '"PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif',
  },
  {
    id: 'noto',
    label: 'Noto Sans',
    value: '"Noto Sans SC", "Noto Sans", "Segoe UI", sans-serif',
  },
]

export const TERMINAL_FONTS = [
  {
    id: 'consolas',
    label: 'Consolas',
    value: 'Consolas, "Courier New", monospace',
  },
  {
    id: 'cascadia',
    label: 'Cascadia Code',
    value: '"Cascadia Code", "Cascadia Mono", Consolas, monospace',
  },
  {
    id: 'jetbrains',
    label: 'JetBrains Mono',
    value: '"JetBrains Mono", Consolas, monospace',
  },
  {
    id: 'fira',
    label: 'Fira Code',
    value: '"Fira Code", Consolas, monospace',
  },
  {
    id: 'source-code',
    label: 'Source Code Pro',
    value: '"Source Code Pro", Consolas, monospace',
  },
  {
    id: 'sarasa',
    label: '等距更纱 / 雅黑 Mono',
    value: '"Sarasa Mono SC", "Microsoft YaHei Mono", Consolas, monospace',
  },
  {
    id: 'courier',
    label: 'Courier New',
    value: '"Courier New", Courier, monospace',
  },
]

export function systemFontId(family) {
  return `sys:${family}`
}

export function isSystemFontId(id) {
  return typeof id === 'string' && id.startsWith('sys:')
}

export function systemFontFamily(id) {
  if (!isSystemFontId(id)) return ''
  return id.slice(4)
}

function quoteFamily(name) {
  const n = String(name || '').replace(/"/g, '')
  return `"${n}"`
}

export function makeSystemUiFont(family) {
  return {
    id: systemFontId(family),
    label: family,
    value: `${quoteFamily(family)}, ${UI_FALLBACK}`,
    system: true,
  }
}

export function makeSystemTerminalFont(family) {
  return {
    id: systemFontId(family),
    label: family,
    value: `${quoteFamily(family)}, ${TERM_FALLBACK}`,
    system: true,
  }
}

/** 是否为自定义强调色（直接存 hex） */
export function isCustomUiAccent(id) {
  return typeof id === 'string' && /^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/.test(id.trim())
}

function normalizeHex(hex) {
  let h = String(hex || '').trim()
  if (!h.startsWith('#')) h = `#${h}`
  if (/^#[0-9a-fA-F]{3}$/.test(h)) {
    h = `#${h[1]}${h[1]}${h[2]}${h[2]}${h[3]}${h[3]}`
  }
  return h.toLowerCase()
}

function parseHex(hex) {
  const h = normalizeHex(hex).slice(1)
  return {
    r: parseInt(h.slice(0, 2), 16),
    g: parseInt(h.slice(2, 4), 16),
    b: parseInt(h.slice(4, 6), 16),
  }
}

function toHex({ r, g, b }) {
  const c = (n) => Math.max(0, Math.min(255, Math.round(n))).toString(16).padStart(2, '0')
  return `#${c(r)}${c(g)}${c(b)}`
}

/** t=0 取 a，t=1 取 b */
function mixHex(a, b, t) {
  const A = parseHex(a)
  const B = parseHex(b)
  return toHex({
    r: A.r + (B.r - A.r) * t,
    g: A.g + (B.g - A.g) * t,
    b: A.b + (B.b - A.b) * t,
  })
}

/**
 * 将强调色同步为 Element Plus primary 色阶（按钮/链接悬浮、树选中等依赖这些变量）。
 * 暗色模式需向深色底混色，否则 light-9 会变成近白色，SFTP 文件树选中会刺眼。
 */
export function applyElementPrimaryCssVars(root, accentHex, dark = false) {
  if (!root?.style) return
  const accent = normalizeHex(accentHex || '#409eff')
  const { r, g, b } = parseHex(accent)
  root.style.setProperty('--el-color-primary', accent)
  root.style.setProperty('--el-color-primary-rgb', `${r}, ${g}, ${b}`)
  if (dark) {
    // 与 Element Plus html.dark 一致：light-* 向页面底色混，dark-2 向白提亮
    const base = '#141414'
    root.style.setProperty('--el-color-primary-light-3', mixHex(accent, base, 0.3))
    root.style.setProperty('--el-color-primary-light-5', mixHex(accent, base, 0.5))
    root.style.setProperty('--el-color-primary-light-7', mixHex(accent, base, 0.7))
    root.style.setProperty('--el-color-primary-light-8', mixHex(accent, base, 0.8))
    root.style.setProperty('--el-color-primary-light-9', mixHex(accent, base, 0.9))
    root.style.setProperty('--el-color-primary-dark-2', mixHex(accent, '#ffffff', 0.2))
  } else {
    root.style.setProperty('--el-color-primary-light-3', mixHex(accent, '#ffffff', 0.3))
    root.style.setProperty('--el-color-primary-light-5', mixHex(accent, '#ffffff', 0.5))
    root.style.setProperty('--el-color-primary-light-7', mixHex(accent, '#ffffff', 0.7))
    root.style.setProperty('--el-color-primary-light-8', mixHex(accent, '#ffffff', 0.8))
    root.style.setProperty('--el-color-primary-light-9', mixHex(accent, '#ffffff', 0.9))
    root.style.setProperty('--el-color-primary-dark-2', mixHex(accent, '#000000', 0.2))
  }
}

/** 由任意 hex 生成浅色/深色强调色板 */
export function buildAccentFromHex(hex) {
  const accent = normalizeHex(hex)
  return {
    id: 'custom',
    label: '自定义',
    light: {
      accent,
      accentBg: mixHex(accent, '#ffffff', 0.88),
    },
    dark: {
      accent: mixHex(accent, '#ffffff', 0.28),
      // 仅作回退；实际 DOM 由 resolveAccentBg 生成半透明主题色
      accentBg: mixHex(accent, '#000000', 0.55),
    },
  }
}

/** 深色模式强调底用半透明主题色，避免写死深紫/酒红底看不出主题色 */
export function resolveAccentBg(accentHex, dark, lightBg) {
  if (!dark) return lightBg
  return `color-mix(in srgb, ${accentHex} 22%, transparent)`
}

export function getUiAccent(id) {
  if (isCustomUiAccent(id)) return buildAccentFromHex(id)
  return UI_ACCENTS.find((a) => a.id === id) || UI_ACCENTS[0]
}

/** 调色盘预定义色：各预设浅色强调色 */
export function collectUiAccentPredefineColors() {
  return UI_ACCENTS.map((a) => a.light.accent)
}

export function getTerminalPreset(id) {
  return TERMINAL_PRESETS.find((p) => p.id === id) || TERMINAL_PRESETS[0]
}

export function getUiFont(id) {
  const found = UI_FONTS.find((f) => f.id === id)
  if (found) return found
  if (isSystemFontId(id)) return makeSystemUiFont(systemFontFamily(id))
  if (id) return makeSystemUiFont(id)
  return UI_FONTS[0]
}

export function getTerminalFont(id) {
  const found = TERMINAL_FONTS.find((f) => f.id === id)
  if (found) return found
  if (isSystemFontId(id)) return makeSystemTerminalFont(systemFontFamily(id))
  if (id) return makeSystemTerminalFont(id)
  return TERMINAL_FONTS[0]
}

export function terminalThemeForPreset(preset) {
  return getTerminalPreset(preset).theme
}

export function mergeUiFontOptions(systemFonts = []) {
  const seen = new Set(UI_FONTS.map((f) => f.id))
  const extras = []
  for (const f of systemFonts) {
    const family = f?.family || f?.Family
    if (!family) continue
    const item = makeSystemUiFont(family)
    if (seen.has(item.id)) continue
    seen.add(item.id)
    extras.push(item)
  }
  return [...UI_FONTS, ...extras]
}

export function mergeTerminalFontOptions(systemFonts = []) {
  const seen = new Set(TERMINAL_FONTS.map((f) => f.id))
  const mono = []
  const other = []
  for (const f of systemFonts) {
    const family = f?.family || f?.Family
    if (!family) continue
    const item = makeSystemTerminalFont(family)
    if (seen.has(item.id)) continue
    seen.add(item.id)
    if (f.mono || f.Mono) mono.push(item)
    else other.push(item)
  }
  return [...TERMINAL_FONTS, ...mono, ...other]
}
