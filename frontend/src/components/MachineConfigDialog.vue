<template>
    <div class="machine-config-container" :class="{ embedded }">
        <div class="machine-list">
            <div class="list-header">
                <h4 v-if="!embedded">机器列表</h4>
                <div v-else class="list-header-spacer"></div>
                <div class="header-actions">
                    <div class="filter-bar">
                        <el-input
                            v-model="machineKeyword"
                            clearable
                            size="small"
                            placeholder="搜索名称 / IP"
                            class="app-toolbar-search compact list-search"
                        >
                            <template #prefix>
                                <el-icon class="filter-icon"><Search /></el-icon>
                            </template>
                        </el-input>
                        <el-select
                            v-model="importGroup"
                            clearable
                            filterable
                            allow-create
                            default-first-option
                            size="small"
                            placeholder="导入分组"
                            class="import-group-select"
                        >
                            <el-option
                                v-for="g in groupOptions"
                                :key="g"
                                :label="g"
                                :value="g === DEFAULT_MACHINE_GROUP ? '' : g"
                            />
                        </el-select>
                        <el-select
                            v-model="importAccountId"
                            clearable
                            size="small"
                            placeholder="导入帐号"
                            class="import-account-select"
                        >
                            <el-option
                                v-for="account in globalAccounts"
                                :key="account.id"
                                :label="account.name"
                                :value="account.id"
                            />
                        </el-select>
                    </div>
                <div class="toolbar-ops icon-actions">
                        <el-radio-group v-model="listViewMode" size="small" class="view-mode-toggle">
                            <el-radio-button label="table">
                                <el-icon><List /></el-icon>
                            </el-radio-button>
                            <el-radio-button label="board">
                                <el-icon><Grid /></el-icon>
                            </el-radio-button>
                        </el-radio-group>
                        <el-tooltip content="分组管理" placement="top">
                            <el-button size="small" circle @click="groupManageVisible = true">
                                <el-icon><FolderOpened /></el-icon>
                            </el-button>
                        </el-tooltip>
                        <el-tooltip content="添加机器" placement="top">
                            <el-button size="small" type="primary" circle @click="addMachine">
                                <el-icon><Plus /></el-icon>
                            </el-button>
                        </el-tooltip>
                        <el-dropdown
                            trigger="hover"
                            :show-timeout="120"
                            :hide-timeout="160"
                            @command="handleAddCommand"
                        >
                            <el-button size="small" circle title="导入机器">
                                <el-icon><Upload /></el-icon>
                            </el-button>
                            <template #dropdown>
                                <el-dropdown-menu>
                                    <el-dropdown-item command="import-xshell">导入 Xshell</el-dropdown-item>
                                    <el-dropdown-item command="import-finalshell">导入 FinalShell</el-dropdown-item>
                                    <el-dropdown-item command="import-putty">导入 PuTTY</el-dropdown-item>
                                    <el-dropdown-item command="import-mobaxterm">导入 MobaXterm</el-dropdown-item>
                                    <el-dropdown-item command="import-securecrt">导入 SecureCRT</el-dropdown-item>
                                    <el-dropdown-item command="import-openssh-default">导入本地 SSH 配置 (~/.ssh/config)</el-dropdown-item>
                                    <el-dropdown-item command="import-openssh">从文件导入 OpenSSH config</el-dropdown-item>
                                    <el-dropdown-item command="import-csv">导入 CSV</el-dropdown-item>
                                    <el-dropdown-item command="export-csv">导出 CSV</el-dropdown-item>
                                    <el-dropdown-item command="export-template" divided>导出连接模板</el-dropdown-item>
                                    <el-dropdown-item command="import-template">导入连接模板</el-dropdown-item>
                                </el-dropdown-menu>
                            </template>
                        </el-dropdown>
                    </div>
                </div>
            </div>

            <div v-if="listViewMode === 'table'" class="machine-table-wrap" v-loading="machinesLoading">
                <div v-if="!filteredMachines.length" class="app-empty">
                    <p class="app-empty-desc">暂无机器</p>
                </div>
                <ul v-else class="ml-list">
                    <li
                        v-for="machine in filteredMachines"
                        :key="machine.id || machine.name"
                        class="ml-item"
                        :class="{ 'is-context-target': isContextTarget(machine) }"
                        @click="editMachine(machine)"
                        @contextmenu.prevent="onMachineContextMenu($event, machine)"
                    >
                        <div class="ml-machine-icon" aria-hidden="true">
                            <span v-if="hostIconText(machine)" class="ml-machine-emoji">{{ hostIconText(machine) }}</span>
                            <el-icon v-else :size="16"><Monitor /></el-icon>
                        </div>
                        <div class="ml-body">
                            <div class="ml-line">
                                <TextOverflowTooltip :text="machine.name" text-class="ml-name" />
                                <span class="ml-badge is-group">{{ machine.group || DEFAULT_MACHINE_GROUP }}</span>
                            </div>
                            <TextOverflowTooltip :text="formatMachineAddr(machine)" text-class="ml-addr" />
                        </div>
                        <div class="ml-side ml-actions-fade icon-actions" @click.stop>
                            <el-tooltip content="连接" placement="top">
                                <el-button size="small" text type="primary" @click="connectMachine(machine)">
                                    <el-icon><VideoPlay /></el-icon>
                                </el-button>
                            </el-tooltip>
                            <el-tooltip content="编辑" placement="top">
                                <el-button size="small" text @click="editMachine(machine)">
                                    <el-icon><Edit /></el-icon>
                                </el-button>
                            </el-tooltip>
                            <el-tooltip content="测试连接" placement="top">
                                <el-button size="small" text type="success" :loading="machine.testing" @click="testConnection(machine)">
                                    <el-icon v-if="!machine.testing"><Connection /></el-icon>
                                </el-button>
                            </el-tooltip>
                            <el-tooltip content="删除" placement="top">
                                <el-button size="small" text type="danger" @click="deleteMachine(machine)">
                                    <el-icon><Delete /></el-icon>
                                </el-button>
                            </el-tooltip>
                        </div>
                    </li>
                </ul>
            </div>

            <div v-else class="machine-board-wrap" v-loading="machinesLoading">
                <p class="board-hint">拖动机器卡片到其他分组即可快速改组</p>
                <div class="machine-board">
                    <section
                        v-for="group in boardGroups"
                        :key="group.name"
                        class="board-column"
                        :class="{ 'is-drop-target': dragOverGroup === group.name }"
                        @dragover.prevent="onBoardDragOver(group.name, $event)"
                        @dragleave="onBoardDragLeave(group.name, $event)"
                        @drop.prevent="onBoardDrop(group.name, $event)"
                    >
                        <header class="board-column-head">
                            <span class="board-column-title">{{ group.name }}</span>
                            <span class="ml-group-count">{{ group.machines.length }}</span>
                        </header>
                        <div class="board-column-body">
                            <div
                                v-for="machine in group.machines"
                                :key="machine.id || machine.name"
                                class="board-card"
                                :class="{
                                    'is-dragging': draggingMachineId === (machine.id || machine.name),
                                    'is-context-target': isContextTarget(machine),
                                }"
                                draggable="true"
                                @dragstart="onBoardDragStart(machine, $event)"
                                @dragend="onBoardDragEnd"
                                @dblclick="editMachine(machine)"
                                @contextmenu.prevent="onMachineContextMenu($event, machine)"
                            >
                                <div class="ml-machine-icon" aria-hidden="true">
                                    <span v-if="hostIconText(machine)" class="ml-machine-emoji">{{ hostIconText(machine) }}</span>
                                    <el-icon v-else :size="16"><Monitor /></el-icon>
                                </div>
                                <div class="board-card-main">
                                    <TextOverflowTooltip :text="machine.name" text-class="ml-name" />
                                    <TextOverflowTooltip :text="formatMachineAddr(machine)" text-class="ml-addr" />
                                </div>
                                <div class="board-card-actions icon-actions" @mousedown.stop @click.stop>
                                    <el-tooltip content="连接" placement="top">
                                        <el-button size="small" text type="primary" @click="connectMachine(machine)">
                                            <el-icon><VideoPlay /></el-icon>
                                        </el-button>
                                    </el-tooltip>
                                    <el-tooltip content="编辑" placement="top">
                                        <el-button size="small" text @click="editMachine(machine)">
                                            <el-icon><Edit /></el-icon>
                                        </el-button>
                                    </el-tooltip>
                                    <el-tooltip content="删除" placement="top">
                                        <el-button size="small" text type="danger" @click="deleteMachine(machine)">
                                            <el-icon><Delete /></el-icon>
                                        </el-button>
                                    </el-tooltip>
                                </div>
                            </div>
                            <div v-if="!group.machines.length" class="board-empty">拖到此处</div>
                        </div>
                    </section>
                </div>
            </div>
        </div>

        <el-dialog
            v-model="machineEditVisible"
            :title="editingMachine ? '编辑机器' : '添加机器'"
            width="520px"
            class="settings-sub-dialog machine-edit-dialog"
            append-to-body
        >
            <el-form
                class="machine-edit-form"
                :model="machineForm"
                :rules="machineRules"
                ref="machineFormRef"
                label-position="top"
                require-asterisk-position="right"
            >
                <section class="machine-form-section">
                    <header class="machine-form-section-head">
                        <el-icon><Monitor /></el-icon>
                        <span>通用</span>
                    </header>
                    <div class="machine-form-section-body">
                        <el-form-item label="名称" prop="name">
                            <el-input v-model="machineForm.name" placeholder="机器名称" />
                        </el-form-item>
                        <el-form-item label="分组" prop="group">
                            <el-select
                                v-model="machineForm.group"
                                clearable
                                filterable
                                allow-create
                                default-first-option
                                placeholder="选择或输入分组"
                                style="width: 100%"
                            >
                                <el-option
                                    v-for="g in groupOptions"
                                    :key="g"
                                    :label="g"
                                    :value="g === DEFAULT_MACHINE_GROUP ? '' : g"
                                />
                            </el-select>
                            <el-button class="section-link-btn" text type="primary" size="small" @click="applyGroupDefaultsToForm">
                                应用分组默认
                            </el-button>
                        </el-form-item>
                        <el-form-item label="标签">
                            <el-select
                                v-model="machineForm.tags"
                                multiple
                                filterable
                                allow-create
                                default-first-option
                                collapse-tags
                                collapse-tags-tooltip
                                placeholder="输入后回车添加标签"
                                style="width: 100%"
                            >
                                <el-option v-for="t in knownTagOptions" :key="t" :label="t" :value="t" />
                            </el-select>
                        </el-form-item>
                        <el-form-item label="AI 策略">
                            <el-select v-model="machineForm.aiPolicy" placeholder="trusted（默认）" clearable style="width: 100%">
                                <el-option label="disabled · 禁止任何 MCP" value="disabled" />
                                <el-option label="readonly · 只读自动放行，改动拒绝" value="readonly" />
                                <el-option label="approval · 读写均需审批" value="approval" />
                                <el-option label="allowlist · 命中正则 auto，否则审批" value="allowlist" />
                                <el-option label="trusted · 自动放行（仍拦致命命令/sudo）" value="trusted" />
                            </el-select>
                        </el-form-item>
                        <el-form-item v-if="machineForm.aiPolicy === 'allowlist'" label="AI 白名单">
                            <el-input
                                v-model="machineForm.aiAllowlistText"
                                type="textarea"
                                :rows="3"
                                placeholder="每行一条命令前缀或正则，如 ^df\s 或 systemctl status"
                            />
                            <span class="form-hint">未命中白名单时升级为审批，不会直接拒绝</span>
                        </el-form-item>
                        <el-form-item label="允许 AI sudo">
                            <el-switch v-model="machineForm.aiAllowSudo" />
                            <span class="form-hint" style="margin-left: 8px">含 sudo 无视档位强制审批；关闭则直接拒绝</span>
                        </el-form-item>
                        <el-form-item label="备注">
                            <el-input
                                v-model="machineForm.notes"
                                type="textarea"
                                :rows="3"
                                placeholder="运维备注（支持检索）"
                                maxlength="4000"
                                show-word-limit
                            />
                        </el-form-item>
                        <el-form-item label="主机图标">
                            <el-select v-model="machineForm.icon" filterable allow-create clearable placeholder="预设或自定义 emoji" style="width: 100%">
                                <el-option
                                    v-for="opt in hostIconPresets"
                                    :key="opt.id || 'default'"
                                    :label="opt.emoji ? `${opt.emoji} ${opt.label}` : opt.label"
                                    :value="opt.id"
                                />
                            </el-select>
                        </el-form-item>
                    </div>
                </section>

                <section class="machine-form-section">
                    <header class="machine-form-section-head">
                        <el-icon><Location /></el-icon>
                        <span>地址</span>
                    </header>
                    <div class="machine-form-section-body">
                        <el-form-item label="主机地址" prop="host">
                            <el-input v-model="machineForm.host" placeholder="IP 或主机名" />
                        </el-form-item>
                    </div>
                </section>

                <section class="machine-form-section">
                    <header class="machine-form-section-head">
                        <el-icon><Key /></el-icon>
                        <span>端口与凭据</span>
                    </header>
                    <div class="machine-form-section-body">
                        <div class="machine-form-row-2">
                            <el-form-item label="用户名" prop="user">
                                <el-input v-model="machineForm.user" placeholder="SSH 用户名" />
                            </el-form-item>
                            <el-form-item label="端口" prop="port">
                                <el-input-number v-model="machineForm.port" :min="1" :max="65535" controls-position="right" />
                            </el-form-item>
                        </div>
                        <el-form-item label="全局帐号">
                            <el-select
                                v-model="selectedAccountId"
                                clearable
                                placeholder="选择后自动填充用户名和密码"
                                style="width: 100%"
                                @change="applyGlobalAccount"
                            >
                                <el-option
                                    v-for="account in globalAccounts"
                                    :key="account.id"
                                    :label="account.name"
                                    :value="account.id"
                                />
                            </el-select>
                        </el-form-item>
                        <el-form-item label="密码" prop="password">
                            <el-input v-model="machineForm.password" type="password" placeholder="可选" show-password />
                        </el-form-item>
                        <el-form-item label="密钥文件" prop="key_file">
                            <div class="key-file-input">
                                <el-input v-model="machineForm.key_file" placeholder="私钥路径" readonly />
                                <el-tooltip content="选择文件" placement="top">
                                    <el-button type="primary" @click="selectKeyFile">
                                        <el-icon><Folder /></el-icon>
                                    </el-button>
                                </el-tooltip>
                            </div>
                        </el-form-item>
                        <el-form-item label="密钥口令">
                            <el-input v-model="machineForm.keyPassphrase" type="password" placeholder="加密私钥口令（可选）" show-password clearable />
                        </el-form-item>
                    </div>
                </section>

                <section class="machine-form-section">
                    <header class="machine-form-section-head">
                        <el-icon><Link /></el-icon>
                        <span>跳板与代理</span>
                    </header>
                    <div class="machine-form-section-body">
                        <el-form-item label="跳板机">
                            <el-input v-model="machineForm.proxyJump" placeholder="单跳：机器名或 host[:port]" clearable />
                        </el-form-item>
                        <el-form-item label="跳板链">
                            <el-select
                                v-model="machineForm.jumpChain"
                                multiple
                                filterable
                                allow-create
                                default-first-option
                                collapse-tags
                                collapse-tags-tooltip
                                placeholder="按顺序选择或输入跳板（优先于单跳）"
                                style="width: 100%"
                            >
                                <el-option
                                    v-for="m in machines"
                                    :key="m.id || m.name"
                                    :label="m.name"
                                    :value="m.name"
                                />
                            </el-select>
                            <div v-if="machineForm.jumpChain?.length" class="jump-chain-order">
                                <div
                                    v-for="(hop, idx) in machineForm.jumpChain"
                                    :key="`${hop}-${idx}`"
                                    class="jump-chain-row"
                                >
                                    <span class="jump-chain-idx">{{ idx + 1 }}</span>
                                    <span class="jump-chain-name">{{ hop }}</span>
                                    <el-button size="small" text :disabled="idx === 0" @click="moveJumpHop(idx, -1)">上移</el-button>
                                    <el-button size="small" text :disabled="idx === machineForm.jumpChain.length - 1" @click="moveJumpHop(idx, 1)">下移</el-button>
                                </div>
                            </div>
                            <p class="field-hint">多跳时按从左到右顺序连接，最后一跳再连目标主机</p>
                        </el-form-item>
                        <el-form-item label="代理模式">
                            <el-select v-model="machineForm.proxyOverride.mode" style="width: 100%">
                                <el-option label="继承全局" value="inherit" />
                                <el-option label="直连（不走代理）" value="none" />
                                <el-option label="独立代理" value="manual" />
                            </el-select>
                        </el-form-item>
                        <template v-if="machineForm.proxyOverride.mode === 'manual'">
                            <el-form-item label="代理类型">
                                <el-select v-model="machineForm.proxyOverride.type" style="width: 100%">
                                    <el-option label="HTTP" value="http" />
                                    <el-option label="SOCKS5" value="socks" />
                                </el-select>
                            </el-form-item>
                            <el-form-item label="代理主机">
                                <el-input v-model="machineForm.proxyOverride.host" placeholder="主机" />
                            </el-form-item>
                            <el-form-item label="代理端口">
                                <el-input-number v-model="machineForm.proxyOverride.port" :min="1" :max="65535" style="width: 100%" />
                            </el-form-item>
                            <el-form-item label="代理用户">
                                <el-input v-model="machineForm.proxyOverride.user" clearable />
                            </el-form-item>
                            <el-form-item label="代理密码">
                                <el-input v-model="machineForm.proxyOverride.password" type="password" show-password clearable />
                            </el-form-item>
                        </template>
                    </div>
                </section>

                <section class="machine-form-section">
                    <header class="machine-form-section-head">
                        <el-icon><FolderOpened /></el-icon>
                        <span>SFTP 设置</span>
                    </header>
                    <div class="machine-form-section-body">
                        <el-form-item label="文件协议">
                            <el-select v-model="machineForm.sftpFileProtocol" style="width: 100%">
                                <el-option label="自动（SFTP 优先，失败回退 SCP）" value="auto" />
                                <el-option label="仅 SFTP" value="sftp" />
                                <el-option label="仅 SCP" value="scp" />
                            </el-select>
                            <p class="field-hint">远端无 SFTP 子系统时可用 SCP 回退完成浏览与传输</p>
                        </el-form-item>
                        <el-form-item label="文件名编码">
                            <el-select v-model="machineForm.sftpEncoding" style="width: 100%">
                                <el-option label="自动" value="auto" />
                                <el-option label="UTF-8" value="utf-8" />
                                <el-option label="GB18030（中文 Windows 远端）" value="gb18030" />
                            </el-select>
                            <p class="field-hint">影响 SFTP 文件名编解码；远端中文乱码时优先试 GB18030</p>
                        </el-form-item>
                        <el-form-item label="Sudo 提权">
                            <div class="machine-form-switch-row">
                                <span class="machine-form-switch-label">以 sudo 打开 SFTP</span>
                                <el-switch
                                    v-model="machineForm.sftpSudo"
                                    :disabled="machineForm.sftpFileProtocol === 'scp'"
                                />
                            </div>
                            <p class="field-hint">需密码认证与远端 sudo 权限；与「仅 SCP」互斥</p>
                        </el-form-item>
                    </div>
                </section>

                <section class="machine-form-section">
                    <header class="machine-form-section-head">
                        <el-icon><Setting /></el-icon>
                        <span>高级选项</span>
                    </header>
                    <div class="machine-form-section-body">
                        <el-form-item label="旧设备算法">
                            <div class="machine-form-switch-row">
                                <span class="machine-form-switch-label">启用兼容算法</span>
                                <el-switch v-model="machineForm.legacyAlgorithms" />
                            </div>
                        </el-form-item>
                        <el-form-item label="主机密钥">
                            <div class="machine-form-switch-row">
                                <span class="machine-form-switch-label">跳过 ECDSA 主机密钥</span>
                                <el-switch v-model="machineForm.skipEcdsaHostKey" />
                            </div>
                        </el-form-item>
                        <el-form-item label="启动命令">
                            <el-input v-model="machineForm.startupCommand" placeholder="连接后自动执行（单行）" clearable />
                        </el-form-item>
                        <el-form-item label="Agent 转发">
                            <div class="machine-form-switch-row">
                                <span class="machine-form-switch-label">启用 SSH Agent 转发</span>
                                <el-switch v-model="machineForm.agentForwarding" />
                            </div>
                        </el-form-item>
                        <el-form-item label="本地回显">
                            <div class="machine-form-switch-row">
                                <span class="machine-form-switch-label">高延迟时本机立即显示输入</span>
                                <el-switch v-model="machineForm.localEcho" />
                            </div>
                            <p class="field-hint">可打印字符由客户端立刻显示，并抑制远端重复回显。全屏 TUI / 密码提示时自动停用。这不是 X11 图形转发。</p>
                        </el-form-item>
                        <el-form-item label="终端配色">
                            <el-select v-model="machineForm.terminalPreset" clearable placeholder="跟随全局主题" style="width: 100%">
                                <el-option label="跟随全局" value="" />
                                <el-option v-for="preset in terminalPresetOptions" :key="preset.id" :label="preset.label" :value="preset.id" />
                            </el-select>
                        </el-form-item>
                    </div>
                </section>

                <section class="machine-form-section">
                    <header class="machine-form-section-head">
                        <el-icon><Share /></el-icon>
                        <span>SSH 隧道</span>
                        <el-button class="section-head-action" size="small" text type="primary" @click="addTunnel">
                            <el-icon><Plus /></el-icon>
                            添加
                        </el-button>
                    </header>
                    <div class="machine-form-section-body">
                        <p class="field-hint tunnel-top-hint">连接成功后自动建立；本地转发：本机端口 → 远端地址</p>
                        <div
                            v-for="(tun, idx) in machineForm.tunnels"
                            :key="idx"
                            class="tunnel-row"
                        >
                            <el-switch v-model="tun.enabled" size="small" />
                            <el-select v-model="tun.type" size="small" style="width: 88px">
                                <el-option label="本地" value="local" />
                                <el-option label="远程" value="remote" />
                                <el-option label="SOCKS" value="dynamic" />
                            </el-select>
                            <el-input v-model="tun.name" size="small" placeholder="名称" style="width: 72px" />
                            <el-input-number v-model="tun.localPort" size="small" :min="1" :max="65535" controls-position="right" style="width: 96px" />
                            <template v-if="tun.type !== 'dynamic'">
                                <el-input v-model="tun.remoteHost" size="small" placeholder="远端主机" style="width: 96px" />
                                <el-input-number v-model="tun.remotePort" size="small" :min="1" :max="65535" controls-position="right" style="width: 96px" />
                            </template>
                            <el-button size="small" text type="danger" @click="machineForm.tunnels.splice(idx, 1)">
                                <el-icon><Delete /></el-icon>
                            </el-button>
                        </div>
                        <p v-if="!machineForm.tunnels?.length" class="tunnel-empty">暂无隧道，点击右上角添加</p>
                    </div>
                </section>
            </el-form>

            <template #footer>
                <div class="dialog-footer icon-actions">
                    <el-tooltip content="取消" placement="top">
                        <el-button circle @click="machineEditVisible = false">
                            <el-icon><Close /></el-icon>
                        </el-button>
                    </el-tooltip>
                    <el-tooltip content="测试连接" placement="top">
                        <el-button :loading="testingDraft" circle @click="testDraftConnection">
                            <el-icon v-if="!testingDraft"><Connection /></el-icon>
                        </el-button>
                    </el-tooltip>
                    <el-tooltip :content="editingMachine ? '更新' : '添加'" placement="top">
                        <el-button type="primary" circle :loading="savingMachine" @click="saveMachine">
                            <el-icon v-if="!savingMachine"><Check /></el-icon>
                        </el-button>
                    </el-tooltip>
                </div>
            </template>
        </el-dialog>

        <el-dialog
            v-model="groupManageVisible"
            title="分组管理"
            width="480px"
            class="settings-sub-dialog"
            append-to-body
            @open="loadGroups"
        >
            <div class="group-add-row">
                <el-input v-model="newGroupName" placeholder="新分组名称" clearable @keydown.enter.exact.prevent="addGroup" />
                <el-tooltip content="添加分组" placement="top">
                    <el-button type="primary" circle @click="addGroup">
                        <el-icon><Plus /></el-icon>
                    </el-button>
                </el-tooltip>
            </div>
            <el-table :data="managedGroups" size="small" empty-text="暂无自定义分组">
                <el-table-column prop="name" label="分组名称" />
                <el-table-column label="操作" width="140" align="center">
                    <template #default="{ row }">
                        <div class="icon-actions">
                            <el-tooltip content="默认配置" placement="top">
                                <el-button size="small" text type="primary" @click="editGroupDefaults(row.name)">
                                    <el-icon><Setting /></el-icon>
                                </el-button>
                            </el-tooltip>
                            <el-tooltip content="重命名" placement="top">
                                <el-button size="small" text type="primary" @click="renameGroup(row.name)">
                                    <el-icon><Edit /></el-icon>
                                </el-button>
                            </el-tooltip>
                            <el-tooltip content="删除" placement="top">
                                <el-button size="small" text type="danger" @click="deleteGroup(row.name)">
                                    <el-icon><Delete /></el-icon>
                                </el-button>
                            </el-tooltip>
                        </div>
                    </template>
                </el-table-column>
            </el-table>
            <template #footer>
                <div class="dialog-footer icon-actions">
                    <el-tooltip content="关闭" placement="top">
                        <el-button circle @click="groupManageVisible = false">
                            <el-icon><Close /></el-icon>
                        </el-button>
                    </el-tooltip>
                </div>
            </template>
        </el-dialog>

        <el-dialog
            v-model="groupDefaultsVisible"
            :title="`分组默认：${groupDefaultsForm.name}`"
            width="480px"
            class="settings-sub-dialog"
            append-to-body
        >
            <el-form :model="groupDefaultsForm" label-width="100px" size="small">
                <el-form-item label="默认用户">
                    <el-input v-model="groupDefaultsForm.user" clearable />
                </el-form-item>
                <el-form-item label="密钥文件">
                    <div class="key-file-input">
                        <el-input v-model="groupDefaultsForm.keyFile" readonly />
                        <el-button type="primary" circle @click="selectGroupDefaultKeyFile">
                            <el-icon><Folder /></el-icon>
                        </el-button>
                    </div>
                </el-form-item>
                <el-form-item label="跳板机">
                    <el-input v-model="groupDefaultsForm.proxyJump" clearable />
                </el-form-item>
                <el-form-item label="启动命令">
                    <el-input v-model="groupDefaultsForm.startupCommand" clearable />
                </el-form-item>
                <el-form-item label="SFTP 编码">
                    <el-select v-model="groupDefaultsForm.sftpEncoding" style="width: 100%">
                        <el-option label="自动" value="auto" />
                        <el-option label="UTF-8" value="utf-8" />
                        <el-option label="GB18030" value="gb18030" />
                    </el-select>
                </el-form-item>
                <el-form-item label="Agent 转发">
                    <el-switch v-model="groupDefaultsForm.agentForwarding" active-text="启用 SSH Agent 转发" />
                </el-form-item>
                <el-form-item label="本地回显">
                    <el-switch v-model="groupDefaultsForm.localEcho" active-text="启用本地回显" />
                </el-form-item>
                <el-form-item label="代理覆盖">
                    <el-select v-model="groupDefaultsForm.proxyOverride.mode" style="width: 100%">
                        <el-option label="继承全局" value="inherit" />
                        <el-option label="直连" value="none" />
                        <el-option label="独立代理" value="manual" />
                    </el-select>
                </el-form-item>
                <template v-if="groupDefaultsForm.proxyOverride.mode === 'manual'">
                    <el-form-item label="代理类型">
                        <el-select v-model="groupDefaultsForm.proxyOverride.type" style="width: 100%">
                            <el-option label="HTTP" value="http" />
                            <el-option label="SOCKS" value="socks" />
                        </el-select>
                    </el-form-item>
                    <el-form-item label="代理主机">
                        <el-input v-model="groupDefaultsForm.proxyOverride.host" placeholder="主机" />
                    </el-form-item>
                    <el-form-item label="代理端口">
                        <el-input-number v-model="groupDefaultsForm.proxyOverride.port" :min="1" :max="65535" />
                    </el-form-item>
                    <el-form-item label="代理用户">
                        <el-input v-model="groupDefaultsForm.proxyOverride.user" clearable />
                    </el-form-item>
                    <el-form-item label="代理密码">
                        <el-input v-model="groupDefaultsForm.proxyOverride.password" type="password" show-password clearable />
                    </el-form-item>
                </template>
                <el-form-item label="默认标签">
                    <el-select
                        v-model="groupDefaultsForm.tags"
                        multiple
                        filterable
                        allow-create
                        default-first-option
                        collapse-tags
                        style="width: 100%"
                    />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="groupDefaultsVisible = false">取消</el-button>
                <el-button type="primary" :loading="savingGroupDefaults" @click="saveGroupDefaults">保存</el-button>
            </template>
        </el-dialog>

        <MachineContextMenu
            :ctx="ctx"
            @connect="onContextConnect"
            @copy="onContextCopy"
            @edit="onContextEdit"
            @delete="onContextDelete"
            @hide="hideContextMenu"
        />

    </div>
