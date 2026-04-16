package downloader

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
)

// =============================================================================
// FileWriter.Write() 测试
// =============================================================================

func TestFileWriter_Write_Normal(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "filewriter_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 FileWriter
	fw := NewFileWriter(nil)

	// 准备测试数据
	testData := []byte("hello world")
	testPath := filepath.Join(tempDir, "test.txt")

	// 执行写入
	req := WriteRequest{
		Path: testPath,
		Data: testData,
		Options: WriteOptions{
			CreateVersion: false,
			SkipUnchanged: false,
		},
	}
	result, err := fw.Write(req)
	if err != nil {
		t.Fatalf("写入失败: %v", err)
	}

	// 验证结果
	if !result.Success {
		t.Error("期望 Success=true, 实际 false")
	}
	if result.Skipped {
		t.Error("期望 Skipped=false, 实际 true")
	}
	if result.NewSize != int64(len(testData)) {
		t.Errorf("期望 NewSize=%d, 实际 %d", len(testData), result.NewSize)
	}

	// 验证文件内容
	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	if string(content) != string(testData) {
		t.Errorf("期望内容 %q, 实际 %q", string(testData), string(content))
	}
}

func TestFileWriter_Write_SkipUnchanged(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "filewriter_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 FileWriter
	fw := NewFileWriter(nil)

	// 准备测试数据
	testData := []byte("hello world")
	testPath := filepath.Join(tempDir, "test.txt")

	// 第一次写入
	req := WriteRequest{
		Path: testPath,
		Data: testData,
		Options: WriteOptions{
			SkipUnchanged: true,
		},
	}
	result1, err := fw.Write(req)
	if err != nil {
		t.Fatalf("第一次写入失败: %v", err)
	}
	if result1.Skipped {
		t.Error("第一次写入不应该被跳过")
	}

	// 第二次写入相同内容（应该跳过）
	result2, err := fw.Write(req)
	if err != nil {
		t.Fatalf("第二次写入失败: %v", err)
	}

	// 验证跳过
	if !result2.Skipped {
		t.Error("期望第二次写入被跳过, 实际未跳过")
	}
	if !result2.Success {
		t.Error("跳过的写入仍应标记为成功")
	}

	// 第三次写入不同内容（不应该跳过）
	newData := []byte("hello world 2")
	req.Data = newData
	result3, err := fw.Write(req)
	if err != nil {
		t.Fatalf("第三次写入失败: %v", err)
	}
	if result3.Skipped {
		t.Error("写入不同内容不应该被跳过")
	}
}

func TestFileWriter_Write_CreateVersion(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "filewriter_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 VersionManager 和 FileWriter
	vm := NewVersionManagerWithWriter(".versions", nil)
	fw := NewFileWriter(vm)

	// 准备测试数据
	testPath := filepath.Join(tempDir, "test.txt")
	oldData := []byte("old content")
	newData := []byte("new content")

	// 第一次写入（创建文件，不需要创建版本）
	req := WriteRequest{
		Path: testPath,
		Data: oldData,
		Options: WriteOptions{
			CreateVersion: true,
		},
	}
	_, err = fw.Write(req)
	if err != nil {
		t.Fatalf("第一次写入失败: %v", err)
	}

	// 第二次写入（应该创建版本）
	req.Data = newData
	result, err := fw.Write(req)
	if err != nil {
		t.Fatalf("第二次写入失败: %v", err)
	}
	if !result.Success {
		t.Error("写入应该成功")
	}

	// 验证版本文件存在
	versionsDir := filepath.Join(tempDir, ".versions")
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		t.Fatalf("读取版本目录失败: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("期望存在版本文件, 但未找到")
	}

	// 验证版本文件内容是旧内容
	versionPath := filepath.Join(versionsDir, entries[0].Name())
	versionContent, err := os.ReadFile(versionPath)
	if err != nil {
		t.Fatalf("读取版本文件失败: %v", err)
	}
	if string(versionContent) != string(oldData) {
		t.Errorf("版本文件内容应为 %q, 实际 %q", string(oldData), string(versionContent))
	}

	// 验证当前文件内容是新内容
	currentContent, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("读取当前文件失败: %v", err)
	}
	if string(currentContent) != string(newData) {
		t.Errorf("当前文件内容应为 %q, 实际 %q", string(newData), string(currentContent))
	}
}

