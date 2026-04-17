package handlers

import (
	"context"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	pb "github.com/phuthien0308/ordering-base/contracts/product"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProductServiceHandler struct {
	pb.UnimplementedProductServiceServer
	repo *Repository
}

// NewProductHandler initialises the handler with a DynamoDB client.
// Reads DYNAMODB_ENDPOINT from the environment so it works both locally and in k8s.
func NewProductHandler() *ProductServiceHandler {
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:4566" // default for local dev
	}

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		),
	)
	if err != nil {
		panic("failed to load AWS config: " + err.Error())
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &ProductServiceHandler{
		repo: NewRepository(client),
	}
}

// ─── RPCs ────────────────────────────────────────────────────────────────────

func (h *ProductServiceHandler) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.ProductResponse, error) {
	err := h.repo.Create(
		ctx,
		req.Sku,
		req.Name,
		req.Category,
		req.Badge,
		req.Image,
		req.Description,
		req.Price,
		req.Rating,
		req.ReviewCount,
		req.Features,
		req.Attributes,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create product: %v", err)
	}

	return &pb.ProductResponse{
		Product: &pb.Product{
			Sku:         req.Sku,
			Name:        req.Name,
			Category:    req.Category,
			Badge:       req.Badge,
			Image:       req.Image,
			Description: req.Description,
			Price:       req.Price,
			Rating:      req.Rating,
			ReviewCount: req.ReviewCount,
			Features:    req.Features,
			Attributes:  req.Attributes,
		},
	}, nil
}

func (h *ProductServiceHandler) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.ProductResponse, error) {
	if req.Sku == "" {
		return nil, status.Error(codes.InvalidArgument, "sku is required")
	}

	// Build partial update map — only include fields that were set by the caller
	updates := make(map[string]types.AttributeValue)

	if req.Name != nil {
		updates["name"] = &types.AttributeValueMemberS{Value: *req.Name}
	}
	if req.Category != nil {
		updates["category"] = &types.AttributeValueMemberS{Value: *req.Category}
	}
	if req.Badge != nil {
		updates["badge"] = &types.AttributeValueMemberS{Value: *req.Badge}
	}
	if req.Image != nil {
		updates["image"] = &types.AttributeValueMemberS{Value: *req.Image}
	}
	if req.Description != nil {
		updates["description"] = &types.AttributeValueMemberS{Value: *req.Description}
	}
	if req.Price != nil {
		updates["price"] = &types.AttributeValueMemberN{Value: strconv.FormatFloat(*req.Price, 'f', -1, 64)}
	}
	if req.Rating != nil {
		updates["rating"] = &types.AttributeValueMemberN{Value: strconv.FormatFloat(*req.Rating, 'f', -1, 64)}
	}
	if req.ReviewCount != nil {
		updates["review_count"] = &types.AttributeValueMemberN{Value: strconv.Itoa(int(*req.ReviewCount))}
	}
	if len(req.Features) > 0 {
		featureList := make([]types.AttributeValue, 0, len(req.Features))
		for _, f := range req.Features {
			featureList = append(featureList, &types.AttributeValueMemberS{Value: f})
		}
		updates["features"] = &types.AttributeValueMemberL{Value: featureList}
	}
	if len(req.Attributes) > 0 {
		attrMap := make(map[string]types.AttributeValue, len(req.Attributes))
		for k, v := range req.Attributes {
			attrMap[k] = &types.AttributeValueMemberS{Value: v}
		}
		updates["attributes"] = &types.AttributeValueMemberM{Value: attrMap}
	}

	item, err := h.repo.Update(ctx, req.Sku, updates)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update product: %v", err)
	}

	return &pb.ProductResponse{Product: itemToProto(item)}, nil
}

// DeleteProduct — returns google.protobuf.Empty as defined in the proto
func (h *ProductServiceHandler) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*emptypb.Empty, error) {
	if req.Sku == "" {
		return nil, status.Error(codes.InvalidArgument, "sku is required")
	}

	if err := h.repo.Delete(ctx, req.Sku); err != nil {
		return nil, status.Errorf(codes.Internal, "delete product: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *ProductServiceHandler) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.SearchProductsResponse, error) {
	category, ok := req.AttributeFilters["category"]
	if !ok || category == "" {
		return nil, status.Error(codes.InvalidArgument, "attribute_filters must include 'category'")
	}

	items, lastKey, err := h.repo.Search(ctx, category, req.PageSize, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search products: %v", err)
	}

	products := make([]*pb.Product, 0, len(items))
	for _, item := range items {
		products = append(products, itemToProto(item))
	}

	// Encode the last evaluated key as next page token (sku of the last item)
	nextToken := ""
	if lastKey != nil {
		if v, ok := lastKey["sku"].(*types.AttributeValueMemberS); ok {
			nextToken = v.Value
		}
	}

	return &pb.SearchProductsResponse{
		Products:      products,
		NextPageToken: nextToken,
	}, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// itemToProto converts a raw DynamoDB item map into a proto Product message.
func itemToProto(item map[string]types.AttributeValue) *pb.Product {
	sku, name, category, badge, image, description, price, rating, reviewCount, features, attributes := itemToFields(item)
	return &pb.Product{
		Sku:         sku,
		Name:        name,
		Category:    category,
		Badge:       badge,
		Image:       image,
		Description: description,
		Price:       price,
		Rating:      rating,
		ReviewCount: reviewCount,
		Features:    features,
		Attributes:  attributes,
	}
}
