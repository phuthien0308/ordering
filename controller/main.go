package main

import (
	"context"
	"fmt"

	"github.com/phuthien0308/ordering/controller/client"
	"github.com/phuthien0308/ordering/orderservice/pb"
)

func main() {
	orderClient, err := client.NewOrderServiceClient()
	if err != nil {
		panic(err)
	}
	response, err := orderClient.PlaceOrder(context.Background(), &pb.OrderRequest{AccountId: "ABC"})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(response)
}
