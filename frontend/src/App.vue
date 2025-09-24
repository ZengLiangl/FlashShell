<template>
  <div class="app-container">
    <el-container style="height: 100vh">
      <!-- 左侧面板 -->
      <el-aside width="400px" class="left-panel">
        <!-- 项目列表 -->
        <div class="panel-section">
          <div class="section-header">
            <h3>项目列表</h3>
            <div>
              <el-button size="small" @click="refreshConfig">
                <el-icon><Refresh /></el-icon>
              </el-button>
              <el-button size="small" @click="debugConfig" type="warning">
                调试
              </el-button>
            </div>
          </div>
          <div v-if="projects.length === 0" class="no-projects">
            <p>暂无项目配置</p>
            <p>项目数量: {{ projects.length }}</p>
          </div>
          <div v-else class="project-list">
            <div
              v-for="project in projects"
              :key="project.name"
              class="project-item"
              :class="{ active: selectedProject?.name === project.name }"
              @click="selectProject(project)"
            >
              <div class="project-name">{{ project.name }}</div>
              <div class="project-desc">{{ project.description }}</div>
            </div>
          </div>
        </div>

        <!-- SubProject 列表 -->
        <div class="panel-section">
          <div class="section-header">
            <h3>可执行项目</h3>
            <el-tag v-if="selectedProject" size="small">{{
              selectedProject.name
            }}</el-tag>
          </div>
          <div v-if="subProjects.length > 0" class="subproject-list">
            <div v-for="subProject in subProjects" :key="subProject.name" class="subproject-item">
              <div class="subproject-info">
                <div class="subproject-name">{{ subProject.name }}</div>
                <div class="subproject-desc">{{ subProject.description }}</div>
                <div class="subproject-meta">
                  <el-tag size="small" type="info">{{ subProject.commandCount }} 个命令</el-tag>
                </div>
              </div>
              <div class="subproject-actions">
                <el-button
                  size="small"
                  type="primary"
                  @click="executeSubProject(subProject)"
                  :loading="isSubProjectRunning(subProject)"
                  :disabled="status.isRunning && !isSubProjectRunning(subProject)"
                >
                  {{ isSubProjectRunning(subProject) ? "运行中" : "执行" }}
                </el-button>
                <el-button
                  v-if="isSubProjectRunning(subProject)"
                  size="small"
                  type="danger"
                  @click="stopSubProject(subProject)"
                >
                  停止
                </el-button>
              </div>
            </div>
          </div>
          <el-empty v-else description="请选择项目查看可执行项目" />
        </div>
      </el-aside>

      <!-- 右侧终端输出 -->
      <el-main class="terminal-container">
        <div class="terminal-header">
          <h3>终端输出</h3>
          <div class="terminal-actions">
            <el-button size="small" @click="clearOutput">
              <el-icon><Delete /></el-icon>
              清空
            </el-button>
            <el-button size="small" @click="refreshOutput">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </div>
        <div class="terminal-output" ref="terminalOutput">
          <div
            v-for="(line, index) in outputLines"
            :key="index"
            class="output-line"
            :class="{
              'error-line': line.isError,
              'success-line': line.isSuccess,
            }"
            v-html="line.html"
          >
          </div>
          <div v-if="outputLines.length === 0" class="empty-output">
            等待命令输出...
          </div>
        </div>
      </el-main>
    </el-container>

    <!-- 状态栏 -->
    <div class="status-bar">
      <div class="status-info">
        <el-tag v-if="status.isRunning" type="warning" size="small">
          <el-icon><Loading /></el-icon>
          执行中: {{ status.subProjectName }}
          <span v-if="status.currentCommand"> - {{ status.currentCommand }}</span>
        </el-tag>
        <el-tag v-else type="success" size="small">
          <el-icon><Check /></el-icon>
          就绪
        </el-tag>
        <el-tag v-if="status.isRunning && status.totalCommands > 0" size="small" type="info">
          进度: {{ status.completedCommands }}/{{ status.totalCommands }}
        </el-tag>
      </div>
      <div class="status-actions">
        <el-button
          v-if="status.isRunning"
          size="small"
          type="danger"
          @click="stopAllCommands"
        >
          停止执行
        </el-button>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, reactive, onMounted, onUnmounted, nextTick } from "vue";
import * as App from "../wailsjs/go/app/App";
import Convert from "ansi-to-html";

