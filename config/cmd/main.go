package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/phuthien0308/ordering-contracts/gen/config"
	"github.com/phuthien0308/ordering/config/configuration"
	_ "github.com/phuthien0308/ordering/config/configuration"
	"github.com/phuthien0308/ordering/config/consumer"
	"github.com/phuthien0308/ordering/config/handler"
	"github.com/phuthien0308/ordering/config/internal"
	"github.com/phuthien0308/ordering/config/storage"
	"github.com/phuthien0308/ordering/config/worker"
	"github.com/phuthien0308/simplelog"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	errMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
	logger             *simplelog.SimpleZapLogger
)

func main() {
	port := 50051

	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("can not initialize logger")
		panic(err)
	}

	if configuration.Config.Env != "development" {
		zapLogger, err = zap.NewProduction()
		if err != nil {
			panic(err)
		}
	}

	defer zapLogger.Sync()

	logger = &simplelog.SimpleZapLogger{
		Logger: zapLogger,
	}

	zapLogger = zapLogger.With()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	redisClient := initRedisClient()

	server := grpc.NewServer(grpc.Creds(insecure.NewCredentials()),
		grpc.UnaryInterceptor(requestInterceptor(logger)),
	)

	go func() {
		fmt.Printf("the server is running with port %v\n", port)
		config.RegisterConfigServiceServer(server, &handler.ConfigImpl{LogicV1: internal.NewConfigV1(redisClient, zapLogger)})
		grpc_health_v1.RegisterHealthServer(server, &healthCheckServer{})
		if err := server.Serve(lis); err != nil {
			panic(err)
		}
	}()

	interval := 10 * time.Second
	worker := worker.NewHealhCheckWorker(zapLogger, redisClient, interval)
	go worker.Start(context.Background())

	consumer := consumer.NewConsumer(zapLogger, storage.NewAddressStorage(redisClient))
	consumerSignal, err := consumer.Start(context.Background())
	if err != nil {
		panic("can't start consumer")
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	broastcastShutdown(sig, consumerSignal)

	fmt.Println("Shutting down gracefully...")
	fmt.Println("Service stopped.")
}

type healthCheckServer struct {
	grpc_health_v1.UnimplementedHealthServer
}

func (health *healthCheckServer) Check(context.Context, *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func broastcastShutdown(sig chan os.Signal, chans ...chan interface{}) {
	<-sig
	for _, ch := range chans {
		ch <- struct{}{}
	}
	os.Exit(1)
}

func initRedisClient() *redis.Client {
	redisAddress := configuration.Config.RedisHost
	if redisAddress == "" {
		panic("can not connect to redis server")
	}
	client := redis.NewClient(&redis.Options{
		Addr: redisAddress,
	})
	return client
}

func requestInterceptor(logger *simplelog.SimpleZapLogger) grpc.UnaryServerInterceptor {

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		//sc := trace.SpanContextFromContext(ctx)

		fields := []zap.Field{zap.String(
			"rpc.system", "grpc"),
			zap.String("grpc.method", info.FullMethod),
			zap.String("service", "config"),
			zap.String("environment", configuration.Config.Env),
		}

		// It works if the trace provider is initialized
		// if sc.IsValid() {
		// 	fields = append(fields,
		// 		zap.String("trace-id", sc.TraceID().String()),
		// 		zap.String("span-id", sc.SpanID().String()),
		// 	)
		// }

		var requestID string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if v := md.Get("x-request-id"); len(v) > 0 {
				requestID = v[0]
			}
		}
		if requestID != "" {
			fields = append(fields, zap.String("request-id", requestID))
		}
		logger.Info("grpc request", fields...)

		ctx = context.WithValue(ctx, simplelog.SimpleLogKeyCtx, fields)

		resp, err := handler(ctx, req)

		duration := time.Since(start)

		code := status.Convert(err)
		fields = append(fields, zap.String("grpc.code", code.String()), zap.Duration("duration", duration))

		if err == nil {
			logger.WithContext(ctx).Info(
				"grpc request succeeded",
				append(fields, zap.Error(err))...,
			)
		} else {
			logger.WithContext(ctx).Error(
				"grpc request failed",
				append(fields, zap.Error(err))...,
			)
		}
		return resp, err
	}
}
