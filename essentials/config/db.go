package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectDB() *mongo.Client {
	// Check for environment variable first (for Docker), fallback to hardcoded
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		// Hardcoded for development - in production, use MONGO_URI env var
		mongoURI = "mongodb+srv://foxyninenineee:akame112@clusterfinanceapi.ccf3ywa.mongodb.net/?appName=ClusterFinanceApi"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Error connecting to MongoDB:", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Error pinging MongoDB:", err)
	}

	fmt.Println("✅ Connected to MongoDB!")

	// Initialize Indexes for performance
	InitIndexes(client)

	return client
}

// InitIndexes creates necessary indexes for robust database performance
func InitIndexes(client *mongo.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := GetDB(client)

	// Transaction index: speed up GetReport (find by user_id and sort/filter by date)
	trxCol := db.Collection("transactions")
	_, err := trxCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "date", Value: -1},
		},
	})
	if err != nil {
		fmt.Printf("⚠️ Failed to index transactions: %v\n", err)
	}

	// Budget index: find budget by user_id and month
	budgetsCol := db.Collection("budgets")
	_, err = budgetsCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "month", Value: -1},
		},
	})
	if err != nil {
		fmt.Printf("⚠️ Failed to index budgets: %v\n", err)
	}

	// Bills index: filter by active bills and due date
	billsCol := db.Collection("bill_reminders")
	_, err = billsCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "is_paid", Value: 1},
			{Key: "next_due_date", Value: 1},
		},
	})
	if err != nil {
		fmt.Printf("⚠️ Failed to index bill_reminders: %v\n", err)
	}

	fmt.Println("✅ MongoDB Indexes Initialized")
}

var DB *mongo.Client = ConnectDB()

func GetCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	return client.Database("mydb").Collection(collectionName)
}

func GetDB(client *mongo.Client) *mongo.Database {
	return client.Database("mydb")
}
