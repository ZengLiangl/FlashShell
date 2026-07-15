<template>
  <div class="app-container" :class="themeClass">
    <AppMenuBar />

    <!-- 全局加载遮罩 -->
    <div v-if="isReloading" class="global-loading">
      <div class="loading-content">
        <el-icon class="loading-icon">
          <Loading />
        </el-icon>
        <div class="loading-text">正在重新加载...</div>
      </div>
    </div>

    <!-- 任务详情视图（与 Shell 可并行，仅切换展示） -->
    <template v-if="activeView === 'task' && selectedProject">
      <el-container class="main-container">
        <!-- 左侧面板 -->
        <el-aside :width="leftPanelWidth + 'px'" class="left-panel" :class="{ resizing: isResizing }">
          <!-- 拖拽手柄 -->
          <div class="resize-handle" @mousedown="startResize"></div>
          <SubProjectList :selected-project="selectedProject" :sub-projects="subProjects"
            :expanded-sub-projects="expandedSubProjects" :expanded-commands="expandedCommands" :status="status"
            :get-command-tag-type="getCommandTagType" :get-command-type-text="getCommandTypeText"
            :is-sub-project-running="isSubProjectRunning" @toggle-sub="toggleSubProject" @toggle-cmd="toggleCommand"
            @execute-sub="executeSubProject" @stop-sub="stopSubProject" @back="backToProjectList" />
        </el-aside>

        <!-- 右侧终端输出 -->
        <el-main class="terminal-container">
          <TerminalHeader :show-back="false" :search-visible="terminalSearchVisible"
            v-model:search-query="terminalSearchQuery" :match-summary="terminalMatchSummary" @clear="clearOutput"
            @refresh="refreshOutput" @toggle-search="toggleTerminalSearch" @search-next="gotoNextSearchMatch"
            @search-prev="gotoPrevSearchMatch" @close-search="closeTerminalSearch" />
          <TerminalOutput ref="terminalOutputRef" :status="status" :output-lines="outputLines"
            :progress-percentage="progressPercentage" :progress-status="progressStatus"
            :search-query="terminalSearchQuery" :active-match-index="terminalActiveMatchIndex"
            @search-matches="handleSearchMatches" />
        </el-main>
      </el-container>

      <!-- 状态栏（仅详情视图显示） -->
      <StatusBar :status="status" :selected-project="selectedProject" :app-info="statusBarInfo"
        @stop-all="stopAllCommands" />
    </template>

    <!-- Shell 视图：挂载后用 v-show 保留会话，可与任务并行 -->
    <div v-show="activeView === 'shell'" class="shell-view-host">
      <ShellWorkspace v-if="shellMounted" :active="activeView === 'shell'"
        :left-panel-width="Math.min(leftPanelWidth, 320)" :is-resizing="isResizing" :app-info="statusBarInfo"
        :machines="shellMachines" :sessions="shellSessions" :workspace-sessions="workspaceSessions"
        :connected-count="connectedCount" :open-session-count="openSessionCount" v-model:active-machine="activeMachine"
        :connecting-name="connectingName" :testing-name="testingName" @back="leaveShellMode"
        @connect="(name) => connectShell(name)" @disconnect="disconnectShell" @close-session="closeShellSession"
        @test="testShellConnection" @add-machine="openShellMachineDialog" @edit-machine="openShellMachineEdit"
        @start-resize="startResize" />
    </div>

    <!-- 首页：任务模式 + Shell 模式入口 -->
    <template v-if="activeView === 'home'">
      <div class="projectlist-fullscreen">
        <HomePage ref="homePageRef" :projects="projects" :connected-count="connectedCount" :has-task="!!selectedProject"
          :task-running="status.isRunning" :connecting-name="connectingName" @refresh="refreshConfig" @select-project="selectProject"
          @resume-task="resumeTaskView" @open-shell="enterShellMode" @connect-machine="openShellAndConnect"
          @add-machine="openShellMachineDialog" @open-system-settings="openSettingsHub('general')"
          @open-execution-history="openSettingsHub('history')" />
      </div>
    </template>

    <SettingsHubDialog v-model="settingsHubVisible" :initial-section="settingsSection" :edit-machine-id="machineEditId"
      @machines-changed="onMachinesChanged" @machines-closed="machineEditId = ''" />

    <!-- 关于弹框 -->
    <AboutDialog v-model="aboutVisible" :intro-html="aboutIntroHtml" />

    <ConfigEditorDialog v-model="configEditorVisible" @saved="refreshProjectConfig" />
  </div>
