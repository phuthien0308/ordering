package internal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-zookeeper/zk"
	"github.com/samber/lo"
	"go.uber.org/zap"
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

func (cf *Config) Register(ctx context.Context, appName string, ip string) error {
	cf.Logger.Info("starting registering", zap.String("appname", appName), zap.String("ip", ip))
	defer cf.Logger.Info("finished registering", zap.String("appname", appName), zap.String("ip", ip))
	path := fmt.Sprintf("/services/%s", appName)
	existed, _, err := cf.Conn.Exists(path)
	if err != nil {
		return err
	}

	if !existed {
		newIP, _ := json.Marshal([]string{ip})
		_, err := cf.Conn.Create(path, newIP, zk.FlagPersistent, zk.WorldACL(zk.PermAll))
		if err != nil {
			cf.Logger.Error("can not register", zap.Error(err))
			return err
		}
		return nil
	}

	data, stats, err := cf.Conn.Get(path)
	if err != nil {
		cf.Logger.Error("can not register", zap.Error(err))
		return err
	}
	var ips []string
	if err := json.Unmarshal(data, &ips); err != nil {
		cf.Logger.Error("can not register", zap.Error(err))
		return err
	}

	isExisted := lo.Contains(ips, ip)
	if isExisted {
		cf.Logger.Info("the ip is existing", zap.String("ip", ip))
		return nil
	}
	ips = append(ips, ip)
	ipsJson, _ := json.Marshal(ips)
	_, err = cf.Conn.Set(path, ipsJson, stats.Version)
	if err != nil {
		cf.Logger.Error("can not set new registers", zap.Error(err), zap.String("ips", string(ipsJson)))
		return err
	}
	return nil
}

func (cf *Config) Deregister(ctx context.Context, appName string, ip string) error {
	path := fmt.Sprintf("/services/%s", appName)
	existed, _, err := cf.Conn.Exists(path)
	if err != nil {
		cf.Logger.Error("can not deregister", zap.String("path", path), zap.Error(err))
		return err
	}
	if !existed {
		return nil
	}
	data, stats, err := cf.Conn.Get(path)
	if err != nil {
		cf.Logger.Error("can not deregister", zap.String("path", path), zap.Error(err))
		return err
	}
	var ips []string
	if err := json.Unmarshal(data, &ips); err != nil {
		cf.Logger.Error("can not deregister", zap.Strings("ips", ips), zap.Error(err))
		return err
	}

	newIps := lo.Filter(ips, func(item string, i int) bool {
		return item != ip
	})
	ipsJson, err := json.Marshal(newIps)
	if err != nil {
		cf.Logger.Error("can not deregister", zap.String("ips", string(ipsJson)), zap.Error(err))
		return err
	}
	_, err = cf.Conn.Set(path, ipsJson, stats.Version)
	if err != nil {
		cf.Logger.Error("can not deregister", zap.String("ips", string(ipsJson)), zap.Error(err))
		return err
	}

	return nil
}
