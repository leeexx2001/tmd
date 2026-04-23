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
	log.Infoln("[syncListAndGetMembers] start for list:", lst.Title(), "id:", lst.GetId())
	
	if v, ok := lst.(*twitter.List); ok {
		log.Infoln("[syncListAndGetMembers] syncing list to database:", v.Name)
		if err := syncList(db, v); err != nil {
			log.Errorln("[syncListAndGetMembers] failed to sync list:", err)
			return nil, err
		}
		log.Infoln("[syncListAndGetMembers] list synced successfully")
	}

	expectedTitle := naming.NewListNamingFromBase(lst).SanitizedTitle()
	log.Infoln("[syncListAndGetMembers] expected title:", expectedTitle)
	
	ent, err := entity.NewListEntity(db, lst.GetId(), dir)
	if err != nil {
		log.Errorln("[syncListAndGetMembers] failed to create list entity:", err)
		return nil, err
	}
	log.Infoln("[syncListAndGetMembers] list entity created")
	
	if err := entity.Sync(ent, expectedTitle); err != nil {
		log.Errorln("[syncListAndGetMembers] failed to sync entity:", err)
		return nil, err
	}
	log.Infoln("[syncListAndGetMembers] entity synced")

	log.Infoln("[syncListAndGetMembers] getting members from Twitter API...")
	membersResult, err := lst.GetMembers(ctx, client)
	if err != nil {
		log.Errorln("[syncListAndGetMembers] failed to get members:", err)
		return nil, err
	}
	log.Infoln("[syncListAndGetMembers] got", len(membersResult.Users), "members from API")

	eid, err := ent.Id()
	if err != nil {
		log.Errorln("[syncListAndGetMembers] failed to get entity id:", err)
		return nil, err
	}
	log.Infoln("[syncListAndGetMembers] entity id:", eid)

	members := membersResult.Users
	if len(members) == 0 {
		log.Warnln("[syncListAndGetMembers] no members found in list:", lst.Title())
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
