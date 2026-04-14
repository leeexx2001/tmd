package twitter

import (
	"fmt"
	"html"
	"time"

	"github.com/tidwall/gjson"
)

type Tweet struct {
	Id        uint64
	Text      string
	CreatedAt time.Time
	Creator   *User
	Urls      []string
	RawJSON   string
}

func parseTweetResults(tweet_results *gjson.Result) (*Tweet, error) {
	var tweet Tweet
	var err error

	result := tweet_results.Get("result")
	if !result.Exists() || result.Get("__typename").String() == "TweetTombstone" {
		return nil, nil
	}
	if result.Get("__typename").String() == "TweetWithVisibilityResults" {
		result = result.Get("tweet")
	}
	legacy := result.Get("legacy")
	if !legacy.Exists() {
		return nil, nil
	}
	user_results := result.Get("core.user_results")

	tweet.Id = result.Get("rest_id").Uint()
	tweet.RawJSON = result.Raw

	noteTweet := result.Get("note_tweet.note_tweet_results.result.text")
	if noteTweet.Exists() && noteTweet.String() != "" {
		tweet.Text = html.UnescapeString(noteTweet.String())
	} else {
		tweet.Text = html.UnescapeString(legacy.Get("full_text").String())
	}

	tweet.Creator, _ = parseUserResults(&user_results)
	tweet.CreatedAt, err = time.Parse(time.RubyDate, legacy.Get("created_at").String())
	if err != nil {
		return nil, fmt.Errorf("invalid time format %v", err)
	}
	media := legacy.Get("extended_entities.media")
	if media.Exists() {
		tweet.Urls = getUrlsFromMedia(&media)
	}
	return &tweet, nil
}

func getUrlsFromMedia(media *gjson.Result) []string {
	results := []string{}
	for _, m := range media.Array() {
		switch m.Get("type").String() {
		case "video", "animated_gif":
			results = append(results, m.Get("video_info.variants.@reverse.0.url").String())
		case "photo":
			results = append(results, m.Get("media_url_https").String())
		}
	}
	return results
}

// ended audio space

/*
id = ?
media_key = audio_space_by_id()
live_video_stream = get https://x.com/i/api/1.1/live_video_stream/status/{media_key}?client=web&use_syndication_guest_id=false&cookie_set_host=x.com
playlist = live_video_stream.source.location
handle playlist...
*/
