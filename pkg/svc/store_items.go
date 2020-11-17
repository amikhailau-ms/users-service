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

type StoreItemsServerConfig struct {
	Database *gorm.DB
}

type StoreItemsServer struct {
	pb.StoreItemsServer
	cfg *StoreItemsServerConfig
}

var _ pb.StoreItemsServer = &StoreItemsServer{}

func NewStoreItemsServer(cfg *StoreItemsServerConfig) (*StoreItemsServer, error) {
	return &StoreItemsServer{
		StoreItemsServer: &pb.StoreItemsDefaultServer{},
		cfg:              cfg,
	}, nil
}

func (s *StoreItemsServer) Create(ctx context.Context, req *pb.CreateStoreItemRequest) (*pb.CreateStoreItemResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithFields(logrus.Fields{
		"name":     req.GetName(),
		"image_id": req.GetImageId(),
	})
	logger.Debug("Create Item")

	if err := s.checkIfItemExists(logger, req.GetName(), req.GetImageId(), req.GetType()); err != nil {
		return nil, err
	}

	newItem := pb.StoreItemORM{
		Id:          uuid.NewV4().String(),
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Type:        req.GetType(),
		CoinsPrice:  req.GetCoinsPrice(),
		GemsPrice:   req.GetGemsPrice(),
		ImageId:     req.GetImageId(),
	}

	if err := s.cfg.Database.Create(&newItem).Error; err != nil {
		logger.WithError(err).Error("Could not create new item")
		return nil, status.Error(codes.Internal, "Could not create new item")
	}

	pbItem, err := newItem.ToPB(ctx)
	if err != nil {
		logger.WithError(err).Error("Could not create new item")
		return nil, status.Error(codes.Internal, "Could not create new item")
	}

	return &pb.CreateStoreItemResponse{Result: &pbItem}, nil
}

func (s *StoreItemsServer) Read(ctx context.Context, req *pb.ReadStoreItemRequest) (*pb.ReadStoreItemResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithFields(logrus.Fields{
		"id": req.GetId(),
	})
	logger.Debug("Read Item")

	res, err := pb.DefaultReadStoreItem(ctx, &pb.StoreItem{Id: req.GetId()}, s.cfg.Database)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.WithError(err).Error("Item not found")
			return nil, status.Error(codes.NotFound, "Item not found")
		}
		logger.WithError(err).Error("Could not read item")
		return nil, status.Error(codes.Internal, "Could not read item")
	}

	return &pb.ReadStoreItemResponse{Result: res}, nil
}

func (s *StoreItemsServer) Update(ctx context.Context, req *pb.UpdateStoreItemRequest) (*pb.UpdateStoreItemResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithFields(logrus.Fields{
		"id": req.GetPayload().GetId(),
	})
	logger.Debug("Update Item")

	if err := s.checkIfItemExists(logger, req.GetPayload().GetName(), req.GetPayload().GetImageId(), req.GetPayload().GetType()); err != nil {
		return nil, err
	}

	res, err := pb.DefaultStrictUpdateStoreItem(ctx, req.GetPayload(), s.cfg.Database)
	if err != nil {
		logger.WithError(err).Error("Could not update item")
		return nil, status.Error(codes.Internal, "Could not update item")
	}

	return &pb.UpdateStoreItemResponse{Result: res}, nil
}

func (s *StoreItemsServer) Delete(ctx context.Context, req *pb.DeleteStoreItemRequest) (*pb.DeleteStoreItemResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithField("id", req.GetId())
	logger.Debug("Delete Item")

	var item pb.StoreItemORM
	if err := s.cfg.Database.Where("id = ?", req.GetId()).Delete(&item).Error; err != nil && err != gorm.ErrRecordNotFound {
		logger.WithError(err).Error("Could not delete item")
		return nil, status.Error(codes.Internal, "Could not delete item")
	}

	return &pb.DeleteStoreItemResponse{}, nil
}

func (s *StoreItemsServer) List(ctx context.Context, req *pb.ListStoreItemsRequest) (*pb.ListStoreItemsResponse, error) {
	logger := ctxlogrus.Extract(ctx)
	logger.Debug("List Items")

	res, err := pb.DefaultListStoreItem(ctx, s.cfg.Database, req.GetFilter(), req.GetOrderBy(), req.GetPaging(), req.GetFields())
	if err != nil {
		logger.WithError(err).Error("Could not list items")
		return nil, status.Error(codes.Internal, "Could not list items")
	}

	return &pb.ListStoreItemsResponse{Results: res}, nil
}