func TestFileWriter_Write_SetModTime(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "filewriter_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 FileWriter
	fw := NewFileWriter(nil)

	// 准备测试数据
	testData := []byte("hello world")
	testPath := filepath.Join(tempDir, "test.txt")

	// 设置特定的修改时间
	modTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// 执行写入
	req := WriteRequest{
		Path: testPath,
		Data: testData,
		Options: WriteOptions{
			ModTime: &modTime,
		},
	}
	_, err = fw.Write(req)
	if err != nil {
		t.Fatalf("写入失败: %v", err)
	}

	// 验证修改时间
	info, err := os.Stat(testPath)
	if err != nil {
		t.Fatalf("获取文件信息失败: %v", err)
	}

	// 比较修改时间（允许1秒误差，因为文件系统精度可能不同）
	actualModTime := info.ModTime().UTC()
	diff := actualModTime.Sub(modTime)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Second {
		t.Errorf("期望修改时间 %v, 实际 %v", modTime, actualModTime)
	}
}

func TestFileWriter_Write_NonExistentDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filewriter_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fw := NewFileWriter(nil)

	testData := []byte("hello world")
	nonExistentDir := filepath.Join(tempDir, "nonexistent", "nested", "dir")
	testPath := filepath.Join(nonExistentDir, "test.txt")

	req := WriteRequest{
		Path: testPath,
		Data: testData,
	}
	_, err = fw.Write(req)
	if err != nil {
		t.Fatalf("写入到不存在的目录失败: %v", err)
	}

	data, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	if string(data) != string(testData) {
		t.Errorf("期望内容 %q, 实际 %q", string(testData), string(data))
	}
}

// =============================================================================
// VersionManager.CreateVersion() 测试
// =============================================================================

func TestVersionManager_CreateVersion(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "versionmanager_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 VersionManager
	vm := NewVersionManagerWithWriter(".versions", nil)

	// 创建源文件
	sourcePath := filepath.Join(tempDir, "document.txt")
	sourceData := []byte("source content")
	if err := os.WriteFile(sourcePath, sourceData, 0644); err != nil {
		t.Fatalf("创建源文件失败: %v", err)
	}

	// 创建版本
	versionPath, err := vm.CreateVersion(sourcePath)
	if err != nil {
		t.Fatalf("创建版本失败: %v", err)
	}

	// 验证版本路径
	expectedDir := filepath.Join(tempDir, ".versions")
	if !strings.HasPrefix(versionPath, expectedDir) {
		t.Errorf("版本路径应在 %s 目录下, 实际: %s", expectedDir, versionPath)
	}

	// 验证版本文件命名格式: document_20060102_150405_NNN.txt
	versionFilename := filepath.Base(versionPath)
	pattern := `^document_\d{8}_\d{6}_\d{1,3}\.txt$`
	matched, err := regexp.MatchString(pattern, versionFilename)
	if err != nil {
		t.Fatalf("正则匹配失败: %v", err)
	}
	if !matched {
		t.Errorf("版本文件名格式不正确, 期望匹配 %s, 实际: %s", pattern, versionFilename)
	}

	// 验证版本文件内容
	versionContent, err := os.ReadFile(versionPath)
	if err != nil {
		t.Fatalf("读取版本文件失败: %v", err)
	}
	if string(versionContent) != string(sourceData) {
		t.Errorf("版本文件内容应为 %q, 实际 %q", string(sourceData), string(versionContent))
	}

	// 验证版本目录存在
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Error("版本目录应该存在")
	}
}

func TestVersionManager_CreateVersion_EmptyPath(t *testing.T) {
	vm := NewVersionManagerWithWriter(".versions", nil)

	_, err := vm.CreateVersion("")
	if err == nil {
		t.Error("期望空路径返回错误，但未返回")
	}
}

