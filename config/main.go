package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/phuthien0308/ordering/config/pb"
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 8080))
	if err != nil {
		panic(err)
	}
	hostEnv := os.Getenv("hosts")
	hosts := strings.Split(hostEnv, ",")
	server := grpc.NewServer()
	pb.RegisterConfigServer(server, &configImpl{hosts: hosts})
	server.Serve(lis)
}

type configImpl struct {
	pb.UnimplementedConfigServer
	hosts []string
}

func zkConnection(hosts []string) (*zk.Conn, <-chan zk.Event, error) {
	return zk.Connect(hosts, 5*time.Second)
}

func (cf *configImpl) Get(ctx context.Context, request *pb.ConfigRequest) (*pb.ConfigResponse, error) {
	conn, _, err := zkConnection(cf.hosts)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	data, _, err := conn.Get(request.Path)
	if err != nil {
		return nil, err
	}
	return &pb.ConfigResponse{Data: string(data)}, nil
}

func (cf *configImpl) Watch(request *pb.ConfigRequest, response grpc.ServerStreamingServer[pb.ConfigResponse]) error {
	conn, _, err := zkConnection(cf.hosts)
	if err != nil {
		return err
	}
	defer conn.Close()
	for {
		data, _, eventChan, err := conn.GetW(request.Path)
		if err != nil {
			return err
		}
		response.Send(&pb.ConfigResponse{Data: string(data)})
		event := <-eventChan
		if event.Type == zk.EventNodeDataChanged || event.Type == zk.EventNodeCreated {
			continue
		}
		if event.Type == zk.EventSession {
			conn, _, err = zkConnection(cf.hosts)
			if err != nil {
				return err
			}
		}

	}

}
