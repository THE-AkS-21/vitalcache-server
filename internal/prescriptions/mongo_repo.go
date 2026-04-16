package prescriptions

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoRepo struct {
	coll *mongo.Collection
}

func NewRepository(db *mongo.Database) Repository {
	return &mongoRepo{
		coll: db.Collection("prescriptions"),
	}
}

func (r *mongoRepo) Create(ctx context.Context, p *Prescription) error {
	p.ID = primitive.NewObjectID()
	p.CreatedAt = primitive.NewDateTimeFromTime(p.CreatedAt).Time()
	p.UpdatedAt = p.CreatedAt

	_, err := r.coll.InsertOne(ctx, p)
	return err
}

func (r *mongoRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*Prescription, error) {
	var p Prescription
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *mongoRepo) ListByPatient(ctx context.Context, patientID string, limit, offset int) ([]Prescription, int64, error) {
	filter := bson.M{"patient_id": patientID}

	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "created_at", Value: -1}})
	findOptions.SetSkip(int64(offset))
	findOptions.SetLimit(int64(limit))

	cursor, err := r.coll.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var list []Prescription
	if err := cursor.All(ctx, &list); err != nil {
		return nil, 0, err
	}

	if list == nil {
		list = []Prescription{}
	}

	return list, total, nil
}
