package profile

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/unkmonster/tmd/internal/twitter"
)

func TestUserToProfileInfo(t *testing.T) {
	user := &twitter.User{
		Id:          12345,
		Name:        "Test User",
		ScreenName:  "testuser",
		AvatarURL:   "https://pbs.twimg.com/profile_images/12345/avatar.jpg",
		BannerURL:   "https://pbs.twimg.com/profile_banners/12345/banner.jpg",
		Description: "This is a test user",
		Location:    "Test City",
		URL:         "https://example.com",
		Verified:    true,
		IsProtected: false,
	}

	profile := userToProfileInfo(user)

	assert.Equal(t, uint64(12345), profile.ID)
	assert.Equal(t, "Test User", profile.Name)
	assert.Equal(t, "testuser", profile.ScreenName)
	assert.Equal(t, "https://pbs.twimg.com/profile_images/12345/avatar.jpg", profile.AvatarURL)
	assert.Equal(t, "https://pbs.twimg.com/profile_banners/12345/banner.jpg", profile.BannerURL)
	assert.Equal(t, "This is a test user", profile.Description)
	assert.Equal(t, "Test City", profile.Location)
	assert.Equal(t, "https://example.com", profile.URL)
	assert.True(t, profile.Verified)
	assert.False(t, profile.Protected)
}

func TestGetHighResAvatarURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		quality  string
		expected string
	}{
		{
			name:     "normal to bigger",
			url:      "https://pbs.twimg.com/profile_images/12345/avatar_normal.jpg",
			quality:  "bigger",
			expected: "https://pbs.twimg.com/profile_images/12345/avatar_bigger.jpg",
		},
		{
			name:     "normal to original (empty quality)",
			url:      "https://pbs.twimg.com/profile_images/12345/avatar_normal.jpg",
			quality:  "",
			expected: "https://pbs.twimg.com/profile_images/12345/avatar_.jpg",
		},
		{
			name:     "empty url returns empty",
			url:      "",
			quality:  "bigger",
			expected: "",
		},
		{
			name:     "url without _normal suffix unchanged",
			url:      "https://example.com/photo.jpg",
			quality:  "400x400",
			expected: "https://example.com/photo.jpg",
		},
		{
			name:     "png extension preserved",
			url:      "https://pbs.twimg.com/profile_images/12345/avatar_normal.png",
			quality:  "400x400",
			expected: "https://pbs.twimg.com/profile_images/12345/avatar_400x400.png",
		},
		{
			name:     "custom quality 200x200",
			url:      "https://pbs.twimg.com/profile_images/12345/avatar_normal.jpg",
			quality:  "200x200",
			expected: "https://pbs.twimg.com/profile_images/12345/avatar_200x200.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetHighResAvatarURL(tt.url, tt.quality)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProfileError(t *testing.T) {
	t.Run("Error with user", func(t *testing.T) {
		err := &ProfileError{Op: "fetch", User: "alice", Err: errors.New("timeout")}
		assert.Contains(t, err.Error(), "fetch")
		assert.Contains(t, err.Error(), "alice")
		assert.Contains(t, err.Error(), "timeout")
	})

	t.Run("Error without user", func(t *testing.T) {
		err := &ProfileError{Op: "init", Err: errors.New("config missing")}
		assert.Contains(t, err.Error(), "init")
		assert.NotContains(t, err.Error(), "for user")
		assert.Contains(t, err.Error(), "config missing")
	})

	t.Run("Unwrap returns inner error", func(t *testing.T) {
		inner := errors.New("inner")
		err := &ProfileError{Op: "test", Err: inner}
		assert.Same(t, inner, err.Unwrap())
	})
}

func TestFileStatusString(t *testing.T) {
	tests := []struct {
		status   FileStatus
		expected string
	}{
		{StatusFailed, "failed"},
		{StatusDownloaded, "downloaded"},
		{StatusSkipped, "skipped"},
		{FileStatus(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.NotNil(t, cfg)
	assert.True(t, cfg.EnableVersioning)
	assert.True(t, cfg.SkipUnchanged)
	assert.Equal(t, "400x400", cfg.AvatarQuality)
}

func TestProfileToJSON(t *testing.T) {
	profile := &ProfileInfo{
		ID:          999,
		Name:        "JSON User",
		ScreenName:  "jsonuser",
		Description: "Test description",
		Location:    "Tokyo",
		URL:         "https://json.user",
		Verified:    true,
		Protected:   false,
		CreatedAt:   "2024-01-01",
	}

	data, err := ProfileToJSON(profile)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"ID": 999`)
	assert.Contains(t, string(data), `"Name": "JSON User"`)
	assert.Contains(t, string(data), `"ScreenName": "jsonuser"`)
	assert.NotContains(t, string(data), `"AvatarURL"`)
	assert.NotContains(t, string(data), `"BannerURL"`)
}

func TestTwitterFetcher_Client(t *testing.T) {
	t.Run("nil clients returns nil", func(t *testing.T) {
		tf := &TwitterFetcher{}
		assert.Nil(t, tf.Client())
	})
}
