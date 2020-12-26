package svc

import (
	"context"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/amikhailau/users-service/pkg/pb"
	testutils "github.com/amikhailau/users-service/pkg/testing"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
)

func TestUsersStats(t *testing.T) {
	logger := logrus.New()
	ctx := ctxlogrus.ToContext(context.TODO(), logrus.NewEntry(logger))
	ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{"Authorization": "Bearer " + token}))

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Could not create mock db: %v", err)
	}
	gdb, err := gorm.Open("postgres", db)
	if err != nil {
		t.Fatalf("Could not create mock gorm db: %v", err)
	}

	server := testutils.NewTestServer(gdb, logger)

	usrServer, err := NewUsersServer(&UsersServerConfig{
		Database: gdb,
	})
	if err != nil {
		t.Fatalf("Could not create users server: %v", err)
	}
	pb.RegisterUsersServer(server.GRPCServer, usrServer)

	stServer, err := NewUsersStatsServer(&UsersStatsServerConfig{
		Database:    gdb,
		UsersServer: usrServer,
	})
	if err != nil {
		t.Fatalf("Could not create users stats server: %v", err)
	}
	pb.RegisterUsersStatsServer(server.GRPCServer, stServer)

	conn, err := server.Serve(ctx, ":0")
	if err != nil {
		t.Fatalf("Could not start test server: %v", err)
	}
	defer server.Close()

	stClient := pb.NewUsersStatsClient(conn)

	userSqlSearchID := `SELECT * FROM "users" WHERE (id = $1) ORDER BY "users"."id" ASC LIMIT 1`
	sqlSearchID := `SELECT * FROM "user_stats"  WHERE ("user_id" = $1)`
	sqlUpdate := `UPDATE "user_stats" SET "games" = $1, "kills" = $2, "top5" = $3, "user_id" = $4, "wins" = $5  WHERE "user_stats"."id" = $6`

	t.Run("Get stats - positive", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "password", "created_at", "updated_at", "name", "coins", "gems", "is_admin"}).
			AddRow("some-id", "someemail@email.com", "some-hash", "2020-01-01 01:05:57", "2020-01-01 01:05:57", "some-name", 0, 0, 'f')
		statsRows := sqlmock.NewRows([]string{"id", "wins", "top5", "kills", "games"}).
			AddRow(1, 10, 10, 100, 20)

		mock.ExpectQuery(regexp.QuoteMeta(userSqlSearchID)).WithArgs("some-id").WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchID)).WithArgs("some-id").WillReturnRows(statsRows)

		_, err := stClient.GetStats(ctx, &pb.ReadUserStatsRequest{
			Username: "some-id",
		})
		if err != nil {
			t.Fatalf("error reading user stats: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}
	})

	t.Run("Update stats - positive", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "password", "created_at", "updated_at", "name", "coins", "gems", "is_admin"}).
			AddRow("some-id", "someemail@email.com", "some-hash", "2020-01-01 01:05:57", "2020-01-01 01:05:57", "some-name", 0, 0, 'f')
		statsRows := sqlmock.NewRows([]string{"id", "wins", "top5", "kills", "games"}).
			AddRow(1, 10, 10, 100, 20)

		mock.ExpectQuery(regexp.QuoteMeta(userSqlSearchID)).WithArgs("some-id").WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchID)).WithArgs("some-id").WillReturnRows(statsRows)
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(sqlUpdate)).WithArgs(21, 102, 10, nil, 10, 1).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		_, err := stClient.UpdateStats(ctx, &pb.UpdateUserStatsRequest{
			Username: "some-id",
			AddGames: 1,
			AddKills: 2,
		})
		if err != nil {
			t.Fatalf("error updating user stats: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}
	})
}
