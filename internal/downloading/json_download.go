package downloading

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"github.com/unkmonster/tmd/internal/downloader"
	"github.com/unkmonster/tmd/internal/naming"
	"github.com/unkmonster/tmd/internal/twitter"
	"github.com/unkmonster/tmd/internal/utils"
)

type RawTweetEntry struct {
	Id        string `json:"id"`
	CreatedAt string `json:"created_at"`
	FullText  string `json:"full_text"`
	Media     []struct {
		Type     string `json:"type"`
		URL      string `json:"url"`
		Original string `json:"original"`
	} `json:"media"`
	ScreenName   string `json:"screen_name"`
	Name         string `json:"name"`
	UserId       string `json:"user_id"`
	OriginalJSON []byte `json:"-"`
}

type RawTweetFile struct {
	Entries []RawTweetEntry `json:"-"`
}

func (f *RawTweetFile) GetTweets() ([]*twitter.Tweet, error) {
	tweets := make([]*twitter.Tweet, 0, len(f.Entries))
	for _, entry := range f.Entries {
		if entry.Id == "" || len(entry.Media) == 0 {
			continue
		}

		tweet := &twitter.Tweet{
			Id:      parseUint64(entry.Id),
			Text:    entry.FullText,
			RawJSON: string(entry.OriginalJSON),
			Urls:    extractUrlsFromRawEntry(&entry),
		}

		tweet.CreatedAt = parseTwitterDate(entry.CreatedAt)

		if entry.UserId != "" || entry.ScreenName != "" {
			tweet.Creator = &twitter.User{
				Id:         parseUint64(entry.UserId),
				Name:       entry.Name,
				ScreenName: entry.ScreenName,
			}
		}

		tweets = append(tweets, tweet)
	}
	return tweets, nil
}

func parseUint64(s string) uint64 {
	var v uint64
	fmt.Sscanf(s, "%d", &v)
	return v
}

func parseTwitterDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Now()
	}

	layouts := []string{
		"2006-01-02 15:04:05 -07:00",
		"2006-01-02 15:04:05 +08:00",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
		time.RubyDate,
		"Mon Jan 02 15:04:05 -0700 2006",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t
		}
	}

	return time.Now()
}

func extractUrlsFromRawEntry(entry *RawTweetEntry) []string {
	urls := make([]string, 0)
	for _, m := range entry.Media {
		if m.Original != "" {
			urls = append(urls, m.Original)
		} else if m.URL != "" && !strings.Contains(m.URL, "t.co/") {
			urls = append(urls, m.URL)
		}
	}
	return urls
}

type FormattedTweetEntry = map[string]any

type FormattedJsonFile struct {
	Entries []FormattedTweetEntry `json:"-"`
}

func (f *FormattedJsonFile) GetTweets() ([]*twitter.Tweet, error) {
	tweets := make([]*twitter.Tweet, 0, len(f.Entries))
	for _, entry := range f.Entries {
		tweet, err := parseFormattedEntry(&entry)
		if err != nil || tweet == nil {
			continue
		}
		tweets = append(tweets, tweet)
	}
	return tweets, nil
}

