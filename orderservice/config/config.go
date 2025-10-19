package config

import (
	"context"
	"encoding/json"
	"os"

	"github.com/phuthien0308/ordering/common/log"
	"github.com/phuthien0308/ordering/config/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var configServer = "localhost:8080"

var Config = struct {
	Logger log.Logger
	Env    string
	Db     *MongoDBConfig
}{
	Logger: log.NewLogger(log.DEBUG, nil),
	Env:    os.Getenv("env"),
	Db:     &MongoDBConfig{},
}

type MongoDBConfig struct {
	UserName string   `json:"userName"`
	Password string   `json:"password"`
	Hosts    []string `json:"hosts"`
}

func init() {
	register()
	loadConfig("/orderservice/db", Config.Db)
}

func register() {
	// grpcClient, err := grpc.NewClient(configServer, grpc.WithTransportCredentials(insecure.NewCredentials()))
	// if err != nil {
	// 	panic(err)
	// }
	// configClient := pb.NewConfigClient(grpcClient)

}
func loadConfig(path string, v any) {

	grpcClient, err := grpc.NewClient(configServer, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	configClient := pb.NewConfigClient(grpcClient)

	response, err := configClient.Watch(context.Background(), &pb.ConfigRequest{Path: path})
	if err != nil {
		panic(err)
	}
	for {
		data, err := response.Recv()
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal([]byte(data.Data), v)
		if err != nil {
			panic(err)
		}
	}
}
