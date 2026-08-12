<template>
    <div class="general-settings-panel" :class="{ embedded, 'is-panel-routed': !!panel }">
        <!-- 设置内搜索暂未发挥明显作用，先隐藏
        <div v-if="!panel" class="settings-search-bar">
            <el-input
                v-model="settingsSearch"
                clearable
                size="small"
                placeholder="搜索设置（标签 / 说明）…"
            />
        </div>
        -->
        <div v-if="!panel" class="settings-subnav">
            <button v-for="tab in settingsTabs" :key="tab.id" type="button" class="subnav-item"
                :class="{ active: settingsTab === tab.id }" @click="settingsTab = tab.id">
                {{ tab.label }}
            </button>
        </div>
        <!--
        <div v-else class="settings-search-bar">
            <el-input
                v-model="settingsSearch"
                clearable
                size="small"
                placeholder="搜索设置（标签 / 说明）…"
            />
        </div>
        -->

        <div ref="panelScrollRef" class="panel-scroll">
            <!-- 系统设置 -->
            <section v-show="settingsTab === 'system'" class="settings-section system-section">
                <div class="system-editor">
                    <div v-if="!panel" class="theme-subnav">
                        <button v-for="tab in systemPanels" :key="tab.id" type="button" class="theme-subnav-item"
                            :class="{ active: systemPanel === tab.id }" @click="systemPanel = tab.id">
                            {{ tab.label }}
                        </button>
                    </div>

                    <div class="appear-pane">
                        <!-- 应用信息 -->
                        <div v-show="systemPanel === 'app'" class="appear-pane-body settings-stack">
                            <div class="setting-block">
                                <h3 class="setting-block-title">应用</h3>
                                <div class="setting-card">
                            <div class="system-setting-row">
                                <div class="system-setting-text">
                                    <span class="system-setting-label">应用名称</span>
                                    <span class="system-setting-hint">窗口标题栏显示名称，留空则使用 FlashDock</span>
                                </div>
                                <div class="system-setting-control system-setting-control--wide">
                                    <el-input v-model="form.windowsName" class="system-setting-text-input" size="small"
                                        maxlength="64" placeholder="FlashDock" clearable />
                                </div>
                            </div>
                            <div class="system-setting-row">
                                <div class="system-setting-text">
                                    <span class="system-setting-label">启动时全屏</span>
                                    <span class="system-setting-hint">开启后下次启动最大化窗口；保存设置时也会立即切换当前窗口</span>
                                </div>
                                <div class="system-setting-control">
                                    <el-switch v-model="form.startupFullscreen" size="small" />
                                </div>
                            </div>
                                </div>
                            </div>
                            <div class="setting-block">
                                <h3 class="setting-block-title">Dock 图标</h3>
                                <div class="setting-card setting-card--padded">
                                <div class="system-setting-text" style="margin-bottom: 10px">
                                    <span class="system-setting-hint">选择预设或上传自定义图片（PNG / JPG），保存至
                                        ~/.flashdock/icons/；保存后立即更新窗口/任务栏图标</span>
                                </div>
                                <div class="dock-icon-presets">
                                    <button v-for="preset in appIconPresets" :key="preset.id" type="button"
                                        class="dock-icon-card"
                                        :class="{ active: form.appIconPreset === preset.id }" :title="preset.label"
                                        @click="form.appIconPreset = preset.id">
                                        <img :src="preset.preview" :alt="preset.label" class="dock-icon-img" />
                                        <span class="dock-icon-name">{{ preset.label }}</span>
                                    </button>
                                    <button type="button" class="dock-icon-card dock-icon-card--upload" title="上传自定义图标"
                                        :disabled="uploadingAppIcon" @click="uploadCustomAppIcon">
                                        <span class="dock-icon-upload-plus">+</span>
                                        <span class="dock-icon-name">{{ uploadingAppIcon ? '上传中…' : '上传' }}</span>
                                    </button>
                                </div>
                                </div>
                            </div>
                        </div>

                        <!-- Shell / 终端 + SFTP -->
                        <div v-show="systemPanel === 'shell'" class="appear-pane-body settings-stack">
                            <div v-show="shellMode !== 'sftp'" class="setting-block">
                                <h3 class="setting-block-title">连接与缓冲</h3>
                                <div class="setting-card">
                            <div class="system-setting-row">
                                <div class="system-setting-text">
                                    <span class="system-setting-label">SSH 握手超时</span>
                                    <span class="system-setting-hint">TCP 连接与 SSH 协商总超时，Shell 终端与任务远程执行共用，范围 5–300 秒</span>
                                </div>
                                <div class="system-setting-control">
                                    <el-input-number v-model="form.sshHandshakeTimeoutSec" class="system-setting-num"
                                        size="small" :min="5" :max="300" :step="5" controls-position="right" />
                                    <span class="system-setting-unit">秒</span>
                                </div>
                            </div>
                            <div class="system-setting-row">
                                <div class="system-setting-text">
                                    <span class="system-setting-label">监控同步间隔</span>
                                    <span class="system-setting-hint">侧边监控面板刷新频率，范围 200–60000 毫秒</span>
                                </div>
                                <div class="system-setting-control">
                                    <el-input-number v-model="form.shellMonitorIntervalMs" class="system-setting-num"
                                        size="small" :min="200" :max="60000" :step="100" controls-position="right" />
                                    <span class="system-setting-unit">毫秒</span>
                                </div>
                            </div>
                            <div class="system-setting-row">
                                <div class="system-setting-text">
                                    <span class="system-setting-label">终端滚动缓冲</span>
                                    <span class="system-setting-hint">Shell 终端可回滚行数，范围 100–100000；已打开的终端保存后立即生效</span>
                                </div>
                                <div class="system-setting-control">
                                    <el-input-number v-model="form.shellTerminalScrollback" class="system-setting-num"
                                        size="small" :min="100" :max="100000" :step="100" controls-position="right" />
                                    <span class="system-setting-unit">行</span>
                                </div>
                            </div>
                            <div class="system-setting-row">
                                <div class="system-setting-text">
                                    <span class="system-setting-label">任务输出上限</span>
                                    <span class="system-setting-hint">任务执行终端保留的最大行数，范围 100–100000</span>
                                </div>
                                <div class="system-setting-control">
                                    <el-input-number v-model="form.taskOutputMaxLines" class="system-setting-num"
                                        size="small" :min="100" :max="100000" :step="100" controls-position="right" />
                                    <span class="system-setting-unit">行</span>
                                </div>
                            </div>
                            <div class="system-setting-row">
                                <div class="system-setting-text">
                                    <span class="system-setting-label">命令历史上限</span>
                                    <span class="system-setting-hint">每个作用域（全局 / 单机）保留的命令条数，范围 50–20000</span>
                                </div>
                                <div class="system-setting-control">
                                    <el-input-number v-model="form.shellCommandHistoryMax" class="system-setting-num"
                                        size="small" :min="50" :max="20000" :step="50" controls-position="right" />
                                    <span class="system-setting-unit">条</span>
                                </div>
                            </div>
                                </div>
                            </div>
                            <div v-show="shellMode !== 'sftp'" class="setting-block">
                                <h3 class="setting-block-title">输入与显示</h3>
                                <div class="setting-card">
                            <div class="system-setting-row">
                                <div class="system-setting-text">
                                    <span class="system-setting-label">终端英文输入</span>
                                    <span class="system-setting-hint">Shell 终端获得焦点时临时关闭中文组词，失焦、切走窗口或离开 Shell 后自动恢复；仅影响终端，设置面板等中文输入不受影响</span>
                                </div>
                                <div class="system-setting-control">
                                    <el-switch v-model="form.shellAsciiInput" size="small" />
                                </div>
                            </div>
                            <div class="system-setting-row">
                                <div class="system-setting-text">
                                    <span class="system-setting-label">光标行高亮</span>
                                    <span class="system-setting-hint">高亮当前光标所在行；已打开的终端保存后立即生效</span>
                                </div>
                                <div class="system-setting-control">
                                    <el-switch v-model="form.shellCursorLineHighlight" size="small" />
                                </div>
                            </div>
                            <div class="system-setting-row">
                                <div class="system-setting-text">
                                    <span class="system-setting-label">行时间戳</span>
                                    <span class="system-setting-hint">在新输出行前缀显示本地时间（HH:MM:SS）；TUI 全屏程序下自动跳过</span>
                                </div>
                                <div class="system-setting-control">
                                    <el-switch v-model="form.shellLineTimestamps" size="small" />
                                </div>
                            </div>
                            <div class="system-setting-row">
                                <div class="system-setting-text">
                                    <span class="system-setting-label">密码提示辅助</span>
                                    <span class="system-setting-hint">检测到 Password: / 密码 提示时，在终端底部显示密码输入条（不记日志）；默认开启</span>
                                </div>
                                <div class="system-setting-control">
                                    <el-switch v-model="form.shellPasswordAssist" size="small" />
                                </div>
                            </div>
                                </div>
                            </div>
                            <div v-show="shellMode !== 'terminal'" class="setting-block">
                                <h3 class="setting-block-title">SFTP</h3>
                                <div class="setting-card">
                            <div class="system-setting-row">
                                <div class="system-setting-text">
                                    <span class="system-setting-label">SFTP 目录压缩上传</span>
                                    <span class="system-setting-hint">目录默认先打 zip 再远端解压；关闭则逐文件上传。文件面板可粘贴剪贴板图片直接上传</span>
                                </div>
                                <div class="system-setting-control">
                                    <el-switch v-model="form.sftpUseCompressedUpload" size="small" />
                                </div>
                            </div>
                            <div class="system-setting-row system-setting-row--stack">
                                <div class="system-setting-text">
                                    <span class="system-setting-label">外置编辑器</span>
                                    <span class="system-setting-hint">SFTP「用系统应用打开」优先用此命令；可用 {path} 占位，留空则系统默认</span>
                                </div>
                                <el-input v-model="form.externalEditorCommand" size="small" placeholder='例如 code "{path}" 或 notepad++' clearable />
                            </div>
                            <div class="system-setting-row system-setting-row--stack">
                                <div class="system-setting-text">
                                    <span class="system-setting-label">文件关联</span>
                                    <span class="system-setting-hint">每行：扩展名=命令，如 .go=code {path}</span>
                                </div>
                                <el-input
                                    v-model="fileAssocText"
                                    type="textarea"
                                    :rows="3"
                                    size="small"
                                    placeholder=".md=code {path}&#10;.log=notepad"
                                />
                            </div>
                                </div>
                            </div>
                        </div>

                        <!-- 主机密钥 -->
                        <div v-show="systemPanel === 'security'" class="appear-pane-body settings-stack">
                            <div class="setting-block">
                                <h3 class="setting-block-title">已信任主机</h3>
                                <div class="setting-card setting-card--padded">
                                <div class="system-setting-text" style="margin-bottom: 10px">
                                    <span class="system-setting-hint">连接 SSH 时校验的主机指纹；可从系统 ~/.ssh/known_hosts 合并导入</span>
                                </div>
                                <div class="known-hosts-toolbar icon-actions">
                                    <el-button size="small" type="primary" :loading="importingKnownHosts" @click="importSystemKnownHosts">
                                        导入系统 known_hosts
                                    </el-button>
                                    <el-button size="small" text :loading="loadingKnownHosts" @click="loadKnownHosts">
                                        刷新
                                    </el-button>
                                </div>
                                <el-table :data="knownHosts" size="small" style="width: 100%; margin-top: 8px" empty-text="暂无已信任主机" max-height="320">
                                    <el-table-column prop="host" label="主机" min-width="120" show-overflow-tooltip />
                                    <el-table-column prop="port" label="端口" width="70" />
                                    <el-table-column prop="fingerprint" label="指纹" min-width="200" show-overflow-tooltip />
                                    <el-table-column label="操作" width="72" align="center">
                                        <template #default="{ row }">
                                            <el-button size="small" text type="danger" @click="removeKnownHost(row)">
                                                删除
                                            </el-button>
                                        </template>
                                    </el-table-column>
                                </el-table>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </section>

            <!-- 账号 / 密钥库 -->
            <section v-show="settingsTab === 'accounts'" class="settings-section">
                <div class="settings-stack">
                <div class="setting-block">
                <div class="section-head">
                    <div>
                        <h4>密钥库 / 全局身份</h4>
                        <p class="section-desc">可复用的用户名、密码与私钥；机器通过「身份」引用。添加机器时可一键填充</p>
                    </div>
                    <el-tooltip content="添加帐号" placement="top">
                        <el-button size="small" type="primary" circle @click="addAccount">
                            <el-icon>
                                <Plus />
                            </el-icon>
                        </el-button>
                    </el-tooltip>
                </div>
                <div class="setting-card setting-card--padded">
                <el-table :data="accounts" size="small" style="width: 100%" empty-text="暂无帐号">
                    <el-table-column prop="name" label="帐号名称" width="160" />
                    <el-table-column prop="user" label="用户名" width="140" />
                    <el-table-column label="密码">
                        <template #default="scope">
                            {{ scope.row.password ? '******' : '未设置' }}
                        </template>
                    </el-table-column>
                    <el-table-column label="操作" width="100" align="center">
                        <template #default="scope">
                            <div class="icon-actions">
                                <el-tooltip content="编辑" placement="top">
                                    <el-button size="small" text type="primary" @click="editAccount(scope.$index)">
                                        <el-icon>
                                            <Edit />
                                        </el-icon>
                                    </el-button>
                                </el-tooltip>
                                <el-tooltip content="删除" placement="top">
                                    <el-button size="small" text type="danger" @click="removeAccount(scope.$index)">
                                        <el-icon>
                                            <Delete />
                                        </el-icon>
                                    </el-button>
                                </el-tooltip>
                            </div>
                        </template>
                    </el-table-column>
                </el-table>
                </div>
                </div>
                </div>
            </section>

            <!-- 主题 -->
            <section v-show="settingsTab === 'theme'" class="settings-section appearance-section">
                <div class="appear-layout">
                    <div class="appear-editor">
                        <div class="theme-subnav">
                            <button v-for="tab in themePanels" :key="tab.id" type="button" class="theme-subnav-item"
                                :class="{ active: themePanel === tab.id }" @click="themePanel = tab.id">
                                {{ tab.label }}
                            </button>
                        </div>

                        <div class="appear-pane">
                            <!-- 界面 -->
                            <div v-show="themePanel === 'ui'" class="appear-pane-body settings-stack">
                                <div class="setting-block">
                                    <h3 class="setting-block-title">界面</h3>
                                    <div class="setting-card setting-card--padded">
                                <div class="appear-field">
                                    <span class="appear-field-label">外观模式</span>
                                    <el-radio-group v-model="form.themeSettings.mode" size="small">
                                        <el-radio-button label="light">浅色</el-radio-button>
                                        <el-radio-button label="dark">深色</el-radio-button>
                                        <el-radio-button label="system">跟随系统</el-radio-button>
                                    </el-radio-group>
                                </div>
                                <div class="appear-field">
                                    <span class="appear-field-label">强调色</span>
                                    <div class="preset-grid accent-grid">
                                        <button v-for="accent in uiAccents" :key="accent.id" type="button"
                                            class="accent-swatch"
                                            :class="{ active: form.themeSettings.uiAccent === accent.id }"
                                            :title="accent.label" :style="{ background: accent.light.accent }"
                                            @click="form.themeSettings.uiAccent = accent.id"></button>
                                        <div class="accent-custom" :class="{ active: isCustomAccentActive }"
                                            title="自定义">
                                            <el-color-picker :model-value="customAccentColor" size="small"
                                                color-format="hex" :predefine="accentPredefine"
                                                @update:model-value="setCustomAccent" />
                                        </div>
                                    </div>
                                </div>
                                <div class="appear-field">
                                    <span class="appear-field-label">界面字体 / 字号</span>
                                    <div class="term-font-row ui-font-row">
                                        <el-select v-model="form.themeSettings.uiFontFamily" size="small"
                                            placeholder="界面字体" filterable class="term-font-select">
                                            <el-option v-for="font in uiFonts" :key="font.id" :label="font.label"
                                                :value="font.id" />
                                        </el-select>
                                        <el-input-number v-model="form.themeSettings.uiFontSize" class="term-num"
                                            size="small" :min="12" :max="20" :step="1" controls-position="right" />
                                    </div>
                                    <div class="term-font-hints ui-font-hints">
                                        <span>字体</span>
                                        <span>字号</span>
                                    </div>
                                </div>
                                    </div>
                                </div>
                            </div>

                            <!-- 终端 -->
                            <div v-show="themePanel === 'terminal'" class="appear-pane-body settings-stack">
                                <div class="setting-block">
                                    <h3 class="setting-block-title">字体与渲染</h3>
                                    <div class="setting-card setting-card--padded">
                                <div class="appear-field">
                                    <span class="appear-field-label">字体 / 字号 / 行高</span>
                                    <div class="term-font-row">
                                        <el-select v-model="form.themeSettings.shellFontFamily" size="small"
                                            placeholder="终端字体" filterable class="term-font-select">
                                            <el-option v-for="font in terminalFonts" :key="font.id" :label="font.label"
                                                :value="font.id" />
                                        </el-select>
                                        <el-input-number v-model="form.themeSettings.shellFontSize" class="term-num"
                                            size="small" :min="8" :max="32" :step="1" controls-position="right" />
                                        <el-input-number v-model="form.themeSettings.shellLineHeight" class="term-num"
                                            size="small" :min="1" :max="2.5" :step="0.1" :precision="1"
                                            controls-position="right" />
                                    </div>
                                    <div class="term-font-hints">
                                        <span>字体</span>
                                        <span>字号 (Ctrl+=/-/0)</span>
                                        <span>行高</span>
                                    </div>
                                </div>
                                <div class="appear-field memory-saver-row">
                                    <span class="memory-saver-label">离开 Shell 时卸载终端界面（省内存，会话保持）</span>
                                    <el-switch v-model="form.themeSettings.shellMemorySaver" size="small" />
                                </div>
                                <div class="appear-field memory-saver-row">
                                    <span class="memory-saver-label">非活动标签休眠（停刷新并释放终端画面，切回时回放缓冲）</span>
                                    <el-switch v-model="form.themeSettings.shellTabHibernate" size="small" />
                                </div>
                                <div class="appear-field memory-saver-row">
                                    <span class="memory-saver-label">使用 WebGL 加速渲染（失败时自动回退）</span>
                                    <el-switch v-model="form.themeSettings.shellUseWebgl" size="small" />
                                </div>
                                <div class="appear-field memory-saver-row">
                                    <span class="memory-saver-label">Shell 意外断开时自动重连</span>
                                    <el-switch v-model="form.themeSettings.shellAutoReconnect" size="small" />
                                </div>
                                    </div>
                                </div>
                                <div class="setting-block">
                                    <h3 class="setting-block-title">配色方案</h3>
                                    <div class="setting-card setting-card--padded">
                                <div class="appear-field appear-field--fill">
                                    <span class="appear-field-label">配色方案</span>
                                    <div class="preset-grid terminal-grid">
                                        <button v-for="preset in terminalPresets" :key="preset.id" type="button"
                                            class="term-card"
                                            :class="{ active: form.themeSettings.terminalPreset === preset.id }"
                                            :title="preset.label"
                                            @click="form.themeSettings.terminalPreset = preset.id">
                                            <span class="term-card-preview"
                                                :style="{ background: preset.theme.background, color: preset.theme.foreground }">
                                                <span class="term-card-dots">
                                                    <i :style="{ background: preset.theme.red }"></i>
                                                    <i :style="{ background: preset.theme.green }"></i>
                                                    <i
                                                        :style="{ background: preset.theme.blue || preset.theme.cursor }"></i>
                                                </span>
                                                <span class="term-card-name">{{ preset.label }}</span>
                                                <code>~/</code>
                                            </span>
                                        </button>
                                    </div>
                                </div>
                                    </div>
                                </div>
                            </div>

                            <!-- 日志高亮 -->
                            <div v-show="themePanel === 'log'" class="appear-pane-body settings-stack">
                                <div class="setting-block">
                                    <h3 class="setting-block-title">日志高亮</h3>
                                    <div class="setting-card setting-card--padded">
                                <div class="appear-field memory-saver-row">
                                    <div>
                                        <span class="appear-field-label">启用日志高亮</span>
                                        <p class="log-hl-colors-hint">
                                            对输出中的时间戳 / 级别 / SQL / 自定义关键字着色，不区分命令（tail、grep、less 等整行输出均可）
                                        </p>
                                    </div>
                                    <el-switch v-model="form.shellLogHighlight" size="small" />
                                </div>
                                <template v-if="form.shellLogHighlight">
                                    <div class="appear-field">
                                        <div class="log-hl-colors-head">
                                            <span class="appear-field-label">配色方案</span>
                                            <el-button size="small" text type="primary"
                                                @click="resetLogHighlightConfig">
                                                恢复默认
                                            </el-button>
                                        </div>
                                        <div class="log-hl-preset-grid">
                                            <button v-for="preset in logColorPresets" :key="preset.id" type="button"
                                                class="log-hl-preset-card"
                                                :class="{ active: activeLogHighlightPreset === preset.id }"
                                                :title="preset.label" @click="applyLogHighlightPreset(preset.id)">
                                                <span class="log-hl-preset-dots" aria-hidden="true">
                                                    <i :style="{ background: preset.colors.error }"></i>
                                                    <i :style="{ background: preset.colors.warn }"></i>
                                                    <i :style="{ background: preset.colors.info }"></i>
                                                    <i :style="{ background: preset.colors.timestamp }"></i>
                                                </span>
                                                <span class="log-hl-preset-name">{{ preset.label }}</span>
                                            </button>
                                            <button type="button" class="log-hl-preset-card is-custom"
                                                :class="{ active: activeLogHighlightPreset === 'custom' }"
                                                title="当前为自定义配色" disabled>
                                                <span class="log-hl-preset-dots" aria-hidden="true">
                                                    <i v-for="key in logHlDotKeys" :key="key"
                                                        :style="{ background: form.shellLogHighlightColors[key] }"></i>
                                                </span>
                                                <span class="log-hl-preset-name">自定义</span>
                                            </button>
                                        </div>
                                    </div>
                                    <div class="appear-field">
                                        <span class="appear-field-label">单项颜色</span>
                                        <div class="log-hl-colors-grid">
                                            <div v-for="item in logColorItems" :key="item.key" class="log-hl-color-row">
                                                <span class="log-hl-color-label">{{ item.label }}</span>
                                                <div class="log-hl-color-actions">
                                                    <el-switch v-model="form.shellLogHighlightRules[item.key]"
                                                        size="small" />
                                                    <el-color-picker v-model="form.shellLogHighlightColors[item.key]"
                                                        size="small" color-format="hex" :predefine="logColorPredefine"
                                                        :disabled="!form.shellLogHighlightRules[item.key]" />
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                    <div class="appear-field">
                                        <div class="log-hl-colors-head">
                                            <span class="appear-field-label">自定义关键字</span>
                                            <el-button size="small" text type="primary" @click="addCustomKeyword">
                                                添加
                                            </el-button>
                                        </div>
                                        <p class="log-hl-colors-hint">命中文本即着色，最多 64 条；适合业务码、TraceId 等</p>
                                        <div v-if="!form.shellLogHighlightKeywords.length" class="log-hl-kw-empty">
                                            暂无自定义关键字
                                        </div>
                                        <div v-else class="log-hl-kw-list">
                                            <div
                                                v-for="(item, idx) in form.shellLogHighlightKeywords"
                                                :key="'kw-' + idx"
                                                class="log-hl-kw-row"
                                            >
                                                <el-switch v-model="item.enabled" size="small" />
                                                <el-input
                                                    v-model="item.keyword"
                                                    size="small"
                                                    placeholder="关键字"
                                                    maxlength="64"
                                                    class="log-hl-kw-input"
                                                />
                                                <el-color-picker
                                                    v-model="item.color"
                                                    size="small"
                                                    color-format="hex"
                                                    :predefine="logColorPredefine"
                                                    :disabled="!item.enabled"
                                                />
                                                <el-button size="small" text type="danger" @click="removeCustomKeyword(idx)">
                                                    删除
                                                </el-button>
                                            </div>
                                        </div>
                                    </div>
                                </template>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <aside class="appear-preview">
                        <div class="block-label">实时预览</div>
                        <div class="theme-preview" :class="{ dark: previewIsDark }">
                            <div v-show="themePanel === 'ui'" class="preview-ui" :style="previewUiStyle">
                                <div class="preview-bar">
                                    <span class="preview-dot"></span>
                                    <span class="preview-title">FlashDock</span>
                                    <span class="preview-pill">{{ previewUiFontLabel }}</span>
                                </div>
                                <div class="preview-body">
                                    <div class="preview-card">
                                        <div class="preview-card-title">界面卡片预览</div>
                                        <p class="preview-card-text">
                                            任务流水线 · Shell 终端 · 系统设置
                                        </p>
                                        <p class="preview-card-sample">
                                            Aa 字号预览 123
                                        </p>
                                    </div>
                                    <button type="button" class="preview-btn">主按钮</button>
                                </div>
                            </div>
                            <div v-show="themePanel === 'terminal'" class="preview-term" :style="previewTermStyle">
                                <div class="preview-term-title">{{ previewTermLabel }}</div>
                                <pre>{{ previewTermSample }}</pre>
                            </div>
                            <div v-show="themePanel === 'log'" class="preview-log" :style="previewTermStyle">
                                <div class="preview-term-title">
                                    {{ form.shellLogHighlight ? '日志高亮预览' : '日志高亮已关闭' }}
                                </div>
                                <pre v-if="form.shellLogHighlight" class="log-hl-preview-line preview-log-line"><span
                            v-for="(part, idx) in logHighlightPreviewParts" :key="idx"
                            :style="part.color ? { color: part.color } : null">{{ part.text }}</span></pre>
                                <p v-else class="preview-log-off">开启后将在此显示着色效果</p>
                            </div>
                        </div>
                    </aside>
                </div>
            </section>

            <!-- 关于 -->
            <section v-show="settingsTab === 'about'" class="settings-section about-section">
                <div class="settings-stack">
                    <div class="setting-block">
                        <h3 class="setting-block-title">版本信息</h3>
                        <div class="setting-card">
                            <div class="system-setting-row">
                                <div class="system-setting-text">
                                    <span class="system-setting-label">当前版本</span>
                                </div>
                                <div class="system-setting-control">
                                    <span class="about-meta-value">{{ appVersion || '—' }}</span>
                                </div>
                            </div>
                            <div class="system-setting-row">
                                <div class="system-setting-text">
                                    <span class="system-setting-label">最新 Release</span>
                                </div>
                                <div class="system-setting-control">
                                    <span class="about-meta-value">
                                        {{ updateResult?.latestVersion || (checkingUpdate ? '检查中…' : '—') }}
                                    </span>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="setting-block">
                        <h3 class="setting-block-title">更新</h3>
                        <div class="setting-card setting-card--padded" v-loading="checkingUpdate">
                            <div class="about-update-actions">
                                <el-button size="small" :loading="checkingUpdate" @click="() => checkUpdate(true)">
                                    检查更新
                                </el-button>
                                <el-button v-if="updateResult?.releaseURL" size="small" @click="openRelease">
                                    查看 Release
                                </el-button>
                            </div>

                            <div v-if="updateResult?.hasUpdate" class="update-banner">
                                <div class="update-banner-title">发现新版本 {{ updateResult.latestVersion }}</div>
                                <div class="update-banner-sub">当前 {{ updateResult.currentVersion || appVersion }}</div>
                                <div v-if="updateResult.assetName" class="asset-line">
                                    适配安装包：{{ updateResult.assetName }}
                                </div>
                                <div class="update-actions">
                                    <el-select v-model="selectedDownloadSource" size="small" class="source-select"
                                        :disabled="downloading || installing" placeholder="下载源">
                                        <el-option v-for="src in downloadSources" :key="src.label" :label="src.label"
                                            :value="src.label" />
                                    </el-select>
                                    <el-button type="primary" size="small" :loading="downloading" :disabled="!canDownload"
                                        @click="downloadUpdate">
                                        {{ downloadButtonLabel }}
                                    </el-button>
                                    <el-button v-if="readyToInstall" type="success" size="small" :loading="installing"
                                        :disabled="downloading" @click="installUpdate">
                                        安装并重启
                                    </el-button>
                                    <el-button v-if="readyToInstall" size="small" :disabled="downloading || installing"
                                        @click="openPackage">
                                        打开安装包
                                    </el-button>
                                    <el-button v-if="downloading" size="small" @click="pauseDownload">
                                        暂停
                                    </el-button>
                                </div>
                                <el-progress v-if="downloading || downloadPercent > 0 || downloadPaused"
                                    :percentage="downloadPercent" :stroke-width="10" style="margin-top: 10px" />
                                <div v-if="downloadMessage" class="download-msg"
                                    :class="{ err: downloadFailed, paused: downloadPaused }">
                                    {{ downloadMessage }}
                                </div>
                            </div>
                            <div v-else-if="updateResult" class="update-ok">
                                已是最新版本
                            </div>

                            <div v-if="updateResult?.releaseNotes" class="release-section">
                                <div class="release-section-title">
                                    <span>{{ updateResult.hasUpdate ? '更新说明' : '最新 Release' }}</span>
                                </div>
                                <div class="update-notes" v-html="renderReleaseNotes(updateResult.releaseNotes)"
                                    @click="onNotesClick">
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </section>
        </div>

        <div v-if="settingsTab !== 'about' && settingsTab !== 'accounts'" class="panel-actions icon-actions">
            <el-tooltip content="保存设置" placement="top">
                <el-button type="primary" circle :loading="saving" @click="save">
                    <el-icon v-if="!saving">
                        <Check />
                    </el-icon>
                </el-button>
            </el-tooltip>
        </div>

        <el-dialog v-model="accountEditVisible" :title="editingAccountIndex >= 0 ? '编辑帐号' : '添加帐号'" width="480px"
            class="settings-sub-dialog" append-to-body>
            <el-form :model="accountForm" label-width="90px">
                <el-form-item label="帐号名称">
                    <el-input v-model="accountForm.name" placeholder="例如：生产环境" />
                </el-form-item>
                <el-form-item label="用户名">
                    <el-input v-model="accountForm.user" placeholder="SSH 用户名" />
                </el-form-item>
                <el-form-item label="密码">
                    <el-input v-model="accountForm.password" type="password" show-password placeholder="SSH 密码" />
                </el-form-item>
                <el-form-item label="密钥文件">
                    <div class="key-file-input">
                        <el-input v-model="accountForm.keyFile" placeholder="私钥路径" readonly />
                        <el-button type="primary" circle @click="selectAccountKeyFile">
                            <el-icon><Folder /></el-icon>
                        </el-button>
                    </div>
                </el-form-item>
                <el-form-item label="密钥口令">
                    <el-input v-model="accountForm.keyPassphrase" type="password" show-password placeholder="加密私钥口令（可选）" />
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer icon-actions">
                    <el-tooltip content="取消" placement="top">
                        <el-button circle @click="accountEditVisible = false">
                            <el-icon>
                                <Close />
                            </el-icon>
                        </el-button>
                    </el-tooltip>
                    <el-tooltip content="确定" placement="top">
                        <el-button type="primary" circle :loading="savingAccount" @click="confirmAccount">
                            <el-icon v-if="!savingAccount">
                                <Check />
                            </el-icon>
                        </el-button>
                    </el-tooltip>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script>
