package main

import (
	"fmt"
	"net"

	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 8080))
	if err != nil {
		panic(err)
	}
	server := grpc.NewServer()

	server.Serve(lis)
}
