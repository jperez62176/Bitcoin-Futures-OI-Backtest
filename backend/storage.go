package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Storage interface {
	GetBTCFuturesOIData() (*BitcoinFuturesOI, error)
}

type MongoStore struct {
	collection *mongo.Collection
}

func GetStorage() (*MongoStore, error){
	clientOptions:= options.Client().ApplyURI(os.Getenv("MONGO_CONNECTION_STRING"))
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal("Error while connecting to database")
	}

	// defer client.Disconnect(context.TODO())
	collection := client.Database("TimeSeriesData").Collection("DataObjects")
	return &MongoStore{
		collection: collection,
	}, nil
}

func (m MongoStore) GetBTCFuturesOIData(url string) (*BitcoinFuturesOI, error){
	filter := bson.M{"_id": url}
	var result BitcoinFuturesOI
	err := m.collection.FindOne(context.TODO(),filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No document found with that filter")
		} else {
			log.Fatal(err)
		}
		return nil, err
	}
	return &result, nil
}