func TestVersionManager_CreateVersion_DotFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "versionmanager_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	vm := NewVersionManagerWithWriter(".versions", nil)

	sourcePath := filepath.Join(tempDir, ".gitignore")
	sourceData := []byte("gitignore content")
	if err := os.WriteFile(sourcePath, sourceData, 0644); err != nil {
		t.Fatalf("创建源文件失败: %v", err)
	}

	versionPath, err := vm.CreateVersion(sourcePath)
	if err != nil {
		t.Fatalf("创建版本失败: %v", err)
	}

	versionFilename := filepath.Base(versionPath)
	pattern := `^_unknown_\d{8}_\d{6}_\d{1,3}\.gitignore$`
	matched, err := regexp.MatchString(pattern, versionFilename)
	if err != nil {
		t.Fatalf("正则匹配失败: %v", err)
	}
	if !matched {
		t.Errorf("点开头的文件名格式不正确, 期望匹配 %s, 实际: %s", pattern, versionFilename)
	}
}

// =============================================================================
// Downloader.Download() 测试
// =============================================================================

func TestDownloader_Download_Normal(t *testing.T) {
	// 创建模拟 HTTP 服务器
	testData := []byte("test file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(testData)
	}))
	defer server.Close()

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "downloader_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 FileWriter 和 Downloader
	fw := NewFileWriter(nil)
	dl := NewDownloader(fw)

	// 准备下载请求
	destPath := filepath.Join(tempDir, "downloaded.txt")
	req := DownloadRequest{
		Context:     context.Background(),
		Client:      resty.New(),
		URL:         server.URL + "/test.txt",
		Destination: destPath,
		Options:     DownloadOptions{},
	}

	// 执行下载
	result, err := dl.Download(req)
	if err != nil {
		t.Fatalf("下载失败: %v", err)
	}

	// 验证结果
	if !result.Success {
		t.Error("期望 Success=true")
	}
	if result.Skipped {
		t.Error("期望 Skipped=false")
	}
	if result.FilePath != destPath {
		t.Errorf("期望 FilePath=%s, 实际 %s", destPath, result.FilePath)
	}
	if result.FileSize != int64(len(testData)) {
		t.Errorf("期望 FileSize=%d, 实际 %d", len(testData), result.FileSize)
	}

	// 验证文件内容
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	if string(content) != string(testData) {
		t.Errorf("期望内容 %q, 实际 %q", string(testData), string(content))
	}
}

func TestDownloader_Download_Error(t *testing.T) {
	// 创建模拟 HTTP 服务器（返回错误状态码）
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "downloader_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 FileWriter 和 Downloader
	fw := NewFileWriter(nil)
	dl := NewDownloader(fw)

	// 准备下载请求
	destPath := filepath.Join(tempDir, "downloaded.txt")
	req := DownloadRequest{
		Context:     context.Background(),
		Client:      resty.New(),
		URL:         server.URL + "/notfound.txt",
		Destination: destPath,
		Options:     DownloadOptions{},
	}

	// 执行下载（HTTP 404 现在应返回错误）
	result, err := dl.Download(req)
	if err == nil {
		t.Error("期望 HTTP 404 返回错误，但未返回")
	}
	if result == nil {
		t.Fatal("期望 result 不为 nil")
	}
	if result.Error == nil {
		t.Error("期望 result.Error 不为 nil（HTTP 404 应被记录）")
	}
	if result.Success {
		t.Error("期望 Success=false（HTTP 非成功状态码）")
	}

	// 测试无效 URL
	req.URL = "://invalid-url"
	_, err = dl.Download(req)
	if err == nil {
		t.Error("期望无效 URL 返回错误")
	}
}

