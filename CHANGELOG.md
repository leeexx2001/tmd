# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [2.5.0] - 2026-04-04

### Added

#### Profile 下载功能
完整的用户资料下载系统，支持批量下载和版本管理：

**下载内容：**
- `avatar.jpg/png/gif/webp` - 高清头像 (默认 400x400)
- `banner.jpg/png/gif/webp` - 个人主页横幅
- `description.txt` - 用户简介纯文本
- `profile.json` - 完整资料 JSON

**新特性：**
- **去重下载**：基于 MD5 校验，profile文件未变更时自动跳过
- **版本管理**：资料变更时自动备份到 `.versions/` 目录
- **批量下载**：支持并发下载多个用户资料
- **智能复用**：从推文下载中复用已获取的用户数据，避免重复 API 调用

**存储结构：**
```
users/{UserName(screenName)}/.loongtweet/.profile/
├── avatar.jpg           # 当前头像
├── banner.jpg           # 当前横幅
├── description.txt      # 当前简介
├── profile.json         # 当前资料
└── .versions/          # 历史版本备份
    ├── avatar_20240115_103045.jpg
    └── profile_20240115_103045.json
```

**新增模块 `internal/profile/`：**
- `downloader.go` (558 行) - Profile 下载器，支持单用户/批量下载
- `fetcher.go` (257 行) - Twitter API 获取器
- `storage.go` (183 行) - 文件存储管理器，支持版本管理
- `types.go` (158 行) - 类型定义和接口

#### 推文 JSON 保存
- 推文完整信息保存为格式化 JSON 到 `.loongtweet/` 目录
- 即使下载失败也能记录推文信息，便于调试
- 使用 `TweetFileName()` 生成一致的文件名

#### 命令行参数扩展
| 参数 | 类型 | 说明 |
|------|------|------|
| `--profile` | bool | 推文下载时同时下载用户资料（默认开启） |
| `-noprofile` | bool | 跳过 Profile 下载 |
| `-profile-user` | string | 单独指定下载 Profile 的用户（可重复） |
| `-profile-list` | uint64 | 下载指定列表所有成员的 Profile（可重复） |
| `-mark-downloaded` | bool | 仅标记用户为已下载，不下载内容 |
| `-mark-time` | string | 指定标记时间戳（格式：2006-01-02T15:04:05） |

#### Twitter 客户端增强

**代理支持改进：**
- 支持 `HTTPS_PROXY` 环境变量（优先）
- 支持 `HTTP_PROXY` 环境变量（备用）
- 自动适配 Windows/Linux/macOS

**重试机制增强：**
- 网络错误（connection reset, broken pipe, timeout）自动重试
- Twitter API 内部错误（130, 0, -1）自动重试
- HTTP 5xx 服务器错误自动重试
- HTTP 429 速率限制自动等待

**客户端选择策略：**
- `SelectProfileClient()` - Profile 下载专用客户端选择
- `SelectClientMFQ()` - MFQ（多级反馈队列）客户端选择算法
  - 优先使用备用账号（非受保护用户）
  - 受保护用户专用主账号
  - 自动跳过有限制的客户端

#### 文件工具函数
- `TweetFileName(text, tweetId, ext)` - 生成统一的推文文件名
- `CopyFile(src, dst)` - 文件复制工具
- `MaxFileNameLen` - 可配置的文件名长度限制（默认 155，范围 50-250）
- `WinFileName()` - Windows 文件名清理（移除非法字符）

#### 依赖更新
**新增依赖：**
- `github.com/tidwall/gjson v1.17.3` - JSON 快速解析（Profile 获取）
- `github.com/natefinch/lumberjack v2.0.0` - 日志文件轮转

**现有依赖更新：**
- `github.com/mattn/go-sqlite3 v1.14.22`
- `github.com/go-resty/resty/v2 v2.14.0`
- `gopkg.in/yaml.v3 v3.0.1`

### Changed

#### main.go 重构 (+340 行)
- 重新设计命令行参数结构，支持可重复参数
- 添加 Profile 下载完整流程
- 改进配置引导程序，支持保留现有配置
- 优化信号处理，支持优雅退出
- 添加 Profile 下载结果输出格式

#### `internal/twitter/client.go` 重构 (+163 行)
- 重构 `Login()` 函数，增强错误处理
- 改进速率限制器日志输出
- 添加多个客户端选择算法

#### `internal/downloading/features.go` 重构 (+485 行)
- 添加推文 JSON 保存功能
- 重构下载流程错误处理
- 优化并发下载控制

#### `internal/utils/fs.go` 扩展
- 添加 `TweetFileName()` 函数
- 添加 `CopyFile()` 函数
- `MaxFileNameLen` 改为可配置变量

#### README.md 完整重写 (+460 行)
- 重新组织文档结构，添加完整目录
- 新增功能特性详解
- 新增安装与配置指南
- 新增命令行参数详解（表格形式）
- 新增 Profile 下载功能说明
- 新增文件存储结构图示
- 新增 9 个使用场景与示例
- 新增高级设置说明
- 新增参数兼容性速查表
- 新增常见问题解答 (FAQ)
- 新增输出结果格式说明

### Fixed

- 修复文件名过长导致 Windows 保存失败的问题
- 修复代理环境变量在 Windows 上的兼容性问题
- 修复并发下载时的竞态条件
- 修复数据库连接池问题

### Stats

- **23 files changed**
- **+4,554 lines / -240 lines**

---

## [0.x.x] - Previous Versions

历史版本记录请参考 Git 提交历史:
```bash
git log --oneline
```
