// Package downloading 实现 Twitter 特定的下载业务逻辑。
// 依赖 downloader 包提供的基础设施，处理推文媒体下载、用户/列表同步等业务功能。
package downloading

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gookit/color"
	"github.com/jmoiron/sqlx"
	"github.com/panjf2000/ants/v2"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/unkmonster/tmd/internal/database"
	"github.com/unkmonster/tmd/internal/downloader"
	"github.com/unkmonster/tmd/internal/naming"
	"github.com/unkmonster/tmd/internal/twitter"
	"github.com/unkmonster/tmd/internal/utils"
)

type PackgedTweet interface {
	GetTweet() *twitter.Tweet
	GetPath() string
}

type TweetInDir struct {
	tweet *twitter.Tweet
	path  string
}

func (pt TweetInDir) GetTweet() *twitter.Tweet {
	return pt.tweet
}

func (pt TweetInDir) GetPath() string {
	return pt.path
}

// saveTweetJson 将推文完整信息保存为格式化 JSON 文件到 .loongtweet 子目录
// 独立于 downloadTweetMedia，确保即使下载失败也能记录推文信息
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

// saveLoongTweet 保存推文文本到 .loongtweet 子目录（人类可读的txt格式）
// 与 saveTweetJson 同时生成，提供简洁的文本格式便于阅读
// 数据来源统一：RawJSON 存在时完全从中提取，否则使用 tweet 结构体字段
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

// cleanTweetJson 清理推文 JSON 中的冗余字段
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

// cleanMediaRecursive 递归清理 media 对象中的冗余字段
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

