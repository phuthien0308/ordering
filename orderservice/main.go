package main

import (
	"fmt"
	"net"

	"github.com/phuthien0308/orderservice/handler"
	"github.com/phuthien0308/orderservice/pb"
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 8080))
	if err != nil {
		panic(err)
	}
	server := grpc.NewServer()
	pb.RegisterOrderServiceServer(server, handler.NewHandler())
	server.Serve(lis)
}