</template>

<script>
import { ref, reactive, watch, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
    Plus, Search, FolderOpened, Upload, Edit, Delete, Connection, Folder, List, Grid, Close, Check, Monitor, VideoPlay, Setting,
    Location, Key, Link, Share,
} from '@element-plus/icons-vue'
import {
    GetMachines,
    GetMachineGroups,
    GetMachineGroupDefaults,
    SaveMachineGroupDefaults,
    AddMachineGroup,
    RenameMachineGroup,
    DeleteMachineGroup,
    GetGlobalAccounts,
    GetMachineSensitiveData,
    UpdateMachine,
    UpdateMachineGroup,
    SetMachineSensitiveData,
    CreateMachine,
    DeleteMachine,
    TestMachineConnection,
    TestMachineDraftConnection,
    SelectKeyFile,
    ImportXshellPick,
    ImportFinalShellPick,
    ImportPuttyPick,
    ImportMobaXtermPick,
    ImportSecureCRTPick,
    ImportOpenSSHConfigPick,
    ImportOpenSSHConfigDefault,
    ImportMachinesCSVPick,
    ExportMachinesCSVPick,
    ExportMachineTemplateToFile,
    ImportMachineTemplateFromFile,
} from '../../wailsjs/go/app/App'
import {
    DEFAULT_MACHINE_GROUP,
    sortMachinesByName,
    machineMatchesKeyword,
    getMachineGroup,
    formatMachineAddr,
    normalizeMachineTags,
    collectMachineTags,
} from '../utils/machineGroups'
import { copyMachineRecord } from '../utils/machineCopy'
import { TERMINAL_PRESETS } from '../utils/themePresets'
import { HOST_ICON_PRESETS, resolveHostIcon } from '../utils/hostIcons'
import { useMachineContextMenu } from '../composables/useMachineContextMenu'
import MachineContextMenu from './shell/MachineContextMenu.vue'
import TextOverflowTooltip from './TextOverflowTooltip.vue'