export default {
  name: "App",
  setup() {
    const projects = ref([]);
    const subProjects = ref([]);
    const outputLines = ref([]);
    const selectedProject = ref(null);
    const selectedSubProject = ref(null);
    const status = reactive({
      isRunning: false,
      command: "",
    });
    const terminalOutput = ref(null);

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

        if (projects.value.length > 0) {
          console.log("第一个项目:", projects.value[0]);
        }
      } catch (error) {
        console.error("加载配置失败:", error);
        console.error("错误详情:", error.stack);
      }
    };

    // 刷新配置
    const refreshConfig = () => {
      // 刷新整个页面
      window.location.reload();
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
        alert(`配置加载成功！项目数量: ${config.projects?.length || 0}`);
      } catch (error) {
        console.error("调试失败:", error);
        alert(`调试失败: ${error.message}`);
      }
    };

    // 选择项目
    const selectProject = (project) => {
      selectedProject.value = project;
      selectedSubProject.value = null;
      
      // 显示该项目下的所有 SubProjects
      if (project.subprojects) {
        subProjects.value = project.subprojects.map(subproject => ({
          ...subproject,
          projectName: project.name,
          commandCount: subproject.commands ? subproject.commands.length : 0
        }));
      } else {
        subProjects.value = [];
      }
      
      console.log(`选择项目: ${project.name}, 找到 ${subProjects.value.length} 个 SubProjects`);
    };

    // 执行 SubProject
    const executeSubProject = async (subProject) => {
      if (!selectedProject.value) {
        return;
      }

      try {
        await App.ExecuteSubProject(
          subProject.projectName,
          subProject.name
        );
        // 开始轮询输出
        startOutputPolling();
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

    // 获取输出
    const getOutput = async () => {
      try {
        const output = await App.GetOutput();
        if (output && output.length > 0) {
          // 处理每行输出的 ANSI 转义序列
          const processedOutput = output.map(line => ({
            raw: line,
            html: processAnsiOutput(line),
            isError: line.includes('STDERR') || line.includes('失败') || line.includes('错误'),
            isSuccess: line.includes('完成') || line.includes('成功')
          }));
          
          outputLines.value.push(...processedOutput);
          
          // 滚动到底部
          nextTick(() => {
            if (terminalOutput.value) {
              terminalOutput.value.scrollTop =
                terminalOutput.value.scrollHeight;
            }
          });
        }
      } catch (error) {
        console.error("获取输出失败:", error);
      }
    };

    // 获取状态
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
        await App.ClearOutput();
        outputLines.value = [];
      } catch (error) {
        console.error("清空输出失败:", error);
      }
    };

    // 刷新输出
    const refreshOutput = () => {
      getOutput();
    };

    // 开始输出轮询
    let outputTimer = null;
    const startOutputPolling = () => {
      if (outputTimer) {
        clearInterval(outputTimer);
      }
      outputTimer = setInterval(() => {
        getOutput();
        getStatus();
      }, 500);
    };

    // 停止输出轮询
    const stopOutputPolling = () => {
      if (outputTimer) {
        clearInterval(outputTimer);
        outputTimer = null;
      }
    };

    onMounted(() => {
      console.log("组件已挂载，开始加载配置...");
      loadConfig();
      startOutputPolling();
    });

    // 组件卸载时清理定时器
    onUnmounted(() => {
      stopOutputPolling();
    });

    return {
      projects,
      subProjects,
      outputLines,
      selectedProject,
      selectedSubProject,
      status,
      terminalOutput,
      refreshConfig,
      debugConfig,
      selectProject,
      executeSubProject,
      stopSubProject,
      executeCommand,
      stopCommand,
      stopAllCommands,
      isSubProjectRunning,
      isCommandRunning,
      clearOutput,
      refreshOutput,
    };
  },
};
</script>

<style scoped>
.app-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
}

.left-panel {
  border-right: 1px solid #e4e7ed;
  background-color: #f5f7fa;
  display: flex;
  flex-direction: column;
}

.panel-section {
  flex: 1;
  padding: 16px;
  border-bottom: 1px solid #e4e7ed;
}

.panel-section:last-child {
  border-bottom: none;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.section-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.project-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.project-item {
  padding: 12px;
  background: white;
  border-radius: 6px;
  border: 1px solid #e4e7ed;
  cursor: pointer;
  transition: all 0.2s;
}

.project-item:hover {
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.1);
}

.project-item.active {
  border-color: #409eff;
  background: #ecf5ff;
}

.project-name {
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
}

.project-desc {
  font-size: 12px;
  color: #909399;
}

.subproject-list {
  max-height: 300px;
  overflow-y: auto;
}

.subproject-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  margin-bottom: 8px;
  background: white;
  border-radius: 6px;
  border: 1px solid #e4e7ed;
}

