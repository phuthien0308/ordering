package configuration

import (
	"encoding/json"
	"os"
)

var Config Configuration

func init() {
	Config = loadConfiguration("./config.json")
}

type KafkaHost struct {
	Host          string `json:"host"`
	Topic         string `json:"topic"`
	ConsumerGroup string `json:"consumer_group"`
	OffsetType    string `json:"offset_type"`
}
type RedisHost struct {
	Host     string `json:"host"`
	UserName string `json:"user_name"`
	Password string `json:"password"`
}
type Worker struct {
	IntervalInMs int64 `json:"interval_in_ms"`
}
type Configuration struct {
	Env               string    `json:"environment"`
	RedisHost         string    `json:"redis_host"`
	Kafka             KafkaHost `json:"kafka"`
	HealthCheckWorker Worker    `json:"worker"`
}

func loadConfiguration(fileName string) Configuration {
	data, err := os.ReadFile(fileName)
	if err != nil {
		panic("can not read the provided file")
	}
	var configuration Configuration
	err = json.Unmarshal(data, &configuration)
	if err != nil {
		panic("loading configuration failed miserably")
	}
	return configuration
}
