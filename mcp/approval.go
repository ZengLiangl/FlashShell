package mcp

import (
	"context"
	"encoding/json"
	"os/user"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

const approvalTimeout = 5 * time.Minute

// ApprovalItem 待审批 MCP 操作（对齐 Reeve 审批队列）
type ApprovalItem struct {
	ID            string   `json:"id"`
	Tool          string   `json:"tool"`
	Server        string   `json:"server"`
	Preview       string   `json:"preview"`
	Summary       string   `json:"summary"`
	ParamsJSON    string   `json:"paramsJson,omitempty"`
	Source        string   `json:"source"`
	Reason        string   `json:"reason,omitempty"`
	OutboundHosts []string `json:"outboundHosts,omitempty"`
	CreatedAt     string   `json:"createdAt"`
	ExpiresAt     string   `json:"expiresAt"`
	RemainingSecs int      `json:"remainingSecs"`
	IsDanger      bool     `json:"isDanger"`
}

type pendingApproval struct {
	item      ApprovalItem
	expiresAt time.Time
	ch        chan bool
}

type ApprovalHub struct {
	mu        sync.Mutex
	pending   map[string]*pendingApproval
	emit      func(event string, data any)
	onTimeout func(ApprovalItem)
}

func newApprovalHub() *ApprovalHub {
	return &ApprovalHub{pending: map[string]*pendingApproval{}}
}

func (h *ApprovalHub) SetEmitter(fn func(event string, data any)) {
	h.mu.Lock()
	h.emit = fn
	h.mu.Unlock()
}

func (h *ApprovalHub) SetTimeoutHandler(fn func(ApprovalItem)) {
	h.mu.Lock()
	h.onTimeout = fn
	h.mu.Unlock()
}

func (h *ApprovalHub) emitQueued(item ApprovalItem) {
	emit := h.emit
	if emit == nil {
		return
	}
	// 走 JSON 再还原成 map，确保 Wails EventsEmit 前端一定能拿到字段
	payload := approvalPayload(item)
	emit("approval:queued", payload)
	emit("mcp:approval", payload) // 兼容旧前端
}

func approvalPayload(item ApprovalItem) map[string]any {
	b, err := json.Marshal(item)
	if err != nil {
		return map[string]any{
			"id":      item.ID,
			"tool":    item.Tool,
			"server":  item.Server,
			"preview": item.Preview,
			"summary": item.Summary,
			"source":  item.Source,
			"reason":  item.Reason,
		}
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]any{"id": item.ID, "tool": item.Tool, "server": item.Server}
	}
	return m
}

func (h *ApprovalHub) emitResolved(id, decision string) {
	emit := h.emit
	if emit == nil {
		return
	}
	emit("approval:resolved", map[string]any{"id": id, "decision": decision})
}

func enrichItem(item ApprovalItem, expiresAt time.Time) ApprovalItem {
	item.ExpiresAt = expiresAt.Format(time.RFC3339)
	sec := int(time.Until(expiresAt).Seconds())
	if sec < 0 {
		sec = 0
	}
	item.RemainingSecs = sec
	if item.Summary == "" {
		item.Summary = clip(item.Preview, 200)
	}
	return item
}

func (h *ApprovalHub) Request(ctx context.Context, tool, server, preview, paramsJSON, source, reason string, outbound []string, isDanger bool) (bool, string, error) {
	id := "ap_" + uuid.NewString()[:10]
	now := time.Now()
	exp := now.Add(approvalTimeout)
	item := ApprovalItem{
		ID:            id,
		Tool:          tool,
		Server:        server,
		Preview:       clip(preview, 4000),
		Summary:       clip(preview, 200),
		ParamsJSON:    clip(paramsJSON, 8000),
		Source:        source,
		Reason:        clip(reason, 500),
		OutboundHosts: append([]string{}, outbound...),
		CreatedAt:     now.Format("2006-01-02 15:04:05"),
		IsDanger:      isDanger,
	}
	item = enrichItem(item, exp)
	p := &pendingApproval{item: item, expiresAt: exp, ch: make(chan bool, 1)}
	h.mu.Lock()
	h.pending[id] = p
	h.mu.Unlock()
	h.emitQueued(item)

	timer := time.NewTimer(approvalTimeout)
	defer timer.Stop()
	select {
	case ok := <-p.ch:
		return ok, id, nil
	case <-timer.C:
		h.mu.Lock()
		delete(h.pending, id)
		onTimeout := h.onTimeout
		h.mu.Unlock()
		if onTimeout != nil {
			onTimeout(item)
		}
		h.emitResolved(id, "denied_by_timeout")
		return false, id, wrapErr("[approval]", "审批超时（5 分钟），已自动拒绝")
	case <-ctx.Done():
		h.mu.Lock()
		delete(h.pending, id)
		h.mu.Unlock()
		h.emitResolved(id, "cancelled")
		return false, id, wrapErr("[approval]", "审批已取消")
	}
}

func (h *ApprovalHub) Peek(id string) (ApprovalItem, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	p, ok := h.pending[id]
	if !ok {
		return ApprovalItem{}, wrapErr("[notfound]", "没有这条待审批记录")
	}
	return enrichItem(p.item, p.expiresAt), nil
}

func (h *ApprovalHub) Decide(id string, allow bool) (ApprovalItem, error) {
	h.mu.Lock()
	p, ok := h.pending[id]
	if ok {
		delete(h.pending, id)
	}
	h.mu.Unlock()
	if !ok {
		return ApprovalItem{}, wrapErr("[notfound]", "没有这条待审批记录")
	}
	select {
	case p.ch <- allow:
	default:
	}
	decision := "approved"
	if !allow {
		decision = "denied"
	}
	h.emitResolved(id, decision)
	return p.item, nil
}

func (h *ApprovalHub) DecideBatch(ids []string, allow bool) ([]ApprovalItem, error) {
	var items []ApprovalItem
	for _, id := range ids {
		item, err := h.Decide(id, allow)
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (h *ApprovalHub) List() []ApprovalItem {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]ApprovalItem, 0, len(h.pending))
	for _, p := range h.pending {
		out = append(out, enrichItem(p.item, p.expiresAt))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out
}

func (h *ApprovalHub) Count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.pending)
}

func localApprover() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return "local-user"
}
