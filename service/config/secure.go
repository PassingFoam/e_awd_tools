package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"

	"gopkg.in/ini.v1"
)

// ensureSecrets 在 Server 模式下确保 cs_token 与 admin_password 存在。
// 若为空则随机生成、回填到 config.ini（cs_token 同时镜像写入 [client]，便于
// server+client 同机自动同步），并同步到对应的全局变量。
// 仅在 Server_mode 时调用（admin_password / 生成 token 都只属于 server）。
func ensureSecrets(cfg *ini.File) {
	srv := cfg.Section("server")

	// cs_token：C/S 共享密钥，强制非空
	if srv.Key("cs_token").String() == "" {
		tok := randHex(32) // 64 个 hex 字符
		Cs_token = tok
		srv.Key("cs_token").SetValue(tok)
		// 镜像写入 [client]，便于 server+client 同配置自动同步
		cfg.Section("client").Key("cs_token").SetValue(tok)
		log.Printf("[SEC] 已自动生成 cs_token 并写入 [server] 与 [client]，请将该 token 分发到各 client 配置")
	}

	// admin_password：管理端登录密码，强制非空（避免裸奔）
	if srv.Key("admin_password").String() == "" {
		pwd := randHex(8) // 16 个 hex 字符
		Admin_password = pwd
		srv.Key("admin_password").SetValue(pwd)
		log.Printf("[SEC] ==================== 管理端登录密码（首次自动生成）: %s ====================", pwd)
	}

	// cs_whitelist 为空时打印告警（不强制，空=仅靠 token 保护）
	if Cs_whitelist == "" {
		log.Printf("[SEC] 警告: cs_whitelist 为空，C/S 端口将仅靠 cs_token 保护（任意 IP 均可连）")
	}
}

// randHex 生成 n 字节的随机 hex 字符串（长度为 2n）。
func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand 极少失败；失败直接 fatal 让用户立刻察觉，避免用弱密钥
		log.Fatalf("生成随机密钥失败: %v", err)
	}
	return hex.EncodeToString(b)
}