.subproject-info {
  flex: 1;
}

.subproject-name {
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
}

.subproject-desc {
  font-size: 12px;
  color: #909399;
  margin-bottom: 6px;
}

.subproject-meta {
  display: flex;
  gap: 6px;
  align-items: center;
}

.subproject-actions {
  display: flex;
  gap: 8px;
}

/* 保持向后兼容的样式 */
.command-list {
  max-height: 300px;
  overflow-y: auto;
}

.command-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  margin-bottom: 8px;
  background: white;
  border-radius: 6px;
  border: 1px solid #e4e7ed;
}

.command-info {
  flex: 1;
}

.command-name {
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
}

.command-desc {
  font-size: 12px;
  color: #909399;
  margin-bottom: 6px;
}

.command-meta {
  display: flex;
  gap: 6px;
  align-items: center;
}

.command-actions {
  display: flex;
  gap: 8px;
}

.terminal-container {
  display: flex;
  flex-direction: column;
  padding: 0;
}

.terminal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #e4e7ed;
  background: #f5f7fa;
}

.terminal-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.terminal-actions {
  display: flex;
  gap: 8px;
}

.terminal-output {
  flex: 1;
  padding: 16px;
  background: #1e1e1e;
  color: #d4d4d4;
  font-family: "Consolas", "Monaco", "Courier New", monospace;
  font-size: 13px;
  line-height: 1.4;
  overflow-y: auto;
  white-space: pre-wrap;
}

.output-line {
  margin-bottom: 2px;
  word-break: break-all;
}

.error-line {
  color: #f56c6c;
}

.success-line {
  color: #67c23a;
}

.empty-output {
  color: #909399;
  text-align: center;
  margin-top: 50px;
}

.status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 16px;
  background: #f5f7fa;
  border-top: 1px solid #e4e7ed;
  height: 40px;
}

.status-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-actions {
  display: flex;
  gap: 8px;
}

.no-projects {
  text-align: center;
  color: #909399;
  padding: 20px;
}

/* ANSI 颜色支持 */
.terminal-output :deep(.ansi-black-fg) { color: #000000; }
.terminal-output :deep(.ansi-red-fg) { color: #cd3131; }
.terminal-output :deep(.ansi-green-fg) { color: #0dbc79; }
.terminal-output :deep(.ansi-yellow-fg) { color: #e5e510; }
.terminal-output :deep(.ansi-blue-fg) { color: #2472c8; }
.terminal-output :deep(.ansi-magenta-fg) { color: #bc3fbc; }
.terminal-output :deep(.ansi-cyan-fg) { color: #11a8cd; }
.terminal-output :deep(.ansi-white-fg) { color: #e5e5e5; }
.terminal-output :deep(.ansi-bright-black-fg) { color: #666666; }
.terminal-output :deep(.ansi-bright-red-fg) { color: #f14c4c; }
.terminal-output :deep(.ansi-bright-green-fg) { color: #23d18b; }
.terminal-output :deep(.ansi-bright-yellow-fg) { color: #f5f543; }
.terminal-output :deep(.ansi-bright-blue-fg) { color: #3b8eea; }
.terminal-output :deep(.ansi-bright-magenta-fg) { color: #d670d6; }
.terminal-output :deep(.ansi-bright-cyan-fg) { color: #29b8db; }
.terminal-output :deep(.ansi-bright-white-fg) { color: #ffffff; }

/* ANSI 背景颜色 */
.terminal-output :deep(.ansi-black-bg) { background-color: #000000; }
.terminal-output :deep(.ansi-red-bg) { background-color: #cd3131; }
.terminal-output :deep(.ansi-green-bg) { background-color: #0dbc79; }
.terminal-output :deep(.ansi-yellow-bg) { background-color: #e5e510; }
.terminal-output :deep(.ansi-blue-bg) { background-color: #2472c8; }
.terminal-output :deep(.ansi-magenta-bg) { background-color: #bc3fbc; }
.terminal-output :deep(.ansi-cyan-bg) { background-color: #11a8cd; }
.terminal-output :deep(.ansi-white-bg) { background-color: #e5e5e5; }

/* ANSI 样式 */
.terminal-output :deep(.ansi-bold) { font-weight: bold; }
.terminal-output :deep(.ansi-dim) { opacity: 0.5; }
.terminal-output :deep(.ansi-italic) { font-style: italic; }
.terminal-output :deep(.ansi-underline) { text-decoration: underline; }
.terminal-output :deep(.ansi-strikethrough) { text-decoration: line-through; }
</style>
