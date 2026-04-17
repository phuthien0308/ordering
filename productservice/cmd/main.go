package main

import (
	"net"

	"github.com/phuthien0308/ordering-base/contracts/product"
	"github.com/phuthien0308/ordering/productservice/handlers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	listener, err := net.Listen("tcp", "localhost:5000")
	if err != nil {
		panic(err)
	}
	server := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	product.RegisterProductServiceServer(server, handlers.NewProductHandler())
	if err := server.Serve(listener); err != nil {
		panic(err)
	}
}
