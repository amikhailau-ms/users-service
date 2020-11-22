package svc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"testing"

	"google.golang.org/grpc/metadata"

	"github.com/amikhailau/users-service/pkg/pb"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	testutils "github.com/amikhailau/users-service/pkg/testing"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

const (
	token = "eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYTMyYzk2Y2QtOGNlZS00OTA2LTgyZjItODMwMmVhYTAzYTE5IiwidXNlcm5hbWUiOiJQcm90aGVhbiIsInVzZXJfZW1haWwiOiJhbnRob255Lm1pa2hheWxvdkBnbWFpbC5jb20iLCJpc19hZG1pbiI6dHJ1ZSwiYXVkIjoibWVkaWV2YWwiLCJleHAiOjE5Njc4NDQ4MTMsImp0aSI6ImE1YWMxOGU4LThiNTYtNDJjZS1iNDUzLWRiZTA0YzQzYzNmYyIsImlhdCI6MTYwNjA0NDgxMywiaXNzIjoidXNlcnMtc2VydmljZSIsIm5iZiI6MTYwNjA0NDgxM30.a6dJ3-XWMKgkBkuoanXBQJ4RNoBVuE-M9lJ5VNeCJuBHKQTcQ6PCjQ3_bCs0eML-RfYYnIfeUv-ggLkTwtNIOFmp9XgVwxED_IT2t1WKZATWP7BwbnlefE_sXxjHCKsbO2adI4kjfAW_fIn989_m4AR_QXe9fbHln_l_cTp-4fj26WhozlmYnnBebwUtv2uk1FTnzpYLunVWaATnrSP7eTM0IzvbqigV0-4d_Xn_b3GCvj8hl1KlhkW5IuG2d3c23RmAElHQ0uVazmIZq35Fw1wFgJPaJIm4Jqh8wTTvNyxzqHU0pHLR_RdUKcZ5Jaxacwm6bEFpfj6YI6BMpFh_7BMcxlUJ1_x67wF6U7tITTgbH8Zdjx2MJnDQKm-vFJ0AQ9t5kdhs8-M0MfCKwG_C7OGk2HlBb-mF6t2RNKUL2X7j8xaviH9TSbAewQB1r2sKoQYWml3atA90B81fw5FWBmkaBS95V2r5LwcB5Nny2x4L0xqachAw2fCC4Sq735oa-KjDOrpCxh6f0ZdRA_69jgoBwlseIFTkfRlXIR53HmVadH5Q_v0zfZtO4kAc57dWZ1scPu3rpw1cey_qa20fFqj27sSE8Qe9rMYH2D_UESU_UjKC79LfoJyW_jkMEjHbXQLVrn8p7x-7Ld9Y8gA7YimaDlw5YQh-3RXkLErvMbo"
)