func TestDownloader_Download_Callbacks(t *testing.T) {
	// 创建模拟 HTTP 服务器
	testData := []byte("test file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(testData)
	}))
	defer server.Close()

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "downloader_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 FileWriter 和 Downloader
	fw := NewFileWriter(nil)
	dl := NewDownloader(fw)

	// 准备回调测试变量
	var beforeCalled bool
	var afterCalled bool
	var receivedReq *DownloadRequest
	var receivedResult *DownloadResult
	var mu sync.Mutex

	// 准备下载请求
	destPath := filepath.Join(tempDir, "downloaded.txt")
	req := DownloadRequest{
		Context:     context.Background(),
		Client:      resty.New(),
		URL:         server.URL + "/test.txt",
		Destination: destPath,
		Options: DownloadOptions{
			OnBeforeDownload: func(r *DownloadRequest) {
				mu.Lock()
				beforeCalled = true
				receivedReq = r
				mu.Unlock()
			},
			OnAfterDownload: func(r *DownloadResult) {
				mu.Lock()
				afterCalled = true
				receivedResult = r
				mu.Unlock()
			},
		},
	}

	// 执行下载
	_, err = dl.Download(req)
	if err != nil {
		t.Fatalf("下载失败: %v", err)
	}

	// 验证回调被调用
	mu.Lock()
	defer mu.Unlock()

	if !beforeCalled {
		t.Error("OnBeforeDownload 回调未被调用")
	}
	if !afterCalled {
		t.Error("OnAfterDownload 回调未被调用")
	}
	if receivedReq == nil {
		t.Error("OnBeforeDownload 未接收到请求")
	}
	if receivedResult == nil {
		t.Error("OnAfterDownload 未接收到结果")
	}
	if receivedResult != nil && !receivedResult.Success {
		t.Error("OnAfterDownload 接收到的结果应为成功")
	}
}

// =============================================================================
// Downloader.BatchDownload() 测试
// =============================================================================

func TestDownloader_BatchDownload_Concurrent(t *testing.T) {
	// 创建模拟 HTTP 服务器
	var requestCount int
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()
		time.Sleep(50 * time.Millisecond) // 模拟延迟
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "content of %s", r.URL.Path)
	}))
	defer server.Close()

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "downloader_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 FileWriter 和 Downloader
	fw := NewFileWriter(nil)
	dl := NewDownloader(fw)

	// 准备多个下载请求
	numRequests := 5
	reqs := make([]DownloadRequest, numRequests)
	for i := 0; i < numRequests; i++ {
		reqs[i] = DownloadRequest{
			Context:     context.Background(),
			Client:      resty.New(),
			URL:         fmt.Sprintf("%s/file%d.txt", server.URL, i),
			Destination: filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i)),
			Options:     DownloadOptions{},
		}
	}

	// 执行批量下载
	start := time.Now()
	results, err := dl.BatchDownload(context.Background(), reqs)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("批量下载失败: %v", err)
	}

	// 验证结果数量
	if len(results) != numRequests {
		t.Errorf("期望 %d 个结果, 实际 %d", numRequests, len(results))
	}

	// 验证所有下载成功
	for i, result := range results {
		if !result.Success {
			t.Errorf("文件 %d 下载失败", i)
		}
	}

	// 验证并发执行（串行需要 5*50ms=250ms，并发应小于此时间）
	if elapsed >= 200*time.Millisecond {
		t.Errorf("并发下载耗时 %v, 似乎未正确并发执行", elapsed)
	}

	// 验证所有请求都被处理
	mu.Lock()
	count := requestCount
	mu.Unlock()
	if count != numRequests {
		t.Errorf("期望 %d 个请求, 实际 %d", numRequests, count)
	}
}

func TestDownloader_BatchDownload_ErrorAggregation(t *testing.T) {
	// 创建模拟 HTTP 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "error") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "downloader_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 FileWriter 和 Downloader
	fw := NewFileWriter(nil)
	dl := NewDownloader(fw)

	// 准备混合请求（成功和失败）
	reqs := []DownloadRequest{
		{
			Context:     context.Background(),
			Client:      resty.New(),
			URL:         server.URL + "/ok1.txt",
			Destination: filepath.Join(tempDir, "ok1.txt"),
			Options:     DownloadOptions{},
		},
		{
			Context:     context.Background(),
			Client:      resty.New(),
			URL:         server.URL + "/error.txt",
			Destination: filepath.Join(tempDir, "error.txt"),
			Options:     DownloadOptions{},
		},
		{
			Context:     context.Background(),
			Client:      resty.New(),
			URL:         server.URL + "/ok2.txt",
			Destination: filepath.Join(tempDir, "ok2.txt"),
			Options:     DownloadOptions{},
		},
	}

	// 执行批量下载
	results, firstErr := dl.BatchDownload(context.Background(), reqs)

	// 验证返回了结果数量
	if len(results) != 3 {
		t.Errorf("期望 3 个结果, 实际 %d", len(results))
	}

	// 验证成功的下载（HTTP 500 现在应返回错误）
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}
	if successCount != 2 {
		t.Errorf("期望 2 个成功(ok1+ok2), 实际 %d", successCount)
	}

	// firstErr 应为非 nil（HTTP 500 触发了错误）
	if firstErr == nil {
		t.Error("期望 firstErr 不为 nil（HTTP 500 应返回错误）")
	}
}

