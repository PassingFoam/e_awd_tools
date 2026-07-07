package mw

import (
	"0E7/service/config"
	"crypto/subtle"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

// CSToken 校验请求头 X-CS-Token 是否等于配置的共享 cs_token（常量时间比较）。
// 用于 C/S 端口（rCS），保护 /api/* 不被未授权方调用。
func CSToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.Cs_token == "" {
			// 兜底：正常 server 启动时 ensureSecrets 已生成 token；为空则放行避免锁死
			c.Next()
			return
		}
		got := c.GetHeader("X-CS-Token")
		if subtle.ConstantTimeCompare([]byte(got), []byte(config.Cs_token)) != 1 {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid cs token"})
			return
		}
		c.Next()
	}
}

// CSWhitelist 校验客户端 IP 是否在配置的 cs_whitelist（逗号分隔 IP/CIDR）内。
// 空白名单 = 不限制。前置条件：rCS 已调用 SetTrustedProxies(nil)，
// 此时 c.ClientIP() 取的是裸 TCP RemoteAddr，否则攻击者可伪造 X-Forwarded-For 绕过。
func CSWhitelist() gin.HandlerFunc {
	nets := parseWhitelist(config.Cs_whitelist) // 中间件注册时一次性解析
	return func(c *gin.Context) {
		if len(nets) == 0 {
			c.Next()
			return
		}
		ip := net.ParseIP(c.ClientIP())
		if ip == nil {
			c.AbortWithStatusJSON(403, gin.H{"error": "invalid client ip"})
			return
		}
		for _, n := range nets {
			if n.Contains(ip) {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(403, gin.H{"error": "ip not in whitelist"})
	}
}

// parseWhitelist 解析逗号分隔的 IP/CIDR 列表；纯 IP 自动补 /32（IPv4）或 /128（IPv6）。
func parseWhitelist(s string) []*net.IPNet {
	var out []*net.IPNet
	for _, raw := range strings.Split(s, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if !strings.Contains(raw, "/") {
			if strings.Contains(raw, ":") {
				raw += "/128"
			} else {
				raw += "/32"
			}
		}
		if _, ipn, err := net.ParseCIDR(raw); err == nil {
			out = append(out, ipn)
		}
	}
	return out
}
