package downloading

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gookit/color"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/unkmonster/tmd/internal/downloader"
	"github.com/unkmonster/tmd/internal/naming"
	"github.com/unkmonster/tmd/internal/twitter"
	"github.com/unkmonster/tmd/internal/utils"
)

func saveTweetJson(dir string, tweet *twitter.Tweet, namingObj *naming.TweetNaming) {
	if dir == "" || tweet == nil {
		return
	}

	go func() {
		defer utils.RecoverWithLog("saveTweetJson")

		loongDir := filepath.Join(dir, ".loongtweet")
		os.MkdirAll(loongDir, 0755)

		jsonPath, err := namingObj.FilePath(loongDir, ".json")
		if err != nil {
			return
		}

		if tweet.RawJSON == "" {
			return
		}

		data, err := cleanTweetJson([]byte(tweet.RawJSON))
		if err != nil {
			return
		}

		formatted, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return
		}

		if err := os.WriteFile(jsonPath, formatted, 0645); err == nil {
			os.Chtimes(jsonPath, time.Time{}, tweet.CreatedAt)
		}
	}()
}

func saveLoongTweet(dir string, tweet *twitter.Tweet, namingObj *naming.TweetNaming) {
	if dir == "" || tweet == nil {
		return
	}

	go func() {
		defer utils.RecoverWithLog("saveLoongTweet")

		loongDir := filepath.Join(dir, ".loongtweet")
		os.MkdirAll(loongDir, 0755)

		txtPath, err := namingObj.FilePath(loongDir, ".txt")
		if err != nil {
			return
		}

		var screenName, text string
		var tweetID uint64
		var createdAt time.Time
		var mediaCount int

		if tweet.RawJSON != "" {
			result := gjson.Parse(tweet.RawJSON)
			tweetID = result.Get("rest_id").Uint()
			if createdAtStr := result.Get("legacy.created_at").String(); createdAtStr != "" {
				createdAt, _ = time.Parse(time.RubyDate, createdAtStr)
			}
			noteText := result.Get("note_tweet.note_tweet_results.result.text").String()
			if noteText != "" {
				text = noteText
			} else {
				text = result.Get("legacy.full_text").String()
			}
			screenName = result.Get("core.user_results.result.legacy.screen_name").String()
			if screenName == "" {
				screenName = "unknown"
			}
			if media := result.Get("legacy.extended_entities.media"); media.Exists() {
				mediaCount = len(media.Array())
			}
		} else {
			tweetID = tweet.Id
			createdAt = tweet.CreatedAt
			text = tweet.Text
			if tweet.Creator != nil && tweet.Creator.ScreenName != "" {
				screenName = tweet.Creator.ScreenName
			} else {
				screenName = "unknown"
			}
			mediaCount = len(tweet.Urls)
		}

		txtContent := fmt.Sprintf("time:%s\nurl:https://x.com/%s/status/%d\nmedia:%d\n\n%s",
			createdAt.Format("2006-01-02T15:04:05"),
			screenName,
			tweetID,
			mediaCount,
			text)

		if err := os.WriteFile(txtPath, []byte(txtContent), 0645); err == nil {
			os.Chtimes(txtPath, time.Time{}, createdAt)
		}
	}()
}

func cleanTweetJson(raw []byte) (any, error) {
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}

	m, ok := data.(map[string]any)
	if !ok {
		return data, nil
	}

	if legacy, ok := m["legacy"].(map[string]any); ok {
		delete(legacy, "user_id_str")
		delete(legacy, "id_str")

		if entities, ok := legacy["entities"].(map[string]any); ok {
			delete(entities, "media")
			delete(entities, "symbols")
			delete(entities, "timestamps")
			delete(entities, "urls")
			delete(entities, "user_mentions")
		}
	}

	if core, ok := m["core"].(map[string]any); ok {
		if userResults, ok := core["user_results"].(map[string]any); ok {
			if result, ok := userResults["result"].(map[string]any); ok {
				delete(result, "id")
				if userLegacy, ok := result["legacy"].(map[string]any); ok {
					if profileImg, ok := userLegacy["profile_image_url_https"].(string); ok {
						profileImg = strings.Replace(profileImg, "_normal", "", 1)
						profileImg = strings.Replace(profileImg, "_bigger", "", 1)
						profileImg = strings.Replace(profileImg, "_mini", "", 1)
						userLegacy["profile_image_url_https"] = profileImg
					}
				}
			}
		}
	}

	cleanMediaRecursive(m)

	return m, nil
}

func cleanMediaRecursive(data any) {
	switch v := data.(type) {
	case map[string]any:
		if media, ok := v["extended_entities"].(map[string]any); ok {
			if mediaList, ok := media["media"].([]any); ok {
				for _, item := range mediaList {
					if m, ok := item.(map[string]any); ok {
						delete(m, "media_results")

						if originalInfo, ok := m["original_info"].(map[string]any); ok {
							delete(originalInfo, "focus_rects")
						}

						if features, ok := m["features"].(map[string]any); ok {
							delete(features, "large")
							delete(features, "medium")
							delete(features, "small")
						}

						if mediaType, ok := m["type"].(string); ok && mediaType == "photo" {
							if rawUrl, ok := m["media_url_https"].(string); ok {
								if strings.Contains(rawUrl, "twimg.com") {
									m["media_url_https"] = rawUrl + "?name=4096x4096"
								}
							}
						}
					}
				}
			}
		}
		for _, val := range v {
			cleanMediaRecursive(val)
		}
	case []any:
		for _, item := range v {
			cleanMediaRecursive(item)
		}
	}
}

