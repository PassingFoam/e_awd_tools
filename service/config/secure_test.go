package config

import (
	"testing"

	"gopkg.in/ini.v1"
)

// TestEnsureSecrets_GeneratesWhenEmpty 验证空 token/密码时自动生成、镜像写 [client]、同步全局。
func TestEnsureSecrets_GeneratesWhenEmpty(t *testing.T) {
	cfg, err := ini.Load([]byte("[server]\n[client]\n"))
	if err != nil {
		t.Fatalf("加载 ini 失败: %v", err)
	}

	origT, origP := Cs_token, Admin_password
	t.Cleanup(func() { Cs_token, Admin_password = origT, origP })

	ensureSecrets(cfg)

	srvTok := cfg.Section("server").Key("cs_token").String()
	cliTok := cfg.Section("client").Key("cs_token").String()
	pwd := cfg.Section("server").Key("admin_password").String()

	if srvTok == "" {
		t.Error("server.cs_token 未生成")
	}
	if srvTok != cliTok {
		t.Errorf("client.cs_token 未镜像: server=%s client=%s", srvTok, cliTok)
	}
	if Cs_token != srvTok {
		t.Errorf("全局 Cs_token 未同步: got %s want %s", Cs_token, srvTok)
	}
	if pwd == "" {
		t.Error("server.admin_password 未生成")
	}
	if Admin_password != pwd {
		t.Errorf("全局 Admin_password 未同步: got %s want %s", Admin_password, pwd)
	}
}

// TestEnsureSecrets_KeepsExisting 验证已存在的 token/密码不被覆盖。
func TestEnsureSecrets_KeepsExisting(t *testing.T) {
	cfg, err := ini.Load([]byte("[server]\ncs_token = mytoken\nadmin_password = mypwd\n[client]\n"))
	if err != nil {
		t.Fatalf("加载 ini 失败: %v", err)
	}

	origT, origP := Cs_token, Admin_password
	t.Cleanup(func() { Cs_token, Admin_password = origT, origP })

	ensureSecrets(cfg)

	if got := cfg.Section("server").Key("cs_token").String(); got != "mytoken" {
		t.Errorf("cs_token 被覆盖: got %s want mytoken", got)
	}
	if got := cfg.Section("server").Key("admin_password").String(); got != "mypwd" {
		t.Errorf("admin_password 被覆盖: got %s want mypwd", got)
	}
	// 注意：非空时 ensureSecrets 不修改 ini 也不动全局变量——
	// 全局同步由 Init_conf 的读取阶段负责（生产流程中 Init_conf 先读 ini 到全局，再调 ensureSecrets）。
	// 因此此处仅验证 ini 中的值被保留、未被覆盖。
}

// TestRandHex 验证随机串长度与非空。
func TestRandHex(t *testing.T) {
	for _, n := range []int{1, 8, 32} {
		s := randHex(n)
		if len(s) != n*2 {
			t.Errorf("randHex(%d) 长度 = %d, want %d", n, len(s), n*2)
		}
	}
	// 两次调用应几乎不可能相同（概率性，但 32 字节碰撞可忽略）
	a, b := randHex(32), randHex(32)
	if a == b {
		t.Errorf("randHex(32) 两次调用相同（极不应该）: %s", a)
	}
}
