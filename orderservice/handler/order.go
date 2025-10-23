package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/phuthien0308/ordering/orderservice/config"
	"github.com/phuthien0308/ordering/orderservice/model"
)

var AddOrderHandler = func(ctx *gin.Context) {
	body := []model.Order{}
	if err := ctx.BindJSON(&body); err != nil {
		//Config.Logger.Error(ctx, "can not bind json", err)
		ctx.AbortWithError(http.StatusBadRequest, err)
	}
	data, _ := json.Marshal(config.Config.Db)
	ctx.Writer.WriteString(string(data))
	// if err := repository.Order.AddRange(ctx, body); err != nil {
	// 	ctx.AbortWithError(http.StatusInternalServerError, err)
	// }
}
