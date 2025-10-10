package ratelimiter

import (
	"sync"
	"time"
)

type RateLimiter2 struct {
	cap        int
	tokens     int
	refillRate time.Duration
	lock       *sync.Mutex
	cond       *sync.Cond
	tokensChan chan struct{}
}

func (r *RateLimiter2) Accquire() {
	r.lock.Lock()
	for r.tokens == 0 {
		r.cond.Wait()
	}
	r.tokens--
	r.lock.Unlock()
}

func NewRateLimiter(cap int, refillRate time.Duration) *RateLimiter2 {
	lock := &sync.Mutex{}
	rate := &RateLimiter2{
		cap:        cap,
		tokens:     cap,
		refillRate: refillRate,
		lock:       lock,
		cond:       sync.NewCond(lock),
		tokensChan: make(chan struct{}, cap),
	}
	for i := 1; i <= cap; i++ {
		rate.tokensChan <- struct{}{}
	}
	go rate.refill()
	go rate.refill2()
	return rate
}

func (r *RateLimiter2) refill() {
	after := time.NewTimer(r.refillRate)
	for range after.C {
		r.lock.Lock()
		if r.tokens < r.cap {
			r.tokens++
			r.cond.Signal()
		}
		r.lock.Unlock()
	}
}

func (r *RateLimiter2) Accquire2() {
	<-r.tokensChan
}

func (r *RateLimiter2) refill2() {
	after := time.NewTicker(r.refillRate)
	for range after.C {
		existing := len(r.tokensChan)
		for i := 1; i <= r.cap-existing; i++ {
			r.tokensChan <- struct{}{}
		}

	}
}
