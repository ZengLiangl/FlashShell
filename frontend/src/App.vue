<template>
  <div class="app-container">
    <!-- 全局加载遮罩 -->
    <div v-if="isReloading" class="global-loading">
      <div class="loading-content">
        <el-icon class="loading-icon">
          <Loading />
        </el-icon>
        <div class="loading-text">正在重新加载...</div>
      </div>
    </div>

    <el-container class="main-container">
      <!-- 左侧面板 -->
      <el-aside width="400px" class="left-panel">
        <!-- 项目列表 -->
        <div class="panel-section">
          <div class="section-header">
            <h3>项目列表</h3>
            <div>
              <el-button size="small" @click="refreshConfig">
                <el-icon>
                  <Refresh />
                </el-icon>
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
            <div v-for="project in projects" :key="project.name" class="project-item"
              :class="{ active: selectedProject?.name === project.name }" @click="selectProject(project)">
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
            <div v-for="subProject in subProjects" :key="subProject.name" class="subproject-container">
              <div class="subproject-item">
                <div class="subproject-info">
                  <div class="subproject-header">
                    <el-button size="small" text @click="toggleSubProject(subProject.name)" class="expand-button">
                      <el-icon>
                        <ArrowRight v-if="!expandedSubProjects[subProject.name]" />
                        <ArrowDown v-else />
                      </el-icon>
                    </el-button>
                    <div class="subproject-title">
                      <div class="subproject-name">{{ subProject.name }}</div>
                      <div class="subproject-desc">{{ subProject.description }}</div>
                      <div class="subproject-meta">
                        <el-tag size="small" type="info">{{ subProject.commandCount }} 个命令</el-tag>
                      </div>
                    </div>
                  </div>
                </div>
                <div class="subproject-actions">
                  <el-button size="small" type="primary" @click="executeSubProject(subProject)"
                    :loading="isSubProjectRunning(subProject)"
                    :disabled="status.isRunning && !isSubProjectRunning(subProject)">
                    {{ isSubProjectRunning(subProject) ? "运行中" : "执行" }}
                  </el-button>
                  <el-button v-if="isSubProjectRunning(subProject)" size="small" type="danger"
                    @click="stopSubProject(subProject)">
                    停止
                  </el-button>
                </div>
              </div>

              <!-- Commands 展开区域 -->
              <div v-if="expandedSubProjects[subProject.name]" class="commands-container">
                <div v-for="command in subProject.commands" :key="command.name" class="command-container">
                  <div class="command-item">
                    <div class="command-header">
                      <el-button size="small" text @click="toggleCommand(subProject.name, command.name)"
                        class="expand-button">
                        <el-icon>
                          <ArrowRight v-if="!expandedCommands[`${subProject.name}-${command.name}`]" />
                          <ArrowDown v-else />
                        </el-icon>
                      </el-button>
                      <div class="command-info">
                        <div class="command-name">
                          <!-- <el-icon class="command-type-icon">
                            <Connection v-if="command.type === 'remote'" />
                            <Setting v-else />
                          </el-icon> -->
                          {{ command.name }}
                        </div>
                        <!-- <div class="command-desc">{{ command.description }}</div> -->
                      </div>
                    </div>
                    <div class="command-meta">
                      <el-tag size="small" :type="getCommandTagType(command.type)" effect="light">
                        {{ getCommandTypeText(command.type) }}
                      </el-tag>
                      <el-tag size="small" type="info" effect="plain">
                        {{ command.steps?.length || 0 }} 步骤
                      </el-tag>
                    </div>
                  </div>

                  <!-- Steps 展开区域 -->
                  <div v-if="expandedCommands[`${subProject.name}-${command.name}`]" class="steps-container">
                    <div v-for="(step, index) in command.steps" :key="index" class="step-item">
                      <div class="step-number">{{ index + 1 }}</div>
                      <div class="step-content">
                        <div class="step-command">{{ step }}</div>
                      </div>
                    </div>
                  </div>
                </div>
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
              <el-icon>
                <Delete />
              </el-icon>
              清空
            </el-button>
            <el-button size="small" @click="refreshOutput">
              <el-icon>
                <Refresh />
              </el-icon>
              刷新
            </el-button>
          </div>
        </div>

        <!-- 进度条区域 -->
        <div v-if="status.isRunning" class="progress-section">
          <div class="progress-info">
            <div class="progress-text">
              <span class="project-name">{{ status.subProjectName }}</span>
              <span v-if="status.currentCommand" class="current-command">
                正在执行: {{ status.currentCommand }}
              </span>
            </div>
            <div class="progress-stats">
              {{ Math.max(1, status.completedCommands + 1) }}/{{ status.totalCommands }} 命令
            </div>
          </div>
          <el-progress :percentage="progressPercentage" :status="progressStatus" :stroke-width="8" :show-text="true"
            class="execution-progress" />
        </div>

        <div class="terminal-output" ref="terminalOutput">
          <div v-for="(line, index) in outputLines" :key="index" class="output-line" :class="{
            'error-line': line.isError,
            'success-line': line.isSuccess,
          }" v-html="line.html">
          </div>
          <div v-if="outputLines.length === 0" class="empty-output">
            等待命令输出...
          </div>
        </div>
      </el-main>
    </el-container>

    <!-- 状态栏 - 始终显示在底部 -->
    <div class="status-bar">
      <div class="status-info">
        <!-- 执行状态 -->
        <el-tag v-if="status.isRunning" type="warning" size="small">
          <el-icon>
            <Loading />
          </el-icon>
          执行中: {{ status.subProjectName }}
          <span v-if="status.currentCommand"> - {{ status.currentCommand }}</span>
        </el-tag>
        <el-tag v-else type="success" size="small">
          <el-icon>
            <Check />
          </el-icon>
          就绪
        </el-tag>



        <!-- 项目信息 - 显示当前选中的项目 -->
        <el-tag v-if="selectedProject && !status.isRunning" size="small" type="info">
          项目: {{ selectedProject.name }}
        </el-tag>
      </div>

      <div class="status-actions">
        <!-- 执行控制按钮 -->
        <el-button v-if="status.isRunning" size="small" type="danger" @click="stopAllCommands">
          停止执行
        </el-button>

        <!-- 应用信息 -->
        <span class="app-info">Quick Cmd v1.2.0</span>
      </div>
    </div>

    <!-- 机器配置弹框 -->
    <el-dialog v-model="machineConfigVisible" title="机器配置管理" width="80%" :before-close="handleMachineConfigClose">
      <div class="machine-config-container">
        <!-- 机器列表 -->
        <div class="machine-list">
          <div class="list-header">
            <h4>机器列表</h4>
            <el-button type="primary" @click="addMachine">
              <el-icon>
                <Plus />
              </el-icon>
              添加机器
            </el-button>
          </div>

          <el-table :data="machines" style="width: 100%" v-loading="machinesLoading">
            <el-table-column prop="name" label="机器名称" width="150" />
            <el-table-column prop="key_file" label="密钥文件" overflow-tooltip />
            <el-table-column label="操作" width="250">
              <template #default="scope">
                <el-button size="small" @click="editMachine(scope.row)">编辑</el-button>
                <el-button size="small" @click="testConnection(scope.row)" :loading="scope.row.testing">
                  测试连接
                </el-button>
                <el-button size="small" type="danger" @click="deleteMachine(scope.row)">
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </el-dialog>

    <!-- 机器编辑弹框 -->
    <el-dialog v-model="machineEditVisible" :title="editingMachine ? '编辑机器' : '添加机器'" width="600px">
      <el-form :model="machineForm" :rules="machineRules" ref="machineFormRef" label-width="100px">
        <el-form-item label="机器名称" prop="name">
          <el-input v-model="machineForm.name" placeholder="请输入机器名称" />
        </el-form-item>

        <el-form-item label="密钥文件" prop="key_file">
          <div class="key-file-input">
            <el-input v-model="machineForm.key_file" placeholder="请选择密钥文件" readonly />
            <el-button type="primary" @click="selectKeyFile">选择文件</el-button>
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
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="machineEditVisible = false">取消</el-button>
          <el-button type="primary" @click="saveMachine" :loading="savingMachine">
            {{ editingMachine ? '更新' : '添加' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 环境变量配置弹框 -->
    <el-dialog v-model="workPathConfigVisible" title="环境变量配置管理" width="80%" :before-close="handleWorkPathConfigClose">
      <div class="workpath-config-container">
        <!-- 环境变量列表 -->
        <div class="workpath-list">
          <div class="list-header">
            <h4>环境变量列表</h4>
            <el-button type="primary" @click="addWorkPath">
              <el-icon>
                <Plus />
              </el-icon>
              添加环境变量
            </el-button>
          </div>

          <el-table :data="Object.entries(workPaths).map(([key, value]) => ({ key, value }))" style="width: 100%"
            v-loading="workPathsLoading">
            <el-table-column prop="key" label="变量名" width="200" />
            <el-table-column prop="value" label="变量值" overflow-tooltip />
            <el-table-column label="操作" width="200">
              <template #default="scope">
                <el-button size="small" @click="editWorkPath(scope.row.key)">编辑</el-button>
                <el-button size="small" type="danger" @click="deleteWorkPath(scope.row.key)">
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </el-dialog>

    <!-- 环境变量编辑弹框 -->
    <el-dialog v-model="workPathEditVisible" :title="editingWorkPath ? '编辑环境变量' : '添加环境变量'" width="500px">
      <el-form :model="workPathForm" :rules="workPathRules" ref="workPathFormRef" label-width="100px">
        <el-form-item label="变量名" prop="key">
          <el-input v-model="workPathForm.key" placeholder="请输入变量名（如：PROJECT_HOME）" :disabled="!!editingWorkPath" />
        </el-form-item>

        <el-form-item label="变量值" prop="value">
          <el-input v-model="workPathForm.value" placeholder="请输入变量值（如：/home/user/projects）" />
        </el-form-item>

        <el-form-item label="使用说明">
          <div class="usage-info">
            <p>• 变量名只能包含大写字母、数字和下划线</p>
            <p>• 变量名必须以字母或下划线开头</p>
            <p>• 在配置文件中可以使用 ${变量名} 来引用这些环境变量</p>
            <p>• 例如：workdir: "${PROJECT_HOME}/my-project"</p>
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="workPathEditVisible = false">取消</el-button>
          <el-button type="primary" @click="saveWorkPath" :loading="savingWorkPath">
            {{ editingWorkPath ? '更新' : '添加' }}
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, computed, onMounted, onUnmounted, nextTick } from "vue";
import { ElMessage } from "element-plus";
import * as App from "../wailsjs/go/app/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import Convert from "ansi-to-html";

export default {
  name: "App",
  setup() {
    const projects = ref([]);
    const subProjects = ref([]);
    const outputLines = ref([]);
    const selectedProject = ref(null);
    const selectedSubProject = ref(null);
    const isReloading = ref(false);
    const status = reactive({
      isRunning: false,
      command: "",
    });
    const terminalOutput = ref(null);

    // 展开状态管理
    const expandedSubProjects = ref({});
    const expandedCommands = ref({});

    // 机器配置相关
    const machineConfigVisible = ref(false);
    const machineEditVisible = ref(false);
    const machines = ref([]);
    const machinesLoading = ref(false);
    const savingMachine = ref(false);
    const editingMachine = ref(null);
    const machineFormRef = ref(null);

    // 环境变量配置相关
    const workPathConfigVisible = ref(false);
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
        { pattern: /^[A-Z_][A-Z0-9_]*$/, message: '变量名只能包含大写字母、数字和下划线，且必须以字母或下划线开头', trigger: 'blur' }
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

        if (projects.value.length > 0) {
          console.log("第一个项目:", projects.value[0]);
          // 默认选中第一个项目
          selectProject(projects.value[0]);
        }
      } catch (error) {
        console.error("加载配置失败:", error);
        console.error("错误详情:", error.stack);
      }
    };

    // 刷新配置
    const refreshConfig = async () => {
      try {
        isReloading.value = true;
        setTimeout(() => {
          // await loadConfig();
          window.location.reload();
        }, 200);
      } catch (error) {
        console.error("刷新配置失败:", error);
        ElMessage.error("刷新配置失败: " + error.message);
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

      // 重置展开状态
      expandedSubProjects.value = {};
      expandedCommands.value = {};

      console.log(`选择项目: ${project.name}, 找到 ${subProjects.value.length} 个 SubProjects`);
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
            isError: line.includes('STDERR') || line.includes('失败') || line.includes('错误') || line.includes('Error'),
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

    // 计算进度百分比
    const progressPercentage = computed(() => {
      if (!status.isRunning || status.totalCommands === 0) {
        return 0;
      }
      // 将进度映射到1-100的范围内
      const temp = (Math.max(1, status.completedCommands + 1) == status.totalCommands) ? status.completedCommands : Math.max(1, status.completedCommands + 1)
      const completedRatio = temp / status.totalCommands;
      return Math.round(completedRatio * 99);
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

    onMounted(() => {
      loadConfig();
      startOutputPolling();

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

      // 监听打开机器配置事件
      EventsOn("open:machine-config", async (data) => {
        console.log("收到 open:machine-config 事件:", data);
        await openMachineConfig();
      });

      // 监听打开环境变量配置事件
      EventsOn("open:workpath-config", async (data) => {
        console.log("收到 open:workpath-config 事件:", data);
        await openWorkPathConfig();
      });
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
      machineConfigVisible.value = true;
      await loadMachines();
    };

    const handleMachineConfigClose = () => {
      machineConfigVisible.value = false;
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
      workPathConfigVisible.value = true;
      await loadWorkPaths();
    };

    const handleWorkPathConfigClose = () => {
      workPathConfigVisible.value = false;
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

      // 根据消息类型显示不同的提示
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

      // 如果需要重新加载，显示加载状态并重新加载
      if (event.needReload) {
        setTimeout(() => {
          isReloading.value = true;
          window.location.reload();
        }, 1500);
      }
    };

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
      isReloading,
      status,
      terminalOutput,
      progressPercentage,
      progressStatus,
      expandedSubProjects,
      expandedCommands,
      refreshConfig,
      debugConfig,
      selectProject,
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
      // 机器配置相关
      machineConfigVisible,
      machineEditVisible,
      machines,
      machinesLoading,
      savingMachine,
      editingMachine,
      machineFormRef,
      machineForm,
      machineRules,
      openMachineConfig,
      handleMachineConfigClose,
      addMachine,
      editMachine,
      saveMachine,
      deleteMachine,
      testConnection,
      selectKeyFile,
      // 环境变量配置相关
      workPathConfigVisible,
      workPathEditVisible,
      workPaths,
      workPathsLoading,
      savingWorkPath,
      editingWorkPath,
      workPathFormRef,
      workPathForm,
      workPathRules,
      openWorkPathConfig,
      handleWorkPathConfigClose,
      addWorkPath,
      editWorkPath,
      saveWorkPath,
      deleteWorkPath,
      // 事件处理
      handleOperationEvent,
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
  height: calc(100vh - 40px);
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
  margin-top: 6px;
}

.subproject-actions {
  display: flex;
  gap: 8px;
}

.subproject-container {
  margin-bottom: 8px;
}

.subproject-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.expand-button {
  padding: 4px !important;
  min-width: auto !important;
  width: 24px;
  height: 24px;
}

.subproject-title {
  flex: 1;
}

.commands-container {
  margin-left: 32px;
  margin-top: 8px;
  border-left: 2px solid #e4e7ed;
  padding-left: 12px;
}

.command-container {
  margin-bottom: 8px;
}

.command-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #fafafa;
  border-radius: 4px;
  border: 1px solid #f0f0f0;
}

.command-header {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.command-info {
  flex: 1;
}

.command-name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  color: #606266;
  font-size: 13px;
  margin-bottom: 2px;
}

.command-type-icon {
  font-size: 14px;
  color: #409eff;
}

.command-desc {
  font-size: 11px;
  color: #909399;
}

.command-meta {
  display: flex;
  gap: 6px;
  align-items: center;
}

.steps-container {
  margin-left: 32px;
  margin-top: 6px;
  border-left: 2px solid #f0f0f0;
  padding-left: 12px;
}

.step-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 6px 0;
  border-bottom: 1px solid #f5f5f5;
}

.step-item:last-child {
  border-bottom: none;
}

.step-number {
  background: #409eff;
  color: white;
  border-radius: 50%;
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: 500;
  flex-shrink: 0;
  margin-top: 2px;
}

.step-content {
  flex: 1;
}

.step-command {
  font-family: "Consolas", "Monaco", "Courier New", monospace;
  font-size: 12px;
  color: #303133;
  background: #f8f9fa;
  padding: 4px 8px;
  border-radius: 3px;
  border: 1px solid #e9ecef;
  margin-bottom: 4px;
}

.step-desc {
  font-size: 11px;
  color: #909399;
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

.progress-section {
  padding: 12px 16px;
  background: #f8f9fa;
  border-bottom: 1px solid #e4e7ed;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.progress-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.project-name {
  font-weight: 600;
  color: #303133;
  font-size: 14px;
}

.current-command {
  font-size: 12px;
  color: #606266;
}

.progress-stats {
  font-size: 12px;
  color: #909399;
  font-weight: 500;
}

.execution-progress {
  margin: 0;
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
  padding: 0 16px;
  background: #f5f7fa;
  border-top: 1px solid #e4e7ed;
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
  color: #909399;
  font-weight: 500;
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
</style>