export default {
    name: 'MachineConfigDialog',
    components: {
        Plus, Search, FolderOpened, Upload, Edit, Delete, Connection, Folder, List, Grid, Close, Check, Monitor, VideoPlay, Setting,
        Location, Key, Link, Share,
        MachineContextMenu,
        TextOverflowTooltip,
    },
    props: {
        modelValue: { type: Boolean, default: false },
        embedded: { type: Boolean, default: false },
        active: { type: Boolean, default: false },
        editMachineId: { type: String, default: '' },
    },
    emits: ['update:modelValue', 'closed', 'changed', 'connect'],
    setup(props, { emit }) {
        const visibleProxy = ref(props.modelValue)
        const machines = ref([])
        const machineGroups = ref([])
        const machineKeyword = ref('')
        const listViewMode = ref('board')
        const draggingMachineId = ref('')
        const dragOverGroup = ref('')
        const movingMachine = ref(false)
        const copyingMachine = ref(false)
        const { ctx, hideContextMenu, onMachineContextMenu, isContextTarget } = useMachineContextMenu()
        const sortedMachines = computed(() => sortMachinesByName(machines.value))
        const filteredMachines = computed(() => {
            const kw = machineKeyword.value
            const list = sortedMachines.value
            if (!String(kw || '').trim()) return list
            return list.filter((m) => machineMatchesKeyword(m, kw))
        })
        const boardGroups = computed(() => {
            const map = new Map()
            map.set(DEFAULT_MACHINE_GROUP, [])
            for (const g of machineGroups.value || []) {
                const name = String(g || '').trim()
                if (name && name !== DEFAULT_MACHINE_GROUP && !map.has(name)) {
                    map.set(name, [])
                }
            }
            for (const m of filteredMachines.value) {
                const name = getMachineGroup(m)
                if (!map.has(name)) map.set(name, [])
                map.get(name).push(m)
            }
            const names = Array.from(map.keys()).sort((a, b) => {
                if (a === DEFAULT_MACHINE_GROUP) return -1
                if (b === DEFAULT_MACHINE_GROUP) return 1
                return a.localeCompare(b, 'zh-CN')
            })
            return names.map((name) => ({ name, machines: map.get(name) || [] }))
        })
        const groupOptions = computed(() => {
            const set = new Set([DEFAULT_MACHINE_GROUP])
            for (const g of machineGroups.value || []) {
                if (g) set.add(g)
            }
            for (const m of machines.value || []) {
                if (m.group) set.add(m.group)
            }
            return Array.from(set).sort((a, b) => {
                if (a === DEFAULT_MACHINE_GROUP) return -1
                if (b === DEFAULT_MACHINE_GROUP) return 1
                return a.localeCompare(b, 'zh-CN')
            })
        })
        const managedGroups = computed(() =>
            (machineGroups.value || [])
                .filter((g) => g && g !== DEFAULT_MACHINE_GROUP)
                .map((name) => ({ name })),
        )
        const globalAccounts = ref([])
        const machinesLoading = ref(false)
        const machineEditVisible = ref(false)
        const groupManageVisible = ref(false)
        const groupDefaultsVisible = ref(false)
        const savingGroupDefaults = ref(false)
        const groupDefaultsList = ref([])
        const groupDefaultsForm = reactive({
            name: '',
            user: '',
            keyFile: '',
            proxyJump: '',
            startupCommand: '',
            sftpEncoding: 'auto',
            agentForwarding: false,
            localEcho: false,
            proxyOverride: {
                mode: 'inherit',
                type: 'http',
                host: '',
                port: 7890,
                user: '',
                password: '',
            },
            tags: [],
        })
        const newGroupName = ref('')
        const savingMachine = ref(false)
        const testingDraft = ref(false)
        const editingMachine = ref(null)
        const machineFormRef = ref(null)
        const selectedAccountId = ref('')
        const importAccountId = ref('')
        const importGroup = ref('')

        const machineForm = reactive({
            name: '',
            group: '',
            tags: [],
            notes: '',
            aiPolicy: 'trusted',
            aiAllowSudo: false,
            aiAllowlistText: '',
            identityId: '',
            key_file: '',
            host: '',
            port: 22,
            user: '',
            password: '',
            keyPassphrase: '',
            icon: '',
            proxyJump: '',
            jumpChain: [],
            proxyOverride: {
                mode: 'inherit',
                type: 'http',
                host: '',
                port: 7890,
                user: '',
                password: '',
            },
            legacyAlgorithms: false,
            skipEcdsaHostKey: false,
            sftpEncoding: 'auto',
            sftpFileProtocol: 'auto',
            sftpSudo: false,
            startupCommand: '',
            agentForwarding: false,
            localEcho: false,
            terminalPreset: '',
            tunnels: [],
        })

        const knownTagOptions = computed(() => collectMachineTags(machines.value))
        const hostIconPresets = HOST_ICON_PRESETS
        const hostIconText = (machine) => resolveHostIcon(machine).text
        const moveJumpHop = (idx, delta) => {
            const list = machineForm.jumpChain || []
            const to = idx + delta
            if (to < 0 || to >= list.length) return
            const next = [...list]
            const tmp = next[idx]
            next[idx] = next[to]
            next[to] = tmp
            machineForm.jumpChain = next
        }

        const emptyTunnel = () => ({
            enabled: true,
            name: '',
            type: 'local',
            localHost: '127.0.0.1',
            localPort: 0,
            remoteHost: '127.0.0.1',
            remotePort: 0,
        })

        const addTunnel = () => {
            machineForm.tunnels.push(emptyTunnel())
        }

        const machineRules = {
            name: [{ required: true, message: '请输入机器名称', trigger: 'blur' }],
            host: [{ required: true, message: '请输入主机地址', trigger: 'blur' }],
            port: [{ required: true, message: '请输入端口', trigger: 'blur' }],
            user: [{ required: true, message: '请输入用户名', trigger: 'blur' }]
        }

        const handleClose = () => {
            visibleProxy.value = false
        }

        const loadGroupDefaults = async () => {
            try {
                groupDefaultsList.value = await GetMachineGroupDefaults() || []
            } catch {
                groupDefaultsList.value = []
            }
        }

        const loadGroups = async () => {
            try {
                machineGroups.value = await GetMachineGroups() || []
                await loadGroupDefaults()
            } catch {
                machineGroups.value = []
            }
        }

        const loadMachines = async () => {
            const showDelayedSpinner = machines.value.length === 0
            let spinnerTimer = null
            try {
                // 本地 YAML 通常很快；仅在空列表且超过阈值时再转圈，避免切页闪一下
                if (showDelayedSpinner) {
                    spinnerTimer = setTimeout(() => {
                        machinesLoading.value = true
                    }, 160)
                }
                const machinesData = await GetMachines()
                machines.value = machinesData || []
                await loadGroups()
            } catch (error) {
                console.error('加载机器配置失败:', error)
                ElMessage.error('加载机器配置失败: ' + error.message)
            } finally {
                if (spinnerTimer) clearTimeout(spinnerTimer)
                machinesLoading.value = false
            }
        }

        const loadGlobalAccounts = async () => {
            try {
                globalAccounts.value = await GetGlobalAccounts() || []
            } catch {
                globalAccounts.value = []
            }
        }

        const applyGlobalAccount = (accountId) => {
            machineForm.identityId = accountId || ''
            if (!accountId) return
            const account = globalAccounts.value.find((item) => item.id === accountId)
            if (!account) return
            machineForm.user = account.user || ''
            machineForm.password = account.password || ''
            if (account.keyFile) machineForm.key_file = account.keyFile
        }

        const applyGroupDefaultsToForm = () => {
            const groupName = normalizeGroup(machineForm.group)
            const defaults = groupDefaultsList.value.find((item) => normalizeGroup(item.name) === groupName)
            if (!defaults) {
                ElMessage.info('当前分组暂无默认配置')
                return
            }
            if (defaults.user) machineForm.user = defaults.user
            if (defaults.keyFile) machineForm.key_file = defaults.keyFile
            if (defaults.proxyJump) machineForm.proxyJump = defaults.proxyJump
            if (defaults.startupCommand) machineForm.startupCommand = defaults.startupCommand
            if (defaults.sftpEncoding) machineForm.sftpEncoding = defaults.sftpEncoding
            if (defaults.agentForwarding) machineForm.agentForwarding = true
            if (defaults.localEcho) machineForm.localEcho = true
            if (defaults.proxyOverride?.mode && defaults.proxyOverride.mode !== 'inherit') {
                const po = defaults.proxyOverride
                machineForm.proxyOverride = {
                    mode: po.mode || 'inherit',
                    type: po.type || 'http',
                    host: po.host || '',
                    port: po.port || 7890,
                    user: po.user || '',
                    password: po.password || '',
                }
            }
            if (defaults.tags?.length) machineForm.tags = normalizeMachineTags(defaults.tags)
            ElMessage.success('已应用分组默认')
        }

        const editGroupDefaults = (groupName) => {
            const existing = groupDefaultsList.value.find((item) => item.name === groupName)
            groupDefaultsForm.name = groupName
            groupDefaultsForm.user = existing?.user || ''
            groupDefaultsForm.keyFile = existing?.keyFile || ''
            groupDefaultsForm.proxyJump = existing?.proxyJump || ''
            groupDefaultsForm.startupCommand = existing?.startupCommand || ''
            groupDefaultsForm.sftpEncoding = existing?.sftpEncoding || 'auto'
            groupDefaultsForm.agentForwarding = !!existing?.agentForwarding
            groupDefaultsForm.localEcho = !!existing?.localEcho
            const po = existing?.proxyOverride || {}
            groupDefaultsForm.proxyOverride = {
                mode: po.mode || 'inherit',
                type: po.type || 'http',
                host: po.host || '',
                port: po.port || 7890,
                user: po.user || '',
                password: po.password || '',
            }
            groupDefaultsForm.tags = normalizeMachineTags(existing?.tags || [])
            groupDefaultsVisible.value = true
        }

        const selectGroupDefaultKeyFile = async () => {
            try {
                const filePath = await SelectKeyFile()
                if (filePath) groupDefaultsForm.keyFile = filePath
            } catch (error) {
                ElMessage.error('选择密钥文件失败: ' + error.message)
            }
        }

        const saveGroupDefaults = async () => {
            savingGroupDefaults.value = true
            try {
                const po = { ...groupDefaultsForm.proxyOverride }
                if (po.mode !== 'manual') {
                    po.host = ''
                    po.port = 0
                    po.user = ''
                    po.password = ''
                }
                await SaveMachineGroupDefaults({
                    name: groupDefaultsForm.name,
                    user: groupDefaultsForm.user?.trim() || '',
                    keyFile: groupDefaultsForm.keyFile || '',
                    proxyJump: groupDefaultsForm.proxyJump?.trim() || '',
                    startupCommand: groupDefaultsForm.startupCommand?.trim() || '',
                    sftpEncoding: groupDefaultsForm.sftpEncoding || 'auto',
                    agentForwarding: !!groupDefaultsForm.agentForwarding,
                    localEcho: !!groupDefaultsForm.localEcho,
                    proxyOverride: po.mode === 'inherit' ? null : po,
                    tags: normalizeMachineTags(groupDefaultsForm.tags),
                })
                ElMessage.success('分组默认已保存')
                groupDefaultsVisible.value = false
                await loadGroupDefaults()
            } catch (error) {
                ElMessage.error('保存失败: ' + (error?.message || error))
            } finally {
                savingGroupDefaults.value = false
            }
        }

        const addMachine = () => {
            editingMachine.value = null
            selectedAccountId.value = ''
            resetMachineForm()
            machineEditVisible.value = true
        }

        const editMachine = async (machine) => {
            editingMachine.value = machine
            selectedAccountId.value = machine.identityId || ''
            machineForm.name = machine.name
            machineForm.group = machine.group || ''
            machineForm.tags = normalizeMachineTags(machine.tags)
            machineForm.notes = machine.notes || ''
            machineForm.aiPolicy = machine.aiPolicy || 'trusted'
            machineForm.aiAllowSudo = !!machine.aiAllowSudo
            machineForm.aiAllowlistText = Array.isArray(machine.aiAllowlist) ? machine.aiAllowlist.join('\n') : ''
            machineForm.icon = machine.icon || ''
            machineForm.identityId = machine.identityId || ''
            machineForm.key_file = machine.key_file || ''
            machineForm.proxyJump = machine.proxyJump || ''
            machineForm.jumpChain = Array.isArray(machine.jumpChain) ? [...machine.jumpChain] : []
            const po = machine.proxyOverride || {}
            machineForm.proxyOverride = {
                mode: po.mode || 'inherit',
                type: po.type || 'http',
                host: po.host || '',
                port: po.port || 7890,
                user: po.user || '',
                password: po.password || '',
            }
            machineForm.legacyAlgorithms = !!machine.legacyAlgorithms
            machineForm.skipEcdsaHostKey = !!machine.skipEcdsaHostKey
            machineForm.sftpEncoding = machine.sftpEncoding || 'auto'
            machineForm.sftpFileProtocol = machine.sftpFileProtocol || 'auto'
            machineForm.sftpSudo = !!machine.sftpSudo
            machineForm.startupCommand = machine.startupCommand || ''
            machineForm.agentForwarding = !!machine.agentForwarding
            machineForm.localEcho = !!machine.localEcho
            machineForm.terminalPreset = machine.terminalPreset || ''
            machineForm.tunnels = (machine.tunnels || []).map((t) => ({
                enabled: t.enabled !== false,
                name: t.name || '',
                type: t.type || 'local',
                localHost: t.localHost || '127.0.0.1',
                localPort: t.localPort || 0,
                remoteHost: t.remoteHost || '127.0.0.1',
                remotePort: t.remotePort || 0,
            }))
            try {
                const sensitiveData = await GetMachineSensitiveData(machine.id)
                if (sensitiveData) {
                    machineForm.host = sensitiveData.host || ''
                    machineForm.port = sensitiveData.port || 22
                    machineForm.user = sensitiveData.user || ''
                    machineForm.password = sensitiveData.password || ''
                    machineForm.keyPassphrase = sensitiveData.keyPassphrase || ''
                }
            } catch (error) {
                console.error('获取敏感数据失败:', error)
                ElMessage.warning('获取敏感数据失败，请重新输入')
            }
            machineEditVisible.value = true
        }

        const activate = async () => {
            await Promise.all([loadMachines(), loadGlobalAccounts()])
            if (props.editMachineId) {
                const target = machines.value.find((m) => m.id === props.editMachineId)
                if (target) await editMachine(target)
            }
        }

        watch(() => props.modelValue, async (v) => {
            visibleProxy.value = v
            if (props.embedded) return
            if (v) await activate()
            else emit('closed')
        })
        watch(visibleProxy, (v) => {
            if (!props.embedded) emit('update:modelValue', v)
        })
        watch(() => props.active, async (v) => {
            if (!props.embedded) return
            if (v) await activate()
            // 侧栏切换隐藏时不要当成「关闭」，避免误清 editMachineId；真正关设置窗仍由 Hub handleClose 发出
        }, { immediate: true })

        const resetMachineForm = () => {
            machineForm.name = ''
            machineForm.group = ''
            machineForm.tags = []
            machineForm.notes = ''
            machineForm.aiPolicy = 'trusted'
            machineForm.aiAllowSudo = false
            machineForm.aiAllowlistText = ''
            machineForm.icon = ''
            machineForm.identityId = ''
            machineForm.key_file = ''
            machineForm.host = ''
            machineForm.port = 22
            machineForm.user = ''
            machineForm.password = ''
            machineForm.keyPassphrase = ''
            machineForm.proxyJump = ''
            machineForm.jumpChain = []
            machineForm.proxyOverride = {
                mode: 'inherit',
                type: 'http',
                host: '',
                port: 7890,
                user: '',
                password: '',
            }
            machineForm.legacyAlgorithms = false
            machineForm.skipEcdsaHostKey = false
            machineForm.sftpEncoding = 'auto'
            machineForm.sftpFileProtocol = 'auto'
            machineForm.sftpSudo = false
            machineForm.startupCommand = ''
            machineForm.agentForwarding = false
            machineForm.localEcho = false
            machineForm.terminalPreset = ''
            machineForm.tunnels = []
        }

        const normalizeGroup = (g) => {
            const s = String(g || '').trim()
            if (!s || s === DEFAULT_MACHINE_GROUP) return ''
            return s
        }

        const onBoardDragStart = (machine, event) => {
            draggingMachineId.value = machine.id || machine.name
            dragOverGroup.value = ''
            try {
                event.dataTransfer.effectAllowed = 'move'
                event.dataTransfer.setData('text/plain', machine.id || '')
                event.dataTransfer.setData('application/x-machine-id', machine.id || '')
            } catch {
                // ignore
            }
        }

        const onBoardDragEnd = () => {
            draggingMachineId.value = ''
            dragOverGroup.value = ''
        }

        const onBoardDragOver = (groupName, event) => {
            if (!draggingMachineId.value) return
            event.dataTransfer.dropEffect = 'move'
            dragOverGroup.value = groupName
        }

        const onBoardDragLeave = (groupName, event) => {
            const related = event.relatedTarget
            if (related && event.currentTarget?.contains?.(related)) return
            if (dragOverGroup.value === groupName) dragOverGroup.value = ''
        }

        const onBoardDrop = async (groupName, event) => {
            dragOverGroup.value = ''
            const machineId =
                event.dataTransfer.getData('application/x-machine-id') ||
                event.dataTransfer.getData('text/plain') ||
                draggingMachineId.value
            draggingMachineId.value = ''
            if (!machineId || movingMachine.value) return
            const machine = machines.value.find((m) => m.id === machineId)
            if (!machine) return
            if (getMachineGroup(machine) === groupName) return

            const prevGroup = machine.group || ''
            const nextGroup = normalizeGroup(groupName)
            machine.group = nextGroup
            movingMachine.value = true
            try {
                await UpdateMachineGroup(machine.id, nextGroup)
                emit('changed')
                await loadGroups()
            } catch (error) {
                machine.group = prevGroup
                ElMessage.error('移动分组失败: ' + (error.message || error))
            } finally {
                movingMachine.value = false
            }
        }

        const saveMachine = async () => {
            if (!machineFormRef.value) return
            try {
                await machineFormRef.value.validate()
            } catch {
                // 表单校验失败时 Element Plus 会 reject 字段错误对象，字段下方已有红字提示，勿再弹系统错误
                return
            }
            savingMachine.value = true
            try {
                const proxyOverride = { ...machineForm.proxyOverride }
                if (proxyOverride.mode !== 'manual') {
                    proxyOverride.host = ''
                    proxyOverride.port = 0
                    proxyOverride.user = ''
                    proxyOverride.password = ''
                }
                const machineData = {
                    name: machineForm.name,
                    group: normalizeGroup(machineForm.group),
                    tags: normalizeMachineTags(machineForm.tags),
                    notes: String(machineForm.notes || '').trim(),
                    aiPolicy: machineForm.aiPolicy || '',
                    aiAllowSudo: !!machineForm.aiAllowSudo,
                    aiAllowlist: String(machineForm.aiAllowlistText || '')
                        .split(/\n+/)
                        .map((s) => s.trim())
                        .filter(Boolean),
                    icon: String(machineForm.icon || '').trim(),
                    identityId: machineForm.identityId || '',
                    key_file: machineForm.key_file,
                    proxyJump: machineForm.proxyJump?.trim() || '',
                    jumpChain: (machineForm.jumpChain || []).map((s) => String(s).trim()).filter(Boolean),
                    proxyOverride: proxyOverride.mode === 'inherit' ? null : proxyOverride,
                    legacyAlgorithms: machineForm.legacyAlgorithms,
                    skipEcdsaHostKey: machineForm.skipEcdsaHostKey,
                    sftpEncoding: machineForm.sftpEncoding || 'auto',
                    sftpFileProtocol: machineForm.sftpFileProtocol || 'auto',
                    sftpSudo: !!machineForm.sftpSudo && machineForm.sftpFileProtocol !== 'scp',
                    startupCommand: machineForm.startupCommand?.trim() || '',
                    agentForwarding: machineForm.agentForwarding,
                    localEcho: !!machineForm.localEcho,
                    terminalPreset: machineForm.terminalPreset || '',
                    tunnels: (machineForm.tunnels || []).filter((t) => t.localPort > 0),
                }
                const sensitiveData = {
                    host: machineForm.host,
                    port: machineForm.port,
                    user: machineForm.user,
                    password: machineForm.password,
                    keyPassphrase: machineForm.keyPassphrase || '',
                }
                if (editingMachine.value) {
                    machineData.id = editingMachine.value.id
                    await UpdateMachine(editingMachine.value.id, machineData)
                    await SetMachineSensitiveData(editingMachine.value.id, sensitiveData)
                    ElMessage.success('机器配置更新成功')
                } else {
                    await CreateMachine(machineData, sensitiveData)
                    ElMessage.success('机器配置添加成功')
                }
                machineEditVisible.value = false
                await loadMachines()
                emit('changed')
            } catch (error) {
                console.error('保存机器配置失败:', error)
                ElMessage.error('保存机器配置失败: ' + (error?.message || error))
            } finally {
                savingMachine.value = false
            }
        }

        const deleteMachine = async (machine) => {
            try {
                await ElMessageBox.confirm(`确定删除机器「${machine.name}」吗？`, '确认删除', { type: 'warning' })
                await DeleteMachine(machine.id)
                ElMessage.success('机器配置删除成功')
                await loadMachines()
                emit('changed')
            } catch (error) {
                if (error === 'cancel') return
                console.error('删除机器配置失败:', error)
                ElMessage.error('删除机器配置失败: ' + error.message)
            }
        }

        const copyMachine = async (machine) => {
            if (!machine?.id || copyingMachine.value) return
            copyingMachine.value = true
            try {
                const copyName = await copyMachineRecord(machine, machines.value)
                ElMessage.success(`已复制为「${copyName}」`)
                await loadMachines()
                emit('changed')
            } catch (error) {
                console.error('复制机器配置失败:', error)
                ElMessage.error('复制机器配置失败: ' + (error.message || error))
            } finally {
                copyingMachine.value = false
            }
        }

        const onContextCopy = (machine) => {
            hideContextMenu()
            if (machine) copyMachine(machine)
        }

        const onContextConnect = (machine) => {
            hideContextMenu()
            if (machine) connectMachine(machine)
        }

        const onContextEdit = (machine) => {
            hideContextMenu()
            if (machine) editMachine(machine)
        }

        const onContextDelete = (machine) => {
            hideContextMenu()
            if (machine) deleteMachine(machine)
        }

        const connectMachine = (machine) => {
            const name = String(machine?.name || '').trim()
            if (!name) {
                ElMessage.warning('机器名称无效')
                return
            }
            emit('connect', name)
        }

        const errText = (error) => String(error?.message || error || '')

        const testConnection = async (machine) => {
            try {
                machine.testing = true
                await TestMachineConnection(machine.id)
                ElMessage.success('连接测试成功')
            } catch (error) {
                console.error('连接测试失败:', error)
                ElMessage.error('连接测试失败: ' + errText(error))
            } finally {
                machine.testing = false
            }
        }

        const testDraftConnection = async () => {
            if (!machineFormRef.value) return
            try {
                await machineFormRef.value.validate()
            } catch {
                // 表单校验失败时字段下方已有红字提示，勿再弹系统错误
                return
            }
            testingDraft.value = true
            try {
                await TestMachineDraftConnection(
                    {
                        name: machineForm.name || 'draft-test',
                        group: normalizeGroup(machineForm.group),
                        identityId: machineForm.identityId || '',
                        key_file: machineForm.key_file,
                        proxyJump: machineForm.proxyJump?.trim() || '',
                    },
                    {
                        host: machineForm.host,
                        port: machineForm.port,
                        user: machineForm.user,
                        password: machineForm.password,
                    }
                )
                ElMessage.success('连接测试成功')
            } catch (error) {
                console.error('连接测试失败:', error)
                ElMessage.error('连接测试失败: ' + errText(error))
            } finally {
                testingDraft.value = false
            }
        }

        const selectKeyFile = async () => {
            try {
                const filePath = await SelectKeyFile()
                if (filePath) {
                    machineForm.key_file = filePath
                }
            } catch (error) {
                console.error('选择密钥文件失败:', error)
                ElMessage.error('选择密钥文件失败: ' + error.message)
            }
        }

        const showImportResult = (result) => {
            const errors = result?.errors?.length ? `\n失败: ${result.errors.join('\n')}` : ''
            ElMessage.success(`导入完成：成功 ${result?.imported || 0}，跳过 ${result?.skipped || 0}${errors}`)
        }

        const ensureImportApi = (fn, label) => {
            if (typeof fn !== 'function') {
                ElMessage.error(`${label} 不可用，请停止后重新运行 wails dev`)
                return false
            }
            return true
        }

        const importXshell = async () => {
            if (!ensureImportApi(ImportXshellPick, 'Xshell 导入')) return
            try {
                const result = await ImportXshellPick(importAccountId.value || '', normalizeGroup(importGroup.value))
                if (!result) return
                showImportResult(result)
                await loadMachines()
                emit('changed')
            } catch (error) {
                ElMessage.error('导入失败: ' + error)
            }
        }

        const importFinalShell = async () => {
            if (!ensureImportApi(ImportFinalShellPick, 'FinalShell 导入')) return
            try {
                const result = await ImportFinalShellPick(importAccountId.value || '', normalizeGroup(importGroup.value))
                if (!result) return
                showImportResult(result)
                await loadMachines()
                emit('changed')
            } catch (error) {
                ElMessage.error('导入失败: ' + error)
            }
        }

        const importPutty = async () => {
            if (!ensureImportApi(ImportPuttyPick, 'PuTTY 导入')) return
            try {
                const result = await ImportPuttyPick(importAccountId.value || '', normalizeGroup(importGroup.value))
                if (!result) return
                showImportResult(result)
                await loadMachines()
                emit('changed')
            } catch (error) {
                ElMessage.error('导入失败: ' + error)
            }
        }

        const importMobaXterm = async () => {
            if (!ensureImportApi(ImportMobaXtermPick, 'MobaXterm 导入')) return
            try {
                const result = await ImportMobaXtermPick(importAccountId.value || '', normalizeGroup(importGroup.value))
                if (!result) return
                showImportResult(result)
                await loadMachines()
                emit('changed')
            } catch (error) {
                ElMessage.error('导入失败: ' + error)
            }
        }

        const importSecureCRT = async () => {
            if (!ensureImportApi(ImportSecureCRTPick, 'SecureCRT 导入')) return
            try {
                const result = await ImportSecureCRTPick(importAccountId.value || '', normalizeGroup(importGroup.value))
                if (!result) return
                showImportResult(result)
                await loadMachines()
                emit('changed')
            } catch (error) {
                ElMessage.error('导入失败: ' + error)
            }
        }

        const exportTemplate = async () => {
            try {
                const path = await ExportMachineTemplateToFile()
                if (path) ElMessage.success('已导出: ' + path)
            } catch (e) {
                ElMessage.error('导出失败: ' + e)
            }
        }

        const importTemplate = async () => {
            try {
                const { value } = await ElMessageBox.confirm(
                    '导入时若机器名称已存在，是否合并更新？',
                    '导入连接模板',
                    { confirmButtonText: '合并更新', cancelButtonText: '仅新增', distinguishCancelAndClose: true },
                ).then(() => true).catch((action) => {
                    if (action === 'cancel') return false
                    throw action
                })
                const result = await ImportMachineTemplateFromFile(!!value)
                ElMessage.success(`导入完成：新增 ${result.added}，更新 ${result.updated}，跳过 ${result.skipped}`)
                await loadMachines()
                emit('changed')
            } catch (e) {
                if (e !== 'close') ElMessage.error('导入失败: ' + e)
            }
        }

        const showOpenSSHImportResult = (result) => {
            const errors = result?.errors?.length ? `\n失败: ${result.errors.join('\n')}` : ''
            ElMessage.success(`导入完成：新增 ${result?.added || 0}，更新 ${result?.updated || 0}，跳过 ${result?.skipped || 0}${errors}`)
        }

        const importOpenSSHDefault = async () => {
            if (!ensureImportApi(ImportOpenSSHConfigDefault, '本地 SSH 配置导入')) return
            try {
                const result = await ImportOpenSSHConfigDefault(importAccountId.value || '', normalizeGroup(importGroup.value))
                if (!result) return
                showOpenSSHImportResult(result)
                await loadMachines()
                emit('changed')
            } catch (error) {
                ElMessage.error('导入失败: ' + error)
            }
        }

        const importOpenSSH = async () => {
            if (!ensureImportApi(ImportOpenSSHConfigPick, 'OpenSSH 导入')) return
            try {
                const result = await ImportOpenSSHConfigPick(importAccountId.value || '', normalizeGroup(importGroup.value))
                if (!result) return
                showOpenSSHImportResult(result)
                await loadMachines()
                emit('changed')
            } catch (error) {
                ElMessage.error('导入失败: ' + error)
            }
        }

        const importCSV = async () => {
            if (!ensureImportApi(ImportMachinesCSVPick, 'CSV 导入')) return
            try {
                const result = await ImportMachinesCSVPick()
                if (!result) return
                showImportResult(result)
                await loadMachines()
                emit('changed')
            } catch (error) {
                ElMessage.error('导入失败: ' + error)
            }
        }

        const exportCSV = async () => {
            try {
                const fn = typeof ExportMachinesCSVPick === 'function'
                    ? ExportMachinesCSVPick
                    : window?.go?.app?.App?.ExportMachinesCSVPick
                if (typeof fn !== 'function') {
                    ElMessage.warning('导出 API 不可用：请完全重启 wails dev')
                    return
                }
                const path = await fn()
                if (path) ElMessage.success('已导出: ' + path)
            } catch (error) {
                ElMessage.error('导出失败: ' + error)
            }
        }

        const handleAddCommand = (command) => {
            if (command === 'import-finalshell') importFinalShell()
            else if (command === 'import-xshell') importXshell()
            else if (command === 'import-putty') importPutty()
            else if (command === 'import-mobaxterm') importMobaXterm()
            else if (command === 'import-securecrt') importSecureCRT()
            else if (command === 'import-openssh-default') importOpenSSHDefault()
            else if (command === 'import-openssh') importOpenSSH()
            else if (command === 'import-csv') importCSV()
            else if (command === 'export-csv') exportCSV()
            else if (command === 'export-template') exportTemplate()
            else if (command === 'import-template') importTemplate()
        }

        const addGroup = async () => {
            const name = newGroupName.value.trim()
            if (!name) {
                ElMessage.warning('请输入分组名称')
                return
            }
            try {
                await AddMachineGroup(name)
                newGroupName.value = ''
                ElMessage.success('分组已添加')
                await loadGroups()
                emit('changed')
            } catch (e) {
                ElMessage.error('添加分组失败: ' + e)
            }
        }

        const renameGroup = async (oldName) => {
            try {
                const { value } = await ElMessageBox.prompt('请输入新的分组名称', '重命名分组', {
                    inputValue: oldName,
                    confirmButtonText: '确定',
                    cancelButtonText: '取消',
                    inputValidator: (v) => {
                        const s = String(v || '').trim()
                        if (!s) return '名称不能为空'
                        if (s === DEFAULT_MACHINE_GROUP) return '不能使用默认分组名称'
                        return true
                    },
                })
                await RenameMachineGroup(oldName, String(value).trim())
                ElMessage.success('分组已重命名')
                await loadMachines()
                emit('changed')
            } catch (e) {
                if (e === 'cancel') return
                ElMessage.error('重命名失败: ' + e)
            }
        }

        const deleteGroup = async (name) => {
            try {
                await ElMessageBox.confirm(
                    `删除分组「${name}」后，该分组下的机器将归入「${DEFAULT_MACHINE_GROUP}」。确定删除？`,
                    '删除分组',
                    { type: 'warning' },
                )
                await DeleteMachineGroup(name)
                ElMessage.success('分组已删除')
                await loadMachines()
                emit('changed')
            } catch (e) {
                if (e === 'cancel') return
                ElMessage.error('删除失败: ' + e)
            }
        }

        return {
            embedded: computed(() => props.embedded),
            DEFAULT_MACHINE_GROUP,
            formatMachineAddr,
            visibleProxy,
            machines,
            sortedMachines,
            filteredMachines,
            boardGroups,
            listViewMode,
            draggingMachineId,
            dragOverGroup,
            groupOptions,
            knownTagOptions,
            hostIconPresets,
            hostIconText,
            moveJumpHop,
            managedGroups,
            machineKeyword,
            globalAccounts,
            machinesLoading,
            machineEditVisible,
            groupManageVisible,
            groupDefaultsVisible,
            groupDefaultsForm,
            savingGroupDefaults,
            newGroupName,
            savingMachine,
            testingDraft,
            editingMachine,
            machineFormRef,
            machineForm,
            machineRules,
            terminalPresetOptions: TERMINAL_PRESETS,
            addTunnel,
            selectedAccountId,
            importAccountId,
            importGroup,
            handleClose,
            addMachine,
            editMachine,
            saveMachine,
            deleteMachine,
            copyMachine,
            onMachineContextMenu,
            isContextTarget,
            onContextCopy,
            onContextConnect,
            onContextEdit,
            onContextDelete,
            ctx,
            hideContextMenu,
            connectMachine,
            testConnection,
            testDraftConnection,
            selectKeyFile,
            applyGlobalAccount,
            applyGroupDefaultsToForm,
            editGroupDefaults,
            selectGroupDefaultKeyFile,
            saveGroupDefaults,
            handleAddCommand,
            loadGroups,
            addGroup,
            renameGroup,
            deleteGroup,
            onBoardDragStart,
            onBoardDragEnd,
            onBoardDragOver,
            onBoardDragLeave,
            onBoardDrop,
        }
    }
}
</script>

