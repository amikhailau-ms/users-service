package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/amikhailau/users-service/db"
	"github.com/amikhailau/users-service/pkg/pb"
	"github.com/golang/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/infobloxopen/atlas-app-toolkit/gateway"
	"github.com/infobloxopen/atlas-app-toolkit/requestid"
	"github.com/infobloxopen/atlas-app-toolkit/server"

	"github.com/infobloxopen/atlas-app-toolkit/gorm/resource"
)

func main() {
	doneC := make(chan error)
	logger := NewLogger()

	connURL := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(*flagDatabaseUser, *flagDatabasePassword),
		Host:     fmt.Sprintf("%s:%s", *flagDatabaseAddress, *flagDatabasePort),
		Path:     "/" + *flagDatabaseName,
		RawQuery: fmt.Sprintf("sslmode=%s", *flagDatabaseSSL),
	}

	fileURL := &url.URL{
		Scheme: "file",
		Path:   *flagDatabaseMigration,
	}

	logger.WithFields(logrus.Fields{
		"db":     connURL.Hostname(),
		"schema": fileURL.String(),
	}).Warn("migrating db")
	if err := db.Migrate(connURL, fileURL); err != nil {
		if !strings.HasSuffix(err.Error(), "no change") {
			logger.WithFields(logrus.Fields{
				"db":     connURL.String(),
				"schema": fileURL.String(),
			}).WithError(err).Fatal("failed to migrate db")
		}
	}
	logger.WithFields(logrus.Fields{
		"db":     connURL.Hostname(),
		"schema": fileURL.String(),
	}).Warn("migrated db")

	go func() { doneC <- ServeExternal(logger) }()

	if err := <-doneC; err != nil {
		logger.Fatal(err)
	}
}

func NewLogger() *logrus.Logger {
	logger := logrus.StandardLogger()
	logrus.SetFormatter(&logrus.JSONFormatter{})

	// Set the log level on the default logger based on command line flag
	logLevels := map[string]logrus.Level{
		"debug":   logrus.DebugLevel,
		"info":    logrus.InfoLevel,
		"warning": logrus.WarnLevel,
		"error":   logrus.ErrorLevel,
		"fatal":   logrus.FatalLevel,
		"panic":   logrus.PanicLevel,
	}
	if level, ok := logLevels[viper.GetString("logging.level")]; !ok {
		logger.Errorf("Invalid %q provided for log level", viper.GetString("logging.level"))
		logger.SetLevel(logrus.InfoLevel)
	} else {
		logger.SetLevel(level)
	}

	return logger
}

// ServeExternal builds and runs the server that listens on ServerAddress and GatewayAddress
func ServeExternal(logger *logrus.Logger) error {

	if viper.GetString("database.dsn") == "" {
		setDBConnection()
	}
	grpcServer, err := NewGRPCServer(logger, viper.GetString("database.dsn"))
	if err != nil {
		logger.Fatalln(err)
	}

	s, err := server.NewServer(
		server.WithGrpcServer(grpcServer),
		server.WithGateway(
			gateway.WithGatewayOptions(
				runtime.WithForwardResponseOption(forwardResponseOption),
				runtime.WithIncomingHeaderMatcher(gateway.ExtendedDefaultHeaderMatcher(
					requestid.DefaultRequestIDKey)),
			),
			gateway.WithServerAddress(fmt.Sprintf("%s:%s", viper.GetString("server.address"), viper.GetString("server.port"))),
			gateway.WithEndpointRegistration(viper.GetString("gateway.endpoint"), pb.RegisterUsersServiceHandlerFromEndpoint),
			gateway.WithEndpointRegistration(viper.GetString("gateway.endpoint"), pb.RegisterUsersHandlerFromEndpoint),
			gateway.WithEndpointRegistration(viper.GetString("gateway.endpoint"), pb.RegisterUsersStatsHandlerFromEndpoint),
			gateway.WithEndpointRegistration(viper.GetString("gateway.endpoint"), pb.RegisterNewsServiceHandlerFromEndpoint),
			gateway.WithEndpointRegistration(viper.GetString("gateway.endpoint"), pb.RegisterStoreItemsHandlerFromEndpoint),
		),
		server.WithHandler("/swagger/", NewSwaggerHandler(viper.GetString("gateway.swaggerFile"))),
	)
	if err != nil {
		logger.Fatalln(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = viper.GetString("gateway.port")
	}

	grpcL, err := net.Listen("tcp", fmt.Sprintf("%s:%s", viper.GetString("server.address"), viper.GetString("server.port")))
	if err != nil {
		logger.Fatalln(err)
	}

	httpL, err := net.Listen("tcp", fmt.Sprintf("%s:%s", viper.GetString("gateway.address"), port))
	if err != nil {
		logger.Fatalln(err)
	}

	logger.Printf("serving gRPC at %s:%s", viper.GetString("server.address"), viper.GetString("server.port"))
	logger.Printf("serving http at %s:%s", viper.GetString("gateway.address"), port)

	return s.Serve(grpcL, httpL)
}

func init() {
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AddConfigPath(viper.GetString("config.source"))
	if viper.GetString("config.file") != "" {
		log.Printf("Serving from configuration file: %s", viper.GetString("config.file"))
		viper.SetConfigName(viper.GetString("config.file"))
		if err := viper.ReadInConfig(); err != nil {
			log.Fatalf("cannot load configuration: %v", err)
		}
	} else {
		log.Printf("Serving from default values, environment variables, and/or flags")
	}
	resource.RegisterApplication(viper.GetString("app.id"))
	resource.SetPlural()
}

func forwardResponseOption(ctx context.Context, w http.ResponseWriter, resp proto.Message) error {
	w.Header().Set("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate")
	return nil
}

// setDBConnection sets the db connection string
func setDBConnection() {
	viper.Set("database.dsn", fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=%s dbname=%s",
		viper.GetString("database.address"), viper.GetString("database.port"),
		viper.GetString("database.user"), viper.GetString("database.password"),
		viper.GetString("database.ssl"), viper.GetString("database.name")))
}
