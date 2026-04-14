# SmartPath 接口重构计划

## 1. 问题分析

### 1.1 当前问题

`SmartPath` 接口同时被 `UserEntity` 和 `ListEntity` 实现，但两者的业务逻辑差异较大：

| 特性 | UserEntity | ListEntity |
|------|-----------|------------|
| 关联ID | `Uid uint64`（用户ID） | `LstId int64`（列表ID） |
| 特有方法 | `LatestReleaseTime()`, `SetLatestReleaseTime()`, `ClearLatestReleaseTime()`, `Uid()` | 无 |
| 用途 | 管理用户推文下载目录 | 管理列表推文下载目录 |

### 1.2 设计问题

1. **接口污染**：`SmartPath` 接口被用于两种不同业务场景
2. **职责不单一**：`UserEntity` 包含推文下载特有的方法，但 `SmartPath` 接口无法表达这种差异
3. **类型断言需求**：调用方需要使用类型断言来访问 `UserEntity` 特有方法
4. **扩展困难**：如果后续需要为 `ListEntity` 添加类似功能，接口设计会变得混乱

## 2. 重构方案

### 2.1 目标架构

```
internal/
├── entity/              # 新增：实体管理模块
│   ├── interface.go     # 实体接口定义
│   ├── user.go          # UserEntity 实现
│   ├── list.go          # ListEntity 实现
│   └── sync.go          # 实体同步逻辑
├── downloading/         # 保持：推文下载核心逻辑
│   └── ...
└── database/            # 保持：数据库操作
    └── ...
```

### 2.2 接口设计

#### 2.2.1 基础接口（最小化）

```go
// internal/entity/interface.go

// Entity 基础实体接口，定义所有实体共有的行为
type Entity interface {
    Path() (string, error)
    Create(name string) error
    Rename(string) error
    Remove() error
    Name() (string, error)
    Id() (int, error)
    Recorded() bool
}

// Syncer 实体同步接口
type Syncer interface {
    Sync(expectedName string) error
}
```

#### 2.2.2 UserEntity 特有接口

```go
// internal/entity/user.go

// UserEntity 用户实体
type UserEntity struct {
    record  *database.UserEntity
    db      *sqlx.DB
    created bool
}

// 实现 Entity 接口
func (ue *UserEntity) Path() (string, error) { ... }
func (ue *UserEntity) Create(name string) error { ... }
func (ue *UserEntity) Rename(title string) error { ... }
func (ue *UserEntity) Remove() error { ... }
func (ue *UserEntity) Name() (string, error) { ... }
func (ue *UserEntity) Id() (int, error) { ... }
func (ue *UserEntity) Recorded() bool { ... }

// UserEntity 特有方法
func (ue *UserEntity) Uid() uint64 { ... }
func (ue *UserEntity) LatestReleaseTime() (time.Time, error) { ... }
func (ue *UserEntity) SetLatestReleaseTime(t time.Time) error { ... }
func (ue *UserEntity) ClearLatestReleaseTime() error { ... }
func (ue *UserEntity) ParentDir() string { ... }
```

#### 2.2.3 ListEntity 实现

```go
// internal/entity/list.go

// ListEntity 列表实体
type ListEntity struct {
    record  *database.LstEntity
    db      *sqlx.DB
    created bool
}

// 实现 Entity 接口
func (le *ListEntity) Path() (string, error) { ... }
func (le *ListEntity) Create(name string) error { ... }
func (le *ListEntity) Rename(title string) error { ... }
func (le *ListEntity) Remove() error { ... }
func (le *ListEntity) Name() (string, error) { ... }
func (le *ListEntity) Id() (int, error) { ... }
func (le *ListEntity) Recorded() bool { ... }
```

#### 2.2.4 同步逻辑提取

```go
// internal/entity/sync.go

// Sync 同步实体路径和名称
func Sync(e Entity, expectedName string) error {
    if !e.Recorded() {
        return e.Create(expectedName)
    }

    name, err := e.Name()
    if err != nil {
        return err
    }
    if name != expectedName {
        return e.Rename(expectedName)
    }

    p, err := e.Path()
    if err != nil {
        return err
    }

    return os.MkdirAll(p, 0755)
}
```

