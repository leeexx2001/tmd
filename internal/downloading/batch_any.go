package downloading

import (
	"context"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	"github.com/unkmonster/tmd/internal/downloader"
	"github.com/unkmonster/tmd/internal/twitter"
	log "github.com/sirupsen/logrus"
)

func BatchDownloadAny(ctx context.Context, client *resty.Client, db *sqlx.DB, lists []twitter.ListBase, users []*twitter.User, dir string, realDir string, autoFollow bool, additional []*resty.Client, dwn downloader.Downloader, fileWriter downloader.FileWriter) ([]*TweetInEntity, error) {
	log.Infoln("[BatchDownloadAny] start collecting users, lists count:", len(lists), "users count:", len(users))
	packgedUsers := make([]userInListEntity, 0)
	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	for i, lst := range lists {
		log.Infoln("[BatchDownloadAny] processing list", i, ":", lst.Title())
		wg.Add(1)
		go func(index int, lst twitter.ListBase) {
			defer wg.Done()
			log.Infoln("[BatchDownloadAny] getting members for list:", lst.Title())
			res, err := syncListAndGetMembers(ctx, client, db, lst, dir)
			if err != nil {
				log.Errorln("[BatchDownloadAny] failed to get members for list:", lst.Title(), "error:", err)
				cancel(err)
				return
			}
			log.Infoln("[BatchDownloadAny] list", lst.Title(), "has", len(res), "members")
			mtx.Lock()
			defer mtx.Unlock()
			packgedUsers = append(packgedUsers, res...)
		}(i, lst)
	}
	log.Infoln("[BatchDownloadAny] waiting for all lists to complete...")
	wg.Wait()
	log.Infoln("[BatchDownloadAny] all lists completed")
	if err := context.Cause(ctx); err != nil {
		return nil, err
	}

	for _, usr := range users {
		packgedUsers = append(packgedUsers, userInListEntity{user: usr, leid: nil})
	}

	log.Debugln("collected users:", len(packgedUsers))
	return BatchUserDownload(ctx, client, db, packgedUsers, realDir, autoFollow, additional, dwn, fileWriter)
}