import { ref, reactive, watch, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { marked } from 'marked'
import { ElMessage } from 'element-plus'
import { Edit, Delete, Plus, Close, Check, Folder } from '@element-plus/icons-vue'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { useTheme } from '../composables/useTheme'
import {
    getCachedUpdateCheck,
    setCachedUpdateCheck,
    isUsableUpdateResult,
} from '../utils/updateCheckCache'
import { resolveUpdateDownloadSources } from '../utils/updateDownloadSources'
import {
    UI_ACCENTS,
    TERMINAL_PRESETS,
    UI_FONTS,
    TERMINAL_FONTS,
    getUiAccent,
    resolveAccentBg,
    getUiFont,
    getTerminalFont,
    getTerminalPreset,
    mergeUiFontOptions,
    mergeTerminalFontOptions,
    isCustomUiAccent,
    collectUiAccentPredefineColors,
} from '../utils/themePresets'
import {
    DEFAULT_SHELL_LOG_COLORS,
    DEFAULT_CUSTOM_KEYWORD_COLOR,
    SHELL_LOG_COLOR_PRESETS,
    mergeLogHighlightColors,
    mergeLogHighlightRules,
    normalizeCustomKeywords,
    rulesToDisabled,
    logHighlightPreviewSegments,
    getLogHighlightPreset,
    matchLogHighlightPreset,
    collectLogHighlightPredefineColors,
} from '../utils/shellLogHighlight'
import {
    SHELL_TERMINAL_SCROLLBACK,
    TASK_OUTPUT_MAX_LINES,
    SHELL_COMMAND_HISTORY_MAX,
    clampShellTerminalScrollback,
    clampTaskOutputMaxLines,
    clampShellCommandHistoryMax,
} from '../constants/shellMemory'

const LOG_HIGHLIGHT_SAMPLE =
    '2024-07-16 14:40:35.719 9992000288484 INFO o.g.f.c.d.r.Framework - Preparing: SELECT id FROM t'

marked.setOptions({
    breaks: true,
    gfm: true,
})

export default {
    name: 'SystemSettingsDialog',
    components: { Edit, Delete, Plus, Close, Check },
    props: {
        modelValue: { type: Boolean, default: false },
        embedded: { type: Boolean, default: false },
        active: { type: Boolean, default: false },
        /** Hub 侧栏路由：app | terminal | sftp | security | accounts | appearance | about | general */
        panel: { type: String, default: '' },
    },
    emits: ['update:modelValue', 'saved'],
    setup(props, { emit }) {
        const saving = ref(false)
        const savingAccount = ref(false)
        const settingsSearch = ref('')
        const panelScrollRef = ref(null)
        const sessionId = ref('')
        const appVersion = ref('')
        const checkingUpdate = ref(false)
        const updateResult = ref(null)
        const downloading = ref(false)
        const installing = ref(false)
        const readyToInstall = ref(false)
        const downloadPaused = ref(false)
        const downloadPercent = ref(0)
        const downloadMessage = ref('')
        const downloadFailed = ref(false)
        const selectedDownloadSource = ref('GitHub')
        const accounts = ref([])
        const knownHosts = ref([])
        const loadingKnownHosts = ref(false)
        const importingKnownHosts = ref(false)
        const accountEditVisible = ref(false)
        const editingAccountIndex = ref(-1)
        const accountForm = reactive({ id: '', name: '', user: '', password: '', keyFile: '', keyPassphrase: '' })
        const fileAssocText = ref('')
        const { applyThemeSettings } = useTheme()
        const settingsTab = ref('system')
        const settingsTabs = [
            { id: 'system', label: '系统设置' },
            { id: 'accounts', label: '密钥库' },
            { id: 'theme', label: '主题' },
            { id: 'about', label: '关于' },
        ]
        const systemPanel = ref('app')
        const systemPanels = [
            { id: 'app', label: '应用信息' },
            { id: 'shell', label: 'Shell' },
            { id: 'security', label: '主机密钥' },
        ]
        /** all | terminal | sftp — Hub 分栏时拆开终端与 SFTP */
        const shellMode = ref('all')
        const themePanel = ref('ui')
        const themePanels = [
            { id: 'ui', label: '界面' },
            { id: 'terminal', label: '终端' },
            { id: 'log', label: '日志高亮' },
        ]

        const applyPanelRoute = (panelId) => {
            const id = String(panelId || '').trim()
            if (!id) {
                shellMode.value = 'all'
                return
            }
            if (id === 'general' || id === 'app') {
                settingsTab.value = 'system'
                systemPanel.value = 'app'
                shellMode.value = 'all'
                return
            }
            if (id === 'terminal') {
                settingsTab.value = 'system'
                systemPanel.value = 'shell'
                shellMode.value = 'terminal'
                return
            }
            if (id === 'sftp') {
                settingsTab.value = 'system'
                systemPanel.value = 'shell'
                shellMode.value = 'sftp'
                return
            }
            if (id === 'security') {
                settingsTab.value = 'system'
                systemPanel.value = 'security'
                shellMode.value = 'all'
                return
            }
            if (id === 'accounts') {
                settingsTab.value = 'accounts'
                shellMode.value = 'all'
                return
            }
            if (id === 'appearance' || id === 'theme') {
                settingsTab.value = 'theme'
                shellMode.value = 'all'
                return
            }
            if (id === 'about') {
                settingsTab.value = 'about'
                shellMode.value = 'all'
            }
        }

        watch(() => props.panel, (p) => applyPanelRoute(p), { immediate: true })
        const form = reactive({
            windowsName: 'FlashDock',
            appIconPreset: 'default',
            startupFullscreen: false,
            themeSettings: {
                mode: 'light',
                uiAccent: 'blue',
                terminalPreset: 'classic',
                uiFontFamily: 'system',
                uiFontSize: 14,
                shellFontFamily: 'consolas',
                shellFontSize: 13,
                shellLineHeight: 1.2,
                shellMemorySaver: false,
                shellAutoReconnect: false,
                shellUseWebgl: false,
                shellTabHibernate: true,
            },
            shellMonitorIntervalMs: 1000,
            sshHandshakeTimeoutSec: 30,
            shellTerminalScrollback: SHELL_TERMINAL_SCROLLBACK,
            taskOutputMaxLines: TASK_OUTPUT_MAX_LINES,
            shellCommandHistoryMax: SHELL_COMMAND_HISTORY_MAX,
            shellLogHighlight: true,
            shellAsciiInput: true,
            shellCursorLineHighlight: false,
            shellLineTimestamps: false,
            shellPasswordAssist: true,
            sftpUseCompressedUpload: true,
            externalEditorCommand: '',
            shellLogHighlightColors: { ...DEFAULT_SHELL_LOG_COLORS },
            shellLogHighlightRules: mergeLogHighlightRules([]),
            shellLogHighlightKeywords: [],
        })

        const appIconPresets = ref([])
        const uploadingAppIcon = ref(false)

        let cachedAppIconPresets = null

        const loadAppIconPresets = async (force = false) => {
            if (!force && cachedAppIconPresets?.length) {
                appIconPresets.value = cachedAppIconPresets
                return
            }
            try {
                const list = (await App.ListAppIconPresets()) || []
                cachedAppIconPresets = list
                appIconPresets.value = list
            } catch {
                if (!appIconPresets.value.length) appIconPresets.value = []
            }
        }

        const uploadCustomAppIcon = async () => {
            uploadingAppIcon.value = true
            try {
                const id = await App.PickCustomAppIcon()
                if (!id) return
                form.appIconPreset = id
                await loadAppIconPresets(true)
                ElMessage.success('自定义图标已保存到 ~/.flashdock/icons/')
            } catch (e) {
                ElMessage.error(`上传失败: ${e}`)
            } finally {
                uploadingAppIcon.value = false
            }
        }

        const logColorItems = [
            { key: 'timestamp', label: '时间戳' },
            { key: 'threadId', label: '线程号' },
            { key: 'info', label: 'INFO' },
            { key: 'debug', label: 'DEBUG' },
            { key: 'warn', label: 'WARN' },
            { key: 'error', label: 'ERROR' },
            { key: 'logger', label: 'Logger' },
            { key: 'sql', label: 'SQL 关键字' },
            { key: 'label', label: 'SQL 标签' },
        ]
        const logColorPresets = SHELL_LOG_COLOR_PRESETS
        const logHlDotKeys = ['error', 'warn', 'info', 'timestamp']
        const logColorPredefine = collectLogHighlightPredefineColors()

        const activeLogHighlightPreset = computed(() =>
            matchLogHighlightPreset(form.shellLogHighlightColors),
        )

        const logHighlightPreviewParts = computed(() =>
            logHighlightPreviewSegments(
                LOG_HIGHLIGHT_SAMPLE,
                form.shellLogHighlightColors,
                form.shellLogHighlightRules,
                form.shellLogHighlightKeywords,
            ),
        )

        const applyLogHighlightPreset = (id) => {
            const preset = getLogHighlightPreset(id)
            Object.assign(form.shellLogHighlightColors, mergeLogHighlightColors(preset.colors))
        }

        const resetLogHighlightConfig = () => {
            applyLogHighlightPreset('windterm')
            Object.assign(form.shellLogHighlightRules, mergeLogHighlightRules([]))
            form.shellLogHighlightKeywords.splice(0, form.shellLogHighlightKeywords.length)
        }

        const addCustomKeyword = () => {
            if (form.shellLogHighlightKeywords.length >= 64) {
                ElMessage.warning('最多添加 64 条自定义关键字')
                return
            }
            form.shellLogHighlightKeywords.push({
                keyword: '',
                color: DEFAULT_CUSTOM_KEYWORD_COLOR,
                enabled: true,
            })
        }

        const removeCustomKeyword = (idx) => {
            if (idx < 0 || idx >= form.shellLogHighlightKeywords.length) return
            form.shellLogHighlightKeywords.splice(idx, 1)
        }

        const uiAccents = UI_ACCENTS
        const accentPredefine = collectUiAccentPredefineColors()
        const lastCustomAccent = ref('#ff4da6')
        const isCustomAccentActive = computed(() => isCustomUiAccent(form.themeSettings.uiAccent))
        const customAccentColor = computed(() =>
            isCustomAccentActive.value ? form.themeSettings.uiAccent : lastCustomAccent.value,
        )
        const setCustomAccent = (color) => {
            if (!color) return
            const hex = String(color).trim().toLowerCase()
            lastCustomAccent.value = hex
            form.themeSettings.uiAccent = hex
        }
        const terminalPresets = TERMINAL_PRESETS
        const uiFonts = ref([...UI_FONTS])
        const terminalFonts = ref([...TERMINAL_FONTS])
        const systemFontsLoaded = ref(false)

        const canDownload = computed(() =>
            !!(updateResult.value?.hasUpdate && updateResult.value?.downloadURL && !downloading.value && !installing.value)
        )

        const downloadSources = computed(() => resolveUpdateDownloadSources(updateResult.value))

        const downloadButtonLabel = computed(() => {
            if (downloading.value) return `下载中 ${downloadPercent.value}%`
            if (downloadPaused.value) return '继续下载'
            if (readyToInstall.value) return '重新下载'
            return '下载安装包'
        })

        watch(downloadSources, (list) => {
            if (!list.length) return
            if (!list.some((s) => s.label === selectedDownloadSource.value)) {
                selectedDownloadSource.value = list[0].label
            }
        })

        const renderReleaseNotes = (md) => {
            const text = String(md || '').trim()
            if (!text) return ''
            try {
                return marked.parse(text)
            } catch {
                return text
                    .replace(/&/g, '&amp;')
                    .replace(/</g, '&lt;')
                    .replace(/>/g, '&gt;')
                    .replace(/\n/g, '<br>')
            }
        }

        const previewIsDark = computed(() => {
            const mode = form.themeSettings.mode
            if (mode === 'dark') return true
            if (mode === 'light') return false
            return window.matchMedia('(prefers-color-scheme: dark)').matches
        })

        const previewUiStyle = computed(() => {
            const accent = getUiAccent(form.themeSettings.uiAccent)
            const palette = previewIsDark.value ? accent.dark : accent.light
            const font = getUiFont(form.themeSettings.uiFontFamily)
            const size = form.themeSettings.uiFontSize > 0 ? form.themeSettings.uiFontSize : 14
            return {
                '--preview-accent': palette.accent,
                '--preview-accent-bg': resolveAccentBg(palette.accent, previewIsDark.value, palette.accentBg),
                fontFamily: font.value,
                fontSize: `${size}px`,
                background: previewIsDark.value ? '#1d1e1f' : '#f5f7fa',
                color: previewIsDark.value ? '#e5eaf3' : '#303133',
                borderColor: previewIsDark.value ? '#414243' : '#e4e7ed',
            }
        })

        const previewTermStyle = computed(() => {
            const theme = getTerminalPreset(form.themeSettings.terminalPreset).theme
            const font = getTerminalFont(form.themeSettings.shellFontFamily)
            return {
                background: theme.background,
                color: theme.foreground,
                fontFamily: font.value,
                fontSize: `${form.themeSettings.shellFontSize || 13}px`,
                lineHeight: form.themeSettings.shellLineHeight || 1.2,
            }
        })

        const previewUiFontLabel = computed(() => {
            const font = getUiFont(form.themeSettings.uiFontFamily).label
            const size = form.themeSettings.uiFontSize > 0 ? form.themeSettings.uiFontSize : 14
            return `${font} · ${size}px`
        })
        const previewTermLabel = computed(() => getTerminalPreset(form.themeSettings.terminalPreset).label)
        const previewTermSample = computed(() => {
            const theme = getTerminalPreset(form.themeSettings.terminalPreset).theme
            return `user@host:~$ ls -la
drwxr-xr-x  12 user  staff  384 Jul 16 09:00 .
-rw-r--r--   1 user  staff  128 Jul 16 09:00 README.md
user@host:~$ echo "theme: ${getTerminalPreset(form.themeSettings.terminalPreset).label}"
theme preview · ${theme.foreground}`
        })

        const visibleProxy = computed({
            get: () => props.modelValue,
            set: (v) => emit('update:modelValue', v)
        })

        const load = async () => {
            const config = await App.GetSystemSettings()
            form.windowsName = (config.windowsName || 'FlashDock').trim() || 'FlashDock'
            form.appIconPreset = config.appIconPreset || 'default'
            form.startupFullscreen = !!config.startupFullscreen
            form.themeSettings = {
                mode: config.themeSettings?.mode || 'light',
                uiAccent: config.themeSettings?.uiAccent || 'blue',
                terminalPreset: config.themeSettings?.terminalPreset || 'classic',
                uiFontFamily: config.themeSettings?.uiFontFamily || 'system',
                uiFontSize: config.themeSettings?.uiFontSize > 0 ? config.themeSettings.uiFontSize : 14,
                shellFontFamily: config.themeSettings?.shellFontFamily || 'consolas',
                shellFontSize: config.themeSettings?.shellFontSize > 0 ? config.themeSettings.shellFontSize : 13,
                shellLineHeight: config.themeSettings?.shellLineHeight > 0 ? config.themeSettings.shellLineHeight : 1.2,
                shellMemorySaver: !!config.themeSettings?.shellMemorySaver,
                shellAutoReconnect: !!config.themeSettings?.shellAutoReconnect,
                shellUseWebgl: !!config.themeSettings?.shellUseWebgl,
                shellTabHibernate: config.themeSettings?.shellTabHibernate !== false,
            }
            if (isCustomUiAccent(form.themeSettings.uiAccent)) {
                lastCustomAccent.value = form.themeSettings.uiAccent
            }
            const interval = Number(config.shellMonitorIntervalMs)
            form.shellMonitorIntervalMs = Number.isFinite(interval) && interval >= 200
                ? Math.min(60000, Math.round(interval))
                : 1000
            const sshTimeout = Number(config.sshHandshakeTimeoutSec)
            form.sshHandshakeTimeoutSec = Number.isFinite(sshTimeout) && sshTimeout >= 5
                ? Math.min(300, Math.round(sshTimeout))
                : 30
            form.shellTerminalScrollback = clampShellTerminalScrollback(config.shellTerminalScrollback)
            form.taskOutputMaxLines = clampTaskOutputMaxLines(config.taskOutputMaxLines)
            form.shellCommandHistoryMax = clampShellCommandHistoryMax(config.shellCommandHistoryMax)
            form.shellLogHighlight = config.shellLogHighlight !== false
            form.shellAsciiInput = config.shellAsciiInput !== false
            form.shellCursorLineHighlight = !!config.shellCursorLineHighlight
            form.shellLineTimestamps = !!config.shellLineTimestamps
            form.shellPasswordAssist = config.shellPasswordAssist !== false
            form.sftpUseCompressedUpload = config.sftpUseCompressedUpload !== false
            form.externalEditorCommand = config.externalEditorCommand || ''
            const assoc = config.fileAssociations || {}
            fileAssocText.value = Object.entries(assoc)
                .map(([ext, cmd]) => `${ext}=${cmd}`)
                .join('\n')
            Object.assign(
                form.shellLogHighlightColors,
                mergeLogHighlightColors(config.shellLogHighlightColors),
            )
            Object.assign(
                form.shellLogHighlightRules,
                mergeLogHighlightRules(config.shellLogHighlightDisabled),
            )
            form.shellLogHighlightKeywords.splice(
                0,
                form.shellLogHighlightKeywords.length,
                ...normalizeCustomKeywords(config.shellLogHighlightKeywords),
            )
            accounts.value = await App.GetGlobalAccounts() || []
            const session = await App.GetSessionInfo()
            sessionId.value = session.sessionId || ''
            try {
                appVersion.value = await App.GetAppVersion() || ''
            } catch {
                appVersion.value = ''
            }
            await loadSystemFonts()
            // Dock 图标缩略图不阻塞设置页打开；有缓存时几乎瞬时
            loadAppIconPresets()
            await loadKnownHosts()
        }

        const loadKnownHosts = async () => {
            loadingKnownHosts.value = true
            try {
                knownHosts.value = (await App.GetKnownHosts()) || []
            } catch {
                knownHosts.value = []
            } finally {
                loadingKnownHosts.value = false
            }
        }

        const importSystemKnownHosts = async () => {
            importingKnownHosts.value = true
            try {
                const n = await App.ImportSystemKnownHosts()
                await loadKnownHosts()
                ElMessage.success(n > 0 ? `已导入 ${n} 条主机密钥` : '未导入新记录（文件为空或均为哈希主机名）')
            } catch (e) {
                ElMessage.error('导入失败: ' + e)
            } finally {
                importingKnownHosts.value = false
            }
        }

        const removeKnownHost = async (row) => {
            if (!row?.host) return
            try {
                await App.RemoveKnownHost(row.host, row.port || 22)
                await loadKnownHosts()
                ElMessage.success('已删除')
            } catch (e) {
                ElMessage.error('删除失败: ' + e)
            }
        }

        const loadSystemFonts = async () => {
            if (systemFontsLoaded.value) return
            try {
                const fonts = await App.ListSystemFonts()
                uiFonts.value = mergeUiFontOptions(fonts || [])
                terminalFonts.value = mergeTerminalFontOptions(fonts || [])
                systemFontsLoaded.value = true
            } catch {
                // 保留内置字体列表
            }
        }

        const applyUpdateResult = (result) => {
            if (isUsableUpdateResult(result)) {
                updateResult.value = result
                setCachedUpdateCheck(result)
                readyToInstall.value = !!(result.downloaded && result.downloadPath)
                if (readyToInstall.value && !downloadMessage.value) {
                    downloadMessage.value = `安装包已就绪：${result.downloadPath}`
                    downloadPercent.value = 100
                }
            } else {
                updateResult.value = null
                readyToInstall.value = false
            }
        }

        const checkUpdate = async (force = false) => {
            if (!force) {
                const hit = getCachedUpdateCheck()
                if (hit) {
                    applyUpdateResult(hit)
                    return
                }
            }
            checkingUpdate.value = true
            downloadPercent.value = 0
            downloadMessage.value = ''
            downloadFailed.value = false
            downloadPaused.value = false
            readyToInstall.value = false
            try {
                const result = await App.CheckForUpdates()
                applyUpdateResult(result)
            } catch {
                updateResult.value = null
                readyToInstall.value = false
            } finally {
                checkingUpdate.value = false
            }
        }

        const openRelease = () => {
            const url = updateResult.value?.releaseURL
            if (url) App.OpenReleaseURL(url)
        }

        const onNotesClick = (e) => {
            const a = e.target?.closest?.('a')
            if (!a) return
            const href = a.getAttribute('href') || ''
            if (!/^https?:\/\//i.test(href)) return
            e.preventDefault()
            App.OpenReleaseURL(href)
        }

        const onDownloadProgress = (payload) => {
            if (!payload) return
            // 丢弃 done 之后迟到的 downloading，避免卡在「下载中 100%」且按钮失灵
            if (payload.status === 'downloading' && readyToInstall.value && !downloading.value) {
                return
            }
            const pct = Number(payload.percent)
            if (!Number.isNaN(pct)) {
                downloadPercent.value = Math.max(0, Math.min(100, pct))
            }
            if (payload.status === 'start' || payload.status === 'downloading') {
                downloading.value = true
                downloadPaused.value = false
                downloadFailed.value = false
                readyToInstall.value = false
                downloadMessage.value = payload.message || (payload.status === 'start' ? '开始下载…' : '正在下载…')
            } else if (payload.status === 'done') {
                downloading.value = false
                downloadPaused.value = false
                downloadPercent.value = 100
                downloadFailed.value = false
                readyToInstall.value = true
                downloadMessage.value = '下载完成，可安装并重启'
            } else if (payload.status === 'paused') {
                downloading.value = false
                downloadPaused.value = true
                downloadFailed.value = false
                downloadMessage.value = payload.message || '已暂停，可更换下载源后继续'
            } else if (payload.status === 'error') {
                downloading.value = false
                downloadPaused.value = false
                downloadFailed.value = true
                downloadMessage.value = payload.message || '下载失败'
            }
        }

        const pauseDownload = async () => {
            if (!downloading.value) return
            try {
                await App.PauseUpdateDownload()
            } catch (e) {
                ElMessage.error('暂停失败: ' + e)
            }
        }

        const downloadUpdate = async () => {
            if (!canDownload.value) return
            downloading.value = true
            installing.value = false
            downloadPaused.value = false
            downloadFailed.value = false
            readyToInstall.value = false
            downloadMessage.value = '准备下载…'
            downloadPercent.value = 0
            try {
                const result = await App.DownloadUpdate(selectedDownloadSource.value || '')
                if (result?.success) {
                    readyToInstall.value = !!(result.readyToInstall || result.filePath)
                    ElMessage.success(result.message || '下载完成')
                    downloadMessage.value = result.message || '下载完成，可安装并重启'
                    downloadPercent.value = 100
                    downloadPaused.value = false
                    if (result.filePath && updateResult.value) {
                        updateResult.value = {
                            ...updateResult.value,
                            downloaded: true,
                            downloadPath: result.filePath,
                        }
                        setCachedUpdateCheck(updateResult.value)
                    }
                } else if (result?.paused) {
                    downloadPaused.value = true
                    downloadFailed.value = false
                    downloadMessage.value = result.message || '已暂停，可更换下载源后继续'
                } else {
                    downloadFailed.value = true
                    downloadPaused.value = false
                    downloadMessage.value = result?.message || '下载失败'
                    ElMessage.error(downloadMessage.value)
                }
            } catch (e) {
                downloadFailed.value = true
                downloadPaused.value = false
                downloadMessage.value = String(e)
                ElMessage.error(downloadMessage.value)
            } finally {
                downloading.value = false
            }
        }

        const installUpdate = async () => {
            if (!readyToInstall.value || installing.value || downloading.value) return
            installing.value = true
            try {
                const result = await App.InstallUpdateAndRestart()
                if (result?.success) {
                    ElMessage.success(result.message || '正在安装并重启…')
                    downloadMessage.value = result.message || '正在安装并重启…'
                } else {
                    ElMessage.error(result?.message || '安装失败')
                    downloadMessage.value = result?.message || '安装失败'
                    downloadFailed.value = true
                }
            } catch (e) {
                downloadFailed.value = true
                downloadMessage.value = String(e)
                ElMessage.error(downloadMessage.value)
            } finally {
                installing.value = false
            }
        }

        const openPackage = async () => {
            try {
                await App.OpenDownloadedUpdatePackage()
            } catch (e) {
                ElMessage.error('打开安装包失败: ' + e)
            }
        }

        watch(systemPanel, (panel) => {
            if (panel === 'security') loadKnownHosts()
        })

        watch(() => props.modelValue, (open) => {
            if (!props.embedded && open) load()
        })
        watch(() => props.active, (open) => {
            if (props.embedded && open) load()
        }, { immediate: true })
        watch(settingsTab, (tab) => {
            if (tab === 'about' && !checkingUpdate.value) {
                checkUpdate(false)
            }
        })

        let offDownloadProgress = null
        onMounted(() => {
            offDownloadProgress = EventsOn('update:download-progress', onDownloadProgress)
        })
        onUnmounted(() => {
            offDownloadProgress?.()
            offDownloadProgress = null
        })

        const applySettingsSearch = () => {
            const root = panelScrollRef.value
            if (!root) return
            const q = String(settingsSearch.value || '').trim().toLowerCase()
            const rows = root.querySelectorAll('.system-setting-row, .appear-field, .memory-saver-row, .section-head')
            rows.forEach((el) => {
                if (!q) {
                    el.style.display = ''
                    return
                }
                const label = el.querySelector?.('.system-setting-label, .appear-field-label, .memory-saver-label')
                const hint = el.querySelector?.('.system-setting-hint')
                const text = `${label?.textContent || ''} ${hint?.textContent || ''} ${el.textContent || ''}`.toLowerCase()
                el.style.display = text.includes(q) ? '' : 'none'
            })
        }

        watch(settingsSearch, () => nextTick(applySettingsSearch))
        watch([settingsTab, systemPanel, themePanel], () => nextTick(applySettingsSearch))

        const resetAccountForm = () => {
            accountForm.id = crypto.randomUUID()
            accountForm.name = ''
            accountForm.user = ''
            accountForm.password = ''
            accountForm.keyFile = ''
            accountForm.keyPassphrase = ''
        }

        const addAccount = () => {
            editingAccountIndex.value = -1
            resetAccountForm()
            accountEditVisible.value = true
        }

        const editAccount = (index) => {
            editingAccountIndex.value = index
            const account = accounts.value[index]
            accountForm.id = account.id || crypto.randomUUID()
            accountForm.name = account.name || ''
            accountForm.user = account.user || ''
            accountForm.password = account.password || ''
            accountForm.keyFile = account.keyFile || ''
            accountForm.keyPassphrase = account.keyPassphrase || ''
            accountEditVisible.value = true
        }

        const saveGlobalAccounts = async (message = '全局 SSH 帐号已保存') => {
            savingAccount.value = true
            try {
                await App.SaveGlobalAccountsFromDTO(accounts.value)
                if (message) ElMessage.success(message)
            } finally {
                savingAccount.value = false
            }
        }

        const removeAccount = async (index) => {
            accounts.value.splice(index, 1)
            try {
                await saveGlobalAccounts('帐号已删除')
            } catch (e) {
                ElMessage.error(`删除失败: ${e}`)
                await load()
            }
        }

        const selectAccountKeyFile = async () => {
            try {
                const filePath = await App.SelectKeyFile()
                if (filePath) accountForm.keyFile = filePath
            } catch (e) {
                ElMessage.error('选择密钥文件失败: ' + e)
            }
        }

        const confirmAccount = async () => {
            if (!accountForm.name.trim() || !accountForm.user.trim()) {
                ElMessage.warning('请填写帐号名称和用户名')
                return
            }
            const payload = {
                id: accountForm.id || crypto.randomUUID(),
                name: accountForm.name.trim(),
                user: accountForm.user.trim(),
                password: accountForm.password,
                keyFile: accountForm.keyFile || '',
                keyPassphrase: accountForm.keyPassphrase || '',
            }
            if (editingAccountIndex.value >= 0) {
                accounts.value[editingAccountIndex.value] = payload
            } else {
                accounts.value.push(payload)
            }
            try {
                await saveGlobalAccounts()
                accountEditVisible.value = false
            } catch (e) {
                ElMessage.error(`保存失败: ${e}`)
                await load()
            }
        }

        const save = async () => {
            saving.value = true
            try {
                const config = await App.GetSystemSettings()
                config.windowsName = (form.windowsName || '').trim() || 'FlashDock'
                config.appIconPreset = form.appIconPreset || 'default'
                config.startupFullscreen = !!form.startupFullscreen
                config.themeSettings = { ...form.themeSettings }
                config.shellMonitorIntervalMs = form.shellMonitorIntervalMs
                config.sshHandshakeTimeoutSec = form.sshHandshakeTimeoutSec
                config.shellTerminalScrollback = clampShellTerminalScrollback(form.shellTerminalScrollback)
                config.taskOutputMaxLines = clampTaskOutputMaxLines(form.taskOutputMaxLines)
                config.shellCommandHistoryMax = clampShellCommandHistoryMax(form.shellCommandHistoryMax)
                config.shellLogHighlight = !!form.shellLogHighlight
                config.shellAsciiInput = !!form.shellAsciiInput
                config.shellCursorLineHighlight = !!form.shellCursorLineHighlight
                config.shellLineTimestamps = !!form.shellLineTimestamps
                config.shellPasswordAssist = !!form.shellPasswordAssist
                config.sftpUseCompressedUpload = !!form.sftpUseCompressedUpload
                config.externalEditorCommand = String(form.externalEditorCommand || '').trim()
                const assocMap = {}
                String(fileAssocText.value || '').split(/\r?\n/).forEach((line) => {
                    const t = line.trim()
                    if (!t || t.startsWith('#')) return
                    const eq = t.indexOf('=')
                    if (eq <= 0) return
                    let ext = t.slice(0, eq).trim().toLowerCase()
                    const cmd = t.slice(eq + 1).trim()
                    if (!ext || !cmd) return
                    if (!ext.startsWith('.')) ext = `.${ext}`
                    assocMap[ext] = cmd
                })
                config.fileAssociations = assocMap
                config.shellLogHighlightColors = mergeLogHighlightColors(form.shellLogHighlightColors)
                config.shellLogHighlightDisabled = rulesToDisabled(form.shellLogHighlightRules)
                config.shellLogHighlightKeywords = normalizeCustomKeywords(form.shellLogHighlightKeywords)
                await App.SaveSystemSettings(config)
                form.shellLogHighlightKeywords.splice(
                    0,
                    form.shellLogHighlightKeywords.length,
                    ...normalizeCustomKeywords(config.shellLogHighlightKeywords),
                )
                form.windowsName = config.windowsName
                form.appIconPreset = config.appIconPreset
                applyThemeSettings(form.themeSettings)
                ElMessage.success('系统设置已保存')
                emit('saved')
                if (!props.embedded) visibleProxy.value = false
                await loadAppIconPresets()
            } catch (e) {
                ElMessage.error(`保存失败: ${e}`)
            } finally {
                saving.value = false
            }
        }

        return {
            embedded: computed(() => props.embedded),
            settingsSearch,
            panelScrollRef,
            settingsTab,
            settingsTabs,
            systemPanel,
            systemPanels,
            shellMode,
            themePanel,
            themePanels,
            panel: computed(() => props.panel),
            form,
            appIconPresets,
            uploadingAppIcon,
            uploadCustomAppIcon,
            logColorItems,
            logColorPresets,
            logHlDotKeys,
            logColorPredefine,
            activeLogHighlightPreset,
            logHighlightPreviewParts,
            applyLogHighlightPreset,
            resetLogHighlightConfig,
            addCustomKeyword,
            removeCustomKeyword,
            uiAccents,
            accentPredefine,
            isCustomAccentActive,
            customAccentColor,
            setCustomAccent,
            terminalPresets,
            uiFonts,
            terminalFonts,
            previewIsDark,
            previewUiStyle,
            previewTermStyle,
            previewUiFontLabel,
            previewTermLabel,
            previewTermSample,
            accounts,
            fileAssocText,
            knownHosts,
            loadingKnownHosts,
            importingKnownHosts,
            loadKnownHosts,
            importSystemKnownHosts,
            removeKnownHost,
            saving,
            savingAccount,
            sessionId,
            appVersion,
            checkingUpdate,
            updateResult,
            canDownload,
            downloading,
            installing,
            readyToInstall,
            downloadPaused,
            downloadPercent,
            downloadMessage,
            downloadFailed,
            downloadSources,
            selectedDownloadSource,
            downloadButtonLabel,
            checkUpdate,
            openRelease,
            renderReleaseNotes,
            onNotesClick,
            downloadUpdate,
            installUpdate,
            openPackage,
            pauseDownload,
            accountEditVisible,
            editingAccountIndex,
            accountForm,
            addAccount,
            editAccount,
            removeAccount,
            confirmAccount,
            selectAccountKeyFile,
            save,
        }
    }
}
</script>

<style scoped>
.general-settings-panel {
    display: flex;
    flex-direction: column;
    min-height: 0;
    height: 100%;
    overflow: hidden;
}

.general-settings-panel.embedded {
    padding-bottom: 0;
}

.general-settings-panel.is-panel-routed .system-editor,
.general-settings-panel.is-panel-routed .appear-editor {
    border: none;
    border-radius: 0;
    background: transparent;
}

.general-settings-panel.is-panel-routed .panel-actions {
    margin-left: -20px;
    margin-right: -20px;
    padding-left: 20px;
    padding-right: 20px;
}

.settings-stack {
    display: flex;
    flex-direction: column;
    gap: 20px;
    padding: 4px 2px 12px;
}

.setting-block {
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-width: 0;
}

.setting-block-title {
    margin: 0;
    font-size: 13px;
    font-weight: 650;
    color: var(--app-text);
    letter-spacing: 0.01em;
}

.setting-block > .section-head {
    margin-bottom: 0;
}

.setting-card {
    border: 1px solid var(--app-border);
    border-radius: 10px;
    background: var(--app-card-bg, var(--app-panel-bg));
    padding: 0 14px;
    overflow: hidden;
}

.setting-card--padded {
    padding: 14px;
}

.setting-card > .system-setting-row {
    border-bottom: 1px solid var(--app-border);
}

.setting-card > .system-setting-row:last-child {
    border-bottom: none;
}

.setting-card--padded > .system-setting-row {
    padding-left: 0;
    padding-right: 0;
}

.settings-search-bar {
    flex-shrink: 0;
    padding: 0 0 10px;
}

.settings-subnav {
    flex-shrink: 0;
    display: flex;
    gap: 4px;
    padding: 0 0 10px;
    border-bottom: 1px solid var(--app-border);
    margin-bottom: 12px;
}

.subnav-item {
    border: none;
    background: transparent;
    color: var(--app-text-muted);
    font-size: 13px;
    padding: 6px 12px;
    border-radius: 8px;
    cursor: pointer;
}

.subnav-item:hover {
    color: var(--app-accent-color);
    background: var(--app-accent-bg);
}

.subnav-item.active {
    color: var(--app-accent-color);
    background: var(--app-accent-bg);
    font-weight: 650;
}

.panel-scroll {
    flex: 1;
    min-height: 0;
    overflow: hidden;
    padding: 0;
    display: flex;
    flex-direction: column;
}

.panel-actions {
    flex-shrink: 0;
    display: flex;
    justify-content: flex-end;
    align-items: center;
    gap: 10px;
    margin: 0 -18px 0;
    padding: 12px 18px;
    border-top: 1px solid var(--app-border);
    background: var(--app-panel-bg);
}

.panel-actions.icon-actions {
    display: flex;
    width: auto;
    align-self: stretch;
}

.settings-section {
    flex: 1;
    min-height: 0;
    overflow: auto;
}

.section-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 12px;
}

.settings-section>.section-head+.section-head,
.system-setting-row+.section-head {
    margin-top: 22px;
    padding-top: 16px;
    border-top: 1px solid var(--app-border);
}

.section-head h4 {
    margin: 0;
    font-size: 14px;
    color: var(--app-text);
}

.section-head p {
    margin: 4px 0 0;
    font-size: 12px;
    color: var(--app-text-muted);
}

.appearance-section {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    height: 100%;
    min-height: 0;
}

.appear-layout {
    flex: 1;
    min-height: 0;
    display: grid;
    grid-template-columns: minmax(0, 1.15fr) minmax(260px, 0.85fr);
    gap: 16px;
    align-items: stretch;
    overflow: hidden;
}

.system-section {
    display: flex;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
}

.system-editor {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    border: 1px solid var(--app-border);
    border-radius: 10px;
    background: var(--app-bg);
}

.general-settings-panel.is-panel-routed .system-editor .appear-pane-body {
    padding: 0;
}

.appear-editor {
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
    overflow: hidden;
    border: 1px solid var(--app-border);
    border-radius: 10px;
    background: var(--app-bg);
}

.theme-subnav {
    flex-shrink: 0;
    display: flex;
    gap: 4px;
    padding: 10px 12px;
    border-bottom: 1px solid var(--app-border);
    background: color-mix(in srgb, var(--app-panel-bg) 70%, var(--app-bg));
}

.theme-subnav-item {
    border: none;
    background: transparent;
    color: var(--app-text-muted);
    font-size: 13px;
    padding: 6px 12px;
    border-radius: 8px;
    cursor: pointer;
}

.theme-subnav-item:hover {
    color: var(--app-accent-color);
    background: var(--app-accent-bg);
}

.theme-subnav-item.active {
    color: var(--app-accent-color);
    background: var(--app-accent-bg);
    font-weight: 650;
}

.appear-pane {
    flex: 1;
    min-height: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
}

.appear-pane-body {
    flex: 1;
    min-height: 0;
    overflow-x: hidden;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 16px;
    padding: 14px 14px 16px;
}

.appear-field {
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-width: 0;
}

.appear-field--fill {
    flex: 1;
    min-height: 0;
}

.appear-field-label {
    font-size: 12px;
    font-weight: 650;
    color: var(--app-text-secondary);
}

.appear-block {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 12px;
    border: 1px solid var(--app-border);
    border-radius: 10px;
    background: var(--app-bg);
    min-width: 0;
    overflow: hidden;
    flex-shrink: 0;
}

.appear-block--terminal {
    flex: 1;
    min-height: 0;
    flex-shrink: 1;
}

.appear-controls {
    display: flex;
    flex-direction: column;
    gap: 12px;
    min-width: 0;
    min-height: 0;
    overflow: hidden;
}

.block-label {
    font-size: 12px;
    font-weight: 650;
    color: var(--app-text-secondary);
    flex-shrink: 0;
}

.accent-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
}

