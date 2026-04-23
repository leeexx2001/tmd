package database

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
)

type User struct {
	Id           uint64 `db:"id"`
	ScreenName   string `db:"screen_name"`
	Name         string `db:"name"`
	IsProtected  bool   `db:"protected"`
	FriendsCount int    `db:"friends_count"`
	IsAccessible bool   `db:"is_accessible"`
}

type UserEntity struct {
	Id                sql.NullInt32 `db:"id"`
	Uid               uint64        `db:"user_id"`
	Name              string        `db:"name"`
	LatestReleaseTime sql.NullTime  `db:"latest_release_time"`
	ParentDir         string        `db:"parent_dir"`
	MediaCount        sql.NullInt32 `db:"media_count"`
}

type UserLink struct {
	Id                int32  `db:"id" json:"id"`
	UserId            uint64 `db:"user_id" json:"user_id"`
	Name              string `db:"name" json:"name"`
	ParentLstEntityId int32  `db:"parent_lst_entity_id" json:"parent_lst_entity_id"`
}

// UserPreviousName 用户历史名称
type UserPreviousName struct {
	Id         int32     `db:"id" json:"id"`
	Uid        uint64    `db:"uid" json:"uid"`
	ScreenName string    `db:"screen_name" json:"screen_name"`
	Name       string    `db:"name" json:"name"`
	RecordDate time.Time `db:"record_date" json:"record_date"`
}

// List 表示 Twitter 列表
type List struct {
	Id      uint64 `db:"id"`
	Name    string `db:"name"`
	OwnerId uint64 `db:"owner_uid"`
}

// Lst 是 List 的别名，用于向后兼容
// Deprecated: 使用 List 替代
type Lst = List

type LstEntity struct {
	Id        sql.NullInt32 `db:"id"`
	LstId     int64         `db:"lst_id"`
	Name      string        `db:"name"`
	ParentDir string        `db:"parent_dir"`
}

func (le *LstEntity) Path() (string, error) {
	if le.ParentDir == "" || le.Name == "" {
		return "", fmt.Errorf("no enough info to get path for lst entity: parentDir=%q, name=%q", le.ParentDir, le.Name)
	}
	return filepath.Join(le.ParentDir, le.Name), nil
}

func (ue *UserEntity) Path() (string, error) {
	if ue.ParentDir == "" || ue.Name == "" {
		return "", fmt.Errorf("no enough info to get path for user entity: parentDir=%q, name=%q", ue.ParentDir, ue.Name)
	}
	return filepath.Join(ue.ParentDir, ue.Name), nil
}

// LstEntityGetter 获取列表实体的接口
type LstEntityGetter interface {
	GetLstEntity(id int) (*LstEntity, error)
}

// dbGetter 使用 *sqlx.DB 获取列表实体
type dbGetter struct {
	db *sqlx.DB
}

func (g *dbGetter) GetLstEntity(id int) (*LstEntity, error) {
	return GetLstEntity(g.db, id)
}

// txGetter 使用 *sqlx.Tx 获取列表实体
type txGetter struct {
	tx *sqlx.Tx
}

func (g *txGetter) GetLstEntity(id int) (*LstEntity, error) {
	return GetLstEntityTx(g.tx, id)
}

func (ul *UserLink) getPath(getter LstEntityGetter) (string, error) {
	le, err := getter.GetLstEntity(int(ul.ParentLstEntityId))
	if err != nil {
		return "", err
	}
	if le == nil {
		return "", fmt.Errorf("parent list does not exist")
	}

	lePath, err := le.Path()
	if err != nil {
		return "", err
	}
	return filepath.Join(lePath, ul.Name), nil
}

func (ul *UserLink) Path(db *sqlx.DB) (string, error) {
	return ul.getPath(&dbGetter{db: db})
}

func (ul *UserLink) PathTx(tx *sqlx.Tx) (string, error) {
	return ul.getPath(&txGetter{tx: tx})
}

// NullInt32 辅助函数：将 sql.NullInt32 转换为 int32
func NullInt32(n sql.NullInt32) int32 {
	if n.Valid {
		return n.Int32
	}
	return 0
}
