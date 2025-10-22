package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/phuthien0308/ordering/config/pb"
	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 50051))
	if err != nil {
		panic(err)
	}
	hostEnv := os.Getenv("hosts")
	hosts := strings.Split(hostEnv, ",")
	server := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	conn, _, err := zkConnection(hosts)
	if err != nil {
		panic(err)
	}

	err = createParentZnodeIfNotExists(conn, "/services")
	if err != nil {
		panic(err)
	}

	go func() {
		pb.RegisterConfigServer(server, &configImpl{conn: conn, hosts: hosts})
		grpc_health_v1.RegisterHealthServer(server, &healthCheckServer{})
		if err := server.Serve(lis); err != nil {
			panic(err)
		}
	}()

	waitForShutdown()
}

type healthCheckServer struct {
	grpc_health_v1.UnimplementedHealthServer
}

func (health *healthCheckServer) Check(context.Context, *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

type configImpl struct {
	pb.UnimplementedConfigServer
	conn  *zk.Conn
	hosts []string
}

func zkConnection(hosts []string) (*zk.Conn, <-chan zk.Event, error) {
	return zk.Connect(hosts, 5*time.Second)
}

func (cf *configImpl) Get(ctx context.Context, request *pb.ConfigRequest) (*pb.ConfigResponse, error) {
	data, _, err := cf.conn.Get(request.Path)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &pb.ConfigResponse{Data: string(data)}, nil
}

func (cf *configImpl) Watch(request *pb.ConfigRequest, response grpc.ServerStreamingServer[pb.ConfigResponse]) error {
	for {
		data, _, eventChan, err := cf.conn.GetW(request.Path)
		if err != nil {
			return err
		}
		response.Send(&pb.ConfigResponse{Data: string(data)})
		event := <-eventChan
		if event.Type == zk.EventNodeDataChanged || event.Type == zk.EventNodeCreated {
			continue
		}
		if event.Type == zk.EventSession {
			cf.conn, _, err = zkConnection(cf.hosts)
			if err != nil {
				return err
			}
		}

	}

}

func (cf *configImpl) Register(ctx context.Context, rq *pb.RegisterRequest) (*pb.RegisterResponse, error) {

	path := fmt.Sprintf("/services/%s", rq.Appname)
	existed, _, err := cf.conn.Exists(path)
	if err != nil {
		return nil, err
	}

	if !existed {
		newIP, _ := json.Marshal([]string{rq.Ip})
		_, err := cf.conn.Create(path, newIP, zk.FlagPersistent, zk.WorldACL(zk.PermAll))
		if err != nil {
			fmt.Println(err, path)
			return nil, err
		}
		return &pb.RegisterResponse{}, nil
	}

	data, stats, err := cf.conn.Get(path)
	if err != nil {
		return nil, err
	}
	var ips []string
	if err := json.Unmarshal(data, &ips); err != nil {
		return nil, err
	}

	isExisted := lo.Contains(ips, rq.Ip)
	if isExisted {
		return &pb.RegisterResponse{}, nil
	}
	ips = append(ips, rq.Ip)
	ipsJson, _ := json.Marshal(ips)
	_, err = cf.conn.Set(path, ipsJson, stats.Version)
	if err != nil {
		return nil, err
	}
	return &pb.RegisterResponse{}, nil
}

func (cf *configImpl) Deregister(ctx context.Context, rq *pb.DeregisterRequest) (*pb.DeregisterResponse, error) {
	path := fmt.Sprintf("/services/%s", rq.Appname)
	existed, _, err := cf.conn.Exists(path)
	if err != nil {
		return nil, err
	}
	if !existed {
		return &pb.DeregisterResponse{}, nil
	}
	data, stats, err := cf.conn.Get(path)
	if err != nil {
		return nil, err
	}
	var ips []string
	if err := json.Unmarshal(data, &ips); err != nil {
		return nil, err
	}

	newIps := lo.Filter(ips, func(item string, i int) bool {
		return item != rq.Ip
	})
	ipsJson, err := json.Marshal(newIps)
	if err != nil {
		return nil, err
	}
	_, err = cf.conn.Set(path, ipsJson, stats.Version)
	if err != nil {
		return nil, err
	}

	return &pb.DeregisterResponse{}, nil
}

func createParentZnodeIfNotExists(conn *zk.Conn, parentPath string) error {
	exists, _, err := conn.Exists(parentPath)
	if err != nil {
		return err
	}
	if !exists {
		_, err = conn.Create(parentPath, []byte{}, 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			return err
		}
		fmt.Printf("Created parent znode: %s\n", parentPath)
	}
	return nil
}

func waitForShutdown() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM)
	<-sig
	fmt.Println("Shutting down gracefully...")

	fmt.Println("Service stopped.")
}
