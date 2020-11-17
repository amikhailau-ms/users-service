package svc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/amikhailau/users-service/pkg/pb"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (s *UsersServer) Create(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithFields(logrus.Fields{
		"name":  req.GetName(),
		"email": req.GetEmail(),
	})
	logger.Debug("User registration started")

	db := s.cfg.Database

	var existingUser pb.UserORM
	if err := db.Where("name = ?", req.GetName()).First(&existingUser).Error; err == nil {
		logger.Error("User with such name already exists")
		return nil, status.Error(codes.InvalidArgument, "User with such name already exists")
	} else if err != gorm.ErrRecordNotFound {
		logger.WithError(err).Error("Could not create new user")
		return nil, status.Error(codes.Internal, "Could not create new user")
	}

	if err := db.Where("email = ?", req.GetEmail()).First(&existingUser).Error; err == nil {
		logger.Error("User with such email already exists")
		return nil, status.Error(codes.InvalidArgument, "User with such email already exists")
	} else if err != gorm.ErrRecordNotFound {
		logger.WithError(err).Error("Could not create new user")
		return nil, status.Error(codes.Internal, "Could not create new user")
	}

	passwordBytes := []byte(req.GetPassword())
	encryptedPasswordBytes := sha256.Sum256(passwordBytes)
	tmpSlice := encryptedPasswordBytes[:]
	hexPassword := hex.EncodeToString(tmpSlice)
	userID := uuid.NewV4().String()

	newUser := pb.UserORM{
		Id:       userID,
		Email:    req.GetEmail(),
		Name:     req.GetName(),
		Password: hexPassword,
		Stats:    &pb.UserStatsORM{},
		Coins:    0,
		Gems:     0,
	}

	if err := db.Create(&newUser).Error; err != nil {
		logger.WithError(err).Error("Could not create new user")
		return nil, status.Error(codes.Internal, "Could not create new user")
	}

	pbUser, err := newUser.ToPB(ctx)
	if err != nil {
		logger.WithError(err).Error("Could not create new user")
		return nil, status.Error(codes.Internal, "Could not create new user")
	}

	logger.Debug("User registration finished")

	return &pb.CreateUserResponse{Result: &pbUser}, nil
}

func (s *UsersServer) Read(ctx context.Context, req *pb.ReadUserRequest) (*pb.ReadUserResponse, error) {
	logger := ctxlogrus.Extract(ctx)
	logger.Debug("Read user")

	var existingUser pb.UserORM
	searchCriterias := []string{"id", "name", "email"}
	userFound := false
	for _, searchCriteria := range searchCriterias {
		if err := s.cfg.Database.Where(fmt.Sprintf("%v = ?", searchCriteria), req.GetId()).First(&existingUser).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			} else {
				logger.WithError(err).Error("Could not find user")
				return nil, status.Error(codes.Internal, "Could not find user")
			}
		}
		userFound = true
		break
	}

	if userFound {
		pbUser, err := existingUser.ToPB(ctx)
		if err != nil {
			logger.WithError(err).Error("Could not find user")
			return nil, status.Error(codes.Internal, "Could not find user")
		}
		return &pb.ReadUserResponse{Result: &pbUser}, nil
	}

	logger.Error("Could not find user by any criteria")
	return nil, status.Error(codes.NotFound, "Could not find user")
}

func (s *UsersServer) Delete(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithField("id", req.GetId())
	logger.Debug("Delete user")

	var user pb.UserORM
	if err := s.cfg.Database.Where("id = ?", req.GetId()).Delete(&user).Error; err != nil && err != gorm.ErrRecordNotFound {
		logger.WithError(err).Error("Could not delete user")
		return nil, status.Error(codes.Internal, "Could not delete user")
	}

	return &pb.DeleteUserResponse{}, nil
}

func (s *UsersServer) Update(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithField("id", req.GetId())
	logger.Debug("Update user")

	return nil, status.Error(codes.Unimplemented, "Non-MVP endpoint")
}

func (s *UsersServer) List(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	logger := ctxlogrus.Extract(ctx)
	logger.Debug("List users")

	return nil, status.Error(codes.Unimplemented, "Non-MVP endpoint")
}
