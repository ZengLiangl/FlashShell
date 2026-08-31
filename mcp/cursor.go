package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const cursorServerKey = "flashshell"

func cursorMCPPath() (string, error) {
	return homeJoin(".cursor", "mcp.json")
}

func readJSONFile(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var m map[string]any
	if len(strings.TrimSpace(string(b))) == 0 {
		return map[string]any{}, nil
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败 (%s): %w", filepath.Base(path), err)
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

func writeJSONFile(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}

func mcpServersMap(root map[string]any) map[string]any {
	raw, ok := root["mcpServers"]
	if !ok {
		m := map[string]any{}
		root["mcpServers"] = m
		return m
	}
	if m, ok := raw.(map[string]any); ok {
		return m
	}
	m := map[string]any{}
	root["mcpServers"] = m
	return m
}

func (s *Service) installCursor(tok Token) error {
	path, err := cursorMCPPath()
	if err != nil {
		return err
	}
	root, err := readJSONFile(path)
	if err != nil {
		return err
	}
	servers := mcpServersMap(root)
	servers[cursorServerKey] = s.httpEntry(tok)
	return writeJSONFile(path, root)
}

func (s *Service) InstallCursor() error {
	_, err := s.InstallClientWith("cursor", InstallOpts{})
	return err
}

func (s *Service) RefreshCursor() error {
	_, err := s.RefreshClient("cursor")
	return err
}

func cursorLinked() (string, bool) {
	path, err := cursorMCPPath()
	if err != nil {
		return "", false
	}
	root, err := readJSONFile(path)
	if err != nil {
		return path, false
	}
	servers := mcpServersMap(root)
	_, ok := servers[cursorServerKey]
	return path, ok
}

func (s *Service) UninstallCursor() error {
	path, err := cursorMCPPath()
	if err != nil {
		return err
	}
	root, err := readJSONFile(path)
	if err != nil {
		return err
	}
	servers := mcpServersMap(root)
	delete(servers, cursorServerKey)
	_ = s.tokens.RevokeByClient("cursor")
	return writeJSONFile(path, root)
}