func TestDownloader_BatchDownload_ContextCancel(t *testing.T) {
	// 创建模拟 HTTP 服务器（有延迟）
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "downloader_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建 FileWriter 和 Downloader
	fw := NewFileWriter(nil)
	dl := NewDownloader(fw)

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 准备多个下载请求（使用可取消的上下文）
	numRequests := 5
	reqs := make([]DownloadRequest, numRequests)
	for i := 0; i < numRequests; i++ {
		reqs[i] = DownloadRequest{
			Context:     ctx, // 使用可取消的上下文
			Client:      resty.New(),
			URL:         fmt.Sprintf("%s/file%d.txt", server.URL, i),
			Destination: filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i)),
			Options:     DownloadOptions{},
		}
	}

	// 在短时间内取消上下文
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// 执行批量下载
	results, err := dl.BatchDownload(context.Background(), reqs)

	// 验证结果数量正确
	if len(results) != numRequests {
		t.Errorf("期望 %d 个结果, 实际 %d", numRequests, len(results))
	}

	// 验证部分结果可能包含取消错误（由于请求上下文被取消）
	cancelledCount := 0
	for _, result := range results {
		if result != nil && result.Error != nil {
			cancelledCount++
		}
	}

	// 由于请求上下文被取消，应该有一些请求失败
	if cancelledCount == 0 {
		t.Log("警告: 没有请求失败，可能所有请求在取消前已完成")
	}

	// 验证 BatchDownload 返回的错误（第一个错误）
	if err == nil && cancelledCount > 0 {
		t.Error("期望 BatchDownload 返回错误")
	}
}

// =============================================================================
// Phase 1.1 验证: per-file sync.Map 锁正确性
// =============================================================================

func TestFileWriter_ConcurrentDifferentFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filewriter_concurrent_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fw := NewFileWriter(nil)

	numFiles := 20
	var mu sync.Mutex
	var errors []error
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < numFiles; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			data := []byte(fmt.Sprintf("content-%d", idx))
			path := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", idx))

			req := WriteRequest{
				Path: path,
				Data: data,
				Options: WriteOptions{
					CreateVersion: false,
					SkipUnchanged: false,
				},
			}
			result, writeErr := fw.Write(req)
			if writeErr != nil || !result.Success {
				mu.Lock()
				errors = append(errors, fmt.Errorf("file %d failed: %v", idx, writeErr))
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	if len(errors) > 0 {
		for _, e := range errors {
			t.Error(e)
		}
	}

	for i := 0; i < numFiles; i++ {
		path := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("读取文件 %d 失败: %v", i, err)
			continue
		}
		expected := fmt.Sprintf("content-%d", i)
		if string(content) != expected {
			t.Errorf("文件 %d 内容不匹配: 期望 %q, 实际 %q", i, expected, string(content))
		}
	}

	t.Logf("%d 个不同文件并发写入耗时: %v", numFiles, elapsed)
}

