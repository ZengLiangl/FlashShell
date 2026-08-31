<template>
  <div class="app-container app" :class="themeClass">
    <AppMenuBar
      v-show="activeView !== 'shell'"
      :active-view="activeView"
      :has-projects="projects.length > 0"
      :has-task="!!selectedProject"
      :task-running="status.isRunning"
      :connected-count="connectedCount"
      :open-session-count="openSessionCount"
      @change-view="switchActiveView"
      @open-config-editor="configEditorVisible = true"
      @refresh="refreshConfig"
    />

    <!-- 全局加载遮罩 -->
    <div v-if="isReloading" class="global-loading">
      <div class="loading-content">
        <el-icon class="loading-icon">
          <Loading />
        </el-icon>
        <div class="loading-text">正在重新加载...</div>
      </div>
    </div>

    <div
      v-show="activeView === 'shell'"
      id="shell-top-chrome-host"
      class="shell-top-chrome-host"
    />

    <div class="app-body">
      <AppRail
        :active-view="activeView"
        :has-projects="projects.length > 0"
        :open-session-count="openSessionCount"
        :connected-count="connectedCount"
        :show-audit="mcpServiceEnabled"
        @change-view="switchActiveView"
        @open-settings="(section) => openSettingsHub(section || 'general')"
      />

      <!-- 首页：任务项目 + 主机单页 -->
      <main v-show="activeView === 'home'" class="view home active">
        <HomePage
          ref="homePageRef"
          :projects="projects"
          :machines="shellMachines"
          :connected-count="connectedCount"
          :has-task="!!selectedProject"
          :task-running="status.isRunning"
          :connecting-name="connectingName"
          :sessions="shellSessions"
          :workspace-sessions="workspaceSessions"
          @refresh="refreshConfig"
          @select-project="selectProject"
          @resume-task="resumeTaskView"
          @open-shell="enterShellMode"
          @connect-machine="openShellAndConnect"
          @focus-session="onQuickFocusSession"
          @add-machine="openShellMachineDialog"
          @edit-machine="openShellMachineEdit"
          @copy-machine="copyShellMachine"
          @delete-machine="deleteShellMachine"
          @open-config-editor="configEditorVisible = true"
        />
      </main>

      <!-- 任务详情（v-show 保活） -->
      <main
        v-if="selectedProject"
        v-show="activeView === 'task'"
        class="view task active"
      >
        <aside
          class="task-left left-panel"
          :class="{ resizing: isResizing }"
          :style="{ flexBasis: leftPanelWidth + 'px', width: leftPanelWidth + 'px' }"
        >
          <div class="resize-handle" @mousedown="startResize" />
          <SubProjectList
            :selected-project="selectedProject"
            :projects="projects"
            :sub-projects="subProjects"
            :expanded-sub-projects="expandedSubProjects"
            :expanded-commands="expandedCommands"
            :status="status"
            :get-command-tag-type="getCommandTagType"
            :get-command-type-text="getCommandTypeText"
            :is-sub-project-running="isSubProjectRunning"
            @toggle-sub="toggleSubProject"
            @toggle-cmd="toggleCommand"
            @execute-sub="executeSubProject"
            @execute-cmd="executeCommand"
            @stop-sub="stopSubProject"
            @dry-run-sub="dryRunSubProject"
            @select-project="selectProject"
          />
        </aside>

        <div class="task-right">
          <TerminalHeader
            :show-back="false"
            :search-visible="terminalSearchVisible"
            v-model:search-query="terminalSearchQuery"
            :match-summary="terminalMatchSummary"
            :show-chrome="false"
            :show-search-toggle="false"
            :show-inline-actions="false"
            :status-running="status.isRunning"
            active-view="task"
            @toggle-search="toggleTerminalSearch"
            @search-next="gotoNextSearchMatch"
            @search-prev="gotoPrevSearchMatch"
            @close-search="closeTerminalSearch"
          />
          <TerminalOutput
            ref="terminalOutputRef"
            :status="status"
            :output-lines="outputLines"
            :progress-percentage="progressPercentage"
            :progress-status="progressStatus"
            :search-query="terminalSearchQuery"
            :active-match-index="terminalActiveMatchIndex"
            :remote-failure="lastRemoteFailure"
            :show-inline-actions="true"
            @clear="clearOutput"
            @refresh="refreshOutput"
            @search-matches="handleSearchMatches"
            @open-failure-shell="openFailureShell"
          />
          <StatusBar
            :status="status"
            :selected-project="selectedProject"
            :app-info="statusBarInfo"
            :progress-percentage="progressPercentage"
            :remote-failure="lastRemoteFailure"
            @stop-all="stopAllCommands"
            @open-failure-shell="openFailureShell"
          />
        </div>
      </main>

      <main
        v-if="activeView === 'audit'"
        class="view audit active"
      >
        <AuditLogView />
      </main>

      <!-- Shell 视图 -->
      <main v-show="activeView === 'shell'" class="view shell active">
        <ShellWorkspace
          ref="shellWorkspaceRef"
          v-if="shellMounted"
          hide-app-chrome
          :active="activeView === 'shell'"
          :block-shortcuts="settingsHubVisible"
          :left-panel-width="leftPanelWidth"
          :is-resizing="isResizing"
          :app-info="statusBarInfo"
          :machines="shellMachines"
          :sessions="shellSessions"
          :workspace-sessions="workspaceSessions"
          :connected-count="connectedCount"
          :open-session-count="openSessionCount"
          v-model:active-machine="activeMachine"
          :connecting-name="connectingName"
          :testing-name="testingName"
          :broadcast-enabled="broadcastEnabled"
          :broadcast-targets="broadcastTargets"
          :split-session-ids="splitSessionIds"
          :has-task="!!selectedProject"
          :has-projects="projects.length > 0"
          :has-machines="shellMachines.length > 0"
          :task-running="status.isRunning"
          :projects="projects"
          :selected-project-name="selectedProject?.name || ''"
          @back="leaveShellMode"
          @connect="(name) => connectShell(name)"
          @disconnect="disconnectShell"
          @close-session="closeShellSession"
          @close-sessions="closeShellSessions"
          @duplicate-session="(id) => duplicateShellSession(id)"
          @reconnect="(name) => connectOrReconnectShell(name)"
          @add-local="() => connectLocalShell()"
          @add-local-command="(cmd) => connectLocalShell('', cmd)"
          @open-window="onOpenMachineWindow"
          @focus-session="onQuickFocusSession"
          @connect-machines="onQuickConnectMachines"
          @test="testShellConnection"
          @update:broadcast-enabled="(v) => (broadcastEnabled = v)"
          @update:broadcast-targets="(v) => (broadcastTargets = v)"
          @update:split-session-ids="(v) => (splitSessionIds = v)"
          @reorder-tabs="({ from, to }) => reorderTabs(from, to)"
          @cwd-sync="({ machineName, cwd }) => updateTabLastCwd(machineName, cwd)"
          @add-machine="openShellMachineDialog"
          @edit-machine="openShellMachineEdit"
          @copy-machine="copyShellMachine"
          @delete-machine="deleteShellMachine"
          @start-resize="startResize"
          @machines-changed="onMachinesChanged"
          @change-view="switchActiveView"
          @select-project="selectProject"
        />
      </main>
    </div>

    <McpApprovalDialog />
    <VaultUnlockOverlay />

    <SettingsHubDialog v-model="settingsHubVisible" :initial-section="settingsSection" :edit-machine-id="machineEditId"
      @machines-changed="onMachinesChanged" @machines-closed="machineEditId = ''"
      @connect-machine="onSettingsConnectMachine" />

    <MachineAsidePanel
      :open="machineAsideOpen"
      :machine="machineAsideMachine"
      :machines="shellMachines"
      @close="machineAsideOpen = false"
      @saved="onMachinesChanged"
      @connect="onAsideConnect"
    />

    <!-- 关于弹框 -->
    <AboutDialog v-model="aboutVisible" :intro-html="aboutIntroHtml" :prompt-mode="aboutPromptMode"
      :initial-update-result="aboutInitialUpdate" @dismissed="onAboutPromptDismissed" @skipped="onAboutPromptSkipped" />

    <ConfigEditorDialog v-model="configEditorVisible" @saved="refreshProjectConfig" />

    <HostKeyTrustDialog v-model="hostKeyDialogVisible" :host-key-info="pendingHostKey" @trusted="onHostKeyTrusted" />
  </div>
</template>

<script>
import { ref, reactive, computed, watch, onMounted, onUnmounted, nextTick, defineAsyncComponent } from "vue";
import { ElMessage, ElMessageBox, ElNotification } from "element-plus";
import { h } from "vue";
import * as App from "../wailsjs/go/app/App";
import { EventsOn, EventsOff } from "../wailsjs/runtime/runtime";
import Convert from "ansi-to-html";
import TerminalOutput from "./components/TerminalOutput.vue";
import StatusBar from "./components/StatusBar.vue";
import ProjectList from "./components/ProjectList.vue";
import HomePage from "./components/HomePage.vue";
import AuditLogView from "./components/AuditLogView.vue";
import McpApprovalDialog from "./components/McpApprovalDialog.vue";
import VaultUnlockOverlay from "./components/VaultUnlockOverlay.vue";
import { useShell } from "./composables/useShell";
import SubProjectList from "./components/SubProjectList.vue";
import TerminalHeader from "./components/TerminalHeader.vue";
import AppMenuBar from "./components/AppMenuBar.vue";
import AppRail from "./components/layout/AppRail.vue";
import { useTheme } from "./composables/useTheme";
import { mergeShortcuts, matchesShortcut, isFormFieldTarget, isXtermInput } from "./utils/shortcuts";
import {
  clampShellFontSize,
  SHELL_FONT_SIZE_DEFAULT,
} from "./utils/shellTerminalUx";
import { hasOverlayAboveSettingsHub } from "./utils/dialogOverlay";
import { setCachedUpdateCheck, isUsableUpdateResult } from "./utils/updateCheckCache";
import { TASK_OUTPUT_MAX_LINES, clampTaskOutputMaxLines } from "./constants/shellMemory";
import { copyMachineRecord } from "./utils/machineCopy";
import {
  ensureShellAsciiInputListeners,
  notifyLeaveShellMode,
  setShellAsciiInputEnabled,
} from "./utils/shellAsciiInput";