.accent-swatch {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    border: 2px solid transparent;
    cursor: pointer;
    padding: 0;
}

.accent-swatch.active {
    box-shadow: 0 0 0 2px var(--app-panel-bg), 0 0 0 4px currentColor;
    outline: none;
    border-color: #fff;
}

.accent-custom {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    flex-shrink: 0;
}

.accent-custom.active {
    box-shadow: 0 0 0 2px var(--app-panel-bg), 0 0 0 4px currentColor;
}

.accent-custom :deep(.el-color-picker),
.accent-custom :deep(.el-color-picker__trigger) {
    width: 24px;
    height: 24px;
    padding: 0;
    border: 2px solid transparent;
    border-radius: 50%;
    overflow: hidden;
}

.accent-custom :deep(.el-color-picker__color),
.accent-custom :deep(.el-color-picker__color-inner) {
    border: none;
    border-radius: 50%;
}

.accent-custom:not(.active) :deep(.el-color-picker__color-inner) {
    background: conic-gradient(#ff4da6, #409eff, #16a34a, #d97706, #c026d3, #ff4da6) !important;
}

.accent-custom :deep(.el-color-picker__icon),
.accent-custom :deep(.el-icon) {
    display: none;
}

.terminal-grid {
    flex: 1;
    min-height: 140px;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(118px, 1fr));
    gap: 10px;
    overflow-y: auto;
    overflow-x: hidden;
    padding: 2px 4px 4px 0;
    align-content: start;
}