func parseFormattedEntry(entry *FormattedTweetEntry) (*twitter.Tweet, error) {
	if entry == nil {
		return nil, nil
	}

	restId := getStringFromMap(*entry, "rest_id")
	if restId == "" {
		return nil, nil
	}

	tweet := &twitter.Tweet{
		Id: parseUint64(restId),
		RawJSON: func() string {
			if b, err := json.Marshal(entry); err == nil {
				return string(b)
			}
			return ""
		}(),
	}

	if legacy, ok := (*entry)["legacy"].(map[string]any); ok {
		tweet.Text = getStringFromMap(legacy, "full_text")
		if tweet.Text == "" {
			tweet.Text = getStringFromMap(legacy, "text")
		}
		if createdAt := getStringFromMap(legacy, "created_at"); createdAt != "" {
			tweet.CreatedAt = parseTwitterDate(createdAt)
		}

		if extendedEntities, ok := legacy["extended_entities"].(map[string]any); ok {
			if mediaList, ok := extendedEntities["media"].([]any); ok {
				for _, m := range mediaList {
					if mm, ok := m.(map[string]any); ok {
						mediaType := getStringFromMap(mm, "type")
						switch mediaType {
						case "video", "animated_gif":
							if variants, ok := mm["video_info"].(map[string]any); ok {
								if variantList, ok := variants["variants"].([]any); ok {
									var bestURL string
									var maxBitrate int
									for _, v := range variantList {
										if vv, ok := v.(map[string]any); ok {
											if url := getStringFromMap(vv, "url"); url != "" {
												if bitrate := getIntFromMap(vv, "bitrate"); bitrate > maxBitrate {
													maxBitrate = bitrate
													bestURL = url
												}
											}
										}
									}
									if bestURL != "" {
										tweet.Urls = append(tweet.Urls, bestURL)
									}
								}
							}
						case "photo":
							if url := getStringFromMap(mm, "media_url_https"); url != "" {
								tweet.Urls = append(tweet.Urls, url)
							}
						}
					}
				}
			}
		}
	}

	if core, ok := (*entry)["core"].(map[string]any); ok {
		if userResults, ok := core["user_results"].(map[string]any); ok {
			if result, ok := userResults["result"].(map[string]any); ok {
				tweet.Creator = &twitter.User{}

				if id := getStringFromMap(result, "rest_id"); id != "" {
					tweet.Creator.Id = parseUint64(id)
				}

				if legacy, ok := result["legacy"].(map[string]any); ok {
					tweet.Creator.Name = getStringFromMap(legacy, "name")
					tweet.Creator.ScreenName = getStringFromMap(legacy, "screen_name")
					if avatar := getStringFromMap(legacy, "profile_image_url_https"); avatar != "" {
						tweet.Creator.AvatarURL = utils.StripAvatarSuffix(avatar)
					}
				}
			}
		}
	}

	if len(tweet.Urls) == 0 {
		return nil, nil
	}

	return tweet, nil
}

