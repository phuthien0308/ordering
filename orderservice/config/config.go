package config

import (
	"context"
	"encoding/json"
	"os"

	"github.com/phuthien0308/ordering/config/pb"
	"github.com/phuthien0308/ordering/orderservice/clients"
	"github.com/phuthien0308/ordering/orderservice/helper"
)

var Config = struct {
	Env string
	Db  *MongoDBConfig
}{

	Env: os.Getenv("env"),
	Db:  &MongoDBConfig{},
}

type MongoDBConfig struct {
	UserName string   `json:"userName"`
	Password string   `json:"password"`
	Hosts    []string `json:"hosts"`
}

func init() {
	//register()
	//go func() { loadConfig("/orderservice/db", Config.Db) }()
}

func register() {
	_, err := clients.ConfigClient.Register(context.Background(), &pb.RegisterRequest{
		Appname: helper.AppName,
		Ip:      helper.HealthCheckEndpoint(8081),
	})
	if err != nil {
		panic(err)
	}
}

func loadConfig(path string, v any) {

	response, err := clients.ConfigClient.Watch(context.Background(), &pb.ConfigRequest{Path: path})
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
