package naming

import (
	"testing"
)

func TestTweetNaming_LogFormat(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		tweetID  uint64
		creator  string
		expected string
	}{
		{
			name:     "normal case",
			text:     "比基尼",
			tweetID:  1355100264760393735,
			creator:  "吕布(QqiRru)",
			expected: "[吕布(QqiRru)] 比基尼_1355100264760393735",
		},
		{
			name:     "empty text",
			text:     "",
			tweetID:  123,
			creator:  "test",
			expected: "[test] tweet_123",
		},
		{
			name:     "special chars",
			text:     "hello\nworld",
			tweetID:  456,
			creator:  "user",
			expected: "[user] hello world_456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tn := NewTweetNaming(tt.text, tt.tweetID, tt.creator)
			if got := tn.LogFormat(); got != tt.expected {
				t.Errorf("LogFormat() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTweetNaming_FileName(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		tweetID  uint64
		ext      string
		expected string
	}{
		{
			name:     "normal case",
			text:     "比基尼",
			tweetID:  1355100264760393735,
			ext:      ".jpg",
			expected: "比基尼_1355100264760393735.jpg",
		},
		{
			name:     "empty text",
			text:     "",
			tweetID:  123,
			ext:      ".json",
			expected: "tweet_123.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tn := NewTweetNaming(tt.text, tt.tweetID, "creator")
			if got := tn.FileName(tt.ext); got != tt.expected {
				t.Errorf("FileName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestUserNaming(t *testing.T) {
	un := NewUserNaming("吕布", "QqiRru")

	expected := "吕布(QqiRru)"
	if un.SanitizedTitle() != expected {
		t.Errorf("SanitizedTitle() = %q, want %q", un.SanitizedTitle(), expected)
	}
}

func TestListNaming(t *testing.T) {
	ln := NewListNamingFromBase(&mockListBase{id: 9876543210, title: "Test List(9876543210)"})

	expected := "Test List(9876543210)"
	if ln.SanitizedTitle() != expected {
		t.Errorf("SanitizedTitle() = %q, want %q", ln.SanitizedTitle(), expected)
	}
}

type mockListBase struct {
	id    int64
	title string
}

func (m *mockListBase) GetId() int64  { return m.id }
func (m *mockListBase) Title() string { return m.title }
