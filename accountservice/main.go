package main

import (
	"fmt"
	"net"

	"github.com/phuthien0308/ordering/accountservice/internal/account"
	"github.com/phuthien0308/ordering/accountservice/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	cred, err := credentials.NewServerTLSFromFile("server.crt", "server.key")
	if err != nil {
		panic(fmt.Errorf("can not load certificate, %w", err))
	}
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		panic(fmt.Errorf("port is occupied, %w", err))
	}
	server := grpc.NewServer(grpc.Creds(cred))
	pb.RegisterAccountServiceServer(server, account.AccountHandler{})
	fmt.Println("server is running")
	if err := server.Serve(lis); err != nil {
		panic("can not start the server")
	}
}
