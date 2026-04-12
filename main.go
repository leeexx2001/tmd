package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gookit/color"
	"github.com/jmoiron/sqlx"
	"github.com/natefinch/lumberjack"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
	"github.com/unkmonster/tmd/internal/database"
	"github.com/unkmonster/tmd/internal/downloader"
	"github.com/unkmonster/tmd/internal/downloading"
	"github.com/unkmonster/tmd/internal/naming"
	"github.com/unkmonster/tmd/internal/profile"
	"github.com/unkmonster/tmd/internal/twitter"
	"github.com/unkmonster/tmd/internal/utils"
	"gopkg.in/yaml.v3"
)

type Cookie struct {
	AuthToken string `yaml:"auth_token"`
	Ct0       string `yaml:"ct0"`
}

type Config struct {
	RootPath           string `yaml:"root_path"`
	Cookie             Cookie `yaml:"cookie"`
	MaxDownloadRoutine int    `yaml:"max_download_routine"`
	MaxFileNameLen     int    `yaml:"max_file_name_len"` // 文件名长度限制（0=使用默认值250）
}

type userArgs struct {
	id         []uint64
	screenName []string
}

func (u *userArgs) GetUser(ctx context.Context, client *resty.Client) ([]*twitter.User, error) {
	users := []*twitter.User{}
	for _, id := range u.id {
		usr, err := twitter.GetUserById(ctx, client, id)
		if err != nil {
			return nil, err
		}
		users = append(users, usr)
	}

	for _, screenName := range u.screenName {
		usr, err := twitter.GetUserByScreenName(ctx, client, screenName)
		if err != nil {
			return nil, err
		}
		users = append(users, usr)
	}
	return users, nil
}

func (u *userArgs) Set(str string) error {
	if u.id == nil {
		u.id = make([]uint64, 0)
		u.screenName = make([]string, 0)
	}

	id, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		str, _ := strings.CutPrefix(str, "@")
		u.screenName = append(u.screenName, str)
	} else {
		u.id = append(u.id, id)
	}
	return nil
}

func (u *userArgs) String() string {
	return "string"
}

type intArgs struct {
	id []uint64
}

func (l *intArgs) Set(str string) error {
	if l.id == nil {
		l.id = make([]uint64, 0)
	}

	id, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return err
	}
	l.id = append(l.id, id)
	return nil
}

func (a *intArgs) String() string {
	return "string array"
}

type ListArgs struct {
	intArgs
}

type jsonPathsArgs struct {
	paths []string
}

func (j *jsonPathsArgs) Set(str string) error {
	if j.paths == nil {
		j.paths = make([]string, 0)
	}
	j.paths = append(j.paths, str)
	return nil
}

func (j *jsonPathsArgs) String() string {
	return "json file paths"
}

func (j *jsonPathsArgs) GetPaths() []string {
	return j.paths
}

func (l ListArgs) GetList(ctx context.Context, client *resty.Client) ([]*twitter.List, error) {
	lists := []*twitter.List{}
	for _, id := range l.id {
		list, err := twitter.GetLst(ctx, client, id)
		if err != nil {
			return nil, err
		}
		lists = append(lists, list)
	}
	return lists, nil
}

type Task struct {
	users []*twitter.User
	lists []twitter.ListBase
}

func printTask(task *Task) {
	if len(task.users) != 0 {
		fmt.Printf("users: %d\n", len(task.users))
	}
	for _, u := range task.users {
		fmt.Printf("    - %s\n", u.Title())
	}
	if len(task.lists) != 0 {
		fmt.Printf("lists: %d\n", len(task.lists))
	}
	for _, l := range task.lists {
		fmt.Printf("    - %s\n", l.Title())
	}
}

func MakeTask(ctx context.Context, client *resty.Client, usrArgs userArgs, listArgs ListArgs, follArgs userArgs) (*Task, error) {
	task := Task{}
	task.users = make([]*twitter.User, 0)
	task.lists = make([]twitter.ListBase, 0)

	users, err := usrArgs.GetUser(ctx, client)
	if err != nil {
		return nil, err
	}
	task.users = append(task.users, users...)

	lists, err := listArgs.GetList(ctx, client)
	if err != nil {
		return nil, err
	}
	for _, list := range lists {
		task.lists = append(task.lists, list)
	}

	// fo
	users, err = follArgs.GetUser(ctx, client)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		task.lists = append(task.lists, user.Following())
	}
	return &task, nil
}

