package mw

import (
	"0E7/service/config"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) } // 抑制测试时的 gin debug 日志

// TestParseWhitelist 覆盖 IP/CIDR 解析的各类边界。
func TestParseWhitelist(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string // 期望的各 IPNet.String()
	}{
		{"空字符串", "", nil},
		{"纯IPv4自动补/32", "127.0.0.1", []string{"127.0.0.1/32"}},
		{"CIDR原样", "10.0.0.0/8", []string{"10.0.0.0/8"}},
		{"逗号分隔多项", "1.2.3.4,10.0.0.0/8", []string{"1.2.3.4/32", "10.0.0.0/8"}},
		{"含空格trim", " 1.2.3.4 , 5.6.7.8 ", []string{"1.2.3.4/32", "5.6.7.8/32"}},
		{"IPv6自动补/128", "::1", []string{"::1/128"}},
		{"非法项被跳过保留合法", "foobar,1.2.3.4", []string{"1.2.3.4/32"}},
		{"全是非法项", "xxx,yyy", nil},
		{"尾部分隔符", "1.2.3.4,", []string{"1.2.3.4/32"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseWhitelist(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("条目数 = %d, want %d (got %+v)", len(got), len(tc.want), got)
			}
			for i, n := range got {
				if n.String() != tc.want[i] {
					t.Errorf("[%d] = %s, want %s", i, n.String(), tc.want[i])
				}
			}
		})
	}
}

// TestCSToken 覆盖 token 校验：空配置兜底放行、无/错/正确 token。
func TestCSToken(t *testing.T) {
	orig := config.Cs_token
	t.Cleanup(func() { config.Cs_token = orig })

	cases := []struct {
		name       string
		confToken  string
		header     string // X-CS-Token 值；"" 表示不带该 header
		wantStatus int
	}{
		{"配置为空-兜底放行", "", "", 200},
		{"配置非空-无header-拒绝", "secret", "", 401},
		{"配置非空-错误token-拒绝", "secret", "wrong", 401},
		{"配置非空-正确token-放行", "secret", "secret", 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config.Cs_token = tc.confToken
			w := httptest.NewRecorder()
			r := gin.New()
			r.Use(CSToken())
			r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

			req := httptest.NewRequest("GET", "/x", nil)
			if tc.header != "" {
				req.Header.Set("X-CS-Token", tc.header)
			}
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

// TestCSWhitelist 覆盖 IP 白名单：空白名单放行、在/不在、CIDR 包含。
func TestCSWhitelist(t *testing.T) {
	orig := config.Cs_whitelist
	t.Cleanup(func() { config.Cs_whitelist = orig })

	cases := []struct {
		name       string
		whitelist  string
		remoteAddr string // 模拟 TCP 对端地址
		wantStatus int
	}{
		{"空白名单-放行", "", "1.2.3.4:1234", 200},
		{"IP在白名单", "127.0.0.1/32", "127.0.0.1:1234", 200},
		{"IP不在白名单-拒绝", "8.8.8.8/32", "127.0.0.1:1234", 403},
		{"落在CIDR内", "10.0.0.0/8", "10.1.2.3:1234", 200},
		{"落在CIDR外-拒绝", "10.0.0.0/8", "192.168.0.1:1234", 403},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config.Cs_whitelist = tc.whitelist // 必须在 Use 前设置：CSWhitelist 注册时一次性解析
			w := httptest.NewRecorder()
			r := gin.New()
			_ = r.SetTrustedProxies(nil) // 与生产一致：用裸 RemoteAddr，防 XFF 伪造
			r.Use(CSWhitelist())
			r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

			req := httptest.NewRequest("GET", "/x", nil)
			req.RemoteAddr = tc.remoteAddr
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

// TestCSToken_Whitelist_Order 验证 rCS 中间件顺序：白名单先生效（IP 不在则 403，不依赖 token）。
func TestCSToken_Whitelist_Order(t *testing.T) {
	origT, origW := config.Cs_token, config.Cs_whitelist
	t.Cleanup(func() { config.Cs_token, config.Cs_whitelist = origT, origW })

	config.Cs_token = "secret"
	config.Cs_whitelist = "8.8.8.8/32" // 本机 127.0.0.1 不在

	w := httptest.NewRecorder()
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.Use(CSWhitelist(), CSToken()) // 与生产 rCS 顺序一致
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-CS-Token", "secret") // 即使带正确 token
	req.RemoteAddr = "127.0.0.1:1234"      // IP 不在白名单
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("白名单应先生效：status = %d, want 403", w.Code)
	}
}
