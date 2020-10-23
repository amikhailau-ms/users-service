package main

import (
	"time"

	"github.com/amikhailau/users-service/pkg/pb"
	"github.com/amikhailau/users-service/pkg/svc"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func CreateServer(logger *logrus.Logger, db *gorm.DB, interceptors []grpc.UnaryServerInterceptor) (*grpc.Server, error) {
	// create new gRPC grpcServer with middleware chain
	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(
			keepalive.ServerParameters{
				Time:    time.Duration(viper.GetInt("config.keepalive.time")) * time.Second,
				Timeout: time.Duration(viper.GetInt("config.keepalive.timeout")) * time.Second,
			},
		), grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(interceptors...)))

	// register all of our services into the grpcServer

	user, err := svc.NewUsersServer(db)
	if err != nil {
		return nil, err
	}
	pb.RegisterUsersServer(grpcServer, user)

	item, err := svc.NewItemsServer(db)
	if err != nil {
		return nil, err
	}
	pb.RegisterItemsServer(grpcServer, item)

	return grpcServer, nil
}