type storePath struct {
	root   string
	users  string
	data   string
	db     string
	errorj string
}

func newStorePath(root string) (*storePath, error) {
	ph := storePath{}
	ph.root = root
	ph.users = filepath.Join(root, "users")
	ph.data = filepath.Join(root, ".data")

	ph.db = filepath.Join(ph.data, "foo.db")
	ph.errorj = filepath.Join(ph.data, "errors.json")

	// ensure folder exist
	err := os.Mkdir(ph.root, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}

	err = os.Mkdir(ph.users, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}

	err = os.Mkdir(ph.data, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}
	return &ph, nil
}

func initLogger(dbg bool, logFile io.Writer) {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:    true,
		FullTimestamp:  true,
		DisableSorting: true,
		PadLevelText:   false,
	})

	if dbg {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	log.AddHook(lfshook.NewHook(logFile, nil))
}

func main() {
	//flags
	var usrArgs userArgs
	var listArgs ListArgs
	var follArgs userArgs
	var confArg bool
	var dbg bool
	var autoFollow bool
	var noRetry bool
	var markDownloaded bool
	var markTime string
	var noProfile bool
	var profileUsers userArgs
	var profileList ListArgs
	var jsonArgs jsonPathsArgs

	flag.BoolVar(&confArg, "conf", false, "reconfigure")
	flag.Var(&usrArgs, "user", "download tweets from the user specified by user_id/screen_name since the last download")
	flag.Var(&listArgs, "list", "batch download each member from list specified by list_id")
	flag.Var(&follArgs, "foll", "batch download each member followed by the user specified by user_id/screen_name")
	flag.BoolVar(&dbg, "dbg", false, "display debug message")
	flag.BoolVar(&autoFollow, "auto-follow", false, "send follow request automatically to protected users (enabled by default for list downloads)")
	flag.BoolVar(&noRetry, "no-retry", false, "quickly exit without retrying failed tweets")
	flag.BoolVar(&markDownloaded, "mark-downloaded", false, "mark users as downloaded without downloading content (sets latest_release_time to now)")
	flag.StringVar(&markTime, "mark-time", "", "timestamp for mark-downloaded (format: 2006-01-02T15:04:05), empty means now")
	flag.BoolVar(&noProfile, "noprofile", false, "skip downloading user profiles")
	flag.Var(&profileUsers, "profile-user", "download profile for specified user (can be used multiple times)")
	flag.Var(&profileList, "profile-list", "download profiles for all members in the specified list")
	flag.Var(&jsonArgs, "json", "download media from JSON file(s) exported by other tools (supports raw API JSON and formatted .loongtweet JSON)")
	flag.Parse()

	var err error

	// context
	ctx, cancel := context.WithCancel(context.Background())

	var homepath string
	if runtime.GOOS == "windows" {
		homepath = os.Getenv("appdata")
	} else {
		homepath = os.Getenv("HOME")
	}
	if homepath == "" {
		panic("failed to get home path from env")
	}

	appRootPath := filepath.Join(homepath, ".tmd2")
	confPath := filepath.Join(appRootPath, "conf.yaml")
	cliLogPath := filepath.Join(appRootPath, "client.log")
	logPath := filepath.Join(appRootPath, "tmd2.log")
	additionalCookiesPath := filepath.Join(appRootPath, "additional_cookies.yaml")
	if err = os.MkdirAll(appRootPath, 0755); err != nil {
		log.Fatalln("failed to make app dir", err)
	}

	// init logger with rotation (使用 lumberjack，参考 TMD 控制器: 5MB max, 2 backups)
	logWriter := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    5, // 5MB
		MaxBackups: 2,
		MaxAge:     7, // 保留7天
		Compress:   false,
	}
	defer logWriter.Close()
	initLogger(dbg, logWriter)

	// report at exit
	defer func() {
		if dbg {
			twitter.ReportRequestCount()
		}
	}()

	// read/write config
	conf, err := readConf(confPath)
	if os.IsNotExist(err) || confArg {
		conf, err = promptConfig(confPath)
		if err != nil {
			log.Fatalln("config failure with", err)
		}
	}
	if err != nil {
		log.Fatalln("failed to load config:", err)
	}
	if confArg {
		log.Println("config done")
		return
	}
	log.Infoln("config is loaded")
	if conf.MaxDownloadRoutine > 0 {
		downloading.MaxDownloadRoutine = conf.MaxDownloadRoutine
	}
	// 设置文件名长度限制（范围：50-250，0=使用默认值）
	if conf.MaxFileNameLen > 0 {
		naming.SetMaxFileNameLen(conf.MaxFileNameLen)
		log.Infoln("max file name length set to:", naming.MaxFileNameLen)
	}

	// ensure store path exist
	pathHelper, err := newStorePath(conf.RootPath)
	if err != nil {
		log.Fatalln("failed to make store dir:", err)
	}

	// sign in
	client, screenName, err := twitter.Login(ctx, conf.Cookie.AuthToken, conf.Cookie.Ct0)
	if err != nil {
		log.Fatalln("failed to login:", err)
	}
	twitter.EnableRateLimit(client)
	if dbg {
		twitter.EnableRequestCounting(client)
	}
	log.Infoln("signed in as:", color.FgLightBlue.Render(screenName))

	// load additional cookies
	cookies, err := readAdditionalCookies(additionalCookiesPath)
	if err != nil {
		log.Warnln("failed to load additional cookies:", err)
	}
	log.Debugln("loaded additional cookies:", len(cookies))
	addtional := batchLogin(ctx, dbg, cookies, screenName)

	// set clients logger
	cliLogFile, err := os.OpenFile(cliLogPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalln("failed to create log file:", err)
	}
	defer cliLogFile.Close()
	setClientLogger(client, cliLogFile)
	for _, cli := range addtional {
		setClientLogger(cli, cliLogFile)
	}

	// load previous tweets
	dumper := downloading.NewDumper()
	err = dumper.Load(pathHelper.errorj)
	if err != nil {
		log.Fatalln("failed to load previous tweets", err)
	}
	log.Infoln("loaded previous failed tweets:", dumper.Count())

	// collect tasks
	task, err := MakeTask(ctx, client, usrArgs, listArgs, follArgs)
	if err != nil {
		log.Fatalln("failed to parse cmd args:", err)
	}

	// connect db
	db, err := connectDatabase(pathHelper.db)
	if err != nil {
		log.Fatalln("failed to connect to database:", err)
	}
	defer db.Close()
	log.Infoln("database is connected")

	// listen signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer close(sigChan)
	defer signal.Stop(sigChan)
	go func() {
		sig, ok := <-sigChan
		if ok {
			log.Warnln("[listener] caught signal:", sig)
			cancel()
		}
	}()

	// 创建版本管理器
	versionManager := downloader.NewVersionManager(".versions")

	// 创建文件写入器
	fileWriter := downloader.NewFileWriter(versionManager)

	// 创建下载器
	dwn := downloader.NewDownloader(fileWriter)

	// dump failed tweets at exit
	var todump = make([]*downloading.TweetInEntity, 0)
	defer func() {
		dumper.Dump(pathHelper.errorj)
		log.Infof("%d tweets have been dumped and will be downloaded the next time the program runs", dumper.Count())
	}()

	// retry failed tweets at exit
	defer func() {
		for _, te := range todump {
			dumper.Push(te.Entity.Id(), te.Tweet)
		}
		// 如果手动取消，不尝试重试，快速终止进程
		if ctx.Err() != context.Canceled && !noRetry {
			retryFailedTweets(ctx, dumper, db, client, dwn)
		}
	}()

	// do job - 推文下载先执行
	if len(task.users) == 0 && len(task.lists) == 0 && len(jsonArgs.GetPaths()) == 0 {
		// 没有推文下载任务，直接执行 profile 下载（如果有）
		goto handleProfile
	}
	log.Infoln("start working for...")
	printTask(task)

	// 如果指定了 --mark-downloaded，只更新数据库时间戳，不下载内容
	if markDownloaded {
		results, err := downloading.MarkUsersAsDownloaded(ctx, client, db, task.lists, task.users, pathHelper.users, markTime)
		if err != nil {
			log.Errorln("failed to mark users as downloaded:", err)
			os.Exit(1)
		}
		// 输出结果供外部程序解析（JSON格式）
		if len(results) > 0 {
			fmt.Println("\n=== MARK_DOWNLOADED_RESULTS ===")
			for _, r := range results {
				status := "OK"
				if !r.Success {
					status = "FAIL"
				}
				fmt.Printf("ENTITY_ID:%d|USER_ID:%d|SCREEN_NAME:%s|STATUS:%s\n", r.EntityID, r.UserID, r.ScreenName, status)
			}
			fmt.Println("=== END_RESULTS ===")
		}
	} else if len(jsonArgs.GetPaths()) > 0 {
		// 从 JSON 文件下载媒体（不需要 API 调用）
		log.Infof("downloading from %d JSON file(s)...", len(jsonArgs.GetPaths()))
		results := downloading.DownloadJsonDir(ctx, client, pathHelper.root, dwn, jsonArgs.GetPaths()...)
		var successCount, failCount int
		for _, r := range results {
			if r.Success {
				successCount++
				log.Infof("✓ %s: %d tweets processed in %v", filepath.Base(r.Path), r.TweetCount, r.Duration)
			} else {
				failCount++
				log.Errorf("✗ %s: %v", filepath.Base(r.Path), r.Error)
			}
		}
		log.Infof("JSON download completed: %d success, %d failed", successCount, failCount)
	} else {
		todump, err = downloading.BatchDownloadAny(ctx, client, db, task.lists, task.users, pathHelper.root, pathHelper.users, autoFollow, addtional, dwn)
		if err != nil {
			log.Errorln("failed to download:", err)
		}
	}

