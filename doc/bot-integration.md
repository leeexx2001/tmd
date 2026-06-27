# TMD Bot 集成文档

## 概述

TMD Server 模式支持接入多个 Bot 平台，提供下载控制、任务通知和错误告警能力。目前支持 6 个平台：

| 平台 | 类型 | 命令交互 | 通知推送 |
|---|---|---|---|
| Telegram | 双向命令 + 通知 | ✅ | ✅ 任务完成 + 错误日志 |
| Discord | 双向命令 + 通知 | ✅ | ✅ 任务完成 + 错误日志 |
| WeChat iLink | 双向命令 + 通知 | ✅ | ✅ 任务完成 + 错误日志 |
| 飞书 / Lark | 双向命令 + 通知 | ✅ | ✅ 任务完成 + 错误日志 |
| Gotify | 单向推送 | ❌ | ✅ 任务完成 + 错误日志 |
| Pushover | 单向推送 | ❌ | ✅ 任务完成 + 错误日志 |

---

## 后端架构

### 分层设计

```
Bot Platform (Telegram/Discord/WeChat/Feishu/etc.)
        │
        ▼
  internal/bot/{platform}/
    └── Bot impl (Bot 接口实现)
          ├── Start() / Stop()           ← 生命周期管理
          ├── handle* (消息接收)          ← 命令解析和路由
          ├── cmd* (命令处理)             ← /dl /status /cancel 等
          └── handleEvents / handleLogs  ← 事件订阅
                │
                ▼
  internal/api/
    ├── TaskManager     ← 创建/查询/取消任务
    ├── EventBus        ← 订阅任务事件
    └── DownloadQueue   ← 异步执行下载
                │
                ▼
  internal/service/DownloadService  ← 下载业务编排
```

Bot 只调用 `api.TaskManager`、`api.EventBus`、`consolelog.Hub`，不直接接触下载逻辑。

### Bot 接口

```go
type Bot interface {
    Start() error   // 非阻塞启动
    Stop()          // 停止
    Name() string   // 名称（如 "telegram"）
}
```

所有平台实现此接口，由 `BotManager` 统一管理生命周期。

### BotManager

```go
bm := bot.NewBotManager(bot1, bot2, ...)
bm.Start()  // 依次启动所有 bot
bm.Stop()   // 依次停止所有 bot
```

通过 `server.InitBot(bm)` 注入到 Server。Server 在 `Start()` 中调用 `bm.Start()`，在 `GracefulShutdown()` 中调用 `bm.Stop()`。

### 消息流程

**接收消息 → 处理命令**（Telegram 示例）：

```
用户发送 /dl elonmusk
  → Bot 接收消息
  → handleCommand("/dl elonmusk")
  → parseDLArgs → (type="user", target="elonmusk")
  → taskManager.CreateTask(TaskTypeUserDownload, &UserDownloadTaskData{...})
  → 返回 task_id 给用户
  → DownloadQueue 异步执行下载
```

**通知推送**：

```
任务完成/失败
  → TaskManager 更新状态
  → EventBus.Publish("tasks", tasks)
  → Bot.handleEvents() 收到事件
  → notifyTaskChanges() 发送消息给用户
```

---

## 配置方式

所有 Bot 配置在 `conf.yaml` 的 `bot:` 下，未配置的平台不会启动。

### Telegram

```yaml
bot:
  telegram:
    token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
    allowed_users: [123456789, 987654321]
```

