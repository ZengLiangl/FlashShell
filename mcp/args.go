package mcp

type EmptyArgs struct{}

type ServerOnly struct {
	Server string `json:"server" jsonschema:"目标服务器别名。"`
}

type SshExecArgs struct {
	Server      string `json:"server" jsonschema:"目标服务器别名（在 FlashShell 服务器清单里配的那个名字；不是 IP / 不是凭据）。"`
	Command     string `json:"command" jsonschema:"要执行的 shell 命令。"`
	TimeoutSecs *int64 `json:"timeout_secs,omitempty" jsonschema:"可选：超时秒数（默认 30，范围 1～600）。"`
}

type SshExecMultiArgs struct {
	Servers     []string `json:"servers" jsonschema:"目标服务器别名数组（必填，1～50；每个都按各自服务器的策略档独立裁决）。"`
	Command     string   `json:"command" jsonschema:"要并发执行的 shell 命令（每台跑同一条；危险黑名单、sudo 强制审批仍生效）。"`
	TimeoutSecs *int64   `json:"timeout_secs,omitempty" jsonschema:"可选：单台超时秒数（默认 30，范围 1～600）。"`
}

type SshExecScriptArgs struct {
	Server      string  `json:"server" jsonschema:"目标服务器别名。"`
	Script      string  `json:"script" jsonschema:"完整脚本内容（多行 OK）。会经 base64 包装在远端解码后执行，策略引擎按原始脚本明文判定。"`
	Interpreter *string `json:"interpreter,omitempty" jsonschema:"可选解释器：bash(默认) / sh / python3 / python。"`
	TimeoutSecs *int64  `json:"timeout_secs,omitempty" jsonschema:"可选超时秒数（默认 60，范围 1～600）。"`
}

type DiskUsageArgs struct {
	Server string  `json:"server" jsonschema:"目标服务器别名。"`
	Path   *string `json:"path,omitempty" jsonschema:"可选：要查的路径（如 /var）。留空则列所有文件系统。"`
}

type PortCheckArgs struct {
	Server string `json:"server" jsonschema:"目标服务器别名。"`
	Port   int    `json:"port" jsonschema:"端口号 1～65535。"`
}

type ServiceStatusArgs struct {
	Server  string `json:"server" jsonschema:"目标服务器别名。"`
	Service string `json:"service" jsonschema:"systemd unit 名（如 nginx / sshd.service / docker）。"`
}

type TailLogArgs struct {
	Server string `json:"server" jsonschema:"目标服务器别名。"`
	Path   string `json:"path" jsonschema:"文件绝对路径（如 /var/log/syslog）。"`
	Lines  *int64 `json:"lines,omitempty" jsonschema:"可选：要看的最末行数（默认 200，范围 1～5000）。"`
}

type SftpListArgs struct {
	Server string  `json:"server" jsonschema:"目标服务器别名。"`
	Path   *string `json:"path,omitempty" jsonschema:"远端目录路径；留空则用家目录。"`
}

type SftpReadArgs struct {
	Server   string `json:"server" jsonschema:"目标服务器别名。"`
	Path     string `json:"path" jsonschema:"远端文件绝对路径。"`
	MaxBytes *int64 `json:"max_bytes,omitempty" jsonschema:"可选：最大读取字节数，默认 262144（256 KiB），上限 4 MiB。"`
}

type SftpWriteArgs struct {
	Server        string  `json:"server" jsonschema:"目标服务器别名。"`
	Path          string  `json:"path" jsonschema:"远端文件绝对路径。"`
	Content       *string `json:"content,omitempty" jsonschema:"文本内容（utf-8）。与 content_base64 二选一。"`
	ContentBase64 *string `json:"content_base64,omitempty" jsonschema:"二进制内容（base64）。与 content 二选一。"`
}

type SftpUploadArgs struct {
	Server     string `json:"server" jsonschema:"目标服务器别名。"`
	LocalPath  string `json:"local_path" jsonschema:"本机（运行 FlashShell 主应用那台机器）上的文件绝对路径。"`
	RemotePath string `json:"remote_path" jsonschema:"远端目标绝对路径。"`
}

type EvaluateSkillsArgs struct {
	Prompt string `json:"prompt" jsonschema:"用户问题/任务描述。"`
}

type NameOnly struct {
	Name string `json:"name" jsonschema:"名字（不带 .md 后缀；仅字母数字 . _ -）"`
}

type RecallExperienceArgs struct {
	Query *string `json:"query,omitempty" jsonschema:"检索关键字（substring，不区分大小写）。留空则列每条 experience 的首行摘要。"`
}

type ListInstalledArgs struct {
	Server *string `json:"server,omitempty" jsonschema:"可选：只列某台服务器（按别名）。空则列所有可见服务器的已装服务。"`
}

type DeleteInstalledArgs struct {
	VaultID string `json:"vaultId" jsonschema:"要删的服务凭据 id（list_installed_services 返回的 id 字段）。"`
}

type SaveCredentialArgs struct {
	Server          string            `json:"server"`
	Kind            string            `json:"kind"`
	Label           string            `json:"label"`
	Notes           *string           `json:"notes,omitempty"`
	Fields          map[string]string `json:"fields,omitempty"`
	FieldsFromVault map[string]string `json:"fieldsFromVault,omitempty"`
	SecretFields    []string          `json:"secretFields,omitempty"`
}

