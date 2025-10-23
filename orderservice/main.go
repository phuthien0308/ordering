package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/phuthien0308/ordering/config/pb"
	"github.com/phuthien0308/ordering/orderservice/clients"
	"github.com/phuthien0308/ordering/orderservice/handler"
	"github.com/phuthien0308/ordering/orderservice/helper"
)

func main() {
	// lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 8080))
	// if err != nil {
	// 	panic(err)
	// }
	// server := grpc.NewServer()
	// pb.RegisterOrderServiceServer(server, handler.NewHandler())
	// fmt.Println("Server is starting")
	// server.Serve(lis)
	// http.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {

	// })

	// var pingCounter = prometheus.NewCounter(
	// 	prometheus.CounterOpts{
	// 		Name: "ping_request_count",
	// 		Help: "No of request handled by Ping handler",
	// 	},
	// )
	// var requestDuration = prometheus.NewGauge(prometheus.GaugeOpts{
	// 	Name: "ping_request_duration",
	// })
	// prometheus.MustRegister(pingCounter, requestDuration)
	// http.Handle("/metrics", promhttp.Handler())
	// http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {

	// 	now := time.Now()
	// 	defer func(n time.Time) {
	// 		escapled := n.Add(1 * time.Second)
	// 		requestDuration.Add(float64(escapled.Second()))
	// 	}(now)

	// 	pingCounter.Add(10)
	// 	w.Write([]byte("hello world"))
	// })
	// http.ListenAndServe(":8090", nil)

	r := gin.Default()
	r.POST("/orders", handler.AddOrderHandler)
	r.GET("/healthz", func(ctx *gin.Context) {
		ctx.Writer.WriteHeader(http.StatusOK)
	})

	go func() {
		fmt.Println("starting the service")
		r.Run(":8081")
	}()

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan

	clients.ConfigClient.Deregister(context.Background(), &pb.DeregisterRequest{
		Appname: helper.AppName,
		Ip:      helper.HealthCheckEndpoint,
	})
}
