package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/phuthien0308/ordering/config/handler"
	"github.com/phuthien0308/ordering/config/internal"
	"github.com/phuthien0308/ordering/config/pb"
	"github.com/phuthien0308/ordering/config/worker"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	hostEnv := os.Getenv("hosts")
	hosts := strings.Split(hostEnv, ",")
	environment := "dev"
	if os.Getenv("env") != "" {
		environment = os.Getenv("env")
	}

	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("can not initialize logger")
		panic(err)
	}
	if environment != "dev" {
		zapLogger, err = zap.NewProduction()
		if err != nil {
			panic(err)
		}
	}

	defer zapLogger.Sync()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 50051))
	if err != nil {
		panic(err)
	}

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
		pb.RegisterConfigServer(server, &handler.ConfigImpl{Conn: conn, Hosts: hosts, Logic: &internal.Config{Conn: conn, Logger: zapLogger}})
		grpc_health_v1.RegisterHealthServer(server, &healthCheckServer{})
		if err := server.Serve(lis); err != nil {
			panic(err)
		}
	}()
	interval := 5 * time.Second
	retry := 1 * time.Second

	if envRetry := os.Getenv("retryInSecond"); envRetry != "" {
		newRetry, _ := strconv.Atoi(envRetry)
		retry = time.Duration(newRetry) * time.Second
	}

	if envHealthCheck := os.Getenv("healthCheckInSecond"); envHealthCheck != "" {
		newHealthCheck, _ := strconv.Atoi(envHealthCheck)
		interval = time.Duration(newHealthCheck) * time.Second
	}

	worker := worker.NewHealhCheckWorker(zapLogger, conn, interval, retry)
	go worker.Start(context.Background())

	waitForShutdown()
}

type healthCheckServer struct {
	grpc_health_v1.UnimplementedHealthServer
}

func (health *healthCheckServer) Check(context.Context, *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func zkConnection(Hosts []string) (*zk.Conn, <-chan zk.Event, error) {
	return zk.Connect(Hosts, 5*time.Second)
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