type SecretSpec struct {
	Kind     string `json:"kind"`
	Length   int    `json:"length,omitempty"`
	Alphabet string `json:"alphabet,omitempty"`
	Min      int    `json:"min,omitempty"`
	Max      int    `json:"max,omitempty"`
}

type InstallWithSecretArgs struct {
	Server            string                `json:"server" jsonschema:"目标服务器别名。"`
	Label             string                `json:"label"`
	Kind              string                `json:"kind"`
	Secrets           map[string]SecretSpec `json:"secrets"`
	InstallScript     string                `json:"installScript"`
	SaveFields        []string              `json:"saveFields"`
	Public            map[string]string     `json:"public,omitempty"`
	Notes             *string               `json:"notes,omitempty"`
	InstallPath       *string               `json:"installPath,omitempty"`
	AccessUrlTemplate *string               `json:"accessUrlTemplate,omitempty"`
	TimeoutSecs       *int64                `json:"timeoutSecs,omitempty"`
	VerifyScript      *string               `json:"verifyScript,omitempty"`
}

type InstallAppArgs struct {
	Server  string  `json:"server" jsonschema:"目标服务器别名。"`
	App     string  `json:"app" jsonschema:"应用商店目录里的应用 id：mysql / redis / postgres / mongodb / openresty 等。"`
	Port    *int    `json:"port,omitempty"`
	Version *string `json:"version,omitempty"`
}

type DeployHistoryArgs struct {
	Target string `json:"target" jsonschema:"部署目标名（deploy_targets.name）"`
}

type DeployRunArgs struct {
	Target  string  `json:"target"`
	Note    *string `json:"note,omitempty"`
	Version *string `json:"version,omitempty"`
}

type DtArtifactArg struct {
	LocalDir string   `json:"localDir,omitempty"`
	Exclude  []string `json:"exclude,omitempty"`
}

type DtDbArg struct {
	VaultID string `json:"vaultId,omitempty"`
	Mode    string `json:"mode,omitempty"`
	DBName  string `json:"dbName,omitempty"`
	DBUser  string `json:"dbUser,omitempty"`
}

type DtRedisArg struct {
	VaultID string `json:"vaultId,omitempty"`
}

type DtComposeArg struct {
	Template   string      `json:"template,omitempty"`
	RemotePath string      `json:"remotePath,omitempty"`
	Overwrite  bool        `json:"overwrite,omitempty"`
	Secrets    []string    `json:"secrets,omitempty"`
	InitSqls   []string    `json:"initSqls,omitempty"`
	DB         *DtDbArg    `json:"db,omitempty"`
	Redis      *DtRedisArg `json:"redis,omitempty"`
}

type DtHealthArg struct {
	Cmd         string `json:"cmd,omitempty"`
	IntervalSec *int   `json:"intervalSec,omitempty"`
	Retries     *int   `json:"retries,omitempty"`
}

type DtImageArg struct {
	Name            string  `json:"name,omitempty"`
	Tag             string  `json:"tag,omitempty"`
	AlsoLatest      bool    `json:"alsoLatest,omitempty"`
	VersionStrategy *string `json:"versionStrategy,omitempty"`
}

type DtReleaseArg struct {
	DeployRoot   string   `json:"deployRoot,omitempty"`
	Strategy     string   `json:"strategy,omitempty"`
	Shared       []string `json:"shared,omitempty"`
	KeepReleases *int     `json:"keepReleases,omitempty"`
}

type DeployTargetArg struct {
	Name               string            `json:"name"`
	Recipe             string            `json:"recipe"`
	Servers            []string          `json:"servers,omitempty"`
	Domain             string            `json:"domain,omitempty"`
	HTTPS              bool              `json:"https,omitempty"`
	Workdir            string            `json:"workdir,omitempty"`
	BuildSource        *string           `json:"buildSource,omitempty"`
	BuildCommands      []string          `json:"buildCommands,omitempty"`
	AutoRollback       bool              `json:"autoRollback,omitempty"`
	SkipUnchangedBuild *bool             `json:"skipUnchangedBuild,omitempty"`
	Vars               map[string]string `json:"vars,omitempty"`
	Artifact           *DtArtifactArg    `json:"artifact,omitempty"`
	Compose            *DtComposeArg     `json:"compose,omitempty"`
	Health             *DtHealthArg      `json:"health,omitempty"`
	Image              *DtImageArg       `json:"image,omitempty"`
	Release            *DtReleaseArg     `json:"release,omitempty"`
}

type DeployUpsertTargetArgs struct {
	Target DeployTargetArg `json:"target"`
}

type WebCreateProxyArgs struct {
	Server   string `json:"server"`
	Domain   string `json:"domain"`
	Upstream string `json:"upstream"`
}

type WebCreateStaticArgs struct {
	Server string `json:"server"`
	Domain string `json:"domain"`
}

type WebIssueSslArgs struct {
	Server    string `json:"server"`
	Domain    string `json:"domain"`
	Email     string `json:"email"`
	AutoRenew *bool  `json:"auto_renew,omitempty"`
}
