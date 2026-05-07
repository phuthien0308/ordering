# Config Service Client

A robust, resilient Go client for fetching configuration from the centralized `configsvc`. 

Features:
- Built-in Exponential Backoff with Full Jitter for retries.
- OpenTelemetry instrumentation.
- Seamless context-aware local Rate Limiting (acting natively as a 429 server fail-fast).

## Rate Limiting Configuration

The Config Client abstracts away HTTP URIs and uses well-defined **Operation Keys** for rate limiting. This ensures that you don't need to write complex Regular Expressions to match configuration paths.

### Available Operation Keys

Consumers should use the following constants provided by the `config` package:
- `config.OpGetConfig` ("configsvc.GetConfig"): Rate limits fetching a configuration document.
- `config.OpListVersions` ("configsvc.ListVersions"): Rate limits listing semantic versions.

### 1. JSON Configuration Format

Since the client decodes configurations into a Go struct, you can easily define rate limits in your central JSON configuration file. Use the exported `KeyCapacity` structure natively.

```json
{
  "database": {
    "endpoint": "http://localhost:4566",
    "table_name": "products"
  },
  "config_service_client": {
    "timeout": "5s",
    "max_retries": 3,
    "rate_limits": [
      {
        "key": "configsvc.GetConfig",
        "rate": 10.0,
        "capacity": 50.0
      },
      {
        "key": "configsvc.ListVersions",
        "rate": 2.0,
        "capacity": 5.0
      }
    ]
  }
}
```

### 2. Corresponding Go Struct

When starting up a service (e.g., `productservice`), decode this JSON by directly embedding `ratelimiter.KeyCapacity` inside your struct:

```go
package main

import (
	"context"
	
	"github.com/phuthien0308/ordering-base/clients/config"
	"github.com/phuthien0308/ordering-base/clients/options"
	"github.com/phuthien0308/ordering-base/ratelimiter"
)

// 1. Define the Struct matching the JSON file
type ServiceConfig struct {
	Database struct {
		Endpoint  string `json:"endpoint"`
		TableName string `json:"table_name"`
	} `json:"database"`

	ConfigServiceClient struct {
		Timeout    string                    `json:"timeout"`
		MaxRetries int                       `json:"max_retries"`
		RateLimits []ratelimiter.KeyCapacity `json:"rate_limits"` // <--- Plugs in natively
	} `json:"config_service_client"`
}

func main() {
	// Setup initial transient client to fetch the config
	initialClient, _ := config.NewClient("http://configsvc:8080")
	var cfg ServiceConfig
	
	// Fetch the JSON payload
	_ = initialClient.GetConfig(context.Background(), "productservice", "v1.0", "prod", &cfg)

	// Build the Rate Limiter directly from the parsed config file array
	limiter := ratelimiter.NewRedisTokenBucket(
		logger, 
		redisClient, 
		ratelimiter.WithKeyCapacity(cfg.ConfigServiceClient.RateLimits),
	)

	// Initialize the resilient, long-lived client with the configured rate limiter
	prodClient, _ := config.NewClient(
		"http://configsvc:8080",
		options.WithRateLimiter(limiter),
	)
}
```
