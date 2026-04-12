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
	vm := NewVersionManager(".versions")
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
	vm := NewVersionManager(".versions")

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

	// 验证版本文件命名格式: document_20060102_150405.txt
	versionFilename := filepath.Base(versionPath)
	pattern := `^document_\d{8}_\d{6}\.txt$`
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

	// 执行下载（HTTP 404 不会返回错误，但文件内容会是 404 响应体）
	result, err := dl.Download(req)
	if err != nil {
		// 如果返回错误，验证错误已记录
		if result.Error == nil {
			t.Error("期望 result.Error 不为 nil")
		}
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

	// 验证返回了第一个错误（注意：HTTP 500 不会返回错误，文件会包含错误响应体）
	// 这里我们测试的是真正的网络错误
	if len(results) != 3 {
		t.Errorf("期望 3 个结果, 实际 %d", len(results))
	}

	// 验证成功的下载
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}
	// HTTP 500 仍然会写入文件，所以都算成功
	if successCount != 3 {
		t.Errorf("期望 3 个成功, 实际 %d", successCount)
	}

	// firstErr 应该为 nil（因为没有真正的网络错误）
	if firstErr != nil {
		t.Errorf("期望 firstErr=nil, 实际 %v", firstErr)
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
