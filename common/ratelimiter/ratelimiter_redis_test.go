package ratelimiter

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRateLimiterRedis(t *testing.T) {
	r := NewRateLimiterRedis("hello", 15*time.Second, 1)
	result, _ := r.Accquire(context.Background())
	fmt.Println(result)
	result, _ = r.Accquire(context.Background())
	fmt.Println(result)
	time.Sleep(6 * time.Second)
}
