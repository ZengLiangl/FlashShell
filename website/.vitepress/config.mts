import { defineConfig } from 'vitepress'

export default defineConfig({
  lang: 'zh-CN',
  title: 'FlashShell',
  description:
    'FlashShell — 多会话 SSH / SFTP 桌面终端。YAML 任务流水线 × 多会话 Shell × 本地 PTY，把构建、发布、联调、登舰运维收进同一个桌面港。',
  cleanUrls: true,
  lastUpdated: true,
  ignoreDeadLinks: true,

  // 开发/预览默认只绑 127.0.0.1；开放到局域网需监听 0.0.0.0
  vite: {
    server: {
      host: true,
      port: 5173,
    },
    preview: {
      host: true,
      port: 4173,
    },
  },

  head: [
    ['link', { rel: 'icon', type: 'image/png', href: '/favicon.png' }],
    ['link', { rel: 'apple-touch-icon', href: '/logo.png' }],
    ['meta', { name: 'theme-color', content: '#409eff' }],
    ['meta', { name: 'keywords', content: 'FlashShell,SSH,SFTP,终端,桌面,Wails,Vue,运维,任务流水线,YAML' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:locale', content: 'zh-CN' }],
    ['meta', { property: 'og:title', content: 'FlashShell — 多会话 SSH / SFTP 桌面终端' }],
    ['meta', { property: 'og:site_name', content: 'FlashShell' }],
    [
      'meta',
      {
        property: 'og:description',
        content: 'YAML 任务流水线 × 多会话 SSH / SFTP × 本地 Shell，任务与终端并行，互不抢舵。',
      },
    ],
    ['meta', { name: 'twitter:card', content: 'summary' }],
  ],

  themeConfig: {
    logo: '/logo.png',
    siteTitle: 'FlashShell',
    outline: { label: '本页目录', level: [2, 3] },
    lastUpdated: { text: '最后更新' },
    darkModeSwitchLabel: '外观',
    lightModeSwitchTitle: '切换到浅色',
    darkModeSwitchTitle: '切换到深色',
    returnToTopLabel: '回到顶部',
    docFooter: { prev: '上一页', next: '下一页' },
    sidebarMenuLabel: '菜单',

    search: {
      provider: 'local',
      options: {
        translations: {
          button: { buttonText: '搜索文档', buttonAriaLabel: '搜索文档' },
          modal: {
            displayDetails: '显示详情',
            resetButtonTitle: '清除查询',
            backButtonTitle: '关闭搜索',
            noResultsText: '没有找到相关结果',
            footer: {
              selectText: '选择',
              navigateText: '切换',
              closeText: '关闭',
            },
          },
        },
      },
    },

    nav: [
      { text: '首页', link: '/' },
      { text: '为什么用', link: '/why' },
      { text: '快速开始', link: '/guide/quick-start' },
      {
        text: '功能',
        items: [
          { text: 'Shell 工作台', link: '/features/shell' },
          { text: '任务流水线', link: '/features/tasks' },
          { text: 'SFTP 文件管理', link: '/features/sftp' },
          { text: 'SSH 隧道', link: '/features/tunnel' },
          { text: '安全与配置', link: '/features/security' },
        ],
      },
      { text: '下载', link: '/download' },
      { text: '常见问题', link: '/faq' },
      {
        text: 'GitHub',
        link: 'https://github.com/ZengLiangl/FlashShell',
      },
    ],

    sidebar: {
      '/guide/': [
        {
          text: '入门',
          items: [
            { text: '快速开始', link: '/guide/quick-start' },
            { text: '安装与构建', link: '/guide/installation' },
          ],
        },
      ],
      '/features/': [
        {
          text: '功能',
          items: [
            { text: 'Shell 工作台', link: '/features/shell' },
            { text: '任务流水线', link: '/features/tasks' },
            { text: 'SFTP 文件管理', link: '/features/sftp' },
            { text: 'SSH 隧道', link: '/features/tunnel' },
            { text: '安全与配置', link: '/features/security' },
          ],
        },
      ],
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/ZengLiangl/FlashShell' },
    ],

    footer: {
      message: 'MIT License · Go + Wails v2 · Vue 3 + xterm.js',
      copyright: 'Copyright © 2026 FlashShell',
    },
  },
})
