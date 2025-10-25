package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"net/http"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/phuthien0308/ordering/config/internal"
	"go.uber.org/zap"
)

type Worker interface {
	// start the background health check for all services.
	Start(ctx context.Context)
}

type serviceAddress struct {
	appName             string
	address             string
	appNode             string
	healthCheckEndpoint string
	err                 error
}

func (svc *serviceAddress) String() string {
	return fmt.Sprintf("{'name':%v,'adddress':%v,'appNode':%v,'healhCheckEndpoint':%v,'err':%v}", svc.appName, svc.address, svc.appNode, svc.healthCheckEndpoint, svc.err)
}

type workerImpl struct {
	logger   *zap.Logger
	conn     *zk.Conn
	interval time.Duration
	retry    time.Duration
	logic    *internal.Config
}

func NewHealhCheckWorker(l *zap.Logger, con *zk.Conn, interval time.Duration, retry time.Duration) Worker {
	return &workerImpl{
		conn:     con,
		logger:   l,
		interval: interval,
		retry:    retry,
		logic:    internal.NewConfig(con, l),
	}
}

func (w *workerImpl) Start(ctx context.Context) {

	defer func() {
		if err := recover(); err != nil {
			w.logger.Error("panic", zap.Error(fmt.Errorf("error: %v", err)))
		}
	}()
	w.logger.Info("starting worker health check", zap.String("interval", w.interval.String()), zap.String("retryRate", w.retry.String()))

	ticker := time.NewTicker(w.interval)
	for range ticker.C {
		apps, _, err := w.conn.Children("/services")
		if err != nil {
			w.logger.Error("can not pull the children", zap.Error(err))
			// should fire the alert because the application does not get the children node
			continue
		}
		for _, app := range apps {
			go func() {
				w.logger.Info("pulled all ips", zap.String("app", app))

				serviceChan, err := w.pullAddress(app)
				if err != nil {
					w.logger.Error("error while pulling ip for app", zap.Error(err), zap.String("app", app))
				}

				for svc := range serviceChan {
					if svc.err != nil {
						// should fire alert
						continue
					}
					go w.healthCheck(ctx, svc)
				}
			}()
		}
	}
}

func (w *workerImpl) healthCheck(ctx context.Context, svc *serviceAddress) {
	defer w.logger.Info("finished the health check for service", zap.Stringer("service", svc))
	w.logger.Info("starting the health check for service", zap.Stringer("service", svc))
	// retry with backoff time.
	// we only support http healthz check
	res, err := http.Get(svc.healthCheckEndpoint)
	if err != nil || res.StatusCode != http.StatusOK {
		w.logger.Error("remove the unhealthy address", zap.Error(err), zap.String("address", svc.healthCheckEndpoint))
		retry(func() error {
			return w.logic.Deregister(ctx, svc.appName, svc.appNode)
		}, w.retry)
	}
}

func (w *workerImpl) pullAddress(appName string) (<-chan *serviceAddress, error) {
	path := fmt.Sprintf("/services/%v", appName)
	allChildren, _, err := w.conn.Children(path)
	if err != nil {
		w.logger.Error("can not get children", zap.Error(err))
		return nil, err
	}
	serviceChan := make(chan *serviceAddress)
	wg := &sync.WaitGroup{}
	wg.Add(len(allChildren))
	for _, node := range allChildren {
		go func() {
			nodePath := fmt.Sprintf("%v/%v", path, node)
			data, _, err := w.conn.Get(nodePath)
			if err != nil {
				serviceChan <- &serviceAddress{appName: appName, err: err}
			}
			reg := internal.Registration{}
			err = json.Unmarshal(data, &reg)
			if err != nil {
				w.logger.Error("can not marshal", zap.Error(err))
				serviceChan <- &serviceAddress{appName: appName, err: err}
			}
			serviceChan <- &serviceAddress{appName: appName,
				appNode: node, address: reg.IpAddress, healthCheckEndpoint: reg.HealthCheckEndpoint}
		}()
	}

	go func() {
		wg.Wait()
		close(serviceChan)
	}()

	return serviceChan, nil

}

func retry(f func() error, rate time.Duration) {

	ticker := time.NewTicker(rate)

	randomMillis := rand.Intn(100)
	if err := f(); err != nil {
		for err != nil {
			fmt.Println("retry")
			<-ticker.C
			err = f()
			ticker = time.NewTicker(rate + time.Duration(randomMillis)*time.Millisecond)
		}
	}
}
