package downloading

import (
	"context"

	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/unkmonster/tmd/internal/downloader"
)

func RetryFailedTweets(ctx context.Context, dumper *TweetDumper, db *sqlx.DB, client *resty.Client, dwn downloader.Downloader) error {
	if dumper.Count() == 0 {
		return nil
	}

	log.Infoln("starting to retry failed tweets")
	legacy, err := dumper.GetTotal(db)
	if err != nil {
		return err
	}

	toretry := make([]PackagedTweet, 0, len(legacy))
	for _, leg := range legacy {
		toretry = append(toretry, leg)
	}

	newFails := BatchDownloadTweet(ctx, client, true, dwn, toretry...)
	dumper.Clear()
	for _, pt := range newFails {
		te := pt.(*TweetInEntity)
		eid, err := te.Entity.Id()
		if err != nil {
			log.Warnln("failed to get entity id:", err)
			continue
		}
		dumper.Push(eid, te.Tweet)
	}

	return nil
}
