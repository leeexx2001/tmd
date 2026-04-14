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
