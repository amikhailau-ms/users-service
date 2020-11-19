package svc

import (
	"context"
	"regexp"
	"testing"

	"github.com/amikhailau/users-service/pkg/pb"
	"github.com/infobloxopen/atlas-app-toolkit/query"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	testutils "github.com/amikhailau/users-service/pkg/testing"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

func TestStoreItems(t *testing.T) {
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

	stiServer, err := NewStoreItemsServer(&StoreItemsServerConfig{
		Database: gdb,
	})
	if err != nil {
		t.Fatalf("Could not create store items server: %v", err)
	}
	pb.RegisterStoreItemsServer(server.GRPCServer, stiServer)

	conn, err := server.Serve(ctx, ":0")
	if err != nil {
		t.Fatalf("Could not start test server: %v", err)
	}
	defer server.Close()

	stiClient := pb.NewStoreItemsClient(conn)

	sqlSearchID := `SELECT * FROM "store_items" WHERE ("store_items"."id" = $1) ORDER BY "store_items"."id" ASC LIMIT 1`
	sqlSearchIDEdited := `SELECT * FROM "store_items"  WHERE (id=$1) ORDER BY "store_items"."id" ASC LIMIT 1 FOR UPDATE`
	//sqlSearchIDEdited2 := `SELECT * FROM "store_items" WHERE (id = $1) ORDER BY "store_items"."id" ASC LIMIT 1`
	//sqlSearchIDUser := `SELECT * FROM "users" WHERE (id = $1) ORDER BY "users"."id" ASC LIMIT 1`
	sqlSearchNameType := `SELECT * FROM "store_items" WHERE (name = $1 AND type = $2) ORDER BY "store_items"."id" ASC LIMIT 1`
	sqlSearchImageID := `SELECT * FROM "store_items" WHERE (image_id = $1) ORDER BY "store_items"."id" ASC LIMIT 1`
	sqlList := `SELECT * FROM "store_items" ORDER BY "id"`
	sqlListOrdered := `SELECT * FROM "store_items" ORDER BY store_items.name,"id"`
	sqlUpdateItem := `UPDATE "store_items" SET "coins_price" = $1, "description" = $2, "gems_price" = $3, "image_id" = $4, "name" = $5, "type" = $6  WHERE "store_items"."id" = $7`
	sqlCreateItem := `INSERT INTO "store_items" ("coins_price","description","gems_price","id","image_id","name","type") VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING "store_items"."id"`
	sqlDeleteItem := `DELETE FROM "store_items"  WHERE (id = $1)`
	//sqlUpdateUser := `UPDATE "users" SET "coins" = $1, "email" = $2, "gems" = $3, "name" = $4, "password" = $5  WHERE "users"."id" = $6`
	//sqlBuyItem := `INSERT INTO "users_store_items" ("user_id","store_item_id") SELECT $1,$2  WHERE NOT EXISTS (SELECT * FROM "users_store_items" WHERE "user_id" = $3 AND "store_item_id" = $4)`

	newItemData := &pb.CreateStoreItemRequest{
		Name:        "some-name",
		Description: "some description",
		Type:        1,
		CoinsPrice:  100,
		GemsPrice:   10,
		ImageId:     "some-id",
	}
	updateItemData := &pb.UpdateStoreItemRequest{
		Payload: &pb.StoreItem{
			Name:        "some-name",
			Description: "some description",
			Type:        1,
			CoinsPrice:  100,
			GemsPrice:   10,
			ImageId:     "some-im-id",
			Id:          "some-id",
		},
	}

	t.Run("Create Item - positive", func(t *testing.T) {

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchNameType)).WithArgs(newItemData.Name, newItemData.Type).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchImageID)).WithArgs(newItemData.ImageId).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(sqlCreateItem)).WithArgs(newItemData.CoinsPrice, newItemData.Description,
			newItemData.GemsPrice, sqlmock.AnyArg(), newItemData.ImageId, newItemData.Name, newItemData.Type).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectCommit()

		_, err := stiClient.Create(ctx, newItemData)
		if err != nil {
			t.Fatalf("error creating new item: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}

	})

	t.Run("Create Item - same name and id", func(t *testing.T) {

		rows := sqlmock.NewRows([]string{"coins_price", "description", "gems_price", "id", "image_id", "name", "type", "created_at", "updated_at"}).
			AddRow(100, "desc", 0, "some-id", "some-im-id", "some-name", 1, "2020-01-01 01:05:57", "2020-01-01 01:05:57")

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchNameType)).WithArgs(newItemData.Name, newItemData.Type).WillReturnRows(rows)

		_, err := stiClient.Create(ctx, newItemData)
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}

	})

	t.Run("Create Item - same image id", func(t *testing.T) {

		rows := sqlmock.NewRows([]string{"coins_price", "description", "gems_price", "id", "image_id", "name", "type", "created_at", "updated_at"}).
			AddRow(100, "desc", 0, "some-id", "some-im-id", "some-name", 1, "2020-01-01 01:05:57", "2020-01-01 01:05:57")

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchNameType)).WithArgs(newItemData.Name, newItemData.Type).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchImageID)).WithArgs(newItemData.ImageId).WillReturnRows(rows)

		_, err := stiClient.Create(ctx, newItemData)
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}

	})

	t.Run("Read Item - found", func(t *testing.T) {

		rows := sqlmock.NewRows([]string{"coins_price", "description", "gems_price", "id", "image_id", "name", "type", "created_at", "updated_at"}).
			AddRow(100, "desc", 0, "some-id", "some-im-id", "some-name", 1, "2020-01-01 01:05:57", "2020-01-01 01:05:57")

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchID)).WithArgs("some-id").WillReturnRows(rows)

		_, err := stiClient.Read(ctx, &pb.ReadStoreItemRequest{
			Id: "some-id",
		})
		if err != nil {
			t.Fatalf("error reading item: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}

	})

	t.Run("Read Item - not found", func(t *testing.T) {

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchID)).WithArgs("some-id").WillReturnRows(sqlmock.NewRows(nil))

		_, err := stiClient.Read(ctx, &pb.ReadStoreItemRequest{
			Id: "some-id",
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}

	})

	t.Run("Delete Item", func(t *testing.T) {

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(sqlDeleteItem)).WithArgs("some-id").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		_, err := stiClient.Delete(ctx, &pb.DeleteStoreItemRequest{
			Id: "some-id",
		})
		if err != nil {
			t.Fatalf("error deleting item: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}

	})

	t.Run("Update Item - positive", func(t *testing.T) {

		rows := sqlmock.NewRows([]string{"coins_price", "description", "gems_price", "id", "image_id", "name", "type", "created_at", "updated_at"}).
			AddRow(100, "desc", 0, "some-id", "some-im-id", "some-name", 1, "2020-01-01 01:05:57", "2020-01-01 01:05:57")

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchID)).WithArgs(updateItemData.Payload.Id).WillReturnRows(rows)
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchNameType)).WithArgs(updateItemData.Payload.Name, updateItemData.Payload.Type).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchImageID)).WithArgs(updateItemData.Payload.ImageId).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchIDEdited)).WithArgs(updateItemData.Payload.Id).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(sqlUpdateItem)).WithArgs(updateItemData.Payload.CoinsPrice, updateItemData.Payload.Description,
			updateItemData.Payload.GemsPrice, updateItemData.Payload.ImageId,
			updateItemData.Payload.Name, updateItemData.Payload.Type, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		_, err := stiClient.Update(ctx, updateItemData)
		if err != nil {
			t.Fatalf("error updating new item: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}

	})

	t.Run("Update Item - same name and id", func(t *testing.T) {

		otherRows := sqlmock.NewRows([]string{"coins_price", "description", "gems_price", "id", "image_id", "name", "type", "created_at", "updated_at"}).
			AddRow(100, "desc", 0, "some-id", "some-im-id", "some-name", 1, "2020-01-01 01:05:57", "2020-01-01 01:05:57")

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchID)).WithArgs(updateItemData.Payload.Id).WillReturnRows(otherRows)

		rows := sqlmock.NewRows([]string{"coins_price", "description", "gems_price", "id", "image_id", "name", "type", "created_at", "updated_at"}).
			AddRow(100, "desc", 0, "some-id", "some-im-id", "some-name", 1, "2020-01-01 01:05:57", "2020-01-01 01:05:57")

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchNameType)).WithArgs(updateItemData.Payload.Name, updateItemData.Payload.Type).WillReturnRows(rows)

		_, err := stiClient.Update(ctx, updateItemData)
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}

	})

	t.Run("Update Item - same image id", func(t *testing.T) {
		otherRows := sqlmock.NewRows([]string{"coins_price", "description", "gems_price", "id", "image_id", "name", "type", "created_at", "updated_at"}).
			AddRow(100, "desc", 0, "some-id", "some-im-id", "some-name", 1, "2020-01-01 01:05:57", "2020-01-01 01:05:57")

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchID)).WithArgs(updateItemData.Payload.Id).WillReturnRows(otherRows)

		rows := sqlmock.NewRows([]string{"coins_price", "description", "gems_price", "id", "image_id", "name", "type", "created_at", "updated_at"}).
			AddRow(100, "desc", 0, "some-id", "some-im-id", "some-name", 1, "2020-01-01 01:05:57", "2020-01-01 01:05:57")

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchNameType)).WithArgs(updateItemData.Payload.Name, updateItemData.Payload.Type).WillReturnRows(sqlmock.NewRows(nil))
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchImageID)).WithArgs(updateItemData.Payload.ImageId).WillReturnRows(rows)

		_, err := stiClient.Update(ctx, updateItemData)
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}

	})

	t.Run("List Item", func(t *testing.T) {

		rows := sqlmock.NewRows([]string{"coins_price", "description", "gems_price", "id", "image_id", "name", "type", "created_at", "updated_at"}).
			AddRow(100, "desc", 0, "some-id", "some-im-id", "some-name", 1, "2020-01-01 01:05:57", "2020-01-01 01:05:57")

		mock.ExpectQuery(regexp.QuoteMeta(sqlList)).WillReturnRows(rows)

		_, err := stiClient.List(ctx, &pb.ListStoreItemsRequest{})
		if err != nil {
			t.Fatalf("error listing items: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}

	})

	t.Run("List Item - sorting", func(t *testing.T) {

		rows := sqlmock.NewRows([]string{"coins_price", "description", "gems_price", "id", "image_id", "name", "type", "created_at", "updated_at"}).
			AddRow(100, "desc", 0, "some-id", "some-im-id", "some-name", 1, "2020-01-01 01:05:57", "2020-01-01 01:05:57")

		mock.ExpectQuery(regexp.QuoteMeta(sqlListOrdered)).WillReturnRows(rows)

		_, err := stiClient.List(ctx, &pb.ListStoreItemsRequest{
			OrderBy: &query.Sorting{Criterias: []*query.SortCriteria{&query.SortCriteria{Tag: "name", Order: 0}}},
		})
		if err != nil {
			t.Fatalf("error listing items: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}

	})

	/*t.Run("BuyByUser - positive", func(t *testing.T) {
		uRows := sqlmock.NewRows([]string{"id", "email", "password", "created_at", "updated_at", "name", "coins", "gems", "is_admin"}).
			AddRow("some-id", "someemail@email.com", "some-hash", "2020-01-01 01:05:57", "2020-01-01 01:05:57", "some-name", 1000, 100, 'f')

		iRows := sqlmock.NewRows([]string{"coins_price", "description", "gems_price", "id", "image_id", "name", "type", "created_at", "updated_at"}).
			AddRow(100, "desc", 0, "some-item-id", "some-im-id", "some-name", 1, "2020-01-01 01:05:57", "2020-01-01 01:05:57")

		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchIDUser)).WithArgs("some-id").WillReturnRows(uRows)
		mock.ExpectQuery(regexp.QuoteMeta(sqlSearchIDEdited2)).WithArgs("some-item-id").WillReturnRows(iRows)
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(sqlUpdateUser)).WithArgs(900, "someemail@email.com", 100, "some-name", "some-hash", "some-id").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(regexp.QuoteMeta(sqlBuyItem)).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "some-id", "some-item-id").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		_, err := stiClient.BuyByUser(ctx, &pb.BuyByUserRequest{
			UserId: "some-id",
			ItemId: "some-item-id",
		})
		if err != nil {
			t.Fatalf("error buying item: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("mock shows different data: %v", err)
		}
	})*/

}
