package mcp

import (
	"encoding/json"
	"strings"
)

var auditSecretKeys = map[string]struct{}{
	"content": {}, "content_base64": {}, "contentbase64": {},
	"fields": {}, "value": {}, "password": {}, "token": {},
	"secrets": {}, "secret": {}, "installscript": {},
	"template": {}, "script": {},
}

func sanitizeAuditParams(params any) string {
	if params == nil {
		return "{}"
	}
	b, err := json.Marshal(params)
	if err != nil {
		return "{}"
	}
	var v any
	if json.Unmarshal(b, &v) != nil {
		return string(b)
	}
	redactJSONValue(&v, "")
	out, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(out)
}

func redactJSONValue(v *any, key string) {
	if v == nil || *v == nil {
		return
	}
	lk := strings.ToLower(key)
	if _, secret := auditSecretKeys[lk]; secret || strings.Contains(lk, "password") || strings.Contains(lk, "secret") || strings.Contains(lk, "token") {
		switch (*v).(type) {
		case string:
			*v = "[REDACTED]"
			return
		case map[string]any:
			*v = map[string]any{"_redacted": true}
			return
		}
	}
	switch t := (*v).(type) {
	case map[string]any:
		for k, child := range t {
			c := child
			redactJSONValue(&c, k)
			t[k] = c
		}
	case []any:
		for i := range t {
			c := t[i]
			redactJSONValue(&c, key)
			t[i] = c
		}
	}
}
