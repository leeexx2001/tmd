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