handleProfile:
	// handle profile download - 推文下载完成后执行
	// 默认下载profile，除非指定 --noprofile
	// Profile 下载使用独立的 context，不影响推文下载的速率限制状态
	shouldDownloadProfile := !noProfile && (len(usrArgs.screenName) > 0 || len(listArgs.id) > 0 || len(follArgs.screenName) > 0)

	if shouldDownloadProfile || len(profileUsers.screenName) > 0 || len(profileList.id) > 0 {
		profileCtx, profileCancel := context.WithCancel(context.Background())
		profileDone := make(chan struct{})
		go func() {
			defer close(profileDone)
			// skipAPIFetch = shouldDownloadProfile，因为从推文下载中已经获取了用户数据
			handleProfileDownload(profileCtx, client, addtional, pathHelper.users, profileUsers, profileList, task, db, shouldDownloadProfile, dwn, fileWriter)
		}()
		// 等待 profile 下载完成或主 context 被取消
		select {
		case <-profileDone:
			// profile 下载完成
		case <-ctx.Done():
			// 主 context 被取消，取消 profile 下载
			profileCancel()
			<-profileDone
		}
		profileCancel()
	}
}

func setClientLogger(client *resty.Client, out io.Writer) {
	logger := log.New()
	logger.SetLevel(log.InfoLevel)
	logger.SetOutput(out)
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp:  true,
		DisableQuote:   true,
		DisableSorting: true,
		PadLevelText:   false,
	})
	client.SetLogger(logger)
}

