package client

import (
	orderSvc "github.com/phuthien0308/ordering/orderservice/pb"
	"google.golang.org/grpc"
)

// we need to implement a DNS resolver.
var dnsResolver = map[string]string{
	"orderService": "localhost:8080",
}

// NewOrderServiceClient ...
func NewOrderServiceClient() (orderSvc.OrderServiceClient, error) {
	// we need to think about service-to-service authorization
	conn, err := grpc.NewClient(dnsResolver["orderService"], grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return orderSvc.NewOrderServiceClient(conn), nil
}
