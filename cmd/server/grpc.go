package main

import (
	"io/ioutil"
	"time"

	"github.com/amikhailau/users-service/pkg/pb"
	"github.com/amikhailau/users-service/pkg/svc"
	"github.com/dgrijalva/jwt-go"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/infobloxopen/atlas-app-toolkit/gateway"
	"github.com/infobloxopen/atlas-app-toolkit/requestid"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func NewGRPCServer(logger *logrus.Logger, dbConnectionString string) (*grpc.Server, error) {
	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(
			keepalive.ServerParameters{
				Time:    time.Duration(viper.GetInt("config.keepalive.time")) * time.Second,
				Timeout: time.Duration(viper.GetInt("config.keepalive.timeout")) * time.Second,
			},
		),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				// logging middleware
				grpc_logrus.UnaryServerInterceptor(logrus.NewEntry(logger)),

				// Request-Id interceptor
				requestid.UnaryServerInterceptor(),

				// validation middleware
				grpc_validator.UnaryServerInterceptor(),

				// collection operators middleware
				gateway.UnaryServerInterceptor(),
			),
		),
	)

	publicKeyPath := viper.GetString("session.key.public.path")
	pubKeyBytes, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to read public key file")
		return nil, err
	}
	sessionPublicKey, err := jwt.ParseRSAPublicKeyFromPEM(pubKeyBytes)
	if err != nil {
		logger.WithError(err).Fatal("Failed to parse public key")
		return nil, err
	}

	privateKeyPath := viper.GetString("session.key.private.path")
	privKeyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to read private key file")
		return nil, err
	}
	sessionPrivateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKeyBytes)
	if err != nil {
		logger.WithError(err).Fatal("Failed to parse private key")
		return nil, err
	}

	// create new postgres database
	db, err := gorm.Open("postgres", dbConnectionString)
	if err != nil {
		return nil, err
	}

	// register service implementation with the grpcServer
	s, err := svc.NewBasicServer(db)
	if err != nil {
		return nil, err
	}
	pb.RegisterUsersServiceServer(grpcServer, s)

	usrS, err := svc.NewUsersServer(&svc.UsersServerConfig{
		Database:      db,
		RSAPrivateKey: sessionPrivateKey,
		RSAPublicKey:  sessionPublicKey,
	})
	if err != nil {
		return nil, err
	}
	pb.RegisterUsersServer(grpcServer, usrS)

	stiS, err := svc.NewStoreItemsServer(&svc.StoreItemsServerConfig{
		Database: db,
	})
	if err != nil {
		return nil, err
	}
	pb.RegisterStoreItemsServer(grpcServer, stiS)

	usrstsS, err := svc.NewUsersStatsServer(&svc.UsersStatsServerConfig{
		Database: db,
	})
	if err != nil {
		return nil, err
	}
	pb.RegisterUsersStatsServer(grpcServer, usrstsS)

	newsS, err := svc.NewNewsServer(&svc.NewsServerConfig{
		Database: db,
	})
	if err != nil {
		return nil, err
	}
	pb.RegisterNewsServiceServer(grpcServer, newsS)

	return grpcServer, nil
}
