/** 设置中心可搜索条目：section 对应 SettingsHub 侧栏 id。 */

export const SETTINGS_SEARCH_CATALOG = [
  { id: 'app-name', section: 'app', label: '应用名称', keywords: '窗口 标题 名称 windowsName' },
  { id: 'app-fullscreen', section: 'app', label: '启动时全屏', keywords: '最大化 启动 fullscreen' },
  { id: 'app-dock-icon', section: 'app', label: 'Dock 图标', keywords: '任务栏 图标 托盘 icon' },
  { id: 'app-close-tray', section: 'app', label: '关闭到托盘', keywords: '托盘 后台 隐藏 close tray' },
	{ id: 'app-min-tray', section: 'app', label: '最小化到托盘', keywords: '托盘 最小化 minimize tray' },
  { id: 'appearance-mode', section: 'appearance', label: '外观模式', keywords: '浅色 深色 主题 theme dark light' },
  { id: 'appearance-accent', section: 'appearance', label: '强调色', keywords: '颜色 accent 配色' },
  { id: 'appearance-ui-font', section: 'appearance', label: '界面字体', keywords: '字体 字号 font' },
  { id: 'appearance-opacity', section: 'appearance', label: '窗口透明度', keywords: '透明 不透明 opacity 窗口' },
  { id: 'terminal-font', section: 'appearance', label: '终端字体', keywords: '字号 行高 consolas' },
  { id: 'terminal-webgl', section: 'appearance', label: 'WebGL 渲染', keywords: '加速 webgl' },
  { id: 'terminal-reconnect', section: 'terminal', label: 'SSH 握手超时', keywords: '连接 超时 timeout' },
  { id: 'terminal-scrollback', section: 'terminal', label: '终端滚动缓冲', keywords: '回滚 行数 scrollback' },
  { id: 'terminal-ascii', section: 'terminal', label: '终端英文输入', keywords: '中文 输入法 ascii' },
  { id: 'sftp-zip', section: 'sftp', label: 'SFTP 目录压缩上传', keywords: 'zip 压缩 上传' },
  { id: 'sftp-sync', section: 'sftp', label: '外置打开自动同步', keywords: '编辑器 回传' },
  { id: 'sftp-skip', section: 'sftp', label: '跳过未变更文件', keywords: '传输 增量' },
  { id: 'sftp-concurrency', section: 'sftp', label: '传输并发数', keywords: '并发 传输' },
  { id: 'files-default', section: 'files', label: '默认文件打开方式', keywords: '打开 编辑器 关联 opener' },
  { id: 'files-assoc', section: 'files', label: 'SFTP 文件关联', keywords: '扩展名 打开方式 association 文件' },
  { id: 'accounts', section: 'accounts', label: '密钥库', keywords: '帐号 身份 密码 密钥 ssh' },
  { id: 'credentials', section: 'credentials', label: '凭据安全', keywords: '主密码 锁定 vault' },
  { id: 'security-hosts', section: 'security', label: '已信任主机', keywords: 'known_hosts 指纹 hostkey' },
  { id: 'machines', section: 'machines', label: '机器配置', keywords: '主机 ssh 服务器 导入' },
  { id: 'env', section: 'env', label: '环境变量', keywords: '工作路径 workpath' },
  { id: 'portforwards', section: 'portforwards', label: '端口转发', keywords: '隧道 tunnel 转发' },
  { id: 'proxy', section: 'proxy', label: 'HTTP 代理', keywords: 'socks 代理 proxy' },
  { id: 'shortcuts', section: 'shortcuts', label: '快捷键', keywords: '热键 片段 snippet keymap' },
  { id: 'mcp', section: 'mcp', label: 'MCP 接入', keywords: 'ai cursor 令牌 审批' },
  { id: 'about', section: 'about', label: '关于', keywords: '版本 更新 about' },
]

export function searchSettingsCatalog(query) {
  const q = String(query || '').trim().toLowerCase()
  if (!q) return []
  return SETTINGS_SEARCH_CATALOG.filter((item) => {
    const hay = `${item.label} ${item.keywords} ${item.section} ${item.id}`.toLowerCase()
    return hay.includes(q)
  })
}
