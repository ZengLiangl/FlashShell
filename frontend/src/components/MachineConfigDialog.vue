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
                            <el-icon :size="16"><Monitor /></el-icon>
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
                                    <el-icon :size="16"><Monitor /></el-icon>
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
            width="600px"
            class="settings-sub-dialog"
            append-to-body
        >
            <el-form :model="machineForm" :rules="machineRules" ref="machineFormRef" label-width="100px">
                <el-form-item label="机器名称" prop="name">
                    <el-input v-model="machineForm.name" placeholder="请输入机器名称" />
                </el-form-item>

                <el-form-item label="分组" prop="group">
                    <el-select
                        v-model="machineForm.group"
                        clearable
                        filterable
                        allow-create
                        default-first-option
                        placeholder="选择或输入分组，留空为默认分组"
                        style="width: 100%"
                    >
                        <el-option
                            v-for="g in groupOptions"
                            :key="g"
                            :label="g"
                            :value="g === DEFAULT_MACHINE_GROUP ? '' : g"
                        />
                    </el-select>
                </el-form-item>

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

                <el-form-item label="密钥文件" prop="key_file">
                    <div class="key-file-input">
                        <el-input v-model="machineForm.key_file" placeholder="请选择密钥文件" readonly />
                        <el-tooltip content="选择文件" placement="top">
                            <el-button type="primary" circle @click="selectKeyFile">
                                <el-icon><Folder /></el-icon>
                            </el-button>
                        </el-tooltip>
                    </div>
                </el-form-item>

                <el-divider content-position="left">连接信息</el-divider>

                <el-form-item label="主机地址" prop="host">
                    <el-input v-model="machineForm.host" placeholder="请输入主机地址" />
                </el-form-item>

                <el-form-item label="端口" prop="port">
                    <el-input-number v-model="machineForm.port" :min="1" :max="65535" placeholder="SSH端口" />
                </el-form-item>

                <el-form-item label="用户名" prop="user">
                    <el-input v-model="machineForm.user" placeholder="请输入用户名" />
                </el-form-item>

                <el-form-item label="密码" prop="password">
                    <el-input v-model="machineForm.password" type="password" placeholder="请输入密码（可选）" show-password />
                </el-form-item>

                <el-form-item label="跳板机">
                    <el-input v-model="machineForm.proxyJump" placeholder="单跳：机器名或 host[:port]（多跳请用下方跳板链）" clearable />
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
                        placeholder="按顺序选择或输入跳板（优先于单跳跳板机）"
                        style="width: 100%"
                    >
                        <el-option
                            v-for="m in machines"
                            :key="m.id || m.name"
                            :label="m.name"
                            :value="m.name"
                        />
                    </el-select>
                    <p class="field-hint">多跳时按从左到右顺序连接，最后一跳再连目标主机</p>
                </el-form-item>

                <el-divider content-position="left">每主机代理</el-divider>
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
                        <el-input-number v-model="machineForm.proxyOverride.port" :min="1" :max="65535" />
                    </el-form-item>
                    <el-form-item label="代理用户">
                        <el-input v-model="machineForm.proxyOverride.user" clearable />
                    </el-form-item>
                    <el-form-item label="代理密码">
                        <el-input v-model="machineForm.proxyOverride.password" type="password" show-password clearable />
                    </el-form-item>
                </template>

                <el-divider content-position="left">高级选项</el-divider>
                <el-form-item label="旧设备算法">
                    <el-switch v-model="machineForm.legacyAlgorithms" active-text="启用兼容算法" />
                </el-form-item>
                <el-form-item label="跳过 ECDSA 主机密钥">
                    <el-switch v-model="machineForm.skipEcdsaHostKey" />
                </el-form-item>
                <el-form-item label="SFTP 编码">
                    <el-select v-model="machineForm.sftpEncoding" style="width: 100%">
                        <el-option label="自动" value="auto" />
                        <el-option label="UTF-8" value="utf-8" />
                        <el-option label="GB18030" value="gb18030" />
                    </el-select>
                </el-form-item>
                <el-form-item label="文件协议">
                    <el-select v-model="machineForm.sftpFileProtocol" style="width: 100%">
                        <el-option label="自动（SFTP 优先，失败回退 SCP）" value="auto" />
                        <el-option label="仅 SFTP" value="sftp" />
                        <el-option label="仅 SCP" value="scp" />
                    </el-select>
                    <p class="field-hint">远端无 SFTP 子系统时可用 SCP 回退完成浏览与传输</p>
                </el-form-item>
                <el-form-item label="启动命令">
                    <el-input v-model="machineForm.startupCommand" placeholder="连接后自动执行（单行）" clearable />
                </el-form-item>
                <el-form-item label="Agent 转发">
                    <el-switch v-model="machineForm.agentForwarding" active-text="启用 SSH Agent 转发" />
                </el-form-item>

                <el-divider content-position="left">SSH 隧道</el-divider>
                <div class="tunnel-head">
                    <span class="tunnel-hint">连接成功后自动建立；本地转发：本机端口 → 远端地址</span>
                    <el-button size="small" text type="primary" class="tunnel-add-btn" @click="addTunnel">
                        <el-icon><Plus /></el-icon>
                        添加隧道
                    </el-button>
                </div>
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
                <el-table-column label="操作" width="100" align="center">
                    <template #default="{ row }">
                        <div class="icon-actions">
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
    Plus, Search, FolderOpened, Upload, Edit, Delete, Connection, Folder, List, Grid, Close, Check, Monitor, VideoPlay,
} from '@element-plus/icons-vue'
import {
    GetMachines,
    GetMachineGroups,
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
    ExportMachineTemplateToFile,
    ImportMachineTemplateFromFile,
} from '../../wailsjs/go/app/App'
import {
    DEFAULT_MACHINE_GROUP,
    sortMachinesByName,
    machineMatchesKeyword,
    getMachineGroup,
    formatMachineAddr,
} from '../utils/machineGroups'
import { copyMachineRecord } from '../utils/machineCopy'
import { useMachineContextMenu } from '../composables/useMachineContextMenu'
import MachineContextMenu from './shell/MachineContextMenu.vue'
import TextOverflowTooltip from './TextOverflowTooltip.vue'

