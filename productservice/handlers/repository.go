package handlers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const tableName = "Products"

type Repository struct {
	client *dynamodb.Client
}

func NewRepository(client *dynamodb.Client) *Repository {
	return &Repository{client: client}
}

// productToItem converts product fields into a DynamoDB attribute map.
func productToItem(sku, name, category, badge, image, description string, price, rating float64, reviewCount int32, features []string, attributes map[string]string) map[string]types.AttributeValue {
	item := map[string]types.AttributeValue{
		"sku":          &types.AttributeValueMemberS{Value: sku},
		"name":         &types.AttributeValueMemberS{Value: name},
		"category":     &types.AttributeValueMemberS{Value: category},
		"badge":        &types.AttributeValueMemberS{Value: badge},
		"image":        &types.AttributeValueMemberS{Value: image},
		"description":  &types.AttributeValueMemberS{Value: description},
		"price":        &types.AttributeValueMemberN{Value: strconv.FormatFloat(price, 'f', -1, 64)},
		"rating":       &types.AttributeValueMemberN{Value: strconv.FormatFloat(rating, 'f', -1, 64)},
		"review_count": &types.AttributeValueMemberN{Value: strconv.Itoa(int(reviewCount))},
	}

	// features → DynamoDB List
	if len(features) > 0 {
		featureList := make([]types.AttributeValue, 0, len(features))
		for _, f := range features {
			featureList = append(featureList, &types.AttributeValueMemberS{Value: f})
		}
		item["features"] = &types.AttributeValueMemberL{Value: featureList}
	}

	// attributes → DynamoDB Map
	if len(attributes) > 0 {
		attrMap := make(map[string]types.AttributeValue, len(attributes))
		for k, v := range attributes {
			attrMap[k] = &types.AttributeValueMemberS{Value: v}
		}
		item["attributes"] = &types.AttributeValueMemberM{Value: attrMap}
	}

	return item
}

// itemToFields extracts raw fields from a DynamoDB item map.
func itemToFields(item map[string]types.AttributeValue) (sku, name, category, badge, image, description string, price, rating float64, reviewCount int32, features []string, attributes map[string]string) {
	getString := func(key string) string {
		if v, ok := item[key].(*types.AttributeValueMemberS); ok {
			return v.Value
		}
		return ""
	}
	getFloat := func(key string) float64 {
		if v, ok := item[key].(*types.AttributeValueMemberN); ok {
			f, _ := strconv.ParseFloat(v.Value, 64)
			return f
		}
		return 0
	}
	getInt32 := func(key string) int32 {
		if v, ok := item[key].(*types.AttributeValueMemberN); ok {
			i, _ := strconv.ParseInt(v.Value, 10, 32)
			return int32(i)
		}
		return 0
	}

	sku = getString("sku")
	name = getString("name")
	category = getString("category")
	badge = getString("badge")
	image = getString("image")
	description = getString("description")
	price = getFloat("price")
	rating = getFloat("rating")
	reviewCount = getInt32("review_count")

	if v, ok := item["features"].(*types.AttributeValueMemberL); ok {
		for _, f := range v.Value {
			if s, ok := f.(*types.AttributeValueMemberS); ok {
				features = append(features, s.Value)
			}
		}
	}

	if v, ok := item["attributes"].(*types.AttributeValueMemberM); ok {
		attributes = make(map[string]string, len(v.Value))
		for k, av := range v.Value {
			if s, ok := av.(*types.AttributeValueMemberS); ok {
				attributes[k] = s.Value
			}
		}
	}

	return
}

// Create writes a new product item. Fails if sku already exists.
func (r *Repository) Create(ctx context.Context, sku, name, category, badge, image, description string, price, rating float64, reviewCount int32, features []string, attributes map[string]string) error {
	item := productToItem(sku, name, category, badge, image, description, price, rating, reviewCount, features, attributes)
	_, err := r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(sku)"), // prevent overwrite
	})
	if err != nil {
		return fmt.Errorf("create product: %w", err)
	}
	return nil
}

// Update applies partial updates to an existing product.
func (r *Repository) Update(ctx context.Context, sku string, updates map[string]types.AttributeValue) (map[string]types.AttributeValue, error) {
	if len(updates) == 0 {
		return r.Get(ctx, sku)
	}

	exprNames := make(map[string]string)
	exprValues := make(map[string]types.AttributeValue)
	setParts := make([]string, 0, len(updates))

	i := 0
	for field, val := range updates {
		nameKey := fmt.Sprintf("#f%d", i)
		valKey := fmt.Sprintf(":v%d", i)
		exprNames[nameKey] = field
		exprValues[valKey] = val
		setParts = append(setParts, fmt.Sprintf("%s = %s", nameKey, valKey))
		i++
	}

	updateExpr := "SET " + joinStrings(setParts, ", ")

	out, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"sku": &types.AttributeValueMemberS{Value: sku},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: exprValues,
		ConditionExpression:       aws.String("attribute_exists(sku)"), // fail if not found
		ReturnValues:              types.ReturnValueAllNew,
	})
	if err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}
	return out.Attributes, nil
}

// Delete removes a product by SKU.
func (r *Repository) Delete(ctx context.Context, sku string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"sku": &types.AttributeValueMemberS{Value: sku},
		},
		ConditionExpression: aws.String("attribute_exists(sku)"), // fail silently if not found would skip this
	})
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	return nil
}

// Get fetches a single product by SKU.
func (r *Repository) Get(ctx context.Context, sku string) (map[string]types.AttributeValue, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"sku": &types.AttributeValueMemberS{Value: sku},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	if out.Item == nil {
		return nil, fmt.Errorf("product not found: %s", sku)
	}
	return out.Item, nil
}

// Search queries the CategoryPriceIndex GSI by category, with optional price range.
func (r *Repository) Search(ctx context.Context, category string, pageSize int32, pageToken map[string]types.AttributeValue) ([]map[string]types.AttributeValue, map[string]types.AttributeValue, error) {
	if pageSize <= 0 {
		pageSize = 20
	}

	limit := int32(pageSize)
	input := &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		IndexName:              aws.String("CategoryPriceIndex"),
		KeyConditionExpression: aws.String("category = :cat"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":cat": &types.AttributeValueMemberS{Value: category},
		},
		Limit:             &limit,
		ExclusiveStartKey: pageToken,
	}

	out, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, nil, fmt.Errorf("search products: %w", err)
	}
	return out.Items, out.LastEvaluatedKey, nil
}

func joinStrings(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}
