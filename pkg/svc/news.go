package svc

import (
	"context"

	"github.com/amikhailau/users-service/pkg/pb"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NewsServerConfig struct {
	Database *gorm.DB
}

type NewsServer struct {
	pb.NewsServiceServer
	cfg *NewsServerConfig
}

var _ pb.NewsServiceServer = &NewsServer{}

func NewNewsServer(cfg *NewsServerConfig) (*NewsServer, error) {
	return &NewsServer{
		NewsServiceServer: &pb.NewsServiceDefaultServer{},
		cfg:               cfg,
	}, nil
}

func (s *NewsServer) Create(ctx context.Context, req *pb.CreateNewsRequest) (*pb.CreateNewsResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithFields(logrus.Fields{
		"title": req.GetTitle(),
	})
	logger.Debug("Create News")

	var existingNews pb.NewsORM
	if err := s.cfg.Database.Where("title = ?", req.GetTitle()).First(&existingNews).Error; err == nil {
		logger.Error("News with such title already exists")
		return nil, status.Error(codes.InvalidArgument, "News with such title already exists")
	} else if err != gorm.ErrRecordNotFound {
		logger.WithError(err).Error("Could not create news")
		return nil, status.Error(codes.Internal, "Could not create news")
	}

	news := pb.NewsORM{
		Id:          uuid.NewV4().String(),
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		ImageLink:   req.GetImageLink(),
	}

	if err := s.cfg.Database.Create(&news).Error; err != nil {
		logger.WithError(err).Error("Could not create news")
		return nil, status.Error(codes.Internal, "Could not create news")
	}

	if err := s.cfg.Database.Where("title = ?", req.GetTitle()).First(&news).Error; err != nil {
		logger.WithError(err).Error("Could not fetch news")
	}

	pbNews, err := news.ToPB(ctx)
	if err != nil {
		logger.WithError(err).Error("Could not create news")
		return nil, status.Error(codes.Internal, "Could not create news")
	}

	return &pb.CreateNewsResponse{Result: &pbNews}, nil
}

func (s *NewsServer) Read(ctx context.Context, req *pb.ReadNewsRequest) (*pb.ReadNewsResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithFields(logrus.Fields{
		"id": req.GetId(),
	})
	logger.Debug("Read News")

	res, err := pb.DefaultReadNews(ctx, &pb.News{Id: req.GetId()}, s.cfg.Database)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.WithError(err).Error("News not found")
			return nil, status.Error(codes.NotFound, "News not found")
		}
		logger.WithError(err).Error("Could not read news")
		return nil, status.Error(codes.Internal, "Could not read news")
	}

	return &pb.ReadNewsResponse{Result: res}, nil
}

func (s *NewsServer) List(ctx context.Context, req *pb.ListNewsRequest) (*pb.ListNewsResponse, error) {
	logger := ctxlogrus.Extract(ctx)
	logger.Debug("List news")

	res, err := pb.DefaultListNews(ctx, s.cfg.Database, req.GetFilter(), req.GetOrderBy(), req.GetPaging(), req.GetFields())
	if err != nil {
		logger.WithError(err).Error("Could not list news")
		return nil, status.Error(codes.Internal, "Could not list news")
	}

	return &pb.ListNewsResponse{Results: res}, nil
}

func (s *NewsServer) Update(ctx context.Context, req *pb.UpdateNewsRequest) (*pb.UpdateNewsResponse, error) {
	logger := ctxlogrus.Extract(ctx)
	logger.Debug("Update news")

	var existingNews pb.NewsORM
	if err := s.cfg.Database.Where("title = ?", req.GetTitle()).First(&existingNews).Error; err == nil {
		logger.Error("News with such title already exists")
		return nil, status.Error(codes.InvalidArgument, "News with such title already exists")
	} else if err != gorm.ErrRecordNotFound {
		logger.WithError(err).Error("Could not update news")
		return nil, status.Error(codes.Internal, "Could not update news")
	}

	if err := s.cfg.Database.Where("id = ?", req.GetId()).First(&existingNews).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			logger.Error("News with such title already exists")
			return nil, status.Error(codes.InvalidArgument, "News with such title already exists")
		}
		logger.WithError(err).Error("Could not update news")
		return nil, status.Error(codes.Internal, "Could not update news")
	}

	if req.GetTitle() != "" {
		existingNews.Title = req.GetTitle()
	}
	if req.GetDescription() != "" {
		existingNews.Description = req.GetDescription()
	}
	if req.GetImageLink() != "" {
		existingNews.ImageLink = req.GetImageLink()
	}

	if err := s.cfg.Database.Save(&existingNews).Error; err != nil {
		logger.WithError(err).Error("Could not update news")
		return nil, status.Error(codes.Internal, "Could not update news")
	}

	return &pb.UpdateNewsResponse{}, nil
}