func downloadTweetMedia(ctx context.Context, client *resty.Client, dir string, tweet *twitter.Tweet, skipLoongTweet bool, dwn downloader.Downloader) error {
	var creatorTitle string
	if tweet.Creator != nil {
		creatorTitle = tweet.Creator.Title()
	} else {
		creatorTitle = "unknown"
	}
	tweetNaming := naming.NewTweetNaming(tweet.Text, tweet.Id, creatorTitle)

	if !skipLoongTweet {
		saveTweetJson(dir, tweet, tweetNaming)
		saveLoongTweet(dir, tweet, tweetNaming)
	}

	reqs := make([]downloader.DownloadRequest, 0, len(tweet.Urls))
	for i, u := range tweet.Urls {
		ext, err := utils.GetExtFromUrl(u)
		if err != nil {
			return err
		}

		// 为同一推文的多个媒体文件生成唯一文件名
		var path string
		if len(tweet.Urls) > 1 {
			path, err = tweetNaming.FilePath(dir, fmt.Sprintf("_%d%s", i+1, ext))
		} else {
			path, err = tweetNaming.FilePath(dir, ext)
		}
		if err != nil {
			return err
		}

		queryParams := make(map[string]string)
		if !strings.Contains(u, "tweet_video") && !strings.Contains(u, "video.twimg.com") {
			queryParams["name"] = "4096x4096"
		}

		reqs = append(reqs, downloader.DownloadRequest{
			Context:     ctx,
			Client:      client,
			URL:         u,
			Destination: path,
			Options: downloader.DownloadOptions{
				QueryParams:   queryParams,
				SkipUnchanged: false,
				CreateVersion: false,
				SetModTime:    &tweet.CreatedAt,
			},
		})
	}

	results, err := dwn.BatchDownload(ctx, reqs)
	if err != nil {
		return err
	}

	for _, result := range results {
		if !result.Success && !result.Skipped {
			return result.Error
		}
	}

	fmt.Printf("%s\n", color.FgLightMagenta.Render(tweetNaming.LogFormat()))
	return nil
}

func tweetDownloader(client *resty.Client, config *workerConfig, errch chan<- PackagedTweet, twech <-chan PackagedTweet) {
	var pt PackagedTweet
	var ok bool

	defer config.wg.Done()
	defer func() {
		if p := recover(); p != nil {
			config.cancel(fmt.Errorf("%v", p))
			log.Errorln("✗ [downloading] - panic:", p)

			if pt != nil {
				errch <- pt
			}
			for pt := range twech {
				errch <- pt
			}
		}
	}()

	for {
		select {
		case pt, ok = <-twech:
			if !ok {
				return
			}
		case <-config.ctx.Done():
			for pt := range twech {
				errch <- pt
			}
			return
		}

		path := pt.GetPath()
		if path == "" {
			if !config.skipLoongTweet {
				tweet := pt.GetTweet()
				if tweet != nil && tweet.Creator != nil {
					if tie, ok := pt.(TweetInEntity); ok && tie.Entity != nil {
						parentDir := tie.Entity.ParentDir()
						if parentDir != "" {
							userNaming := naming.NewUserNaming(tweet.Creator.Name, tweet.Creator.ScreenName)
							userDirName := userNaming.SanitizedTitle()
							userDir := filepath.Join(parentDir, userDirName)
							tweetNaming := naming.NewTweetNaming(tweet.Text, tweet.Id, tweet.Creator.Title())
							saveTweetJson(userDir, tweet, tweetNaming)
							saveLoongTweet(userDir, tweet, tweetNaming)
						}
					}
				}
			}
			errch <- pt
			continue
		}
		err := downloadTweetMedia(config.ctx, client, path, pt.GetTweet(), config.skipLoongTweet, config.downloader)
		if err != nil && !utils.IsStatusCode(err, 404) && !utils.IsStatusCode(err, 403) {
			errch <- pt
		}

		if errors.Is(err, syscall.ENOSPC) {
			config.cancel(err)
		}
	}
}

func BatchDownloadTweet(ctx context.Context, client *resty.Client, skipLoongTweet bool, dwn downloader.Downloader, pts ...PackagedTweet) []PackagedTweet {
	if len(pts) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancelCause(ctx)

	var errChan = make(chan PackagedTweet)
	var tweetChan = make(chan PackagedTweet, len(pts))
	var wg sync.WaitGroup
	var numRoutine = min(len(pts), MaxDownloadRoutine)

	for _, pt := range pts {
		tweetChan <- pt
	}
	close(tweetChan)

	config := workerConfig{
		ctx:            ctx,
		cancel:         cancel,
		wg:             &wg,
		skipLoongTweet: skipLoongTweet,
		downloader:     dwn,
	}
	for i := 0; i < numRoutine; i++ {
		wg.Add(1)
		go tweetDownloader(client, &config, errChan, tweetChan)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	errors := []PackagedTweet{}
	for pt := range errChan {
		errors = append(errors, pt)
	}
	return errors
}
