/*
 */
package ratelimiter

import (
	"sync"
	"time"
)

type TokenBucket struct {
	capacity int
	tokens   chan struct{}
	quit     chan struct{}
	once     sync.Once
}

func NewTokenBucket(cap int, refillInterval time.Duration) *TokenBucket {
	bucket := &TokenBucket{
		capacity: cap,
		tokens:   make(chan struct{}, cap),
		quit:     make(chan struct{}),
	}

	for i := 1; i <= cap; i++ {
		bucket.tokens <- struct{}{}
	}
	go func() {
		t := time.NewTicker(refillInterval)
		defer bucket.Stop()
		select {
		case <-t.C:
			bucket.tokens <- struct{}{}
		case <-bucket.quit:
			return
		}
	}()
	return bucket
}

func (b *TokenBucket) Allow() bool {
	select {
	case <-b.tokens:
		return true
	default:
		return false
	}
}

func (b *TokenBucket) Stop() {
	b.once.Do(func() {
		close(b.tokens)
	})
}
