package data

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"FlashDock/define"

	"github.com/google/uuid"
)

var csvImportColumns = []string{"name", "host", "port", "user", "password", "group", "tags"}

// ImportMachinesCSV 从 CSV 导入机器（首行可为表头）
func (gcm *GlobalConfigManager) ImportMachinesCSV(path string) (*MachineImportResult, error) {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return nil, err
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	colIndex := mapCSVHeader(header)

	result := &MachineImportResult{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.Skipped++
			continue
		}
		if len(record) == 0 {
			continue
		}
		get := func(key string) string {
			idx, ok := colIndex[key]
			if !ok || idx < 0 || idx >= len(record) {
				return ""
			}
			return strings.TrimSpace(record[idx])
		}
		name := get("name")
		host := get("host")
		if name == "" || host == "" {
			result.Skipped++
			continue
		}
		port := 22
		if p := get("port"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				port = v
			}
		}
		user := get("user")
		password := get("password")
		group := get("group")
		tags := parseCSVTags(get("tags"))

		existing := gcm.findMachineByName(name)
		machine := existing
		if machine == nil {
			machine = &define.Machine{
				ID:    uuid.NewString(),
				Name:  name,
				Group: group,
				Tags:  tags,
			}
		} else {
			machine.EnsureID()
			if group != "" {
				machine.Group = group
			}
			if len(tags) > 0 {
				machine.Tags = tags
			}
		}
		sensitive := &define.SensitiveData{
			Host:     host,
			Port:     port,
			User:     user,
			Password: password,
		}
		if err := machine.SetSensitiveData(sensitive); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: 加密失败: %v", name, err))
			result.Skipped++
			continue
		}
		gcm.EnsureMachineGroupRegistered(machine.Group)
		if err := gcm.upsertMachine(machine); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", name, err))
			result.Skipped++
			continue
		}
		result.Imported++
	}
	return result, nil
}

func mapCSVHeader(header []string) map[string]int {
	out := make(map[string]int, len(csvImportColumns))
	normalized := make([]string, len(header))
	for i, h := range header {
		normalized[i] = strings.ToLower(strings.TrimSpace(h))
	}
	hasHeader := false
	for _, col := range csvImportColumns {
		for i, h := range normalized {
			if h == col {
				out[col] = i
				hasHeader = true
			}
		}
	}
	if hasHeader {
		return out
	}
	for i, col := range csvImportColumns {
		if i < len(header) {
			out[col] = i
		}
	}
	return out
}

func parseCSVTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ";")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		key := strings.ToLower(p)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return out
}