func connectDatabase(path string) (*sqlx.DB, error) {
	ex, err := utils.PathExists(path)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&busy_timeout=2147483647", path)
	db, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	database.CreateTables(db)
	//db.SetMaxOpenConns(1)
	if !ex {
		log.Debugln("created new db file", path)
	}
	return db, nil
}

func readConf(path string) (*Config, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var result Config
	err = yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func writeConf(path string, conf *Config) error {
	file, err := os.OpenFile(path, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, bytes.NewReader(data))
	return err
}

func promptConfig(saveto string) (*Config, error) {
	// 先尝试读取现有配置，以便保留未修改的字段
	conf, err := readConf(saveto)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，创建新配置
			fmt.Println("Config file not found, creating new configuration...")
		} else {
			// 配置文件存在但读取失败（损坏或格式错误），备份原文件
			backupPath := saveto + ".backup." + strconv.FormatInt(time.Now().Unix(), 10)
			if renameErr := os.Rename(saveto, backupPath); renameErr != nil {
				fmt.Printf("Warning: failed to read existing config (%v)\n", err)
				fmt.Printf("Failed to backup config file: %v\n", renameErr)
				fmt.Println("Starting fresh without backup...")
			} else {
				fmt.Printf("Warning: existing config file is corrupted (%v)\n", err)
				fmt.Printf("Original config has been backed up to: %s\n", backupPath)
				fmt.Println("Creating new configuration...")
			}
		}
		conf = &Config{}
	}

	scan := bufio.NewScanner(os.Stdin)

	// 辅助函数：如果输入为空则保留原值
	getInputOrDefault := func(prompt string, defaultValue string) string {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
		scan.Scan()
		input := scan.Text()
		if strings.TrimSpace(input) == "" {
			return defaultValue
		}
		return input
	}

	// 存储目录
	storePath := getInputOrDefault("enter storage dir", conf.RootPath)
	if strings.TrimSpace(storePath) == "" {
		return nil, fmt.Errorf("storage dir cannot be empty")
	}
	// 确保路径可用
	err = os.MkdirAll(storePath, 0755)
	if err != nil {
		return nil, err
	}
	storePath, err = filepath.Abs(storePath)
	if err != nil {
		return nil, err
	}
	conf.RootPath = storePath

	// Auth Token
	conf.Cookie.AuthToken = getInputOrDefault("enter auth_token", conf.Cookie.AuthToken)

	// Ct0
	conf.Cookie.Ct0 = getInputOrDefault("enter ct0", conf.Cookie.Ct0)

	// Max Download Routine
	routineStr := getInputOrDefault("enter max download routine", strconv.Itoa(conf.MaxDownloadRoutine))
	if strings.TrimSpace(routineStr) != "" {
		routine, err := strconv.Atoi(routineStr)
		if err != nil {
			return nil, fmt.Errorf("invalid max download routine: %v", err)
		}
		conf.MaxDownloadRoutine = routine
	}

	return conf, writeConf(saveto, conf)
}

