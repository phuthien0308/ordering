package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/phuthien0308/ordering/orderservice/config"
	"github.com/phuthien0308/ordering/orderservice/model"
	"github.com/phuthien0308/ordering/orderservice/repository"
)

var AddOrderHandler = func(ctx *gin.Context) {
	body := []model.Order{}
	if err := ctx.BindJSON(&body); err != nil {
		config.Logger.Error(ctx, "can not bind json", err)
		ctx.AbortWithError(http.StatusBadRequest, err)
	}
	fmt.Println(body)
	if err := repository.Order.AddRange(ctx, body); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
	}
}
