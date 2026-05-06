package api

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:web-vue/dist
var webFS embed.FS

// getWebFS 获取web-vue/dist子文件系统
func getWebFS() fs.FS {
	subFS, err := fs.Sub(webFS, "web-vue/dist")
	if err != nil {
		// 如果失败，返回原始FS
		return webFS
	}
	return subFS
}

// handleWeb 返回 Web 管理页面
func (s *Server) handleWeb(w http.ResponseWriter, r *http.Request) {
	fsys := getWebFS()
	data, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load web page")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("ETag", "\"v2.0.0\"")
	if _, err := w.Write(data); err != nil {
		// 这里只记录日志，因为头部可能已经发送，无法再返回 HTTP 500
		return
	}
}

// handleStatic 静态文件服务
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	reqPath := r.PathValue("path")
	
	// 调试日志
	fmt.Printf("[DEBUG] handleStatic called, URL=%s, path=%s\n", r.URL.Path, reqPath)
	
	if reqPath == "" {
		fmt.Printf("[DEBUG] reqPath is empty, returning 404\n")
		http.NotFound(w, r)
		return
	}

	// 使用 path.Clean 来规范化路径，自动处理掉所有的 "." 和 ".." 以及多余的斜杠
	cleanPath := path.Clean("/" + reqPath)

	// 确保规范化后的路径不会逃逸出根目录
	if strings.Contains(cleanPath, "..") {
		http.NotFound(w, r)
		return
	}

	cleanPath = strings.TrimPrefix(cleanPath, "/")
	
	fmt.Printf("[DEBUG] Looking for file: %s\n", cleanPath)

	fsys := getWebFS()
	data, err := fs.ReadFile(fsys, cleanPath)
	if err != nil {
		fmt.Printf("[DEBUG] File not found: %s, error: %v\n", cleanPath, err)
		http.NotFound(w, r)
		return
	}
	
	fmt.Printf("[DEBUG] File found: %s, size: %d bytes\n", cleanPath, len(data))

	contentType := "application/octet-stream"
	switch {
	case strings.HasSuffix(cleanPath, ".html"):
		contentType = "text/html; charset=utf-8"
	case strings.HasSuffix(cleanPath, ".css"):
		contentType = "text/css; charset=utf-8"
	case strings.HasSuffix(cleanPath, ".js"):
		contentType = "application/javascript; charset=utf-8"
	case strings.HasSuffix(cleanPath, ".json"):
		contentType = "application/json"
	case strings.HasSuffix(cleanPath, ".png"):
		contentType = "image/png"
	case strings.HasSuffix(cleanPath, ".jpg"), strings.HasSuffix(cleanPath, ".jpeg"):
		contentType = "image/jpeg"
	case strings.HasSuffix(cleanPath, ".svg"):
		contentType = "image/svg+xml"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("ETag", "\"v2.0.0\"")
	if _, err := w.Write(data); err != nil {
		return
	}
}
