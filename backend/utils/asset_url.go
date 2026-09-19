package utils

import (
	"net/http"
	"os"
	"strings"
)

// AssetURL 将后端相对资源路径（/storage/*、/static/*）解析为绝对 URL，
// 避免 Capacitor WebView（非同源）下图片/语音加载失败。
// 可用环境变量 PUBLIC_BASE_URL 强制指定对外基址（反代时推荐）。
func AssetURL(r *http.Request, path string) string {
	if path == "" {
		return path
	}
	lower := strings.ToLower(path)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "blob:") ||
		strings.HasPrefix(lower, "capacitor:") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	base := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/")
	if base == "" && r != nil {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
			scheme = strings.ToLower(strings.TrimSpace(p))
		}
		if r.Host != "" {
			base = scheme + "://" + r.Host
		}
	}
	if base == "" {
		return path
	}
	return base + path
}
