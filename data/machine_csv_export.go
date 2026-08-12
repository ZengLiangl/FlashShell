package data

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ExportMachinesCSV 将机器列表导出为 CSV 文件
func (gcm *GlobalConfigManager) ExportMachinesCSV(path string) error {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("导出路径无效")
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"name", "host", "port", "user", "group", "tags", "notes", "proxyJump"})
	for _, m := range gcm.config.Machines {
		tags := strings.Join(m.Tags, ";")
		_ = w.Write([]string{
			m.Name,
			m.Host,
			strconv.Itoa(m.Port),
			m.User,
			m.Group,
			tags,
			strings.ReplaceAll(m.Notes, "\n", " "),
			m.ProxyJump,
		})
	}
	w.Flush()
	return w.Error()
}