function sendOsNotification(opts) {
  const rt = typeof window !== "undefined" ? window.runtime : null;
  if (!rt || typeof rt.IsNotificationAvailable !== "function" || typeof rt.SendNotification !== "function") {
    return;
  }
  Promise.resolve(rt.IsNotificationAvailable())
    .then((ok) => {
      if (!ok) return;
      return rt.SendNotification(opts);
    })
    .catch(() => {});
}

const ShellWorkspace = defineAsyncComponent(() => import("./views/ShellWorkspace.vue"));
const AboutDialog = defineAsyncComponent(() => import("./components/AboutDialog.vue"));
const ConfigEditorDialog = defineAsyncComponent(() => import("./components/ConfigEditorDialog.vue"));
const SettingsHubDialog = defineAsyncComponent(() => import("./components/SettingsHubDialog.vue"));
const HostKeyTrustDialog = defineAsyncComponent(() => import("./components/shell/HostKeyTrustDialog.vue"));
const MachineAsidePanel = defineAsyncComponent(() => import("./components/MachineAsidePanel.vue"));

export default {
  name: "App",
  components: { AppMenuBar, AppRail, TerminalOutput, StatusBar, ProjectList, HomePage, AuditLogView, McpApprovalDialog, VaultUnlockOverlay, ShellWorkspace, SubProjectList, TerminalHeader, AboutDialog, ConfigEditorDialog, SettingsHubDialog, HostKeyTrustDialog, MachineAsidePanel },
  setup() {
    const { isDark, themeMode, terminalPreset, shellFontSize, loadTheme, applyThemeSettings, saveTheme } = useTheme();
    const projects = ref([]);
    const subProjects = ref([]);
    const outputLines = ref([]);
    const lastRemoteFailure = ref(null);
    const selectedProject = ref(null);
    const selectedSubProject = ref(null);
    const isReloading = ref(false);
    const status = reactive({
      isRunning: false,
      command: "",
      currentStep: "",
      completedSteps: 0,
      totalSteps: 0,
    });
    // 终端输出行数上限，防止内存增长过快（可由系统设置覆盖）
    const maxOutputLines = ref(TASK_OUTPUT_MAX_LINES);

    const enforceOutputLimit = () => {
      const overflow = outputLines.value.length - maxOutputLines.value;
      if (overflow > 0) {
        outputLines.value.splice(0, overflow);
      }
    };

    const applyTaskOutputMaxLines = (n) => {
      maxOutputLines.value = clampTaskOutputMaxLines(n);
      enforceOutputLimit();
    };

    const loadTaskOutputLimit = async () => {
      try {
        const cfg = await App.GetSystemSettings();
        applyTaskOutputMaxLines(cfg?.taskOutputMaxLines);
        setShellAsciiInputEnabled(cfg?.shellAsciiInput !== false);
      } catch {
        // 保持默认
      }
    };

    // 左侧面板宽度控制
    const leftPanelWidth = ref(256);
    const minPanelWidth = 180;
    const maxPanelWidth = 640;
    const isResizing = ref(false);

    // 展开状态管理
    const expandedSubProjects = ref({});
    const expandedCommands = ref({});

    // 统一系统设置 Hub
    const settingsHubVisible = ref(false);
    const settingsSection = ref('general');
    const mcpServiceEnabled = ref(false);
    const machineEditId = ref('');
    const machineAsideOpen = ref(false);
    const machineAsideMachine = ref(null);
    const machines = ref([]);
    const machinesLoading = ref(false);

    // 关于弹框 / 更新提示
    const aboutVisible = ref(false);
    const aboutIntroHtml = ref('');
    const aboutPromptMode = ref(false);
    const aboutInitialUpdate = ref(null);
    const updatePromptDismissedThisSession = ref(false);
    const configEditorVisible = ref(false);
    let approvalNotif = null;
    let approvalPendingCount = 0;
    let offApprovalQueued = null;
    let offApprovalResolved = null;
    // home | task | shell —— 任务与 Shell 互不打断，仅切换当前视图
    const activeView = ref('home');
    const shellMounted = ref(false);
    const shellMode = computed(() => activeView.value === 'shell');
    const shellWorkspaceRef = ref(null);
    const homePageRef = ref(null);
    const {
      sessions: shellSessions,
      activeMachine,
      shellMachines,
      connectingName,
      testingName,
      workspaceSessions,
      connectedSessions,
      connectedCount,
      openSessionCount,
      syncSessions,
      loadMachines: loadShellMachines,
      ensureShellReady,
      connect: connectShell,
      connectLocal: connectLocalShell,
      connectOrReconnect: connectOrReconnectShell,
      duplicateSession: duplicateShellSession,
      disconnect: disconnectShell,
      closeSession: closeShellSession,
      closeSessions: closeShellSessions,
      pendingHostKey,
      testMachine: testShellConnection,
      broadcastEnabled,
      broadcastTargets,
      splitSessionIds,
      toggleBroadcastTarget,
      setSplitSessions,
      toggleSplitSession,
      reorderTabs,
      updateTabLastCwd,
    } = useShell();
    const hostKeyDialogVisible = computed({
      get: () => !!pendingHostKey.value,
      set: (v) => { if (!v) pendingHostKey.value = null },
    });
    const onHostKeyTrusted = async () => {
      const hk = pendingHostKey.value;
      pendingHostKey.value = null;
      if (!hk) return;
      if (hk.configName) {
        await connectShell(hk.configName);
      } else if (hk.sessionId) {
        await connectOrReconnectShell(hk.sessionId);
      }
    };
    const terminalOutputRef = ref(null);
    const terminalSearchVisible = ref(false);
    const terminalSearchQuery = ref('');
    const terminalMatchIndices = ref([]);
    const terminalActiveMatchIndex = ref(-1);
    const sessionId = ref('');
    const appVersion = ref('');
    const shortcutMap = ref(mergeShortcuts());

    const loadShortcutMap = async () => {
      try {
        shortcutMap.value = mergeShortcuts(await App.GetShortcutSettings());
      } catch {
        shortcutMap.value = mergeShortcuts();
      }
    };
    const statusBarInfo = computed(() => {
      const ver = appVersion.value || '…';
      const base = ``;
      // 右下角暂不展示会话 ID
      // if (!sessionId.value) return base;
      // return `${base} · 会话 ${sessionId.value.slice(0, 8)}`;
      return base;
    });

    const themeClass = computed(() => ({
      'theme-dark': isDark.value,
      [`terminal-preset-${terminalPreset.value || 'classic'}`]: true,
    }));

    const terminalMatchSummary = computed(() => {
      const total = terminalMatchIndices.value.length;
      if (!terminalSearchQuery.value.trim() || total === 0) return total ? `0/${total}` : '0/0';
      const current = terminalActiveMatchIndex.value >= 0 ? terminalActiveMatchIndex.value + 1 : 0;
      return `${current}/${total}`;
    });

    const openAbout = () => {
      aboutPromptMode.value = false;
      aboutInitialUpdate.value = null;
      aboutVisible.value = true;
    };

    const normalizeVer = (v) => String(v || '').trim().replace(/^v/i, '').toLowerCase();

    const maybePromptUpdateOnHome = async () => {
      if (updatePromptDismissedThisSession.value) return;
      if (aboutVisible.value) return;
      try {
        const result = await App.CheckForUpdates();
        if (isUsableUpdateResult(result)) {
          setCachedUpdateCheck(result);
        }
        if (!result?.hasUpdate) return;
        const skipped = await App.GetSkippedUpdateVersion();
        if (skipped && normalizeVer(skipped) === normalizeVer(result.latestVersion)) {
          return;
        }
        aboutPromptMode.value = true;
        aboutInitialUpdate.value = result;
        aboutVisible.value = true;
      } catch {
        // 静默失败：不影响首页使用
      }
    };

    const onAboutPromptDismissed = () => {
      // 仅关闭：本会话内不再弹，下次启动软件再检查
      updatePromptDismissedThisSession.value = true;
      aboutPromptMode.value = false;
      aboutInitialUpdate.value = null;
    };

    const onAboutPromptSkipped = () => {
      updatePromptDismissedThisSession.value = true;
      aboutPromptMode.value = false;
      aboutInitialUpdate.value = null;
    };

    const workPathEditVisible = ref(false);
    const workPaths = ref({});
    const workPathsLoading = ref(false);
    const savingWorkPath = ref(false);
    const editingWorkPath = ref(null);
    const workPathFormRef = ref(null);

    const machineForm = reactive({
      name: '',
      key_file: '',
      host: '',
      port: 22,
      user: '',
      password: ''
    });

    const machineRules = {
      name: [
        { required: true, message: '请输入机器名称', trigger: 'blur' }
      ],
      host: [
        { required: true, message: '请输入主机地址', trigger: 'blur' }
      ],
      port: [
        { required: true, message: '请输入端口', trigger: 'blur' }
      ],
      user: [
        { required: true, message: '请输入用户名', trigger: 'blur' }
      ]
    };

    const workPathForm = reactive({
      key: '',
      value: ''
    });

    const workPathRules = {
      key: [
        { required: true, message: '请输入变量名', trigger: 'blur' },
      ],
      value: [
        { required: true, message: '请输入变量值', trigger: 'blur' }
      ]
    };

    // 创建 ANSI 转换器
    const convert = new Convert({
      fg: '#d4d4d4',
      bg: '#1e1e1e',
      newline: false,
      escapeXML: true,
      stream: false
    });

    const mapProjectSummaries = (summaries) =>
      (summaries || []).map((item) => ({
        name: item.name,
        description: item.description,
        subprojects: Array.from({ length: item.subProjectCount || 0 }),
      }));

    // 加载配置（首页仅摘要，完整项目在 selectProject 时加载）
    const loadConfig = async () => {
      try {
        console.log("开始加载配置...");
        const summaries = await App.GetProjectSummaries();
        projects.value = mapProjectSummaries(summaries);
        console.log("设置的项目数据:", projects.value);
        console.log("项目数量:", projects.value.length);

        selectedProject.value = null;
        subProjects.value = [];
      } catch (error) {
        console.error("加载配置失败:", error);
        console.error("错误详情:", error.stack);
      }
    };

    // 刷新全局配置（从文件读取最新数据到内存）
    const loadGlobalConfigForRefresh = async () => {
      try {
        console.log("开始刷新全局配置...");
        console.log("App 对象:", App);

        const globalConfig = await App.GetGlobalConfigForRefresh();
        console.log("刷新后的全局配置数据:", globalConfig);

        // 这里可以根据需要更新前端相关的全局配置状态
        // 例如：更新最后打开的文件、配置文件列表等

        console.log("全局配置刷新完成");
      } catch (error) {
        console.error("刷新全局配置失败:", error);
        console.error("错误详情:", error.stack);
      }
    };

    // 刷新配置（从文件读取最新数据，不更新全局配置）
    const loadConfigForRefresh = async () => {
      try {
        console.log("开始刷新项目配置...");
        const summaries = await App.GetProjectSummaries();
        projects.value = mapProjectSummaries(summaries);
        console.log("设置的项目数据:", projects.value);
        console.log("项目数量:", projects.value.length);

        selectedProject.value = null;
        subProjects.value = [];

        console.log("项目配置刷新完成");
      } catch (error) {
        console.error("刷新项目配置失败:", error);
        console.error("错误详情:", error.stack);
      }
    };

    const refreshProjectConfig = loadConfigForRefresh;

    // 刷新配置
    const refreshConfig = async () => {
      try {
        isReloading.value = true;
        console.log("开始刷新配置...");

        // 第一步：先从全局配置文件读取最新数据，重置内存中的全局配置
        await loadGlobalConfigForRefresh();

        // 第二步：再刷新项目配置
        await loadConfigForRefresh();
        await loadShellMachines();

        console.log("配置刷新完成");
        ElMessage.success("配置已刷新");
      } catch (error) {
        console.error("刷新配置失败:", error);
        ElMessage.error("刷新配置失败: " + error.message);
      } finally {
        isReloading.value = false;
      }
    };

    // 调试配置
    const debugConfig = async () => {
      console.log("=== 调试信息 ===");
      console.log("当前项目数据:", projects.value);
      console.log("项目数量:", projects.value.length);
      console.log("App 模块:", App);

      try {
        const config = await App.GetConfig();
        console.log("直接调用 GetConfig 结果:", config);
        // alert(`配置加载成功！项目数量: ${config.projects?.length || 0}`);
      } catch (error) {
        console.error("调试失败:", error);
        // alert(`调试失败: ${error.message}`);
      }
    };
    // 选择项目（可与 Shell 会话并行）
    const selectProject = async (project) => {
      try {
        const full = await App.GetProject(project.name);
        selectedProject.value = full;
        selectedSubProject.value = null;
        activeView.value = 'task';

        if (full?.subprojects) {
          subProjects.value = full.subprojects.map(subproject => ({
            ...subproject,
            projectName: full.name,
            commandCount: subproject.commands ? subproject.commands.length : 0,
            stepCount: subproject.commands ? subproject.commands.reduce((total, command) => total + (command.steps?.length || 0), 0) : 0
          }));
        } else {
          subProjects.value = [];
        }

        expandedSubProjects.value = {};
        expandedCommands.value = {};

        console.log(`选择项目: ${full.name}, 找到 ${subProjects.value.length} 个 SubProjects`);
        await scrollTaskOutputToBottom();
      } catch (error) {
        console.error('加载项目详情失败:', error);
        ElMessage.error('加载项目详情失败: ' + (error.message || error));
      }
    };

    const backToProjectList = () => {
      // 回到首页但不销毁任务上下文，可与 Shell 来回切换
      activeView.value = 'home';
    };

    const scrollTaskOutputToBottom = async () => {
      if (!outputLines.value?.length) return;
      await nextTick();
      terminalOutputRef.value?.scrollToBottom?.();
      // 虚拟列表布局后再滚一次，确保触底
      await nextTick();
      requestAnimationFrame(() => {
        terminalOutputRef.value?.scrollToBottom?.();
      });
    };

    const resumeTaskView = async () => {
      if (!selectedProject.value) return;
      if (activeView.value === 'shell') {
        notifyLeaveShellMode();
      }
      activeView.value = 'task';
      await scrollTaskOutputToBottom();
    };

    const enterShellMode = async () => {
      shellMounted.value = true;
      activeView.value = 'shell';
      await ensureShellReady();
      await syncSessions();
    };

    const openConnectionManager = async () => {
      const firstMount = !shellMounted.value;
      await enterShellMode();
      await nextTick();
      if (firstMount) await nextTick();
      // 首次挂载时 ref 可能晚一拍，短重试
      for (let i = 0; i < 5; i++) {
        if (shellWorkspaceRef.value?.openPicker) {
          shellWorkspaceRef.value.openPicker();
          return;
        }
        await nextTick();
      }
    };

    const leaveShellMode = async () => {
      notifyLeaveShellMode();
      activeView.value = 'home';
      try {
        const cfg = await App.GetSystemSettings();
        if (cfg?.themeSettings?.shellMemorySaver) {
          shellMounted.value = false;
        }
      } catch {
        // 读取设置失败时保持现有 v-show 行为
      }
    };

    const onOpenMachineWindow = async (machine) => {
      const name = machine?.name || machine;
      if (!name) return;
      try {
        await App.OpenMachineInNewWindow(name);
        ElMessage.success(`已在新窗口打开 ${name}`);
      } catch (e) {
        ElMessage.error('打开新窗口失败: ' + e);
      }
    };

    /** 顶栏模式切换：任务 ↔ Shell 直达，不强制经首页 */
    const switchActiveView = async (view) => {
      if (view === activeView.value) return;
      if (view === 'home') {
        if (activeView.value === 'shell') {
          await leaveShellMode();
        } else {
          activeView.value = 'home';
        }
        return;
      }
      if (view === 'task') {
        if (!selectedProject.value) {
          if (projects.value.length) {
            await selectProject(projects.value[0]);
            return;
          }
          ElMessage.info('请先在首页选择项目，或悬停「任务」从列表打开');
          return;
        }
        await resumeTaskView();
        return;
      }
      if (view === 'audit') {
        if (!mcpServiceEnabled.value) {
          ElMessage.info('请先开启 MCP 服务');
          openSettingsHub('mcp');
          return;
        }
        if (activeView.value === 'shell') {
          await leaveShellMode();
        }
        activeView.value = view;
        return;
      }
      if (view === 'shell') {
        await enterShellMode();
      }
    };

    const openShellAndConnect = async (machineName) => {
      await enterShellMode();
      await connectShell(machineName);
    };

    /** 快速切换：已有会话 → 聚焦该标签（必要时重连） */
    const onQuickFocusSession = async (sessionId) => {
      if (!sessionId) return;
      await enterShellMode();
      const tab = workspaceSessions.value.find((s) => s.machineName === sessionId);
      if (tab?.connected) {
        activeMachine.value = sessionId;
        return;
      }
      await connectOrReconnectShell(sessionId);
    };

    /** 快速切换：机器配置 → 新建/打开连接 */
    const onQuickConnectMachine = async (machineName) => {
      if (!machineName) return;
      const existing = workspaceSessions.value.find(
        (s) =>
          s.connected &&
          (s.configName === machineName || s.machineName === machineName || s.tabLabel === machineName),
      );
      if (existing) {
        await onQuickFocusSession(existing.machineName);
        return;
      }
      await openShellAndConnect(machineName);
    };

    /** 错峰批量连接（150–300ms 间隔）；已连接则聚焦，不重复开会话 */
    const onQuickConnectMachines = async (names) => {
      const list = Array.isArray(names) ? names.filter(Boolean) : []
      if (!list.length) return
      await enterShellMode()
      for (let i = 0; i < list.length; i++) {
        const name = list[i]
        try {
          await onQuickConnectMachine(name)
        } catch (e) {
          console.error('批量连接失败:', name, e)
        }
        if (i < list.length - 1) {
          const delay = 150 + Math.floor(Math.random() * 151)
          await new Promise((r) => setTimeout(r, delay))
        }
      }
    };

    const onSettingsConnectMachine = async (machineName) => {
      machineEditId.value = '';
      settingsHubVisible.value = false;
      await openShellAndConnect(machineName);
    };

    const openSettingsHub = (section = 'general') => {
      settingsSection.value = section || 'general';
      settingsHubVisible.value = true;
    };

    const refreshMcpNavVisibility = async () => {
      try {
        const st = (await App.GetMCPStatus()) || {};
        mcpServiceEnabled.value = !!(st.enabled || st.online);
      } catch {
        mcpServiceEnabled.value = false;
      }
    };

    watch(mcpServiceEnabled, (enabled) => {
      if (!enabled && activeView.value === 'audit') {
        activeView.value = 'home';
      }
    });

    const openShellMachineDialog = async () => {
      machineAsideMachine.value = null;
      machineAsideOpen.value = true;
      await loadShellMachines();
    };

    const openShellMachineEdit = async (machine) => {
      machineAsideMachine.value = machine || null;
      machineAsideOpen.value = true;
      await loadShellMachines();
    };

    const onAsideConnect = async (machineName) => {
      machineAsideOpen.value = false;
      await openShellAndConnect(machineName);
    };

    const copyShellMachine = async (machine) => {
      if (!machine?.id) return;
      try {
        const copyName = await copyMachineRecord(machine, shellMachines.value);
        ElMessage.success(`已复制为「${copyName}」`);
        await onMachinesChanged();
      } catch (error) {
        console.error('复制机器配置失败:', error);
        ElMessage.error('复制机器配置失败: ' + (error.message || error));
      }
    };

    const deleteShellMachine = async (machine) => {
      if (!machine?.id) return;
      try {
        await ElMessageBox.confirm(`确定删除机器「${machine.name}」吗？`, '确认删除', { type: 'warning' });
        await App.DeleteMachine(machine.id);
        ElMessage.success('机器配置删除成功');
        await onMachinesChanged();
      } catch (error) {
        if (error === 'cancel') return;
        console.error('删除机器配置失败:', error);
        ElMessage.error('删除机器配置失败: ' + (error.message || error));
      }
    };

    const onMachinesChanged = async () => {
      await loadShellMachines();
    };

    // 切换 SubProject 展开状态
    const toggleSubProject = (subProjectName) => {
      expandedSubProjects.value[subProjectName] = !expandedSubProjects.value[subProjectName];
    };

    // 切换 Command 展开状态
    const toggleCommand = (subProjectName, commandName) => {
      const key = `${subProjectName}-${commandName}`;
      expandedCommands.value[key] = !expandedCommands.value[key];
    };

    // 获取命令类型对应的标签类型
    const getCommandTagType = (type) => {
      switch (type) {
        case 'remote':
          return 'warning';
        case 'batch':
          return 'success';
        default:
          return 'info';
      }
    };

    // 获取命令类型文本
    const getCommandTypeText = (type) => {
      switch (type) {
        case 'remote':
          return 'remote';
        case 'batch':
          return 'batch';
        default:
          return 'batch';
      }
    };

    // 执行 SubProject
    const executeSubProject = async (subProject) => {
      if (!selectedProject.value) {
        return;
      }

      try {
        outputLines.value = [];
        lastRemoteFailure.value = null;
        await App.ExecuteSubProject(
          subProject.projectName,
          subProject.name
        );
      } catch (error) {
        console.error("执行 SubProject 失败:", error);
      }
    };

    // 停止 SubProject
    const stopSubProject = async (subProject) => {
      if (!selectedProject.value) {
        return;
      }

      try {
        await App.StopSubProject(
          subProject.projectName,
          subProject.name
        );
      } catch (error) {
        console.error("停止 SubProject 失败:", error);
      }
    };

    const dryRunSubProject = async (subProject) => {
      if (!selectedProject.value) return;
      try {
        outputLines.value = [];
        lastRemoteFailure.value = null;
        await App.DryRunSubProject(subProject.projectName, subProject.name);
      } catch (error) {
        console.error('干跑失败:', error);
        ElMessage.error('干跑失败: ' + (error.message || error));
      }
    };

    const openFailureShell = async () => {
      const failure = lastRemoteFailure.value;
      if (!failure?.machineName) return;
      enterShellMode();
      await nextTick();
      try {
        const existing = workspaceSessions.value.find(
          (s) => s.configName === failure.machineName && s.connected,
        );
        if (existing) {
          activeMachine.value = existing.machineName;
        } else {
          await connectShell(failure.machineName);
          await nextTick();
        }
        const sessionId = activeMachine.value
          || workspaceSessions.value.find((s) => s.configName === failure.machineName)?.machineName;
        if (failure.workdir && sessionId) {
          await App.SendShellInput(sessionId, `cd ${failure.workdir}\n`);
        }
      } catch (error) {
        console.error('打开失败 Shell 失败:', error);
        ElMessage.error('连接 Shell 失败: ' + (error.message || error));
      }
    };

    // 执行单个 Command
    const executeCommand = async (payload) => {
      const projectName = payload?.projectName || payload?.subProject?.projectName
      const subProjectName = payload?.subProjectName || payload?.subProject?.name
      const commandName = payload?.commandName || payload?.command?.name
      if (!projectName || !subProjectName || !commandName) return
      try {
        outputLines.value = []
        lastRemoteFailure.value = null
        await App.ExecuteCommand(projectName, subProjectName, commandName)
      } catch (error) {
        console.error('执行命令失败:', error)
        ElMessage.error('执行命令失败: ' + (error.message || error))
      }
    };

    // 停止命令
    const stopCommand = async (cmd) => {
      if (cmd.subprojectName && cmd.projectName) {
        const subProject = { name: cmd.subprojectName, projectName: cmd.projectName };
        return stopSubProject(subProject);
      }
    };

    // 停止所有命令
    const stopAllCommands = async () => {
      try {
        await App.StopAllCommands();
      } catch (error) {
        console.error("停止所有命令失败:", error);
      }
    };

    // 检查 SubProject 是否正在运行
    const isSubProjectRunning = (subProject) => {
      if (!selectedProject.value) {
        return false;
      }
      return status.isRunning &&
        status.projectName === subProject.projectName &&
        status.subProjectName === subProject.name;
    };

    // 检查命令是否正在运行 (保持向后兼容)
    const isCommandRunning = (cmd) => {
      if (!selectedProject.value || !cmd.subprojectName) {
        return false;
      }
      const subProject = { name: cmd.subprojectName, projectName: cmd.projectName };
      return isSubProjectRunning(subProject);
    };

    // 处理 ANSI 转义序列
    const processAnsiOutput = (text) => {
      try {
        return convert.toHtml(text);
      } catch (error) {
        console.warn("ANSI 转换失败:", error);
        return text;
      }
    };

    // 处理单行输出
    const appendOutputLine = (line) => {
      if (line.startsWith('PROGRESS_UPDATE:')) {
        const parts = line.split(':');
        if (parts.length >= 3) {
          const progressID = parts[1];
          const progressText = parts.slice(2).join(':');
          const progressItem = newProcessedOutput(progressText, !progressText.includes('传输完成'));
          progressItem.progressID = progressID;
          progressItem.isSuccess = !progressText.includes('传输完成');

          for (let i = outputLines.value.length - 1; i >= 0; i--) {
            if (outputLines.value[i].isProgress && outputLines.value[i].progressID === progressID) {
              outputLines.value[i] = progressItem;
              return;
            }
          }
          outputLines.value.push(progressItem);
        }
        return;
      }

      outputLines.value.push(newProcessedOutput(line));
      enforceOutputLimit();
    };

    const newProcessedOutput = (line, isProgress = false) => {
      return {
        raw: line,
        html: processAnsiOutput(line),
        isError: line.includes('STDERR') || line.includes('失败') || line.includes('错误') || line.includes('Error'),
        isSuccess: line.includes('完成') || line.includes('成功'),
        isProgress: isProgress
      };
    };

    const handleOutputLine = (line) => {
      if (typeof line !== 'string' || line.length === 0) {
        return;
      }
      appendOutputLine(line);
    };

    const handleOutputClear = () => {
      outputLines.value = [];
    };

    const handleExecutionStatus = (currentStatus) => {
      if (!currentStatus) {
        return;
      }
      const running = !!(currentStatus.isRunning ?? currentStatus.IsRunning);
      Object.assign(status, currentStatus, { isRunning: running });
    };

    const handleExecutionOpenShell = (payload) => {
      if (!payload?.machineName) return;
      lastRemoteFailure.value = payload;
    };

    // 手动刷新输出（保留一次性拉取，不再轮询）
    const getOutput = async () => {
      try {
        const output = await App.GetOutput();
        if (output && output.length > 0) {
          output.forEach((line) => appendOutputLine(line));
        }
      } catch (error) {
        console.error("获取输出失败:", error);
      }
    };

    const getStatus = async () => {
      try {
        const currentStatus = await App.GetSubProjectStatus();
        Object.assign(status, currentStatus);
      } catch (error) {
        console.error("获取状态失败:", error);
      }
    };

    // 清空输出
    const clearOutput = async () => {
      try {
        outputLines.value = [];
        await App.ClearOutput();
      } catch (error) {
        console.error("清空输出失败:", error);
      }
    };

    const refreshOutput = async () => {
      await getOutput();
      await getStatus();
    };

    // 计算进度百分比
    const progressPercentage = computed(() => {
      if (!status.isRunning || status.totalSteps === 0) {
        return 0;
      }
      // 基于已完成的步骤数计算进度
      const completedRatio = Math.max(1, status.completedSteps + 1) / status.totalSteps;
      return Math.round(completedRatio * 95);

    });

    // 计算进度状态
    const progressStatus = computed(() => {
      if (!status.isRunning) {
        return '';
      }
      if (progressPercentage.value === 100) {
        return 'success';
      }
      return '';
    });

    const toggleTerminalSearch = () => {
      terminalSearchVisible.value = !terminalSearchVisible.value;
      if (!terminalSearchVisible.value) {
        terminalSearchQuery.value = '';
        terminalActiveMatchIndex.value = -1;
      }
    };

    const closeTerminalSearch = () => {
      terminalSearchVisible.value = false;
      terminalSearchQuery.value = '';
      terminalActiveMatchIndex.value = -1;
    };

    const handleSearchMatches = (indices) => {
      terminalMatchIndices.value = indices;
      if (indices.length === 0) {
        terminalActiveMatchIndex.value = -1;
        return;
      }
      if (terminalActiveMatchIndex.value < 0 || terminalActiveMatchIndex.value >= indices.length) {
        terminalActiveMatchIndex.value = 0;
      }
    };

    const gotoNextSearchMatch = () => {
      if (terminalMatchIndices.value.length === 0) return;
      terminalActiveMatchIndex.value = (terminalActiveMatchIndex.value + 1) % terminalMatchIndices.value.length;
    };

    const gotoPrevSearchMatch = () => {
      if (terminalMatchIndices.value.length === 0) return;
      terminalActiveMatchIndex.value = (terminalActiveMatchIndex.value - 1 + terminalMatchIndices.value.length) % terminalMatchIndices.value.length;
    };

    // 键盘快捷键处理（可在系统设置中自定义，保存至 app_data.json）
    // 使用捕获阶段，确保终端 / xterm 聚焦时也能收到
    const handleKeyDown = (e) => {
      // Escape：只关最上层弹框；有子 Dialog / MessageBox 时不关系统设置
      // 放在输入框判断之前，避免焦点在搜索框时无法关闭设置壳
      if (e.key === 'Escape') {
        if (settingsHubVisible.value && !hasOverlayAboveSettingsHub()) {
          e.preventDefault();
          settingsHubVisible.value = false;
        }
        return;
      }

      // 普通表单输入中不抢快捷键；xterm 隐藏 textarea 除外
      if (isFormFieldTarget(e.target)) {
        return;
      }

      const sc = shortcutMap.value;
      const inXterm = isXtermInput(e.target);

      const take = () => {
        e.preventDefault();
        e.stopPropagation();
      };

      // 终端搜索 — 任务模式 / Shell 模式统一在此处理
      if (matchesShortcut(e, sc.find)) {
        take();
        if (activeView.value === 'shell') {
          shellWorkspaceRef.value?.openSearch?.();
        } else {
          terminalSearchVisible.value = true;
        }
        return;
      }

      // Ctrl+C：终端内交给 shell（SIGINT / 选区复制由终端处理），不抢
      if (matchesShortcut(e, sc.copy)) {
        if (inXterm) return;
        take();
        copySelectedText();
        return;
      }

      // Ctrl+V：终端内交给 xterm 原生粘贴；Shell 其它焦点则写入当前会话
      if (matchesShortcut(e, sc.paste)) {
        if (inXterm) return;
        if (activeView.value === 'shell') {
          take();
          shellWorkspaceRef.value?.pasteClipboard?.();
        }
        return;
      }

      if (matchesShortcut(e, sc.clearOutput)) {
        take();
        clearOutput();
        return;
      }

      if (matchesShortcut(e, sc.machineConfig)) {
        take();
        openMachineConfig();
        return;
      }

      if (matchesShortcut(e, sc.connectionManager)) {
        take();
        openConnectionManager();
        return;
      }

      if (matchesShortcut(e, sc.envVars)) {
        take();
        openWorkPathConfig();
        return;
      }

      if (matchesShortcut(e, sc.newWindow)) {
        take();
        App.NewWindow();
        return;
      }

      if (matchesShortcut(e, sc.systemSettings)) {
        take();
        App.OpenSystemSettings();
        return;
      }

      if (matchesShortcut(e, sc.commandPalette)) {
        take();
        if (activeView.value === 'shell') {
          shellWorkspaceRef.value?.openCommandPalette?.();
        }
        return;
      }

      if (matchesShortcut(e, sc.paneZoom)) {
        if (activeView.value === 'shell') {
          take();
          shellWorkspaceRef.value?.togglePaneZoom?.();
        }
        return;
      }

      // Shell 标签 / 分屏导航
      if (activeView.value === 'shell' && shellMounted.value) {
        if (matchesShortcut(e, sc.nextTab)) {
          take();
          shellWorkspaceRef.value?.selectNextTab?.(1);
          return;
        }
        if (matchesShortcut(e, sc.prevTab)) {
          take();
          shellWorkspaceRef.value?.selectNextTab?.(-1);
          return;
        }
        if (matchesShortcut(e, sc.closeTab)) {
          take();
          shellWorkspaceRef.value?.closeActiveTab?.();
          return;
        }
        if (matchesShortcut(e, sc.toggleBroadcast)) {
          take();
          shellWorkspaceRef.value?.toggleBroadcast?.();
          return;
        }
        if (matchesShortcut(e, sc.openSftp)) {
          take();
          shellWorkspaceRef.value?.toggleFilePanel?.();
          return;
        }
        if (matchesShortcut(e, sc.openLocalShell)) {
          take();
          connectLocalShell();
          return;
        }
        if (matchesShortcut(e, sc.splitFocusLeft) || matchesShortcut(e, sc.splitFocusUp)) {
          take();
          shellWorkspaceRef.value?.focusSplitNeighbor?.('left');
          return;
        }
        if (matchesShortcut(e, sc.splitFocusRight) || matchesShortcut(e, sc.splitFocusDown)) {
          take();
          shellWorkspaceRef.value?.focusSplitNeighbor?.('right');
          return;
        }
        // Ctrl+1..9 跳转到第 N 个标签（Ctrl+0 仍保留字号重置）
        if ((e.ctrlKey || e.metaKey) && !e.altKey && !e.shiftKey) {
          const code = String(e.code || '');
          const m = /^Digit([1-9])$/.exec(code) || /^Numpad([1-9])$/.exec(code);
          if (m) {
            take();
            shellWorkspaceRef.value?.selectTabByIndex?.(Number(m[1]) - 1);
            return;
          }
        }
      }

      // Shell 终端字号：Ctrl/⌘ + = / - / 0（固定，不占用可配置快捷键表）
      if (activeView.value === 'shell' && (e.ctrlKey || e.metaKey) && !e.altKey) {
        const code = String(e.code || '');
        const key = String(e.key || '');
        let nextSize = null;
        if (key === '=' || key === '+' || code === 'Equal' || code === 'NumpadAdd') {
          nextSize = clampShellFontSize((shellFontSize.value || SHELL_FONT_SIZE_DEFAULT) + 1);
        } else if (!e.shiftKey && (key === '-' || key === '_' || code === 'Minus' || code === 'NumpadSubtract')) {
          nextSize = clampShellFontSize((shellFontSize.value || SHELL_FONT_SIZE_DEFAULT) - 1);
        } else if (!e.shiftKey && (key === '0' || code === 'Digit0' || code === 'Numpad0')) {
          nextSize = SHELL_FONT_SIZE_DEFAULT;
        }
        if (nextSize != null) {
          take();
          if (nextSize !== shellFontSize.value) {
            saveTheme({ shellFontSize: nextSize }).catch(() => {});
          }
        }
      }
    };

    // 复制选中的文本
    const copySelectedText = async () => {
      try {
        const selectedText = window.getSelection().toString();
        if (selectedText) {
          await navigator.clipboard.writeText(selectedText);
        } else {
          // 如果没有选中文本，复制所有终端输出
          const allText = outputLines.value.map(line => line.text || line.html.replace(/<[^>]*>/g, '')).join('\n');
          if (allText.trim()) {
            await navigator.clipboard.writeText(allText);
          } else {
          }
        }
      } catch (err) {
        console.error('复制失败:', err);
        ElMessage.error('复制失败');
      }
    };

    // 拖拽调整面板宽度
    const startResize = (e) => {
      e.preventDefault();
      isResizing.value = true;

      const startX = e.clientX;
      const startWidth = leftPanelWidth.value;

      const handleMouseMove = (e) => {
        const deltaX = e.clientX - startX;
        const newWidth = startWidth + deltaX;

        // 限制宽度范围
        if (newWidth >= minPanelWidth && newWidth <= maxPanelWidth) {
          leftPanelWidth.value = newWidth;
        }
      };

      const handleMouseUp = () => {
        isResizing.value = false;
        document.removeEventListener('mousemove', handleMouseMove);
        document.removeEventListener('mouseup', handleMouseUp);
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
      };

      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    };

    let quitConfirmOpen = false;

    const handleConfirmQuit = async () => {
      if (quitConfirmOpen) return;
      quitConfirmOpen = true;
      try {
        await ElMessageBox.confirm(
          "确定要退出 FlashShell 吗？",
          "退出应用",
          {
            confirmButtonText: "退出",
            cancelButtonText: "取消",
            type: "warning",
          },
        );
        await App.ConfirmQuit();
      } catch {
        // 用户取消
      } finally {
        quitConfirmOpen = false;
      }
    };

    onMounted(() => {
      // 如果上一次刷新触发了页面重载，则在重载完成后再弹提示
      const pendingReloadToast = sessionStorage.getItem('pendingReloadToastMessage');
      if (pendingReloadToast) {
        sessionStorage.removeItem('pendingReloadToastMessage');
        ElMessage.success(pendingReloadToast);
      }

      ensureShellAsciiInputListeners();
      loadConfig();
      loadTheme();
      loadShortcutMap();
      loadShellMachines();
      loadTaskOutputLimit();
      refreshMcpNavVisibility();
      // 空闲锁定：真实用户活动重置计时；定时检查是否超时
      let lastVaultTouch = 0;
      const onUserActivity = () => {
        const now = Date.now();
        if (now - lastVaultTouch < 5000) return;
        lastVaultTouch = now;
        App.VaultTouchActivity?.().catch(() => {});
      };
      window.addEventListener('mousemove', onUserActivity, { passive: true });
      window.addEventListener('keydown', onUserActivity, { passive: true });
      const idleCheckTimer = setInterval(() => {
        App.GetVaultStatus?.().catch(() => {});
      }, 15000);
      window.__flashshellVaultActivityCleanup = () => {
        window.removeEventListener('mousemove', onUserActivity);
        window.removeEventListener('keydown', onUserActivity);
        clearInterval(idleCheckTimer);
      };
      App.GetSessionInfo().then((info) => { sessionId.value = info.sessionId || ''; }).catch(() => { });
      App.GetAppVersion().then((v) => { appVersion.value = v || ''; }).catch(() => { appVersion.value = ''; });

      // 监听输出与执行状态事件（替代轮询）
      EventsOn("output:line", handleOutputLine);
      EventsOn("output:clear", handleOutputClear);
      EventsOn("execution:status", handleExecutionStatus);
      EventsOn("execution:open-shell", handleExecutionOpenShell);
      EventsOn("theme:changed", applyThemeSettings);
      EventsOn("system-settings:changed", (payload) => {
        if (payload && Object.prototype.hasOwnProperty.call(payload, 'taskOutputMaxLines')) {
          applyTaskOutputMaxLines(payload.taskOutputMaxLines);
        }
        if (payload && Object.prototype.hasOwnProperty.call(payload, 'shellAsciiInput')) {
          setShellAsciiInputEnabled(payload.shellAsciiInput !== false);
        }
      });
      EventsOn("shortcuts:changed", (data) => {
        shortcutMap.value = mergeShortcuts(data);
        loadShortcutMap();
      });

      // 添加全局键盘事件监听器
      document.addEventListener('keydown', handleKeyDown, true);

      // 新窗口自动连接
      App.ConsumePendingConnectMachine?.().then(async (name) => {
        const machineName = String(name || '').trim();
        if (!machineName) return;
        await enterShellMode();
        await connectShell(machineName);
      }).catch(() => {});

      // 监听统一的操作结果事件
      EventsOn("operation:result", async (event) => {
        console.log("收到操作结果事件:", event);
        handleOperationEvent(event);
      });

      // 监听配置变更事件（软刷新项目/机器；勿依赖整页 reload，且勿被其它组件 EventsOff 清掉）
      EventsOn("config:changed", async (data) => {
        console.log("收到 config:changed 事件:", data);
        isReloading.value = true;
        try {
          await loadConfig();
          await loadShellMachines();
          selectedProject.value = null;
          selectedSubProject.value = null;
          subProjects.value = [];
          if (activeView.value === 'task') {
            activeView.value = 'home';
          }
          ElMessage.success("已切换配置");
        } catch (error) {
          console.error("切换配置后刷新失败:", error);
          ElMessage.error("切换配置后刷新失败: " + (error?.message || error));
        } finally {
          isReloading.value = false;
        }
      });

      // 监听打开设置相关事件（统一进 Settings Hub）
      EventsOn("open:machine-config", async () => {
        await openMachineConfig();
      });
      EventsOn("open:connection-manager", async () => {
        await openConnectionManager();
      });
      EventsOn("open:workpath-config", async () => {
        await openWorkPathConfig();
      });
      EventsOn("open:about", () => { openAbout(); });
      EventsOn("open:config-editor", () => { configEditorVisible.value = true; });
      EventsOn("open:system-settings", () => { openSettingsHub('general'); });
      EventsOn("mcp:status-changed", (st) => {
        mcpServiceEnabled.value = !!(st?.enabled || st?.online);
      });
      EventsOn("app:confirm-quit", handleConfirmQuit);

      const closeApprovalNotification = () => {
        if (approvalNotif) {
          approvalNotif.close();
          approvalNotif = null;
        }
      };

      const showApprovalNotification = (payload) => {
        if (activeView.value === "audit") return;
        const tool = payload?.tool || "MCP 工具";
        const server = payload?.server ? ` · ${payload.server}` : "";
        const summary = String(payload?.summary || payload?.preview || "").slice(0, 160);
        closeApprovalNotification();
        approvalNotif = ElNotification({
          title: "MCP 待审批",
          type: "warning",
          duration: 0,
          showClose: true,
          customClass: "approval-queue-notif",
          message: h("div", { class: "approval-notif-body" }, [
            h("div", { class: "approval-notif-line" }, `${tool}${server}`),
            h("div", { class: "approval-notif-summary" }, summary || "点击查看详情"),
            h(
              "button",
              {
                type: "button",
                class: "approval-notif-go",
                onClick: (ev) => {
                  ev.stopPropagation();
                  closeApprovalNotification();
                  switchActiveView("audit");
                },
              },
              "去审批 →",
            ),
          ]),
        });
      };

      const onApprovalQueued = (payload) => {
        approvalPendingCount += 1;
        // 弹窗由 McpApprovalDialog 统一弹出；此处补应用内通知（人不在审计页时）
        if (activeView.value !== "audit") {
          showApprovalNotification(payload);
        }
        if (document.hidden) {
          sendOsNotification({
            id: `mcp-approval-${payload?.id || Date.now()}`,
            title: "FlashShell MCP 待审批",
            body: `${payload?.tool || "MCP"}${payload?.server ? ` · ${payload.server}` : ""}`,
          });
        }
      };
      const onApprovalResolved = () => {
        approvalPendingCount = Math.max(0, approvalPendingCount - 1);
        if (approvalPendingCount === 0) {
          closeApprovalNotification();
        }
      };
      // 保存 disposer，卸载时只解绑自己，勿用 EventsOff(事件名)
      offApprovalQueued = EventsOn("approval:queued", onApprovalQueued);
      offApprovalResolved = EventsOn("approval:resolved", onApprovalResolved);

      // 启动进入首页时检查新版本并弹窗
      if (activeView.value === 'home') {
        maybePromptUpdateOnHome();
        nextTick().then(() => homePageRef.value?.focusSearchInput?.());
      }
    });

    watch(activeView, async (view, prev) => {
      if (prev === 'shell' && view !== 'shell') {
        notifyLeaveShellMode();
      }
      if (view === 'audit' && approvalNotif) {
        approvalNotif.close();
        approvalNotif = null;
      }
      if (view === 'home') {
        maybePromptUpdateOnHome();
        await nextTick();
        homePageRef.value?.focusSearchInput?.();
      }
    });

    // 机器配置相关方法
    const loadMachines = async () => {
      try {
        machinesLoading.value = true;
        await loadShellMachines();
        machines.value = shellMachines.value || [];
      } catch (error) {
        console.error('加载机器配置失败:', error);
        ElMessage.error('加载机器配置失败: ' + error.message);
      } finally {
        machinesLoading.value = false;
      }
    };

    const openMachineConfig = async () => {
      machineEditId.value = '';
      openSettingsHub('machines');
      await loadMachines();
    };

    const addMachine = () => {
      editingMachine.value = null;
      resetMachineForm();
      machineEditVisible.value = true;
    };

    const editMachine = async (machine) => {
      editingMachine.value = machine;
      machineForm.name = machine.name;
      machineForm.key_file = machine.key_file || '';

      try {
        const sensitiveData = await App.GetMachineSensitiveData(machine.name);
        if (sensitiveData) {
          machineForm.host = sensitiveData.host || '';
          machineForm.port = sensitiveData.port || 22;
          machineForm.user = sensitiveData.user || '';
          machineForm.password = sensitiveData.password || '';
        }
      } catch (error) {
        console.error('获取敏感数据失败:', error);
        ElMessage.warning('获取敏感数据失败，请重新输入');
      }

      machineEditVisible.value = true;
    };

    const resetMachineForm = () => {
      machineForm.name = '';
      machineForm.key_file = '';
      machineForm.host = '';
      machineForm.port = 22;
      machineForm.user = '';
      machineForm.password = '';
    };

    const saveMachine = async () => {
      if (!machineFormRef.value) return;

      try {
        await machineFormRef.value.validate();
      } catch {
        // 表单校验失败时 Element Plus 会 reject 字段错误对象，字段下方已有红字提示，勿再弹系统错误
        return;
      }

      savingMachine.value = true;
      try {
        const machineData = {
          name: machineForm.name,
          key_file: machineForm.key_file
        };

        const sensitiveData = {
          host: machineForm.host,
          port: machineForm.port,
          user: machineForm.user,
          password: machineForm.password
        };
        if (editingMachine.value) {
          // 更新机器
          await App.UpdateMachine(editingMachine.value.name, machineData);
          await App.SetMachineSensitiveData(machineForm.name, sensitiveData);
          ElMessage.success('机器配置更新成功');
        } else {
          // 添加机器
          await App.AddMachine(machineData);
          await App.SetMachineSensitiveData(machineForm.name, sensitiveData);
          ElMessage.success('机器配置添加成功');
        }

        machineEditVisible.value = false;
        await loadMachines();
        await loadShellMachines();
      } catch (error) {
        console.error('保存机器配置失败:', error);
        ElMessage.error('保存机器配置失败: ' + (error?.message || error));
      } finally {
        savingMachine.value = false;
      }
    };

    const deleteMachine = async (machine) => {
      try {
        await App.DeleteMachine(machine.name);
        ElMessage.success('机器配置删除成功');
        await loadMachines();
        await loadShellMachines();
      } catch (error) {
        console.error('删除机器配置失败:', error);
        ElMessage.error('删除机器配置失败: ' + error.message);
      }
    };

    const testConnection = async (machine) => {
      try {
        machine.testing = true;
        await App.TestMachineConnection(machine.name);
        ElMessage.success('连接测试成功');
      } catch (error) {
        console.error('连接测试失败:', error);
        ElMessage.error('连接测试失败: ' + error.message);
      } finally {
        machine.testing = false;
      }
    };

    const selectKeyFile = async () => {
      try {
        const filePath = await App.SelectKeyFile();
        if (filePath) {
          machineForm.key_file = filePath;
        }
      } catch (error) {
        console.error('选择密钥文件失败:', error);
        ElMessage.error('选择密钥文件失败: ' + error.message);
      }
    };

    // 环境变量配置相关方法
    const loadWorkPaths = async () => {
      try {
        workPathsLoading.value = true;
        const workPathsData = await App.GetWorkPaths();
        workPaths.value = workPathsData || {};
      } catch (error) {
        console.error('加载环境变量配置失败:', error);
        ElMessage.error('加载环境变量配置失败: ' + error.message);
      } finally {
        workPathsLoading.value = false;
      }
    };

    const openWorkPathConfig = async () => {
      openSettingsHub('env');
    };

    const addWorkPath = () => {
      editingWorkPath.value = null;
      resetWorkPathForm();
      workPathEditVisible.value = true;
    };

    const editWorkPath = (key) => {
      editingWorkPath.value = key;
      workPathForm.key = key;
      workPathForm.value = workPaths.value[key] || '';
      workPathEditVisible.value = true;
    };

    const resetWorkPathForm = () => {
      workPathForm.key = '';
      workPathForm.value = '';
    };

    const saveWorkPath = async () => {
      if (!workPathFormRef.value) return;

      try {
        await workPathFormRef.value.validate();
        savingWorkPath.value = true;

        if (editingWorkPath.value) {
          // 更新环境变量
          await App.UpdateWorkPath(workPathForm.key, workPathForm.value);
          ElMessage.success('环境变量更新成功');
        } else {
          // 添加环境变量
          await App.AddWorkPath(workPathForm.key, workPathForm.value);
          ElMessage.success('环境变量添加成功');
        }

        workPathEditVisible.value = false;
        await loadWorkPaths();

      } catch (error) {
        console.error('保存环境变量失败:', error);
        ElMessage.error('保存环境变量失败: ' + error.message);
      } finally {
        savingWorkPath.value = false;
      }
    };

    const deleteWorkPath = async (key) => {
      try {
        await App.DeleteWorkPath(key);
        ElMessage.success('环境变量删除成功');
        await loadWorkPaths();
      } catch (error) {
        console.error('删除环境变量失败:', error);
        ElMessage.error('删除环境变量失败: ' + error.message);
      }
    };

    // 处理操作事件
    const handleOperationEvent = (event) => {
      console.log("处理操作事件:", event);

      // 如果需要重新加载，显示加载状态并重新加载
      if (event.needReload) {
        // 需要 reload 时不在当前页面立刻弹 toast；
        // 而是把提示内容存到 sessionStorage，reload 完成后再弹。
        if (event.messageType === 'success' && event.message) {
          sessionStorage.setItem('pendingReloadToastMessage', event.message);
        }
        setTimeout(() => {
          isReloading.value = true;
          window.location.reload();
        }, 1500);
        return;
      }

      // 根据消息类型显示不同的提示（非重载场景）
      switch (event.messageType) {
        case 'success':
          ElMessage.success(event.message);
          break;
        case 'error':
          ElMessage.error(event.message);
          break;
        case 'warning':
          ElMessage.warning(event.message);
          break;
        case 'info':
          ElMessage.info(event.message);
          break;
        default:
          ElMessage.info(event.message);
      }
    };

    onUnmounted(() => {
      document.removeEventListener('keydown', handleKeyDown, true);
      try {
        offApprovalQueued?.();
        offApprovalResolved?.();
      } catch (e) {
        // ignore
      }
      try {
        EventsOff(
          "operation:result",
          "config:changed",
          "open:machine-config",
          "open:connection-manager",
          "open:workpath-config",
          "open:about",
          "open:config-editor",
          "open:system-settings",
          "mcp:status-changed",
          "theme:changed",
          "system-settings:changed",
          "shortcuts:changed",
          "output:line",
          "output:clear",
          "execution:status",
          "app:confirm-quit",
          "shell:data",
          "shell:line",
          "shell:clear",
          "shell:status",
        );
      } catch (e) {
        // 忽略解绑异常，确保卸载流程不中断
      }
    });

    return {
      projects,
      subProjects,
      outputLines,
      lastRemoteFailure,
      selectedProject,
      selectedSubProject,
      isReloading,
      status,
      progressPercentage,
      progressStatus,
      expandedSubProjects,
      expandedCommands,
      refreshConfig,
      debugConfig,
      selectProject,
      backToProjectList,
      toggleSubProject,
      toggleCommand,
      getCommandTagType,
      getCommandTypeText,
      executeSubProject,
      dryRunSubProject,
      openFailureShell,
      stopSubProject,
      executeCommand,
      stopCommand,
      stopAllCommands,
      isSubProjectRunning,
      isCommandRunning,
      clearOutput,
      refreshOutput,
      // 面板拖拽相关
      leftPanelWidth,
      isResizing,
      startResize,
      // 机器配置相关
      machineEditId,
      machineAsideOpen,
      machineAsideMachine,
      onAsideConnect,
      machines,
      machinesLoading,
      settingsHubVisible,
      settingsSection,
      mcpServiceEnabled,
      openSettingsHub,
      openMachineConfig,
      openWorkPathConfig,
      // 关于 / 更新提示
      aboutVisible,
      aboutIntroHtml,
      aboutPromptMode,
      aboutInitialUpdate,
      openAbout,
      onAboutPromptDismissed,
      onAboutPromptSkipped,
      // 事件处理
      handleOperationEvent,
      // 键盘快捷键
      handleKeyDown,
      copySelectedText,
      themeClass,
      configEditorVisible,
      terminalSearchVisible,
      terminalSearchQuery,
      terminalMatchSummary,
      terminalActiveMatchIndex,
      toggleTerminalSearch,
      closeTerminalSearch,
      gotoNextSearchMatch,
      gotoPrevSearchMatch,
      handleSearchMatches,
      refreshProjectConfig,
      sessionId,
      statusBarInfo,
      // Shell / 视图切换
      activeView,
      shellMounted,
      shellMode,
      resumeTaskView,
      shellSessions,
      shellMachines,
      activeMachine,
      connectingName,
      testingName,
      connectedSessions,
      workspaceSessions,
      connectedCount,
      openSessionCount,
      enterShellMode,
      leaveShellMode,
      switchActiveView,
      openConnectionManager,
      openShellAndConnect,
      onQuickFocusSession,
      onQuickConnectMachine,
      onQuickConnectMachines,
      onSettingsConnectMachine,
      connectShell,
      connectLocalShell,
      connectOrReconnectShell,
      duplicateShellSession,
      disconnectShell,
      closeShellSession,
      closeShellSessions,
      hostKeyDialogVisible,
      pendingHostKey,
      onHostKeyTrusted,
      openShellMachineDialog,
      openShellMachineEdit,
      copyShellMachine,
      deleteShellMachine,
      onMachinesChanged,
      homePageRef,
      shellWorkspaceRef,
      testShellConnection,
      broadcastEnabled,
      broadcastTargets,
      splitSessionIds,
      toggleBroadcastTarget,
      setSplitSessions,
      toggleSplitSession,
    };
  },
};
</script>

