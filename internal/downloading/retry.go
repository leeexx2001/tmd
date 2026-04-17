package downloading

import (
	"context"

	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/unkmonster/tmd/internal/downloader"
)

func RetryFailedTweets(ctx context.Context, dumper *TweetDumper, db *sqlx.DB, client *resty.Client, dwn downloader.Downloader, fileWriter downloader.FileWriter) error {
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
		// 只保留还有URL需要下载的推文
		if len(leg.Tweet.Urls) > 0 {
			toretry = append(toretry, leg)
		}
	}

	if len(toretry) == 0 {
		log.Infoln("no tweets need to be retried")
		dumper.Clear()
		return nil
	}

	log.Infof("retrying %d tweets with %d total media(s)", len(toretry), countTotalUrls(toretry))

	newFails := BatchDownloadTweet(ctx, client, true, dwn, fileWriter, toretry...)
	dumper.Clear()
	for _, pt := range newFails {
		te := pt.(*TweetInEntity)
		eid, err := te.Entity.Id()
		if err != nil {
			log.Warnln("failed to get entity id:", err)
			continue
		}

		// 只保留还有URL需要下载的推文
		if len(te.Tweet.Urls) > 0 {
			dumper.Push(eid, te.Tweet)
			log.Warnf("tweet %d still has %d media(s) to download", te.Tweet.Id, len(te.Tweet.Urls))
		} else {
			log.Infof("tweet %d all media downloaded successfully on retry", te.Tweet.Id)
		}
	}

	return nil
}

// countTotalUrls 统计所有推文中需要下载的URL总数
func countTotalUrls(tweets []PackagedTweet) int {
	count := 0
	for _, pt := range tweets {
		count += len(pt.GetTweet().Urls)
	}
	return count
}
