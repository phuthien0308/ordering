package repository

import (
	"context"

	"github.com/phuthien0308/ordering/common/log"
	"github.com/phuthien0308/ordering/orderservice/model"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type OrderRepository interface {
	Add(ctx context.Context, item model.Order) (string, error)
	AddRange(ctx context.Context, items []model.Order) error
	Update(ctx context.Context, item model.Order) error
	Delete(ctx context.Context, id string) error
}

var Order OrderRepository = &orderRepository{}

type orderRepository struct {
	db     mongo.Database
	logger log.Logger
}

func (repo *orderRepository) Add(ctx context.Context, item model.Order) (string, error) {
	result, err := repo.db.Collection("orders").InsertOne(ctx, item)
	if err != nil {
		repo.logger.Error(ctx, "can not insert item", err, log.NewTag("item", item))
		return "", err
	}
	return string(result.InsertedID.([]byte)), nil
}

func (repo *orderRepository) Update(ctx context.Context, item model.Order) error {
	panic("not implemented")
}

func (repo *orderRepository) Delete(ctx context.Context, id string) error {
	panic("not implemented")
}

func (repo *orderRepository) AddRange(ctx context.Context, items []model.Order) error {
	panic("not implemented")
}
