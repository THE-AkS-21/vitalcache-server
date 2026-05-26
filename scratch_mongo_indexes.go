package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	_ = godotenv.Load(".env")
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb+srv://root:rHirELm5uGLUonAD@vitalcache.d2fof.mongodb.net/?retryWrites=true&w=majority&appName=VitalCache"
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("vitalcache")

	// 1. Medicines Indexes
	medicinesColl := db.Collection("medicines")
	_, err = medicinesColl.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "name", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "category", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "manufacturer", Value: 1}},
		},
	})
	if err != nil {
		log.Printf("Failed to create medicines indexes: %v", err)
	}

	// 2. Prescriptions Indexes
	prescriptionsColl := db.Collection("prescriptions")
	_, err = prescriptionsColl.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "patient_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "doctor_id", Value: 1}},
		},
	})
	if err != nil {
		log.Printf("Failed to create prescriptions indexes: %v", err)
	}

	fmt.Println("Successfully created MongoDB indexes!")
}
