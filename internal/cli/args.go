package cli

import (
	"flag"

	"github.com/unkmonster/tmd/internal/downloading"
)

// CLIConfig CLI 配置
type CLIConfig struct {
	UsrArgs        downloading.UserArgs
	ListArgs       downloading.ListArgs
	FollArgs       downloading.UserArgs
	ProfileUsers   downloading.UserArgs
	ProfileList    downloading.ListArgs
	JsonArgs       downloading.JsonPathsArgs
	AutoFollow     bool
	NoRetry        bool
	MarkDownloaded bool
	MarkTime       string
	NoProfile      bool
}

// ParseArgs 解析命令行参数
func ParseArgs(args []string) (*flag.FlagSet, *CLIConfig, error) {
	cfg := &CLIConfig{
		UsrArgs:      downloading.UserArgs{},
		ListArgs:     downloading.ListArgs{},
		FollArgs:     downloading.UserArgs{},
		ProfileUsers: downloading.UserArgs{},
		ProfileList:  downloading.ListArgs{},
		JsonArgs:     downloading.JsonPathsArgs{},
	}

	fs := flag.NewFlagSet("tmd", flag.ContinueOnError)
	fs.Var(&cfg.UsrArgs, "user", "download tweets from the user")
	fs.Var(&cfg.ListArgs, "list", "batch download from list")
	fs.Var(&cfg.FollArgs, "foll", "batch download following")
	fs.Var(&cfg.ProfileUsers, "profile-user", "download profile")
	fs.Var(&cfg.ProfileList, "profile-list", "download list profiles")
	fs.Var(&cfg.JsonArgs, "json", "download from JSON")
	fs.BoolVar(&cfg.AutoFollow, "auto-follow", false, "auto follow")
	fs.BoolVar(&cfg.NoRetry, "no-retry", false, "no retry")
	fs.BoolVar(&cfg.MarkDownloaded, "mark-downloaded", false, "mark downloaded")
	fs.StringVar(&cfg.MarkTime, "mark-time", "", "mark time")
	fs.BoolVar(&cfg.NoProfile, "noprofile", false, "skip profile")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	return fs, cfg, nil
}
