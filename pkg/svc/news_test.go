package svc

import (
	"context"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/amikhailau/users-service/pkg/pb"
	testutils "github.com/amikhailau/users-service/pkg/testing"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
)

func TestNews(t *testing.T) {
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

	newsServer, err := NewNewsServer(&NewsServerConfig{
		Database: gdb,
	})
	if err != nil {
		t.Fatalf("Could not create users server: %v", err)
	}
	pb.RegisterNewsServiceServer(server.GRPCServer, newsServer)

	conn, err := server.Serve(ctx, ":0")
	if err != nil {
		t.Fatalf("Could not start test server: %v", err)
	}
	defer server.Close()

	newsClient := pb.NewNewsServiceClient(conn)
	newNewsData := pb.News{
		Title:       "First",
		Description: "Second",
		ImageLink:   "http://placeholder",
	}
	cRequest := &pb.CreateNewsRequest{
		Title:       newNewsData.Title,
		Description: newNewsData.Description,
		ImageLink:   newNewsData.ImageLink,
	}

	sqlSearchID := `SELECT * FROM "news" WHERE ("news"."id" = $1) ORDER BY "news"."id" ASC LIMIT 1`
	sqlSearchID2 := `SELECT * FROM "news" WHERE (id = $1) ORDER BY "news"."id" ASC LIMIT 1`
	sqlSearchTitle := `SELECT * FROM "news" WHERE (title = $1) ORDER BY "news"."id" ASC LIMIT 1`
	sqlBackSearch := `SELECT * FROM "news"  WHERE "news"."id" = $1 AND ((title = $2)) ORDER BY "news"."id" ASC LIMIT 1`
	sqlCreateNews := `INSERT INTO "news" ("created_at","description","id","image_link","title") VALUES ($1,$2,$3,$4,$5) RETURNING "news"."id"`
	sqlUpdateNews := `UPDATE "news" SET "created_at" = $1, "description" = $2, "image_link" = $3, "title" = $4  WHERE "news"."id" = $5`

	t.Run("Create News - positive", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "title", "description", "image_link", "created_at"}).
			AddRow("some-id", "title", "description", "http://placeholder", time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchTitle)).WithArgs(newNewsData.Title).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(sqlCreateNews)).WithArgs(sqlmock.AnyArg(), newNewsData.Description, sqlmock.AnyArg(),
			newNewsData.ImageLink, newNewsData.Title).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectCommit()
		mock.ExpectQuery(regexp.QuoteMeta(sqlBackSearch)).WithArgs(sqlmock.AnyArg(), newNewsData.Title).WillReturnRows(rows)

		_, err := newsClient.Create(ctx, cRequest)
		if err != nil {
			t.Fatalf("error creating new user: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}
	})

	t.Run("Create News - link exists", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "title", "description", "image_link", "created_at"}).
			AddRow("some-id", "title", "description", "http://placeholder", time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchTitle)).WithArgs(newNewsData.Title).WillReturnRows(rows)

		_, err := newsClient.Create(ctx, cRequest)
		if err == nil {
			t.Fatalf("expecting error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}
	})

	t.Run("Read News - found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "title", "description", "image_link", "created_at"}).
			AddRow("some-id", "title", "description", "http://placeholder", time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchID)).WithArgs("some-id").WillReturnRows(rows)

		_, err := newsClient.Read(ctx, &pb.ReadNewsRequest{
			Id: "some-id",
		})
		if err != nil {
			t.Fatalf("error reading user: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}
	})

	t.Run("Read News - not found", func(t *testing.T) {

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchID)).WithArgs("some-id").WillReturnRows(sqlmock.NewRows(nil))

		_, err := newsClient.Read(ctx, &pb.ReadNewsRequest{
			Id: "some-id",
		})
		if err == nil {
			t.Fatalf("expecting error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}
	})

	t.Run("Update Item - positive", func(t *testing.T) {

		rows := sqlmock.NewRows([]string{"id", "title", "description", "image_link", "created_at"}).
			AddRow("some-id", "title", "description", "http://placeholder", time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchTitle)).WithArgs("new-title").WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchID2)).WithArgs("some-id").WillReturnRows(rows)
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(sqlUpdateNews)).WithArgs(sqlmock.AnyArg(), "description", "http://placeholder",
			"new-title", "some-id").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		_, err := newsClient.Update(ctx, &pb.UpdateNewsRequest{
			Id:    "some-id",
			Title: "new-title",
		})
		if err != nil {
			t.Fatalf("error updating new item: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}

	})

	t.Run("Update Item - title exists", func(t *testing.T) {

		rows := sqlmock.NewRows([]string{"id", "title", "description", "image_link", "created_at"}).
			AddRow("some-id", "new-title", "description", "http://placeholder", time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchTitle)).WithArgs("new-title").WillReturnRows(rows)

		_, err := newsClient.Update(ctx, &pb.UpdateNewsRequest{
			Id:    "some-id",
			Title: "new-title",
		})
		if err == nil {
			t.Fatalf("expecting error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}

	})

	t.Run("Update Item - not found", func(t *testing.T) {

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchTitle)).WithArgs("new-title").WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchID2)).WithArgs("some-id").WillReturnRows(sqlmock.NewRows(nil))

		_, err := newsClient.Update(ctx, &pb.UpdateNewsRequest{
			Id:    "some-id",
			Title: "new-title",
		})
		if err == nil {
			t.Fatalf("expecting error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}

	})
}
