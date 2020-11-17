package testutils

import (
	"context"
	"net"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/infobloxopen/atlas-app-toolkit/gateway"
	"github.com/infobloxopen/atlas-app-toolkit/requestid"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type testserver struct {
	GRPCServer *grpc.Server
	db         *gorm.DB
	logger     *logrus.Logger

	lis  net.Listener
	conn *grpc.ClientConn
}

func (s *testserver) Serve(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	var err error
	s.lis, err = net.Listen("tcp4", addr)
	if err != nil {
		return nil, err
	}

	go func() {
		s.logger.WithField("addr", s.lis.Addr()).Info("test server up")
		s.GRPCServer.Serve(s.lis)
	}()

	s.conn, err = grpc.DialContext(ctx, s.lis.Addr().String(), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	return s.conn, nil
}

func (s *testserver) DialContext(ctx context.Context, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	return grpc.DialContext(ctx, s.lis.Addr().String(), opts...)
}

func (s *testserver) Close() {
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	if s.lis != nil {
		s.lis.Close()
		s.lis = nil
	}
}

func NewTestServer(db *gorm.DB, logger *logrus.Logger) *testserver {

	interceptors := []grpc.UnaryServerInterceptor{

		grpc_logrus.UnaryServerInterceptor(logrus.NewEntry(logger)),

		requestid.UnaryServerInterceptor(),

		grpc_validator.UnaryServerInterceptor(),

		gateway.UnaryServerInterceptor(),
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				interceptors...,
			),
		),
	)

	return &testserver{
		GRPCServer: grpcServer,
		db:         db,
		logger:     logger,
	}
}
