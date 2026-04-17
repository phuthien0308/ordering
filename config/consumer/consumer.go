package consumer

import (
	"context"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/phuthien0308/ordering-contracts/gen/config"
	"github.com/phuthien0308/ordering/config/configuration"
	"github.com/phuthien0308/ordering/config/storage"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type Consumer interface {
	Start(ctx context.Context) (chan interface{}, error)
}

type consumer struct {
	logger         *zap.Logger
	addressStorage storage.AddressStorage
}

func NewConsumer(logger *zap.Logger, addressStorage storage.AddressStorage) Consumer {
	return &consumer{
		logger:         logger,
		addressStorage: addressStorage,
	}
}

// Start implements Consumer.
func (c *consumer) Start(ctx context.Context) (chan interface{}, error) {
	stopChan := make(chan interface{})
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  configuration.Config.Kafka.Host,
		"group.id":           "service-registry-consumer",
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": "false",
	})

	if err != nil {
		c.logger.Error("can not initialize the kafka consumer")
		panic(err)
	}

	err = consumer.Subscribe("service_registry", nil)

	if err != nil {
		c.logger.Error("can not subcribe service registry topic")
		panic(err)
	}

	run := true
	go func() {
		for run {
			ev := consumer.Poll(1000)
			c.logger.Info("polling")
			switch e := ev.(type) {
			case *kafka.Message:
				var anyMsg anypb.Any
				err = proto.Unmarshal(e.Value, &anyMsg)
				if err != nil {
					c.logger.Error("unable to unmarshal message from kafka", zap.Error(err))
					continue
				}
				m, err := anypb.UnmarshalNew(&anyMsg, proto.UnmarshalOptions{})
				if err != nil {
					c.logger.Error("unable to unmarshal message from kafka", zap.Error(err))
					continue
				}
				switch m := m.(type) {
				case *config.RegisterRequest:
					c.logger.Info("received a register request", zap.String("appname", m.Appname),
						zap.String("ips", m.Ip))
					err = c.addressStorage.Add(context.Background(), m.Appname, m.Ip)
					if err != nil {
						c.logger.Error("can't register the application", zap.String("appname", m.Appname), zap.String("ip", m.Ip))
					} else {
						consumer.CommitMessage(e)
					}
				case *config.DeregisterRequest:
					c.logger.Info("received a deregister request", zap.String("appname", m.Appname),
						zap.String("ips", m.Ip))
					err = c.addressStorage.Remove(context.Background(), m.Appname, m.Ip)
					if err != nil {
						c.logger.Error("can't deregister the application", zap.String("appname", m.Appname), zap.String("ip", m.Ip))
					} else {
						consumer.CommitMessage(e)
					}
				default:
					c.logger.Warn("the message type is unrecognizable")
				}
				//_, err = consumer.CommitMessage(e)
			case kafka.AssignedPartitions:
				c.logger.Info("assigned partitions")
				consumer.Assign(e.Partitions)
			case kafka.RevokedPartitions:
				c.logger.Info("revoked partitions")
				consumer.Unassign()
			case kafka.PartitionEOF:
				c.logger.Info(fmt.Sprintf("%% Reached %v\n", e))

			case kafka.Error:
				c.logger.Error("kafka error", zap.Error(err))
			default:
				fmt.Printf("Ignored %v\n", e)
			}

		}

	}()

	go func() {
		<-stopChan
		consumer.Close()
	}()

	return stopChan, nil
}
