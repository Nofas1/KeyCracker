// main.go
package main

import (
	// "context"
	// "key_cracker/internal"
	"context"
	"fmt"
	"key_cracker/internal/handler"
	"log/slog"
	"net/http"
	"os"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cl, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		logger.Error(
			"mongo did not connected",
			"error", err,
		)
		os.Exit(1)
	}
	defer cl.Disconnect(context.Background())

	var bson_resp bson.D
	if err = cl.Database("admin").RunCommand(context.TODO(), bson.D{{"ping", 1}}).Decode(&bson_resp); err != nil {
		logger.Error(
			"failed to decode",
			"error", err,
		)
		os.Exit(1)
	}

	mongoDB := cl.Database("key_cracker")
	keys := mongoDB.Collection("keys")

	type Key struct {
		ID           bson.ObjectID `bson:"_id,omitempty"`
		InitialState string        `bson:"initial_state"`
		Relations    [][3]int      `bson:"relations"`
	}

	// res, err := keys.InsertOne(context.TODO(), Key{InitialState: "26243", Relations: [][3]int{{1, 2, -1}}})
	// if err != nil {
	// 	logger.Error("")
	// 	os.Exit(1)
	// }
	// fmt.Println(res.InsertedID)



	var key Key
	objID, err := bson.ObjectIDFromHex("6a99b24907673b220b45b584")
	if err != nil {
		panic("aaa")
	}
	filter := bson.M{"_id": objID}
	if err = keys.FindOne(context.TODO(), filter).Decode(&key); err != nil {
		logger.Error("")
		os.Exit(1)
	}
	fmt.Println(key)

	mux := http.NewServeMux()
	handler.RegisterHandlers(mux, logger)
	addr := ":8089"
	logger.Info("starting server", slog.String("addr", addr))
	if err = http.ListenAndServe(addr, mux); err != nil {
		logger.Error("server failed", slog.Any("error", err))
		os.Exit(1)
	}
}

// mongo add
// subscriptions pgx
// proxi for auth

