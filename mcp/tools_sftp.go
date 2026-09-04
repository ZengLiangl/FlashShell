package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"FlashDock/define"

	"github.com/pkg/sftp"
)

func (s *Service) handleSftpList(_ context.Context, a SftpListArgs) (any, error) {
	path := ""
	if a.Path != nil {
		path = strings.TrimSpace(*a.Path)
	}
	var entries []map[string]any
	err := s.sftpClient(a.Server, func(cli *sftp.Client, _ *define.Machine) error {
		if path == "" {
			wd, err := cli.Getwd()
			if err != nil {
				wd = "."
			}
			path = wd
		}
		list, err := cli.ReadDir(path)
		if err != nil {
			return err
		}
		for _, fi := range list {
			perm := uint32(fi.Mode())
			var uid any
			if st, ok := fi.Sys().(*sftp.FileStat); ok && st != nil {
				perm = st.Mode
				uid = st.UID
			}
			entries = append(entries, map[string]any{
				"name":        fi.Name(),
				"isDir":       fi.IsDir(),
				"size":        fi.Size(),
				"mtime":       fi.ModTime().Unix(),
				"isSymlink":   fi.Mode()&os.ModeSymlink != 0,
				"permissions": perm,
				"owner":       nil,
				"uid":         uid,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []map[string]any{}
	}
	return map[string]any{"entries": entries}, nil
}

func (s *Service) handleSftpRead(ctx context.Context, a SftpReadArgs) (any, error) {
	if pathBlocked(a.Path) {
		return nil, wrapErr("[blocked]", "敏感路径禁止读取: "+a.Path)
	}
	max := int64(262144)
	if a.MaxBytes != nil && *a.MaxBytes > 0 {
		max = *a.MaxBytes
	}
	if max > 4*1024*1024 {
		max = 4 * 1024 * 1024
	}
	var content string
	var enc string
	err := s.sftpClient(a.Server, func(cli *sftp.Client, _ *define.Machine) error {
		b, err := readSFTPFile(cli, a.Path, max)
		if err != nil {
			return err
		}
		if isMostlyUTF8(b) {
			content = s.redactText(ctx, string(b), a.Server)
			enc = "utf-8"
		} else {
			content = base64.StdEncoding.EncodeToString(b)
			enc = "base64"
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"path": a.Path, "encoding": enc, "content": content}, nil
}

func (s *Service) handleSftpWrite(_ context.Context, a SftpWriteArgs) (any, error) {
	if pathBlocked(a.Path) {
		return nil, wrapErr("[blocked]", "敏感路径禁止写入: "+a.Path)
	}
	var data []byte
	switch {
	case a.Content != nil && *a.Content != "":
		content := *a.Content
		resolved, _, err := s.SubstituteVaultPlaceholders(content)
		if err != nil {
			return nil, err
		}
		data = []byte(resolved)
	case a.ContentBase64 != nil && *a.ContentBase64 != "":
		b, err := base64.StdEncoding.DecodeString(*a.ContentBase64)
		if err != nil {
			return nil, fmt.Errorf("content_base64 无效: %w", err)
		}
		data = b
	default:
		return nil, wrapErr("[denied]", "content / content_base64 必须提供其一")
	}
	if int64(len(data)) > 16*1024*1024 {
		return nil, wrapErr("[denied]", "单次写入超过 16 MiB，请改用 sftp_upload")
	}
	err := s.sftpClient(a.Server, func(cli *sftp.Client, _ *define.Machine) error {
		p := strings.ReplaceAll(a.Path, "\\", "/")
		if i := strings.LastIndex(p, "/"); i > 0 {
			_ = cli.MkdirAll(p[:i])
		}
		f, err := cli.Create(p)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(data)
		return err
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "path": a.Path, "bytes": len(data)}, nil
}

func (s *Service) handleSftpUpload(_ context.Context, a SftpUploadArgs) (any, error) {
	if pathBlocked(a.RemotePath) {
		return nil, wrapErr("[blocked]", "敏感路径禁止写入: "+a.RemotePath)
	}
	st, err := os.Stat(a.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("本地文件不存在: %w", err)
	}
	if st.IsDir() {
		return nil, wrapErr("[denied]", "sftp_upload 仅支持单个文件，目录请先打包")
	}
	err = s.sftpClient(a.Server, func(cli *sftp.Client, _ *define.Machine) error {
		return copyLocalToSFTP(cli, a.LocalPath, a.RemotePath)
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "local_path": a.LocalPath, "remote_path": a.RemotePath, "bytes": st.Size()}, nil
}
