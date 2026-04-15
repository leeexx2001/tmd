package profile

import (
	"testing"

	"github.com/unkmonster/tmd/internal/twitter"
	"github.com/stretchr/testify/assert"
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
			name:     "normal to original",
			url:      "https://pbs.twimg.com/profile_images/12345/avatar_normal.jpg",
			quality:  "",
			expected: "https://pbs.twimg.com/profile_images/12345/avatar_.jpg",
		},
		{
			name:     "normal to bigger",
			url:      "https://pbs.twimg.com/profile_images/12345/avatar_normal.jpg",
			quality:  "bigger",
			expected: "https://pbs.twimg.com/profile_images/12345/avatar_bigger.jpg",
		},
		{
			name:     "empty url",
			url:      "",
			quality:  "bigger",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetHighResAvatarURL(tt.url, tt.quality)
			assert.Equal(t, tt.expected, result)
		})
	}
}