.term-card {
    display: flex;
    flex-direction: column;
    gap: 0;
    padding: 0;
    border: 1px solid var(--app-border);
    border-radius: 8px;
    background: transparent;
    cursor: pointer;
    overflow: hidden;
    color: var(--app-text);
    text-align: left;
    min-height: 88px;
}

.term-card.active {
    border-color: var(--app-accent-color);
    box-shadow: 0 0 0 1px var(--app-accent-color);
}

.term-card-preview {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 10px;
    min-height: 78px;
    font-size: 12px;
}

.term-card-dots {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
}

.term-card-dots i {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    display: inline-block;
}

.term-card-name {
    font-size: 12px;
    font-weight: 650;
    line-height: 1.3;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: inherit;
}

.term-card-preview code {
    font-family: Consolas, monospace;
    opacity: 0.75;
    font-size: 11px;
}

.term-font-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 76px 76px;
    gap: 8px;
    align-items: center;
    width: 100%;
    min-width: 0;
    flex-shrink: 0;
}

.term-font-row.ui-font-row {
    grid-template-columns: minmax(0, 1fr) 76px;
}

.term-font-select {
    width: 100%;
    min-width: 0;
}

.term-num {
    width: 76px !important;
}

.term-num :deep(.el-input-number),
.term-num.el-input-number {
    width: 76px;
}

