package handler

import (
	"context"
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/phuthien0308/ordering/config/internal"
	"github.com/phuthien0308/ordering/config/pb"
	"google.golang.org/grpc"
)

type ConfigImpl struct {
	pb.UnimplementedConfigServer
	Conn  *zk.Conn
	Hosts []string
	Logic *internal.Config
}

func (cf *ConfigImpl) Get(ctx context.Context, request *pb.ConfigRequest) (*pb.ConfigResponse, error) {

	result, err := cf.Logic.Get(ctx, request.Path)
	if err != nil {
		return nil, err

	}
	return &pb.ConfigResponse{Data: result}, nil
}

func (cf *ConfigImpl) Watch(request *pb.ConfigRequest, response grpc.ServerStreamingServer[pb.ConfigResponse]) error {
	for {
		data, _, eventChan, err := cf.Conn.GetW(request.Path)
		if err != nil {
			return err
		}
		response.Send(&pb.ConfigResponse{Data: string(data)})
		event := <-eventChan
		if event.Type == zk.EventNodeDataChanged || event.Type == zk.EventNodeCreated {
			continue
		}
		if event.Type == zk.EventSession {
			cf.Conn, _, err = zkConnection(cf.Hosts)
			if err != nil {
				return err
			}
		}

	}

}

func (cf *ConfigImpl) Register(ctx context.Context, rq *pb.RegisterRequest) (*pb.RegisterResponse, error) {

	appNode, err := cf.Logic.Register(ctx, rq.Appname, rq.Ip)
	if err != nil {
		return nil, err
	}
	return &pb.RegisterResponse{AppNode: appNode}, nil

}

func (cf *ConfigImpl) Deregister(ctx context.Context, rq *pb.DeregisterRequest) (*pb.DeregisterResponse, error) {
	err := cf.Logic.Deregister(ctx, rq.Appname, rq.Ip)
	if err != nil {
		return nil, err
	}
	return nil, err
}

func zkConnection(Hosts []string) (*zk.Conn, <-chan zk.Event, error) {
	return zk.Connect(Hosts, 5*time.Second)
}
