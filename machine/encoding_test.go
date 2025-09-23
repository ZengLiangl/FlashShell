package machine

import (
	"testing"
)

func TestConvertToUTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "Valid UTF-8",
			input:    []byte("Hello, 世界"),
			expected: "Hello, 世界",
		},
		{
			name:     "ASCII only",
			input:    []byte("Hello World"),
			expected: "Hello World",
		},
		{
			name:     "Empty input",
			input:    []byte(""),
			expected: "",
		},
		{
			name:     "Invalid UTF-8 bytes",
			input:    []byte{0xFF, 0xFE, 0x48, 0x65, 0x6C, 0x6C, 0x6F}, // Invalid UTF-8 + "Hello"
			expected: "Hello",
		},
		{
			name:     "Mixed valid and invalid",
			input:    []byte{0x48, 0x65, 0x6C, 0x6C, 0x6F, 0xFF, 0x20, 0x57, 0x6F, 0x72, 0x6C, 0x64}, // "Hello" + invalid + " World"
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToUTF8(tt.input)
			if result != tt.expected {
				t.Errorf("convertToUTF8() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConvertToUTF8_ChineseCharacters(t *testing.T) {
	// 测试中文字符处理
	input := []byte("测试中文字符")
	expected := "测试中文字符"

	result := convertToUTF8(input)
	if result != expected {
		t.Errorf("convertToUTF8() with Chinese = %q, want %q", result, expected)
	}
}

func TestConvertToUTF8_SpecialCharacters(t *testing.T) {
	// 测试特殊字符处理
	input := []byte("Line 1\nLine 2\r\nLine 3\t\x00")

	result := convertToUTF8(input)

	// 结果应该包含换行符和制表符，但跳过空字节
	if len(result) == 0 {
		t.Error("convertToUTF8() should not return empty string for input with valid characters")
	}

	// 检查是否包含可见字符
	if !contains(result, "Line 1") || !contains(result, "Line 2") || !contains(result, "Line 3") {
		t.Errorf("convertToUTF8() should preserve valid text, got: %q", result)
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsAt(s, substr)))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
