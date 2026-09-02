package mcp

import (
	"fmt"
	"strings"
)

func (s *Service) ApprovalContext(server string, limit int) []AuditEntry {
	if limit <= 0 {
		limit = 8
	}
	if limit > 50 {
		limit = 50
	}
	f := AuditFilter{Server: server, Limit: limit}
	rows := s.audit.Query(f)
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows
}

func (s *Service) DecideApproval(id string, allow bool, addOutboundHosts bool, rejectReason string) error {
	if allow && addOutboundHosts {
		peek, err := s.approvals.Peek(id)
		if err != nil {
			return err
		}
		if len(peek.OutboundHosts) > 0 {
			for _, h := range peek.OutboundHosts {
				if err := s.AddOutboundHost(h); err != nil {
					return fmt.Errorf("加入出站白名单失败，未放行: %w", err)
				}
			}
		}
	}
	item, err := s.approvals.Decide(id, allow, rejectReason)
	if err != nil {
		return err
	}
	approver := localApprover()
	decision := "approved"
	result := "人工批准"
	reason := item.Reason
	if reason == "" {
		reason = "审批队列人工放行"
	} else {
		reason = reason + " → 人工放行"
	}
	if !allow {
		decision = "denied"
		result = "人工拒绝"
		rejectReason = strings.TrimSpace(rejectReason)
		if rejectReason != "" {
			if item.Reason != "" {
				reason = item.Reason + " → 人工拒绝: " + rejectReason
			} else {
				reason = "审批队列人工拒绝: " + rejectReason
			}
			result = "人工拒绝: " + rejectReason
		} else if item.Reason != "" {
			reason = item.Reason + " → 人工拒绝"
		} else {
			reason = "审批队列人工拒绝"
		}
	} else if addOutboundHosts && len(item.OutboundHosts) > 0 {
		result = "人工批准并加入出站白名单"
		reason = reason + "；已加白: " + strings.Join(item.OutboundHosts, ", ")
	}
	s.audit.Append(AuditEntry{
		Source:   "FlashShell UI",
		Caller:   "human",
		Approver: approver,
		Tool:     item.Tool,
		Module:   toolModule(item.Tool),
		Server:   item.Server,
		Params:   clip(item.ParamsJSON, 4000),
		Result:   result,
		Decision: decision,
		Reason:   reason,
	})
	return nil
}

func (s *Service) DecideApprovalBatch(ids []string, allow bool, rejectReason string) error {
	approver := localApprover()
	items, err := s.approvals.DecideBatch(ids, allow, rejectReason)
	if err != nil {
		return err
	}
	decision := "approved"
	result := "批量人工批准"
	reasonBase := "审批队列批量放行"
	if !allow {
		decision = "denied"
		result = "批量人工拒绝"
		reasonBase = "审批队列批量拒绝"
		rejectReason = strings.TrimSpace(rejectReason)
		if rejectReason != "" {
			reasonBase = reasonBase + ": " + rejectReason
			result = result + ": " + rejectReason
		}
	}
	for _, item := range items {
		reason := reasonBase
		if item.Reason != "" {
			reason = item.Reason + " → " + reasonBase
		}
		s.audit.Append(AuditEntry{
			Source:   "FlashShell UI",
			Caller:   "human",
			Approver: approver,
			Tool:     item.Tool,
			Module:   toolModule(item.Tool),
			Server:   item.Server,
			Params:   clip(item.ParamsJSON, 4000),
			Result:   result,
			Decision: decision,
			Reason:   reason,
		})
	}
	return nil
}
