package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/phuthien0308/ordering/config/pb"
	"github.com/phuthien0308/ordering/orderservice/clients"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

var once sync.Once
var MongoDB *mongo.Database
var Logger *zap.Logger
var Env = os.Getenv("env")

func InitConfig() {
	once.Do(func() {
		MongoDB = getMongoDB(Env, "orders")
		Logger = getLogger(Env)
	})
}

type mongoDBConfig struct {
	UserName string   `json:"userName"`
	Password string   `json:"password"`
	Hosts    []string `json:"hosts"`
}

func getMongoDB(env string, db string) *mongo.Database {
	path := fmt.Sprintf("/%v/orderservice/db", env)
	response, err := clients.ConfigClient.Get(context.Background(), &pb.ConfigRequest{Path: path})
	if err != nil {
		panic(err)
	}
	dbConfig := mongoDBConfig{}
	err = json.Unmarshal([]byte(response.Data), &dbConfig)
	if err != nil {
		fmt.Println("terminated becuase of the incorrect format")
		panic(err)
	}
	connect, err := mongo.Connect(options.Client().SetHosts(dbConfig.Hosts),
		options.Client().SetAuth(options.Credential{Username: dbConfig.UserName, Password: dbConfig.Password}))
	if err != nil {
		panic(err)
	}
	return connect.Database(db)
}

func getLogger(env string) *zap.Logger {
	if env == "production" {
		logger, err := zap.NewProduction()
		if err != nil {
			panic(err)
		}
		return logger
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return logger
}
