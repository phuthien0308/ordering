package main

import (
	"github.com/gin-gonic/gin"
	"github.com/phuthien0308/ordering/orderservice/handler"
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

	r.Run()

}
