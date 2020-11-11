package svc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"testing"

	"github.com/amikhailau/users-service/pkg/pb"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	testutils "github.com/amikhailau/users-service/pkg/testing"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

func TestUsers(t *testing.T) {
	logger := logrus.New()
	ctx := ctxlogrus.ToContext(context.TODO(), logrus.NewEntry(logger))

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

	conn, err := server.Serve(ctx, ":0")
	if err != nil {
		t.Fatalf("Could not start test server: %v", err)
	}
	defer server.Close()

	usrClient := pb.NewUsersClient(conn)

	t.Run("Create User - positive", func(t *testing.T) {
		newUserData := pb.User{
			Name:     "Prothean",
			Email:    "someemail@email.com",
			Password: "SomePassword1",
		}

		passwordBytes := []byte(newUserData.Password)
		encryptedPasswordBytes := sha256.Sum256(passwordBytes)
		tmpSlice := encryptedPasswordBytes[:]
		hexPassword := hex.EncodeToString(tmpSlice)

		sqlSearchName := `SELECT * FROM "users" WHERE (name = $1) ORDER BY "users"."id" ASC LIMIT 1`
		sqlSearchEmail := `SELECT * FROM "users" WHERE (email = $1) ORDER BY "users"."id" ASC LIMIT 1`
		sqlCreateUser := `INSERT INTO "users" ("coins","email","gems","id","name","password") VALUES ($1,$2,$3,$4,$5,$6) RETURNING "users"."id"`
		sqlCreateStats := `INSERT INTO "user_stats" ("games","kills","top5","user_id","wins") VALUES ($1,$2,$3,$4,$5) RETURNING "user_stats"."id"`
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchName)).WithArgs(newUserData.Name).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchEmail)).WithArgs(newUserData.Email).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(sqlCreateUser)).WithArgs(0, newUserData.Email,
			0, sqlmock.AnyArg(), newUserData.Name, hexPassword).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectQuery(regexp.QuoteMeta(sqlCreateStats)).WithArgs(0, 0, 0, sqlmock.AnyArg(), 0).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectCommit()

		_, err := usrClient.Create(ctx, &pb.CreateUserRequest{
			Name:     newUserData.Name,
			Email:    newUserData.Email,
			Password: newUserData.Password,
		})
		if err != nil {
			t.Fatalf("error creating new user: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}
	})
}
