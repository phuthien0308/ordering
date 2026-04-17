package handler

import (
	"context"
	"fmt"

	"github.com/phuthien0308/ordering-contracts/gen/config"
	"github.com/phuthien0308/ordering/config/internal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ConfigImpl struct {
	config.UnimplementedConfigServiceServer
	LogicV1 internal.ConfigV1
	logger  *zap.Logger
}

func (cf *ConfigImpl) Register(ctx context.Context, rq *config.RegisterRequest) (*emptypb.Empty, error) {
	err := cf.LogicV1.Register(ctx, buildServiceAddress(rq.Appname), rq.Ip)
	if err != nil {
		cf.logger.Error("can not register", zap.Error(err))
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (cf *ConfigImpl) Deregister(ctx context.Context, rq *config.DeregisterRequest) (*emptypb.Empty, error) {
	err := cf.LogicV1.Deregister(ctx, buildServiceAddress(rq.Appname), rq.Ip)
	if err != nil {
		cf.logger.Error("can not deregister", zap.Error(err))
		return nil, err
	}
	return &emptypb.Empty{}, err
}

func (cf *ConfigImpl) GetAllAddresses(ctx context.Context, rq *config.GetAllAddressesRequest) (*config.GetAllAddressesResponse, error) {
	data, err := cf.LogicV1.GetAllAddresses(ctx, buildServiceAddress(rq.Appname))
	if err != nil {
		cf.logger.Error("can not get all addresses", zap.Error(err))
		return nil, err
	}
	return &config.GetAllAddressesResponse{Ips: data}, nil
}

func buildServiceAddress(appname string) string {
	return fmt.Sprintf("service_addresses:%v", appname)
}
