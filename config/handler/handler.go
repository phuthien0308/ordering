package handler

import (
	"context"

	"github.com/phuthien0308/ordering-contracts/gen/config"
	"github.com/phuthien0308/ordering/config/dto"
	"github.com/phuthien0308/ordering/config/internal"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ConfigImpl struct {
	config.UnimplementedConfigServiceServer
	LogicV1 internal.ConfigV1
}

func (cf *ConfigImpl) Register(ctx context.Context, rq *config.RegisterRequest) (*emptypb.Empty, error) {
	err := cf.LogicV1.Register(ctx, dto.BuildServiceAddress(rq.Appname), rq.Ip)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (cf *ConfigImpl) Deregister(ctx context.Context, rq *config.DeregisterRequest) (*emptypb.Empty, error) {
	err := cf.LogicV1.Deregister(ctx, dto.BuildServiceAddress(rq.Appname), rq.Ip)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, err
}

func (cf *ConfigImpl) GetAllAddresses(ctx context.Context, rq *config.GetAllAddressesRequest) (*config.GetAllAddressesResponse, error) {
	data, err := cf.LogicV1.GetAllAddresses(ctx, dto.BuildServiceAddress(rq.Appname))
	if err != nil {
		return nil, err
	}
	return &config.GetAllAddressesResponse{Ips: data}, nil
}