</template>

<script>
import { ref, reactive, computed, onMounted, onUnmounted, nextTick } from "vue";
import { ElMessage } from "element-plus";
import * as App from "../wailsjs/go/app/App";
import { EventsOn, EventsOff } from "../wailsjs/runtime/runtime";
import Convert from "ansi-to-html";
import TerminalOutput from "./components/TerminalOutput.vue";
import StatusBar from "./components/StatusBar.vue";
import ProjectList from "./components/ProjectList.vue";
import HomePage from "./components/HomePage.vue";
import ShellWorkspace from "./views/ShellWorkspace.vue";
import { useShell } from "./composables/useShell";
import SubProjectList from "./components/SubProjectList.vue";
import TerminalHeader from "./components/TerminalHeader.vue";
import AboutDialog from "./components/AboutDialog.vue";
import ConfigEditorDialog from "./components/ConfigEditorDialog.vue";
import SettingsHubDialog from "./components/SettingsHubDialog.vue";
import AppMenuBar from "./components/AppMenuBar.vue";
import { useTheme } from "./composables/useTheme";
import { mergeShortcuts, matchesShortcut } from "./utils/shortcuts";

export default {
  name: "App",
  components: { AppMenuBar, TerminalOutput, StatusBar, ProjectList, HomePage, ShellWorkspace, SubProjectList, TerminalHeader, AboutDialog, ConfigEditorDialog, SettingsHubDialog },
  setup() {
    const { isDark, themeMode, terminalPreset, loadTheme, applyThemeSettings } = useTheme();
    const projects = ref([]);
    const subProjects = ref([]);
    const outputLines = ref([]);
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
    // 终端输出行数上限，防止内存增长过快
    const MAX_OUTPUT_LINES = 2000;

    const enforceOutputLimit = () => {
      const overflow = outputLines.value.length - MAX_OUTPUT_LINES;
      if (overflow > 0) {
        outputLines.value.splice(0, overflow);
      }
    };

    // 左侧面板宽度控制
    const leftPanelWidth = ref(320);
    const minPanelWidth = 180;
    const maxPanelWidth = 640;
    const isResizing = ref(false);

    // 展开状态管理
    const expandedSubProjects = ref({});
    const expandedCommands = ref({});

    // 统一系统设置 Hub
    const settingsHubVisible = ref(false);
    const settingsSection = ref('general');
    const machineEditId = ref('');
    const machines = ref([]);
    const machinesLoading = ref(false);

    // 关于弹框
    const aboutVisible = ref(false);
    const aboutIntroHtml = ref('');
    const configEditorVisible = ref(false);
    // home | task | shell —— 任务与 Shell 互不打断，仅切换当前视图
    const activeView = ref('home');
    const shellMounted = ref(false);
    const shellMode = computed(() => activeView.value === 'shell');
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
      connect: connectShell,
      disconnect: disconnectShell,
      closeSession: closeShellSession,
      testMachine: testShellConnection,
    } = useShell();
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
      const base = `FlashDock ${ver}`;
      if (!sessionId.value) return base;
      return `${base} · 会话 ${sessionId.value.slice(0, 8)}`;
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
      aboutVisible.value = true;
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

    // 加载配置
    const loadConfig = async () => {
      try {
        console.log("开始加载配置...");
        console.log("App 对象:", App);

        const config = await App.GetConfig();
        console.log("原始配置数据:", config);

        projects.value = config.projects || [];
        console.log("设置的项目数据:", projects.value);
        console.log("项目数量:", projects.value.length);

        // 初始不自动选中项目，展示项目列表
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
        console.log("App 对象:", App);

        const config = await App.GetConfigForRefresh();
        console.log("刷新后的项目配置数据:", config);

        projects.value = config.projects || [];
        console.log("设置的项目数据:", projects.value);
        console.log("项目数量:", projects.value.length);

        // 刷新后保持在项目列表，避免误触发执行
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
    const selectProject = (project) => {
      selectedProject.value = project;
      selectedSubProject.value = null;
      activeView.value = 'task';

      // 显示该项目下的所有 SubProjects
      if (project.subprojects) {
        subProjects.value = project.subprojects.map(subproject => ({
          ...subproject,
          projectName: project.name,
          commandCount: subproject.commands ? subproject.commands.length : 0,
          stepCount: subproject.commands ? subproject.commands.reduce((total, command) => total + (command.steps?.length || 0), 0) : 0
        }));
      } else {
        subProjects.value = [];
      }

      // 重置展开状态
      expandedSubProjects.value = {};
      expandedCommands.value = {};

      console.log(`选择项目: ${project.name}, 找到 ${subProjects.value.length} 个 SubProjects`);
    };

    const backToProjectList = () => {
      // 回到首页但不销毁任务上下文，可与 Shell 来回切换
      activeView.value = 'home';
    };

    const resumeTaskView = () => {
      if (!selectedProject.value) return;
      activeView.value = 'task';
    };

    const enterShellMode = async () => {
      shellMounted.value = true;
      activeView.value = 'shell';
      await loadShellMachines();
      await syncSessions();
    };

    const leaveShellMode = () => {
      // 保留 Shell 会话与任务状态，仅回到首页
      activeView.value = 'home';
    };

    const openShellAndConnect = async (machineName) => {
      // 先发起连接（立即标记 connectingName），再切到 Shell 展示「连接中」
      const connectPromise = connectShell(machineName);
      await enterShellMode();
      await connectPromise;
    };

    const openSettingsHub = (section = 'general') => {
      settingsSection.value = section || 'general';
      settingsHubVisible.value = true;
    };

    const openShellMachineDialog = async () => {
      machineEditId.value = '';
      openSettingsHub('machines');
      await loadMachines();
      await loadShellMachines();
    };

    const openShellMachineEdit = async (machine) => {
      machineEditId.value = machine?.id || '';
      openSettingsHub('machines');
      await loadMachines();
      await loadShellMachines();
    };

    const onMachinesChanged = async () => {
      await loadMachines();
      await loadShellMachines();
      await homePageRef.value?.loadMachines?.();
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

    // 执行命令 (保持向后兼容)
    const executeCommand = async (cmd) => {
      // 为了向后兼容，如果有人调用这个方法，我们执行对应的 SubProject
      if (cmd.subprojectName && cmd.projectName) {
        const subProject = { name: cmd.subprojectName, projectName: cmd.projectName };
        return executeSubProject(subProject);
      }
    };

    // 停止命令 (保持向后兼容)
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
      Object.assign(status, currentStatus);
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

    // 键盘快捷键处理（可在系统设置中自定义，保存至 shortcuts.json）
    const handleKeyDown = (e) => {
      // 检查是否在输入框中，如果是则不处理快捷键
      if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.contentEditable === 'true') {
        return;
      }

      const sc = shortcutMap.value;

      // 终端搜索 — 任务模式；Shell 模式由 ShellWorkspace 处理
      if (matchesShortcut(e, sc.find)) {
        e.preventDefault();
        if (activeView.value !== 'shell') {
          terminalSearchVisible.value = true;
        }
        return;
      }

      if (matchesShortcut(e, sc.copy)) {
        e.preventDefault();
        copySelectedText();
        return;
      }

      if (matchesShortcut(e, sc.clearOutput)) {
        e.preventDefault();
        clearOutput();
        return;
      }

      if (matchesShortcut(e, sc.machineConfig)) {
        e.preventDefault();
        openMachineConfig();
        return;
      }

      if (matchesShortcut(e, sc.envVars)) {
        e.preventDefault();
        openWorkPathConfig();
        return;
      }

      if (matchesShortcut(e, sc.newWindow)) {
        e.preventDefault();
        App.NewWindow();
        return;
      }

      if (matchesShortcut(e, sc.refreshConfig)) {
        e.preventDefault();
        App.RefreshConfigMenuWithEvent();
        return;
      }

      if (matchesShortcut(e, sc.systemSettings)) {
        e.preventDefault();
        App.OpenSystemSettings();
        return;
      }

      // Escape 键关闭对话框
      if (e.key === 'Escape') {
        if (settingsHubVisible.value) {
          settingsHubVisible.value = false;
        }
        return;
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

    onMounted(() => {
      // 如果上一次刷新触发了页面重载，则在重载完成后再弹提示
      const pendingReloadToast = sessionStorage.getItem('pendingReloadToastMessage');
      if (pendingReloadToast) {
        sessionStorage.removeItem('pendingReloadToastMessage');
        ElMessage.success(pendingReloadToast);
      }

      loadConfig();
      loadTheme();
      loadShortcutMap();
      App.GetSessionInfo().then((info) => { sessionId.value = info.sessionId || ''; }).catch(() => { });
      App.GetAppVersion().then((v) => { appVersion.value = v || ''; }).catch(() => { appVersion.value = ''; });

      // 监听输出与执行状态事件（替代轮询）
      EventsOn("output:line", handleOutputLine);
      EventsOn("output:clear", handleOutputClear);
      EventsOn("execution:status", handleExecutionStatus);
      EventsOn("theme:changed", applyThemeSettings);
      EventsOn("shortcuts:changed", (data) => {
        shortcutMap.value = mergeShortcuts(data);
      });

      // 添加全局键盘事件监听器
      document.addEventListener('keydown', handleKeyDown);

      // 监听统一的操作结果事件
      EventsOn("operation:result", async (event) => {
        console.log("收到操作结果事件:", event);
        handleOperationEvent(event);
      });

      // 监听配置变更事件
      EventsOn("config:changed", async (data) => {
        console.log("收到 config:changed 事件:", data);
        // 显示全局加载状态
        isReloading.value = true;
        // 延迟一下再重新加载，让用户看到加载效果
        setTimeout(() => {
          window.location.reload();
        }, 200);
      });

      // 监听打开设置相关事件（统一进 Settings Hub）
      EventsOn("open:machine-config", async () => {
        await openMachineConfig();
      });
      EventsOn("open:workpath-config", async () => {
        await openWorkPathConfig();
      });
      EventsOn("open:about", () => { openAbout(); });
      EventsOn("open:config-editor", () => { configEditorVisible.value = true; });
      EventsOn("open:system-settings", () => { openSettingsHub('general'); });
      EventsOn("open:execution-history", () => { openSettingsHub('history'); });
    });

    // 机器配置相关方法
    const loadMachines = async () => {
      try {
        machinesLoading.value = true;
        const machinesData = await App.GetMachines();
        machines.value = machinesData || [];
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
        savingMachine.value = true;

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
        ElMessage.error('保存机器配置失败: ' + error.message);
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
      document.removeEventListener('keydown', handleKeyDown);
      try {
        EventsOff(
          "operation:result",
          "config:changed",
          "open:machine-config",
          "open:workpath-config",
          "open:about",
          "open:config-editor",
          "open:system-settings",
          "open:execution-history",
          "theme:changed",
          "shortcuts:changed",
          "output:line",
          "output:clear",
          "execution:status",
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
      machines,
      machinesLoading,
      settingsHubVisible,
      settingsSection,
      openSettingsHub,
      openMachineConfig,
      openWorkPathConfig,
      // 关于
      aboutVisible,
      aboutIntroHtml,
      openAbout,
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
      openShellAndConnect,
      connectShell,
      disconnectShell,
      closeShellSession,
      openShellMachineDialog,
      openShellMachineEdit,
      onMachinesChanged,
      homePageRef,
      testShellConnection,
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
  background: var(--app-bg);
  color: var(--app-text);
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
  background: white;
  padding: 32px;
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

.loading-icon {
  font-size: 32px;
  color: #409eff;
  animation: spin 1s linear infinite;
}

.loading-text {
  font-size: 16px;
  color: #606266;
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
  background-color: var(--app-panel-bg);
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
  background: rgba(64, 158, 255, 0.3);
}

.resize-handle:active {
  background: rgba(64, 158, 255, 0.5);
}

/* 拖拽时的视觉反馈 */
.left-panel.resizing {
  user-select: none;
}

.left-panel.resizing .resize-handle {
  background: rgba(64, 158, 255, 0.5);
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
</style>
