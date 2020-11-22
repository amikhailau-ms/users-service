package svc

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/amikhailau/users-service/pkg/auth"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/protobuf/ptypes"

	"github.com/amikhailau/users-service/pkg/pb"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UsersServerConfig struct {
	Database      *gorm.DB
	RSAPrivateKey *rsa.PrivateKey
	RSAPublicKey  *rsa.PublicKey
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

const (
	adminQuery = "SELECT is_admin FROM users WHERE id = $1"
)

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

	s.hideSensitiveInfo(&pbUser)
	logger.Debug("User registration finished")

	return &pb.CreateUserResponse{Result: &pbUser}, nil
}

func (s *UsersServer) Read(ctx context.Context, req *pb.ReadUserRequest) (*pb.ReadUserResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithField("provided_id", req.GetId())
	logger.Debug("Read user")

	claims, _ := auth.GetAuthorizationData(ctx)
	if !claims.IsAdmin && claims.UserId != req.GetId() && claims.UserName != req.GetId() && claims.UserEmail != req.GetId() {
		logger.Error("User can only use this endpoint for themselves")
		return nil, status.Error(codes.Unauthenticated, "Not authorized for another user")
	}

	usr, err := s.findUserByProvidedID(ctx, logger, req.GetId())
	if err != nil {
		return nil, err
	}
	s.hideSensitiveInfo(usr)
	return &pb.ReadUserResponse{Result: usr}, nil
}

func (s *UsersServer) Delete(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithField("id", req.GetId())
	logger.Debug("Delete user")

	claims, _ := auth.GetAuthorizationData(ctx)
	if !claims.IsAdmin && claims.UserId != req.GetId() {
		logger.Error("User can only use this endpoint for themselves")
		return nil, status.Error(codes.Unauthenticated, "Not authorized for another user")
	}

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

func (s *UsersServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithField("provided_id", req.GetId())
	logger.Debug("Login")

	usr, err := s.findUserByProvidedID(ctx, logger, req.GetId())
	if err != nil {
		logger.WithError(err).Error("Login failed")
		return nil, status.Error(codes.InvalidArgument, "Invalid login/password")
	}

	passwordBytes := []byte(req.GetPassword())
	encryptedPasswordBytes := sha256.Sum256(passwordBytes)
	tmpSlice := encryptedPasswordBytes[:]
	hexPassword := hex.EncodeToString(tmpSlice)

	if usr.Password != hexPassword {
		logger.Error("Login failed - wrong password")
		return nil, status.Error(codes.InvalidArgument, "Invalid login/password")
	}

	isAdmin := false
	if err := s.cfg.Database.DB().QueryRow(adminQuery, usr.GetId()).Scan(&isAdmin); err != nil {
		logger.WithError(err).Error("Failed to fetch is_admin attribute")
		return nil, status.Error(codes.Internal, "Unable to login")
	}

	expiresAt := time.Now().Add(8 * time.Hour)
	claims := &auth.GameClaims{
		UserId:    usr.GetId(),
		UserName:  usr.GetName(),
		UserEmail: usr.GetEmail(),
		IsAdmin:   isAdmin,
		StandardClaims: jwt.StandardClaims{
			Audience:  "medieval",
			ExpiresAt: expiresAt.Unix(),
			Id:        uuid.NewV4().String(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "users-service",
			NotBefore: time.Now().Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	tokenString, err := token.SignedString(s.cfg.RSAPrivateKey)
	if err != nil {
		logger.WithError(err).Error("Failed to sign claim")
		return nil, status.Error(codes.Internal, "Unable to login")
	}

	expiresAtPb, err := ptypes.TimestampProto(expiresAt)
	if err != nil {
		logger.WithError(err).Error("Failed to convert unix time to proto timestamp")
		return nil, status.Errorf(codes.Internal, "Unable to login")
	}

	return &pb.LoginResponse{Token: tokenString, ExpiresAt: expiresAtPb, IsAdmin: isAdmin}, nil
}

func (s *UsersServer) GrantCurrencies(ctx context.Context, req *pb.GrantCurrenciesRequest) (*pb.GrantCurrenciesResponse, error) {
	logger := ctxlogrus.Extract(ctx).WithField("provided_id", req.GetId())
	logger.Debug("Grant Currencies")

	usr, err := s.findUserByProvidedID(ctx, logger, req.GetId())
	if err != nil {
		return nil, err
	}

	usrORM := pb.UserORM{Id: usr.Id}
	if err := s.cfg.Database.Model(&usrORM).
		Updates(map[string]interface{}{"coins": usr.GetCoins() + req.GetAddCoins(), "gems": usr.GetGems() + req.GetAddGems()}).Error; err != nil {
		logger.WithError(err).Error("Unable to grant currencies")
		return nil, status.Error(codes.Internal, "Unable to grant currencies")
	}

	return &pb.GrantCurrenciesResponse{}, nil
}

func (s *UsersServer) hideSensitiveInfo(usr *pb.User) {
	usr.Password = ""
}

func (s *UsersServer) findUserByProvidedID(ctx context.Context, logger *logrus.Entry, providedID string) (*pb.User, error) {
	var existingUser pb.UserORM
	searchCriterias := []string{"id", "name", "email"}
	userFound := false
	for _, searchCriteria := range searchCriterias {
		if err := s.cfg.Database.Where(fmt.Sprintf("%v = ?", searchCriteria), providedID).First(&existingUser).Error; err != nil {
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
		return &pbUser, nil
	}

	logger.Error("Could not find user by any criteria")
	return nil, status.Error(codes.NotFound, "Could not find user")
}