## 3. 修改步骤

### 步骤 1: 创建 `internal/entity` 包

**新增文件**: `internal/entity/interface.go`

```go
package entity

// Entity 基础实体接口，定义所有实体共有的行为
type Entity interface {
    Path() (string, error)
    Create(name string) error
    Rename(string) error
    Remove() error
    Name() (string, error)
    Id() (int, error)
    Recorded() bool
}
```

### 步骤 2: 迁移 UserEntity

**新增文件**: `internal/entity/user.go`

```go
package entity

import (
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/jmoiron/sqlx"
    "github.com/unkmonster/tmd/internal/database"
)

// UserEntity 用户实体
type UserEntity struct {
    record  *database.UserEntity
    db      *sqlx.DB
    created bool
}

// NewUserEntity 创建或加载用户实体
func NewUserEntity(db *sqlx.DB, uid uint64, parentDir string) (*UserEntity, error) {
    created := true
    record, err := database.LocateUserEntity(db, uid, parentDir)
    if err != nil {
        return nil, err
    }
    if record == nil {
        record = &database.UserEntity{}
        record.Uid = uid
        record.ParentDir = parentDir
        created = false
    }
    return &UserEntity{record: record, db: db, created: created}, nil
}

// 实现 Entity 接口
func (ue *UserEntity) Create(name string) error {
    ue.record.Name = name
    path, _ := ue.Path()
    if err := os.MkdirAll(path, 0755); err != nil {
        return err
    }

    if err := database.CreateUserEntity(ue.db, ue.record); err != nil {
        return err
    }
    ue.created = true
    return nil
}

func (ue *UserEntity) Remove() error {
    path, _ := ue.Path()

    if err := os.RemoveAll(path); err != nil {
        return err
    }
    if err := database.DelUserEntity(ue.db, uint32(ue.record.Id.Int32)); err != nil {
        return err
    }
    ue.created = false
    return nil
}

func (ue *UserEntity) Rename(title string) error {
    if !ue.created {
        return fmt.Errorf("user entity [%s:%d] was not created", ue.record.ParentDir, ue.record.Uid)
    }

    old, _ := ue.Path()
    newPath := filepath.Join(filepath.Dir(old), title)

    err := os.Rename(old, newPath)
    if os.IsNotExist(err) {
        err = os.Mkdir(newPath, 0755)
    }
    if err != nil && !os.IsExist(err) {
        return err
    }

    ue.record.Name = title
    return database.UpdateUserEntity(ue.db, ue.record)
}

func (ue *UserEntity) Path() (string, error) {
    return ue.record.Path()
}

func (ue *UserEntity) ParentDir() string {
    if ue.record == nil {
        return ""
    }
    return ue.record.ParentDir
}

func (ue *UserEntity) Name() (string, error) {
    if ue.record.Name == "" {
        return "", fmt.Errorf("the name of user entity [%s:%d] was unset", ue.record.ParentDir, ue.record.Uid)
    }
    return ue.record.Name, nil
}

func (ue *UserEntity) Id() (int, error) {
    if !ue.created {
        return 0, fmt.Errorf("user entity [%s:%d] was not created", ue.record.ParentDir, ue.record.Uid)
    }
    return int(ue.record.Id.Int32), nil
}

func (ue *UserEntity) Recorded() bool {
    return ue.created
}

// UserEntity 特有方法
func (ue *UserEntity) Uid() uint64 {
    return ue.record.Uid
}

func (ue *UserEntity) LatestReleaseTime() (time.Time, error) {
    if !ue.created {
        return time.Time{}, fmt.Errorf("user entity [%s:%d] was not created", ue.record.ParentDir, ue.record.Uid)
    }
    return ue.record.LatestReleaseTime.Time, nil
}

func (ue *UserEntity) SetLatestReleaseTime(t time.Time) error {
    if !ue.created {
        return fmt.Errorf("user entity [%s:%d] was not created", ue.record.ParentDir, ue.record.Uid)
    }
    err := database.SetUserEntityLatestReleaseTime(ue.db, int(ue.record.Id.Int32), t)
    if err == nil {
        ue.record.LatestReleaseTime.Scan(t)
    }
    return err
}

func (ue *UserEntity) ClearLatestReleaseTime() error {
    if !ue.created {
        return fmt.Errorf("user entity [%s:%d] was not created", ue.record.ParentDir, ue.record.Uid)
    }
    err := database.ClearUserEntityLatestReleaseTime(ue.db, int(ue.record.Id.Int32))
    if err == nil {
        ue.record.LatestReleaseTime.Valid = false
    }
    return err
}
```

