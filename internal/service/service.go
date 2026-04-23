package service

import (
	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"

	"github.com/unkmonster/tmd/internal/config"
)

// Services 所有服务的集合
type Services struct {
	Download *DownloadService
	Mark     *MarkService
	Json     *JsonService
	Profile  *ProfileService
}

// NewServices 创建所有服务
func NewServices(client *resty.Client, additionalClients []*resty.Client, db *sqlx.DB, conf *config.Config, appRootPath string) *Services {
	profileSvc := NewProfileService(client, additionalClients, db, conf, appRootPath)

	return &Services{
		Download: NewDownloadService(client, additionalClients, db, conf, appRootPath, profileSvc),
		Mark:     NewMarkService(client, db),
		Json:     NewJsonService(client, additionalClients, db, conf),
		Profile:  profileSvc,
	}
}
