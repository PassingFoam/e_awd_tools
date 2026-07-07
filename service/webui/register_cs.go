package webui

import "github.com/gin-gonic/gin"

// RegisterCS 把 C/S 端口（rCS）需要的 webui handler 注册到给定 engine。
//
// 背景：pcap_upload 同时被两端调用——
//   - 管理端浏览器手动上传（走 rAdmin 的 /webui/pcap_upload，由 Register 注册）
//   - client 程序自动上传设备抓包（走 rCS 的 /api/pcap_upload，由本函数注册）
//
// 两端复用同一个 handler（pcap_upload，定义在 pcap.go），仅路径前缀不同。
func RegisterCS(r *gin.Engine) {
	r.POST("/api/pcap_upload", pcap_upload)
}
