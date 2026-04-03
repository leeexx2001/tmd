# `-mark-downloaded` 功能详解

> 本文档详细说明 Twitter Media Downloader (tmd) 中 `-mark-downloaded` 参数的实现原理、使用方法和内部机制。

---

## 目录

1. [功能概述](#功能概述)
2. [参数说明](#参数说明)
3. [核心原理](#核心原理)
4. [使用方法](#使用方法)
5. [实现细节](#实现细节)
6. [数据库结构](#数据库结构)
7. [时间过滤机制](#时间过滤机制)
8. [新用户处理流程](#新用户处理流程)
9. [错误处理](#错误处理)
10. [输出格式](#输出格式)
11. [常见场景](#常见场景)
12. [注意事项](#注意事项)

---

## 功能概述

`-mark-downloaded` 是一个标记功能，用于**在不下载任何内容的情况下**，更新数据库中用户的 `latest_release_time` 时间戳。这个时间戳决定了下次下载时从哪个时间点开始获取推文。

### 核心用途

| 用途 | 说明 |
|------|------|
| **跳过历史** | 标记为当前时间，下次只下载新推文 |
| **指定起点** | 标记为指定时间，从该时间点开始下载 |
| **重置记录** | 设置为 NULL，允许全量重新下载 |

---

## 参数说明

### 命令行参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-mark-downloaded` | bool | false | 启用标记模式，不下载内容 |
| `-mark-time` | string | "" | 时间戳值 |

### `-mark-time` 支持的值

| 值 | 效果 | 数据库操作 |
|----|------|-----------|
| **空（不提供）** | 使用当前时间 | `UPDATE ... SET latest_release_time=NOW()` |
| `"2024-01-01T00:00:00"` | 使用指定时间 | `UPDATE ... SET latest_release_time='2024-01-01 00:00:00'` |
| `"null"` 或 `"nil"` | 设置为 NULL | `UPDATE ... SET latest_release_time=NULL` |

### 时间格式

- 格式：`2006-01-02T15:04:05`
- 示例：`2024-06-15T10:30:00`
- 时区：使用本地时区解析

---

## 核心原理

### 工作流程图

```
┌─────────────────────────────────────────────────────────────┐
│                    tmd -user xxx -mark-downloaded           │
│                         [-mark-time "xxx"]                  │
└─────────────────────────────────────────────────────────────┘
                              ↓
                    ┌─────────────────┐
                    │  解析 markTimeStr │
                    └─────────────────┘
                              ↓
        ┌─────────────────────┼─────────────────────┐
        ↓                     ↓                     ↓
     空字符串            "null"/"nil"           指定时间
        ↓                     ↓                     ↓
    当前时间            timestamp=nil          解析时间
        ↓                     ↓                     ↓
        └─────────────────────┼─────────────────────┘
                              ↓
                    ┌─────────────────┐
                    │ 遍历 lists/users │
                    └─────────────────┘
                              ↓
                    ┌─────────────────┐
                    │syncUserAndEntity│
                    └─────────────────┘
                              ↓
        ┌─────────────────────┼─────────────────────┐
        ↓                     ↓                     ↓
   syncUser()          NewUserEntity()        syncPath()
  (更新users表)        (定位/创建实体)        (创建文件夹)
        └─────────────────────┼─────────────────────┘
                              ↓
                    ┌─────────────────┐
                    │ 设置时间戳       │
                    │ Set/Clear       │
                    │ LatestReleaseTime│
                    └─────────────────┘
                              ↓
                    ┌─────────────────┐
                    │ 输出结果         │
                    └─────────────────┘
```

### 与正常下载的关系

```
正常下载流程:
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ 获取推文  │ → │ 更新时间戳 │ → │ 下载媒体  │ → │ 保存文件  │
└──────────┘    └──────────┘    └──────────┘    └──────────┘

-mark-downloaded 流程:
┌──────────┐    ┌──────────┐
│ 同步用户  │ → │ 更新时间戳 │  (跳过获取推文和下载)
└──────────┘    └──────────┘
```

---

## 使用方法

### 基础用法

```bash
# 标记单个用户为当前时间
tmd -user elonmusk -mark-downloaded

# 标记多个用户
tmd -user user1 -user user2 -user user3 -mark-downloaded

# 使用用户ID
tmd -user 44196397 -mark-downloaded

# 使用 @ 前缀
tmd -user @elonmusk -mark-downloaded
```

### 指定时间

```bash
# 从指定时间点开始下载
tmd -user elonmusk -mark-downloaded -mark-time "2024-06-01T00:00:00"

# 标记为一年前
tmd -user elonmusk -mark-downloaded -mark-time "2023-01-01T00:00:00"
```

### 重置为全量下载

```bash
# 清除时间记录，允许全量下载
tmd -user elonmusk -mark-downloaded -mark-time "null"

# "nil" 效果相同
tmd -user elonmusk -mark-downloaded -mark-time "nil"
```

### 批量操作

```bash
# 标记整个列表
tmd -list 123456789 -mark-downloaded

# 标记关注列表
tmd -foll myusername -mark-downloaded

# 混合操作
tmd -user user1 -list 123456 -foll myusername -mark-downloaded
```

### 不同终端的引号处理

```powershell
# PowerShell - null 需要引号
tmd -user elonmusk -mark-downloaded -mark-time "null"
tmd -user elonmusk -mark-downloaded -mark-time "2024-01-01T00:00:00"

# CMD - 引号可选
tmd -user elonmusk -mark-downloaded -mark-time null
tmd -user elonmusk -mark-downloaded -mark-time 2024-01-01T00:00:00
```

---

## 实现细节

### 入口函数 (main.go)

```go
// main.go:429-447
if markDownloaded {
    results, err := downloading.MarkUsersAsDownloaded(
        ctx, client, db, task.lists, task.users, pathHelper.root, markTime)
    if err != nil {
        log.Errorln("failed to mark users as downloaded:", err)
        os.Exit(1)
    }
    // 输出结果供外部程序解析
    if len(results) > 0 {
        fmt.Println("\n=== MARK_DOWNLOADED_RESULTS ===")
        for _, r := range results {
            status := "OK"
            if !r.Success {
                status = "FAIL"
            }
            fmt.Printf("ENTITY_ID:%d|USER_ID:%d|SCREEN_NAME:%s|STATUS:%s\n",
                r.EntityID, r.UserID, r.ScreenName, status)
        }
        fmt.Println("=== END_RESULTS ===")
    }
}
```

### 核心函数 (features.go)

```go
// features.go:849-938
func MarkUsersAsDownloaded(ctx context.Context, client *resty.Client, 
    db *sqlx.DB, lists []twitter.ListBase, users []*twitter.User, 
    dir string, markTimeStr string) ([]MarkedUserInfo, error) {
    
    // 1. 解析时间戳
    var timestamp *time.Time
    if markTimeStr == "" {
        now := time.Now()
        timestamp = &now
        log.Infoln("marking users as downloaded, timestamp:", timestamp.Format(time.RFC3339))
    } else if strings.ToLower(markTimeStr) == "null" || 
              strings.ToLower(markTimeStr) == "nil" {
        timestamp = nil
        log.Infoln("marking users as downloaded, timestamp: NULL (full download)")
    } else {
        loc, locErr := time.LoadLocation("Local")
        if locErr != nil {
            loc = time.UTC
        }
        parsedTime, err := time.ParseInLocation(
            "2006-01-02T15:04:05", markTimeStr, loc)
        if err != nil {
            return nil, fmt.Errorf("invalid mark-time format '%s'...", markTimeStr)
        }
        timestamp = &parsedTime
        log.Infoln("marking users as downloaded, timestamp:", timestamp.Format(time.RFC3339))
    }

    var results []MarkedUserInfo
    var successCount, failCount int

    // 2. 处理列表中的用户
    for _, lst := range lists {
        if err := context.Cause(ctx); err != nil {
            return results, err
        }
        if lst == nil {
            continue
        }
        members, err := lst.GetMembers(ctx, client)
        if err != nil {
            errStr := err.Error()
            if strings.Contains(errStr, "does not exist or is not accessible") ||
                strings.Contains(errStr, "unable to get timeline data") {
                return nil, fmt.Errorf("list %s does not exist or is not accessible", lst.Title())
            }
            log.WithField("list", lst.Title()).Warnln("failed to get list members:", err)
            continue
        }
        for _, user := range members {
            if err := context.Cause(ctx); err != nil {
                return results, err
            }
            if user == nil {
                continue
            }
            info := markSingleUserWithInfo(db, user, dir, timestamp)
            results = append(results, info)
            if info.Success {
                successCount++
            } else {
                failCount++
            }
        }
    }

    // 3. 处理直接指定的用户
    for _, user := range users {
        if user == nil {
            continue
        }
        info := markSingleUserWithInfo(db, user, dir, timestamp)
        results = append(results, info)
        if info.Success {
            successCount++
        } else {
            failCount++
        }
    }

    log.Infoln("finished marking users as downloaded, success:", successCount, "failed:", failCount)
    return results, nil
}
```

### 标记单个用户 (features.go)

```go
// features.go:941-995
// markSingleUserWithInfo 标记单个用户为已下载并返回详细信息
func markSingleUserWithInfo(db *sqlx.DB, user *twitter.User, 
    dir string, timestamp *time.Time) (info MarkedUserInfo) {
    
    // 防御性检查：确保 user 不为 nil
    if user == nil {
        info.Success = false
        info.Error = "user is nil"
        return info
    }

    info = MarkedUserInfo{
        UserID:     user.Id,
        ScreenName: user.ScreenName,
        Success:    false,
    }

    // 捕获可能的 panic，增加健壮性
    defer func() {
        if r := recover(); r != nil {
            info.Success = false
            info.Error = fmt.Sprintf("panic: %v", r)
            log.WithField("user", user.Title()).Errorln("panic in markSingleUserWithInfo:", r)
        }
    }()

    // 同步用户和实体（与正常下载使用相同的逻辑）
    entity, err := syncUserAndEntity(db, user, dir)
    if err != nil {
        info.Error = fmt.Sprintf("failed to sync user and entity: %v", err)
        log.WithField("user", user.Title()).Warnln("failed to mark user:", err)
        return info
    }

    // 设置 latest_release_time
    if timestamp == nil {
        // 设置为 NULL，用于全量下载
        if err := entity.ClearLatestReleaseTime(); err != nil {
            info.Error = fmt.Sprintf("failed to clear latest release time: %v", err)
            log.WithField("user", user.Title()).Warnln("failed to clear latest release time:", err)
            return info
        }
        log.WithField("user", user.Title()).Infoln("cleared latest release time for full download")
    } else {
        // 设置为指定时间
        if err := entity.SetLatestReleaseTime(*timestamp); err != nil {
            info.Error = fmt.Sprintf("failed to set latest release time: %v", err)
            log.WithField("user", user.Title()).Warnln("failed to set latest release time:", err)
            return info
        }
    }

    info.Success = true
    info.EntityID = entity.Id()
    log.WithField("user", user.Title()).Infoln("marked as downloaded")
    return info
}
```

---

## 数据库结构

### users 表

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    screen_name TEXT UNIQUE,
    name TEXT,
    protected BOOLEAN,
    friends_count INTEGER
);
```

### user_entities 表

```sql
CREATE TABLE user_entities (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,                -- 用户ID（外键关联 users.id）
    name TEXT,                      -- 用户文件夹名称
    parent_dir TEXT,                -- 父目录路径
    latest_release_time DATETIME,   -- 最新推文时间（可为NULL）
    media_count INTEGER,
    UNIQUE (user_id, parent_dir),
    FOREIGN KEY(user_id) REFERENCES users (id)
);
```

**Go 结构体映射**：
```go
type UserEntity struct {
    Id                sql.NullInt32 `db:"id"`
    Uid               uint64        `db:"user_id"`    // Go 字段 Uid → 数据库 user_id
    Name              string        `db:"name"`
    LatestReleaseTime sql.NullTime  `db:"latest_release_time"`
    ParentDir         string        `db:"parent_dir"`
    MediaCount        sql.NullInt32 `db:"media_count"`
}
```

### 数据库操作函数 (crud.go)

```go
// 设置时间戳
func SetUserEntityLatestReleaseTime(db *sqlx.DB, id int, t time.Time) error {
    stmt := `UPDATE user_entities SET latest_release_time=? WHERE id=?`
    result, err := db.Exec(stmt, t, id)
    // ...
}

// 清除时间戳（设置为NULL）
func ClearUserEntityLatestReleaseTime(db *sqlx.DB, id int) error {
    stmt := `UPDATE user_entities SET latest_release_time=NULL WHERE id=?`
    result, err := db.Exec(stmt, id)
    // ...
}
```

---

## 时间过滤机制

### 下载时的时间过滤 (user.go)

```go
// user.go:174-223
func (u *User) GetMeidas(ctx context.Context, client *resty.Client, 
    timeRange *utils.TimeRange) ([]*Tweet, error) {
    
    if !u.IsVisiable() {
        return nil, nil
    }

    api := userMedia{}
    api.count = 100
    api.cursor = ""
    api.userId = u.Id

    results := make([]*Tweet, 0)

    var minTime *time.Time
    var maxTime *time.Time

    if timeRange != nil {
        minTime = &timeRange.Min
        maxTime = &timeRange.Max
    }

    for {
        currentTweets, next, err := u.getMediasOnePage(ctx, &api, client)
        if err != nil {
            return nil, err
        }

        if len(currentTweets) == 0 {
            break // empty page
        }

        api.SetCursor(next)

        if timeRange == nil {
            results = append(results, currentTweets...)
            continue
        }

        // 筛选推文，并判断是否获取下页
        cutMin, cutMax, currentTweets := filterTweetsByTimeRange(currentTweets, minTime, maxTime)
        results = append(results, currentTweets...)

        if cutMin {
            break
        }
        if cutMax && len(currentTweets) != 0 {
            maxTime = nil
        }
    }
    return results, nil
}
```

### 时间过滤函数 (user.go)

```go
// user.go:139-172
func filterTweetsByTimeRange(tweets []*Tweet, min *time.Time, max *time.Time) 
    (cutMin bool, cutMax bool, res []*Tweet) {
    
    n := len(tweets)
    begin, end := 0, n

    // 从左到右查找第一个小于 min 的推文
    if min != nil && !min.IsZero() {
        for i := 0; i < n; i++ {
            if !tweets[i].CreatedAt.After(*min) {
                end = i // 找到第一个不大于 min 的推文位置
                cutMin = true
                break
            }
        }
    }

    // 从右到左查找最后一个大于 max 的推文
    if max != nil && !max.IsZero() {
        for i := n - 1; i >= 0; i-- {
            if !tweets[i].CreatedAt.Before(*max) {
                begin = i + 1 // 找到第一个不小于 max 的推文位置
                cutMax = true
                break
            }
        }
    }

    if begin >= end {
        // 如果最终的范围无效，返回空结果
        return cutMin, cutMax, nil
    }

    res = tweets[begin:end]
    return
}
```

### 关键行为

| `latest_release_time` 值 | `min.IsZero()` | 过滤行为 |
|-------------------------|----------------|---------|
| NULL | true | 不过滤，获取全部历史推文 |
| 零值 `time.Time{}` | true | 不过滤，获取全部历史推文 |
| 有效时间 | false | 只获取该时间之后的推文 |

---

## 新用户处理流程

### 流程图

```
用户不存在于数据库
        ↓
    syncUserAndEntity()
        ↓
    ├── syncUser()
    │       ↓
    │   GetUserById() 返回 nil
    │       ↓
    │   创建 User 记录
    │       ↓
    │   CreateUser() INSERT
    │
    ├── NewUserEntity()
    │       ↓
    │   LocateUserEntity() 返回 nil
    │       ↓
    │   创建 UserEntity 记录（内存）
    │       ↓
    │   created = false
    │
    └── syncPath()
            ↓
        path.Recorded() 返回 false
            ↓
        path.Create(expectedName)
            ↓
        os.MkdirAll() 创建文件夹
            ↓
        CreateUserEntity() INSERT
            ↓
        created = true
```

### 代码实现

```go
// features.go:348-362
func syncUserAndEntity(db *sqlx.DB, user *twitter.User, dir string) (*UserEntity, error) {
    // 1. 同步用户信息到 users 表
    if err := syncUser(db, user); err != nil {
        return nil, err
    }
    
    // 2. 创建或定位用户实体
    entity, err := NewUserEntity(db, user.Id, dir)
    if err != nil {
        return nil, err
    }
    
    // 3. 同步文件夹路径
    expectedTitle := utils.WinFileName(user.Title())
    if err = syncPath(entity, expectedTitle); err != nil {
        return nil, err
    }
    
    return entity, nil
}

// features.go:271-304
func syncUser(db *sqlx.DB, user *twitter.User) error {
    renamed := false
    isNew := false
    usrdb, err := database.GetUserById(db, user.Id)
    if err != nil {
        return err
    }

    if usrdb == nil {
        isNew = true
        usrdb = &database.User{}
        usrdb.Id = user.Id
    } else {
        renamed = usrdb.Name != user.Name || usrdb.ScreenName != user.ScreenName
    }

    usrdb.FriendsCount = user.FriendsCount
    usrdb.IsProtected = user.IsProtected
    usrdb.Name = user.Name
    usrdb.ScreenName = user.ScreenName

    if isNew {
        err = database.CreateUser(db, usrdb)
    } else {
        err = database.UpdateUser(db, usrdb)
    }
    if err != nil {
        return err
    }
    if renamed || isNew {
        err = database.RecordUserPreviousName(db, user.Id, user.Name, user.ScreenName)
    }
    return err
}

// entity.go:47-60
func NewUserEntity(db *sqlx.DB, uid uint64, parentDir string) (*UserEntity, error) {
    created := true
    record, err := database.LocateUserEntity(db, uid, parentDir)
    
    if record == nil {
        // 新用户：创建实体记录（尚未保存到数据库）
        record = &database.UserEntity{}
        record.Uid = uid
        record.ParentDir = parentDir
        created = false
    }
    return &UserEntity{record: record, db: db, created: created}, nil
}

// entity.go:24-39
func syncPath(path SmartPath, expectedName string) error {
    if !path.Recorded() {
        // 新用户：创建文件夹 + 数据库记录
        return path.Create(expectedName)
    }
    // 已存在：检查是否需要重命名
    if path.Name() != expectedName {
        return path.Rename(expectedName)
    }
    
    p, err := path.Path()
    if err != nil {
        return err
    }
    return os.MkdirAll(p, 0755)
}
```

---

## 错误处理

### 防御性检查

```go
// 1. nil 用户检查
if user == nil {
    info.Success = false
    info.Error = "user is nil"
    return info
}

// 2. panic 恢复
defer func() {
    if r := recover(); r != nil {
        info.Success = false
        info.Error = fmt.Sprintf("panic: %v", r)
    }
}()

// 3. 同步失败处理
entity, err := syncUserAndEntity(db, user, dir)
if err != nil {
    info.Error = fmt.Sprintf("failed to sync user and entity: %v", err)
    return info
}

// 4. 时间戳设置失败处理
if err := entity.SetLatestReleaseTime(*timestamp); err != nil {
    info.Error = fmt.Sprintf("failed to set latest release time: %v", err)
    return info
}
```

### 列表访问错误

```go
members, err := lst.GetMembers(ctx, client)
if err != nil {
    errStr := err.Error()
    if strings.Contains(errStr, "does not exist or is not accessible") {
        return nil, fmt.Errorf("list %s does not exist or is not accessible", lst.Title())
    }
    log.WithField("list", lst.Title()).Warnln("failed to get list members:", err)
    continue  // 继续处理其他列表
}
```

---

## 输出格式

### 标准输出

```
=== MARK_DOWNLOADED_RESULTS ===
ENTITY_ID:1|USER_ID:44196397|SCREEN_NAME:elonmusk|STATUS:OK
ENTITY_ID:2|USER_ID:23248887|SCREEN_NAME:NASA|STATUS:OK
ENTITY_ID:3|USER_ID:12345|SCREEN_NAME:testuser|STATUS:FAIL
=== END_RESULTS ===
```

### 字段说明

| 字段 | 说明 |
|------|------|
| `ENTITY_ID` | user_entities 表中的记录ID |
| `USER_ID` | Twitter 用户ID |
| `SCREEN_NAME` | Twitter 用户名 |
| `STATUS` | `OK` 成功 / `FAIL` 失败 |

### 日志输出

```
INFO[0000] marking users as downloaded, timestamp: 2024-06-15T10:30:00+08:00
INFO[0001] marked as downloaded                              user=Elon Musk(elonmusk)
INFO[0001] finished marking users as downloaded, success: 3 failed: 0
```

---

## 常见场景

### 场景1：首次下载后跳过历史

```bash
# 首次下载
tmd -user elonmusk

# 以后只想下载新推文，跳过历史
tmd -user elonmusk -mark-downloaded
```

### 场景2：重新下载特定时间段

```bash
# 重新下载 2024 年的推文
tmd -user elonmusk -mark-downloaded -mark-time "2024-01-01T00:00:00"
```

### 场景3：完全重新下载

```bash
# 清除记录，全量下载
tmd -user elonmusk -mark-downloaded -mark-time "null"
tmd -user elonmusk
```

### 场景4：批量管理列表

```bash
# 标记整个列表为已下载
tmd -list 123456789 -mark-downloaded

# 重置整个列表
tmd -list 123456789 -mark-downloaded -mark-time "null"
```

### 场景5：新用户预处理

```bash
# 添加新用户但不下载历史
tmd -user newuser123 -mark-downloaded
# 以后只下载新推文
```

---

## 注意事项

### 1. 不下载任何内容

`-mark-downloaded` **只更新数据库**，不会：
- 获取推文
- 下载媒体文件
- 创建 .loongtweet 文件

### 2. 幂等性

可以重复执行，每次都会覆盖 `latest_release_time`：
```bash
tmd -user elonmusk -mark-downloaded -mark-time "2024-01-01T00:00:00"
tmd -user elonmusk -mark-downloaded -mark-time "2024-06-01T00:00:00"  # 覆盖
```

### 3. PowerShell 引号问题

```powershell
# ❌ 错误 - null 会被解释为 $null
tmd -user elonmusk -mark-downloaded -mark-time null

# ✅ 正确
tmd -user elonmusk -mark-downloaded -mark-time "null"
```

### 4. 与其他参数的兼容性

| 组合 | 兼容 | 说明 |
|------|:----:|------|
| `-mark-downloaded` + `-user` | ✅ | 标记指定用户 |
| `-mark-downloaded` + `-list` | ✅ | 标记列表成员 |
| `-mark-downloaded` + `-foll` | ✅ | 标记关注用户 |
| `-mark-downloaded` + `--profile` | ⚠️ | 只标记，不下载 profile |
| `-mark-downloaded` + `-mark-time` | ✅ | 指定标记时间 |

### 5. 时间格式严格

格式必须为 `2006-01-02T15:04:05`：
```bash
# ✅ 正确
tmd -user elonmusk -mark-downloaded -mark-time "2024-06-15T10:30:00"

# ❌ 错误
tmd -user elonmusk -mark-downloaded -mark-time "2024-06-15"
tmd -user elonmusk -mark-downloaded -mark-time "2024/06/15 10:30:00"
```

### 6. 数据库文件位置

| 系统 | 路径 |
|------|------|
| Windows | `{存储目录}\.data\foo.db` |
| macOS/Linux | `{存储目录}/.data/foo.db` |

---

## 附录：相关源码文件

| 文件 | 行号 | 说明 |
|------|------|------|
| `main.go` | 252-253 | 参数定义 |
| `main.go` | 429-447 | 入口调用 |
| `internal/downloading/features.go` | 849-938 | MarkUsersAsDownloaded 核心实现 |
| `internal/downloading/features.go` | 942-995 | markSingleUserWithInfo 单用户标记 |
| `internal/downloading/features.go` | 348-362 | syncUserAndEntity 用户同步 |
| `internal/downloading/features.go` | 271-304 | syncUser 用户信息同步 |
| `internal/downloading/entity.go` | 142-162 | SetLatestReleaseTime/ClearLatestReleaseTime |
| `internal/database/crud.go` | 286-310 | 数据库操作函数 |
| `internal/twitter/user.go` | 174-223 | GetMeidas 时间过滤 |
| `internal/twitter/user.go` | 139-172 | filterTweetsByTimeRange 过滤函数 |

---

## 版本历史

- 初始版本：支持基本标记功能
- 当前版本：支持 `null`/`nil` 重置、详细输出、错误处理

---

*文档生成日期：2024年*
