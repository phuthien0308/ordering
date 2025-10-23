package worker

import (
	"context"
	"encoding/json"
	"fmt"
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
	name    string
	address []string
	err     error
}

func (svc *serviceAddress) String() string {
	return fmt.Sprintf("{'name':%v,'adddress':%v,'err':%v}", svc.name, svc.address, svc.err)
}

type workerImpl struct {
	logger   *zap.Logger
	conn     *zk.Conn
	interval time.Duration
	logic    *internal.Config
}

func NewHealhCheckWorker(l *zap.Logger, con *zk.Conn, interval time.Duration) Worker {
	return &workerImpl{
		conn:     con,
		logger:   l,
		interval: interval,
		logic:    internal.NewConfig(con, l),
	}
}

func (w *workerImpl) Start(ctx context.Context) {

	defer func() {
		if err := recover(); err != nil {
			w.logger.Error("panic", zap.Error(fmt.Errorf("error: %v", err)))
		}
	}()
	ticker := time.NewTicker(w.interval)
	for range ticker.C {
		children, _, err := w.conn.Children("/services")
		w.logger.Info("pulled children", zap.Strings("children", children))

		if err != nil {
			// should fire the alert because the application does not get the children node
			continue
		}

		serviceChan := w.pullAddress(children)

		for svc := range serviceChan {
			if svc.err != nil {
				// should fire alert
				w.logger.Error("error while pulling address", zap.Error(svc.err))
				continue
			}
			go w.healthCheck(ctx, svc)

		}
	}
}

func (w *workerImpl) healthCheck(ctx context.Context, svc *serviceAddress) error {
	defer w.logger.Info("finished the health check for service", zap.Stringer("service", svc))
	w.logger.Info("starting the health check for service", zap.Stringer("service", svc))
	for _, adr := range svc.address {
		// retry with backoff time.
		// we only support http healthz check
		res, err := http.Get(adr)
		if err != nil || res.StatusCode != http.StatusOK {
			w.logger.Info("remove the unhealthy address", zap.String("address", adr))
			w.logic.Deregister(ctx, svc.name, adr)
		}
	}
	return nil
}

func (w *workerImpl) pullAddress(children []string) <-chan *serviceAddress {
	serviceChan := make(chan *serviceAddress)
	wg := &sync.WaitGroup{}
	wg.Add(len(children))
	for _, v := range children {
		go func() {
			defer wg.Done()
			data, _, err := w.conn.Get(fmt.Sprintf("/services/%v", v))
			if err != nil {
				serviceChan <- &serviceAddress{
					name: v,
					err:  err,
				}
				return
			}
			var address []string
			err = json.Unmarshal(data, &address)
			if err != nil {
				serviceChan <- &serviceAddress{
					name: v,
					err:  err,
				}
			}
			serviceChan <- &serviceAddress{
				name:    v,
				address: address,
			}
		}()
	}

	go func() {
		wg.Wait()
		close(serviceChan)
	}()
	return serviceChan
}