func TestFileWriter_ConcurrentSameFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filewriter_concurrent_same_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fw := NewFileWriter(nil)

	samePath := filepath.Join(tempDir, "same_file.txt")
	numWriters := 10
	var wg sync.WaitGroup
	var successCount int64
	var mu sync.Mutex

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			data := []byte(fmt.Sprintf("writer-%d-data", idx))
			req := WriteRequest{
				Path: samePath,
				Data: data,
				Options: WriteOptions{
					CreateVersion: false,
					SkipUnchanged: false,
				},
			}
			result, writeErr := fw.Write(req)
			if writeErr == nil && result.Success {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	if successCount != int64(numWriters) {
		t.Errorf("期望 %d 个写入全部成功, 实际成功 %d", numWriters, successCount)
	}

	// 验证最终文件存在且内容是最后一次写入的内容
	content, err := os.ReadFile(samePath)
	if err != nil {
		t.Fatalf("读取目标文件失败: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("文件内容为空")
	}

	validContent := false
	for i := 0; i < numWriters; i++ {
		expected := fmt.Sprintf("writer-%d-data", i)
		if string(content) == expected {
			validContent = true
			break
		}
	}
	if !validContent {
		t.Errorf("文件内容不是任何一次有效写入的结果, 实际: %q", string(content))
	}

	t.Logf("%d 个 goroutine 并发写入同一文件全部完成, 最终内容来自其中一个 writer", numWriters)
}

// =============================================================================
// Phase 1.3 验证: VersionManager 注入 FileWriter 后使用它写入版本文件
// =============================================================================

func TestVersionManager_WithFileWriter(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "versionmanager_fw_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fw := NewFileWriter(nil)
	vm := NewVersionManagerWithWriter(".versions", fw)

	sourcePath := filepath.Join(tempDir, "important_data.json")
	sourceData := []byte(`{"key": "value", "data": [1, 2, 3]}`)
	if err := os.WriteFile(sourcePath, sourceData, 0644); err != nil {
		t.Fatalf("创建源文件失败: %v", err)
	}

	versionPath, err := vm.CreateVersion(sourcePath)
	if err != nil {
		t.Fatalf("创建版本失败: %v", err)
	}

	versionContent, err := os.ReadFile(versionPath)
	if err != nil {
		t.Fatalf("读取版本文件失败: %v", err)
	}
	if string(versionContent) != string(sourceData) {
		t.Errorf("版本文件内容不匹配\n期望: %s\n实际: %s", string(sourceData), string(versionContent))
	}

	versionsDir := filepath.Join(tempDir, ".versions")
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		t.Fatalf("读取版本目录失败: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("版本目录为空")
	}
	t.Logf("注入 FileWriter 的 VersionManager 成功创建版本: %s", entries[0].Name())
}

func TestVersionManager_WithoutFileWriter_Fallback(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "versionmanager_nofw_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	vm := NewVersionManagerWithWriter(".versions", nil)

	sourcePath := filepath.Join(tempDir, "fallback_test.txt")
	sourceData := []byte("fallback content")
	if err := os.WriteFile(sourcePath, sourceData, 0644); err != nil {
		t.Fatalf("创建源文件失败: %v", err)
	}

	versionPath, err := vm.CreateVersion(sourcePath)
	if err != nil {
		t.Fatalf("创建版本失败(回退路径): %v", err)
	}

	versionContent, err := os.ReadFile(versionPath)
	if err != nil {
		t.Fatalf("读取版本文件失败: %v", err)
	}
	if string(versionContent) != string(sourceData) {
		t.Errorf("回退路径版本文件内容不匹配\n期望: %s\n实际: %s", string(sourceData), string(versionContent))
	}
	t.Log("无 FileWriter 注入时回退到 os.WriteFile 成功")
}

// =============================================================================
// Phase 3.1 验证: ExtractImageExtFromURL 工具函数
// =============================================================================

func TestExtractImageExtFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"标准 jpg URL", "https://pbs.twimg.com/media/photo.jpg", ".jpg"},
		{"大写 JPG URL", "https://example.com/IMAGE.JPG", ".jpg"},
		{"混合大小写 JPEG", "https://cdn.example.com/photo.JPEG", ".jpeg"},
		{"PNG 图片", "https://example.com/icon.PNG", ".png"},
		{"GIF 动图", "https://media.example.com/anim.GIF", ".gif"},
		{"WebP 格式", "https://cdn.example.com/img.webp", ".webp"},
		{"带查询参数的 jpg", "https://pbs.twimg.com/media/photo.jpg?name=4096x4096", ".jpg"},
		{"带路径段的 png", "https://cdn.example.com/a/b/c/image.png", ".png"},
		{"无扩展名默认 jpg", "https://pbs.twimg.com/media/noext", ".jpg"},
		{"空字符串默认 jpg", "", ".jpg"},
		{"未知扩展名默认 jpg", "https://example.com/file.xyz", ".jpg"},
		{"视频 URL 默认 jpg", "https://video.twimg.com/tweet_video/123.mp4", ".jpg"},
		{"tweet_video 路径", "https://tweet_video/abc.mp4", ".jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractImageExtFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("ExtractImageExtFromURL(%q) = %q, 期望 %q", tt.url, result, tt.expected)
			}
		})
	}
}
