package profile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/unkmonster/tmd/internal/twitter"
)

type TwitterFetcher struct {
	clients []*resty.Client
	mu      sync.Mutex
}

var reNormalAvatarURL = regexp.MustCompile(`_normal(\.[a-zA-Z]+)$`)

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
	client := twitter.SelectClient(ctx, tf.clients, "/i/api/graphql/xmU6X_CKVnQ5lSrCbAmJsg/UserByScreenName")
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
	usr, _, err := twitter.GetUserByScreenName(ctx, client, screenName)
	if err != nil {
		return nil, err
	}
	return userToProfileInfo(usr), nil
}

func userToProfileInfo(u *twitter.User) *ProfileInfo {
	return &ProfileInfo{
		ID:          u.Id,
		Name:        u.Name,
		ScreenName:  u.ScreenName,
		AvatarURL:   u.AvatarURL,
		BannerURL:   u.BannerURL,
		Description: u.Description,
		Location:    u.Location,
		URL:         u.URL,
		Verified:    u.Verified,
		Protected:   u.IsProtected,
		CreatedAt:   u.CreatedAt,
	}
}

func (tf *TwitterFetcher) handleClientError(client *resty.Client, err error) {
	var apiErr *twitter.TwitterApiError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case twitter.ErrExceedPostLimit, twitter.ErrAccountLocked:
			twitter.SetClientError(client, apiErr)
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
		return nil, fmt.Errorf("avatar fetch failed with status %d, content-length: %d", resp.StatusCode(), len(resp.Body()))
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
		return nil, "", fmt.Errorf("banner fetch failed with status %d, content-length: %d", resp.StatusCode(), len(resp.Body()))
	}

	ext := ".jpg"
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

func (tf *TwitterFetcher) Client() *resty.Client {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	for _, client := range tf.clients {
		if twitter.GetClientError(client) == nil {
			return client
		}
	}

	if len(tf.clients) > 0 {
		return tf.clients[0]
	}
	return nil
}

func GetHighResAvatarURL(url string, quality string) string {
	if url == "" {
		return ""
	}
	return reNormalAvatarURL.ReplaceAllString(url, "_"+quality+"$1")
}

func ProfileToJSON(profile *ProfileInfo) ([]byte, error) {
	return json.MarshalIndent(profile, "", "  ")
}
