# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [2.9.0] - 2026-04-15

### Added

#### 新增 `internal/config/` 包 - 配置管理模块

将配置逻辑从 `main.go` 中抽离，独立管理：

| 文件 | 功能 |
|------|------|
| `config.go` | 配置结构定义、读写、交互式引导 |

**核心功能：**
- `ReadConf()` / `WriteConf()` - YAML 配置文件读写
- `PromptConfig()` - 交互式配置引导（支持默认值、自动备份）
- `ReadAdditionalCookies()` - 多账号 Cookie 读取
- 配置损坏时自动备份并重新创建

#### 数据库层拆分 (`internal/database/`)

将原来的单体 `crud.go`（496行）按职责拆分为多个文件：

| 文件 | 功能 |
|------|------|
| `schema.go` | 数据库表结构与迁移 |
| `helpers.go` | 通用数据库辅助函数（handleGetResult, handleInsertWithId） |
| `user.go` | 用户表 CRUD 操作 + 用户可访问状态管理 |
| `user_entity.go` | 用户实体操作（CRUD + 最新发布时间） |
| `lst.go` / `lst_entity.go` | 列表及列表实体操作 |
| `user_link.go` | 用户符号链接（symlink）管理 |
| `user_sync.go` | 共享用户同步逻辑（`SyncUser()`） |
| `user_sync_test.go` | 同步功能测试（5 个用例） |

#### 下载模块拆分 (`internal/downloading/`)

将原来的单体 `features.go`（1226行）按职责拆分为多个文件：

| 文件 | 功能 |
|------|------|
| `types.go` | 类型定义（PackagedTweet, TweetInEntity, workerConfig 等）+ 全局状态 |
| `tweet_download.go` | 单条推文下载、JSON/LoongTweet 保存、媒体清理 |
| `user_sync.go` | 下载过程中的用户同步（syncUser, syncUserAndEntity, shouldIgnoreUser） |
| `user_download.go` | 单用户推文获取与预处理 |
| `batch_download.go` | 批量用户下载核心逻辑（优先级队列、并发控制、ants 池） |
| `list_download.go` | 列表下载流程（syncList, syncListAndGetMembers） |
| `batch_any.go` | 通用批量下载入口 |
| `mark_downloaded.go` | 标记用户为已下载（支持时间戳、全量重置、JSON 结果输出） |
| `retry.go` | 失败推文重试机制（RetryFailedTweets） |

**批量下载架构：**
```
BatchDownloadAny()
    ├── syncLstAndGetMembers() - 同步列表成员
    ├── BatchUserDownload()
    │   ├── 预处理阶段：用户排序、symlink 创建、深度计算
    │   ├── 生产者池（ants goroutine pool）：并发获取用户推文
    │   ├── 消费者池（MaxDownloadRoutine）：并发下载媒体
    │   └── 错误收集与重试
    └── MarkUsersAsDownloaded() - 标记已下载
```

#### 新增 `internal/twitter/batch_login.go` - 批量登录

多账号并发登录：

```go
func BatchLogin(ctx context.Context, dbg bool, cookies []AccountCookie, master string) []*resty.Client
```

**特性：**
- 并发登录所有账号
- 自动去重（相同 screen_name 只保留一个）
- 主账号优先保证
- 支持调试模式（请求计数）

#### Profile 模块 API 去重

| 文件 | 变化 |
|------|------|
| `internal/profile/fetcher.go` | 精简 ~145 行，删除重复的 `userByScreenName` API 定义和 `makeProfileUrl()`，改用 `twitter.GetUserByScreenName()` + `userToProfileInfo()` 转换；新增 `errors.As` 类型解包修复客户端错误处理 |
| `internal/profile/fetcher_test.go` | 新增测试用例（userToProfileInfo 转换 + GetHighResAvatarURL 各质量参数） |

### Changed

#### 架构重构 - 单体文件拆分与去重