func retryFailedTweets(ctx context.Context, dumper *downloading.TweetDumper, db *sqlx.DB, client *resty.Client, dwn downloader.Downloader) error {
	if dumper.Count() == 0 {
		return nil
	}

	log.Infoln("starting to retry failed tweets")
	legacy, err := dumper.GetTotal(db)
	if err != nil {
		return err
	}

	toretry := make([]downloading.PackgedTweet, 0, len(legacy))
	for _, leg := range legacy {
		toretry = append(toretry, leg)
	}

	// 恢复下载时不生成 .loongtweet 文件（skipLoongTweet=true）
	newFails := downloading.BatchDownloadTweet(ctx, client, true, dwn, toretry...)
	dumper.Clear()
	for _, pt := range newFails {
		te := pt.(*downloading.TweetInEntity)
		dumper.Push(te.Entity.Id(), te.Tweet)
	}

	return nil
}

func readAdditionalCookies(path string) ([]*Cookie, error) {
	res := []*Cookie{}
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return res, yaml.Unmarshal(data, &res)
}

func batchLogin(ctx context.Context, dbg bool, cookies []*Cookie, master string) []*resty.Client {
	if len(cookies) == 0 {
		return nil
	}

	added := sync.Map{}
	msgs := make([]string, len(cookies))
	clients := []*resty.Client{}
	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}
	added.Store(master, struct{}{})

	for i, cookie := range cookies {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			cli, sn, err := twitter.Login(ctx, cookie.AuthToken, cookie.Ct0)
			if _, loaded := added.LoadOrStore(sn, struct{}{}); loaded {
				msgs[index] = "    - ? repeated\n"
				return
			}

			if err != nil {
				msgs[index] = fmt.Sprintf("    - ? %v\n", err)
				return
			}
			twitter.EnableRateLimit(cli)
			if dbg {
				twitter.EnableRequestCounting(cli)
			}
			mtx.Lock()
			defer mtx.Unlock()
			clients = append(clients, cli)
			msgs[index] = fmt.Sprintf("    - %s\n", sn)
		}(i)
	}

	wg.Wait()
	log.Infoln("loaded additional accounts:", len(clients))
	for _, msg := range msgs {
		fmt.Print(msg)
	}
	return clients
}