func getStringFromMap(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getIntFromMap(m map[string]any, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

type JsonPackagedTweet struct {
	tweet *twitter.Tweet
	dir   string
}

func (pt JsonPackagedTweet) GetTweet() *twitter.Tweet {
	return pt.tweet
}

func (pt JsonPackagedTweet) GetPath() string {
	return pt.dir
}

func DownloadFromJsonFiles(ctx context.Context, client *resty.Client, dir string, jsonPaths []string) ([]JsonPackagedTweet, error) {
	allTweets := make([]*twitter.Tweet, 0)

	for _, path := range jsonPaths {
		entries, err := readJsonEntries(path)
		if err != nil {
			log.Warnf("failed to read JSON file %s: %v", path, err)
			continue
		}

		for _, entry := range entries {
			if tweets, err := entry.GetTweets(); err == nil {
				allTweets = append(allTweets, tweets...)
			}
		}
	}

	if len(allTweets) == 0 {
		return nil, fmt.Errorf("no tweets with media found in provided JSON files")
	}

	tweetDir := filepath.Join(dir, "users")
	if err := os.MkdirAll(tweetDir, 0755); err != nil {
		return nil, err
	}

	pts := make([]JsonPackagedTweet, 0, len(allTweets))
	for _, tw := range allTweets {
		userDir := tweetDir
		if tw.Creator != nil {
			userNaming := naming.NewUserNaming(tw.Creator.Name, tw.Creator.ScreenName)
			userDir = filepath.Join(tweetDir, userNaming.SanitizedTitle())
			os.MkdirAll(userDir, 0755)
		}
		pts = append(pts, JsonPackagedTweet{tweet: tw, dir: userDir})
	}

	return pts, nil
}

type JsonEntry interface {
	GetTweets() ([]*twitter.Tweet, error)
}

func readJsonEntries(path string) ([]JsonEntry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return readJsonEntryFile(path)
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	entries := make([]JsonEntry, 0)
	for _, f := range files {
		fullPath := filepath.Join(path, f.Name())
		if f.IsDir() {
			subEntries, err := readJsonEntries(fullPath)
			if err != nil {
				continue
			}
			entries = append(entries, subEntries...)
			continue
		}
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		subEntries, err := readJsonEntryFile(fullPath)
		if err != nil {
			continue
		}
		entries = append(entries, subEntries...)
	}
	return entries, nil
}

func readJsonEntryFile(path string) ([]JsonEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var rawEntries []RawTweetEntry
	if err := json.Unmarshal(data, &rawEntries); err == nil && len(rawEntries) > 0 {
		for i := range rawEntries {
			if entryJSON, err := json.Marshal(rawEntries[i]); err == nil {
				rawEntries[i].OriginalJSON = entryJSON
			}
		}
		return []JsonEntry{&RawTweetFile{Entries: rawEntries}}, nil
	}

	var singleRaw RawTweetEntry
	if err := json.Unmarshal(data, &singleRaw); err == nil && singleRaw.Id != "" {
		singleRaw.OriginalJSON = data
		return []JsonEntry{&RawTweetFile{Entries: []RawTweetEntry{singleRaw}}}, nil
	}

	var formatted FormattedTweetEntry
	if err := json.Unmarshal(data, &formatted); err == nil {
		if _, hasRestId := formatted["rest_id"]; hasRestId {
			return []JsonEntry{&FormattedJsonFile{Entries: []FormattedTweetEntry{formatted}}}, nil
		}
	}

	return nil, fmt.Errorf("unrecognized JSON format in file: %s", path)
}

type JsonDownloadResult struct {
	Path       string        `json:"path"`
	Success    bool          `json:"success"`
	TweetCount int           `json:"tweet_count"`
	Error      string        `json:"error,omitempty"`
	Duration   time.Duration `json:"duration"`
}

func DownloadJsonDir(ctx context.Context, client *resty.Client, baseDir string, dwn downloader.Downloader, fileWriter downloader.FileWriter, jsonPaths ...string) []JsonDownloadResult {
	results := make([]JsonDownloadResult, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, jsonPath := range jsonPaths {
		info, err := os.Stat(jsonPath)
		if err != nil {
			results = append(results, JsonDownloadResult{Path: jsonPath, Success: false, Error: err.Error()})
			continue
		}

		if info.IsDir() {
			entries, err := os.ReadDir(jsonPath)
			if err != nil {
				results = append(results, JsonDownloadResult{Path: jsonPath, Success: false, Error: err.Error()})
				continue
			}
			for _, entry := range entries {
				fullPath := filepath.Join(jsonPath, entry.Name())
				if entry.IsDir() {
					subResults := DownloadJsonDir(ctx, client, baseDir, dwn, fileWriter, fullPath)
					mu.Lock()
					results = append(results, subResults...)
					mu.Unlock()
					continue
				}
				if !strings.HasSuffix(entry.Name(), ".json") {
					continue
				}
				downloadJsonFileAsync(ctx, client, baseDir, fullPath, dwn, fileWriter, &results, &mu, &wg)
			}
		} else {
			downloadJsonFileAsync(ctx, client, baseDir, jsonPath, dwn, fileWriter, &results, &mu, &wg)
		}
	}

	wg.Wait()
	return results
}

func downloadJsonFileAsync(ctx context.Context, client *resty.Client, baseDir, jsonPath string, dwn downloader.Downloader, fileWriter downloader.FileWriter, results *[]JsonDownloadResult, mu *sync.Mutex, wg *sync.WaitGroup) {
	wg.Add(1)
	go func(path string) {
		defer wg.Done()
		start := time.Now()
		result := JsonDownloadResult{Path: path}
		tweetCount, err := downloadSingleJsonFile(ctx, client, baseDir, path, dwn, fileWriter)
		result.TweetCount = tweetCount
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
		}
		result.Duration = time.Since(start)
		mu.Lock()
		*results = append(*results, result)
		mu.Unlock()
	}(jsonPath)
}

func downloadSingleJsonFile(ctx context.Context, client *resty.Client, baseDir string, jsonPath string, dwn downloader.Downloader, fileWriter downloader.FileWriter) (int, error) {
	pts, err := DownloadFromJsonFiles(ctx, client, baseDir, []string{jsonPath})
	if err != nil {
		return 0, err
	}

	tweetCount := len(pts)
	packged := make([]PackagedTweet, len(pts))
	for i, pt := range pts {
		packged[i] = pt
	}
	failedTweets := BatchDownloadTweet(ctx, client, false, dwn, fileWriter, packged...)
	if len(failedTweets) > 0 {
		return tweetCount, fmt.Errorf("%d tweets failed to download", len(failedTweets))
	}

	return tweetCount, nil
}
