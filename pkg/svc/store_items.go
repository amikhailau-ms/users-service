package svc

import (
	"context"
	"fmt"
	"strings"

	"github.com/amikhailau/users-service/pkg/auth"
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

const (
	deequipQuery           = "UPDATE users_store_items SET equipped = 'f' WHERE user_id = $1 AND store_item_id = $2"
	equipQuery             = "UPDATE users_store_items SET equipped = 't' WHERE user_id = $1 AND store_item_id = $2"
	userItemsQuery         = "SELECT store_item_id, equipped FROM users_store_items WHERE user_id = $1"
	equippedUserItemsQuery = "SELECT store_item_id, equipped FROM users_store_items WHERE user_id = $1 AND equipped = 't'"
	findEquippedQuery      = "SELECT si.id FROM store_items si JOIN users_store_items usi ON usi.store_item_id = si.id WHERE si.type = $1 AND usi.user_id = $2 AND usi.equipped = $3"
)

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
		Id:             uuid.NewV4().String(),
		Name:           req.GetName(),
		Description:    req.GetDescription(),
		Type:           req.GetType(),
		CoinsPrice:     req.GetCoinsPrice(),
		GemsPrice:      req.GetGemsPrice(),
		ImageId:        req.GetImageId(),
		OnSale:         req.GetOnSale(),
		SaleCoinsPrice: req.GetSaleCoinsPrice(),
		SaleGemsPrice:  req.GetSaleGemsPrice(),
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

	resp, err := s.Read(ctx, &pb.ReadStoreItemRequest{Id: req.GetPayload().GetId()})
	if err != nil {
		return nil, err
	}

	if err := s.checkIfItemExists(logger, req.GetPayload().GetName(), req.GetPayload().GetImageId(), req.GetPayload().GetType()); err != nil {
		return nil, err
	}

	var gormReq *pb.UpdateStoreItemRequest
	if req.GetFields() != nil {
		gormReq = req
	} else {
		item := resp.GetResult()
		if req.GetPayload().GetCoinsPrice() != 0 {
			item.CoinsPrice = req.GetPayload().GetCoinsPrice()
		}
		if req.GetPayload().GetGemsPrice() != 0 {
			item.GemsPrice = req.GetPayload().GetGemsPrice()
		}
		if req.GetPayload().GetImageId() != "" {
			item.ImageId = req.GetPayload().GetImageId()
		}
		if req.GetPayload().GetName() != "" {
			item.Name = req.GetPayload().GetName()
		}
		if req.GetPayload().GetDescription() != "" {
			item.Description = req.GetPayload().GetDescription()
		}
		if req.GetPayload().GetSaleCoinsPrice() != 0 || req.GetPayload().GetSaleGemsPrice() != 0 {
			item.OnSale = true
			item.SaleCoinsPrice = req.GetPayload().GetSaleCoinsPrice()
			item.SaleGemsPrice = req.GetPayload().GetSaleGemsPrice()
		}
		gormReq = &pb.UpdateStoreItemRequest{Payload: item}
	}

	fmt.Printf("Value: %v", gormReq.GetPayload())

	defaultItemsServer := &pb.StoreItemsDefaultServer{DB: s.cfg.Database}
	res, err := defaultItemsServer.Update(ctx, gormReq)
	if err != nil {
		logger.WithError(err).Error("Could not update item")
		return nil, status.Error(codes.Internal, "Could not update item")
	}

	return res, nil
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

	claims, _ := auth.GetAuthorizationData(ctx)
	if !claims.IsAdmin && claims.UserId != req.GetUserId() {
		logger.Error("User can only use this endpoint for themselves")
		return nil, status.Error(codes.Unauthenticated, "Not authorized for another user")
	}

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

	if item.OnSale {
		if usr.Gems < item.SaleGemsPrice {
			logger.Error("Not enough gems")
			return nil, status.Error(codes.InvalidArgument, "Not enough gems")
		}

		if usr.Coins < item.SaleCoinsPrice {
			logger.Error("Not enough coins")
			return nil, status.Error(codes.InvalidArgument, "Not enough coins")
		}

		usr.Gems -= item.SaleGemsPrice
		usr.Coins -= item.SaleCoinsPrice
	} else {
		if usr.Gems < item.GemsPrice {
			logger.Error("Not enough gems")
			return nil, status.Error(codes.InvalidArgument, "Not enough gems")
		}

		if usr.Coins < item.CoinsPrice {
			logger.Error("Not enough coins")
			return nil, status.Error(codes.InvalidArgument, "Not enough coins")
		}

		usr.Gems -= item.GemsPrice
		usr.Coins -= item.CoinsPrice
	}

	txnDB := s.cfg.Database.Begin()

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

	claims, _ := auth.GetAuthorizationData(ctx)
	if !claims.IsAdmin && claims.StandardClaims.Audience != "svc" && claims.UserId != req.GetUserId() {
		logger.Error("User can only use this endpoint for themselves")
		return nil, status.Error(codes.Unauthenticated, "Not authorized for another user")
	}

	var usr pb.UserORM
	if err := s.cfg.Database.Where("id = ?", req.GetUserId()).First(&usr).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Error("User not found")
			return nil, status.Error(codes.NotFound, "User not found")
		}
		logger.WithError(err).Error("Could not find user")
		return nil, status.Error(codes.Internal, "Could not find user")
	}

	items := []*pb.UserItemInfo{}
	rows, err := s.cfg.Database.DB().Query(userItemsQuery, req.GetUserId())
	if err != nil {
		logger.WithError(err).Error("Could not fetch user items")
		return nil, status.Error(codes.Internal, "Could not fetch user items")
	}
	for rows.Next() {
		item := pb.UserItemInfo{}
		err := rows.Scan(&item.ItemId, &item.Equipped)
		if err != nil {
			logger.WithError(err).Error("Could not fetch user items")
			return nil, status.Error(codes.Internal, "Could not fetch user items")
		}
		items = append(items, &item)
	}

	return &pb.GetUserItemsIdsResponse{Items: items}, nil
}