.term-num :deep(.el-input__wrapper) {
    padding-left: 4px;
    padding-right: 28px;
}

.term-font-hints {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 76px 76px;
    gap: 8px;
    margin-top: -4px;
    font-size: 11px;
    color: var(--app-text-muted);
    flex-shrink: 0;
}

.term-font-hints.ui-font-hints {
    grid-template-columns: minmax(0, 1fr) 76px;
}

.memory-saver-row {
    display: flex;
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    font-size: 12px;
    color: var(--app-text-muted);
}

.memory-saver-label {
    flex: 1;
    min-width: 0;
    line-height: 1.4;
}

.shell-setting-row {
    display: flex;
    align-items: center;
    gap: 10px;
}

.shell-setting-label {
    font-size: 13px;
    color: var(--app-text);
    flex-shrink: 0;
}

.shell-setting-unit {
    font-size: 12px;
    color: var(--app-text-muted);
}

.shell-setting-hint {
    margin: 6px 0 0;
    font-size: 12px;
    color: var(--app-text-muted);
}

.system-setting-row {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    padding: 12px 0;
    border-bottom: 1px solid var(--app-border);
}

.log-hl-colors-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
}

.log-hl-colors-head .el-button {
    flex-shrink: 0;
    padding: 0 4px;
    height: auto;
}