export default {
    name: 'MachineConfigDialog',
    components: {
        Plus, Search, FolderOpened, Upload, Edit, Delete, Connection, Folder, List, Grid, Close, Check, Monitor, VideoPlay,
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
            key_file: '',
            host: '',
            port: 22,
            user: '',
            password: '',
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
            startupCommand: '',
            agentForwarding: false,
            tunnels: [],
        })

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

        const loadGroups = async () => {
            try {
                machineGroups.value = await GetMachineGroups() || []
            } catch {
                machineGroups.value = []
            }
        }

        const loadMachines = async () => {
            try {
                machinesLoading.value = true
                const machinesData = await GetMachines()
                machines.value = machinesData || []
                await loadGroups()
            } catch (error) {
                console.error('加载机器配置失败:', error)
                ElMessage.error('加载机器配置失败: ' + error.message)
            } finally {
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
            if (!accountId) return
            const account = globalAccounts.value.find((item) => item.id === accountId)
            if (!account) return
            machineForm.user = account.user || ''
            machineForm.password = account.password || ''
        }

        const addMachine = () => {
            editingMachine.value = null
            selectedAccountId.value = ''
            resetMachineForm()
            machineEditVisible.value = true
        }

        const editMachine = async (machine) => {
            editingMachine.value = machine
            selectedAccountId.value = ''
            machineForm.name = machine.name
            machineForm.group = machine.group || ''
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
            machineForm.startupCommand = machine.startupCommand || ''
            machineForm.agentForwarding = !!machine.agentForwarding
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
                }
            } catch (error) {
                console.error('获取敏感数据失败:', error)
                ElMessage.warning('获取敏感数据失败，请重新输入')
            }
            machineEditVisible.value = true
        }

        const activate = async () => {
            await loadMachines()
            await loadGlobalAccounts()
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
            else emit('closed')
        }, { immediate: true })

        const resetMachineForm = () => {
            machineForm.name = ''
            machineForm.group = ''
            machineForm.key_file = ''
            machineForm.host = ''
            machineForm.port = 22
            machineForm.user = ''
            machineForm.password = ''
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
            machineForm.startupCommand = ''
            machineForm.agentForwarding = false
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
                savingMachine.value = true
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
                    key_file: machineForm.key_file,
                    proxyJump: machineForm.proxyJump?.trim() || '',
                    jumpChain: (machineForm.jumpChain || []).map((s) => String(s).trim()).filter(Boolean),
                    proxyOverride: proxyOverride.mode === 'inherit' ? null : proxyOverride,
                    legacyAlgorithms: machineForm.legacyAlgorithms,
                    skipEcdsaHostKey: machineForm.skipEcdsaHostKey,
                    sftpEncoding: machineForm.sftpEncoding || 'auto',
                    sftpFileProtocol: machineForm.sftpFileProtocol || 'auto',
                    startupCommand: machineForm.startupCommand?.trim() || '',
                    agentForwarding: machineForm.agentForwarding,
                    tunnels: (machineForm.tunnels || []).filter((t) => t.localPort > 0),
                }
                const sensitiveData = {
                    host: machineForm.host,
                    port: machineForm.port,
                    user: machineForm.user,
                    password: machineForm.password
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
                ElMessage.error('保存机器配置失败: ' + error.message)
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
                testingDraft.value = true
                await TestMachineDraftConnection(
                    {
                        name: machineForm.name || 'draft-test',
                        group: normalizeGroup(machineForm.group),
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
                if (error === false || error?.fields) return
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

        const handleAddCommand = (command) => {
            if (command === 'import-finalshell') importFinalShell()
            else if (command === 'import-xshell') importXshell()
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
            managedGroups,
            machineKeyword,
            globalAccounts,
            machinesLoading,
            machineEditVisible,
            groupManageVisible,
            newGroupName,
            savingMachine,
            testingDraft,
            editingMachine,
            machineFormRef,
            machineForm,
            machineRules,
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
}

.field-hint {
    margin: 4px 0 0;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1.4;
}

.tunnel-head {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 10px;
    flex-wrap: wrap;
}

.tunnel-hint {
    margin: 0;
    font-size: 12px;
    color: var(--app-text-muted);
    line-height: 1.4;
}

.tunnel-add-btn {
    padding: 0;
    height: auto;
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
