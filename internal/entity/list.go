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

func (le *ListEntity) Create(name string) error {
	le.record.Name = name
	path, _ := le.Path()
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
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
