package medicines

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoRepo struct {
	coll *mongo.Collection
}

func NewRepository(db *mongo.Database) Repository {
	return &mongoRepo{
		coll: db.Collection("medicine_catalog"),
	}
}

func (r *mongoRepo) Search(ctx context.Context, query string, limit, offset int) ([]Medicine, int64, error) {
	var filter bson.M
	if query == "" {
		filter = bson.M{}
	} else {
		// Uses MongoDB $text index for fast search
		filter = bson.M{"$text": bson.M{"$search": query}}
	}

	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	findOptions := options.Find()
	findOptions.SetSkip(int64(offset))
	findOptions.SetLimit(int64(limit))

	if query != "" {
		// Sort by text search relevance score
		findOptions.SetSort(bson.D{{Key: "score", Value: bson.M{"$meta": "textScore"}}})
	} else {
		findOptions.SetSort(bson.D{{Key: "name", Value: 1}})
	}

	cursor, err := r.coll.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var list []Medicine
	if err := cursor.All(ctx, &list); err != nil {
		return nil, 0, err
	}

	if list == nil {
		list = []Medicine{}
	}

	return list, total, nil
}
