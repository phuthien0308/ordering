package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/go-zookeeper/zk"
	"go.uber.org/zap"
	"google.golang.org/protobuf/internal/errors"
)

type Config struct {
	Conn   *zk.Conn
	Logger *zap.Logger
}

func NewConfig(Conn *zk.Conn, Logger *zap.Logger) *Config {
	return &Config{
		Conn:   Conn,
		Logger: Logger,
	}
}

func (cf *Config) Get(ctx context.Context, path string) (string, error) {
	data, _, err := cf.Conn.Get(path)
	if err != nil {
		cf.Logger.Error("can not get path", zap.String("path", path), zap.Error(err))
		return "", err
	}
	return string(data), nil
}

func (cf *Config) Register(ctx context.Context, appName string, ip string) (string, error) {

	defer cf.Logger.Info("finished registering", zap.String("appname", appName), zap.String("ip", ip))
	appNode := time.Now().UTC().UnixNano()
	path := fmt.Sprintf("/services/%s/%v", appName, appNode)
	cf.Logger.Info("starting registering", zap.String("path", path), zap.String("appname", appName), zap.String("ip", ip))
	existed, _, err := cf.Conn.Exists(path)
	if err != nil {
		return "", err
	}

	if !existed {
		_, err := cf.Conn.Create(path, []byte(ip), zk.FlagPersistent, zk.WorldACL(zk.PermAll))
		if err != nil {
			cf.Logger.Error("can not register", zap.Error(err), zap.String("ip", ip))
			return "", err
		}
		return string(appNode), nil
	}
	return "", errors.Error("can not register")
}

func (cf *Config) Deregister(ctx context.Context, appName string, ip string) error {
	cf.Logger.Info("starting deregistering", zap.String("appname", appName), zap.String("ip", ip))
	defer cf.Logger.Info("finished deregistering", zap.String("appname", appName), zap.String("ip", ip))
	path := fmt.Sprintf("/services/%s/%s", appName, ip)
	existed, stats, err := cf.Conn.Exists(path)
	if err != nil {
		cf.Logger.Error("can not deregister", zap.String("path", path), zap.String("ip", ip), zap.Error(err))
		return err
	}
	if !existed {
		cf.Logger.Info("the path is not existed", zap.String("path", path))
		return nil
	}
	err = cf.Conn.Delete(path, stats.Version)
	if err != nil {
		cf.Logger.Error("can not deregister", zap.String("path", path), zap.Error(err))
		return err
	}
	return nil
}
