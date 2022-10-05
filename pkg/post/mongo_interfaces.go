package post

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ( // Interfaces
	IMongoDB interface {
		Collection(name string) IMongoCollection
	}

	IMongoCollection interface {
		InsertOne(context.Context, interface{}, ...*options.InsertOneOptions) (IMongoInsertOneResult, error)
		UpdateOne(context.Context, interface{}, interface{}, ...*options.UpdateOptions) (IMongoUpdateResult, error)
		DeleteOne(context.Context, interface{}, ...*options.DeleteOptions) (IMongoDeleteResult, error)
		FindOne(context.Context, interface{}, ...*options.FindOneOptions) IMongoSingleResult
		Find(context.Context, interface{}, ...*options.FindOptions) (IMongoCursor, error)
		Database() *mongo.Database
	}

	IMongoCursor interface {
		Close(context.Context) error
		All(context.Context, interface{}) error
	}

	IMongoSingleResult    interface{ Decode(interface{}) error }
	IMongoInsertOneResult interface{}
	IMongoUpdateResult    interface{}
	IMongoDeleteResult    interface{}
)

type ( // Structs
	MongoCursor struct{ cur *mongo.Cursor }

	MongoCollection struct {
		Coll *mongo.Collection
	}

	MongoSingleResult    struct{ res *mongo.SingleResult }
	MongoInsertOneResult struct{ res *mongo.InsertOneResult }
	MongoUpdateResult    struct{ res *mongo.UpdateResult }
	MongoDeleteResult    struct{ res *mongo.DeleteResult }
)

// MongoSingleResult

func (sr *MongoSingleResult) Decode(v interface{}) error {
	return sr.res.Decode(v)
}

// MongoCursor

func (cur *MongoCursor) Close(ctx context.Context) error {
	return cur.cur.Close(ctx)
}
func (cur *MongoCursor) All(ctx context.Context, post interface{}) error {
	return cur.cur.All(ctx, post)
}

// MongoCollection

func (col *MongoCollection) Database() *mongo.Database {
	return col.Coll.Database()
}

func (col *MongoCollection) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (IMongoInsertOneResult, error) {
	insertOneResult, err := col.Coll.InsertOne(ctx, document, opts...)
	if err != nil {
		return nil, err
	}
	return &MongoInsertOneResult{res: insertOneResult}, nil
}

func (col *MongoCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (IMongoUpdateResult, error) {
	updateResult, err := col.Coll.UpdateOne(ctx, filter, update, opts...)
	if err != nil {
		return nil, err
	}
	return &MongoUpdateResult{res: updateResult}, nil
}

func (col *MongoCollection) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (IMongoDeleteResult, error) {
	deleteResult, err := col.Coll.DeleteOne(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	return &MongoDeleteResult{res: deleteResult}, nil
}

func (col *MongoCollection) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) IMongoSingleResult {
	singleResult := col.Coll.FindOne(ctx, filter, opts...)
	return &MongoSingleResult{res: singleResult}
}

func (col *MongoCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (IMongoCursor, error) {
	cursorResult, err := col.Coll.Find(ctx, filter, opts...)
	return &MongoCursor{cur: cursorResult}, err
}
