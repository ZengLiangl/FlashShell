package mcp

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AuditEntry 一条 MCP 调用审计（对齐 Reeve 字段）
type AuditEntry struct {
	ID         string `json:"id" yaml:"id"`
	Time       string `json:"time" yaml:"time"`
	Source     string `json:"source" yaml:"source"`
	Caller     string `json:"caller" yaml:"caller"`
	Tool       string `json:"tool" yaml:"tool"`
	Module     string `json:"module" yaml:"module"`
	Server     string `json:"server" yaml:"server"`
	Params     string `json:"params" yaml:"params"`
	Result     string `json:"result" yaml:"result"`
	Decision   string `json:"decision" yaml:"decision"` // auto | approved | denied | blocked | cancelled
	Reason     string `json:"reason" yaml:"reason"`
	Approver   string `json:"approver,omitempty" yaml:"approver,omitempty"`
	DurationMs int64  `json:"durationMs" yaml:"durationMs"`
}

// AuditFilter 审计查询过滤
type AuditFilter struct {
	Tool      string `json:"tool"`
	Module    string `json:"module"`
	Server    string `json:"server"`
	Source    string `json:"source"`
	Caller    string `json:"caller"`
	Decision  string `json:"decision"`
	Keyword   string `json:"keyword"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Limit     int    `json:"limit"`
}

// AuditStats 审计汇总（对齐 Reeve 决策类型）
type AuditStats struct {
	Total     int `json:"total"`
	Today     int `json:"today"`
	Auto      int `json:"auto"`
	Approved  int `json:"approved"`
	Denied    int `json:"denied"`
	Blocked   int `json:"blocked"`
	Cancelled int `json:"cancelled"`
	Pending   int `json:"pending"` // 审批队列实时数
}

// AuditMeta 筛选项元数据
type AuditMeta struct {
	Tools   []string `json:"tools"`
	Modules []string `json:"modules"`
	Servers []string `json:"servers"`
	Sources []string `json:"sources"`
}

type AuditLog struct {
	mu sync.Mutex
}

func newAuditLog() *AuditLog {
	return &AuditLog{}
}

func toolModule(tool string) string {
	switch {
	case tool == "list_servers":
		return "servers"
	case strings.HasPrefix(tool, "ssh_"):
		return "ssh"
	case strings.HasPrefix(tool, "sftp_"):
		return "sftp"
	case strings.HasPrefix(tool, "web_") || tool == "cert_list":
		return "web"
	case strings.HasPrefix(tool, "deploy_") || tool == "list_deploy_history":
		return "deploy"
	case strings.Contains(tool, "skill") || tool == "evaluate_skills" || tool == "recall_experience" ||
		strings.Contains(tool, "runbook"):
		return "skills"
	case strings.HasPrefix(tool, "install_") || strings.Contains(tool, "installed") ||
		tool == "save_credential" || tool == "delete_installed_service":
		return "apps"
	case tool == "system_info" || tool == "disk_usage" || tool == "port_check" ||
		tool == "service_status" || tool == "tail_log":
		return "inspect"
	default:
		return "other"
	}
}

func (a *AuditLog) Append(e AuditEntry) AuditEntry {
	if e.ID == "" {
		e.ID = "aud_" + uuid.NewString()[:12]
	}
	if e.Time == "" {
		e.Time = time.Now().Format("2006-01-02 15:04:05")
	}
	if e.Module == "" {
		e.Module = toolModule(e.Tool)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	root, err := homeDir()
	if err != nil {
		return e
	}
	f, err := os.OpenFile(join(root, auditFile), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return e
	}
	defer f.Close()
	b, _ := json.Marshal(e)
	_, _ = f.Write(append(b, '\n'))
	return e
}

func (a *AuditLog) readAll() []AuditEntry {
	root, err := homeDir()
	if err != nil {
		return nil
	}
	f, err := os.Open(join(root, auditFile))
	if err != nil {
		return nil
	}
	defer f.Close()
	var all []AuditEntry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e AuditEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if e.Module == "" {
			e.Module = toolModule(e.Tool)
		}
		all = append(all, e)
	}
	return all
}

func (a *AuditLog) Query(filter AuditFilter) []AuditEntry {
	a.mu.Lock()
	defer a.mu.Unlock()
	limit := filter.Limit
	if limit <= 0 || limit > 5000 {
		limit = 2000
	}
	raw := a.readAll()
	var all []AuditEntry
	for _, e := range raw {
		if !matchAudit(e, filter) {
			continue
		}
		all = append(all, e)
	}
	if len(all) > limit {
		all = all[len(all)-limit:]
	}
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	return all
}

func (a *AuditLog) Stats() AuditStats {
	items := a.Query(AuditFilter{Limit: 5000})
	today := time.Now().Format("2006-01-02")
	var s AuditStats
	s.Total = len(items)
	for _, e := range items {
		if strings.HasPrefix(e.Time, today) {
			s.Today++
		}
		switch normalizeDecision(e.Decision) {
		case "auto":
			s.Auto++
		case "approved":
			s.Approved++
		case "denied":
			s.Denied++
		case "blocked":
			s.Blocked++
		case "cancelled":
			s.Cancelled++
		}
	}
	return s
}

func normalizeDecision(d string) string {
	switch strings.ToLower(strings.TrimSpace(d)) {
	case "auto", "success", "passed", "completed":
		return "auto"
	case "approved":
		return "approved"
	case "manual", "human":
		return "approved" // 人工放行操作归入已批准侧展示兼容
	case "denied", "rejected", "error", "timeout":
		return "denied"
	case "blocked":
		return "blocked"
	case "cancelled", "pending", "approval":
		return "cancelled"
	default:
		return d
	}
}

func (a *AuditLog) PurgeOlderThan(days int) (removed int, err error) {
	if days <= 0 {
		return 0, nil
	}
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	a.mu.Lock()
	defer a.mu.Unlock()
	all := a.readAll()
	var keep []AuditEntry
	for _, e := range all {
		if e.Time >= cutoff {
			keep = append(keep, e)
		} else {
			removed++
		}
	}
	if removed == 0 {
		return 0, nil
	}
	root, err := homeDir()
	if err != nil {
		return 0, err
	}
	f, err := os.Create(join(root, auditFile))
	if err != nil {
		return 0, err
	}
	defer f.Close()
	for _, e := range keep {
		b, _ := json.Marshal(e)
		if _, err := f.Write(append(b, '\n')); err != nil {
			return removed, err
		}
	}
	return removed, nil
}

func (a *AuditLog) Meta() AuditMeta {
	items := a.Query(AuditFilter{Limit: 5000})
	tools := map[string]struct{}{}
	modules := map[string]struct{}{}
	servers := map[string]struct{}{}
	sources := map[string]struct{}{}
	for _, e := range items {
		if e.Tool != "" {
			tools[e.Tool] = struct{}{}
		}
		mod := e.Module
		if mod == "" {
			mod = toolModule(e.Tool)
		}
		if mod != "" {
			modules[mod] = struct{}{}
		}
		if e.Server != "" {
			servers[e.Server] = struct{}{}
		}
		if e.Source != "" {
			sources[e.Source] = struct{}{}
		}
	}
	return AuditMeta{
		Tools:   sortedKeys(tools),
		Modules: sortedKeys(modules),
		Servers: sortedKeys(servers),
		Sources: sortedKeys(sources),
	}
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (a *AuditLog) ExportJSONL(path string, filter AuditFilter) error {
	items := a.Query(filter)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for i := len(items) - 1; i >= 0; i-- {
		b, _ := json.Marshal(items[i])
		if _, err := f.Write(append(b, '\n')); err != nil {
			return err
		}
	}
	return nil
}

func (a *AuditLog) ExportCSV(path string, filter AuditFilter) error {
	items := a.Query(filter)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"id", "time", "source", "caller", "tool", "module", "server", "decision", "reason", "approver", "durationMs", "params", "result"})
	for i := len(items) - 1; i >= 0; i-- {
		e := items[i]
		_ = w.Write([]string{
			e.ID, e.Time, e.Source, e.Caller, e.Tool, e.Module, e.Server, e.Decision, e.Reason, e.Approver,
			fmt.Sprintf("%d", e.DurationMs), e.Params, e.Result,
		})
	}
	w.Flush()
	return w.Error()
}

func (a *AuditLog) Clear() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	root, err := homeDir()
	if err != nil {
		return err
	}
	return os.WriteFile(join(root, auditFile), nil, 0600)
}

func (a *AuditLog) DeleteIDs(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	want := map[string]struct{}{}
	for _, id := range ids {
		want[id] = struct{}{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	all := a.readAll()
	var keep []AuditEntry
	for _, e := range all {
		if _, drop := want[e.ID]; drop {
			continue
		}
		keep = append(keep, e)
	}
	root, err := homeDir()
	if err != nil {
		return err
	}
	f, err := os.Create(join(root, auditFile))
	if err != nil {
		return err
	}
	defer f.Close()
	for _, e := range keep {
		b, _ := json.Marshal(e)
		if _, err := f.Write(append(b, '\n')); err != nil {
			return err
		}
	}
	return nil
}

func matchAudit(e AuditEntry, f AuditFilter) bool {
	if f.Tool != "" && !strings.EqualFold(e.Tool, f.Tool) {
		return false
	}
	mod := e.Module
	if mod == "" {
		mod = toolModule(e.Tool)
	}
	if f.Module != "" && !strings.EqualFold(mod, f.Module) {
		return false
	}
	if f.Server != "" && !strings.Contains(strings.ToLower(e.Server), strings.ToLower(f.Server)) {
		return false
	}
	if f.Source != "" && !strings.Contains(strings.ToLower(e.Source), strings.ToLower(f.Source)) {
		return false
	}
	if f.Caller != "" && !strings.Contains(strings.ToLower(e.Caller), strings.ToLower(f.Caller)) {
		return false
	}
	if f.Decision != "" {
		want := normalizeDecision(f.Decision)
		got := normalizeDecision(e.Decision)
		if want != got {
			return false
		}
	}
	if f.StartTime != "" && e.Time < f.StartTime {
		return false
	}
	if f.EndTime != "" && e.Time > f.EndTime {
		return false
	}
	if kw := strings.ToLower(strings.TrimSpace(f.Keyword)); kw != "" {
		blob := strings.ToLower(e.Tool + " " + e.Params + " " + e.Result + " " + e.Server + " " + e.Source + " " + e.Reason)
		if !strings.Contains(blob, kw) {
			return false
		}
	}
	return true
}

func clip(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmtSprint(v)
	}
	return string(b)
}

func fmtSprint(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
