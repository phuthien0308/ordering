package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/phuthien0308/ordering/config/handler"
	"github.com/phuthien0308/ordering/config/internal"
	"github.com/phuthien0308/ordering/config/pb"
	"github.com/phuthien0308/ordering/config/worker"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	port := 50051
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

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	redisClient := initRedisClient()
	server := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))

	go func() {
		pb.RegisterConfigServer(server, &handler.ConfigImpl{LogicV1: internal.NewConfigV1(redisClient, zapLogger)})
		grpc_health_v1.RegisterHealthServer(server, &healthCheckServer{})
		if err := server.Serve(lis); err != nil {
			panic(err)
		}
	}()

	interval := 10 * time.Second

	worker := worker.NewHealhCheckWorker(zapLogger, redisClient, interval)
	go worker.Start(context.Background())
	fmt.Printf("the server is running with port %v\n", port)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM)
	broastcastShutdown(sig)
	fmt.Println("Shutting down gracefully...")
	fmt.Println("Service stopped.")
}

type healthCheckServer struct {
	grpc_health_v1.UnimplementedHealthServer
}

func (health *healthCheckServer) Check(context.Context, *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func broastcastShutdown(sig chan os.Signal) {
	<-sig

}

func initRedisClient() *redis.Client {
	redisAddress := os.Getenv("REDIS_HOST")
	if redisAddress == "" {
		panic("can not connect to redis server")
	}
	client := redis.NewClient(&redis.Options{
		Addr: redisAddress,
	})
	return client
}
