package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"github.com/unkmonster/tmd/internal/twitter"
)

type TwitterFetcher struct {
	clients []*resty.Client
	mu      sync.Mutex
}

func NewTwitterFetcher(client *resty.Client) *TwitterFetcher {
	return &TwitterFetcher{clients: []*resty.Client{client}}
}

func NewTwitterFetcherWithClients(clients []*resty.Client) *TwitterFetcher {
	if len(clients) == 0 {
		panic("clients cannot be empty")
	}
	return &TwitterFetcher{clients: clients}
}

func (tf *TwitterFetcher) FetchProfile(ctx context.Context, screenName string) (*ProfileInfo, error) {
	tf.mu.Lock()
	client := twitter.SelectProfileClient(ctx, tf.clients)
	tf.mu.Unlock()

	if client == nil {
		return nil, &ProfileError{Op: "fetch", User: screenName, Err: fmt.Errorf("no available client")}
	}

	profile, err := tf.fetchFullProfile(ctx, client, screenName)
	if err != nil {
		tf.handleClientError(client, err)
		return nil, &ProfileError{Op: "fetch", User: screenName, Err: err}
	}

	return profile, nil
}

func (tf *TwitterFetcher) fetchFullProfile(ctx context.Context, client *resty.Client, screenName string) (*ProfileInfo, error) {
	api := &userByScreenName{screenName: screenName}
	url := makeProfileUrl(api)

	resp, err := client.R().SetContext(ctx).Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile: %w", err)
	}

	if err := twitter.CheckApiResp(resp.Body()); err != nil {
		return nil, err
	}

	return parseProfileFromResponse(resp.Body())
}

func (tf *TwitterFetcher) handleClientError(client *resty.Client, err error) {
	if v, ok := err.(*twitter.TwitterApiError); ok {
		switch v.Code {
		case twitter.ErrExceedPostLimit, twitter.ErrAccountLocked:
			twitter.SetClientError(client, err)
		}
	}
}

func (tf *TwitterFetcher) FetchAvatar(ctx context.Context, url string) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("avatar URL is empty")
	}

	client := tf.selectAvailableClient(ctx)
	if client == nil {
		return nil, fmt.Errorf("no available client for avatar fetch")
	}

	resp, err := client.R().SetContext(ctx).Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch avatar: %w", err)
	}

	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("avatar fetch failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	return resp.Body(), nil
}

func (tf *TwitterFetcher) FetchBanner(ctx context.Context, url string) ([]byte, string, error) {
	if url == "" {
		return nil, "", fmt.Errorf("banner URL is empty")
	}

	client := tf.selectAvailableClient(ctx)
	if client == nil {
		return nil, "", fmt.Errorf("no available client for banner fetch")
	}

	resp, err := client.R().SetContext(ctx).Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch banner: %w", err)
	}

	if resp.StatusCode() >= 400 {
		return nil, "", fmt.Errorf("banner fetch failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	// 从Content-Type获取扩展名
	ext := ".jpg" // 默认
	contentType := resp.Header().Get("Content-Type")
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	case "image/gif":
		ext = ".gif"
	}

	return resp.Body(), ext, nil
}

func (tf *TwitterFetcher) selectAvailableClient(ctx context.Context) *resty.Client {
	for _, client := range tf.clients {
		if ctx.Err() != nil {
			return nil
		}
		if twitter.GetClientError(client) == nil {
			return client
		}
	}
	return nil
}

func parseProfileFromResponse(data []byte) (*ProfileInfo, error) {
	user := gjson.GetBytes(data, "data.user")
	if !user.Exists() {
		return nil, fmt.Errorf("user not found in response")
	}

	result := user.Get("result")
	if !result.Exists() {
		return nil, fmt.Errorf("user result not found")
	}

	if result.Get("__typename").String() == "UserUnavailable" {
		return nil, fmt.Errorf("user is unavailable")
	}

	legacy := result.Get("legacy")
	if !legacy.Exists() {
		return nil, fmt.Errorf("user legacy data not found")
	}

	profile := &ProfileInfo{}

	if id := result.Get("rest_id"); id.Exists() {
		profile.ID = uint64(id.Int())
	}

	if screenName := legacy.Get("screen_name"); screenName.Exists() {
		profile.ScreenName = screenName.String()
	}

	if name := legacy.Get("name"); name.Exists() {
		profile.Name = name.String()
	}

	if desc := legacy.Get("description"); desc.Exists() {
		profile.Description = desc.String()
	}

	if location := legacy.Get("location"); location.Exists() {
		profile.Location = location.String()
	}

	if url := legacy.Get("url"); url.Exists() {
		profile.URL = url.String()
	}

	if verified := legacy.Get("verified"); verified.Exists() {
		profile.Verified = verified.Bool()
	}

	if protected := legacy.Get("protected"); protected.Exists() {
		profile.Protected = protected.Bool()
	}

	if created := legacy.Get("created_at"); created.Exists() {
		profile.CreatedAt = created.String()
	}

	if avatar := result.Get("avatar.image_url"); avatar.Exists() && avatar.String() != "" {
		profile.AvatarURL = avatar.String()
	} else if avatar := legacy.Get("profile_image_url_https"); avatar.Exists() && avatar.String() != "" {
		profile.AvatarURL = avatar.String()
	} else if avatar := legacy.Get("profile_image_url"); avatar.Exists() && avatar.String() != "" {
		profile.AvatarURL = avatar.String()
	}

	if banner := legacy.Get("profile_banner_url"); banner.Exists() {
		profile.BannerURL = banner.String()
	}

	return profile, nil
}

func GetHighResAvatarURL(url string, quality string) string {
	if url == "" {
		return ""
	}

	re := regexp.MustCompile(`_normal(\.[a-zA-Z]+)$`)
	return re.ReplaceAllString(url, "_"+quality+"$1")
}

func ProfileToJSON(profile *ProfileInfo) ([]byte, error) {
	return json.MarshalIndent(profile, "", "  ")
}

type userByScreenName struct {
	screenName string
}

func (u *userByScreenName) Path() string {
	return "/i/api/graphql/xmU6X_CKVnQ5lSrCbAmJsg/UserByScreenName"
}

func (u *userByScreenName) QueryParam() url.Values {
	v := url.Values{}
	variables := `{"screen_name":"%s","withSafetyModeUserFields":true}`
	features := `{"hidden_profile_subscriptions_enabled":true,"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"subscriptions_verification_info_is_identity_verified_enabled":true,"subscriptions_verification_info_verified_since_enabled":true,"highlights_tweets_tab_ui_enabled":true,"responsive_web_twitter_article_notes_tab_enabled":true,"subscriptions_feature_can_gift_premium":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true}`
	fieldToggles := `{"withAuxiliaryUserLabels":false}`

	v.Set("variables", fmt.Sprintf(variables, u.screenName))
	v.Set("features", features)
	v.Set("fieldToggles", fieldToggles)
	return v
}

func makeProfileUrl(api interface {
	Path() string
	QueryParam() url.Values
}) string {
	u, _ := url.Parse("https://x.com")
	u = u.JoinPath(api.Path())
	u.RawQuery = api.QueryParam().Encode()
	return u.String()
}