| 参数 | 说明 | 获取方式 |
|---|---|---|
| `token` | Bot token | [@BotFather](https://t.me/BotFather) 创建 Bot 后获取 |
| `allowed_users` | 允许使用的用户 ID（数字） | 向 Bot 发消息 → `https://api.telegram.org/bot<token>/getUpdates` → `message.from.id` |

**可用命令**：`/dl [user\|list\|foll] <target>`、`/status <id>`、`/cancel <id>`、`/tasks`、`/help`

### Discord

```yaml
bot:
  discord:
    token: "MTE5ODk4MjQ2NzE4NTMyMTI5OQ.GnO2X.xxx"
    allowed_users: ["123456789012345678"]
```

| 参数 | 说明 | 获取方式 |
|---|---|---|
| `token` | Bot token | [Discord Developer Portal](https://discord.com/developers/applications) → Application → Bot → Reset Token |
| `allowed_users` | 允许使用的用户 ID（字符串） | Discord 设置 → 高级 → 开发者模式 → 右键用户 → Copy ID |

**可用命令**：`/dl [type:user\|list\|foll] <target>`、`/status <id>`、`/cancel <id>`、`/tasks`、`/help`

### WeChat iLink

```yaml
bot:
  wechat:
    credential_path: ".weixin-token.json"
    allowed_users: ["friend@im.wechat"]
```

| 参数 | 说明 | 获取方式 |
|---|---|---|
| `credential_path` | 凭证文件路径（首次登录后自动生成） | 任意可写路径，相对于工作目录 |
| `allowed_users` | 允许使用的联系人 ID | 启动后向 Bot 发消息，查看服务端日志中的 `FromUserID` |

**首次使用**：启动后查看服务端日志中的 QR Code URL，用微信扫码登录。后续自动复用凭证。

**可用命令**：`/dl [user\|list\|foll] <target>`、`/status <id>`、`/cancel <id>`、`/tasks`、`/help`

### 飞书 / Lark

```yaml
bot:
  feishu:
    app_id: "cli_xxxxxxxxxxxx"
    app_secret: "xxxxxxxxxxxxxxxxxxxxxxxxxx"
    verify_token: "xxxxxxxxxxxx"
    encrypt_key: ""                 # 可选，不配置则不加密
    allowed_users: ["ou_xxxxxxxxxxxxx"]
    callback_path: "/api/v1/bot/feishu/callback"   # 可选，默认值
```

| 参数 | 说明 | 获取方式 |
|---|---|---|
| `app_id` | 应用 App ID | [飞书开发者后台](https://open.feishu.cn/app) → 凭证与基础信息 |
| `app_secret` | 应用 App Secret | 同上 |
| `verify_token` | Verification Token | 事件与回调 → Verification Token |
| `encrypt_key` | Encrypt Key（可选） | 事件与回调 → Encrypt Key |
| `allowed_users` | 允许使用的用户 open_id | 通过[获取用户 open_id API](https://open.feishu.cn/document/server-docs/contact-v3/user/get) 查询 |
| `callback_path` | 回调路径（可选） | TMD 服务端路由，需在开发者后台配置相同地址 |

**飞书开发者后台额外配置**：

1. 创建企业自建应用 → 添加 **机器人** 能力
2. 权限管理 → 开启 `获取用户发给机器人的单聊消息` 权限
3. 事件订阅 → 添加 `接收消息 v2.0` 事件
4. 事件订阅 → 回调地址填写 `https://你的域名/api/v1/bot/feishu/callback`
5. 版本管理与发布 → 创建版本 → 审核发布

**可用命令**：`/dl [user\|list\|foll] <target>`、`/status <id>`、`/cancel <id>`、`/tasks`、`/help`

### Gotify（单向推送）

```yaml
bot:
  gotify:
    server_url: "http://gotify.lan:8080"
    token: "S3cr3tT0k3n"
    priority: 5
```

| 参数 | 说明 | 获取方式 |
|---|---|---|
| `server_url` | Gotify 服务器地址 | 自行部署的 Gotify 服务端地址 |
| `token` | 应用 Token | Gotify Web UI → Apps → Create Application |
| `priority` | 通知优先级（可选，默认5） | 5=normal, 8=emergency |

**触发场景**：任务完成/失败时推送标题和摘要；错误日志（error/fatal 级别）推送完整日志行。

### Pushover（单向推送）

```yaml
bot:
  pushover:
    user: "uKey123..."
    token: "appToken456..."
    device: "iphone"          # 可选
    sound: "gamelan"          # 可选
```

| 参数 | 说明 | 获取方式 |
|---|---|---|
   | `user` | 用户 Key | [pushover.net](https://pushover.net) 登录后首页显示 |
| `token` | 应用 API Token | pushover.net → Create an Application/API Token |
| `device` | 指定设备名（可选） | Pushover App 中设置的设备名称 |
| `sound` | 通知声音（可选） | [可选值列表](https://pushover.net/api#sounds) |

**触发场景**：同 Gotify。

---

## 多平台同时使用

可以同时启用多个平台，互不干扰：

```yaml
bot:
  telegram:
    token: "..."
    allowed_users: [123456789]
  discord:
    token: "..."
    allowed_users: ["123456789012345678"]
  gotify:
    server_url: "http://gotify.lan:8080"
    token: "..."
```

所有开启了通知的平台都会收到任务完成通知。如果需要区分不同平台的通知内容，可以配置不同的参数（如 Pushover 的 `device` 和 `sound`）。

---

## 实现说明

### 包结构

```
internal/bot/
├── bot.go              # Bot 接口定义
├── manager.go          # BotManager 生命周期管理
├── manager_test.go
├── telegram/           # Telegram 实现
│   ├── bot.go          #   核心结构、Start/Stop
│   ├── handlers.go     #   消息路由、权限检查
│   ├── commands.go     #   命令实现
│   ├── notify.go       #   事件/日志订阅
│   └── bot_test.go
├── discord/            # Discord 实现（同上结构）
├── wechat/             # 微信 iLink 实现
├── feishu/             # 飞书/Lark 实现
├── gotify/             # Gotify 推送实现
└── pushover/           # Pushover 推送实现
```

### 双向 Bot vs 单向推送

**双向 Bot**（Telegram、Discord、WeChat、Feishu）：
- 接收用户消息 → 解析命令 → 创建/查询/取消任务 → 回复结果
- 订阅 EventBus → 任务完成后主动推送通知
- 依赖：`TaskManager` + `EventBus` + `LogHub`

**单向推送**（Gotify、Pushover）：
- 仅订阅 EventBus + LogHub → HTTP POST 推送
- 不处理用户命令
- 依赖：`EventBus` + `LogHub`

### 通信方式差异

| 平台 | 消息接收 | 命令模型 | Bot 身份 |
|---|---|---|---|
| Telegram | 长轮询 (`getUpdates`) | 文本命令 `/cmd` | 独立 Bot 账号 + Token |
| Discord | WebSocket Gateway | Slash Command 结构化 | 独立 Bot 账号 + Token |
| WeChat iLink | 长轮询 | 文本命令 `/cmd` | 个人微信号（扫码登录） |
| 飞书/Lark | HTTP Webhook 回调 | 文本命令 `/cmd` | 企业自建应用 + AppID/Secret |
| Gotify | — | — | 应用 Token |
| Pushover | — | — | 应用 Token |

### 外部依赖

| 平台 | 依赖库 |
|---|---|
| Telegram | `github.com/go-telegram-bot-api/telegram-bot-api/v5` |
| Discord | `github.com/bwmarrin/discordgo` |
| WeChat iLink | `github.com/SpellingDragon/wechat-robot-go` |
| 飞书/Lark | `github.com/chyroc/lark` |
| Gotify | 无（标准库 `net/http`） |
| Pushover | 无（标准库 `net/http`） |

---

## 扩展新平台

添加新的 Bot 平台只需三步：

1. **创建实现包** `internal/bot/{name}/`，实现 `Bot` 接口
2. **添加配置** 在 `internal/config/config.go` 的 `BotConfig` 下加结构体，在 `NormalizeLoadedConf` 中加 trim
3. **注册到工厂** 在 `main.go` 的 `initBot()` 中加条件分支

如果平台使用 HTTP Webhook 回调（如飞书），还需要调用 `server.RegisterBotCallback(path, handler)` 注册路由。
