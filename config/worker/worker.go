package worker

import (
	"context"
	"fmt"
	"net/http"

	"time"

	retry "github.com/avast/retry-go/v5"
	"github.com/phuthien0308/ordering/config/storage"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

/*
The worker helps check the healthy of all registered services. Typically, the service deregisters itself when it stops.
However, there are some situations that the service can't deregister it, so this worker will check and remove the service.
The hardest thing is to confirm whether the service is dead or not because the network latency can affect the final decision.
-  If a service was removed from the registry but it was still alive, what would we do?
-- it calls the config service constantly to confirm its status.
*/

type Worker interface {
	// start the background health check for all services.
	Start(ctx context.Context)
}

type workerImpl struct {
	logger      *zap.Logger
	interval    time.Duration
	nodeStorage storage.AddressStorage
}

func NewHealhCheckWorker(l *zap.Logger, rd *redis.Client, interval time.Duration) Worker {
	return &workerImpl{
		logger:      l,
		interval:    interval,
		nodeStorage: storage.NewAddressStorage(rd),
	}
}

func (w *workerImpl) Start(ctx context.Context) {
	w.logger.Info("start the health check worker")

	ticker := time.NewTicker(w.interval)

	for range ticker.C {
		w.logger.Info("a new health check round is started")
		allNodes, err := w.nodeStorage.GetAddressOfAllServices(ctx)
		if err == nil {
			for _, node := range allNodes {
				go func() {
					for _, ip := range node.Ips {
						go func() {
							err := w.healthCheck(ip)
							if err != nil {
								w.nodeStorage.Remove(ctx, node.AppName, ip)
							}
						}()
					}
				}()
			}
		} else {
			w.logger.Error("the health check worker can not connect to redis", zap.Error(err))
		}
	}
}

func (w *workerImpl) healthCheck(ip string) error {
	w.logger.Info(fmt.Sprintf("started health check for ip: %v", ip))

	repeater := retry.New(
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxJitter(100*time.Millisecond))

	err := repeater.Do(func() error {
		healthcheck := fmt.Sprintf("http://%v/healthz", ip)
		_, err := http.Get(healthcheck)
		if err != nil {
			w.logger.Warn("healthcheck result is not healthy", zap.Any("error", err.Error()))
		} else {
			w.logger.Info("healthcheck result is healthy")
		}
		return err
	})
	w.logger.Info(fmt.Sprintf("finised health check for ip: %v with result %v", ip, err), zap.String("ip", ip))
	return err
}