.log-hl-colors-hint {
    margin: 4px 0 0;
    flex: 1;
    min-width: 0;
    font-size: 12px;
    line-height: 1.45;
    color: var(--app-text-muted);
}

.log-hl-preset-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(104px, 1fr));
    gap: 8px;
}

.log-hl-preset-card {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
    padding: 10px 10px 8px;
    border: 1px solid var(--app-border);
    border-radius: var(--app-radius-md, 8px);
    background: var(--app-card-bg, var(--app-bg));
    color: var(--app-text);
    cursor: pointer;
    text-align: left;
    transition: border-color 0.12s ease, background 0.12s ease, box-shadow 0.12s ease;
}

.log-hl-preset-card:hover:not(:disabled) {
    border-color: color-mix(in srgb, var(--app-accent-color) 45%, var(--app-border));
}

.log-hl-preset-card.active {
    border-color: var(--app-accent-color);
    background: var(--app-accent-bg);
    box-shadow: 0 0 0 1px color-mix(in srgb, var(--app-accent-color) 35%, transparent);
}

.log-hl-preset-card.is-custom:disabled {
    cursor: default;
    opacity: 1;
}

.log-hl-preset-card.is-custom:not(.active) {
    opacity: 0.55;
}

.log-hl-preset-dots {
    display: flex;
    gap: 4px;
}

