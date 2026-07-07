package utils

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"net/http"

	"0E7/service/config"
)

// NewCSRequest 构造一个带 X-CS-Token 头的 C/S 请求（client → server）。
// 在 service/client 与 service/update 两处复用，保证所有发往 C/S 端口的请求都自动带 token。
func NewCSRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if config.Cs_token != "" {
		req.Header.Set("X-CS-Token", config.Cs_token)
	}
	return req, nil
}

func GetMd5FromString(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func GetMd5FromBytes(b []byte) string {
	h := md5.New()
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}