func (s *StoreItemsServer) GetEquippedUserItemsIds(ctx context.Context, req *pb.GetEquippedUserItemsIdsRequest) (*pb.GetEquippedUserItemsIdsResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithField("user_id", req.GetUserId())
	logger.Debug("GetEquippedUserItemsIds")

	claims, _ := auth.GetAuthorizationData(ctx)
	if !claims.IsAdmin && claims.StandardClaims.Audience != "svc" && claims.UserId != req.GetUserId() {
		logger.Error("User can only use this endpoint for themselves")
		return nil, status.Error(codes.Unauthenticated, "Not authorized for another user")
	}

	var usr pb.UserORM
	if err := s.cfg.Database.Where("id = ?", req.GetUserId()).First(&usr).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Error("User not found")
			return nil, status.Error(codes.NotFound, "User not found")
		}
		logger.WithError(err).Error("Could not find user")
		return nil, status.Error(codes.Internal, "Could not find user")
	}

	items := []*pb.UserItemInfo{}
	rows, err := s.cfg.Database.DB().Query(equippedUserItemsQuery, req.GetUserId())
	if err != nil {
		logger.WithError(err).Error("Could not fetch user items")
		return nil, status.Error(codes.Internal, "Could not fetch user items")
	}
	for rows.Next() {
		item := pb.UserItemInfo{}
		err := rows.Scan(&item.ItemId, &item.Equipped)
		if err != nil {
			logger.WithError(err).Error("Could not fetch user items")
			return nil, status.Error(codes.Internal, "Could not fetch user items")
		}
		items = append(items, &item)
	}

	return &pb.GetEquippedUserItemsIdsResponse{Items: items}, nil
}

func (s *StoreItemsServer) EquipByUser(ctx context.Context, req *pb.EquipByUserRequest) (*pb.EquipByUserResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithFields(logrus.Fields{
		"user_id": req.GetUserId(),
		"item_id": req.GetItemId(),
	})
	logger.Debug("Buying item")

	claims, _ := auth.GetAuthorizationData(ctx)
	if !claims.IsAdmin && claims.UserId != req.GetUserId() {
		logger.Error("User can only use this endpoint for themselves")
		return nil, status.Error(codes.Unauthenticated, "Not authorized for another user")
	}

	item, err := s.Read(ctx, &pb.ReadStoreItemRequest{Id: req.GetItemId()})
	if err != nil {
		return nil, err
	}

	var itemEquippedId string
	found := true
	if err := s.cfg.Database.DB().QueryRow(findEquippedQuery, item.GetResult().GetType(), req.GetUserId(), true).Scan(&itemEquippedId); err != nil {
		if !strings.Contains(err.Error(), "no rows") {
			logger.WithError(err).Error("Could not fetch equipped item")
			return nil, status.Error(codes.Internal, "Could not equip item")
		}
		found = false
	}

	if found && itemEquippedId == item.GetResult().GetId() {
		logger.Debug("Item has been already equipped")
		return &pb.EquipByUserResponse{}, nil
	}

	txnDB, err := s.cfg.Database.DB().Begin()
	if err != nil {
		logger.WithError(err).Error("Could not start transaction")
		return nil, status.Error(codes.Internal, "Could not equip item")
	}

	if found {
		if _, err := txnDB.Exec(deequipQuery, req.GetUserId(), itemEquippedId); err != nil {
			txnDB.Rollback()
			logger.WithError(err).Error("Could not deequip item")
			return nil, status.Error(codes.Internal, "Could not equip item")
		}
	}

	if _, err := txnDB.Exec(equipQuery, req.GetUserId(), item.GetResult().GetId()); err != nil {
		txnDB.Rollback()
		logger.WithError(err).Error("Could not equip item")
		return nil, status.Error(codes.Internal, "Could not equip item")
	}

	txnDB.Commit()

	return &pb.EquipByUserResponse{}, nil
}

func (s *StoreItemsServer) checkIfItemExists(logger *logrus.Entry, name, image_id string, item_type int32) error {

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
