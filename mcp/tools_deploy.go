package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *Service) handleDeployUpsert(_ context.Context, a DeployUpsertTargetArgs) (any, error) {
	t := a.Target
	if strings.TrimSpace(t.Name) == "" || strings.TrimSpace(t.Recipe) == "" {
		return nil, wrapErr("[denied]", "target 至少需要 name + recipe")
	}
	dt := DeployTarget{
		Name:               t.Name,
		Recipe:             t.Recipe,
		Servers:            t.Servers,
		Domain:             t.Domain,
		HTTPS:              t.HTTPS,
		Workdir:            t.Workdir,
		BuildCommands:      t.BuildCommands,
		AutoRollback:       t.AutoRollback,
		SkipUnchangedBuild: t.SkipUnchangedBuild,
		Vars:               t.Vars,
		Artifact:           t.Artifact,
		Compose:            t.Compose,
		Health:             t.Health,
		Image:              t.Image,
		Release:            t.Release,
	}
	if t.BuildSource != nil {
		dt.BuildSource = *t.BuildSource
	}
	if err := s.ledger.UpsertTarget(dt); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "name": dt.Name, "recipe": dt.Recipe}, nil
}

func (s *Service) handleDeployDryRun(_ context.Context, a DeployHistoryArgs) (any, error) {
	t, ok := s.ledger.GetTarget(a.Target)
	if !ok {
		return nil, wrapErr("[notfound]", "未找到部署目标 "+a.Target)
	}
	return map[string]any{"target": t, "plan": buildDeployPlan(t)}, nil
}

func buildDeployPlan(t DeployTarget) []string {
	var steps []string
	if len(t.BuildCommands) > 0 {
		steps = append(steps, "本机构建: "+strings.Join(t.BuildCommands, " && "))
	}
	root := "/opt/flashshell/stacks/" + t.Name
	if t.Release != nil && t.Release.DeployRoot != "" {
		root = t.Release.DeployRoot
	}
	servers := t.Servers
	if len(servers) == 0 {
		servers = []string{"(未指定服务器)"}
	}
	steps = append(steps, "远端目录: "+root+" @ "+strings.Join(servers, ","))
	if t.Compose != nil && t.Compose.Template != "" {
		steps = append(steps, "写入 docker-compose.yml 并 docker compose up -d")
		if t.Compose.DB != nil && t.Compose.DB.Mode == "create" {
			steps = append(steps, "在共享 MySQL 上建库 "+t.Compose.DB.DBName+" 并登记独立 mysql_conn")
		}
	}
	if t.Recipe == "static-openresty" {
		steps = append(steps, "sftp_upload 静态产物到站点根，必要时签发证书")
	}
	if t.Domain != "" && t.HTTPS {
		steps = append(steps, "为 "+t.Domain+" 建反代并签发证书")
	}
	return steps
}

