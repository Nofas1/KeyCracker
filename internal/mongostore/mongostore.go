package mongostore

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"key_cracker/internal/errs"
)

type Lock struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	InitialState string  `json:"initial_state"`
	Relations    [][]int `json:"relations"`
}

type lockDoc struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	Name         string        `bson:"name"`
	InitialState string        `bson:"initial_state"`
	Relations    [][]int       `bson:"relations"`
}

type LockStore struct {
	client *mongo.Client
	coll *mongo.Collection
}

func NewLockStore(ctx context.Context, uri, dbName string) (*LockStore, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongostore: connect: %w", err)
	}
	return &LockStore{coll: client.Database(dbName).Collection("locks")}, nil
}

func (s *LockStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx, nil)
}

func toLock(doc lockDoc) Lock {
	return Lock{
		ID:           doc.ID.Hex(),
		Name:         doc.Name,
		InitialState: doc.InitialState,
		Relations:    doc.Relations,
	}
}

func (s *LockStore) Create(ctx context.Context, name, initialState string, relations [][]int) (*Lock, error) {
	doc := lockDoc{
		Name:         name,
		InitialState: initialState,
		Relations:    relations,
	}
	res, err := s.coll.InsertOne(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("mongostore: insert lock: %w", err)
	}
	doc.ID = res.InsertedID.(bson.ObjectID)
	lock := toLock(doc)
	return &lock, nil
}

func (s *LockStore) Get(ctx context.Context, id string) (*Lock, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, errs.ErrInvalidID
	}

	var doc lockDoc
	if err := s.coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("mongostore: get lock: %w", err)
	}
	lock := toLock(doc)
	return &lock, nil
}

func (s *LockStore) List(ctx context.Context) ([]Lock, error) {
	cur, err := s.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("mongostore: list locks: %w", err)
	}
	defer cur.Close(ctx)

	var locks []Lock
	for cur.Next(ctx) {
		var doc lockDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, fmt.Errorf("mongostore: decode lock: %w", err)
		}
		locks = append(locks, toLock(doc))
	}
	return locks, cur.Err()
}
