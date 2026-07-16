/** UI 强调色（界面主题变体） */
export const UI_ACCENTS = [
  {
    id: 'blue',
    label: '默认蓝',
    light: { accent: '#409eff', accentBg: '#ecf5ff' },
    dark: { accent: '#79bbff', accentBg: '#1a3a5c' },
  },
  {
    id: 'teal',
    label: '青石',
    light: { accent: '#0d9488', accentBg: '#e6fffa' },
    dark: { accent: '#2dd4bf', accentBg: '#134e4a' },
  },
  {
    id: 'green',
    label: '森绿',
    light: { accent: '#16a34a', accentBg: '#e8f8ee' },
    dark: { accent: '#4ade80', accentBg: '#14532d' },
  },
  {
    id: 'amber',
    label: '琥珀',
    light: { accent: '#d97706', accentBg: '#fff7e8' },
    dark: { accent: '#fbbf24', accentBg: '#78350f' },
  },
  {
    id: 'slate',
    label: '石墨',
    light: { accent: '#475569', accentBg: '#f1f5f9' },
    dark: { accent: '#94a3b8', accentBg: '#1e293b' },
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
]

export const UI_FONTS = [
  {
    id: 'system',
    label: '系统默认',
    value: 'system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Microsoft YaHei", sans-serif',
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

export function getUiAccent(id) {
  return UI_ACCENTS.find((a) => a.id === id) || UI_ACCENTS[0]
}

export function getTerminalPreset(id) {
  return TERMINAL_PRESETS.find((p) => p.id === id) || TERMINAL_PRESETS[0]
}

export function getUiFont(id) {
  return UI_FONTS.find((f) => f.id === id) || UI_FONTS[0]
}

export function getTerminalFont(id) {
  return TERMINAL_FONTS.find((f) => f.id === id) || TERMINAL_FONTS[0]
}

export function terminalThemeForPreset(preset) {
  return getTerminalPreset(preset).theme
}
