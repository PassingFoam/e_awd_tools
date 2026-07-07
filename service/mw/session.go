package mw

import (
	"time"

	cache "github.com/patrickmn/go-cache"
)

const (
	sessionCookie = "0e7_admin"   // 管理端会话 cookie 名
	sessionTTL    = 12 * time.Hour // 会话有效期
)

// sessions 内存会话表：sid -> 登录时间。
// 复用已有依赖 go-cache（与 service/proxy/cache.go 一致）。
// 单实例够用；重启即失效（需重新登录），注销即删——对 AWD 管理端可接受。
var sessions = cache.New(sessionTTL, time.Hour)

func putSession(sid string)      { sessions.Set(sid, time.Now(), cache.DefaultExpiration) }
func hasSession(sid string) bool { _, ok := sessions.Get(sid); return ok }
func dropSession(sid string)     { sessions.Delete(sid) }
