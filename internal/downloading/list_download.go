package downloading

import (
	"context"

	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/unkmonster/tmd/internal/database"
	"github.com/unkmonster/tmd/internal/entity"
	"github.com/unkmonster/tmd/internal/naming"
	"github.com/unkmonster/tmd/internal/twitter"
	"github.com/unkmonster/tmd/internal/utils"
)

func syncList(db *sqlx.DB, list *twitter.List) error {
	listdb, err := database.GetLst(db, list.Id)
	if err != nil {
		return err
	}
	if listdb == nil {
		return database.CreateLst(db, &database.Lst{Id: list.Id, Name: list.Name, OwnerId: list.Creator.Id})
	}
	return database.UpdateLst(db, &database.Lst{Id: list.Id, Name: list.Name, OwnerId: list.Creator.Id})
}

func syncListAndGetMembers(ctx context.Context, client *resty.Client, db *sqlx.DB, lst twitter.ListBase, dir string) ([]userInListEntity, error) {
	if v, ok := lst.(*twitter.List); ok {
		if err := syncList(db, v); err != nil {
			return nil, err
		}
	}

	expectedTitle := naming.NewListNamingFromBase(lst).SanitizedTitle()
	ent, err := entity.NewListEntity(db, lst.GetId(), dir)
	if err != nil {
		return nil, err
	}
	if err := entity.Sync(ent, expectedTitle); err != nil {
		return nil, err
	}

	membersResult, err := lst.GetMembers(ctx, client)
	if err != nil {
		return nil, err
	}

	eid, err := ent.Id()
	if err != nil {
		return nil, err
	}

	members := membersResult.Users
	if len(members) == 0 {
		return nil, nil
	}

	memberIDs := utils.ExtractIDs(members, func(u *twitter.User) uint64 { return u.Id })
	database.MarkListMembersAccessibleByIDs(db, memberIDs)
	syncManager := NewListSyncManager(db)
	if err := syncManager.SyncListMembers(ctx, eid, lst.Title(), memberIDs); err != nil {
		log.Warnln("failed to sync list members for", lst.Title(), ":", err)
	}

	packgedUsers := make([]userInListEntity, 0, len(members))
	for _, user := range members {
		packgedUsers = append(packgedUsers, userInListEntity{user: user, leid: &eid})
	}
	return packgedUsers, nil
}