func TestUsers(t *testing.T) {
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

	conn, err := server.Serve(ctx, ":0")
	if err != nil {
		t.Fatalf("Could not start test server: %v", err)
	}
	defer server.Close()

	usrClient := pb.NewUsersClient(conn)
	newUserData := pb.User{
		Name:     "Prothean",
		Email:    "someemail@email.com",
		Password: "SomePassword1",
	}
	cRequest := &pb.CreateUserRequest{
		Name:     newUserData.Name,
		Email:    newUserData.Email,
		Password: newUserData.Password,
	}

	sqlSearchID := `SELECT * FROM "users" WHERE (id = $1) ORDER BY "users"."id" ASC LIMIT 1`
	sqlSearchName := `SELECT * FROM "users" WHERE (name = $1) ORDER BY "users"."id" ASC LIMIT 1`
	sqlSearchEmail := `SELECT * FROM "users" WHERE (email = $1) ORDER BY "users"."id" ASC LIMIT 1`
	sqlCreateUser := `INSERT INTO "users" ("coins","email","gems","id","name","password") VALUES ($1,$2,$3,$4,$5,$6) RETURNING "users"."id"`
	sqlCreateStats := `INSERT INTO "user_stats" ("games","kills","top5","user_id","wins") VALUES ($1,$2,$3,$4,$5) RETURNING "user_stats"."id"`
	sqlDeleteUser := `DELETE FROM "users"  WHERE (id = $1)`

	t.Run("Create User - positive", func(t *testing.T) {

		passwordBytes := []byte(newUserData.Password)
		encryptedPasswordBytes := sha256.Sum256(passwordBytes)
		tmpSlice := encryptedPasswordBytes[:]
		hexPassword := hex.EncodeToString(tmpSlice)

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchName)).WithArgs(newUserData.Name).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchEmail)).WithArgs(newUserData.Email).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(sqlCreateUser)).WithArgs(0, newUserData.Email,
			0, sqlmock.AnyArg(), newUserData.Name, hexPassword).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectQuery(regexp.QuoteMeta(sqlCreateStats)).WithArgs(0, 0, 0, sqlmock.AnyArg(), 0).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectCommit()

		_, err := usrClient.Create(ctx, cRequest)
		if err != nil {
			t.Fatalf("error creating new user: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}
	})

	t.Run("Create User - name exists", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "password", "created_at", "updated_at", "name", "coins", "gems", "is_admin"}).
			AddRow("some-id", "someemail@email.com", "some-hash", "2020-01-01 01:05:57", "2020-01-01 01:05:57", "some-name", 0, 0, 'f')

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchName)).WithArgs(newUserData.Name).WillReturnRows(rows)

		_, err := usrClient.Create(ctx, cRequest)
		if err == nil {
			t.Fatal("expecting error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}
	})

	t.Run("Create User - email exists", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "password", "created_at", "updated_at", "name", "coins", "gems", "is_admin"}).
			AddRow("some-id", "someemail@email.com", "some-hash", "2020-01-01 01:05:57", "2020-01-01 01:05:57", "some-name", 0, 0, 'f')

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchName)).WithArgs(newUserData.Name).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchEmail)).WithArgs(newUserData.Email).WillReturnRows(rows)

		_, err := usrClient.Create(ctx, cRequest)
		if err == nil {
			t.Fatal("expecting error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}
	})

	t.Run("Read User - found by id", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "password", "created_at", "updated_at", "name", "coins", "gems", "is_admin"}).
			AddRow("some-id", "someemail@email.com", "some-hash", "2020-01-01 01:05:57", "2020-01-01 01:05:57", "some-name", 0, 0, 'f')
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchID)).WithArgs("some-id").WillReturnRows(rows)
		_, err := usrClient.Read(ctx, &pb.ReadUserRequest{
			Id: "some-id",
		})
		if err != nil {
			t.Fatalf("error reading user: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}
	})

	t.Run("Read User - found by name", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "password", "created_at", "updated_at", "name", "coins", "gems", "is_admin"}).
			AddRow("some-id", "someemail@email.com", "some-hash", "2020-01-01 01:05:57", "2020-01-01 01:05:57", "some-name", 0, 0, 'f')
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchID)).WithArgs("some-name").WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchName)).WithArgs("some-name").WillReturnRows(rows)
		_, err := usrClient.Read(ctx, &pb.ReadUserRequest{
			Id: "some-name",
		})
		if err != nil {
			t.Fatalf("error reading user: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}
	})

	t.Run("Read User - found by email", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "password", "created_at", "updated_at", "name", "coins", "gems", "is_admin"}).
			AddRow("some-id", "someemail@email.com", "some-hash", "2020-01-01 01:05:57", "2020-01-01 01:05:57", "some-name", 0, 0, 'f')
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchID)).WithArgs("someemail@email.com").WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchName)).WithArgs("someemail@email.com").WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchEmail)).WithArgs("someemail@email.com").WillReturnRows(rows)
		_, err := usrClient.Read(ctx, &pb.ReadUserRequest{
			Id: "someemail@email.com",
		})
		if err != nil {
			t.Fatalf("error reading user: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}
	})

	t.Run("Read User - not found", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchID)).WithArgs("someemail@email.com").WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchName)).WithArgs("someemail@email.com").WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchEmail)).WithArgs("someemail@email.com").WillReturnRows(sqlmock.NewRows(nil))
		_, err := usrClient.Read(ctx, &pb.ReadUserRequest{
			Id: "someemail@email.com",
		})
		if err == nil {
			t.Fatalf("expecting error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}
	})

	t.Run("Delete User - positive", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(sqlDeleteUser)).WithArgs("some-id").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()
		_, err := usrClient.Delete(ctx, &pb.DeleteUserRequest{
			Id: "some-id",
		})
		if err != nil {
			t.Fatalf("error deleting user: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}
	})
}