// 任何一个 url 下载失败直接返回
// skipLoongTweet: 如果为 true，不生成 .loongtweet 文件（用于恢复下载）
func downloadTweetMedia(ctx context.Context, client *resty.Client, dir string, tweet *twitter.Tweet, skipLoongTweet bool, dwn downloader.Downloader) error {
	var creatorTitle string
	if tweet.Creator != nil {
		creatorTitle = tweet.Creator.Title()
	} else {
		creatorTitle = "unknown"
	}
	tweetNaming := naming.NewTweetNaming(tweet.Text, tweet.Id, creatorTitle)

	// 保存推文 JSON 和 TXT
	if !skipLoongTweet {
		saveTweetJson(dir, tweet, tweetNaming)
		saveLoongTweet(dir, tweet, tweetNaming)
	}

	// 构建批量下载请求
	reqs := make([]downloader.DownloadRequest, 0, len(tweet.Urls))
	for _, u := range tweet.Urls {
		ext, err := utils.GetExtFromUrl(u)
		if err != nil {
			return err
		}

		path, err := tweetNaming.FilePath(dir, ext)
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

	// 批量下载
	results, err := dwn.BatchDownload(ctx, reqs)
	if err != nil {
		return err
	}

	// 检查结果
	for _, result := range results {
		if !result.Success && !result.Skipped {
			return result.Error
		}
	}

	fmt.Printf("%s\n", color.FgLightMagenta.Render(tweetNaming.LogFormat()))
	return nil
}

var MaxDownloadRoutine int

// map[user_id]*UserEntity 记录本次程序运行已同步过的用户
var syncedUsers sync.Map

func init() {
	MaxDownloadRoutine = min(100, runtime.GOMAXPROCS(0)*10)
}

type workerConfig struct {
	ctx            context.Context
	wg             *sync.WaitGroup
	cancel         context.CancelCauseFunc
	skipLoongTweet bool // 恢复下载时不生成 .loongtweet 文件
	downloader     downloader.Downloader
}

// 负责下载推文，保证 tweet chan 内的推文要么下载成功，要么推送至 error chan
func tweetDownloader(client *resty.Client, config *workerConfig, errch chan<- PackgedTweet, twech <-chan PackgedTweet) {
	var pt PackgedTweet
	var ok bool

	defer config.wg.Done()
	defer func() {
		if p := recover(); p != nil {
			config.cancel(fmt.Errorf("%v", p)) // panic 取消上下文，防止生产者死锁
			log.Errorln("✗ [downloading] - panic:", p)

			if pt != nil {
				errch <- pt // push 正下载的推文
			}
			// 确保只有1个协程的情况下，未能下载完毕的推文仍然会全部推送到 errch
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
			// 即使 path 为空，也尝试生成 .loongtweet 文件
			// 使用 Creator 信息构建用户目录
			if !config.skipLoongTweet {
				tweet := pt.GetTweet()
				if tweet != nil && tweet.Creator != nil {
					if tie, ok := pt.(TweetInEntity); ok && tie.Entity != nil {
						parentDir := tie.Entity.ParentDir()
						if parentDir != "" {
							// 使用 Creator.Name 和 Creator.ScreenName 构建目录名
							userNaming := naming.NewUserNaming(tweet.Creator.Name, tweet.Creator.ScreenName)
							userDirName := userNaming.SanitizedTitle()
							userDir := filepath.Join(parentDir, userDirName)
							// 创建推文命名对象用于保存文件
							tweetNaming := naming.NewTweetNaming(tweet.Text, tweet.Id, tweet.Creator.Title())
							saveTweetJson(userDir, tweet, tweetNaming)  // JSON格式（完整数据）
							saveLoongTweet(userDir, tweet, tweetNaming) // TXT格式（人类可读）
						}
					}
				}
			}
			errch <- pt
			continue
		}
		err := downloadTweetMedia(config.ctx, client, path, pt.GetTweet(), config.skipLoongTweet, config.downloader)
		// 403: Dmcaed
		if err != nil && !utils.IsStatusCode(err, 404) && !utils.IsStatusCode(err, 403) {
			errch <- pt
		}

		// cancel context and exit if no disk space
		if errors.Is(err, syscall.ENOSPC) {
			config.cancel(err)
		}
	}
}

// 批量下载推文并返回下载失败的推文，可以保证推文被成功下载或被返回
// skipLoongTweet: 如果为 true，不生成 .loongtweet 文件（用于恢复下载）
func BatchDownloadTweet(ctx context.Context, client *resty.Client, skipLoongTweet bool, dwn downloader.Downloader, pts ...PackgedTweet) []PackgedTweet {
	if len(pts) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancelCause(ctx)

	var errChan = make(chan PackgedTweet)
	var tweetChan = make(chan PackgedTweet, len(pts))
	var wg sync.WaitGroup // number of working goroutines
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

	errors := []PackgedTweet{}
	for pt := range errChan {
		errors = append(errors, pt)
	}
	return errors
}

// 更新数据库中对用户的记录
func syncUser(db *sqlx.DB, user *twitter.User) error {
	renamed := false
	isNew := false
	usrdb, err := database.GetUserById(db, user.Id)
	if err != nil {
		return err
	}

	if usrdb == nil {
		isNew = true
		usrdb = &database.User{}
		usrdb.Id = user.Id
	} else {
		renamed = usrdb.Name != user.Name || usrdb.ScreenName != user.ScreenName
	}

	usrdb.FriendsCount = user.FriendsCount
	usrdb.IsProtected = user.IsProtected
	usrdb.Name = user.Name
	usrdb.ScreenName = user.ScreenName

	if isNew {
		err = database.CreateUser(db, usrdb)
	} else {
		err = database.UpdateUser(db, usrdb)
	}
	if err != nil {
		return err
	}
	if renamed || isNew {
		err = database.RecordUserPreviousName(db, user.Id, user.Name, user.ScreenName)
	}
	return err
}

func getTweetAndUpdateLatestReleaseTime(ctx context.Context, client *resty.Client, user *twitter.User, entity *UserEntity) ([]*twitter.Tweet, error) {
	tweets, err := user.GetMeidas(ctx, client, &utils.TimeRange{Min: entity.LatestReleaseTime()})
	if err != nil || len(tweets) == 0 {
		return nil, err
	}
	if err := entity.SetLatestReleaseTime(tweets[0].CreatedAt); err != nil {
		return nil, err
	}
	return tweets, nil
}

func DownloadUser(ctx context.Context, db *sqlx.DB, client *resty.Client, user *twitter.User, dir string, dwn downloader.Downloader) ([]PackgedTweet, error) {
	if user.Blocking || user.Muting {
		return nil, nil
	}

	_, loaded := syncedUsers.Load(user.Id)
	if loaded {
		log.Debugln("○", user.Title(), "-", "skiped downloaded user")
		return nil, nil
	}
	entity, err := syncUserAndEntity(db, user, dir)
	if err != nil {
		return nil, err
	}

	syncedUsers.Store(user.Id, entity)
	tweets, err := getTweetAndUpdateLatestReleaseTime(ctx, client, user, entity)
	if err != nil || len(tweets) == 0 {
		return nil, err
	}

	// 打包推文
	pts := make([]PackgedTweet, 0, len(tweets))
	for _, tw := range tweets {
		pts = append(pts, TweetInEntity{Tweet: tw, Entity: entity})
	}

	// 正常下载，生成 .loongtweet 文件（skipLoongTweet=false）
	return BatchDownloadTweet(ctx, client, false, dwn, pts...), nil
}

func syncUserAndEntity(db *sqlx.DB, user *twitter.User, dir string) (*UserEntity, error) {
	if err := syncUser(db, user); err != nil {
		return nil, err
	}
	userNaming := naming.NewUserNaming(user.Name, user.ScreenName)
	expectedTitle := userNaming.SanitizedTitle()

	entity, err := NewUserEntity(db, user.Id, dir)
	if err != nil {
		return nil, err
	}
	if err = syncPath(entity, expectedTitle); err != nil {
		return nil, err
	}
	return entity, nil
}

type TweetInEntity struct {
	Tweet  *twitter.Tweet
	Entity *UserEntity
}

func (pt TweetInEntity) GetTweet() *twitter.Tweet {
	return pt.Tweet
}

func (pt TweetInEntity) GetPath() string {
	defer func() {
		recover()
	}()

	path, err := pt.Entity.Path()
	if err != nil {
		return ""
	}
	return path
}

const userTweetRateLimit = 1500
const userTweetMaxConcurrent = 35 // avoid DownstreamOverCapacityError

// var syncedListUsers = make(map[uint64]map[int64]struct{})
var syncedListUsers sync.Map //leid -> uid -> struct{}

// 需要请求多少次时间线才能获取完毕用户的推文？
func calcUserDepth(exist int, total int) int {
	if exist >= total {
		return 1
	}

	miss := total - exist
	depth := miss / twitter.AvgTweetsPerPage
	if miss%twitter.AvgTweetsPerPage != 0 {
		depth++
	}
	if exist == 0 {
		depth++ //对于新用户，需要多获取一个空页
	}
	return depth
}

type userInLstEntity struct {
	user *twitter.User
	leid *int
}

func shouldIngoreUser(user *twitter.User) bool {
	return user.Blocking || user.Muting
}

func BatchUserDownload(ctx context.Context, client *resty.Client, db *sqlx.DB, users []userInLstEntity, dir string, autoFollow bool, additional []*resty.Client, dwn downloader.Downloader) ([]*TweetInEntity, error) {
	if len(users) == 0 {
		return nil, nil
	}

	uidToUser := make(map[uint64]*twitter.User)
	for _, u := range users {
		uidToUser[u.user.Id] = u.user
	}

	// channels
	tweetChan := make(chan PackgedTweet, MaxDownloadRoutine)
	errChan := make(chan PackgedTweet)
	// WG
	prodwg := sync.WaitGroup{}
	conswg := sync.WaitGroup{}
	// ctx
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	symlinkWarnCount := 0
	symlinkWarnMu := sync.Mutex{}

	panicHandler := func() {
		if r := recover(); r != nil {
			cancel(fmt.Errorf("%v", r))
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			fmt.Printf("Recovered from panic: %v\n%s\n", r, buf[:n])
		}
	}

	missingTweets := 0
	depthByEntity := make(map[*UserEntity]int)
	// 大顶堆，以用户深度
	userEntityHeap := utils.NewHeap(func(lhs, rhs *UserEntity) bool {
		luser, ruser := uidToUser[lhs.Uid()], uidToUser[rhs.Uid()]
		lOnlyMater := luser.IsProtected && luser.Followstate == twitter.FS_FOLLOWING
		rOnlyMaster := ruser.IsProtected && ruser.Followstate == twitter.FS_FOLLOWING

		if lOnlyMater == rOnlyMaster {
			return depthByEntity[lhs] > depthByEntity[rhs]
		}
		return lOnlyMater // 优先让 master 获取只有他能看到的
	})

	start := time.Now()
	deepest := 0

	// pre-process
	func() {
		defer panicHandler()
		log.Infoln("start pre processing users")

		for _, userInLST := range users {
			var pathEntity *UserEntity
			var err error
			user := userInLST.user
			leid := userInLST.leid

			if shouldIngoreUser(user) {
				continue
			}

			pe, loaded := syncedUsers.Load(user.Id)
			if !loaded {
				pathEntity, err = syncUserAndEntity(db, user, dir)
				if err != nil {
					log.Warnln("✗", user.Title(), "-", "failed to update user or entity", err)
					continue
				}
				syncedUsers.Store(user.Id, pathEntity)

				// 同步所有现存的指向此用户的符号链接
				upath, _ := pathEntity.Path()
				linkds, err := database.GetUserLinks(db, user.Id)
				if err != nil {
					log.Warnln("✗", user.Title(), "-", "failed to get links to user:", err)
				}
				for _, linkd := range linkds {
					if err = updateUserLink(linkd, db, upath); err != nil {
						symlinkWarnMu.Lock()
						symlinkWarnCount++
						if symlinkWarnCount == 1 {
							log.Warnln("✗", user.Title(), "-", "symlink permission denied (suppressing further warnings)")
						}
						symlinkWarnMu.Unlock()
					}
					sl, _ := syncedListUsers.LoadOrStore(int(linkd.ParentLstEntityId), &sync.Map{})
					syncedList := sl.(*sync.Map)
					syncedList.Store(user.Id, struct{}{})
				}

				// 计算深度
				if user.MediaCount != 0 && user.IsVisiable() {
					missingTweets += max(0, user.MediaCount-int(pathEntity.record.MediaCount.Int32))
					depthByEntity[pathEntity] = calcUserDepth(int(pathEntity.record.MediaCount.Int32), user.MediaCount)
					userEntityHeap.Push(pathEntity)
					deepest = max(deepest, depthByEntity[pathEntity])
				}

				// 自动关注
				if user.IsProtected && user.Followstate == twitter.FS_UNFOLLOW && autoFollow {
					if err := twitter.FollowUser(ctx, client, user); err != nil {
						log.Warnln("✗", user.Title(), "-", "failed to follow user:", err)
					} else {
						log.Debugln("✓", user.Title(), "-", "follow request has been sent")
					}
				}
			} else {
				pathEntity = pe.(*UserEntity)
			}

			// 即便同步一个用户时也同步了所有指向此用户的链接，
			// 但此用户仍可能会是一个新的 "列表-用户"，所以判断此用户链接是否同步过，
			// 如果否，那么创建一个属于此列表的用户链接
			if leid == nil {
				continue
			}
			sl, _ := syncedListUsers.LoadOrStore(*leid, &sync.Map{})
			syncedList := sl.(*sync.Map)
			_, loaded = syncedList.LoadOrStore(user.Id, struct{}{})
			if loaded {
				continue
			}

			// 为当前列表的新用户创建符号链接
			upath, _ := pathEntity.Path()
			var linkname = pathEntity.Name()

			curlink := &database.UserLink{}
			curlink.Name = linkname
			curlink.ParentLstEntityId = int32(*leid)
			curlink.Uid = user.Id

			linkpath, err := curlink.Path(db)
			if err == nil {
				if err = os.Symlink(upath, linkpath); err == nil || os.IsExist(err) {
					err = database.CreateUserLink(db, curlink)
				}
			}
			if err != nil {
				symlinkWarnMu.Lock()
				symlinkWarnCount++
				if symlinkWarnCount == 1 {
					log.Warnln("✗", user.Title(), "-", "symlink permission denied (suppressing further warnings)")
				}
				symlinkWarnMu.Unlock()
			}
		}
	}()

	if userEntityHeap.Empty() {
		return nil, nil
	}
	log.Debugln("preprocessing finish, elapsed:", time.Since(start))
	log.Debugln("real members:", userEntityHeap.Size())
	log.Debugln("missing tweets:", missingTweets)
	log.Debugln("deepest:", deepest)
	if symlinkWarnCount > 0 {
		log.Warnf("symlink permission denied: %d errors suppressed (run as admin to enable symlinks)", symlinkWarnCount)
	}

	producer := func(entity *UserEntity) {
		defer prodwg.Done()
		defer panicHandler()

		user := uidToUser[entity.Uid()]

		// 使用 MFQ 客户端选择
		cli := twitter.SelectClientMFQ(ctx, client, additional, user, twitter.UserMediaPath())
		if ctx.Err() != nil {
			userEntityHeap.Push(entity)
			return
		}
		if cli == nil {
			userEntityHeap.Push(entity)
			cancel(fmt.Errorf("no client available"))
			return
		}

		tweets, err := user.GetMeidas(ctx, cli, &utils.TimeRange{Min: entity.LatestReleaseTime()})
		if err == twitter.ErrWouldBlock {
			userEntityHeap.Push(entity)
			return
		}
		if v, ok := err.(*twitter.TwitterApiError); ok {
			// 客户端不再可用
			switch v.Code {
			case twitter.ErrExceedPostLimit:
				twitter.SetClientError(cli, fmt.Errorf("reached the limit for seeing posts today"))
				userEntityHeap.Push(entity)
				return
			case twitter.ErrAccountLocked:
				twitter.SetClientError(cli, fmt.Errorf("account is locked"))
				userEntityHeap.Push(entity)
				return
			}
		}
		if ctx.Err() != nil {
			userEntityHeap.Push(entity)
			return
		}
		if err != nil {
			log.Warnln("✗", entity.Name(), "-", "failed to get user medias:", err)
			return
		}

		if len(tweets) == 0 {
			if err := database.UpdateUserEntityMediCount(db, entity.Id(), user.MediaCount); err != nil {
				log.Panicln("✗", entity.Name(), "-", "failed to update user medias count:", err)
			}
			return
		}

		// 确保该用户所有推文已推送并更新用户推文状态
		for _, tw := range tweets {
			pt := TweetInEntity{Tweet: tw, Entity: entity}
			select {
			case tweetChan <- &pt:
			case <-ctx.Done():
				return // 防止无消费者导致死锁
			}
		}

		if err := database.UpdateUserEntityTweetStat(db, entity.Id(), tweets[0].CreatedAt, user.MediaCount); err != nil {
			// 影响程序的正确性，必须 Panic
			log.Panicln("✗", entity.Name(), "-", "failed to update user tweets stat:", err)
		}
	}

	// launch worker
	config := workerConfig{
		ctx:        ctx,
		wg:         &conswg,
		cancel:     cancel,
		downloader: dwn,
	}
	for i := 0; i < MaxDownloadRoutine; i++ {
		conswg.Add(1)
		go tweetDownloader(client, &config, errChan, tweetChan)
	}

	producerPool, err := ants.NewPool(min(userTweetMaxConcurrent, userEntityHeap.Size()))
	if err != nil {
		return nil, err
	}
	defer producerPool.Release()

	//closer
	go func() {
		// 按批次调用生产者
		for !userEntityHeap.Empty() && ctx.Err() == nil {
			selected := []int{}
			for count := 0; count < userTweetRateLimit && ctx.Err() == nil; {
				if userEntityHeap.Empty() {
					break
				}

				entity := userEntityHeap.Peek()
				depth := depthByEntity[entity]
				if depth > userTweetRateLimit {
					log.Warnln("user depth exceeds limit:", entity.Name(), "- depth:", depth)
					userEntityHeap.Pop()
					continue
				}

				if depth+count > userTweetRateLimit {
					break
				}

				prodwg.Add(1)
				producerPool.Submit(func() {
					producer(entity)
				})
				selected = append(selected, depth)

				count += depth
				//delete(depthByEntity, entity)
				userEntityHeap.Pop()
			}
			log.Debugln(selected)
			prodwg.Wait()
		}
		close(tweetChan)
		log.Debugf("getting tweets completed, elapsed time: %v", time.Since(start))

		conswg.Wait()
		close(errChan)
	}()

	fails := []*TweetInEntity{}
	for pt := range errChan {
		fails = append(fails, pt.(*TweetInEntity))
	}
	log.Debugf("%d users unable to start", userEntityHeap.Size())
	return fails, context.Cause(ctx)
}

func downloadList(ctx context.Context, client *resty.Client, db *sqlx.DB, list twitter.ListBase, dir string, realDir string, autoFollow bool, additional []*resty.Client, dwn downloader.Downloader) ([]*TweetInEntity, error) {
	expectedTitle := utils.WinFileName(list.Title())
	entity, err := NewListEntity(db, list.GetId(), dir)
	if err != nil {
		return nil, err
	}
	if err := syncPath(entity, expectedTitle); err != nil {
		return nil, err
	}

	members, err := list.GetMembers(ctx, client)
	if err != nil || len(members) == 0 {
		return nil, err
	}

	eid := entity.Id()
	log.Debugln("members:", len(members))
	packgedUsers := make([]userInLstEntity, len(members))
	for i, user := range members {
		packgedUsers[i] = userInLstEntity{user: user, leid: &eid}
	}
	// 列表下载时，强制启用自动关注
	return BatchUserDownload(ctx, client, db, packgedUsers, realDir, true, additional, dwn)
}

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

func DownloadList(ctx context.Context, client *resty.Client, db *sqlx.DB, list twitter.ListBase, dir string, realDir string, autoFollow bool, additional []*resty.Client, dwn downloader.Downloader) ([]*TweetInEntity, error) {
	tlist, ok := list.(*twitter.List)
	if ok {
		if err := syncList(db, tlist); err != nil {
			return nil, err
		}
	}
	return downloadList(ctx, client, db, list, dir, realDir, autoFollow, additional, dwn)
}

func syncLstAndGetMembers(ctx context.Context, client *resty.Client, db *sqlx.DB, lst twitter.ListBase, dir string) ([]userInLstEntity, error) {
	if v, ok := lst.(*twitter.List); ok {
		if err := syncList(db, v); err != nil {
			return nil, err
		}
	}

	// update lst path and record
	expectedTitle := utils.WinFileName(lst.Title())
	entity, err := NewListEntity(db, lst.GetId(), dir)
	if err != nil {
		return nil, err
	}
	if err := syncPath(entity, expectedTitle); err != nil {
		return nil, err
	}

	// get all members
	members, err := lst.GetMembers(ctx, client)
	if err != nil || len(members) == 0 {
		return nil, err
	}

	// 同步列表成员，清理已删除的用户链接
	eid := entity.Id()
	memberIDs := make([]uint64, len(members))
	for i, u := range members {
		memberIDs[i] = u.Id
	}
	syncManager := NewListSyncManager(db)
	if err := syncManager.SyncListMembers(ctx, eid, lst.Title(), memberIDs); err != nil {
		log.Warnln("failed to sync list members for", lst.Title(), ":", err)
	}

	// bind lst entity to users for creating symlink
	packgedUsers := make([]userInLstEntity, 0, len(members))
	for _, user := range members {
		packgedUsers = append(packgedUsers, userInLstEntity{user: user, leid: &eid})
	}
	return packgedUsers, nil
}

func BatchDownloadAny(ctx context.Context, client *resty.Client, db *sqlx.DB, lists []twitter.ListBase, users []*twitter.User, dir string, realDir string, autoFollow bool, additional []*resty.Client, dwn downloader.Downloader) ([]*TweetInEntity, error) {
	log.Debugln("start collecting users")
	packgedUsers := make([]userInLstEntity, 0)
	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	for _, lst := range lists {
		wg.Add(1)
		go func(lst twitter.ListBase) {
			defer wg.Done()
			res, err := syncLstAndGetMembers(ctx, client, db, lst, dir)
			if err != nil {
				cancel(err)
			}
			log.Debugf("members of %s: %d", lst.Title(), len(res))
			mtx.Lock()
			defer mtx.Unlock()
			packgedUsers = append(packgedUsers, res...)
		}(lst)
	}
	wg.Wait()
	if err := context.Cause(ctx); err != nil {
		return nil, err
	}

	for _, usr := range users {
		packgedUsers = append(packgedUsers, userInLstEntity{user: usr, leid: nil})
	}

	log.Debugln("collected users:", len(packgedUsers))
	return BatchUserDownload(ctx, client, db, packgedUsers, realDir, autoFollow, additional, dwn)
}

// MarkedUserInfo 标记用户为已下载的结果信息
type MarkedUserInfo struct {
	UserID     uint64 `json:"user_id"`
	ScreenName string `json:"screen_name"`
	EntityID   int    `json:"entity_id"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// MarkUsersAsDownloaded 将用户标记为已下载，不下载内容，只更新数据库中的 latest_release_time
// 返回标记的用户信息列表，包含 entity_id 等详细信息
// markTimeStr: 时间戳字符串，格式为 "2006-01-02T15:04:05"
//   - 未提供(""): 使用当前时间
//   - 空值("null"/"NULL"/"nil"): 设置 latest_release_time 为 NULL（全量下载）
//   - 指定时间: 使用指定的时间
func MarkUsersAsDownloaded(ctx context.Context, client *resty.Client, db *sqlx.DB, lists []twitter.ListBase, users []*twitter.User, dir string, markTimeStr string) ([]MarkedUserInfo, error) {
	// 解析时间戳（使用本地时区）
	var timestamp *time.Time
	if markTimeStr == "" {
		// 未提供时间，使用当前时间
		now := time.Now()
		timestamp = &now
		log.Infoln("marking users as downloaded, timestamp:", timestamp.Format(time.RFC3339))
	} else if strings.ToLower(markTimeStr) == "null" || strings.ToLower(markTimeStr) == "nil" {
		// 显式设置 NULL，用于全量下载
		timestamp = nil
		log.Infoln("marking users as downloaded, timestamp: NULL (full download)")
	} else {
		// 使用本地时区解析，确保用户输入的时间被正确识别为本地时间
		loc, locErr := time.LoadLocation("Local")
		if locErr != nil {
			loc = time.UTC
		}
		parsedTime, err := time.ParseInLocation("2006-01-02T15:04:05", markTimeStr, loc)
		if err != nil {
			return nil, fmt.Errorf("invalid mark-time format '%s', expected: 2006-01-02T15:04:05 (example: 2024-01-15T10:30:00) or 'null' for full download: %v", markTimeStr, err)
		}
		timestamp = &parsedTime
		log.Infoln("marking users as downloaded, timestamp:", timestamp.Format(time.RFC3339))
	}

	var results []MarkedUserInfo
	var successCount, failCount int

	// 处理列表中的用户
	for _, lst := range lists {
		if err := context.Cause(ctx); err != nil {
			return results, err
		}

		if lst == nil {
			continue
		}

		members, err := lst.GetMembers(ctx, client)
		if err != nil {
			// 检查是否是列表不存在或无法访问的错误
			errStr := err.Error()
			if strings.Contains(errStr, "does not exist or is not accessible") ||
				strings.Contains(errStr, "unable to get timeline data") {
				return nil, fmt.Errorf("list %s does not exist or is not accessible", lst.Title())
			}
			log.Warnln("✗", lst.Title(), "-", "failed to get list members:", err)
			continue
		}
		for _, user := range members {
			if err := context.Cause(ctx); err != nil {
				return results, err
			}

			if user == nil {
				continue
			}

			info := markSingleUserWithInfo(db, user, dir, timestamp)
			results = append(results, info)
			if info.Success {
				successCount++
			} else {
				failCount++
			}
		}
	}

	// 处理直接指定的用户
	for _, user := range users {
		if err := context.Cause(ctx); err != nil {
			return results, err
		}

		if user == nil {
			continue
		}

		info := markSingleUserWithInfo(db, user, dir, timestamp)
		results = append(results, info)
		if info.Success {
			successCount++
		} else {
			failCount++
		}
	}

	log.Infoln("finished marking users as downloaded, success:", successCount, "failed:", failCount)
	return results, nil
}

// markSingleUserWithInfo 标记单个用户为已下载并返回详细信息
func markSingleUserWithInfo(db *sqlx.DB, user *twitter.User, dir string, timestamp *time.Time) (info MarkedUserInfo) {
	// 防御性检查：确保 user 不为 nil
	if user == nil {
		info.Success = false
		info.Error = "user is nil"
		return info
	}

	info = MarkedUserInfo{
		UserID:     user.Id,
		ScreenName: user.ScreenName,
		Success:    false,
	}

	// 捕获可能的 panic，增加健壮性
	defer func() {
		if r := recover(); r != nil {
			info.Success = false
			info.Error = fmt.Sprintf("panic: %v", r)
			log.Errorf("[markSingleUserWithInfo] panic recovered: %v", r)
		}
	}()

	// 同步用户和实体（与正常下载使用相同的逻辑）
	entity, err := syncUserAndEntity(db, user, dir)
	if err != nil {
		info.Error = fmt.Sprintf("failed to sync user and entity: %v", err)
		log.Warnln("✗", user.Title(), "-", "failed to mark user:", err)
		return info
	}

	// 设置 latest_release_time
	if timestamp == nil {
		// 设置为 NULL，用于全量下载
		if err := entity.ClearLatestReleaseTime(); err != nil {
			info.Error = fmt.Sprintf("failed to clear latest release time: %v", err)
			log.Warnln("✗", user.Title(), "-", "failed to clear latest release time:", err)
			return info
		}
		log.Infoln("✓", user.Title(), "-", "cleared latest release time for full download")
	} else {
		// 设置为指定时间
		if err := entity.SetLatestReleaseTime(*timestamp); err != nil {
			info.Error = fmt.Sprintf("failed to set latest release time: %v", err)
			log.Warnln("✗", user.Title(), "-", "failed to set latest release time:", err)
			return info
		}
	}

	info.Success = true
	info.EntityID = entity.Id()
	log.Infoln("✓", user.Title(), "-", "marked as downloaded")
	return info
}