func (s *Service) handleDeployRun(ctx context.Context, a DeployRunArgs) (any, error) {
	t, ok := s.ledger.GetTarget(a.Target)
	if !ok {
		return nil, wrapErr("[notfound]", "未找到部署目标 "+a.Target)
	}
	note := ""
	if a.Note != nil {
		note = *a.Note
	}
	version := time.Now().Format("20060102-150405")
	if a.Version != nil && *a.Version != "" {
		version = *a.Version
	}
	root := "/opt/flashshell/stacks/" + t.Name
	if t.Release != nil && t.Release.DeployRoot != "" {
		root = t.Release.DeployRoot
	}
	servers := t.Servers
	var logs []any
	okAll := true
	for _, sv := range servers {
		if _, _, err := s.gate(ctx, "ssh_exec", sv, "deploy "+t.Name, map[string]any{"target": t.Name, "server": sv}); err != nil {
			okAll = false
			logs = append(logs, map[string]any{"server": sv, "error": err.Error()})
			continue
		}
		if _, err := s.execSSH(context.Background(), sv, "mkdir -p "+shellQuote(root), 20*time.Second); err != nil {
			okAll = false
			logs = append(logs, map[string]any{"server": sv, "error": err.Error()})
			continue
		}
		if t.Compose != nil && t.Compose.Template != "" {
			tpl := t.Compose.Template
			if t.Compose.Secrets != nil {
				for _, k := range t.Compose.Secrets {
					val, _ := generateSecret(SecretSpec{Kind: "random_string", Length: 32, Alphabet: "strong"})
					tpl = strings.ReplaceAll(tpl, "__"+k+"__", val)
					tpl = strings.ReplaceAll(tpl, "{{"+k+"}}", val)
				}
			}
			if t.Compose.Redis != nil && t.Compose.Redis.VaultID != "" {
				if p, ok := s.vault.SecretValue(t.Compose.Redis.VaultID); ok {
					tpl = strings.ReplaceAll(tpl, "__REDIS_PASSWORD__", p)
				}
			}
			if t.Compose.DB != nil && t.Compose.DB.VaultID != "" {
				if p, ok := s.vault.SecretValue(t.Compose.DB.VaultID); ok {
					tpl = strings.ReplaceAll(tpl, "__DB_PASSWORD__", p)
				}
				if t.Compose.DB.Mode == "create" && t.Compose.DB.DBName != "" {
					_ = s.ensureAppDB(ctx, sv, t)
				}
			}
			remote := filepath.ToSlash(filepath.Join(root, "docker-compose.yml"))
			if t.Compose.RemotePath != "" {
				remote = t.Compose.RemotePath
			}
			if err := s.writeRemoteText(sv, remote, tpl); err != nil {
				okAll = false
				logs = append(logs, map[string]any{"server": sv, "error": err.Error()})
				continue
			}
			cmd := "cd " + shellQuote(filepath.Dir(remote)) + " && docker compose up -d"
			res, err := s.execSSH(context.Background(), sv, cmd, 10*time.Minute)
			if err != nil {
				okAll = false
				logs = append(logs, map[string]any{"server": sv, "error": err.Error()})
				continue
			}
			logs = append(logs, map[string]any{"server": sv, "result": res})
			if res.ExitCode != 0 {
				okAll = false
			}
		}
		if t.Recipe == "static-openresty" && t.Artifact != nil && t.Artifact.LocalDir != "" {
			_ = s.deployStaticDir(sv, t)
		}
	}
	_ = s.ledger.AddHistory(DeployHistoryItem{
		Target: t.Name, Version: version, Servers: servers, OK: okAll, Running: okAll, Note: note,
		Detail: mustJSON(logs), Time: time.Now(),
	})
	return map[string]any{"ok": okAll, "version": version, "logs": logs}, nil
}

func (s *Service) ensureAppDB(ctx context.Context, server string, t DeployTarget) error {
	db := t.Compose.DB
	it, sec, ok := s.vault.Find(db.VaultID)
	if !ok {
		return wrapErr("[notfound]", "共享 MySQL 凭据不存在")
	}
	user := t.Name
	if db.DBUser != "" {
		user = db.DBUser
	}
	pass, _ := generateSecret(SecretSpec{Kind: "random_string", Length: 20, Alphabet: "strong"})
	sqls := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`; CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s'; GRANT SELECT,INSERT,UPDATE,DELETE,CREATE,INDEX,ALTER ON `%s`.* TO '%s'@'%%'; FLUSH PRIVILEGES;",
		db.DBName, user, pass, db.DBName, user)
	host := pubGet(it, "host")
	port := pubGet(it, "port")
	rootPass := firstNonEmpty(sec["password"], sec["PASSWORD"])
	cmd := fmt.Sprintf("mysql -h %s -P %s -uroot -p%s -e %s", host, nz(port, "3306"), shellQuote(rootPass), shellQuote(sqls))
	if _, err := s.execSSH(context.Background(), server, cmd, 30*time.Second); err != nil {
		return err
	}
	_, err := s.vault.Save(VaultItem{
		ID:          "vs_" + uuid.NewString()[:8],
		ServerAlias: server,
		Kind:        "mysql_conn",
		Label:       db.DBName,
		Public:      map[string]string{"host": "127.0.0.1", "port": nz(pubGet(it, "port"), "3306"), "user": user, "database": db.DBName, "__tunnel_server_id": server},
		Secrets:     map[string]string{"password": pass},
	})
	return err
}

func (s *Service) deployStaticDir(server string, t DeployTarget) error {
	dir := t.Artifact.LocalDir
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	root := "/opt/flashshell/openresty/html/" + sanitizeDomain(nz(t.Domain, t.Name))
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		local := filepath.Join(dir, e.Name())
		remote := root + "/" + e.Name()
		_, _ = s.handleSftpUpload(context.Background(), SftpUploadArgs{Server: server, LocalPath: local, RemotePath: remote})
	}
	return nil
}
