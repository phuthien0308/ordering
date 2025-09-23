package handler

import (
	"context"
	"fmt"

	"github.com/phuthien0308/orderservice/pb"
)

type Handler struct {
	pb.UnimplementedOrderServiceServer
}

func NewHandler() *Handler {
	return &Handler{UnimplementedOrderServiceServer: pb.UnimplementedOrderServiceServer{}}
}

func (h *Handler) PlaceOrder(ctx context.Context, request *pb.OrderRequest) (*pb.OrderResponse, error) {
	return &pb.OrderResponse{OrderId: fmt.Sprintf("hello world %v", request.AccountId)}, nil
}