### 步骤 3: 迁移 ListEntity

**新增文件**: `internal/entity/list.go`

```go
package entity

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/jmoiron/sqlx"
    "github.com/unkmonster/tmd/internal/database"
)

// ListEntity 列表实体
type ListEntity struct {
    record  *database.LstEntity
    db      *sqlx.DB
    created bool
}

// NewListEntity 创建或加载列表实体
func NewListEntity(db *sqlx.DB, lid int64, parentDir string) (*ListEntity, error) {
    created := true
    record, err := database.LocateLstEntity(db, lid, parentDir)
    if err != nil {
        return nil, err
    }
    if record == nil {
        record = &database.LstEntity{}
        record.LstId = lid
        record.ParentDir = parentDir
        created = false
    }
    return &ListEntity{record: record, db: db, created: created}, nil
}

// 实现 Entity 接口
func (le *ListEntity) Create(name string) error {
    le.record.Name = name
    path, _ := le.Path()
    if err := os.MkdirAll(path, 0755); err != nil {
        return nil
    }

    if err := database.CreateLstEntity(le.db, le.record); err != nil {
        return err
    }
    le.created = true
    return nil
}

func (le *ListEntity) Remove() error {
    if !le.created {
        return fmt.Errorf("list entity [%s:%d] was not created", le.record.ParentDir, le.record.LstId)
    }

    path, _ := le.Path()
    if err := os.RemoveAll(path); err != nil {
        return err
    }
    if err := database.DelLstEntity(le.db, int(le.record.Id.Int32)); err != nil {
        return err
    }
    le.created = false
    return nil
}

func (le *ListEntity) Rename(title string) error {
    if !le.created {
        return fmt.Errorf("list entity [%s:%d] was not created", le.record.ParentDir, le.record.LstId)
    }

    path, _ := le.Path()
    newPath := filepath.Join(filepath.Dir(path), title)
    err := os.Rename(path, newPath)
    if os.IsNotExist(err) {
        err = os.Mkdir(newPath, 0755)
    }
    if err != nil && !os.IsExist(err) {
        return err
    }

    le.record.Name = title
    return database.UpdateLstEntity(le.db, le.record)
}

func (le *ListEntity) Path() (string, error) {
    return le.record.Path()
}

func (le *ListEntity) Name() (string, error) {
    if le.record.Name == "" {
        return "", fmt.Errorf("the name of list entity [%s:%d] was unset", le.record.ParentDir, le.record.LstId)
    }
    return le.record.Name, nil
}

func (le *ListEntity) Id() (int, error) {
    if !le.created {
        return 0, fmt.Errorf("list entity [%s:%d] was not created", le.record.ParentDir, le.record.LstId)
    }
    return int(le.record.Id.Int32), nil
}

func (le *ListEntity) Recorded() bool {
    return le.created
}
```

### 步骤 4: 迁移同步逻辑

**新增文件**: `internal/entity/sync.go`

```go
package entity

import (
    "os"
)

// Sync 同步实体路径和名称
// 如果实体未创建，则创建它
// 如果实体名称与预期不符，则重命名
// 如果名称相同，则确保目录存在
func Sync(e Entity, expectedName string) error {
    if !e.Recorded() {
        return e.Create(expectedName)
    }

    name, err := e.Name()
    if err != nil {
        return err
    }
    if name != expectedName {
        return e.Rename(expectedName)
    }

    p, err := e.Path()
    if err != nil {
        return err
    }

    return os.MkdirAll(p, 0755)
}
```

### 步骤 5: 修改 `internal/downloading/entity.go`

