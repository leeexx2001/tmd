package api

import (
	"fmt"
	"io/fs"
	"testing"
)

func TestEmbedFS(t *testing.T) {
	fsys := getWebFS()

	// 测试读取 index.html
	data, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		t.Errorf("Error reading index.html: %v", err)
	} else {
		fmt.Printf("index.html: %d bytes\n", len(data))
	}

	// 测试读取 assets 下的文件
	data2, err := fs.ReadFile(fsys, "assets/index-BoG-VqNf.js")
	if err != nil {
		t.Errorf("Error reading assets/index-BoG-VqNf.js: %v", err)
	} else {
		fmt.Printf("assets/index-BoG-VqNf.js: %d bytes\n", len(data2))
	}

	// 列出 assets 目录
	entries, err := fs.ReadDir(fsys, "assets")
	if err != nil {
		t.Errorf("Error reading assets dir: %v", err)
	} else {
		fmt.Printf("assets directory has %d entries\n", len(entries))
	}
}
