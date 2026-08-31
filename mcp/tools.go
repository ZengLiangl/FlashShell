package mcp

import (
	"context"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func addTool[In any](s *Service, name, desc string, serverOf func(In) string, previewOf func(In) string, h func(context.Context, In) (any, error)) {
	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{Name: name, Description: desc},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in In) (*mcpsdk.CallToolResult, any, error) {
			start := time.Now()
			server := ""
			preview := name
			if serverOf != nil {
				server = serverOf(in)
			}
			if previewOf != nil {
				preview = previewOf(in)
			}
			viaApproval, reason, err := s.gate(ctx, name, server, preview, in)
			if err != nil {
				s.record(ctx, name, server, in, err.Error(), classifyDecision(err.Error()), reason, start, err)
				return toolErr(err)
			}
			out, err := h(ctx, in)
			if err != nil {
				s.record(ctx, name, server, in, err.Error(), classifyDecision(err.Error()), reason, start, err)
				return toolErr(err)
			}
			text := ""
			if str, ok := out.(string); ok {
				text = str
			} else {
				text = mustJSON(out)
			}
			decision := "auto"
			if viaApproval {
				decision = "approved"
			}
			s.record(ctx, name, server, in, text, decision, reason, start, nil)
			if str, ok := out.(string); ok {
				return textOK(str)
			}
			return jsonOK(out)
		})
}

