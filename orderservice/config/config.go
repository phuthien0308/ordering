package config

import (
	"os"

	"github.com/phuthien0308/ordering/common/log"
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
	Config.Db = loadMongoDBConfig()
}

func loadMongoDBConfig() *MongoDBConfig {

	return &MongoDBConfig{}
}