<style scoped>
.machine-config-container {
    height: min(62vh, 560px);
    display: flex;
    flex-direction: column;
    min-height: 0;
}

.machine-config-container.embedded {
    height: 100%;
    min-height: 360px;
}

.machine-config-container.embedded .machine-table-wrap,
.machine-config-container.embedded .machine-board-wrap {
    min-height: 280px;
}

.machine-list {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
}

.list-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 14px;
    flex-shrink: 0;
}

.list-header-spacer {
    flex: 0;
}

.header-actions {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
    justify-content: flex-end;
    min-width: 0;
}

.filter-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
    padding: 6px 10px;
    border-radius: 10px;
    background: color-mix(in srgb, var(--el-fill-color-light, #f5f7fa) 88%, transparent);
    border: 1px solid color-mix(in srgb, var(--el-border-color-lighter, #ebeef5) 80%, transparent);
}

.filter-icon {
    color: var(--el-text-color-secondary);
}

.list-search {
    width: 160px;
}

.import-group-select {
    width: 128px;
}

.import-account-select {
    width: 140px;
}

.filter-bar :deep(.el-input__wrapper),
.filter-bar :deep(.el-select__wrapper) {
    box-shadow: none !important;
    background: transparent;
    border-radius: 6px;
}

.filter-bar :deep(.el-input__wrapper:hover),
.filter-bar :deep(.el-select__wrapper:hover),
.filter-bar :deep(.el-input__wrapper.is-focus),
.filter-bar :deep(.el-select__wrapper.is-focused) {
    background: var(--el-bg-color, #fff);
    box-shadow: 0 0 0 1px var(--el-border-color, #dcdfe6) inset !important;
}

.toolbar-ops {
    flex-shrink: 0;
}

.machine-table-wrap {
    flex: 1;
    min-height: 0;
    max-height: 100%;
    overflow: auto;
}

.view-mode-toggle {
    margin-right: 2px;
}

.view-mode-toggle :deep(.el-radio-button__inner) {
    padding: 5px 9px;
    display: inline-flex;
    align-items: center;
}

.machine-board-wrap {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
    overflow: hidden;
}

.board-hint {
    margin: 0;
    flex-shrink: 0;
    font-size: 12px;
    color: var(--el-text-color-secondary, #909399);
}

.machine-board {
    flex: 1;
    min-height: 0;
    display: flex;
    gap: 10px;
    overflow-x: auto;
    overflow-y: hidden;
    padding-bottom: 4px;
}

.board-column {
    flex: 1 0 220px;
    min-width: 220px;
    max-width: 280px;
    min-height: 0;
    display: flex;
    flex-direction: column;
    border: 1px solid var(--el-border-color-lighter, var(--app-border));
    border-radius: 8px;
    background: color-mix(in srgb, var(--el-fill-color-light, #f5f7fa) 70%, transparent);
    transition: border-color 0.15s ease, background 0.15s ease;
}

.board-column.is-drop-target {
    border-color: var(--el-color-primary, #409eff);
    background: color-mix(in srgb, var(--el-color-primary, #409eff) 8%, transparent);
}

.board-column-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 10px 12px 8px;
    flex-shrink: 0;
}

.board-column-title {
    font-size: 13px;
    font-weight: 650;
    color: var(--el-text-color-primary, #303133);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.board-column-body {
    flex: 1;
    min-height: 120px;
    overflow-y: auto;
    padding: 0 8px 8px;
    display: flex;
    flex-direction: column;
    gap: 6px;
}

.board-card {
    position: relative;
    display: grid;
    grid-template-columns: 32px minmax(0, 1fr);
    align-items: center;
    gap: 8px;
    padding: 10px 12px 10px 10px;
    border-radius: var(--app-radius-md, 8px);
    border: 1px solid var(--app-border, #ebeef5);
    background: var(--app-card-bg, #fff);
    cursor: grab;
    user-select: none;
    transition: background 0.12s ease, border-color 0.12s ease;
}

.board-card:hover {
    background: var(--app-accent-bg);
    border-color: color-mix(in srgb, var(--app-accent-color) 35%, var(--app-border));
}

.ml-machine-emoji {
    font-size: 14px;
    line-height: 1;
}

.jump-chain-order {
    margin-top: 8px;
    display: grid;
    gap: 4px;
}

.jump-chain-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 8px;
    border-radius: 6px;
    background: var(--app-panel-bg, #f5f7fa);
}

.jump-chain-idx {
    width: 18px;
    height: 18px;
    border-radius: 50%;
    background: var(--app-accent-color, #409eff);
    color: #fff;
    font-size: 11px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
}

.jump-chain-name {
    flex: 1;
    font-size: 12px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.board-card:hover .ml-machine-icon {
    color: var(--app-accent-color);
    background: color-mix(in srgb, var(--app-accent-color) 16%, transparent);
}

.board-card:active {
    cursor: grabbing;
}

.board-card.is-dragging {
    opacity: 0.45;
}

.board-card-main {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
    overflow: hidden;
}

.board-card-main :deep(.el-tooltip__trigger),
.board-card-main :deep(.text-overflow-tooltip) {
    display: block;
    min-width: 0;
    max-width: 100%;
}

.board-card-main :deep(.ml-name),
.board-card-main :deep(.ml-addr) {
    max-width: 100%;
}

.ml-line :deep(.el-tooltip__trigger),
.ml-line :deep(.el-tooltip) {
    flex: 1;
    min-width: 0;
}

.ml-line :deep(.ml-name) {
    flex: 1;
    min-width: 0;
}

/* 操作浮在右侧，默认不占文本宽度；悬停再显现 */
.board-card-actions {
    position: absolute;
    right: 6px;
    top: 50%;
    transform: translateY(-50%);
    z-index: 1;
    display: flex;
    align-items: center;
    gap: 0;
    padding: 2px;
    border-radius: 6px;
    background: color-mix(in srgb, var(--app-card-bg, #fff) 92%, transparent);
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.12);
    opacity: 0;
    pointer-events: none;
    transition: opacity 0.12s ease;
}

.board-card:hover .board-card-actions,
.board-card.is-context-target .board-card-actions {
    opacity: 1;
    pointer-events: auto;
}

.board-card:hover .board-card-main,
.board-card.is-context-target .board-card-main {
    padding-right: 78px;
}

.board-empty {
    margin: 4px 0;
    padding: 18px 8px;
    border: 1px dashed var(--el-border-color, #dcdfe6);
    border-radius: 6px;
    text-align: center;
    font-size: 12px;
    color: var(--el-text-color-placeholder, #a8abb2);
}

.key-file-input {
    display: flex;
    gap: 8px;
    width: 100%;
    align-items: center;
}

.key-file-input .el-input {
    flex: 1;
    min-width: 0;
}

.field-hint {
    margin: 6px 0 0;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1.4;
}

.tunnel-top-hint {
    margin-bottom: 8px;
}

.tunnel-empty {
    margin: 0;
    padding: 12px 0;
    text-align: center;
    font-size: 12px;
    color: var(--app-text-muted, #909399);
}

.tunnel-row {
    display: flex;
    flex-wrap: nowrap;
    align-items: center;
    gap: 6px;
    margin-bottom: 8px;
    overflow-x: auto;
}

.tunnel-row > * {
    flex-shrink: 0;
}

.group-add-row {
    display: flex;
    gap: 8px;
    margin-bottom: 12px;
}
</style>
