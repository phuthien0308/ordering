package internal

import (
	"context"

	"github.com/phuthien0308/ordering/config/storage"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type ConfigV1 interface {
	Register(ctx context.Context, appName string, ip string) error
	Deregister(ctx context.Context, appName string, ip string) error
	GetAllAddresses(ctx context.Context, appname string) ([]string, error)
}

func NewConfigV1(rd *redis.Client, logger *zap.Logger) ConfigV1 {
	nodeStorage := storage.NewAddressStorage(rd)
	return &configV1{
		nodeStorage: nodeStorage,
		logger:      logger,
	}
}

type configV1 struct {
	nodeStorage storage.AddressStorage
	logger      *zap.Logger
}

// GetAllAddresses implements ConfigV1.
func (c *configV1) GetAllAddresses(ctx context.Context, appname string) ([]string, error) {
	ips, err := c.nodeStorage.GetAddresses(ctx, appname)
	if err != nil {
		c.logger.Error("can not get all addresses", zap.String("appname", appname), zap.Error(err))
		return []string{}, err
	}
	return ips, nil
}

// Register implements ConfigV1.
func (c *configV1) Register(ctx context.Context, appName string, ip string) error {
	err := c.nodeStorage.Add(ctx, appName, ip)
	if err != nil {
		return err
	}
	return nil
}

// Deregister implements ConfigV1.
func (c *configV1) Deregister(ctx context.Context, appName string, ip string) error {
	return c.nodeStorage.Remove(ctx, appName, ip)
}
