# FlashShell 官网（VitePress）

产品文档站与落地页，基于 [VitePress](https://vitepress.dev/) 1.6。

## 本地开发

```bash
cd website
npm install
npm run dev
```

浏览器打开终端提示的本地地址（默认 `http://localhost:5173`）。

## 构建

```bash
npm run build      # 产出 website/.vitepress/dist
npm run preview    # 预览构建结果
```

将 `dist/` 部署到任意静态托管（GitHub Pages、EdgeOne Pages、Nginx 等）即可。

## 目录说明

| 路径 | 说明 |
| --- | --- |
| `.vitepress/config.mts` | 站点配置、导航、侧栏、本地搜索 |
| `.vitepress/theme/` | 默认主题 + 品牌色自定义 |
| `index.md` | 首页 |
| `guide/` | 入门文档 |
| `features/` | 功能页 |
| `public/` | 静态资源（logo / favicon） |
