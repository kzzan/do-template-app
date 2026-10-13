package service

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/spf13/viper"

	userv1 "do-template-app/api/pb/userv1"
	v1 "do-template-app/app/home/api/v1"
	"do-template-app/pkg/grpcx"
)

// SiteService 门户站点信息接口（无 DB）。
type SiteService interface {
	Health() v1.HealthData
	Meta() v1.MetaData
	Manifest() v1.ManifestData
	Stats(ctx context.Context) (map[string]int64, error)
}

func NewSiteService(injector do.Injector) (SiteService, error) {
	return &siteService{
		config:   do.MustInvoke[*viper.Viper](injector),
		grpcConn: do.MustInvoke[*grpcx.ClientConn](injector),
	}, nil
}

type siteService struct {
	config   *viper.Viper
	grpcConn *grpcx.ClientConn
}

func (s *siteService) Health() v1.HealthData {
	return v1.HealthData{App: "home", Status: "ok"}
}

func (s *siteService) Meta() v1.MetaData {
	return v1.MetaData{
		App:   "home",
		Stage: s.config.GetString("env"),
		Entry: "app/home/cmd/server",
		Title: s.config.GetString("site.title"),
	}
}

func (s *siteService) Manifest() v1.ManifestData {
	data := v1.ManifestData{
		App:      "home",
		Stage:    s.config.GetString("env"),
		Entry:    "app/home/cmd/server",
		Title:    s.config.GetString("site.title"),
		Headline: s.config.GetString("site.headline"),
		Features: []v1.ManifestFeature{
			{
				Name:        "Public Shell",
				Description: "Serve public pages and lightweight shell APIs from app/home.",
				Route:       "/v1/site/manifest",
			},
		},
	}
	return data
}

// Stats 站点统计：经 gRPC 调 user 服务取注册用户数。
func (s *siteService) Stats(ctx context.Context) (map[string]int64, error) {
	client := userv1.NewUserServiceClient(s.grpcConn.ClientConn)
	resp, err := client.CountUsers(ctx, &userv1.CountUsersRequest{})
	if err != nil {
		return nil, err
	}
	return map[string]int64{"users": resp.GetTotal()}, nil
}