.log-hl-preset-dots i {
    display: block;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    border: 1px solid color-mix(in srgb, var(--app-border) 70%, transparent);
}

.log-hl-preset-name {
    font-size: 12px;
    font-weight: 600;
    line-height: 1.2;
    color: inherit;
}

.log-hl-colors-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px 16px;
}

.log-hl-kw-empty {
    margin-top: 8px;
    font-size: 12px;
    color: var(--app-text-muted, #909399);
}

.log-hl-kw-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 8px;
}

.log-hl-kw-row {
    display: flex;
    align-items: center;
    gap: 8px;
}

.log-hl-kw-input {
    flex: 1;
    min-width: 0;
}

.log-hl-color-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    min-width: 0;
}

.log-hl-color-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
}

.log-hl-color-label {
    font-size: 12px;
    color: var(--app-text-secondary);
}

.log-hl-preview-line {
    margin: 0;
    padding: 0;
    border-radius: 0;
    background: transparent;
    color: inherit;
    font-family: inherit;
    font-size: inherit;
    line-height: inherit;
    white-space: pre-wrap;
    word-break: break-all;
    overflow-x: auto;
}

.system-setting-text {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
}

.system-setting-label {
    font-size: 13px;
    color: var(--app-text);
    font-weight: 500;
}

.system-setting-hint {
    font-size: 12px;
    color: var(--app-text-muted);
    line-height: 1.4;
}