**修改前**:
```go
package downloading

// 路径Plus
type SmartPath interface {
    Path() (string, error)
    Create(name string) error
    Rename(string) error
    Remove() error
    Name() (string, error)
    Id() (int, error)
    Recorded() bool
}

func syncPath(path SmartPath, expectedName string) error { ... }

type UserEntity struct { ... }
type ListEntity struct { ... }
```

**修改后**:
```go
package downloading

import (
    "github.com/unkmonster/tmd/internal/entity"
)

// 为了保持向后兼容，保留类型别名
type UserEntity = entity.UserEntity
type ListEntity = entity.ListEntity

// 为了保持向后兼容，保留函数别名
func syncPath(e entity.Entity, expectedName string) error {
    return entity.Sync(e, expectedName)
}

// NewUserEntity 和 NewListEntity 直接调用 entity 包的函数
func NewUserEntity(db *sqlx.DB, uid uint64, parentDir string) (*UserEntity, error) {
    return entity.NewUserEntity(db, uid, parentDir)
}

func NewListEntity(db *sqlx.DB, lid int64, parentDir string) (*ListEntity, error) {
    return entity.NewListEntity(db, lid, parentDir)
}
```

### 步骤 6: 更新 `internal/downloading/features.go` 中的引用

**修改前**:
```go
import (
    ...
    "github.com/unkmonster/tmd/internal/database"
    "github.com/unkmonster/tmd/internal/downloader"
    ...
)
```

**修改后**:
```go
import (
    ...
    "github.com/unkmonster/tmd/internal/database"
    "github.com/unkmonster/tmd/internal/downloader"
    "github.com/unkmonster/tmd/internal/entity"
    ...
)
```

**修改 `TweetInEntity` 结构体**:

```go
// TweetInEntity 推文与实体的绑定
type TweetInEntity struct {
    Tweet  *twitter.Tweet
    Entity *entity.UserEntity  // 使用 entity 包的类型
}
```

### 步骤 7: 更新 `internal/downloading/dumper.go`

**修改前**:
```go
func (td *TweetDumper) GetTotal(db *sqlx.DB) ([]*TweetInEntity, error) {
    ...
    ue := UserEntity{db: db, record: e, created: true}
    ...
}
```

**修改后**:
```go
import "github.com/unkmonster/tmd/internal/entity"

func (td *TweetDumper) GetTotal(db *sqlx.DB) ([]*TweetInEntity, error) {
    ...
    ue := entity.UserEntity{...}  // 使用 entity 包的类型
    ...
}
```

### 步骤 8: 更新测试文件

**修改 `internal/downloading/download_test.go`**:

```go
import (
    ...
    "github.com/unkmonster/tmd/internal/entity"
)

// 更新 verifyDir 函数
func verifyDir(t *testing.T, e entity.Entity, wantPath string) { ... }

// 更新 verifyUserRecord 函数
func verifyUserRecord(t *testing.T, e entity.Entity, uid uint64, name string, parentDir string) *entity.UserEntity {
    ...
    return e.(*entity.UserEntity)
}

// 更新 verifyLstRecord 函数
func verifyLstRecord(t *testing.T, e entity.Entity, lid int64, name string, parentDir string) { ... }

// 更新 testSyncUser 函数
func testSyncUser(t *testing.T, name string, uid int, parentdir string, exist bool) *entity.UserEntity {
    ue, err := entity.NewUserEntity(db, uint64(uid), parentdir)
    ...
    if err := entity.Sync(ue, name); err != nil {
        ...
    }
    ...
    return ue
}

// 更新 testSyncList 函数
func testSyncList(t *testing.T, name string, lid int, parentDir string, exist bool) *entity.ListEntity {
    le, err := entity.NewListEntity(db, int64(lid), parentDir)
    ...
    if err := entity.Sync(le, name); err != nil {
        ...
    }
    ...
    return le
}
```

## 4. 风险评估

