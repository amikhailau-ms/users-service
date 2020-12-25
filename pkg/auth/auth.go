package auth

import (
	"context"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	svcEndpoints = []string{"Users/GrantCurrencies", "UsersService/GetVersion", "StoreItems/Create", "StoreItems/Update",
		"StoreItems/ThrowAwayByUser", "StoreItems/Delete", "UsersStats/UpdateStats", "NewsService/Create", "NewsService/Update"}
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		logger := ctxlogrus.Extract(ctx)
		logger.Debug("Authorization interceptor")

		str := strings.Split(info.FullMethod, ".")
		method := str[len(str)-1]

		if method != "Users/Create" && method != "Users/Login" {
			logger.Debug("Checking claims")
			claims, err := GetAuthorizationData(ctx)
			if err != nil {
				return nil, status.Error(codes.Unauthenticated, "Authorization failed - invalid header/token")
			}
			if claims.ExpiresAt < time.Now().Unix() {
				return nil, status.Error(codes.Unauthenticated, "Authorization failed - token expired")
			}
			logger.WithField("claims", claims).Debug("Incoming claims")
			requiresHighLevelAccess := false
			for _, svcEndpoint := range svcEndpoints {
				if svcEndpoint == method {
					requiresHighLevelAccess = true
					break
				}
			}
			if requiresHighLevelAccess && (!claims.IsAdmin && claims.StandardClaims.Audience != "svc") {
				return nil, status.Error(codes.Unauthenticated, "Authorization failed - high level access required")
			}
		}

		return handler(ctx, req)
	}
}

func GetAuthorizationData(ctx context.Context) (*GameClaims, error) {
	logger := ctxlogrus.Extract(ctx)
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		logger.WithError(err).Error("Token not found")
		return nil, err
	}
	claims := &GameClaims{}
	parser := &jwt.Parser{}
	_, _, err = parser.ParseUnverified(token, claims)
	if err != nil {
		logger.WithError(err).Error("Not able to parse token")
		return nil, err
	}
	return claims, nil
}