**删除的文件：**
- `internal/database/crud.go` → 拆分为 8 个职责单一的文件
- `internal/downloading/features.go` → 拆分为 9 个职责单一的文件

**代码统计：**
- 删除 ~1722 行（两个巨型单体文件）
- 新增 ~18 个模块化文件
- `main.go` 精简，配置/登录/重试逻辑分别迁移到 config/twitter/downloading 包

#### 命名规范化

全局修正 5 处拼写/命名不一致（编译器级安全替换）：

| 旧名称 | 新名称 | 影响范围 |
|--------|--------|----------|
| `GetMeidas` | `GetMedias` | twitter/user.go, downloading/*, twitter_test.go |
| `PackgedTweet` | `PackagedTweet` | downloading/*, main.go, json_download.go |
| `JsonPackgedTweet` | `JsonPackagedTweet` | json_download.go |
| `shouldIngoreUser` | `shouldIgnoreUser` | downloading/user_sync.go, batch_download.go |
| `userInLstEntity` | `userInListEntity` | downloading/types.go, batch_download.go, list_download.go, batch_any.go |
| `syncLstAndGetMembers` | `syncListAndGetMembers` | list_download.go, batch_any.go |

#### 逻辑统一

| 变化 | 说明 |
|------|------|
| `database.SyncUser()` | 提取共享用户同步逻辑，downloading 和 profile 包复用同一函数 |
| `profile/downloader.syncUserDirectory()` | 简化为调用 `database.SyncUser()`，消除 ~35 行重复代码 |
| `downloading/user_sync.syncUser()` | 简化为调用 `database.SyncUser()` 单行委托 |

#### 错误处理改进

| 文件 | 变化 |
|------|------|
| `profile/fetcher.go` | `handleClientError` 从类型断言改为 `errors.As`，修复因 `GetUserByScreenName` 包装 error 导致客户端限制错误无法被正确识别的问题 |
| `downloading/batch_download.go` | TwitterApiError 类型断言同样改为 `errors.As`，防御性编程 |
| `database/user_sync.go` | `SyncUser` 中 `RecordUserPreviousName` 错误现在向上传播而非仅记录日志，保持与原始行为一致 |
| `downloading/types.go` | `TweetInEntity.GetPath()` 移除不必要的裸 `recover()`，改为直接返回空字符串 |
| `downloading/user_download.go` | 修正 "skiped" → "skipped" 拼写 |

#### 其他修改

| 文件 | 变化 |
|------|------|
| `main.go` | 使用 config 包；使用 twitter.BatchLogin；使用 downloading.RetryFailedTweets |
| `internal/twitter/user.go` | `GetMeidas` → `GetMedias` |
| `internal/twitter/twitter_test.go` | 测试用例适配新方法名 |
| `go.mod` / `go.sum` | 依赖更新（testify） |

### Fixed

- 修复 `profile/fetcher.go` 中 `handleClientError` 因 error 包装导致类型断言永远为 false 的 bug
- 修复 `downloading/batch_download.go` 中同类型的 TwitterApiError 类型断言隐患
- 修复 `database/SyncUser` 吞掉 `RecordUserPreviousName` 错误导致调用方丢失重命名历史失败信息的问题
- 修复 `TweetInEntity.GetPath()` 中不必要的裸 `recover()` 调用
- 修正 `shouldIngoreUser` / `PackgedTweet` / `GetMeidas` / `userInLstEntity` / `syncLstAndGetMembers` 共 6 处拼写/命名错误

---

## [2.8.0] - 2026-04-12

### Added

#### 新增用户可访问状态记录功能（扩展）

新增批量标记用户为可访问的方法：

| 文件 | 变更 |
|------|------|
| `internal/database/crud.go` | 新增 `SetUsersAccessible()` 和 `MarkListMembersAccessibleByIDs()` |
| `internal/utils/user.go` | 新增通用 ID 提取函数 `ExtractIDs()` |

**核心功能：**
- `SetUsersAccessible()` - 批量标记用户为可访问状态
- `MarkListMembersAccessibleByIDs()` - 异步标记列表成员为可访问
- `ExtractIDs()` - 通用 ID 提取函数，使用泛型

**调用位置：**
- `internal/downloading/features.go`: `downloadList()`, `syncLstAndGetMembers()`
- `main.go`: `handleProfileDownload()`

#### 新增用户可访问状态记录功能

在 `users` 表中新增 `is_accessible` 字段，用于记录 Twitter 用户是否可通过 API 正常访问（非封禁/注销状态）：

| 文件 | 变更 |
|------|------|
| `internal/database/model.go` | 新增 `IsAccessible` 字段 |
| `internal/database/crud.go` | 新增 `UpdateUserAccessible()` 方法 |
| `internal/database/db_test.go` | 新增测试用例 |

**核心功能：**
- 区分可访问/不可访问用户：识别 Twitter API 返回的 `UserUnavailable` 类型
- 自动更新：每次获取列表成员时同步更新数据库中的访问状态
- 向后兼容：对已有 `foo.db` 数据库无破坏性影响

**调用链路：**
```
main.go (handleProfileDownload)
    └── lst.GetMembers()
        └── downloading/features.go
            ├── downloadList()
            ├── syncLstAndGetMembers()
            └── MarkUsersAsDownloaded()
```

#### 新增文档

- `doc/user-accessible-status-changelog.md` - 用户可访问状态记录功能说明

### Changed

#### 数据库层 (`internal/database/`)：
- `crud.go` - 新增用户可访问状态更新方法，扩展错误处理

#### 下载层 (`internal/downloading/`)：
- `features.go` - 集成用户可访问状态检测逻辑

#### Twitter API 层 (`internal/twitter/`)：
- `list.go` - 列表成员获取逻辑优化
- `tweet.go` - 推文处理优化
- `user.go` - 用户数据处理优化
- `twitter_test.go` - 测试用例更新

#### 主程序 (`main.go`)：
- 优化配置和错误处理
- 集成用户可访问状态功能

---

## [2.7.0] - 2026-04-12

### Added

#### 新增 `internal/entity/` 包 - 实体类型定义

将分散在各处的实体类型集中管理：

| 文件 | 功能 |
|------|------|
| `interface.go` | 实体接口定义 |
| `list.go` | 列表相关实体 |
| `sync.go` | 同步相关实体 |
| `user.go` | 用户相关实体 |

#### 新增文档

- `doc/用户名变更处理机制.md` - 用户名变更处理机制说明

### Changed

#### 代码重构与优化

**数据库层 (`internal/database/`)：**
- `crud.go` - 重构 CRUD 操作，优化错误处理
- `db_test.go` - 补充测试用例
- `model.go` - 模型定义优化

**下载层 (`internal/downloading/`)：**
- `dumper.go` - 优化文件转储逻辑
- `entity.go` - 移除冗余代码（-256行）
- `features.go` - 重构下载特性
- `json_download.go` - JSON下载优化

**命名服务 (`internal/naming/`)：**
- `naming.go` - 优化命名逻辑
- `naming_test.go` - 测试用例更新

**Twitter API层 (`internal/twitter/`)：**
- `list.go` - 列表功能优化
- `timeline.go` - 时间线处理优化
- `tweet.go` - 推文处理优化
- `user.go` - 用户数据处理优化

**其他：**
- `main.go` - 主程序优化
- `internal/profile/downloader.go` - 下载器优化
- `internal/profile/storage.go` - 存储层优化

### Removed

- `internal/downloading/entity.go` - 实体类型迁移到 `internal/entity/` 包

---

## [2.6.0] - 2026-04-12

### Added

#### 新增 `internal/downloader/` 包 - 通用下载基础设施

将下载逻辑从业务代码中抽离，提供可复用的下载能力：

| 文件 | 行数 | 功能 |
|------|------|------|
| `types.go` | 75 | 接口定义（Downloader, FileWriter, VersionManager） |
| `downloader.go` | 118 | HTTP下载实现，支持批量下载和上下文取消 |
| `file_writer.go` | 145 | 原子写入、MD5去重、版本管理 |
| `version_manager.go` | 95 | 文件版本备份管理 |

**特性：**
- **原子写入**：先写临时文件，再重命名，确保数据完整性
- **MD5 去重**：相同内容自动跳过写入
- **并发安全**：使用 `sync.Mutex` 保护并发写入
- **版本管理**：文件变更时自动备份历史版本

#### 新增 `internal/naming/` 包 - 统一命名服务

集中管理推文和用户的文件命名逻辑：

| 类型 | 功能 |
|------|------|
| `TweetNaming` | 推文文件名生成，支持日志格式、文件名、文件路径 |
| `UserNaming` | 用户目录命名，生成 `Name(ScreenName)` 格式 |
| `SetMaxFileNameLen()` | 统一配置文件名长度限制 |

**特性：**
- 缓存清理后的文本，避免重复计算
- 日志格式与文件名前缀一致
- 单一配置入口，无需手动同步

#### 新增 `internal/utils/recovery.go` - Panic 恢复工具

统一的 panic 恢复机制：

```go
defer utils.RecoverWithLog("functionName")
```

#### 新增 `internal/downloading/json_download.go` - JSON 下载功能

支持从 JSON 文件批量下载推文媒体：

| 函数 | 功能 |
|------|------|
| `BatchDownloadFromJson()` | 从 JSON 批量下载 |
| `DownloadJsonDir()` | 下载目录下所有 JSON 文件 |

### Changed

#### 架构重构

**依赖注入模式：**
- `downloader.Downloader` 接口注入到业务层
- `main.go` 统一创建和注入依赖
- 支持测试时 Mock

**分层架构：**
```
main.go (应用层)
    └── downloading/profile (业务层)
            └── downloader (基础设施层)
                    └── file_writer, version_manager
```

#### `internal/downloading/features.go` 重构

| 变化 | 说明 |
|------|------|
| `downloadTweetMedia()` | 使用 `downloader.Downloader` 接口 |
| `BatchDownloadTweet()` | 新增 `dwn` 参数 |
| `saveLoongTweet()` | 统一数据来源（RawJSON 优先） |
| `saveTweetJson()` | 使用 `naming.TweetNaming` |

#### `internal/profile/downloader.go` 重构

| 变化 | 说明 |
|------|------|
| 构造函数 | 新增 `dwn` 和 `fw` 参数 |
| `downloadAvatar()` | 使用 `downloader.Downloader` |
| `downloadBanner()` | 使用 `downloader.Downloader` |
| `saveDescription()` | 使用 `downloader.FileWriter` |
| `ensureProfileDirs()` | 提取目录创建逻辑 |

#### `internal/utils/fs.go` 修改

| 变化 | 说明 |
|------|------|
| 移除 `TweetFileName()` | 使用 `naming.TweetNaming` 替代 |
| 移除 `MaxFileNameLen` 变量 | 使用 `naming.SetMaxFileNameLen()` |
| 新增 `WinFileNameWithMaxLen()` | 支持自定义长度限制 |

#### `internal/profile/storage.go` 简化

| 变化 | 说明 |
|------|------|
| `EnsureDirectory()` | 移除 `screenName` 参数 |
| `GetFilePath()` | 移除 `screenName` 参数 |

### Fixed

- 修复 `saveLoongTweet` 中 `tweet.Creator` 为 nil 时的 panic
- 修复 `MaxFileNameLen` 双变量同步问题
- 修复循环依赖风险（naming 包不再直接依赖 utils 变量）

### Stats

- **新增文件**: 6 个
- **修改文件**: 8 个
- **+1,200 lines / -300 lines**

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
