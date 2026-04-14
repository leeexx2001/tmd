package database

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER NOT NULL, 
	screen_name VARCHAR NOT NULL, 
	name VARCHAR NOT NULL, 
	protected BOOLEAN NOT NULL, 
	friends_count INTEGER NOT NULL, 
	PRIMARY KEY (id), 
	UNIQUE (screen_name)
);

CREATE TABLE IF NOT EXISTS user_previous_names (
	id INTEGER NOT NULL, 
	uid INTEGER NOT NULL, 
	screen_name VARCHAR NOT NULL, 
	name VARCHAR NOT NULL, 
	record_date DATE NOT NULL, 
	PRIMARY KEY (id), 
	FOREIGN KEY(uid) REFERENCES users (id)
);

CREATE TABLE IF NOT EXISTS lsts (
	id INTEGER NOT NULL, 
	name VARCHAR NOT NULL, 
	owner_uid INTEGER NOT NULL, 
	PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS lst_entities (
	id INTEGER NOT NULL, 
	lst_id INTEGER NOT NULL, 
	name VARCHAR NOT NULL, 
	parent_dir VARCHAR NOT NULL COLLATE NOCASE, 
	PRIMARY KEY (id), 
	UNIQUE (lst_id, parent_dir)
);

CREATE TABLE IF NOT EXISTS user_entities (
	id INTEGER NOT NULL, 
	user_id INTEGER NOT NULL, 
	name VARCHAR NOT NULL, 
	latest_release_time DATETIME, 
	parent_dir VARCHAR COLLATE NOCASE NOT NULL, 
	media_count INTEGER,
	PRIMARY KEY (id), 
	UNIQUE (user_id, parent_dir), 
	FOREIGN KEY(user_id) REFERENCES users (id)
);

CREATE TABLE IF NOT EXISTS user_links (
	id INTEGER NOT NULL,
	user_id INTEGER NOT NULL, 
	name VARCHAR NOT NULL, 
	parent_lst_entity_id INTEGER NOT NULL,
	PRIMARY KEY (id),
	UNIQUE (user_id, parent_lst_entity_id),
	FOREIGN KEY(user_id) REFERENCES users (id), 
	FOREIGN KEY(parent_lst_entity_id) REFERENCES lst_entities (id)
);

CREATE INDEX IF NOT EXISTS idx_user_links_user_id ON user_links (user_id);
`

func CreateTables(db *sqlx.DB) {
	db.MustExec(schema)
}

// handleGetResult 处理单条查询结果，将 sql.ErrNoRows 转换为 nil, nil
func handleGetResult[T any](result *T, err error) (*T, error) {
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// handleInsertWithId 处理插入操作并获取 LastInsertId
func handleInsertWithId(res sql.Result, err error, idScanner func(int64)) error {
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	idScanner(id)
	return nil
}

func CreateUser(db *sqlx.DB, usr *User) error {
	stmt := `INSERT INTO Users(id, screen_name, name, protected, friends_count) VALUES(:id, :screen_name, :name, :protected, :friends_count)`
	_, err := db.NamedExec(stmt, usr)
	if err != nil {
		return fmt.Errorf("failed to create user %d (%s): %w", usr.Id, usr.ScreenName, err)
	}
	return nil
}

func DelUser(db *sqlx.DB, uid uint64) error {
	stmt := `DELETE FROM users WHERE id=?`
	_, err := db.Exec(stmt, uid)
	if err != nil {
		return fmt.Errorf("failed to delete user %d: %w", uid, err)
	}
	return nil
}

func GetUserById(db *sqlx.DB, uid uint64) (*User, error) {
	stmt := `SELECT * FROM users WHERE id=?`
	result := &User{}
	err := db.Get(result, stmt, uid)
	return handleGetResult(result, err)
}

func UpdateUser(db *sqlx.DB, usr *User) error {
	stmt := `UPDATE users SET screen_name=:screen_name, name=:name, protected=:protected, friends_count=:friends_count WHERE id=:id`
	_, err := db.NamedExec(stmt, usr)
	if err != nil {
		return fmt.Errorf("failed to update user %d: %w", usr.Id, err)
	}
	return nil
}

func CreateUserEntity(db *sqlx.DB, entity *UserEntity) error {
	abs, err := filepath.Abs(entity.ParentDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for parent dir %q: %w", entity.ParentDir, err)
	}
	entity.ParentDir = abs

	stmt := `INSERT INTO user_entities(user_id, name, parent_dir) VALUES(:user_id, :name, :parent_dir)`
	res, err := db.NamedExec(stmt, entity)
	if err != nil {
		return fmt.Errorf("failed to create user entity for user %d in %q: %w", entity.Uid, entity.ParentDir, err)
	}
	return handleInsertWithId(res, err, func(id int64) { entity.Id.Scan(id) })
}

func DelUserEntity(db *sqlx.DB, id uint32) error {
	stmt := `DELETE FROM user_entities WHERE id=?`
	_, err := db.Exec(stmt, id)
	if err != nil {
		return fmt.Errorf("failed to delete user entity %d: %w", id, err)
	}
	return nil
}

func LocateUserEntity(db *sqlx.DB, uid uint64, parentDIr string) (*UserEntity, error) {
	parentDIr, err := filepath.Abs(parentDIr)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %q: %w", parentDIr, err)
	}

	stmt := `SELECT * FROM user_entities WHERE user_id=? AND parent_dir=?`
	result := &UserEntity{}
	err = db.Get(result, stmt, uid, parentDIr)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to locate user entity for user %d in %q: %w", uid, parentDIr, err)
	}
	return handleGetResult(result, err)
}

func GetUserEntity(db *sqlx.DB, id int) (*UserEntity, error) {
	result := &UserEntity{}
	stmt := `SELECT * FROM user_entities WHERE id=?`
	err := db.Get(result, stmt, id)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get user entity %d: %w", id, err)
	}
	return handleGetResult(result, err)
}

func UpdateUserEntity(db *sqlx.DB, entity *UserEntity) error {
	stmt := `UPDATE user_entities SET name=?, latest_release_time=?, media_count=? WHERE id=?`
	_, err := db.Exec(stmt, entity.Name, entity.LatestReleaseTime, entity.MediaCount, entity.Id)
	if err != nil {
		return fmt.Errorf("failed to update user entity %d: %w", entity.Id.Int32, err)
	}
	return nil
}

func UpdateUserEntityMediCount(db *sqlx.DB, eid int, count int) error {
	stmt := `UPDATE user_entities SET media_count=? WHERE id=?`
	_, err := db.Exec(stmt, count, eid)
	if err != nil {
		return fmt.Errorf("failed to update media count for user entity %d: %w", eid, err)
	}
	return nil
}

func UpdateUserEntityTweetStat(db *sqlx.DB, eid int, baseline time.Time, count int) error {
	stmt := `UPDATE user_entities SET latest_release_time=?, media_count=? WHERE id=?`
	_, err := db.Exec(stmt, baseline, count, eid)
	if err != nil {
		return fmt.Errorf("failed to update tweet stat for user entity %d: %w", eid, err)
	}
	return nil
}

func CreateLst(db *sqlx.DB, lst *Lst) error {
	stmt := `INSERT INTO lsts(id, name, owner_uid) VALUES(:id, :name, :owner_uid)`
	_, err := db.NamedExec(stmt, &lst)
	if err != nil {
		return fmt.Errorf("failed to create list %d (%s): %w", lst.Id, lst.Name, err)
	}
	return nil
}

func DelLst(db *sqlx.DB, lid uint64) error {
	stmt := `DELETE FROM lsts WHERE id=?`
	_, err := db.Exec(stmt, lid)
	if err != nil {
		return fmt.Errorf("failed to delete list %d: %w", lid, err)
	}
	return nil
}

func GetLst(db *sqlx.DB, lid uint64) (*Lst, error) {
	stmt := `SELECT * FROM lsts WHERE id = ?`
	result := &Lst{}
	err := db.Get(result, stmt, lid)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get list %d: %w", lid, err)
	}
	return handleGetResult(result, err)
}

func UpdateLst(db *sqlx.DB, lst *Lst) error {
	stmt := `UPDATE lsts SET name=? WHERE id=?`
	_, err := db.Exec(stmt, lst.Name, lst.Id)
	if err != nil {
		return fmt.Errorf("failed to update list %d: %w", lst.Id, err)
	}
	return nil
}

func CreateLstEntity(db *sqlx.DB, entity *LstEntity) error {
	abs, err := filepath.Abs(entity.ParentDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for parent dir %q: %w", entity.ParentDir, err)
	}
	entity.ParentDir = abs

	stmt := `INSERT INTO lst_entities(id, lst_id, name, parent_dir) VALUES(:id, :lst_id, :name, :parent_dir)`
	res, err := db.NamedExec(stmt, &entity)
	if err != nil {
		return fmt.Errorf("failed to create list entity for list %d in %q: %w", entity.LstId, entity.ParentDir, err)
	}
	return handleInsertWithId(res, err, func(id int64) { entity.Id.Scan(id) })
}

func DelLstEntity(db *sqlx.DB, id int) error {
	stmt := `DELETE FROM lst_entities WHERE id=?`
	_, err := db.Exec(stmt, id)
	if err != nil {
		return fmt.Errorf("failed to delete list entity %d: %w", id, err)
	}
	return nil
}

func GetLstEntity(db *sqlx.DB, id int) (*LstEntity, error) {
	stmt := `SELECT * FROM lst_entities WHERE id=?`
	result := &LstEntity{}
	err := db.Get(result, stmt, id)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get list entity %d: %w", id, err)
	}
	return handleGetResult(result, err)
}

func LocateLstEntity(db *sqlx.DB, lid int64, parentDir string) (*LstEntity, error) {
	parentDir, err := filepath.Abs(parentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %q: %w", parentDir, err)
	}

	stmt := `SELECT * FROM lst_entities WHERE lst_id=? AND parent_dir=?`
	result := &LstEntity{}
	err = db.Get(result, stmt, lid, parentDir)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to locate list entity for list %d in %q: %w", lid, parentDir, err)
	}
	return handleGetResult(result, err)
}
func UpdateLstEntity(db *sqlx.DB, entity *LstEntity) error {
	stmt := `UPDATE lst_entities SET name=? WHERE id=?`
	_, err := db.Exec(stmt, entity.Name, entity.Id.Int32)
	if err != nil {
		return fmt.Errorf("failed to update list entity %d: %w", entity.Id.Int32, err)
	}
	return nil
}

func SetUserEntityLatestReleaseTime(db *sqlx.DB, id int, t time.Time) error {
	stmt := `UPDATE user_entities SET latest_release_time=? WHERE id=?`
	result, err := db.Exec(stmt, t, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no user entity found with id %d", id)
	}
	return nil
}

func ClearUserEntityLatestReleaseTime(db *sqlx.DB, id int) error {
	stmt := `UPDATE user_entities SET latest_release_time=NULL WHERE id=?`
	result, err := db.Exec(stmt, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no user entity found with id %d", id)
	}
	return nil
}

func RecordUserPreviousName(db *sqlx.DB, uid uint64, name string, screenName string) error {
	stmt := `INSERT INTO user_previous_names(uid, screen_name, name, record_date) VALUES(?, ?, ?, ?)`
	_, err := db.Exec(stmt, uid, screenName, name, time.Now())
	if err != nil {
		return fmt.Errorf("failed to record previous name for user %d (%s -> %s): %w", uid, screenName, name, err)
	}
	return nil
}

func CreateUserLink(db *sqlx.DB, lnk *UserLink) error {
	stmt := `INSERT INTO user_links(user_id, name, parent_lst_entity_id) VALUES(:user_id, :name, :parent_lst_entity_id)`
	res, err := db.NamedExec(stmt, lnk)
	if err != nil {
		return fmt.Errorf("failed to create user link for user %d in list entity %d: %w", lnk.Uid, lnk.ParentLstEntityId, err)
	}
	return handleInsertWithId(res, err, func(id int64) { lnk.Id.Scan(id) })
}

func DelUserLink(db *sqlx.DB, id int32) error {
	stmt := `DELETE FROM user_links WHERE id = ?`
	_, err := db.Exec(stmt, id)
	if err != nil {
		return fmt.Errorf("failed to delete user link %d: %w", id, err)
	}
	return nil
}

func GetUserLinks(db *sqlx.DB, uid uint64) ([]*UserLink, error) {
	stmt := `SELECT * FROM user_links WHERE user_id = ?`
	res := []*UserLink{}
	err := db.Select(&res, stmt, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get user links for user %d: %w", uid, err)
	}
	return res, nil
}

func GetUserLink(db *sqlx.DB, uid uint64, parentLstEntityId int32) (*UserLink, error) {
	stmt := `SELECT * FROM user_links WHERE user_id = ? AND parent_lst_entity_id = ?`
	res := &UserLink{}
	err := db.Get(res, stmt, uid, parentLstEntityId)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get user link for user %d in list entity %d: %w", uid, parentLstEntityId, err)
	}
	return handleGetResult(res, err)
}

func UpdateUserLink(db *sqlx.DB, id int32, name string) error {
	stmt := `UPDATE user_links SET name = ? WHERE id = ?`
	_, err := db.Exec(stmt, name, id)
	if err != nil {
		return fmt.Errorf("failed to update user link %d: %w", id, err)
	}
	return nil
}

// GetUserLinksByLstEntityId 获取指定列表实体下的所有用户链接
func GetUserLinksByLstEntityId(db interface {
	Select(dest interface{}, query string, args ...interface{}) error
}, lstEntityId int) ([]*UserLink, error) {
	stmt := `SELECT * FROM user_links WHERE parent_lst_entity_id = ?`
	res := []*UserLink{}
	err := db.Select(&res, stmt, lstEntityId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user links for list entity %d: %w", lstEntityId, err)
	}
	return res, nil
}
