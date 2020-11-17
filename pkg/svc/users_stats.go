package svc

import (
	"context"

	"github.com/amikhailau/users-service/pkg/pb"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UsersStatsServerConfig struct {
	Database *gorm.DB
}

type UsersStatsServer struct {
	pb.UsersStatsServer
	cfg *UsersStatsServerConfig
}

var _ pb.UsersStatsServer = &UsersStatsServer{}

func NewUsersStatsServer(cfg *UsersStatsServerConfig) (*UsersStatsServer, error) {
	return &UsersStatsServer{
		UsersStatsServer: &pb.UsersStatsDefaultServer{},
		cfg:              cfg,
	}, nil
}

func (s *UsersStatsServer) GetStats(ctx context.Context, req *pb.ReadUserStatsRequest) (*pb.ReadUserStatsResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithFields(logrus.Fields{
		"name": req.GetUsername(),
	})
	logger.Debug("Read user stats")

	stats, err := s.getDBStats(logger, req.GetUsername())
	if err != nil {
		return nil, err
	}

	pbStats, err := stats.ToPB(ctx)
	if err != nil {
		logger.WithError(err).Error("Could not fetch user stats")
		return nil, status.Error(codes.Internal, "Could not fetch user stats")
	}

	return &pb.ReadUserStatsResponse{Result: &pbStats}, nil
}

func (s *UsersStatsServer) UpdateStats(ctx context.Context, req *pb.UpdateUserStatsRequest) (*pb.UpdateUserStatsResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithFields(logrus.Fields{
		"name": req.GetUsername(),
	})
	logger.Debug("Update user stats")

	stats, err := s.getDBStats(logger, req.GetUsername())
	if err != nil {
		return nil, err
	}

	stats.Kills += req.GetAddKills()
	stats.Games += req.GetAddGames()
	stats.Top5 += req.GetAddTop5()
	stats.Wins += req.GetAddWins()

	if err := s.cfg.Database.Save(stats).Error; err != nil {
		logger.WithError(err).Error("Could not update user stats")
		return nil, status.Error(codes.Internal, "Could not update user stats")
	}

	return &pb.UpdateUserStatsResponse{}, nil
}

func (s *UsersStatsServer) getDBStats(logger *logrus.Entry, username string) (*pb.UserStatsORM, error) {
	var usr pb.UserORM
	var stats []*pb.UserStatsORM
	if err := s.cfg.Database.Model(&usr).Where("name = ?", username).Association("UserStats").Find(&stats).Error; err != nil {
		logger.WithError(err).Error("Could not fetch user stats")
		return nil, status.Error(codes.Internal, "Could not fetch user stats")
	}

	if len(stats) != 1 {
		logger.Error("Corrupted user - doesn't have 1 stats object")
		return nil, status.Error(codes.Internal, "Profile is corrupted. Contact support.")
	}

	return stats[0], nil
}