func (s *StoreItemsServer) BuyByUser(ctx context.Context, req *pb.BuyByUserRequest) (*pb.BuyByUserResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithFields(logrus.Fields{
		"user_id": req.GetUserId(),
		"item_id": req.GetItemId(),
	})
	logger.Debug("Buying item")

	var usr pb.UserORM
	var item pb.StoreItemORM

	if err := s.cfg.Database.Where("id = ?", req.GetUserId()).First(&usr).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Error("User not found")
			return nil, status.Error(codes.NotFound, "User not found")
		}
		logger.WithError(err).Error("Could not find user")
		return nil, status.Error(codes.Internal, "Could not find user")
	}

	if err := s.cfg.Database.Where("id = ?", req.GetItemId()).First(&item).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Error("Item not found")
			return nil, status.Error(codes.NotFound, "Item not found")
		}
		logger.WithError(err).Error("Could not find item")
		return nil, status.Error(codes.Internal, "Could not find item")
	}

	if usr.Gems < item.GemsPrice {
		logger.Error("Not enough gems")
		return nil, status.Error(codes.InvalidArgument, "Not enough gems")
	}

	if usr.Coins < item.CoinsPrice {
		logger.Error("Not enough coins")
		return nil, status.Error(codes.InvalidArgument, "Not enough coins")
	}

	txnDB := s.cfg.Database.Begin()

	usr.Gems -= item.GemsPrice
	usr.Coins -= item.CoinsPrice

	if err := txnDB.Save(&usr).Error; err != nil {
		txnDB.Rollback()
		logger.WithError(err).Error("Could not proceed with the operation")
		return nil, status.Error(codes.Internal, "Could not proceed with the operation")
	}

	if err := txnDB.Model(&usr).Association("Items").Append(&item).Error; err != nil {
		txnDB.Rollback()
		logger.WithError(err).Error("Could not proceed with the operation")
		return nil, status.Error(codes.Internal, "Could not proceed with the operation")
	}

	txnDB.Commit()

	return &pb.BuyByUserResponse{}, nil
}

func (s *StoreItemsServer) GetUserItemsIds(ctx context.Context, req *pb.GetUserItemsIdsRequest) (*pb.GetUserItemsIdsResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithField("user_id", req.GetUserId())
	logger.Debug("GetUserItemsIds")

	var usr pb.UserORM
	if err := s.cfg.Database.Where("id = ?", req.GetUserId()).First(&usr).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Error("User not found")
			return nil, status.Error(codes.NotFound, "User not found")
		}
		logger.WithError(err).Error("Could not find user")
		return nil, status.Error(codes.Internal, "Could not find user")
	}

	var items []*pb.StoreItemORM
	if err := s.cfg.Database.Select("id").Model(&usr).Association("Items").Find(&items).Error; err != nil {
		logger.WithError(err).Error("Could not fetch user items")
		return nil, status.Error(codes.Internal, "Could not fetch user items")
	}

	itemsIDs := make([]string, 0, len(items))
	for _, item := range items {
		itemsIDs = append(itemsIDs, item.Id)
	}

	return &pb.GetUserItemsIdsResponse{ItemIds: itemsIDs}, nil
}

func (s *StoreItemsServer) checkIfItemExists(logger *logrus.Entry, name, image_id string, item_type int64) error {

	var existingItem pb.StoreItemORM
	if err := s.cfg.Database.Where("name = ? AND type = ?", name, item_type).First(&existingItem).Error; err == nil {
		logger.Error("Item with such name and type already exists")
		return status.Error(codes.InvalidArgument, "Item with such name and type already exists")
	} else if err != gorm.ErrRecordNotFound {
		logger.WithError(err).Error("Could not create new item")
		return status.Error(codes.Internal, "Could not create new item")
	}

	if err := s.cfg.Database.Where("image_id = ?", image_id).First(&existingItem).Error; err == nil {
		logger.Error("Item with such image id already exists")
		return status.Error(codes.InvalidArgument, "Item with such image id already exists")
	} else if err != gorm.ErrRecordNotFound {
		logger.WithError(err).Error("Could not create new item")
		return status.Error(codes.Internal, "Could not create new item")
	}

	return nil
}