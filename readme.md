# Twitter Media Downloader

跨平台的推特媒体下载器。用于轻松，快速，安全，整洁，批量的下载推特上用户的推文。支持手动指定用户或通过列表、用户关注批量下载。开箱即用！

本项目的代码基于 [unkmonster/tmd](https://github.com/unkmonster/tmd) 项目，修改了部分代码，添加了新的功能特性。新增的功能见 [CHANGELOG.md文件](CHANGELOG.md)

## 目录

- [功能特性](#功能特性)
- [安装与配置](#安装与配置)
- [命令行参数详解](#命令行参数详解)
- [Profile 下载功能](#profile-下载功能)
- [推文 JSON 保存](#推文-json-保存)
- [文件存储结构](#文件存储结构)
- [使用场景与示例](#使用场景与示例)
- [高级设置](#高级设置)
- [常见问题](#常见问题)

---

## 功能特性

### 推文下载

- 下载指定用户的媒体推文 (video, img, gif)
- 保留推文标题
- 保留推文发布日期，设置为文件的修改时间
- 以列表为单位批量下载
- 关注中的用户批量下载
- 在文件系统中保留列表/关注结构
- 同步用户/列表信息：名称，是否受保护，等...
- 记录用户曾用名

### 避免重复

- 每次工作后记录用户的最新发布时间，下次工作仅从这个时间点开始拉取用户推文
- 向列表目录发送指向用户目录的符号链接，无论多少列表包含同一用户，本地仅保存一份用户存档
- 避免重复获取时间线：任意一段时间内的推文仅仅会从 twitter 上拉取一次，即使这些推文下载失败。如果下载失败将它们存储到本地，以待重试或丢弃
- 避免重复同步用户（更新用户信息，获取时间线，下载推文）

### 其他特性

- 速率限制：避免触发 Twitter API 速率限制
- 自动关注受保护的用户
- 添加备用 cookie：提高推文获取速度和总数量
- **Profile 下载**：下载用户头像、横幅、简介等个人资料
- **推文 JSON 保存**：保存推文完整信息为 JSON/TXT 格式

---

## 安装与配置

### 下载/编译

**直接下载**

前往 [Release](https://github.com/unkmonster/tmd/releases/latest) 自行选择合适的版本并下载

**自行编译**

```bash
git clone https://github.com/unkmonster/tmd
cd tmd
go build .
```

### 首次运行

```bash
tmd -conf
```

程序会提示输入以下配置：

| 配置项 | 说明 | 示例 |
|--------|------|------|
| storage dir | 文件存储目录 | `D:\twitter_downloads` |
| auth_token | Twitter Cookie 中的 auth_token | `a1b2c3d4e5f6...` |
| ct0 | Twitter Cookie 中的 ct0 | `x1y2z3...` |
| max download routine | 最大并发下载数（0为默认值） | `20` |

### 配置文件位置

| 系统 | 路径 |
|------|------|
| Windows | `%APPDATA%\.tmd2\conf.yaml` |
| macOS/Linux | `~/.tmd2/conf.yaml` |

### 获取 Cookie

1. 登录 [Twitter/X](https://x.com)
2. 打开浏览器开发者工具 (F12)
3. 进入 Application → Cookies → x.com
4. 复制 `auth_token` 和 `ct0` 的值

> 详细获取方式请参考 [获取 Cookie](https://github.com/unkmonster/tmd/blob/master/doc/help.md#获取-cookie)

---

## 命令行参数详解

### 基础参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-conf` | bool | false | 重新配置程序，配置完成后退出 |
| `-dbg` | bool | false | 显示调试信息，包括请求计数等 |

### 推文下载参数

| 参数 | 类型 | 可重复 | 说明 |
|------|------|--------|------|
| `-user` | string | ✅ | 指定下载用户，支持用户ID或用户名（可带@前缀） |
| `-list` | uint64 | ✅ | 指定下载列表ID |
| `-foll` | string | ✅ | 指定用户，下载其关注的所有用户 |

### 下载行为参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-auto-follow` | bool | false | 自动向受保护用户发送关注请求 |
| `-no-retry` | bool | false | 快速退出，不重试失败的推文 |

### 标记参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-mark-downloaded` | bool | false | 仅标记用户为已下载，不下载内容 |
| `-mark-time` | string | 当前时间 | 指定标记时间戳，格式：`2006-01-02T15:04:05` |

### Profile 下载参数

| 参数 | 类型 | 可重复 | 说明 |
|------|------|--------|------|
| `--profile` | bool | - | 默认，与推文下载参数配合，同时下载用户资料 |
| `-noprofile` | bool | false | 跳过 Profile 下载 |
| `-profile-user` | string | ✅ | 单独指定下载 profile 的用户 |
| `-profile-list` | uint64 | ✅ | 单独指定下载 profile 的列表ID |

---

## Profile 下载功能

### 功能说明

Profile 下载功能可以保存用户的完整个人资料：

| 文件 | 说明 | 格式 |
|------|------|------|
| `avatar.jpg/png/gif/webp` | 高清头像 (400x400) | 图片 |
| `banner.jpg/png/gif/webp` | 个人主页横幅 | 图片 |
| `description.txt` | 用户简介 | 纯文本 |
| `profile.json` | 完整资料信息 | JSON |

### Profile JSON 结构

```json
{
  "ID": 123456789,
  "Name": "用户名称",
  "ScreenName": "username",
  "AvatarURL": "https://...",
  "BannerURL": "https://...",
  "URL": "https://example.com",
  "Location": "地点",
  "Verified": true,
  "Protected": false,
  "CreatedAt": "Wed Oct 01 00:00:00 +0000 2014"
}
```

### 版本管理

当资料变更时，旧文件自动备份：

```
.loongtweet/.profile/.versions/
├── avatar_20240115_103045.jpg
├── banner_20240115_103045.jpg
├── description_20240115_103045.txt
└── profile_20240115_103045.json
```

版本命名格式：`{类型}_{日期}_{时间}.{扩展名}`

---

## 推文 JSON 保存

每次下载推文媒体时，会同时保存推文的完整信息到 `.loongtweet/` 子目录。

### 保存内容

| 文件 | 格式 | 说明 |
|------|------|------|
| `{tweet_id}.json` | JSON | 推文完整信息（格式化 JSON） |
| `{tweet_id}.txt` | TXT | 人类可读的文本格式 |

### JSON 内容

- 推文文本、时间戳、URL
- 用户信息（头像已清理为高清 URL）
- 媒体信息（已清理冗余字段）
- 完整的原始数据

### 用途

- 即使下载失败也能记录推文信息，便于调试
- 可用于数据备份和迁移
- 便于第三方工具读取推文数据

### TXT 格式示例

```
time:2024-01-15T10:30:00
url:https://x.com/username/status/1234567890
media:2

这是推文的文本内容...
```

---

## 文件存储结构

```
{存储目录}/
├── users/                          # 用户目录
│   ├── Elon Musk(elonmusk)/        # 用户文件夹
│   │   ├── 2024/
│   │   │   ├── 01/
│   │   │   │   └── 推文媒体文件...
│   │   └── .loongtweet/
│   │       ├── {tweet_id}.json     # 推文 JSON
│   │       ├── {tweet_id}.txt      # 推文文本
│   │       └── .profile/           # Profile 目录
│   │           ├── avatar.jpg
│   │           ├── banner.jpg
│   │           ├── description.txt
│   │           ├── profile.json
│   │           └── .versions/      # 历史版本
│   └── NASA(NASA)/
│       └── ...
└── .data/                          # 数据目录
    ├── foo.db                      # SQLite 数据库
    └── errors.json                 # 失败推文记录
```

---

## 使用场景与示例

### 场景1：首次使用

```bash
# 1. 配置
tmd -conf

# 2. 测试下载
tmd -user elonmusk -dbg
```

### 场景2：下载单个用户

```bash
# 仅下载推文
tmd -user elonmusk -noprofile

# 下载推文 + Profile
tmd -user elonmusk

# 使用用户ID
tmd -user 44196397

# 使用 @ 前缀
tmd -user @elonmusk
```

### 场景3：批量下载多个用户

```bash
# 下载多个用户的推文
tmd -user elonmusk -user NASA -user SpaceX

# 下载多个用户的推文 + Profile
tmd -user elonmusk -user NASA -user SpaceX

# 下载多个用户的推文，不下载Profile
tmd -user elonmusk -user NASA -user SpaceX  -noprofile

# 仅下载多个用户的 Profile
tmd -profile-user elonmusk -profile-user NASA -profile-user SpaceX
```

### 场景4：下载列表

```bash
# 下载列表成员推文
tmd -list 1234567890123

# 下载列表成员推文 + Profile
tmd -list 1234567890123

# 下载列表成员推文，不下载Profile
tmd -list 1234567890123 -noprofile

# 仅下载列表成员 Profile
tmd -profile-list 1234567890123

# 多个列表
tmd -list 111111 -list 222222 --profile
```

### 场景5：下载关注列表

```bash
# 下载某用户关注的所有人
tmd -foll myusername
```

### 场景6：混合下载

```bash
# 用户 + 列表 + 关注列表
tmd -user elonmusk -list 123456 -foll myusername

# Profile 专用下载，只下载profile
tmd -profile-user elonmusk -profile-list 123456
```

### 场景7：处理受保护用户

```bash
# 自动发送关注请求
tmd -user protected_user -auto-follow
```

### 场景8：标记已下载

```bash
# 标记为当前时间
tmd -user elonmusk -mark-downloaded

# 标记为指定时间
tmd -user elonmusk -mark-downloaded -mark-time "2024-01-01T00:00:00"

# 标记为全量下载（下次会下载所有推文）
tmd -user elonmusk -mark-downloaded -mark-time "null"

# 批量标记
tmd -user a -user b -user c -mark-downloaded
```

### 场景9：调试与排错

```bash
# 调试模式
tmd -user elonmusk -dbg

# 快速退出（不重试）
tmd -user elonmusk -no-retry
```

---

## 高级设置

### 设置代理

运行前通过环境变量指定代理服务器（TUN 模式跳过这一步）

**Windows CMD:**
```bash
set HTTP_PROXY=http://127.0.0.1:7890
set HTTPS_PROXY=http://127.0.0.1:7890
tmd -user elonmusk
```

**Windows PowerShell:**
```powershell
$Env:HTTP_PROXY="http://127.0.0.1:7890"
$Env:HTTPS_PROXY="http://127.0.0.1:7890"
tmd -user elonmusk
```

**Linux/macOS:**
```bash
export HTTP_PROXY=http://127.0.0.1:7890
export HTTPS_PROXY=http://127.0.0.1:7890
tmd -user elonmusk
```

### 忽略用户

程序默认会忽略被静音或被屏蔽的用户，所以当你想要下载的列表中包含你不想包含的用户，可以在推特将他们屏蔽或静音。

### 添加额外 Cookie

程序动态从所有可用 cookie 中选择一个不会被速率限制的 cookie 请求用户推文，以避免因单一 cookie 的速率限制导致程序被阻塞。

按如下格式创建 `$HOME/.tmd2/additional_cookies.yaml` 或 `%appdata%/.tmd2/additional_cookies.yaml`：

```yaml
- auth_token: xxxxxxxxx1
  ct0: xxxxxxxxxxxxxxxxxxxxxxx
- auth_token: xxxxxxxxx2
  ct0: xxxxxxxxxxxxxxxx2
- auth_token: xxxxxxxxxxxxxxxx3
  ct0: xxxxxxxxxxxxxxxxxxxxx3
```

> 这些添加的备用 cookie，仅用来提升获取推文的速率和总量。判断是否忽略用户和自动关注受保护的用户依然使用主账号。

### 关于速率限制

Twitter API 限制一段时间内过快的请求（例如某端点每15分钟仅允许请求500次，超出这个次数会以429响应）。当某一端点将要达到速率限制程序会打印一条通知并阻塞尝试请求这个端点的协程直到余量刷新（这最多是15分钟），但并不会阻塞所有协程，所以其余协程打印的消息可能将这条休眠通知覆盖让人认为程序无响应了，等待余量刷新程序会继续工作。

---

## 参数兼容性速查表

| 组合 | 兼容 | 说明 |
|------|:----:|------|
| `-user` + `-list` + `-foll` | ✅ | 多种来源可叠加 |
| `-user` + `--profile` | ✅ | 下载推文和资料 |
| `-list` + `--profile` | ✅ | 下载列表成员推文和资料 |
| `-foll` + `--profile` | ✅ | 下载关注用户推文和资料 |
| `-profile-user` + `-profile-list` | ✅ | 仅下载资料 |
| `-user` + `-profile-user` | ✅ | 推文下载 + 额外用户资料 |
| `-dbg` + 任意参数 | ✅ | 启用调试输出 |
| `-auto-follow` + 推文下载 | ✅ | 自动关注受保护用户 |
| `-no-retry` + 推文下载 | ✅ | 失败不重试 |
| `-mark-downloaded` + `-mark-time` | ✅ | 指定标记时间 |
| `-mark-downloaded` + 推文下载 | ⚠️ | 只标记，不下载 |
| `-conf` + 其他参数 | ⚠️ | 配置后退出，忽略其他 |
| `-noprofile` + `--profile` | ⚠️ | `-noprofile` 优先，跳过 Profile |

---

## 常见问题

### Q: 如何查看失败的下载？

失败的任务保存在 `{存储目录}/.data/errors.json`，下次运行会自动重试。

### Q: Profile 文件存在时还会重新下载吗？

如果文件内容未变更（MD5校验），会自动跳过。

### Q: 如何更新已下载用户的 Profile？

重新运行相同的命令即可，只会下载变更的文件。

### Q: 下载中断后怎么办？

直接重新运行相同命令，程序会自动恢复。

### Q: `-mark-downloaded` 的用途？

用于标记用户为"已下载到最后"，下次运行时不会下载历史推文，只下载新推文。

### Q: 如何获取列表ID？

在 Twitter 网页版打开列表，URL 格式为：
```
https://x.com/i/lists/1234567890123
```
其中数字就是列表ID。

### Q: 不知道啥是 user_id/list_id/screen_name?

请参考 [获取 list_id, user_id, screen_name](https://github.com/unkmonster/tmd/blob/master/doc/help.md#获取-list_id-user_id-screen_name)

### Q: Windows 上需要管理员权限吗？

为了创建符号链接，在 Windows 上应该以管理员身份运行程序。

### Q: 推文 JSON 文件有什么用？

即使媒体下载失败，推文信息也会保存到 `.loongtweet/` 目录。JSON 文件包含完整的推文数据，可用于数据分析或备份。

---

## 输出结果格式

### 推文下载结果

```
users: 3
    - Elon Musk(elonmusk)
    - NASA(NASA)
    - SpaceX(SpaceX)
```

### Profile 下载结果

```
=== PROFILE_DOWNLOAD_RESULTS ===
SCREEN_NAME:elonmusk|STATUS:OK
SCREEN_NAME:NASA|STATUS:OK
SCREEN_NAME:SpaceX|STATUS:SKIP
SCREEN_NAME:test|STATUS:FAIL
=== END_RESULTS ===
```

状态说明：
- `OK` - 下载成功
- `SKIP` - 跳过（文件未变更）
- `FAIL` - 下载失败

### 标记结果

```
=== MARK_DOWNLOADED_RESULTS ===
ENTITY_ID:1|USER_ID:44196397|SCREEN_NAME:elonmusk|STATUS:OK
ENTITY_ID:2|USER_ID:23248887|SCREEN_NAME:NASA|STATUS:OK
=== END_RESULTS ===
```

---

## 参数类型总结

### 布尔型参数（开关型，无需值）

| 参数 | 说明 |
|------|------|
| `-conf` | 重新配置 |
| `-dbg` | 调试模式 |
| `-auto-follow` | 自动关注受保护用户 |
| `-no-retry` | 不重试失败推文 |
| `-mark-downloaded` | 仅标记已下载 |
| `--profile` | 下载用户资料 |
| `-noprofile` | 跳过 Profile 下载 |

### 可重复参数（可多次使用）

| 参数 | 说明 |
|------|------|
| `-user` | 用户名/ID |
| `-list` | 列表ID |
| `-foll` | 用户名/ID |
| `-profile-user` | 用户名/ID |
| `-profile-list` | 列表ID |

### 字符串参数

| 参数 | 说明 |
|------|------|
| `-mark-time` | 时间戳（2006-01-02T15:04:05 或 "null"） |
