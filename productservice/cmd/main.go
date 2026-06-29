package main

import (
	"fmt"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/phuthien0308/ordering-base/contracts/product"
	"github.com/phuthien0308/ordering/productservice/handlers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	httpServer()
	listener, err := net.Listen("tcp", "localhost:5000")
	if err != nil {
		panic(err)
	}
	server := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	product.RegisterProductServiceServer(server, handlers.NewProductHandler())
	fmt.Println("Server is running")

	if err := server.Serve(listener); err != nil {
		panic(err)
	}
}

func httpServer() {
	r := mux.NewRouter()
	r.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})).Methods("GET")
	err := http.ListenAndServe(":8089", r)
	if err != nil {
		fmt.Println(err)
	}
}
