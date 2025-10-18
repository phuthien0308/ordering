package config

import (
	"bytes"
	"context"
	"encoding/json"
	"os"

	"github.com/phuthien0308/ordering/common/log"
	"github.com/phuthien0308/ordering/config/pb"
	"google.golang.org/grpc"
)

var Config = struct {
	Logger log.Logger
	Env    string
	Db     *MongoDBConfig
}{
	Logger: log.NewLogger(log.DEBUG, nil),
	Env:    os.Getenv("env"),
}

type MongoDBConfig struct {
	UserName string   `json:"userName"`
	Password string   `json:"password"`
	Hosts    []string `json:"hosts"`
}

func Init() {
	db, err := loadMongoDBConfig()
	if err != nil {
		panic(err)
	}
	Config.Db = db
}

func loadMongoDBConfig() (*MongoDBConfig, error) {
	grpcClient, err := grpc.NewClient("localhost:8080")
	if err != nil {
		return nil, err
	}
	configClient := pb.NewConfigClient(grpcClient)
	dbConfig, err := configClient.Get(context.Background(), &pb.ConfigRequest{Path: "/orderservice/db"})
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewBufferString(dbConfig.Data))
	var result *MongoDBConfig
	err = decoder.Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
