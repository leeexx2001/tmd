package downloading

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	"github.com/unkmonster/tmd/internal/database"
	"github.com/unkmonster/tmd/internal/twitter"
)

// UserArgs 用户参数
type UserArgs struct {
	ID         []uint64
	ScreenName []string
}

func (u *UserArgs) Set(str string) error {
	if u.ID == nil {
		u.ID = make([]uint64, 0)
		u.ScreenName = make([]string, 0)
	}

	id, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		str, _ = strings.CutPrefix(str, "@")
		u.ScreenName = append(u.ScreenName, str)
	} else {
		u.ID = append(u.ID, id)
	}
	return nil
}

func (u *UserArgs) String() string {
	return fmt.Sprintf("ids=%v screenNames=%v", u.ID, u.ScreenName)
}

// ResolveUsers 解析用户参数为 User 列表
func (u *UserArgs) ResolveUsers(ctx context.Context, client *resty.Client, db *sqlx.DB) ([]*twitter.User, error) {
	users := []*twitter.User{}
	for _, id := range u.ID {
		usr, uid, err := twitter.GetUserById(ctx, client, id)
		if err != nil {
			database.MarkUserInaccessible(db, uid, "")
			return nil, err
		}
		users = append(users, usr)
	}

	for _, screenName := range u.ScreenName {
		usr, uid, err := twitter.GetUserByScreenName(ctx, client, screenName)
		if err != nil {
			database.MarkUserInaccessible(db, uid, screenName)
			return nil, err
		}
		users = append(users, usr)
	}
	return users, nil
}

// ListArgs 列表参数
type ListArgs struct {
	ID []uint64
}

func (l *ListArgs) Set(str string) error {
	if l.ID == nil {
		l.ID = make([]uint64, 0)
	}
	id, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return err
	}
	l.ID = append(l.ID, id)
	return nil
}

func (l *ListArgs) String() string {
	return fmt.Sprintf("%v", l.ID)
}

// ResolveLists 解析列表参数为 List 列表
func (l *ListArgs) ResolveLists(ctx context.Context, client *resty.Client) ([]twitter.ListBase, error) {
	lists := []twitter.ListBase{}
	for _, id := range l.ID {
		list, err := twitter.GetLst(ctx, client, id)
		if err != nil {
			return nil, err
		}
		lists = append(lists, list)
	}
	return lists, nil
}

// JsonPathsArgs JSON 路径参数
type JsonPathsArgs struct {
	Paths []string
}

func (j *JsonPathsArgs) Set(str string) error {
	if j.Paths == nil {
		j.Paths = make([]string, 0)
	}
	j.Paths = append(j.Paths, str)
	return nil
}

func (j *JsonPathsArgs) String() string {
	return strings.Join(j.Paths, ",")
}

func (j *JsonPathsArgs) GetPaths() []string {
	return j.Paths
}
