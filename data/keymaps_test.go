package data

import (
	"encoding/json"
	"testing"
)

func TestNormalizeKeyMapSettings(t *testing.T) {
	raw := []byte(`{"entries":[{"enabled":true,"binding":{"key":" a ","useMod":true},"sendString":"pwd\\n"}]}`)
	var s KeyMapSettings
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	normalizeKeyMapSettings(&s)
	if len(s.Entries) != 1 {
		t.Fatalf("len=%d", len(s.Entries))
	}
	e := s.Entries[0]
	if e.ID == "" {
		t.Fatal("expected auto id")
	}
	if e.Action != "sendString" {
		t.Fatalf("action=%q", e.Action)
	}
	if e.Binding.Key != "a" {
		t.Fatalf("key=%q", e.Binding.Key)
	}
}

func TestDefaultKeyMapSettingsEmpty(t *testing.T) {
	s := DefaultKeyMapSettings()
	if s.Entries == nil || len(s.Entries) != 0 {
		t.Fatalf("expected empty entries, got %#v", s.Entries)
	}
}
