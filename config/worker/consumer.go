package worker

import (
	"context"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/phuthien0308/ordering/config/pb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type Consumer interface {
	Start(ctx context.Context) error
}
type consumer struct {
	logger *zap.Logger
}

func NewConsumer() Consumer {
	return &consumer{}
}

// Start implements Consumer.
func (c *consumer) Start(ctx context.Context) (chan interface{}, error) {
	stopChan := make(chan interface{})
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"security.protocol": "SASL_SSL",
		"sasl.mechanisms":   "PLAIN",
		"group.id":          "service-registry-consumer",
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		c.logger.Error("can not initialize the kafka consumer")
		return nil, err
	}

	run := true
	go func() {
		for run {
			select {
			case <-stopChan:
				run = false
			default:
				msg, err := consumer.ReadMessage(10 * time.Second)
				if err != nil {
					c.logger.Error("unable to read the message from kafka", zap.Error(err))
					continue
				}
				var anyMsg *anypb.Any
				err = proto.Unmarshal(msg.Value, anyMsg)
				if err != nil {
					c.logger.Error("unable to unmarshal message from kafka", zap.Error(err))
					continue
				}
				m, err := anypb.UnmarshalNew(anyMsg, proto.UnmarshalOptions{})
				switch m.(type) {
				case *pb.RegisterRequest:
					c.logger.Info("received a register request")
				case *pb.DeregisterRequest:
					c.logger.Info("received a deregister request")
				}
			}
		}
	}()
	return stopChan, nil
}
