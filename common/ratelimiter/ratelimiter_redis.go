package ratelimiter

import (
	"context"
	"errors"
	"time"

	"github.com/phuthien0308/ordering/common/log"
	"github.com/redis/go-redis/v9"
)

var luaScript = redis.NewScript(`
local cnt = redis.call('ZCARD', KEYS[1])
if tonumber(cnt) < tonumber(ARGV[3]) then
    redis.call('ZADD', KEYS[1], ARGV[1], ARGV[1])
    redis.call('ZREMRANGEBYSCORE', KEYS[1], 0, ARGV[1] - ARGV[2])
	redis.call('PEXPIRE', KEYS[1], ARGV[2] + 1000)	
  return 1
else
  return 0
end
`)

type ratelimiterRedis struct {
	command     string
	timeWindow  time.Duration
	volumne     int64
	redisClient redis.Client
	logger      log.Logger
}

func NewRateLimiterRedis(command string, window time.Duration, volumne int64) *ratelimiterRedis {

	redisC := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	return &ratelimiterRedis{
		command:     command,
		timeWindow:  window,
		volumne:     volumne,
		logger:      log.NewLogger(log.INFO, nil),
		redisClient: *redisC,
	}
}

func (r *ratelimiterRedis) Accquire(ctx context.Context) (bool, error) {
	now := time.Now().UnixMilli()
	cmd, err := luaScript.Run(ctx, r.redisClient, []string{r.command}, now, r.timeWindow.Milliseconds(), r.volumne).Result()
	if err != nil {
		newError := errors.Join(err, errors.New("can not accquire"))
		r.logger.Error(ctx, "can not execute redis command", newError)
		return false, err
	}
	if cmd.(int64) == 1 {
		return true, nil
	}
	return false, nil
}
