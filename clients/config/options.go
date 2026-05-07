package config

import (
	"time"

	"github.com/phuthien0308/ordering-base/ratelimiter"
)

// ClientOptions holds generic configuration that can be shared across HTTP and gRPC clients.
type ClientOptions struct {
	Timeout             time.Duration
	MaxRetries          int
	RetryDelay          time.Duration
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	RateLimiter         ratelimiter.RateLimiter
}

// ClientOption is a functional option pattern interface.
type ClientOption func(*ClientOptions)

// WithTimeout configures a strict overarching connection timeout for the client.
func WithTimeout(t time.Duration) ClientOption {
	return func(o *ClientOptions) {
		o.Timeout = t
	}
}

// WithRetries enables resilient retry loops on server or network errors.
func WithRetries(maxRetries int, delay time.Duration) ClientOption {
	return func(o *ClientOptions) {
		o.MaxRetries = maxRetries
		o.RetryDelay = delay
	}
}

// WithConnectionPool specifically tunes the underlying HTTP transport to eliminate bottlenecks.
func WithConnectionPool(maxIdleConns, maxIdleConnsPerHost int) ClientOption {
	return func(o *ClientOptions) {
		o.MaxIdleConns = maxIdleConns
		o.MaxIdleConnsPerHost = maxIdleConnsPerHost
	}
}

// EndpointRateLimits holds the strictly-typed capacity configurations for each SDK endpoint.
type EndpointRateLimits struct {
	GetConfigRate        float64
	GetConfigCapacity    float64
	ListVersionsRate     float64
	ListVersionsCapacity float64
}

// WithRateLimiter injects an optional rate limiter (e.g. your RedisTokenBucket)
// and strongly-types the configuration so consumers don't have to guess the keys.
func WithRateLimiter(rl ratelimiter.RateLimiter, limits EndpointRateLimits) ClientOption {
	return func(o *ClientOptions) {
		if rl != nil {
			// The SDK owns the formatting knowledge, so it registers the rules automatically!
			if limits.GetConfigRate > 0 {
				rl.AddRule("api_v1_configs_render", limits.GetConfigRate, limits.GetConfigCapacity)
			}
			rl.AddRule("api_v1_configs_versions", limits.ListVersionsRate, limits.ListVersionsCapacity)
		}
		o.RateLimiter = rl
	}
}

// DefaultClientOptions provides completely safe factory defaults for client connections.
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		Timeout:             10 * time.Second,
		MaxRetries:          3,
		RetryDelay:          500 * time.Millisecond,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
	}
}
