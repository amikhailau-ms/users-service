package main

import (
	"github.com/amikhailau/users-service/pkg/pb"
	"github.com/infobloxopen/atlas-app-toolkit/gateway"
	"github.com/spf13/viper"
)

func RegisterGatewayEndpoints() gateway.Option {
	return gateway.WithEndpointRegistration(viper.GetString("server.version"),
		pb.RegisterUsersHandlerFromEndpoint,
		pb.RegisterItemsHandlerFromEndpoint,
	)
}
