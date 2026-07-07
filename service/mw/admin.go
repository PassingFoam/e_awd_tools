package mw

import (
	"0E7/service/config"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AdminAuth 管理端鉴权中间件。
//   - cookie 会话优先（浏览器 fetch 与 WebSocket 都自动带）；
//   - 否则回落 HTTP Basic（user=admin, pass=Admin_password），覆盖 /git/* 的 git CLI 调用；
//   - 放行首页 / 静态资源 / 登录与状态接口本身。
// 须在 etag/gzip 之前注册，避免 401 响应被 etag 缓存或 gzip 包装。
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		// 放行：首页与静态资源（登录页需要加载）、登录与状态接口
		if p == "/" || strings.HasPrefix(p, "/static/") ||
			p == "/api/admin/login" || p == "/api/admin/status" {
			c.Next()
			return
		}
		// 1) cookie 会话
		if sid, err := c.Cookie(sessionCookie); err == nil && hasSession(sid) {
			c.Next()
			return
		}
		// 2) HTTP Basic 回落（供 git CLI: http://admin:pwd@host）
		if user, pass, ok := c.Request.BasicAuth(); ok &&
			subtle.ConstantTimeCompare([]byte(user), []byte("admin")) == 1 &&
			subtle.ConstantTimeCompare([]byte(pass), []byte(config.Admin_password)) == 1 {
			c.Next()
			return
		}
		c.Header("WWW-Authenticate", `Basic realm="0E7 admin"`)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
}

// Login 校验密码，成功后下发 HttpOnly + SameSite=Strict 的会话 cookie。
func Login(c *gin.Context) {
	var body struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad body"})
		return
	}
	if subtle.ConstantTimeCompare([]byte(body.Password), []byte(config.Admin_password)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}
	sid := randSID()
	putSession(sid)
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(sessionCookie, sid, int(sessionTTL.Seconds()), "/", "",
		config.Server_tls, true) // Secure 随 TLS；HttpOnly=true
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// Logout 清除当前会话。
func Logout(c *gin.Context) {
	if sid, err := c.Cookie(sessionCookie); err == nil {
		dropSession(sid)
	}
	c.SetCookie(sessionCookie, "", -1, "/", "", config.Server_tls, true)
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// Status 供前端首屏探测登录态（不经 AdminAuth，读 cookie 判断）。
func Status(c *gin.Context) {
	sid, err := c.Cookie(sessionCookie)
	c.JSON(http.StatusOK, gin.H{"logged_in": err == nil && hasSession(sid)})
}

func randSID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
