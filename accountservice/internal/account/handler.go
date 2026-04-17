package account

import (
	"context"

	"github.com/phuthien0308/ordering/accountservice/pb"
)

type AccountHandler struct {
	pb.UnimplementedAccountServiceServer
}

func (account AccountHandler) CreateAccount(context.Context, *pb.CreatedAccountRequest) (*pb.CreatedAccountResponse, error) {
	return &pb.CreatedAccountResponse{
		AccountId: "123",
	}, nil
}