| 风险 | 可能性 | 影响 | 缓解措施 |
|------|--------|------|----------|
| 导入循环 | 低 | 中 | 确保 `entity` 包只依赖 `database` 包，不依赖 `downloading` 包 |
| 类型别名导致编译错误 | 低 | 高 | 使用类型别名（`type X = Y`）而非类型定义（`type X Y`）保持完全兼容 |
| 测试失败 | 中 | 中 | 保持所有测试逻辑不变，仅更新导入和类型引用 |
| 运行时 panic | 低 | 高 | 确保所有方法实现完全一致，不修改行为 |

## 5. 测试要点

### 5.1 单元测试

```bash
# 运行 entity 包测试
go test -v ./internal/entity/... -count=1

# 运行 downloading 包测试
go test -v ./internal/downloading/... -count=1
```

### 5.2 集成测试

```bash
# 运行所有测试
go test -v ./... -count=1
```

### 5.3 测试覆盖检查

确保以下场景被覆盖：
1. `UserEntity.Create()` - 创建新用户实体
2. `UserEntity.Rename()` - 重命名用户目录
3. `UserEntity.Remove()` - 删除用户实体
4. `UserEntity.LatestReleaseTime()` - 获取/设置最新发布时间
5. `ListEntity.Create()` - 创建新列表实体
6. `ListEntity.Rename()` - 重命名列表目录
7. `entity.Sync()` - 同步实体路径

## 6. 回滚计划

如果重构出现问题，可以通过以下步骤回滚：

1. 删除 `internal/entity` 目录
2. 恢复 `internal/downloading/entity.go` 到原始版本
3. 恢复 `internal/downloading/download_test.go` 到原始版本
4. 恢复 `internal/downloading/dumper.go` 到原始版本

## 7. 实施顺序

1. **Phase 1**: 创建 `internal/entity` 包（interface.go, user.go, list.go, sync.go）
2. **Phase 2**: 修改 `internal/downloading/entity.go` 使用类型别名
3. **Phase 3**: 更新 `internal/downloading/features.go` 导入
4. **Phase 4**: 更新 `internal/downloading/dumper.go` 导入
5. **Phase 5**: 更新 `internal/downloading/download_test.go` 测试
6. **Phase 6**: 运行全部测试验证
7. **Phase 7**: （可选）删除 `internal/downloading/entity.go` 中的冗余代码

## 8. 代码前后对比

### 8.1 接口定义

**修改前** (`downloading/entity.go`):
```go
type SmartPath interface {
    Path() (string, error)
    Create(name string) error
    Rename(string) error
    Remove() error
    Name() (string, error)
    Id() (int, error)
    Recorded() bool
}
```

**修改后** (`entity/interface.go`):
```go
type Entity interface {
    Path() (string, error)
    Create(name string) error
    Rename(string) error
    Remove() error
    Name() (string, error)
    Id() (int, error)
    Recorded() bool
}
```

### 8.2 同步函数

**修改前** (`downloading/entity.go`):
```go
func syncPath(path SmartPath, expectedName string) error {
    if !path.Recorded() {
        return path.Create(expectedName)
    }
    name, err := path.Name()
    if err != nil {
        return err
    }
    if name != expectedName {
        return path.Rename(expectedName)
    }
    p, err := path.Path()
    if err != nil {
        return err
    }
    return os.MkdirAll(p, 0755)
}
```

**修改后** (`entity/sync.go`):
```go
func Sync(e Entity, expectedName string) error {
    if !e.Recorded() {
        return e.Create(expectedName)
    }
    name, err := e.Name()
    if err != nil {
        return err
    }
    if name != expectedName {
        return e.Rename(expectedName)
    }
    p, err := e.Path()
    if err != nil {
        return err
    }
    return os.MkdirAll(p, 0755)
}
```

### 8.3 UserEntity 特有方法调用

**修改前**:
```go
entity.LatestReleaseTime()  // 直接调用
```

**修改后**:
```go
// 保持不变，因为使用类型别名
tity.LatestReleaseTime()
```

## 9. 最终架构优势

1. **职责分离**：实体管理逻辑独立到 `entity` 包
2. **接口清晰**：`Entity` 接口只包含通用方法
3. **类型安全**：`UserEntity` 特有方法通过具体类型访问
4. **可测试性**：每个包可以独立测试
5. **可扩展性**：新增实体类型只需实现 `Entity` 接口
