package handlers

import (
	"io"
	"net/http"

	pb "github.com/phuthien0308/ordering-base/contracts/product"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ProductHandler holds a long-lived gRPC client.
// The connection is created once at startup and reused across all requests.
type ProductHandler struct {
	client pb.ProductServiceClient
}

func NewProductHandler(conn *grpc.ClientConn) *ProductHandler {
	return &ProductHandler{
		client: pb.NewProductServiceClient(conn),
	}
}

func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req pb.CreateProductRequest
	if err := readProtoBody(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := h.client.CreateProduct(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeProtoResponse(w, http.StatusCreated, res.GetProduct())
}

func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	sku := r.PathValue("sku")
	var req pb.UpdateProductRequest
	if err := readProtoBody(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Sku = sku // path param takes precedence
	res, err := h.client.UpdateProduct(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeProtoResponse(w, http.StatusOK, res.GetProduct())
}

func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	sku := r.PathValue("sku")
	if sku == "" {
		http.Error(w, "sku is required", http.StatusBadRequest)
		return
	}
	_, err := h.client.DeleteProduct(r.Context(), &pb.DeleteProductRequest{Sku: sku})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductHandler) SearchProducts(w http.ResponseWriter, r *http.Request) {
	var req pb.SearchProductsRequest
	if err := readProtoBody(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := h.client.SearchProducts(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeProtoResponse(w, http.StatusOK, res)
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

// readProtoBody deserializes the HTTP request body directly into a proto message.
// No intermediate struct or manual field mapping needed.
func readProtoBody(r *http.Request, msg proto.Message) error {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(body, msg)
}

// writeProtoResponse serializes a proto message to JSON and writes it to the response.
func writeProtoResponse(w http.ResponseWriter, statusCode int, msg proto.Message) {
	data, err := protojson.Marshal(msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(data)
}
