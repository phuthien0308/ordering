/*
Handling error rates
*/
package circuitbreaker

import (
	"sync"
	"time"
)

type state string

const (
	OPEN   state = "OPEN"
	CLOSED state = "CLOSED"
	HALF   state = "HALF"
)

var resources map[string]*CiruitBreaker
var lock sync.Mutex

func init() {
	resources = make(map[string]*CiruitBreaker)
}

type CiruitBreaker struct {
	timeWindow          time.Duration
	sleepTime           time.Duration
	requestVolumne      int // the circuit only works if the requestVolumne is greater than the request volumne.
	st                  state
	errorRatePercentage float32
	lock                sync.Mutex
	errorResults        []int64
	totalCalls          int64
}
type CircuitBreakerOption func(*CiruitBreaker)

func NewCircuitBreaker(opts ...CircuitBreakerOption) *CiruitBreaker {
	// default setting
	cir := &CiruitBreaker{
		timeWindow:          5 * time.Second,
		sleepTime:           5 * time.Second,
		requestVolumne:      100,
		errorRatePercentage: 50,
	}
	for _, opt := range opts {
		opt(cir)
	}
	return cir
}

func (c *CiruitBreaker) Register(resource string, config *CiruitBreaker) bool {
	defer lock.Unlock()
	lock.Lock()
	if _, ok := resources[resource]; ok {
		return false
	}
	resources[resource] = config
	return true
}

func (c *CiruitBreaker) Do(resource string, action func() error) {
	if cir, ok := resources[resource]; ok && !cir.isOpen() {
		result := action()
		c.lock.Lock()
		c.totalCalls++
		if result != nil {
			c.errorResults = append(c.errorResults, time.Now().UnixMilli())
		}
		c.lock.Unlock()
	}
}

func (c *CiruitBreaker) isOpen() bool {
	if c.st == OPEN {
		return false
	}
	var reduceResult []int64
	for _, v := range c.errorResults {
		if time.Now().UnixMilli()-c.timeWindow.Milliseconds() <= v {
			reduceResult = append(reduceResult, v)
		}
	}

	if float32(len(reduceResult))/float32(int(c.totalCalls))*100 > c.errorRatePercentage {
		c.st = OPEN
		return true
	}
	return false
}