<style scoped>
.app-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
  position: relative;
  background: var(--bg);
  color: var(--fg);
  overflow: hidden;
}

.app-body {
  flex: 1;
  min-height: 0;
  display: flex;
  min-width: 0;
}

.shell-top-chrome-host {
  flex: 0 0 auto;
  min-width: 0;
  min-height: 38px;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
}

.view.active {
  display: flex;
}

.view.shell {
  background: var(--app-bg);
}

.view.home,
.view.task,
.view.audit {
  flex: 1;
  min-height: 0;
  min-width: 0;
  width: 100%;
  flex-direction: column;
}

.view.audit.active {
  display: flex;
}

.task-left.left-panel {
  flex-shrink: 0;
  min-width: 0;
}

/* 全局加载遮罩 */
.global-loading {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}

.loading-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  background: var(--surface);
  padding: 32px;
  border-radius: var(--r-lg);
  border: 1px solid var(--border);
  box-shadow: var(--shadow-lg);
}

.loading-icon {
  font-size: 32px;
  color: var(--accent);
  animation: spin 1s linear infinite;
}

.loading-text {
  font-size: 16px;
  color: var(--fg-2);
  font-weight: 500;
}

@keyframes spin {
  0% {
    transform: rotate(0deg);
  }

  100% {
    transform: rotate(360deg);
  }
}

