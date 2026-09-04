package mcp

import (
	"context"
	"fmt"
	"strings"
)

// WriteFromVaultArgs 把服务凭据明文写到远端文件（AI 只传 vaultId，不见明文）。
type WriteFromVaultArgs struct {
	Server  string  `json:"server" jsonschema:"目标服务器别名。"`
	Path    string  `json:"path" jsonschema:"远端文件绝对路径。"`
	VaultID string  `json:"vaultId" jsonschema:"服务凭据 id（list_installed_services 返回的 id）。"`
	Field   *string `json:"field,omitempty" jsonschema:"可选：敏感字段名（如 token / password）。空则按 value→token→password→首个非空。"`
	// Template 可选：写入内容模板。支持 {{value}}（本条明文）以及 {{vault:id}} / {{vault:id.field}}。
	// 留空则整文件内容即为该字段明文（末尾补换行）。
	Template *string `json:"template,omitempty" jsonschema:"可选写入模板；空则只写字段明文。"`
	AppendNL *bool   `json:"appendNewline,omitempty" jsonschema:"可选：无模板时是否在末尾追加换行，默认 true。"`
}

func (s *Service) handleWriteFromVault(ctx context.Context, a WriteFromVaultArgs) (any, error) {
	if pathBlocked(a.Path) {
		return nil, wrapErr("[blocked]", "敏感路径禁止写入: "+a.Path)
	}
	if _, err := s.machineByAlias(a.Server); err != nil {
		return nil, err
	}
	vid := strings.TrimSpace(a.VaultID)
	if vid == "" {
		return nil, wrapErr("[denied]", "vaultId 不能为空")
	}
	field := ""
	if a.Field != nil {
		field = strings.TrimSpace(*a.Field)
	}
	plain, err := s.vault.RevealField(vid, field)
	if err != nil {
		return nil, err
	}
	used := []string{plain}
	content := plain
	if a.Template != nil && strings.TrimSpace(*a.Template) != "" {
		tpl := strings.ReplaceAll(*a.Template, "{{value}}", plain)
		resolved, more, err := s.SubstituteVaultPlaceholders(tpl)
		if err != nil {
			return nil, err
		}
		content = resolved
		used = append(used, more...)
	} else {
		appendNL := true
		if a.AppendNL != nil {
			appendNL = *a.AppendNL
		}
		if appendNL && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
	}
	if int64(len(content)) > 16*1024*1024 {
		return nil, wrapErr("[denied]", "单次写入超过 16 MiB")
	}
	c := content
	_, err = s.handleSftpWrite(ctx, SftpWriteArgs{
		Server:  a.Server,
		Path:    a.Path,
		Content: &c,
	})
	if err != nil {
		return nil, err
	}
	_ = used // 明文不进返回；审计由 record 记 args（仅 vaultId）
	return map[string]any{
		"ok":      true,
		"path":    a.Path,
		"server":  a.Server,
		"vaultId": vid,
		"bytes":   len(content),
	}, nil
}

// SaveInstalledOpts UI/绑定：手动登记服务凭据
type SaveInstalledOpts struct {
	Server string            `json:"server"`
	Kind   string            `json:"kind"`
	Label  string            `json:"label"`
	Field  string            `json:"field"` // 敏感字段名，默认 password
	Value  string            `json:"value"` // 明文（仅本机 UI，需解锁）
	Notes  string            `json:"notes,omitempty"`
	Public map[string]string `json:"public,omitempty"`
}

func (s *Service) SaveInstalledManual(opts SaveInstalledOpts) (map[string]any, error) {
	if s.vault == nil {
		return nil, fmt.Errorf("凭据库未初始化")
	}
	server := strings.TrimSpace(opts.Server)
	// 空服务器 = 共用凭据，不绑定某台机器
	if server != "" {
		if _, err := s.machineByAlias(server); err != nil {
			return nil, err
		}
	}
	kind := strings.TrimSpace(opts.Kind)
	if kind == "" {
		kind = "credential"
	}
	label := strings.TrimSpace(opts.Label)
	if label == "" {
		label = kind
	}
	field := strings.TrimSpace(opts.Field)
	if field == "" {
		if strings.Contains(strings.ToLower(kind), "token") {
			field = "token"
		} else {
			field = "password"
		}
	}
	val := opts.Value
	if strings.TrimSpace(val) == "" {
		return nil, fmt.Errorf("敏感值不能为空")
	}
	pub := map[string]string{}
	for k, v := range opts.Public {
		pub[k] = v
	}
	saved, err := s.vault.Save(VaultItem{
		ServerAlias: server,
		Kind:        kind,
		Label:       label,
		Notes:       opts.Notes,
		Public:      pub,
		Secrets:     map[string]string{field: val},
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":          saved.ID,
		"label":       saved.Label,
		"kind":        saved.Kind,
		"serverAlias": saved.ServerAlias,
	}, nil
}

// WriteInstalledToRemote UI：把某条服务凭据写到远端（不经 AI）
func (s *Service) WriteInstalledToRemote(server, path, vaultID, field string) (map[string]any, error) {
	var f *string
	if strings.TrimSpace(field) != "" {
		x := strings.TrimSpace(field)
		f = &x
	}
	out, err := s.handleWriteFromVault(context.Background(), WriteFromVaultArgs{
		Server:  server,
		Path:    path,
		VaultID: vaultID,
		Field:   f,
	})
	if err != nil {
		return nil, err
	}
	if m, ok := out.(map[string]any); ok {
		return m, nil
	}
	return map[string]any{"ok": true, "path": path}, nil
}