.system-setting-control {
    display: grid;
    grid-template-columns: 120px 36px;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
}

.system-setting-control--wide {
    grid-template-columns: minmax(160px, 240px);
}

.system-setting-text-input {
    width: 100%;
}

.system-setting-row--stack {
    flex-direction: column;
    align-items: stretch;
}

.dock-icon-presets {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(72px, 1fr));
    gap: 10px;
    width: 100%;
}

.dock-icon-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    padding: 10px 8px 8px;
    border: 1px solid var(--app-border);
    border-radius: 10px;
    background: var(--app-bg);
    color: var(--app-text);
    cursor: pointer;
    transition: border-color 0.12s ease, box-shadow 0.12s ease, background 0.12s ease;
}

.dock-icon-card:hover:not(:disabled) {
    border-color: color-mix(in srgb, var(--app-accent-color) 45%, var(--app-border));
}

.dock-icon-card.active {
    border-color: var(--app-accent-color);
    background: var(--app-accent-bg);
    box-shadow: 0 0 0 1px color-mix(in srgb, var(--app-accent-color) 35%, transparent);
}

.dock-icon-card:disabled {
    opacity: 0.7;
    cursor: wait;
}

.dock-icon-img {
    width: 40px;
    height: 40px;
    border-radius: 10px;
    object-fit: cover;
    display: block;
}

.dock-icon-name {
    font-size: 11px;
    font-weight: 600;
    line-height: 1.2;
    text-align: center;
    color: inherit;
}

.dock-icon-card--upload {
    justify-content: center;
    min-height: 78px;
}

.dock-icon-upload-plus {
    width: 40px;
    height: 40px;
    border-radius: 10px;
    border: 1px dashed var(--app-border);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 22px;
    line-height: 1;
    color: var(--app-text-muted);
}

.system-setting-unit {
    font-size: 12px;
    color: var(--app-text-muted);
    white-space: nowrap;
}

.system-setting-num {
    width: 120px !important;
}

.system-setting-num :deep(.el-input-number),
.system-setting-num.el-input-number {
    width: 120px;
}

.appear-preview {
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-width: 0;
    min-height: 0;
    overflow: hidden;
    position: sticky;
    top: 0;
    align-self: stretch;
}

.theme-preview {
    flex: 1;
    min-height: 0;
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 10px;
    overflow: hidden;
}

.preview-ui,
.preview-term,
.preview-log {
    border: 1px solid var(--app-border);
    border-radius: 10px;
    overflow: hidden;
    min-height: 0;
}

.preview-ui {
    display: flex;
    flex-direction: column;
    flex: 1;
}

.preview-term {
    flex: 1;
    padding: 10px 12px;
}

.preview-log {
    flex: 1;
    display: flex;
    flex-direction: column;
    padding: 10px 12px;
}

.preview-log-line {
    flex: 1;
}

.preview-log-off {
    margin: 12px 0 0;
    font-size: 12px;
    opacity: 0.65;
}

.preview-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 10px;
    border-bottom: 1px solid color-mix(in srgb, currentColor 18%, transparent);
}

.preview-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--preview-accent, #409eff);
}

.preview-title {
    font-size: 0.93em;
    font-weight: 650;
}

.preview-pill {
    margin-left: auto;
    font-size: 0.79em;
    padding: 2px 8px;
    border-radius: 999px;
    background: var(--preview-accent-bg, #ecf5ff);
    color: var(--preview-accent, #409eff);
}

.preview-body {
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.preview-card {
    padding: 10px 12px;
    border-radius: 8px;
    border: 1px solid color-mix(in srgb, currentColor 14%, transparent);
    display: flex;
    flex-direction: column;
    gap: 6px;
}

.preview-card-title {
    font-size: 0.93em;
    font-weight: 650;
}

.preview-card-text {
    margin: 0;
    font-size: 0.86em;
    opacity: 0.78;
    line-height: 1.45;
}

.preview-card-sample {
    margin: 0;
    font-size: 1.07em;
    letter-spacing: 0.02em;
    line-height: 1.5;
}

.preview-btn {
    align-self: flex-start;
    border: none;
    border-radius: 6px;
    padding: 6px 12px;
    background: var(--preview-accent, #409eff);
    color: #fff;
    font-size: 0.86em;
    cursor: default;
}

.preview-term-title {
    font-size: 11px;
    opacity: 0.7;
    margin-bottom: 8px;
}

.preview-term pre,
.preview-log pre {
    margin: 0;
    white-space: pre-wrap;
    word-break: break-word;
    font-family: inherit;
    font-size: inherit;
    line-height: inherit;
}

.about-section {
    display: flex;
    flex-direction: column;
    gap: 14px;
}

.about-meta {
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.about-meta-row {
    display: grid;
    grid-template-columns: 96px minmax(0, 1fr);
    gap: 12px;
    align-items: center;
}

.about-meta-label {
    font-size: 13px;
    color: var(--app-text-secondary);
}

.about-meta-value {
    font-size: 13px;
    color: var(--app-text);
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

.about-update-block {
    min-height: 36px;
}

.about-update-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 10px;
}

.update-banner {
    padding: 12px 14px;
    margin-bottom: 8px;
    border-radius: 10px;
    border: 1px solid color-mix(in srgb, #e6a23c 45%, var(--app-border));
    background: color-mix(in srgb, #e6a23c 10%, transparent);
}

.update-banner-title {
    font-size: 14px;
    font-weight: 650;
    color: var(--app-text);
}

.update-banner-sub {
    margin-top: 2px;
    font-size: 12px;
    color: var(--app-text-muted);
}

.asset-line {
    margin: 8px 0 4px;
    font-size: 12px;
    color: var(--app-text-muted);
}

.update-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    align-items: center;
    margin-top: 10px;
}

.source-select {
    width: 200px;
}

.download-msg {
    margin-top: 8px;
    font-size: 12px;
    color: #67c23a;
    line-height: 1.4;
    word-break: break-all;
}

.download-msg.err {
    color: #f56c6c;
}

.download-msg.paused {
    color: #e6a23c;
}

.release-section {
    margin: 10px 0 0;
}

.release-section-title {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 6px;
    font-size: 13px;
    font-weight: 650;
    color: var(--app-text);
}

.update-notes {
    margin: 0;
    max-height: 220px;
    overflow: auto;
    padding: 10px 12px;
    border-radius: 8px;
    background: var(--app-card-bg);
    border: 1px solid var(--app-border);
    font-size: 12px;
    line-height: 1.55;
    word-break: break-word;
    color: var(--app-text-secondary);
}

.update-notes :deep(h1),
.update-notes :deep(h2),
.update-notes :deep(h3) {
    margin: 0.65em 0 0.35em;
    font-size: 13px;
    font-weight: 650;
    color: var(--app-text);
    line-height: 1.35;
}

.update-notes :deep(h1:first-child),
.update-notes :deep(h2:first-child),
.update-notes :deep(h3:first-child) {
    margin-top: 0;
}

.update-notes :deep(p),
.update-notes :deep(ul),
.update-notes :deep(ol) {
    margin: 0.35em 0;
}

.update-notes :deep(ul),
.update-notes :deep(ol) {
    padding-left: 1.35em;
}

.update-notes :deep(a) {
    color: var(--app-accent-color);
    text-decoration: none;
}

.update-notes :deep(code) {
    padding: 0 4px;
    border-radius: 3px;
    background: color-mix(in srgb, var(--app-text-muted) 14%, transparent);
    font-size: 0.95em;
}

.update-ok {
    margin-bottom: 6px;
    font-size: 13px;
    color: #67c23a;
}

.update-err {
    margin-bottom: 6px;
    font-size: 12px;
    color: #f56c6c;
    line-height: 1.45;
}

@media (max-width: 860px) {
    .appear-layout {
        grid-template-columns: 1fr;
        overflow-y: auto;
    }

    .appear-preview {
        position: static;
        min-height: 280px;
    }

    .theme-preview {
        min-height: 240px;
    }
}
</style>
