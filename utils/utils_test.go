package utils

import (
	"0E7/service/config"
	"strings"
	"testing"
)

// TestNewCSRequest 注入 token 验证：配置非空时注入 X-CS-Token；配置空时不注入。
func TestNewCSRequest(t *testing.T) {
	orig := config.Cs_token
	t.Cleanup(func() { config.Cs_token = orig })

	// 有 token：应注入 header
	config.Cs_token = "abc123"
	req, err := NewCSRequest("GET", "http://example.com/x", nil)
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	if got := req.Header.Get("X-CS-Token"); got != "abc123" {
		t.Errorf("X-CS-Token = %q, want abc123", got)
	}
	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}

	// 无 token：不应注入 header，但请求仍可用
	config.Cs_token = ""
	req2, err := NewCSRequest("POST", "http://example.com/y", strings.NewReader("body"))
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	if got := req2.Header.Get("X-CS-Token"); got != "" {
		t.Errorf("空 token 时不应注入 X-CS-Token, got %q", got)
	}
	if req2.Method != "POST" {
		t.Errorf("Method = %q, want POST", req2.Method)
	}
}
