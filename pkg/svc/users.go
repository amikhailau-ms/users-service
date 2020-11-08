package svc

import (
	"github.com/amikhailau/users-service/pkg/pb"
	"github.com/jinzhu/gorm"
)

type UsersServerConfig struct {
	Database *gorm.DB
}

type UsersServer struct {
	pb.UsersServer
	cfg *UsersServerConfig
}

var _ pb.UsersServer = &UsersServer{}

func NewUsersServer(cfg *UsersServerConfig) (*UsersServer, error) {
	return &UsersServer{
		UsersServer: &pb.UsersDefaultServer{},
		cfg:         cfg,
	}, nil
}