func (s *Service) registerTools() {
	addTool(s, "list_servers",
		"列出 FlashShell 中已配置的服务器（只返回别名/主机/端口/用户名/标签/AI 策略档位，绝不含任何凭据）。所有需要 server 参数的工具，server 用这里返回的 alias。",
		nil, nil, func(ctx context.Context, _ EmptyArgs) (any, error) { return s.listServers(ctx) })

	addTool(s, "ssh_exec",
		"在指定服务器（用别名）上执行一条 shell 命令。会经 AI 操作策略引擎裁决：可能直接放行、进审批队列等用户批准、被拒绝（如该服务器对 AI 禁用/只读、未开 sudo、全局已锁定），或命中危险命令黑名单被拦截；无论结果都会写审计。",
		func(a SshExecArgs) string { return a.Server },
		func(a SshExecArgs) string { return a.Command },
		s.handleSSHExec)

	addTool(s, "ssh_exec_multi",
		"在多台服务器上并发执行同一条命令（最多 50 台，内部并发上限 10）。每台独立过策略引擎 + 审计 + 出口脱敏。",
		func(a SshExecMultiArgs) string {
			if len(a.Servers) > 0 {
				return a.Servers[0]
			}
			return ""
		},
		func(a SshExecMultiArgs) string { return a.Command },
		s.handleSSHExecMulti)

	addTool(s, "ssh_exec_script",
		"在指定服务器上执行一段多行脚本（bash/sh/python3/python，默认 bash）。脚本会被 base64 包装后在远端解码执行；策略引擎按原始脚本明文判定。",
		func(a SshExecScriptArgs) string { return a.Server },
		func(a SshExecScriptArgs) string { return a.Script },
		s.handleSSHExecScript)

	addTool(s, "system_info",
		"返回服务器的系统信息（内核 + 主机名 + uptime）。只读工具：在该服务器 AI 策略为 readonly 及以上档位都直接放行。",
		func(a ServerOnly) string { return a.Server }, nil, s.handleSystemInfo)

	addTool(s, "disk_usage",
		"查看服务器磁盘使用（df -hT）。可选 path 指定某个挂载点/路径。只读工具。",
		func(a DiskUsageArgs) string { return a.Server },
		func(a DiskUsageArgs) string {
			if a.Path != nil {
				return *a.Path
			}
			return ""
		}, s.handleDiskUsage)

	addTool(s, "port_check",
		"检查指定端口是否在监听（ss -tlnH sport = :<port>）。返回 stdout 非空 = 在监听。只读工具。",
		func(a PortCheckArgs) string { return a.Server }, nil, s.handlePortCheck)

	addTool(s, "service_status",
		"查看 systemd 服务状态（systemctl status <service> --no-pager --lines=20）。只读工具。",
		func(a ServiceStatusArgs) string { return a.Server },
		func(a ServiceStatusArgs) string { return a.Service }, s.handleServiceStatus)

	addTool(s, "tail_log",
		"查看远端文件末尾若干行（tail -n <lines> <path>）。常用于看日志。lines 默认 200，最多 5000。只读工具。",
		func(a TailLogArgs) string { return a.Server },
		func(a TailLogArgs) string { return a.Path }, s.handleTailLog)

	addTool(s, "sftp_list",
		"列远端目录（SFTP）。返回 entries: [{name, isDir, size, mtime, isSymlink, permissions, owner, uid}]。只读工具。",
		func(a SftpListArgs) string { return a.Server },
		func(a SftpListArgs) string {
			if a.Path != nil {
				return *a.Path
			}
			return ""
		}, s.handleSftpList)

	addTool(s, "sftp_read",
		"读远端小文件（SFTP）。utf-8 内容直接返回 content；非 utf-8 退化为 base64。默认 256 KiB 上限，最高 4 MiB。只读工具。",
		func(a SftpReadArgs) string { return a.Server },
		func(a SftpReadArgs) string { return a.Path }, s.handleSftpRead)

	addTool(s, "sftp_write",
		"写（覆盖创建）远端文件（SFTP）—— 仅用于现写的小文本。已存在于本地磁盘的文件请改用 sftp_upload。敏感路径黑名单任何档位都拦。content/content_base64 二选一，单次 ≤ 16 MiB。",
		func(a SftpWriteArgs) string { return a.Server },
		func(a SftpWriteArgs) string { return a.Path }, s.handleSftpWrite)

	addTool(s, "sftp_upload",
		"把本机一个文件分块上传到远端。流式分块、无大小上限。改动型：按该服务器档位走策略 + 敏感路径黑名单。",
		func(a SftpUploadArgs) string { return a.Server },
		func(a SftpUploadArgs) string { return a.RemotePath }, s.handleSftpUpload)

	addTool(s, "evaluate_skills",
		"传入用户当前的问题/任务（prompt），返回命中相关触发词的技能列表（按命中数 + 来源优先级排序，含 name/source/description/triggers/hitCount）。元工具，不操作服务器。",
		nil, nil, func(ctx context.Context, a EvaluateSkillsArgs) (any, error) {
			hits := s.knowledge.Evaluate(a.Prompt)
			if hits == nil {
				hits = []SkillHit{}
			}
			return map[string]any{"skills": hits}, nil
		})

	addTool(s, "get_skill",
		"读取指定技能的 SKILL.md 正文（按触发词把对应技能拉进上下文用）。元工具。",
		nil, func(a NameOnly) string { return a.Name },
		func(ctx context.Context, a NameOnly) (any, error) {
			body, err := s.knowledge.GetSkill(a.Name)
			if err != nil {
				return nil, err
			}
			return map[string]any{"name": a.Name, "content": body}, nil
		})

	addTool(s, "list_skills",
		"列出全局技能库里的可配置技能（用户自定义存库 + 内置随应用分发）。元工具，不受策略约束。",
		nil, nil, func(ctx context.Context, _ EmptyArgs) (any, error) {
			return map[string]any{"skills": s.knowledge.ListSkillNames()}, nil
		})

	addTool(s, "recall_experience",
		"在全局经验库里检索过往经验（关键字 substring，不区分大小写）。元工具。",
		nil, nil, func(ctx context.Context, a RecallExperienceArgs) (any, error) {
			q := ""
			if a.Query != nil {
				q = *a.Query
			}
			return map[string]any{"hits": s.knowledge.Recall(q)}, nil
		})

	addTool(s, "list_runbooks",
		"列出全局 Runbook 库里固化的多步操作（按名字，不带 .md 后缀）。元工具。",
		nil, nil, func(ctx context.Context, _ EmptyArgs) (any, error) {
			return map[string]any{"runbooks": s.knowledge.ListRunbooks()}, nil
		})

	addTool(s, "run_runbook",
		"运行全局 Runbook 库里的 name：解析其中 ```bash on:<alias> ... ``` 代码块为步骤序列，逐步走 ssh_exec。每步独立过策略 + 审批 + 审计。",
		nil, func(a NameOnly) string { return a.Name }, s.handleRunRunbook)

	addTool(s, "list_installed_services",
		"列出已装服务凭据（id + serverAlias + kind + label + installPath；只返元数据不含字段值）。",
		func(a ListInstalledArgs) string {
			if a.Server != nil {
				return *a.Server
			}
			return ""
		}, nil, func(ctx context.Context, a ListInstalledArgs) (any, error) {
			sv := ""
			if a.Server != nil {
				sv = *a.Server
			}
			return map[string]any{"items": s.vault.ListMeta(sv)}, nil
		})

	addTool(s, "save_credential",
		"把已经装好的服务的真实凭据手动入「服务凭据」库。敏感字段写 fieldsFromVault，AI 全程不见明文。",
		func(a SaveCredentialArgs) string { return a.Server },
		func(a SaveCredentialArgs) string { return a.Kind + " " + a.Label }, s.handleSaveCredential)

	addTool(s, "delete_installed_service",
		"硬删一条服务凭据登记（DELETE 行；含加密字段值）。不动远端，只清 FlashShell 本地。",
		nil, func(a DeleteInstalledArgs) string { return a.VaultID },
		func(ctx context.Context, a DeleteInstalledArgs) (any, error) {
			if err := s.vault.Delete(a.VaultID); err != nil {
				return nil, err
			}
			return map[string]any{"ok": true, "id": a.VaultID}, nil
		})

	addTool(s, "install_with_secret",
		"装『需要设置密码/密钥』的服务的唯一正确方式：AI 传 secret 生成规范 + 带 {{KEY}} 占位的安装脚本，内部生成强随机真值并替换后执行。AI 全程见不到 secret 真值。",
		func(a InstallWithSecretArgs) string { return a.Server },
		func(a InstallWithSecretArgs) string { return a.InstallScript }, s.handleInstallWithSecret)

	addTool(s, "install_app",
		"从应用商店目录装一个应用（app=openresty 等）。密码内部生成并同步进容器+凭据库，容器名 flashshell-<app>，绑 127.0.0.1。需服务器有 Docker。",
		func(a InstallAppArgs) string { return a.Server },
		func(a InstallAppArgs) string { return a.App }, s.handleInstallApp)

	addTool(s, "deploy_upsert_target",
		"创建/更新一个部署目标（DeployTarget）。只写配置、不执行任何部署。至少含 name + recipe。",
		nil, func(a DeployUpsertTargetArgs) string { return a.Target.Name }, s.handleDeployUpsert)

	addTool(s, "deploy_dry_run",
		"预览一次配方部署的执行计划（只读，不执行任何命令）。",
		nil, func(a DeployHistoryArgs) string { return a.Target }, s.handleDeployDryRun)

	addTool(s, "deploy_run",
		"执行一次配方部署。每条远程命令走正常 AI 策略。建议先 deploy_dry_run。",
		nil, func(a DeployRunArgs) string { return a.Target }, s.handleDeployRun)

	addTool(s, "list_deploy_history",
		"列某部署目标的历史版本。只读元工具。",
		nil, func(a DeployHistoryArgs) string { return a.Target },
		func(ctx context.Context, a DeployHistoryArgs) (any, error) {
			return map[string]any{"history": s.ledger.History(a.Target)}, nil
		})

	addTool(s, "cert_list",
		"列服务器上所有 SSL 证书（域名 / 到期日 / 签发方）。只读。",
		func(a ServerOnly) string { return a.Server }, nil, s.handleCertList)

	addTool(s, "web_status",
		"查 OpenResty（网站服务）是否已装/运行/模式。建站前先探这个。只读。",
		func(a ServerOnly) string { return a.Server }, nil, s.handleWebStatus)

	addTool(s, "web_list_sites",
		"列已管理的网站（域名 / 类型 / 根目录 / 是否启用 / 证书）。只读。",
		func(a ServerOnly) string { return a.Server }, nil,
		func(ctx context.Context, a ServerOnly) (any, error) {
			return map[string]any{"sites": s.ledger.ListSites(a.Server)}, nil
		})

	addTool(s, "web_install_openresty",
		"在服务器装 OpenResty（Docker 容器，conf/证书挂宿主）。需服务器有 Docker。",
		func(a ServerOnly) string { return a.Server }, nil, s.handleWebInstall)

	addTool(s, "web_create_proxy",
		"建反向代理站点：域名 → upstream。需先有 OpenResty。改动型走策略。",
		func(a WebCreateProxyArgs) string { return a.Server },
		func(a WebCreateProxyArgs) string { return a.Domain }, s.handleWebProxy)

	addTool(s, "web_create_static",
		"建静态站点（SPA / 前端）。建好后用 sftp_upload 把 dist 传到站根。",
		func(a WebCreateStaticArgs) string { return a.Server },
		func(a WebCreateStaticArgs) string { return a.Domain }, s.handleWebStatic)

	addTool(s, "web_issue_ssl",
		"给域名签 Let's Encrypt 证书（HTTP-01）+ 自动续期 + HTTP→HTTPS。",
		func(a WebIssueSslArgs) string { return a.Server },
		func(a WebIssueSslArgs) string { return a.Domain }, s.handleWebSSL)
}
