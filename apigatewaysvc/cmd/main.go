package main

import (
	"log"
	"net/http"
	"os"

	"github.com/phuthien0308/ordering/apigatewaysvc/handlers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	productAddr := os.Getenv("PRODUCT_SERVICE_ADDR")
	if productAddr == "" {
		productAddr = "localhost:5000" // default for local dev
	}

	// Create one long-lived gRPC connection per downstream service.
	// gRPC connections are multiplexed — reusing them is efficient and correct.
	productConn, err := grpc.NewClient(productAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect to productservice at %s: %v", productAddr, err)
	}
	defer productConn.Close()

	productHandler := handlers.NewProductHandler(productConn)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/products", productHandler.CreateProduct)
	mux.HandleFunc("PUT /v1/products/{sku}", productHandler.UpdateProduct)
	mux.HandleFunc("DELETE /v1/products/{sku}", productHandler.DeleteProduct)
	mux.HandleFunc("GET /v1/products/search", productHandler.SearchProducts)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("apigatewaysvc listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
