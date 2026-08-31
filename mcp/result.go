package mcp

import (
	"encoding/json"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func textOK(s string) (*mcpsdk.CallToolResult, any, error) {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: s}},
	}, nil, nil
}

func jsonOK(v any) (*mcpsdk.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	// 与 Reeve 一致：业务 JSON 只放 text content，不设 structuredContent。
	// Cursor 客户端要求 structuredContent 必须是 object，数组会被拒。
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(b)}},
	}, nil, nil
}

func toolErr(err error) (*mcpsdk.CallToolResult, any, error) {
	if err == nil {
		return textOK("ok")
	}
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: err.Error()}},
		IsError: true,
	}, nil, nil
}

func wrapErr(prefix, msg string) error {
	if prefix == "" {
		return fmt.Errorf("%s", msg)
	}
	return fmt.Errorf("%s %s", prefix, msg)
}