func handleProfileDownload(ctx context.Context, client *resty.Client, additional []*resty.Client, usersPath string, profileUsers userArgs, profileList ListArgs, task *Task, db *sqlx.DB, skipAPIFetch bool, dwn downloader.Downloader, fileWriter downloader.FileWriter) {
	clients := make([]*resty.Client, 0)
	clients = append(clients, client)
	clients = append(clients, additional...)

	storage, err := profile.NewFileStorageManager(usersPath)
	if err != nil {
		log.Fatalln("failed to create profile storage:", err)
	}

	profileDownloader := profile.NewProfileDownloaderWithDB(nil, storage, clients, db, dwn, fileWriter)

	requests := make([]profile.DownloadRequest, 0)

	// 首先处理 task.users（从推文下载中获取的用户数据），这样有预获取数据的用户会优先被处理
	if len(task.users) > 0 {
		for _, user := range task.users {
			req := profile.DownloadRequest{
				ScreenName: user.ScreenName,
				UserTitle:  user.Title(), // 用于目录名: Name(ScreenName)
				Name:       user.Name,    // 纯净的显示名称
				UserID:     user.Id,
			}
			if skipAPIFetch {
				req.AvatarURL = user.AvatarURL
				req.BannerURL = user.BannerURL
				req.Description = user.Description
				req.Location = user.Location
				req.URL = user.URL
				req.Verified = user.Verified
				req.Protected = user.IsProtected
				req.CreatedAt = user.CreatedAt
			}
			requests = append(requests, req)
		}
	}

	for _, screenName := range profileUsers.screenName {
		requests = append(requests, profile.DownloadRequest{
			ScreenName: screenName,
			UserTitle:  "",
			Name:       "",
			UserID:     0,
		})
	}

	if len(profileList.id) > 0 {
		lists, err := profileList.GetList(ctx, client)
		if err != nil {
			log.WithError(err).Errorln("failed to get profile lists")
		} else {
			for _, lst := range lists {
				members, err := lst.GetMembers(ctx, client)
				if err != nil {
					log.WithError(err).WithField("list", lst.Title()).Errorln("failed to get list members")
					continue
				}
				for _, member := range members {
					requests = append(requests, profile.DownloadRequest{
						ScreenName:  member.ScreenName,
						UserTitle:   member.Title(),
						Name:        member.Name,
						UserID:      member.Id,
						AvatarURL:   member.AvatarURL,
						BannerURL:   member.BannerURL,
						Description: member.Description,
						Location:    member.Location,
						URL:         member.URL,
						Verified:    member.Verified,
						Protected:   member.IsProtected,
						CreatedAt:   member.CreatedAt,
					})
				}
			}
		}
	}

	if len(task.lists) > 0 {
		for _, lst := range task.lists {
			members, err := lst.GetMembers(ctx, client)
			if err != nil {
				log.WithError(err).WithField("list", lst.Title()).Errorln("failed to get list members for profile")
				continue
			}
			for _, member := range members {
				requests = append(requests, profile.DownloadRequest{
					ScreenName:  member.ScreenName,
					UserTitle:   member.Title(),
					Name:        member.Name,
					UserID:      member.Id,
					AvatarURL:   member.AvatarURL,
					BannerURL:   member.BannerURL,
					Description: member.Description,
					Location:    member.Location,
					URL:         member.URL,
					Verified:    member.Verified,
					Protected:   member.IsProtected,
					CreatedAt:   member.CreatedAt,
				})
			}
		}
	}

	seen := make(map[string]bool)
	uniqueRequests := make([]profile.DownloadRequest, 0)
	for _, req := range requests {
		if !seen[req.ScreenName] {
			seen[req.ScreenName] = true
			uniqueRequests = append(uniqueRequests, req)
		}
	}

	if len(uniqueRequests) == 0 {
		log.Infoln("no users to download profile")
		return
	}

	log.Infoln("starting profile download for", len(uniqueRequests), "users")

	results := profileDownloader.DownloadMultiple(ctx, uniqueRequests)

	success := 0
	failed := 0
	skipped := 0
	for _, r := range results {
		if r.Success {
			success++
		} else if r.Error != nil {
			failed++
		} else {
			skipped++
		}
	}

	log.Infoln("profile download completed - total:", len(results), "success:", success, "failed:", failed, "skipped:", skipped)

	fmt.Println("\n=== PROFILE_DOWNLOAD_RESULTS ===")
	for _, r := range results {
		if !r.Success {
			status := "SKIP"
			if r.Error != nil {
				status = "FAIL"
			}
			fmt.Printf("SCREEN_NAME:%s|STATUS:%s\n", r.ScreenName, status)
		}
	}
	fmt.Println("=== END_RESULTS ===")
}