.main-container {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.left-panel {
  border-right: 1px solid var(--app-border);
  background-color: var(--app-inset-bg);
  display: flex;
  flex-direction: column;
  position: relative;
  height: 100%;
  overflow-x: hidden;
}

.shell-view-host {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.task-view-host {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.task-view-host > .main-container {
  flex: 1;
  min-height: 0;
}

.projectlist-fullscreen {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  background: var(--app-bg);
  display: flex;
  flex-direction: column;
}

.shell-terminal-container {
  display: flex;
  flex-direction: column;
}

.shell-terminal-container :deep(.terminal-wrapper) {
  flex: 1;
  min-height: 0;
}

/* 子组件已接管样式：ProjectList、SubProjectList、TerminalOutput、StatusBar */

.status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 16px;
  background: var(--app-panel-bg);
  border-top: 1px solid var(--app-border);
  color: var(--app-text);
  height: 40px;
  flex-shrink: 0;
  box-sizing: border-box;
}

.status-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  height: 100%;
}

/* 执行状态固定宽度容器 */
.status-container {
  min-width: 200px;
  max-width: 300px;
  flex-shrink: 0;
}

.status-text {
  display: inline-block;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 100%;
}

/* 确保状态栏内的所有元素垂直居中 */
.status-bar .el-tag {
  margin: 0;
}

.status-bar .el-button {
  margin: 0;
}

.app-info {
  font-size: 12px;
  color: var(--app-text-muted);
  font-weight: 500;
}

.shortcuts-hint {
  font-size: 11px;
  color: #909399;
  font-weight: 400;
  opacity: 0.8;
}

.no-projects {
  text-align: center;
  color: #909399;
  padding: 20px;
}

/* ANSI 颜色支持 */
.terminal-output :deep(.ansi-black-fg) {
  color: #000000;
}

.terminal-output :deep(.ansi-red-fg) {
  color: #cd3131;
}

.terminal-output :deep(.ansi-green-fg) {
  color: #0dbc79;
}

.terminal-output :deep(.ansi-yellow-fg) {
  color: #e5e510;
}

.terminal-output :deep(.ansi-blue-fg) {
  color: #2472c8;
}

.terminal-output :deep(.ansi-magenta-fg) {
  color: #bc3fbc;
}

.terminal-output :deep(.ansi-cyan-fg) {
  color: #11a8cd;
}

.terminal-output :deep(.ansi-white-fg) {
  color: #e5e5e5;
}

.terminal-output :deep(.ansi-bright-black-fg) {
  color: #666666;
}

.terminal-output :deep(.ansi-bright-red-fg) {
  color: #f14c4c;
}

.terminal-output :deep(.ansi-bright-green-fg) {
  color: #23d18b;
}

.terminal-output :deep(.ansi-bright-yellow-fg) {
  color: #f5f543;
}

.terminal-output :deep(.ansi-bright-blue-fg) {
  color: #3b8eea;
}

.terminal-output :deep(.ansi-bright-magenta-fg) {
  color: #d670d6;
}

.terminal-output :deep(.ansi-bright-cyan-fg) {
  color: #29b8db;
}

.terminal-output :deep(.ansi-bright-white-fg) {
  color: #ffffff;
}

/* ANSI 背景颜色 */
.terminal-output :deep(.ansi-black-bg) {
  background-color: #000000;
}

.terminal-output :deep(.ansi-red-bg) {
  background-color: #cd3131;
}

.terminal-output :deep(.ansi-green-bg) {
  background-color: #0dbc79;
}

.terminal-output :deep(.ansi-yellow-bg) {
  background-color: #e5e510;
}

.terminal-output :deep(.ansi-blue-bg) {
  background-color: #2472c8;
}

.terminal-output :deep(.ansi-magenta-bg) {
  background-color: #bc3fbc;
}

.terminal-output :deep(.ansi-cyan-bg) {
  background-color: #11a8cd;
}

.terminal-output :deep(.ansi-white-bg) {
  background-color: #e5e5e5;
}

/* ANSI 样式 */
.terminal-output :deep(.ansi-bold) {
  font-weight: bold;
}

.terminal-output :deep(.ansi-dim) {
  opacity: 0.5;
}

.terminal-output :deep(.ansi-italic) {
  font-style: italic;
}

.terminal-output :deep(.ansi-underline) {
  text-decoration: underline;
}

.terminal-output :deep(.ansi-strikethrough) {
  text-decoration: line-through;
}

/* 机器配置相关样式 */
.machine-config-container {
  padding: 20px;
}

.machine-list {
  margin-bottom: 20px;
}

.list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.list-header h4 {
  margin: 0;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.dialog-footer.icon-actions {
  gap: 10px;
}

/* 机器配置表格样式 */
.machine-config-container .el-table {
  border-radius: 8px;
  overflow: hidden;
}

.machine-config-container .el-table th {
  background-color: #f5f7fa;
  color: #606266;
  font-weight: 600;
}

.machine-config-container .el-table td {
  padding: 12px 0;
}

/* 表单样式优化 */
.machine-config-container .el-form-item {
  margin-bottom: 20px;
}

.machine-config-container .el-form-item__label {
  font-weight: 500;
  color: #606266;
}

.machine-config-container .el-input,
.machine-config-container .el-input-number {
  width: 100%;
}

.machine-config-container .el-divider {
  margin: 20px 0;
}

.machine-config-container .el-divider__text {
  color: #909399;
  font-weight: 500;
}

/* 密钥文件选择组件样式 */
.key-file-input {
  display: flex;
  gap: 8px;
  align-items: center;
}

.key-file-input .el-input {
  flex: 1;
}

.key-file-input .el-button {
  flex-shrink: 0;
}

/* 环境变量配置相关样式 */
.workpath-config-container {
  padding: 20px;
}

.workpath-list {
  margin-bottom: 20px;
}

.workpath-config-container .el-table {
  border-radius: 8px;
  overflow: hidden;
}

.workpath-config-container .el-table th {
  background-color: #f5f7fa;
  color: #606266;
  font-weight: 600;
}

.workpath-config-container .el-table td {
  padding: 12px 0;
}

/* 使用说明样式 */
.usage-info {
  background: #f8f9fa;
  border: 1px solid #e9ecef;
  border-radius: 6px;
  padding: 12px;
  margin-top: 8px;
}

.usage-info p {
  margin: 4px 0;
  font-size: 12px;
  color: #606266;
  line-height: 1.4;
}

.usage-info p:first-child {
  margin-top: 0;
}

.usage-info p:last-child {
  margin-bottom: 0;
}

/* 过渡动画样式 */
/* 进度条区域滑入滑出动画 */
.progress-slide-enter-active,
.progress-slide-leave-active {
  transition: all 0.3s ease-in-out;
  transform-origin: top;
}

.progress-slide-enter-from {
  opacity: 0;
  transform: translateY(-20px) scaleY(0);
}

.progress-slide-leave-to {
  opacity: 0;
  transform: translateY(-10px) scaleY(0);
}

/* 命令状态淡入淡出动画 */
.command-fade-enter-active,
.command-fade-leave-active {
  transition: all 0.2s ease-in-out;
}

.command-fade-enter-from {
  opacity: 0;
  transform: translateY(5px);
}

.command-fade-leave-to {
  opacity: 0;
  transform: translateY(-5px);
}

/* 状态栏状态切换动画 */
.status-fade-enter-active,
.status-fade-leave-active {
  transition: all 0.25s ease-in-out;
}

.status-fade-enter-from {
  opacity: 0;
  transform: translateX(-10px) scale(0.95);
}

.status-fade-leave-to {
  opacity: 0;
  transform: translateX(10px) scale(0.95);
}

/* 项目信息淡入淡出动画 */
.project-fade-enter-active,
.project-fade-leave-active {
  transition: all 0.2s ease-in-out;
}

.project-fade-enter-from {
  opacity: 0;
  transform: translateY(3px);
}

.project-fade-leave-to {
  opacity: 0;
  transform: translateY(-3px);
}

/* 按钮滑入滑出动画 */
.button-slide-enter-active,
.button-slide-leave-active {
  transition: all 0.2s ease-in-out;
}

.button-slide-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.button-slide-leave-to {
  opacity: 0;
  transform: translateX(20px);
}

/* 拖拽手柄样式 */
.resize-handle {
  position: absolute;
  top: 0;
  right: -3px;
  width: 6px;
  height: 100%;
  background: transparent;
  cursor: col-resize;
  z-index: 10;
  transition: background-color 0.2s ease;
}

.resize-handle:hover {
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 30%, transparent);
}

.resize-handle:active {
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 50%, transparent);
}

/* 拖拽时的视觉反馈 */
.left-panel.resizing {
  user-select: none;
}

.left-panel.resizing .resize-handle {
  background: color-mix(in srgb, var(--app-accent-color, #409eff) 50%, transparent);
}
</style>

<style>
html,
body,
#app {
  height: 100%;
  overflow: hidden;
  overscroll-behavior: none;
}

.terminal-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  padding: 0 !important;
  box-sizing: border-box;
}

.approval-queue-notif.el-notification {
  min-width: 320px;
  max-width: 420px;
}
.approval-notif-body { font-size: 13px; line-height: 1.45; }
.approval-notif-line { font-weight: 600; margin-bottom: 4px; }
.approval-notif-summary {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  margin-bottom: 8px;
  word-break: break-all;
}
.approval-notif-go {
  border: none;
  background: none;
  padding: 0;
  color: var(--el-color-warning);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}
.approval-notif-go:hover { text-decoration: underline; }
</style>
